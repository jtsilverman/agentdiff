package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jtsilverman/agentdiff/internal/adapter"
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
type traceSummaryResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Adapter   string `json:"adapter"`
	StepCount int    `json:"step_count"`
	CreatedAt string `json:"created_at"`
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
	case *adapter.GenericAdapter:
		return "generic"
	default:
		return "unknown"
	}
}

// PostTrace handles POST /api/traces.
func PostTrace(database *db.DB) http.HandlerFunc {
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

		trace, err := database.CreateTrace(name, adapterName, metadata)
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to create trace")
			return
		}

		if err := database.InsertSnapshots(trace.ID, steps); err != nil {
			errorResponse(w, http.StatusInternalServerError, "failed to insert snapshots")
			return
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
				ID:        t.ID,
				Name:      t.Name,
				Adapter:   t.Adapter,
				StepCount: t.StepCount,
				CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
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
			if strings.Contains(err.Error(), "get trace") {
				errorResponse(w, http.StatusNotFound, "trace not found")
				return
			}
			errorResponse(w, http.StatusInternalServerError, "failed to get trace")
			return
		}

		steps := make([]stepResponse, len(trace.Steps))
		for i, s := range trace.Steps {
			sr := stepResponse{
				Role:    s.Role,
				Content: s.Content,
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
