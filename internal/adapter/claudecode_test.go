package adapter

import (
	"os"
	"testing"
)

func TestClaudeCodeAdapterParse(t *testing.T) {
	data, err := os.ReadFile("../../testdata/claudecode_stream.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter := &ClaudeCodeAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// 8 lines in testdata: system(skip), assistant text, assistant tool_use, user tool_result,
	// assistant tool_use, user tool_result, assistant text, result(skip) = 6 steps
	if got := len(steps); got != 6 {
		t.Fatalf("expected 6 steps, got %d", got)
	}

	// Step roles in order
	expectedRoles := []string{"assistant", "tool_call", "tool_result", "tool_call", "tool_result", "assistant"}
	for i, expected := range expectedRoles {
		if steps[i].Role != expected {
			t.Errorf("step %d: expected role %q, got %q", i, expected, steps[i].Role)
		}
	}

	// Step 0: assistant text
	if steps[0].Content != "I'll look at main.go first." {
		t.Errorf("step 0: unexpected content: %q", steps[0].Content)
	}

	// Step 1: tool_call Read
	if steps[1].ToolCall == nil {
		t.Fatalf("step 1: ToolCall is nil")
	}
	if steps[1].ToolCall.Name != "Read" {
		t.Errorf("step 1: expected tool name 'Read', got %q", steps[1].ToolCall.Name)
	}
	if fp, ok := steps[1].ToolCall.Args["file_path"]; !ok || fp != "main.go" {
		t.Errorf("step 1: expected file_path='main.go', got %v", steps[1].ToolCall.Args)
	}

	// Step 2: tool_result for Read
	if steps[2].ToolResult == nil {
		t.Fatalf("step 2: ToolResult is nil")
	}
	if steps[2].ToolResult.Name != "Read" {
		t.Errorf("step 2: expected tool name 'Read', got %q", steps[2].ToolResult.Name)
	}
	if steps[2].ToolResult.IsError {
		t.Errorf("step 2: expected IsError=false")
	}

	// Step 3: tool_call Bash
	if steps[3].ToolCall == nil {
		t.Fatalf("step 3: ToolCall is nil")
	}
	if steps[3].ToolCall.Name != "Bash" {
		t.Errorf("step 3: expected tool name 'Bash', got %q", steps[3].ToolCall.Name)
	}

	// Step 4: tool_result for Bash (is_error=true)
	if steps[4].ToolResult == nil {
		t.Fatalf("step 4: ToolResult is nil")
	}
	if steps[4].ToolResult.Name != "Bash" {
		t.Errorf("step 4: expected tool name 'Bash', got %q", steps[4].ToolResult.Name)
	}
	if !steps[4].ToolResult.IsError {
		t.Errorf("step 4: expected IsError=true for failed Bash")
	}

	// Step 5: final assistant text
	if steps[5].Content != "I've fixed the issue." {
		t.Errorf("step 5: unexpected content: %q", steps[5].Content)
	}

	// Metadata
	if meta["model"] != "claude-opus-4-6" {
		t.Errorf("expected model metadata 'claude-opus-4-6', got %q", meta["model"])
	}
	if meta["session_id"] != "sess_abc123" {
		t.Errorf("expected session_id 'sess_abc123', got %q", meta["session_id"])
	}
}

func TestClaudeCodeAdapterEmpty(t *testing.T) {
	adapter := &ClaudeCodeAdapter{}
	steps, _, err := adapter.Parse([]byte{})
	if err != nil {
		t.Fatalf("Parse returned error on empty input: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}

func TestClaudeCodeAdapterSkipsUnknownTypes(t *testing.T) {
	input := []byte(`{"type":"system","subtype":"init","session_id":"s1","model":"test-model"}
{"type":"result","subtype":"success","duration_ms":100}
{"type":"rate_limit_event","retry_after_ms":5000}
`)
	adapter := &ClaudeCodeAdapter{}
	steps, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
	// System line metadata should still be extracted
	if meta["model"] != "test-model" {
		t.Errorf("expected model 'test-model', got %q", meta["model"])
	}
}

func TestClaudeCodeAdapterUnknownToolID(t *testing.T) {
	input := []byte(`{"type":"user","session_id":"s1","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_unknown","content":"some output","is_error":false}]}}
`)
	adapter := &ClaudeCodeAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].ToolResult == nil {
		t.Fatalf("step 0: ToolResult is nil")
	}
	if steps[0].ToolResult.Name != "" {
		t.Errorf("step 0: expected empty tool name for unknown ID, got %q", steps[0].ToolResult.Name)
	}
	if steps[0].ToolResult.Output != "some output" {
		t.Errorf("step 0: unexpected output: %q", steps[0].ToolResult.Output)
	}
}
