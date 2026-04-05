package adapter

import (
	"os"
	"testing"
)

func TestAgentsSdkParseTestdata(t *testing.T) {
	data, err := os.ReadFile("../../testdata/agents_sdk_trace.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	adapter := &AgentsSdkAdapter{}
	steps, meta, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Expected: tool_call(get_weather), tool_result(get_weather),
	//           tool_call(get_time), tool_result(get_time),
	//           assistant(generation output) = 5 steps
	if got := len(steps); got != 5 {
		t.Fatalf("expected 5 steps, got %d", got)
	}

	// Step 0: tool_call get_weather
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

	// Step 1: tool_result get_weather
	if steps[1].Role != "tool_result" {
		t.Errorf("step 1: expected role tool_result, got %q", steps[1].Role)
	}
	if steps[1].ToolResult == nil {
		t.Fatalf("step 1: ToolResult is nil")
	}
	if steps[1].ToolResult.Output != "72F, sunny" {
		t.Errorf("step 1: unexpected output %q", steps[1].ToolResult.Output)
	}

	// Step 2: tool_call get_time (string input wrapped)
	if steps[2].ToolCall == nil {
		t.Fatalf("step 2: ToolCall is nil")
	}
	if steps[2].ToolCall.Name != "get_time" {
		t.Errorf("step 2: expected tool name get_time, got %q", steps[2].ToolCall.Name)
	}
	if v, ok := steps[2].ToolCall.Args["input"]; !ok || v != "America/Los_Angeles" {
		t.Errorf("step 2: expected input=America/Los_Angeles, got %v", steps[2].ToolCall.Args)
	}

	// Step 3: tool_result get_time
	if steps[3].ToolResult == nil {
		t.Fatalf("step 3: ToolResult is nil")
	}
	if steps[3].ToolResult.Output != "2:30 PM PST" {
		t.Errorf("step 3: unexpected output %q", steps[3].ToolResult.Output)
	}

	// Step 4: assistant (generation)
	if steps[4].Role != "assistant" {
		t.Errorf("step 4: expected role assistant, got %q", steps[4].Role)
	}
	if steps[4].Content != "The weather in SF is 72F and sunny. The time is 2:30 PM PST." {
		t.Errorf("step 4: unexpected content %q", steps[4].Content)
	}

	// Metadata
	if meta["trace_id"] != "trace_abc123" {
		t.Errorf("expected trace_id=trace_abc123, got %q", meta["trace_id"])
	}
	if meta["model"] != "gpt-4o" {
		t.Errorf("expected model=gpt-4o, got %q", meta["model"])
	}
}

func TestAgentsSdkEmptyTrace(t *testing.T) {
	input := []byte(`{"trace_id": "t1", "spans": []}`)
	adapter := &AgentsSdkAdapter{}
	steps, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(steps))
	}
	if meta["trace_id"] != "t1" {
		t.Errorf("expected trace_id=t1, got %q", meta["trace_id"])
	}
}

func TestAgentsSdkSingleFunction(t *testing.T) {
	input := []byte(`{
		"trace_id": "t2",
		"spans": [
			{
				"type": "function",
				"name": "search",
				"span_data": {
					"input": {"query": "test"},
					"output": "found 3 results"
				}
			}
		]
	}`)

	adapter := &AgentsSdkAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Role != "tool_call" || steps[0].ToolCall.Name != "search" {
		t.Errorf("step 0: unexpected %+v", steps[0])
	}
	if steps[1].Role != "tool_result" || steps[1].ToolResult.Output != "found 3 results" {
		t.Errorf("step 1: unexpected %+v", steps[1])
	}
}

func TestAgentsSdkUnknownTypeIgnored(t *testing.T) {
	input := []byte(`{
		"trace_id": "t3",
		"spans": [
			{
				"type": "handoff",
				"name": "transfer",
				"span_data": {"target": "agent_b"}
			}
		]
	}`)

	adapter := &AgentsSdkAdapter{}
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(steps) != 0 {
		t.Errorf("expected 0 steps for unknown type, got %d", len(steps))
	}
}

func TestAgentsSdkMetadataExtraction(t *testing.T) {
	input := []byte(`{
		"trace_id": "trace_meta",
		"spans": [
			{
				"type": "agent",
				"name": "TestAgent",
				"span_data": {"model": "gpt-4o-mini"},
				"children": []
			}
		]
	}`)

	adapter := &AgentsSdkAdapter{}
	_, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if meta["trace_id"] != "trace_meta" {
		t.Errorf("expected trace_id=trace_meta, got %q", meta["trace_id"])
	}
	if meta["model"] != "gpt-4o-mini" {
		t.Errorf("expected model=gpt-4o-mini, got %q", meta["model"])
	}
}

func TestAgentsSdkNestedAgentWithChildren(t *testing.T) {
	input := []byte(`{
		"trace_id": "t_nested",
		"spans": [
			{
				"type": "agent",
				"name": "Outer",
				"span_data": {"model": "gpt-4o"},
				"children": [
					{
						"type": "agent",
						"name": "Inner",
						"span_data": {"model": "gpt-4o-mini"},
						"children": [
							{
								"type": "function",
								"name": "lookup",
								"span_data": {"input": {"id": "123"}, "output": "found"}
							}
						]
					},
					{
						"type": "generation",
						"name": "gen",
						"span_data": {"output": "Done."}
					}
				]
			}
		]
	}`)

	adapter := &AgentsSdkAdapter{}
	steps, meta, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// tool_call(lookup), tool_result(lookup), assistant("Done.") = 3
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[0].ToolCall.Name != "lookup" {
		t.Errorf("expected lookup tool_call, got %q", steps[0].ToolCall.Name)
	}
	if steps[2].Content != "Done." {
		t.Errorf("expected Done., got %q", steps[2].Content)
	}

	// Inner model overwrites outer model.
	if meta["model"] != "gpt-4o-mini" {
		t.Errorf("expected model=gpt-4o-mini (inner), got %q", meta["model"])
	}
}
