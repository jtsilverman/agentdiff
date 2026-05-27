package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/adapter"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// traceResponse is the JSON response for POST /api/traces.
type traceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Adapter   string `json:"adapter"`
	StepCount int    `json:"step_count"`
}

// traceSummaryResponse is the JSON response for GET /api/traces list items.
// baseline_id + baseline_name are populated when the trace belongs to a
// baseline (oldest by created_at wins on multi-membership); empty strings
// otherwise. See db.ListTraces.
type traceSummaryResponse struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Adapter      string            `json:"adapter"`
	StepCount    int               `json:"step_count"`
	Metadata     map[string]string `json:"metadata"`
	BaselineID   string            `json:"baseline_id"`
	BaselineName string            `json:"baseline_name"`
	CreatedAt    string            `json:"created_at"`
}

// traceDetailResponse is the JSON response for GET /api/traces/:id.
type traceDetailResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Adapter   string            `json:"adapter"`
	Source    string            `json:"source"`
	Metadata  map[string]string `json:"metadata"`
	Steps     []stepResponse    `json:"steps"`
	CreatedAt string            `json:"created_at"`
}

// stepResponse is the JSON representation of a single step.
type stepResponse struct {
	Role       string              `json:"role"`
	Content    string              `json:"content"`
	ToolCall   *toolCallResponse   `json:"tool_call,omitempty"`
	ToolResult *toolResultResponse `json:"tool_result,omitempty"`
	CostTokens *int                `json:"cost_tokens,omitempty"`
	LatencyMs  *int                `json:"latency_ms,omitempty"`
}

// toolCallResponse is the JSON representation of a tool call.
type toolCallResponse struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// toolResultResponse is the JSON representation of a tool result.
type toolResultResponse struct {
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error"`
}

// errorResponse writes a JSON error with the given status code.
func errorResponse(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// adapterSourceName returns a string name for a detected adapter.
func adapterSourceName(a adapter.Adapter) string {
	switch a.(type) {
	case *adapter.ClaudeAdapter:
		return "claude"
	case *adapter.OpenAIAdapter:
		return "openai"
	case *adapter.AgentsSdkAdapter:
		return "agents_sdk"
	case *adapter.LangChainAdapter:
		return "langchain"
	case *adapter.ClaudeCodeAdapter:
		return "claudecode"
	case *adapter.GenericAdapter:
		return "generic"
	default:
		return "unknown"
	}
}

// PostTrace handles POST /api/traces.
func PostTrace(database *db.DB, embedder Embedder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			errorResponse(w, http.StatusBadRequest, "name query parameter is required")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "failed to read request body")
			return
		}
		defer r.Body.Close()

		if len(body) == 0 {
			errorResponse(w, http.StatusBadRequest, "request body is empty")
			return
		}

		// Parse optional user-provided metadata.
		metadataParam := r.URL.Query().Get("metadata")
		var userMeta map[string]string
		if metadataParam != "" {
			if err := json.Unmarshal([]byte(metadataParam), &userMeta); err != nil {
				errorResponse(w, http.StatusBadRequest, "invalid metadata: "+err.Error())
				return
			}
		}

		adapterParam := r.URL.Query().Get("adapter")
		var detectedAdapter adapter.Adapter
		var adapterName string

		if adapterParam == "" || adapterParam == "auto" {
			detectedAdapter, err = adapter.Detect(body)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, "failed to detect adapter: "+err.Error())
				return
			}
			adapterName = adapterSourceName(detectedAdapter)
		} else {
			detectedAdapter, err = adapter.Get(adapterParam)
			if err != nil {
				errorResponse(w, http.StatusBadRequest, "unknown adapter: "+adapterParam)
				return
			}
			adapterName = adapterParam
		}

		steps, metadata, err := detectedAdapter.Parse(body)
		if err != nil {
			errorResponse(w, http.StatusBadRequest, "failed to parse trace: "+err.Error())
			return
		}

		// Merge user-provided metadata (user keys override adapter keys).
		if len(userMeta) > 0 {
			if metadata == nil {
				metadata = make(map[string]string)
			}
			for k, v := range userMeta {
				metadata[k] = v
			}
		}

		trace, err := database.CreateTrace(name, adapterName, metadata)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to create trace")
			return
		}

		if err := database.InsertSnapshots(trace.ID, steps); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to insert snapshots")
			return
		}

		// Best-effort, non-blocking embedding generation. Nil embedder = off
		// (tests pass nil, see no embedding writes). Failures log and skip;
		// /similar returns empty matches for traces without an embedding row.
		if embedder != nil {
			go generateEmbeddingAsync(embedder, database, trace.ID, trace.Name, steps)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(traceResponse{
			ID:        trace.ID,
			Name:      trace.Name,
			Adapter:   trace.Adapter,
			StepCount: len(steps),
		})
	}
}

// generateEmbeddingAsync runs the embedder off the request-handling path so a
// slow or failing Voyage call never blocks trace upload. Independent timeout
// keeps the goroutine from outliving its usefulness.
func generateEmbeddingAsync(embedder Embedder, database *db.DB, traceID, traceName string, steps []snapshot.Step) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := embedder.Embed(ctx, EmbedRequest{
		TraceID:   traceID,
		TraceName: traceName,
		Steps:     steps,
	})
	if err != nil {
		log.Printf("embedding generation failed for trace %s: %v", traceID, err)
		return
	}
	if err := database.InsertEmbedding(traceID, result.Vector, result.ModelName); err != nil {
		log.Printf("embedding persistence failed for trace %s: %v", traceID, err)
	}
}

// ListTraces handles GET /api/traces.
func ListTraces(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		traces, err := database.ListTraces()
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to list traces")
			return
		}

		resp := make([]traceSummaryResponse, len(traces))
		for i, t := range traces {
			resp[i] = traceSummaryResponse{
				ID:           t.ID,
				Name:         t.Name,
				Adapter:      t.Adapter,
				StepCount:    t.StepCount,
				Metadata:     t.Metadata,
				BaselineID:   t.BaselineID,
				BaselineName: t.BaselineName,
				CreatedAt:    t.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// GetTrace handles GET /api/traces/:id.
func GetTrace(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		trace, err := database.GetTrace(id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to get trace")
			return
		}

		steps := make([]stepResponse, len(trace.Steps))
		for i, s := range trace.Steps {
			sr := stepResponse{
				Role:       s.Role,
				Content:    s.Content,
				CostTokens: s.CostTokens,
				LatencyMs:  s.LatencyMs,
			}
			if s.ToolCall != nil {
				sr.ToolCall = &toolCallResponse{
					Name: s.ToolCall.Name,
					Args: s.ToolCall.Args,
				}
			}
			if s.ToolResult != nil {
				sr.ToolResult = &toolResultResponse{
					Name:    s.ToolResult.Name,
					Output:  s.ToolResult.Output,
					IsError: s.ToolResult.IsError,
				}
			}
			steps[i] = sr
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(traceDetailResponse{
			ID:        trace.ID,
			Name:      trace.Name,
			Adapter:   trace.Adapter,
			Source:    trace.Source,
			Metadata:  trace.Metadata,
			Steps:     steps,
			CreatedAt: trace.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
}
