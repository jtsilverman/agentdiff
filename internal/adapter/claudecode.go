package adapter

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// ClaudeCodeAdapter parses Claude Code stream-json traces into Steps.
type ClaudeCodeAdapter struct{}

// claudeCodeLine represents a single line in a Claude Code stream-json trace.
type claudeCodeLine struct {
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Model     string          `json:"model,omitempty"`
}

// claudeCodeMessage represents the message object within a stream-json line.
type claudeCodeMessage struct {
	Model   string          `json:"model,omitempty"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// claudeCodeToolResultBlock represents a tool_result block within user message content.
type claudeCodeToolResultBlock struct {
	Type      string          `json:"type"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// Parse reads Claude Code stream-json and returns normalized Steps and metadata.
func (c *ClaudeCodeAdapter) Parse(input []byte) ([]snapshot.Step, map[string]string, error) {
	var steps []snapshot.Step
	meta := map[string]string{}

	// Track tool_use IDs to tool names so tool_result can reference them.
	toolNames := map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(input))
	scanner.Buffer(make([]byte, 0, 64*1024), 50*1024*1024) // 50MB max line size
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var cl claudeCodeLine
		if err := json.Unmarshal(line, &cl); err != nil {
			return nil, nil, fmt.Errorf("line %d: invalid JSON: %w", lineNum, err)
		}

		switch cl.Type {
		case "system":
			// Extract model from top-level field on system lines.
			if cl.Model != "" {
				meta["model"] = cl.Model
			}
			if cl.SessionID != "" {
				meta["session_id"] = cl.SessionID
			}
			// Skip as step.

		case "assistant":
			if cl.SessionID != "" {
				meta["session_id"] = cl.SessionID
			}
			var msg claudeCodeMessage
			if err := json.Unmarshal(cl.Message, &msg); err != nil {
				continue
			}
			if msg.Model != "" {
				meta["model"] = msg.Model
			}
			// Reuse parseContentBlocks from claude.go (same package).
			blocks, err := parseContentBlocks(msg.Content)
			if err != nil {
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

		case "user":
			var msg claudeCodeMessage
			if err := json.Unmarshal(cl.Message, &msg); err != nil {
				continue
			}
			var blocks []claudeCodeToolResultBlock
			if err := json.Unmarshal(msg.Content, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				if b.Type != "tool_result" {
					continue
				}
				name := toolNames[b.ToolUseID]
				// Try to unmarshal content as a JSON string first, fall back to raw bytes.
				var output string
				if err := json.Unmarshal(b.Content, &output); err != nil {
					output = string(b.Content)
				}
				steps = append(steps, snapshot.Step{
					Role: "tool_result",
					ToolResult: &snapshot.ToolResult{
						Name:    name,
						Output:  output,
						IsError: b.IsError,
					},
				})
			}

		case "result":
			// Skip result lines.

		default:
			// Unknown type: skip gracefully.
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("reading input: %w", err)
	}

	return steps, meta, nil
}
