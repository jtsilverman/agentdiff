package adapter

import (
	"os"
	"testing"
)

func TestOpenAIParseChatCompletionUsage(t *testing.T) {
	data, err := os.ReadFile("../../testdata/openai_with_usage.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}
	adapter := &OpenAIAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if meta["model"] != "gpt-4o-2024-08-06" {
		t.Errorf("expected model from wrapper, got %q", meta["model"])
	}
	// Wrapper has 1 choice with 1 assistant message that has a single tool_call.
	// extractMessages flattens to 1 message, role-dispatcher emits 1 tool_call step.
	if got := len(steps); got != 1 {
		t.Fatalf("expected 1 step from single-choice wrapper, got %d", got)
	}
	if steps[0].Role != "tool_call" {
		t.Fatalf("expected role 'tool_call', got %q", steps[0].Role)
	}
	if steps[0].CostTokens == nil {
		t.Fatalf("expected CostTokens populated from wrapper.usage, got nil")
	}
	// usage.prompt_tokens=142 + completion_tokens=18 = 160.
	if *steps[0].CostTokens != 160 {
		t.Errorf("expected CostTokens=160 (prompt 142 + completion 18), got %d", *steps[0].CostTokens)
	}
	// OpenAI ChatCompletion response has no per-message latency field; expect nil.
	if steps[0].LatencyMs != nil {
		t.Errorf("expected LatencyMs=nil (no latency in ChatCompletion shape), got %d", *steps[0].LatencyMs)
	}
}

func TestOpenAIParseTestdata(t *testing.T) {
	data, err := os.ReadFile("../../testdata/openai_trace.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter := &OpenAIAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// System message is skipped, so 9 messages - 1 system = 8 steps.
	// But 2 assistant messages have tool_calls (no text), so those become tool_call steps.
	// Expected steps: user, tool_call, tool_result, assistant, user, tool_call, tool_result, assistant = 8
	if got := len(steps); got != 8 {
		t.Fatalf("expected 8 steps, got %d", got)
	}

	// Step 0: user
	if steps[0].Role != "user" {
		t.Errorf("step 0: expected role user, got %q", steps[0].Role)
	}
	if steps[0].Content != "What's the weather in SF?" {
		t.Errorf("step 0: unexpected content %q", steps[0].Content)
	}

	// Step 1: tool_call (get_weather for SF)
	if steps[1].Role != "tool_call" {
		t.Errorf("step 1: expected role tool_call, got %q", steps[1].Role)
	}
	if steps[1].ToolCall == nil {
		t.Fatalf("step 1: ToolCall is nil")
	}
	if steps[1].ToolCall.Name != "get_weather" {
		t.Errorf("step 1: expected tool name get_weather, got %q", steps[1].ToolCall.Name)
	}
	if loc, ok := steps[1].ToolCall.Args["location"]; !ok || loc != "San Francisco" {
		t.Errorf("step 1: expected location=San Francisco, got %v", steps[1].ToolCall.Args)
	}

	// Step 2: tool_result
	if steps[2].Role != "tool_result" {
		t.Errorf("step 2: expected role tool_result, got %q", steps[2].Role)
	}
	if steps[2].ToolResult == nil {
		t.Fatalf("step 2: ToolResult is nil")
	}
	if steps[2].ToolResult.Name != "get_weather" {
		t.Errorf("step 2: expected tool result name get_weather, got %q", steps[2].ToolResult.Name)
	}
	if steps[2].ToolResult.Output != "72\u00b0F, sunny" {
		t.Errorf("step 2: unexpected output %q", steps[2].ToolResult.Output)
	}

	// Step 3: assistant text
	if steps[3].Role != "assistant" {
		t.Errorf("step 3: expected role assistant, got %q", steps[3].Role)
	}
	if steps[3].Content != "The weather in San Francisco is 72\u00b0F and sunny." {
		t.Errorf("step 3: unexpected content %q", steps[3].Content)
	}

	// Step 5: tool_call (get_weather for NYC)
	if steps[5].ToolCall == nil {
		t.Fatalf("step 5: ToolCall is nil")
	}
	if loc, ok := steps[5].ToolCall.Args["location"]; !ok || loc != "New York" {
		t.Errorf("step 5: expected location=New York, got %v", steps[5].ToolCall.Args)
	}

	// Step 7: assistant text (NYC)
	if steps[7].Content != "New York is 55\u00b0F and cloudy." {
		t.Errorf("step 7: unexpected content %q", steps[7].Content)
	}

	// Metadata: no model in array format.
	if len(meta) != 0 {
		t.Errorf("expected empty metadata for array format, got %v", meta)
	}
}

func TestOpenAIParseAPIResponse(t *testing.T) {
	input := []byte(`{
		"model": "gpt-4",
		"choices": [
			{"message": {"role": "assistant", "content": "Hello!"}}
		]
	}`)

	adapter := &OpenAIAdapter{}
	steps, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Role != "assistant" {
		t.Errorf("expected role assistant, got %q", steps[0].Role)
	}
	if steps[0].Content != "Hello!" {
		t.Errorf("expected content Hello!, got %q", steps[0].Content)
	}
	if meta["model"] != "gpt-4" {
		t.Errorf("expected model gpt-4, got %q", meta["model"])
	}
}

func TestOpenAIParseEmptyInput(t *testing.T) {
	adapter := &OpenAIAdapter{}
	_, _, err := adapter.Parse([]byte(""))
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestOpenAIParseEmptyArray(t *testing.T) {
	adapter := &OpenAIAdapter{}
	steps, _, err := adapter.Parse([]byte("[]"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
}
