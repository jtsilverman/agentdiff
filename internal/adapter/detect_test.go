package adapter

import (
	"os"
	"testing"
)

func TestDetectOpenAIArray(t *testing.T) {
	data, err := os.ReadFile("../../testdata/openai_trace.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter, err := Detect(data)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := adapter.(*OpenAIAdapter); !ok {
		t.Errorf("expected OpenAIAdapter, got %T", adapter)
	}
}

func TestDetectOpenAIAPIResponse(t *testing.T) {
	input := []byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)

	adapter, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := adapter.(*OpenAIAdapter); !ok {
		t.Errorf("expected OpenAIAdapter, got %T", adapter)
	}
}

func TestDetectOpenAIMessagesWrapper(t *testing.T) {
	input := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)

	adapter, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := adapter.(*OpenAIAdapter); !ok {
		t.Errorf("expected OpenAIAdapter, got %T", adapter)
	}
}

func TestDetectClaudeJSONL(t *testing.T) {
	data, err := os.ReadFile("../../testdata/claude_trace.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter, err := Detect(data)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := adapter.(*ClaudeAdapter); !ok {
		t.Errorf("expected ClaudeAdapter, got %T", adapter)
	}
}

func TestDetectAgentsSdk(t *testing.T) {
	input := []byte(`{"trace_id": "abc", "spans": []}`)

	a, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*AgentsSdkAdapter); !ok {
		t.Errorf("expected AgentsSdkAdapter, got %T", a)
	}
}

func TestDetectLangChainJSONL(t *testing.T) {
	input := []byte(`{"run_id": "r1", "type": "on_tool_start", "name": "search"}
{"run_id": "r1", "type": "on_tool_end", "name": "search"}`)

	a, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*LangChainAdapter); !ok {
		t.Errorf("expected LangChainAdapter, got %T", a)
	}
}

func TestDetectClaudeNotMisrouted(t *testing.T) {
	// Claude JSONL without run_id should still detect as Claude.
	input := []byte(`{"type": "assistant", "content": "hello"}
{"type": "tool_use", "name": "bash"}`)

	a, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*ClaudeAdapter); !ok {
		t.Errorf("expected ClaudeAdapter, got %T", a)
	}
}

func TestDetectJSONLWithoutRunId(t *testing.T) {
	// JSONL without run_id and without on_ prefix falls through to Claude.
	input := []byte(`{"event": "start", "data": "something"}
{"event": "end", "data": "done"}`)

	a, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*ClaudeAdapter); !ok {
		t.Errorf("expected ClaudeAdapter, got %T", a)
	}
}

func TestDetectClaudeCodeStreamJSON(t *testing.T) {
	data, err := os.ReadFile("../../testdata/claudecode_stream.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	a, err := Detect(data)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*ClaudeCodeAdapter); !ok {
		t.Errorf("expected ClaudeCodeAdapter, got %T", a)
	}
}

func TestDetectClaudeCodeSingleLine(t *testing.T) {
	input := []byte(`{"type":"assistant","session_id":"s1","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`)

	a, err := Detect(input)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*ClaudeCodeAdapter); !ok {
		t.Errorf("expected ClaudeCodeAdapter, got %T", a)
	}
}

func TestDetectClaudeNotMisroutedToClaudeCode(t *testing.T) {
	data, err := os.ReadFile("../../testdata/claude_trace.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	a, err := Detect(data)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if _, ok := a.(*ClaudeAdapter); !ok {
		t.Errorf("expected ClaudeAdapter, got %T", a)
	}
}

func TestDetectGarbageInput(t *testing.T) {
	_, err := Detect([]byte("this is not json at all!!!"))
	if err == nil {
		t.Error("expected error for garbage input, got nil")
	}
}

func TestDetectEmptyInput(t *testing.T) {
	_, err := Detect([]byte(""))
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestDetectWhitespaceOnly(t *testing.T) {
	_, err := Detect([]byte("   \n\t  "))
	if err == nil {
		t.Error("expected error for whitespace-only input, got nil")
	}
}
