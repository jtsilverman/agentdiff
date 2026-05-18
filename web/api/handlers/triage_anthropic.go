package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// AnthropicTriager calls api.anthropic.com/v1/messages to produce a triage answer.
// Use NewAnthropicTriager to construct; pass into GetTriage as the production Triager.
type AnthropicTriager struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicTriager constructs a triager from an API key.
// model defaults to claude-sonnet-4-6 when empty.
func NewAnthropicTriager(apiKey, model string) *AnthropicTriager {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicTriager{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Triage calls Anthropic and returns the structured TriageResult.
// On any failure (network, parse, validation), returns a deterministic fallback:
// classification="variance", summary names the failure mode. The producer-side
// validation pattern: never let LLM error pierce the contract; consumer (handler)
// re-validates as belt-and-braces.
func (a *AnthropicTriager) Triage(ctx context.Context, req TriageRequest) (TriageResult, error) {
	body := a.buildRequestBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fallbackTriage("marshal request: " + err.Error()), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	if err != nil {
		return fallbackTriage("build http request: " + err.Error()), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fallbackTriage("call anthropic: " + err.Error()), nil
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fallbackTriage(fmt.Sprintf("anthropic %d: %s", resp.StatusCode, truncate(string(respBytes), 200))), nil
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fallbackTriage("parse anthropic envelope: " + err.Error()), nil
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	parsed, err := parseTriageJSON(text)
	if err != nil {
		return fallbackTriage("parse triage json: " + err.Error()), nil
	}

	if !isValidClassification(parsed.Classification) {
		return fallbackTriage("invalid classification: " + parsed.Classification), nil
	}

	return parsed, nil
}

// buildRequestBody assembles the Anthropic /v1/messages payload with prompt caching.
// System prompt is split: static rubric+schema block carries cache_control=ephemeral so
// it stays warm across requests; trace JSON travels in the user message (per-request).
func (a *AnthropicTriager) buildRequestBody(req TriageRequest) map[string]interface{} {
	systemRubric := triageSystemRubric

	userContent := fmt.Sprintf(
		"Trace A (%q):\n%s\n\nTrace B (%q):\n%s",
		req.TraceAName, marshalSteps(req.TraceASteps),
		req.TraceBName, marshalSteps(req.TraceBSteps),
	)

	return map[string]interface{}{
		"model":      a.model,
		"max_tokens": 1024,
		"system": []map[string]interface{}{
			{
				"type":          "text",
				"text":          systemRubric,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]interface{}{
			{"role": "user", "content": userContent},
		},
	}
}

// marshalSteps converts a step sequence to a compact, model-readable JSON array.
func marshalSteps(steps []snapshot.Step) string {
	b, err := json.MarshalIndent(steps, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}

// parseTriageJSON pulls a TriageResult out of a model response.
// Accepts either a bare JSON object or a fenced ```json ... ``` block.
func parseTriageJSON(text string) (TriageResult, error) {
	stripped := stripFence(text)

	var out TriageResult
	if err := json.Unmarshal([]byte(stripped), &out); err != nil {
		return TriageResult{}, err
	}
	return out, nil
}

// stripFence removes a leading/trailing markdown code fence if present.
func stripFence(text string) string {
	t := bytes.TrimSpace([]byte(text))
	if bytes.HasPrefix(t, []byte("```json")) {
		t = bytes.TrimPrefix(t, []byte("```json"))
	} else if bytes.HasPrefix(t, []byte("```")) {
		t = bytes.TrimPrefix(t, []byte("```"))
	}
	t = bytes.TrimSuffix(bytes.TrimSpace(t), []byte("```"))
	return string(bytes.TrimSpace(t))
}

// isValidClassification gates the LLM's classification field against the allowed set.
func isValidClassification(c string) bool {
	return c == "regression" || c == "variance" || c == "additive"
}

// fallbackTriage returns the deterministic fallback when the LLM call fails.
// Classification is "variance" because the safest assumption is "differences
// exist but cause is unclear"; the reason string surfaces in the summary so an
// operator can diagnose.
func fallbackTriage(reason string) TriageResult {
	return TriageResult{
		Summary:        "Triage unavailable: " + reason,
		Classification: "variance",
		LikelyCause:    "could not classify automatically",
	}
}

// truncate caps a string at n runes, appending an ellipsis when truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// triageSystemRubric is the cacheable static prefix of the Anthropic system prompt.
const triageSystemRubric = `You are an AI assistant analyzing agent traces to diagnose behavioral differences.

You will receive two traces (Trace A is the baseline, Trace B is a new run). Compare them and return your analysis as STRICT JSON matching this schema:

{
  "summary": "<one or two sentence plain-English summary of what changed>",
  "classification": "<one of: regression | variance | additive>",
  "likely_cause": "<one short sentence naming the most likely cause>"
}

Classification taxonomy:
- "regression": Trace B is worse than A in a measurable way (wrong answer, broken tool call, hallucinated step, missed required step).
- "variance": Differences look like model nondeterminism (different word choices, different but equally valid paths, reordering of equivalent steps).
- "additive": Trace B does additional useful work (extra verification, helpful sub-call, better error handling) without breaking what A did right.

Return ONLY the JSON object, no prose before or after, no markdown fence.`
