package adapter

import (
	"os"
	"testing"
)

func TestClaudeAdapter_ParseTestdata(t *testing.T) {
	data, err := os.ReadFile("../../testdata/claude_trace.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter := &ClaudeAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// 9 lines in testdata: 1 human, 1 text, 3 tool_use, 3 tool_result, 1 text = 9 steps
	if got := len(steps); got != 9 {
		t.Fatalf("expected 9 steps, got %d", got)
	}

	// Step 0: user message
	if steps[0].Role != "user" {
		t.Errorf("step 0: expected role 'user', got %q", steps[0].Role)
	}
	if steps[0].Content != "Fix the bug in main.go" {
		t.Errorf("step 0: unexpected content: %q", steps[0].Content)
	}

	// Step 1: assistant text
	if steps[1].Role != "assistant" {
		t.Errorf("step 1: expected role 'assistant', got %q", steps[1].Role)
	}
	if steps[1].Content != "I'll look at main.go first." {
		t.Errorf("step 1: unexpected content: %q", steps[1].Content)
	}

	// Step 2: tool_call Read
	if steps[2].Role != "tool_call" {
		t.Errorf("step 2: expected role 'tool_call', got %q", steps[2].Role)
	}
	if steps[2].ToolCall == nil {
		t.Fatalf("step 2: ToolCall is nil")
	}
	if steps[2].ToolCall.Name != "Read" {
		t.Errorf("step 2: expected tool name 'Read', got %q", steps[2].ToolCall.Name)
	}
	if fp, ok := steps[2].ToolCall.Args["file_path"]; !ok || fp != "main.go" {
		t.Errorf("step 2: expected file_path='main.go', got %v", steps[2].ToolCall.Args)
	}

	// Step 3: tool_result for Read
	if steps[3].Role != "tool_result" {
		t.Errorf("step 3: expected role 'tool_result', got %q", steps[3].Role)
	}
	if steps[3].ToolResult == nil {
		t.Fatalf("step 3: ToolResult is nil")
	}
	if steps[3].ToolResult.Name != "Read" {
		t.Errorf("step 3: expected tool name 'Read', got %q", steps[3].ToolResult.Name)
	}
	if steps[3].ToolResult.IsError {
		t.Errorf("step 3: expected IsError=false")
	}

	// Step 4: tool_call Bash
	if steps[4].ToolCall == nil || steps[4].ToolCall.Name != "Bash" {
		t.Errorf("step 4: expected Bash tool_call, got %+v", steps[4])
	}

	// Step 5: tool_result for Bash (is_error=true)
	if steps[5].ToolResult == nil {
		t.Fatalf("step 5: ToolResult is nil")
	}
	if !steps[5].ToolResult.IsError {
		t.Errorf("step 5: expected IsError=true for failed Bash")
	}
	if steps[5].ToolResult.Name != "Bash" {
		t.Errorf("step 5: expected tool name 'Bash', got %q", steps[5].ToolResult.Name)
	}

	// Step 6: tool_call Edit
	if steps[6].ToolCall == nil || steps[6].ToolCall.Name != "Edit" {
		t.Errorf("step 6: expected Edit tool_call, got %+v", steps[6])
	}

	// Step 7: tool_result for Edit
	if steps[7].ToolResult == nil || steps[7].ToolResult.Name != "Edit" {
		t.Errorf("step 7: expected Edit tool_result")
	}

	// Step 8: final assistant text
	if steps[8].Role != "assistant" {
		t.Errorf("step 8: expected role 'assistant', got %q", steps[8].Role)
	}

	// Metadata: model should be extracted
	if meta["model"] != "claude-opus-4-20250514" {
		t.Errorf("expected model metadata 'claude-opus-4-20250514', got %q", meta["model"])
	}
}

func TestClaudeAdapter_EmptyInput(t *testing.T) {
	adapter := &ClaudeAdapter{}
	steps, meta, err := adapter.Parse([]byte{})
	if err != nil {
		t.Fatalf("Parse returned error on empty input: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
	if len(meta) != 0 {
		t.Errorf("expected empty metadata, got %v", meta)
	}
}

func TestClaudeAdapter_UnknownTypes(t *testing.T) {
	input := []byte(`{"type":"human","content":"hello"}
{"type":"system","content":"ignored"}
{"type":"debug","info":"also ignored"}
{"type":"assistant","content":[{"type":"text","text":"hi"}]}
`)
	adapter := &ClaudeAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps (skipping unknown types), got %d", len(steps))
	}
	if steps[0].Role != "user" {
		t.Errorf("step 0: expected 'user', got %q", steps[0].Role)
	}
	if steps[1].Role != "assistant" {
		t.Errorf("step 1: expected 'assistant', got %q", steps[1].Role)
	}
}

func TestGet_Claude(t *testing.T) {
	a, err := Get("claude")
	if err != nil {
		t.Fatalf("Get('claude') returned error: %v", err)
	}
	if a == nil {
		t.Fatal("Get('claude') returned nil adapter")
	}
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("Get('nonexistent') should return error")
	}
}
