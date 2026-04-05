package adapter

import (
	"os"
	"testing"
)

func TestLangChainParseTestdata(t *testing.T) {
	data, err := os.ReadFile("../../testdata/langchain_callbacks.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter := &LangChainAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Expected: tool_call, tool_result, tool_call, tool_result, assistant = 5 steps
	// (chain_start/chain_end skipped)
	if got := len(steps); got != 5 {
		t.Fatalf("expected 5 steps, got %d", got)
	}

	// Step 0: tool_call (get_weather)
	if steps[0].Role != "tool_call" {
		t.Errorf("step 0: expected role tool_call, got %q", steps[0].Role)
	}
	if steps[0].ToolCall == nil {
		t.Fatalf("step 0: ToolCall is nil")
	}
	if steps[0].ToolCall.Name != "get_weather" {
		t.Errorf("step 0: expected tool name get_weather, got %q", steps[0].ToolCall.Name)
	}
	if loc, ok := steps[0].ToolCall.Args["location"]; !ok || loc != "San Francisco" {
		t.Errorf("step 0: expected location=San Francisco, got %v", steps[0].ToolCall.Args)
	}

	// Step 1: tool_result (get_weather)
	if steps[1].Role != "tool_result" {
		t.Errorf("step 1: expected role tool_result, got %q", steps[1].Role)
	}
	if steps[1].ToolResult == nil {
		t.Fatalf("step 1: ToolResult is nil")
	}
	if steps[1].ToolResult.Name != "get_weather" {
		t.Errorf("step 1: expected tool result name get_weather, got %q", steps[1].ToolResult.Name)
	}
	if steps[1].ToolResult.Output != "72F, sunny" {
		t.Errorf("step 1: unexpected output %q", steps[1].ToolResult.Output)
	}

	// Step 2: tool_call (get_forecast)
	if steps[2].ToolCall == nil {
		t.Fatalf("step 2: ToolCall is nil")
	}
	if steps[2].ToolCall.Name != "get_forecast" {
		t.Errorf("step 2: expected tool name get_forecast, got %q", steps[2].ToolCall.Name)
	}
	// Verify numeric arg preserved
	if days, ok := steps[2].ToolCall.Args["days"]; !ok || days != float64(3) {
		t.Errorf("step 2: expected days=3, got %v", steps[2].ToolCall.Args["days"])
	}

	// Step 3: tool_result (get_forecast)
	if steps[3].ToolResult == nil {
		t.Fatalf("step 3: ToolResult is nil")
	}
	if steps[3].ToolResult.Output != "Sunny for the next 3 days" {
		t.Errorf("step 3: unexpected output %q", steps[3].ToolResult.Output)
	}

	// Step 4: assistant (llm_end)
	if steps[4].Role != "assistant" {
		t.Errorf("step 4: expected role assistant, got %q", steps[4].Role)
	}
	if steps[4].Content != "The weather in San Francisco is 72F and sunny. The forecast shows sunny skies for the next 3 days." {
		t.Errorf("step 4: unexpected content %q", steps[4].Content)
	}

	// Metadata: agent_name from first chain_start
	if meta["agent_name"] != "AgentExecutor" {
		t.Errorf("expected agent_name=AgentExecutor, got %q", meta["agent_name"])
	}
}

func TestLangChainParseEmpty(t *testing.T) {
	adapter := &LangChainAdapter{}
	steps, meta, err := adapter.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
	if len(meta) != 0 {
		t.Errorf("expected empty metadata, got %v", meta)
	}
}

func TestLangChainParseSingleToolRoundTrip(t *testing.T) {
	input := []byte(`{"type":"on_tool_start","run_id":"t1","name":"search","inputs":{"query":"hello"}}
{"type":"on_tool_end","run_id":"t1","name":"search","outputs":{"output":"result"}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Role != "tool_call" || steps[0].ToolCall.Name != "search" {
		t.Errorf("step 0: expected tool_call search, got %s %v", steps[0].Role, steps[0].ToolCall)
	}
	if steps[1].Role != "tool_result" || steps[1].ToolResult.Output != "result" {
		t.Errorf("step 1: expected tool_result with output=result, got %s %v", steps[1].Role, steps[1].ToolResult)
	}
}

func TestLangChainParseMultipleTools(t *testing.T) {
	input := []byte(`{"type":"on_tool_start","run_id":"t1","name":"tool_a","inputs":{"x":"1"}}
{"type":"on_tool_end","run_id":"t1","name":"tool_a","outputs":{"output":"a_out"}}
{"type":"on_tool_start","run_id":"t2","name":"tool_b","inputs":{"y":"2"}}
{"type":"on_tool_end","run_id":"t2","name":"tool_b","outputs":{"output":"b_out"}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	if steps[0].ToolCall.Name != "tool_a" {
		t.Errorf("step 0: expected tool_a, got %q", steps[0].ToolCall.Name)
	}
	if steps[2].ToolCall.Name != "tool_b" {
		t.Errorf("step 2: expected tool_b, got %q", steps[2].ToolCall.Name)
	}
}

func TestLangChainParseLLMEndGeneration(t *testing.T) {
	input := []byte(`{"type":"on_llm_end","run_id":"l1","name":"ChatOpenAI","outputs":{"generations":[[{"text":"Hello world"}]]}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Role != "assistant" {
		t.Errorf("expected role assistant, got %q", steps[0].Role)
	}
	if steps[0].Content != "Hello world" {
		t.Errorf("expected content 'Hello world', got %q", steps[0].Content)
	}
}

func TestLangChainParseLLMEndFallbackOutput(t *testing.T) {
	input := []byte(`{"type":"on_llm_end","run_id":"l1","name":"ChatOpenAI","outputs":{"output":"fallback text"}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Content != "fallback text" {
		t.Errorf("expected 'fallback text', got %q", steps[0].Content)
	}
}

func TestLangChainParseLLMEndNoContent(t *testing.T) {
	input := []byte(`{"type":"on_llm_end","run_id":"l1","name":"ChatOpenAI","outputs":{}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps for llm_end with no content, got %d", len(steps))
	}
}

func TestLangChainParseUnknownEventsIgnored(t *testing.T) {
	input := []byte(`{"type":"on_retriever_start","run_id":"r1","name":"Retriever","inputs":{}}
{"type":"on_custom_event","run_id":"c1","name":"Custom","inputs":{}}
{"type":"on_tool_start","run_id":"t1","name":"search","inputs":{"q":"test"}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step (unknown events skipped), got %d", len(steps))
	}
	if steps[0].ToolCall.Name != "search" {
		t.Errorf("expected search tool_call, got %q", steps[0].ToolCall.Name)
	}
}

func TestLangChainParseNonMapInputs(t *testing.T) {
	input := []byte(`{"type":"on_tool_start","run_id":"t1","name":"echo","inputs":"just a string"}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].ToolCall.Args["input"] != "just a string" {
		t.Errorf("expected wrapped input, got %v", steps[0].ToolCall.Args)
	}
}

func TestLangChainParseToolEndNoOutputField(t *testing.T) {
	input := []byte(`{"type":"on_tool_end","run_id":"t1","name":"search","outputs":{"result":"data","count":5}}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	// Should JSON-stringify entire outputs since no "output" key
	out := steps[0].ToolResult.Output
	if out == "" {
		t.Error("expected non-empty output")
	}
	// Verify it contains the data
	if !containsStr(out, "data") || !containsStr(out, "count") {
		t.Errorf("expected stringified outputs, got %q", out)
	}
}

func TestLangChainParseMissingOptionalFields(t *testing.T) {
	// Minimal event: no parent_run_id, no inputs, no outputs
	input := []byte(`{"type":"on_tool_start","run_id":"t1","name":"noop"}
{"type":"on_tool_end","run_id":"t1","name":"noop"}
`)
	adapter := &LangChainAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if len(steps[0].ToolCall.Args) != 0 {
		t.Errorf("expected empty args, got %v", steps[0].ToolCall.Args)
	}
	if steps[1].ToolResult.Output != "" {
		t.Errorf("expected empty output, got %q", steps[1].ToolResult.Output)
	}
}

func TestLangChainParseOnlyFirstChainStartSetsAgent(t *testing.T) {
	input := []byte(`{"type":"on_chain_start","run_id":"c1","name":"OuterAgent","inputs":{}}
{"type":"on_chain_start","run_id":"c2","parent_run_id":"c1","name":"InnerChain","inputs":{}}
`)
	adapter := &LangChainAdapter{}
	_, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if meta["agent_name"] != "OuterAgent" {
		t.Errorf("expected agent_name=OuterAgent, got %q", meta["agent_name"])
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
