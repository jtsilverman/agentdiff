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
