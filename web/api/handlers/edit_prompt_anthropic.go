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

// AnthropicEditPrompt calls api.anthropic.com/v1/messages to continue an agent
// trace from a midpoint with the user/system prompt at that step rewritten.
// Use NewAnthropicEditPrompt to construct; pass into PostEditPrompt as the
// production EditPrompter.
type AnthropicEditPrompt struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicEditPrompt constructs an EditPrompter from an API key.
// model defaults to claude-sonnet-4-6 when empty.
func NewAnthropicEditPrompt(apiKey, model string) *AnthropicEditPrompt {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicEditPrompt{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Replay calls Anthropic and returns the agent continuation as a step sequence.
// Mirrors AnthropicCounterfactual.Replay's contract: returns (fallback, nil) on
// any failure mode so the handler always has a parseable continuation to persist.
func (a *AnthropicEditPrompt) Replay(ctx context.Context, req EditPromptRequest) (EditPromptResult, error) {
	body := a.buildRequestBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fallbackEditPrompt("marshal request: " + err.Error()), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	if err != nil {
		return fallbackEditPrompt("build http request: " + err.Error()), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fallbackEditPrompt("call anthropic: " + err.Error()), nil
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fallbackEditPrompt(fmt.Sprintf("anthropic %d: %s", resp.StatusCode, truncate(string(respBytes), 200))), nil
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fallbackEditPrompt("parse anthropic envelope: " + err.Error()), nil
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	parsed, err := parseEditPromptJSON(text)
	if err != nil {
		return fallbackEditPrompt("parse edit-prompt json: " + err.Error()), nil
	}

	if len(parsed.NewSteps) == 0 {
		return fallbackEditPrompt("empty continuation from model"), nil
	}

	return parsed, nil
}

// buildRequestBody assembles the Anthropic /v1/messages payload with prompt caching.
// System prompt is the static rubric block carrying cache_control=ephemeral; the
// per-request prefix + modified prompt travel in the user message.
func (a *AnthropicEditPrompt) buildRequestBody(req EditPromptRequest) map[string]interface{} {
	userContent := fmt.Sprintf(
		"Original trace: %q\n\nSteps the agent took BEFORE step %d:\n%s\n\nThe user's prompt at step %d has been rewritten to:\n%s\n\nContinue the agent's execution from here, given the rewritten prompt. Return the continuation as JSON matching the schema.",
		req.OriginalTraceName,
		req.StepIndex,
		marshalSteps(req.StepsBefore),
		req.StepIndex,
		req.ModifiedPrompt,
	)

	return map[string]interface{}{
		"model":      a.model,
		"max_tokens": 2048,
		"system": []map[string]interface{}{
			{
				"type":          "text",
				"text":          editPromptSystemRubric,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]interface{}{
			{"role": "user", "content": userContent},
		},
	}
}

// parseEditPromptJSON pulls an EditPromptResult out of a model response.
// Accepts a bare JSON object or a fenced ```json ... ``` block. Reuses stripFence
// from triage_anthropic.go.
func parseEditPromptJSON(text string) (EditPromptResult, error) {
	stripped := stripFence(text)

	var wire struct {
		Steps []snapshot.Step `json:"steps"`
	}
	if err := json.Unmarshal([]byte(stripped), &wire); err != nil {
		return EditPromptResult{}, err
	}
	return EditPromptResult{NewSteps: wire.Steps}, nil
}

// fallbackEditPrompt returns the deterministic fallback when the LLM call fails.
func fallbackEditPrompt(reason string) EditPromptResult {
	return EditPromptResult{
		NewSteps: []snapshot.Step{
			{
				Role:    "assistant",
				Content: "Prompt-edit replay unavailable: " + reason,
			},
		},
	}
}

// editPromptSystemRubric is the cacheable static prefix of the Anthropic system prompt.
const editPromptSystemRubric = `You are an AI assistant simulating what an agent WOULD have done if the user's prompt at one step had been rewritten.

You will receive: (1) the agent's step history before the prompt-edit point, (2) the rewritten user prompt at that step. Continue the agent's execution as if the user had given the rewritten prompt instead.

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
- If the rewritten prompt is obviously a dead end (malformed, off-topic), produce a short continuation where the agent recognizes the issue and stops.

Return ONLY the JSON object, no prose before or after, no markdown fence.`
