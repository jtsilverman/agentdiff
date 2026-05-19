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

// AnthropicCounterfactual calls api.anthropic.com/v1/messages to continue an agent
// trace from a midpoint with a modified input.
// Use NewAnthropicCounterfactual to construct; pass into PostCounterfactual as the
// production Counterfactualer.
type AnthropicCounterfactual struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicCounterfactual constructs a Counterfactualer from an API key.
// model defaults to claude-sonnet-4-6 when empty.
func NewAnthropicCounterfactual(apiKey, model string) *AnthropicCounterfactual {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicCounterfactual{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Replay calls Anthropic and returns the agent continuation as a step sequence.
// On any failure (network, parse, validation), returns a synthetic minimal
// continuation describing the failure mode so the handler's contract still holds.
// Note: returns (result, nil) on failure modes — the deterministic fallback is the
// success path from the caller's perspective. Returns (zero, err) only when the
// fallback itself cannot be constructed, which is unreachable in practice.
func (a *AnthropicCounterfactual) Replay(ctx context.Context, req CounterfactualRequest) (CounterfactualResult, error) {
	body := a.buildRequestBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fallbackCounterfactual("marshal request: " + err.Error()), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	if err != nil {
		return fallbackCounterfactual("build http request: " + err.Error()), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fallbackCounterfactual("call anthropic: " + err.Error()), nil
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fallbackCounterfactual(fmt.Sprintf("anthropic %d: %s", resp.StatusCode, truncate(string(respBytes), 200))), nil
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fallbackCounterfactual("parse anthropic envelope: " + err.Error()), nil
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	parsed, err := parseCounterfactualJSON(text)
	if err != nil {
		return fallbackCounterfactual("parse counterfactual json: " + err.Error()), nil
	}

	if len(parsed.NewSteps) == 0 {
		return fallbackCounterfactual("empty continuation from model"), nil
	}

	return parsed, nil
}

// buildRequestBody assembles the Anthropic /v1/messages payload with prompt caching.
// System prompt is the static rubric block carrying cache_control=ephemeral; the
// per-request prefix + modified input travel in the user message.
func (a *AnthropicCounterfactual) buildRequestBody(req CounterfactualRequest) map[string]interface{} {
	userContent := fmt.Sprintf(
		"Original trace: %q\n\nSteps the agent took BEFORE step %d:\n%s\n\nThe input at step %d is now:\n%s\n\nContinue the agent's execution from here. Return the continuation as JSON matching the schema.",
		req.OriginalTraceName,
		req.StepIndex,
		marshalSteps(req.StepsBefore),
		req.StepIndex,
		req.ModifiedInput,
	)

	return map[string]interface{}{
		"model":      a.model,
		"max_tokens": 2048,
		"system": []map[string]interface{}{
			{
				"type":          "text",
				"text":          counterfactualSystemRubric,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]interface{}{
			{"role": "user", "content": userContent},
		},
	}
}

// parseCounterfactualJSON pulls a CounterfactualResult out of a model response.
// Accepts a bare JSON object or a fenced ```json ... ``` block. Reuses stripFence
// from triage_anthropic.go.
func parseCounterfactualJSON(text string) (CounterfactualResult, error) {
	stripped := stripFence(text)

	// Wire shape: {"steps": [...]} so the model has a clear schema.
	var wire struct {
		Steps []snapshot.Step `json:"steps"`
	}
	if err := json.Unmarshal([]byte(stripped), &wire); err != nil {
		return CounterfactualResult{}, err
	}
	return CounterfactualResult{NewSteps: wire.Steps}, nil
}

// fallbackCounterfactual returns the deterministic fallback when the LLM call fails.
// The continuation is a single assistant step naming the failure mode so the user
// sees a real trace with a clear "this is a fallback" signal in the content.
func fallbackCounterfactual(reason string) CounterfactualResult {
	return CounterfactualResult{
		NewSteps: []snapshot.Step{
			{
				Role:    "assistant",
				Content: "Counterfactual replay unavailable: " + reason,
			},
		},
	}
}

// counterfactualSystemRubric is the cacheable static prefix of the Anthropic system prompt.
const counterfactualSystemRubric = `You are an AI assistant simulating what an agent WOULD have done if its input at one step had been different.

You will receive: (1) the agent's step history before the modification point, (2) the new input at the modification step. Continue the agent's execution as if it had received the new input at that step.

Return your continuation as STRICT JSON matching this schema:

{
  "steps": [
    {
      "role": "<assistant | tool | user>",
      "content": "<text content, may be empty>",
      "tool_call":   {"name": "<tool>", "args": {<json args>}},
      "tool_result": {"name": "<tool>", "output": "<text>", "is_error": false}
    }
  ]
}

Guidance:
- Produce 2-6 steps of continuation.
- Match the trace's prior style: same tools, same prose register, same role conventions.
- tool_call and tool_result are both optional per step; include them only when the step is a tool call or its result.
- Plausible continuation only — don't invent capabilities the agent never used in the prefix.
- If the new input is obviously a dead end (malformed, off-topic), produce a short continuation where the agent recognizes the issue and stops.

Return ONLY the JSON object, no prose before or after, no markdown fence.`
