package snapshot

import "time"

// Snapshot captures a single agent execution trace.
type Snapshot struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Source    string            `json:"source"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
	Steps     []Step            `json:"steps"`
}

// Step represents one turn in the agent conversation.
type Step struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolCall represents a tool invocation by the agent.
type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// ToolResult represents the output of a tool invocation.
type ToolResult struct {
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error"`
}
