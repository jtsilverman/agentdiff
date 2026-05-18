package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicSummarizer calls api.anthropic.com/v1/messages to produce a transcript summary.
// Use NewAnthropicSummarizer to construct; pass into GetTranscript as the production Summarizer.
type AnthropicSummarizer struct {
	apiKey string
	model  string
	client *http.Client
}

// NewAnthropicSummarizer constructs a summarizer from an API key.
// model defaults to claude-sonnet-4-6 when empty.
func NewAnthropicSummarizer(apiKey, model string) *AnthropicSummarizer {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &AnthropicSummarizer{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Summarize calls Anthropic and returns the structured TranscriptResult.
// On any failure (network, parse, validation), returns a deterministic fallback
// describing the failure mode in the summary so the contract still holds.
func (a *AnthropicSummarizer) Summarize(ctx context.Context, req TranscriptRequest) (TranscriptResult, error) {
	body := a.buildRequestBody(req)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fallbackTranscript("marshal request: " + err.Error()), nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	if err != nil {
		return fallbackTranscript("build http request: " + err.Error()), nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fallbackTranscript("call anthropic: " + err.Error()), nil
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fallbackTranscript(fmt.Sprintf("anthropic %d: %s", resp.StatusCode, truncate(string(respBytes), 200))), nil
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return fallbackTranscript("parse anthropic envelope: " + err.Error()), nil
	}

	var text string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}

	parsed, err := parseTranscriptJSON(text)
	if err != nil {
		return fallbackTranscript("parse transcript json: " + err.Error()), nil
	}

	if parsed.Summary == "" {
		return fallbackTranscript("empty summary from model"), nil
	}

	return parsed, nil
}

// buildRequestBody assembles the Anthropic /v1/messages payload with prompt caching.
// System prompt is split: static rubric block carries cache_control=ephemeral so it
// stays warm across requests; trace JSON travels in the user message (per-request).
func (a *AnthropicSummarizer) buildRequestBody(req TranscriptRequest) map[string]interface{} {
	userContent := fmt.Sprintf(
		"Trace name: %q\n\nSteps:\n%s",
		req.TraceName, marshalSteps(req.TraceSteps),
	)

	return map[string]interface{}{
		"model":      a.model,
		"max_tokens": 1024,
		"system": []map[string]interface{}{
			{
				"type":          "text",
				"text":          transcriptSystemRubric,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]interface{}{
			{"role": "user", "content": userContent},
		},
	}
}

// parseTranscriptJSON pulls a TranscriptResult out of a model response.
// Accepts either a bare JSON object or a fenced ```json ... ``` block.
// Reuses stripFence from triage_anthropic.go since the model wraps inconsistently.
func parseTranscriptJSON(text string) (TranscriptResult, error) {
	stripped := stripFence(text)

	var out TranscriptResult
	if err := json.Unmarshal([]byte(stripped), &out); err != nil {
		return TranscriptResult{}, err
	}
	return out, nil
}

// fallbackTranscript returns the deterministic fallback when the LLM call fails.
// Summary surfaces the failure mode; key_decisions is empty so the UI can render
// a "no decisions extracted" state without crashing.
func fallbackTranscript(reason string) TranscriptResult {
	return TranscriptResult{
		Summary:      "Transcript unavailable: " + reason,
		KeyDecisions: []string{},
	}
}

// transcriptSystemRubric is the cacheable static prefix of the Anthropic system prompt.
const transcriptSystemRubric = `You are an AI assistant summarizing agent execution traces in plain English for a human reader.

You will receive a single trace as a JSON array of steps (user prompts, assistant turns, tool calls, tool results). Return your summary as STRICT JSON matching this schema:

{
  "summary": "<one short paragraph (2-4 sentences) describing what the agent did and how it ended, in plain English>",
  "key_decisions": ["<3-5 short phrases naming concrete decisions or pivots the agent made>"]
}

Guidance for the summary:
- Lead with what the user asked and how the agent responded overall.
- Name concrete tools used and concrete outputs when they were load-bearing.
- Do not editorialize about quality unless the trace contains an explicit error.

Guidance for key_decisions:
- Each item is a short phrase, not a sentence.
- Pick decisions a reader would care about: tool choice, branching point, escalation, retry, giving up.

Return ONLY the JSON object, no prose before or after, no markdown fence.`
