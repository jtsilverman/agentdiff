package adapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// ClaudeAdapter parses Claude Code JSONL traces into Steps.
type ClaudeAdapter struct{}

// claudeLine represents a single line in a Claude Code JSONL trace.
type claudeLine struct {
	Type      string            `json:"type"`
	Content   json.RawMessage   `json:"content"`
	Model     string            `json:"model,omitempty"`
	ToolUseID string            `json:"tool_use_id,omitempty"`
	IsError   bool              `json:"is_error,omitempty"`
}

// claudeContentBlock is one element in an assistant content array.
type claudeContentBlock struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// Parse reads Claude Code JSONL and returns normalized Steps and metadata.
func (c *ClaudeAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	var steps []snapshot.Step
	meta := map[string]string{}

	// Track tool_use IDs to tool names so tool_result can reference them.
	toolNames := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(input))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var cl claudeLine
		if err := json.Unmarshal(line, &cl); err != nil {
			return nil, nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		switch cl.Type {
		case "human":
			var content string
			if err := json.Unmarshal(cl.Content, &content); err != nil {
				// Content might be structured; skip if not a plain string.
				continue
			}
			steps = append(steps, snapshot.Step{
				Role:    "user",
				Content: content,
			})

		case "assistant":
			if cl.Model != "" {
				meta["model"] = cl.Model
			}
			blocks, err := parseContentBlocks(cl.Content)
			if err != nil {
				// If content isn't an array, skip gracefully.
				continue
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					steps = append(steps, snapshot.Step{
						Role:    "assistant",
						Content: b.Text,
					})
				case "tool_use":
					toolNames[b.ID] = b.Name
					steps = append(steps, snapshot.Step{
						Role: "tool_call",
						ToolCall: &snapshot.ToolCall{
							Name: b.Name,
							Args: b.Input,
						},
					})
				}
			}

		case "tool_result":
			var output string
			if err := json.Unmarshal(cl.Content, &output); err != nil {
				output = string(cl.Content)
			}
			name := toolNames[cl.ToolUseID]
			steps = append(steps, snapshot.Step{
				Role: "tool_result",
				ToolResult: &snapshot.ToolResult{
					Name:    name,
					Output:  output,
					IsError: cl.IsError,
				},
			})

		default:
			// Unknown type: skip gracefully.
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading input: %w", err)
	}

	return steps, meta, nil
}

// parseContentBlocks parses the content field as an array of content blocks.
func parseContentBlocks(raw json.RawMessage) ([]claudeContentBlock, error) {
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
