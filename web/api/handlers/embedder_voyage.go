package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// VoyageEmbedder is the production Embedder that calls Voyage AI's embeddings
// endpoint. Per spec Open Question #5: Voyage is the default for v1.0.0 for
// simplicity (managed API, no infra weight). If the demo grows expensive,
// swap to a local sentence-transformers binding in a follow-up.
type VoyageEmbedder struct {
	apiKey string
	model  string
	client *http.Client
}

// NewVoyageEmbedder constructs the production embedder. Returns nil when
// apiKey is empty — call sites use the nil to signal "embedding off"
// (default-noop-hook pattern from `default-noop-hook-for-additive-side-effects`).
func NewVoyageEmbedder(apiKey, model string) *VoyageEmbedder {
	if apiKey == "" {
		return nil
	}
	if model == "" {
		model = "voyage-3-lite"
	}
	return &VoyageEmbedder{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

// Embed turns a trace's step sequence into a vector via Voyage AI.
// On any failure (auth, network, malformed response) returns an error so the
// caller can log and skip the persistence write — no synthetic fallback
// vector. /similar naturally returns empty matches for traces without rows.
func (v *VoyageEmbedder) Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error) {
	text := stepsToText(req.Steps)
	if text == "" {
		return EmbedResult{}, fmt.Errorf("empty trace text")
	}

	body, err := json.Marshal(map[string]interface{}{
		"input": []string{text},
		"model": v.model,
	})
	if err != nil {
		return EmbedResult{}, fmt.Errorf("marshal voyage request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return EmbedResult{}, fmt.Errorf("build voyage request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+v.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return EmbedResult{}, fmt.Errorf("voyage call: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return EmbedResult{}, fmt.Errorf("voyage %d: %s", resp.StatusCode, string(raw))
	}

	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return EmbedResult{}, fmt.Errorf("parse voyage response: %w", err)
	}
	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return EmbedResult{}, fmt.Errorf("voyage returned no embedding")
	}

	return EmbedResult{Vector: parsed.Data[0].Embedding, ModelName: v.model}, nil
}

// stepsToText renders a trace's step sequence as a single string the embedding
// model can consume. Includes tool names + content so semantically-related
// traces (same tool order, similar prompts) cluster in vector space.
func stepsToText(steps []snapshot.Step) string {
	var sb strings.Builder
	for _, s := range steps {
		if s.ToolCall != nil {
			sb.WriteString(s.Role)
			sb.WriteString(" tool_call ")
			sb.WriteString(s.ToolCall.Name)
			sb.WriteString(": ")
			// Args as JSON so the model sees actual values without massive blowup.
			if args, err := json.Marshal(s.ToolCall.Args); err == nil && len(args) < 500 {
				sb.Write(args)
			}
			sb.WriteString("\n")
			continue
		}
		if s.ToolResult != nil {
			sb.WriteString("tool_result ")
			sb.WriteString(s.ToolResult.Name)
			sb.WriteString(": ")
			out := s.ToolResult.Output
			if len(out) > 200 {
				out = out[:200]
			}
			sb.WriteString(out)
			sb.WriteString("\n")
			continue
		}
		sb.WriteString(s.Role)
		sb.WriteString(": ")
		sb.WriteString(s.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}
