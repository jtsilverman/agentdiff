package adapter

import (
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// OpenAIAdapter parses OpenAI conversation logs into Steps.
type OpenAIAdapter struct{}

// openaiMessage represents a single message in an OpenAI conversation.
type openaiMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []openaiToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

// openaiToolCall represents a tool call in an OpenAI assistant message.
type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

// openaiToolFunction holds the function name and arguments string.
type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openaiAPIResponse represents an OpenAI API response wrapper.
type openaiAPIResponse struct {
	Model   string         `json:"model,omitempty"`
	Choices []openaiChoice `json:"choices"`
	Usage   *openaiUsage   `json:"usage,omitempty"`
}

// openaiUsage captures the per-call token accounting from a ChatCompletion
// response. Cost for the call = PromptTokens + CompletionTokens (the billable
// surface; OpenAI bills input + output at different rates but both bill).
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// openaiChoice represents one choice in an API response.
type openaiChoice struct {
	Message openaiMessage `json:"message"`
}

// Parse reads OpenAI conversation JSON and returns normalized Steps and metadata.
// Supports two formats:
//   - Direct messages array: [{"role":"user","content":"..."},...]
//   - API response wrapper: {"choices":[{"message":{...}}],"model":"..."}
func (o *OpenAIAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	meta := map[string]string{}

	messages, usage, err := o.extractMessages(input, meta)
	if err != nil {
		return nil, nil, err
	}

	// Track tool call IDs to names for tool results.
	toolNames := map[string]string{}

	// Wrapper-level usage applies to the assistant messages extracted from a
	// single ChatCompletion response. Attach to each assistant-originated step;
	// in the typical 1-choice response this is exactly one assistant turn.
	var wrapperCost *int
	if usage != nil {
		cost := usage.PromptTokens + usage.CompletionTokens
		wrapperCost = &cost
	}

	var steps []snapshot.Step
	for i, msg := range messages {
		switch msg.Role {
		case "system":
			// Skip system messages.
			continue

		case "user":
			steps = append(steps, snapshot.Step{
				Role:    "user",
				Content: msg.Content,
			})

		case "assistant":
			// Capture text content first (present even when tool_calls exist).
			if msg.Content != "" {
				steps = append(steps, snapshot.Step{
					Role:       "assistant",
					Content:    msg.Content,
					CostTokens: wrapperCost,
				})
			}
			for _, tc := range msg.ToolCalls {
				toolNames[tc.ID] = tc.Function.Name
				args, parseErr := parseToolArgs(tc.Function.Arguments)
				if parseErr != nil {
					return nil, nil, fmt.Errorf("message %d: tool_call %q: invalid arguments JSON: %w", i, tc.ID, parseErr)
				}
				steps = append(steps, snapshot.Step{
					Role: "tool_call",
					ToolCall: &snapshot.ToolCall{
						Name: tc.Function.Name,
						Args: args,
					},
					CostTokens: wrapperCost,
				})
			}

		case "tool":
			name := toolNames[msg.ToolCallID]
			steps = append(steps, snapshot.Step{
				Role: "tool_result",
				ToolResult: &snapshot.ToolResult{
					Name:   name,
					Output: msg.Content,
				},
			})

		default:
			// Unknown role: skip.
		}
	}

	return steps, meta, nil
}

// extractMessages determines the input format and returns messages, the
// wrapper-level usage (nil for direct-array inputs), and metadata.
func (o *OpenAIAdapter) extractMessages(input []byte, meta map[string]string) ([]openaiMessage, *openaiUsage, error) {
	// Try direct array first.
	var messages []openaiMessage
	if err := json.Unmarshal(input, &messages); err == nil {
		return messages, nil, nil
	}

	// Try API response wrapper.
	var resp openaiAPIResponse
	if err := json.Unmarshal(input, &resp); err == nil && len(resp.Choices) > 0 {
		if resp.Model != "" {
			meta["model"] = resp.Model
		}
		var msgs []openaiMessage
		for _, c := range resp.Choices {
			msgs = append(msgs, c.Message)
		}
		return msgs, resp.Usage, nil
	}

	return nil, nil, fmt.Errorf("unrecognized OpenAI format: expected JSON array or {\"choices\":[...]} object")
}

// parseToolArgs parses a JSON arguments string into a map.
func parseToolArgs(raw string) (map[string]interface{}, error) {
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	return args, nil
}
