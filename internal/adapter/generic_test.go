package adapter

import (
	"os"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/config"
)

func TestResolveFieldPath(t *testing.T) {
	obj := map[string]interface{}{
		"role": "user",
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": "found",
			},
		},
		"flat": "hello",
		"bad":  42, // non-map intermediate
	}

	tests := []struct {
		name   string
		path   string
		want   interface{}
		wantOK bool
	}{
		{"flat field", "flat", "hello", true},
		{"nested field", "nested.deep.value", "found", true},
		{"missing field", "nonexistent", nil, false},
		{"missing nested", "nested.missing.value", nil, false},
		{"non-map intermediate", "bad.something", nil, false},
		{"empty path", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveFieldPath(obj, tt.path)
			if ok != tt.wantOK {
				t.Errorf("resolveFieldPath(%q): ok=%v, want %v", tt.path, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("resolveFieldPath(%q): got %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGenericParse(t *testing.T) {
	data, err := os.ReadFile("../../testdata/generic_trace.jsonl")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	cfg := config.GenericAdapterConfig{
		RoleField: "kind",
		RoleMap: map[string]string{
			"msg":       "user", // will be overridden by actor check below
			"fn_call":   "tool_call",
			"fn_output": "tool_result",
		},
		ToolNameField:   "action.fn",
		ToolArgsField:   "action.params",
		ToolOutputField: "result",
		ContentField:    "body",
	}

	// The test fixture uses "kind" for role with values msg/fn_call/fn_output.
	// "msg" maps to "user" -- but we also have actor=bot msgs that should be "assistant".
	// Since the generic adapter maps based on role_field only, both msgs map to "user".
	// Let's update the role_map to use actor field instead for a better test.

	// Actually, re-read the spec: role_field resolves to a value, then role_map maps it.
	// With "kind" as role_field, "msg" always maps to the same role.
	// For a proper test, let's use "actor" as role_field instead.

	cfg = config.GenericAdapterConfig{
		RoleField: "actor",
		RoleMap: map[string]string{
			"human":  "user",
			"bot":    "assistant",
			"system": "tool_result",
		},
		ToolNameField:   "action.fn",
		ToolArgsField:   "action.params",
		ToolOutputField: "result",
		ContentField:    "body",
	}

	// But wait -- line 2 (fn_call) has actor=bot, which maps to "assistant", not "tool_call".
	// We need a combined approach. Let's use "kind" and have distinct mappings.
	cfg = config.GenericAdapterConfig{
		RoleField: "kind",
		RoleMap: map[string]string{
			"msg":       "assistant", // both user and bot msgs become assistant for simplicity
			"fn_call":   "tool_call",
			"fn_output": "tool_result",
		},
		ToolNameField:   "action.fn",
		ToolArgsField:   "action.params",
		ToolOutputField: "result",
		ContentField:    "body",
	}

	adapter := NewGenericAdapter(cfg)
	steps, _, err := adapter.Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(steps) != 4 {
		t.Fatalf("got %d steps, want 4", len(steps))
	}

	// Step 0: msg -> assistant
	if steps[0].Role != "assistant" {
		t.Errorf("step 0 role: got %q, want %q", steps[0].Role, "assistant")
	}
	if steps[0].Content != "What is the weather in NYC?" {
		t.Errorf("step 0 content: got %q", steps[0].Content)
	}

	// Step 1: fn_call -> tool_call
	if steps[1].Role != "tool_call" {
		t.Errorf("step 1 role: got %q, want %q", steps[1].Role, "tool_call")
	}
	if steps[1].ToolCall == nil {
		t.Fatal("step 1 tool_call is nil")
	}
	if steps[1].ToolCall.Name != "get_weather" {
		t.Errorf("step 1 tool name: got %q, want %q", steps[1].ToolCall.Name, "get_weather")
	}
	if city, ok := steps[1].ToolCall.Args["city"]; !ok || city != "NYC" {
		t.Errorf("step 1 tool args: got %v", steps[1].ToolCall.Args)
	}

	// Step 2: fn_output -> tool_result
	if steps[2].Role != "tool_result" {
		t.Errorf("step 2 role: got %q, want %q", steps[2].Role, "tool_result")
	}
	if steps[2].ToolResult == nil {
		t.Fatal("step 2 tool_result is nil")
	}
	if steps[2].ToolResult.Name != "get_weather" {
		t.Errorf("step 2 tool name: got %q, want %q", steps[2].ToolResult.Name, "get_weather")
	}
	if steps[2].ToolResult.Output != "72F and sunny" {
		t.Errorf("step 2 tool output: got %q", steps[2].ToolResult.Output)
	}

	// Step 3: msg -> assistant
	if steps[3].Role != "assistant" {
		t.Errorf("step 3 role: got %q, want %q", steps[3].Role, "assistant")
	}
	if steps[3].Content != "It is 72F and sunny in NYC." {
		t.Errorf("step 3 content: got %q", steps[3].Content)
	}
}

func TestGenericMissingFields(t *testing.T) {
	// Lines with missing role field or unmapped roles should be skipped.
	input := []byte(`{"kind":"msg","body":"hello"}
{"body":"no role field"}
{"kind":"unknown_role","body":"bad role"}
`)

	cfg := config.GenericAdapterConfig{
		RoleField: "kind",
		RoleMap: map[string]string{
			"msg": "user",
		},
		ContentField: "body",
	}

	adapter := NewGenericAdapter(cfg)
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Only the first line should parse (msg -> user).
	// Line 2: missing "kind" field -> skipped.
	// Line 3: "unknown_role" not in role_map, raw value "unknown_role" not a valid role -> skipped.
	if len(steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(steps))
	}
	if steps[0].Role != "user" {
		t.Errorf("step 0 role: got %q, want %q", steps[0].Role, "user")
	}
	if steps[0].Content != "hello" {
		t.Errorf("step 0 content: got %q, want %q", steps[0].Content, "hello")
	}
}

func TestGenericNoRoleMap(t *testing.T) {
	// When no role_map is provided, raw values are used directly.
	input := []byte(`{"role":"user","text":"hi"}
{"role":"assistant","text":"hello"}
{"role":"bogus","text":"skip me"}
`)

	cfg := config.GenericAdapterConfig{
		RoleField:    "role",
		ContentField: "text",
	}

	adapter := NewGenericAdapter(cfg)
	steps, _, err := adapter.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(steps))
	}
	if steps[0].Role != "user" {
		t.Errorf("step 0 role: got %q, want %q", steps[0].Role, "user")
	}
	if steps[1].Role != "assistant" {
		t.Errorf("step 1 role: got %q, want %q", steps[1].Role, "assistant")
	}
}
