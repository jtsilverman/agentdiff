package diff

import (
	"math"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// helper to build a step with a tool call.
func toolStep(name string, args map[string]interface{}) snapshot.Step {
	return snapshot.Step{
		Role: "tool_call",
		ToolCall: &snapshot.ToolCall{
			Name: name,
			Args: args,
		},
	}
}

// helper to build a step without a tool call (e.g., assistant text).
func textStep(content string) snapshot.Step {
	return snapshot.Step{
		Role:    "assistant",
		Content: content,
	}
}

func TestCompareTools_IdenticalSequences(t *testing.T) {
	steps := []snapshot.Step{
		toolStep("read_file", map[string]interface{}{"path": "/foo.go"}),
		toolStep("write_file", map[string]interface{}{"path": "/bar.go", "content": "x"}),
	}

	result := CompareTools(steps, steps)

	if result.Score != 0.0 {
		t.Errorf("identical sequences: expected score 0.0, got %f", result.Score)
	}
	if result.EditDist != 0 {
		t.Errorf("identical sequences: expected edit distance 0, got %d", result.EditDist)
	}
	if len(result.Added) != 0 {
		t.Errorf("identical sequences: expected no added, got %v", result.Added)
	}
	if len(result.Removed) != 0 {
		t.Errorf("identical sequences: expected no removed, got %v", result.Removed)
	}
	if result.Reordered {
		t.Error("identical sequences: expected reordered=false")
	}
}

func TestCompareTools_CompletelyDifferent(t *testing.T) {
	a := []snapshot.Step{
		toolStep("read_file", map[string]interface{}{"path": "/a"}),
		toolStep("search", map[string]interface{}{"q": "hello"}),
	}
	b := []snapshot.Step{
		toolStep("deploy", map[string]interface{}{"env": "prod"}),
		toolStep("notify", map[string]interface{}{"msg": "done"}),
	}

	result := CompareTools(a, b)

	if result.Score != 1.0 {
		t.Errorf("completely different: expected score 1.0, got %f", result.Score)
	}
	if result.EditDist != 2 {
		t.Errorf("completely different: expected edit distance 2, got %d", result.EditDist)
	}
}

func TestCompareTools_Reordered(t *testing.T) {
	a := []snapshot.Step{
		toolStep("read_file", nil),
		toolStep("write_file", nil),
		toolStep("search", nil),
	}
	b := []snapshot.Step{
		toolStep("search", nil),
		toolStep("read_file", nil),
		toolStep("write_file", nil),
	}

	result := CompareTools(a, b)

	if !result.Reordered {
		t.Error("reordered: expected reordered=true")
	}
	if result.Score <= 0.0 {
		t.Errorf("reordered: expected score > 0, got %f", result.Score)
	}
	if len(result.Added) != 0 {
		t.Errorf("reordered: expected no added, got %v", result.Added)
	}
	if len(result.Removed) != 0 {
		t.Errorf("reordered: expected no removed, got %v", result.Removed)
	}
}

func TestCompareTools_SameToolsDifferentArgs(t *testing.T) {
	a := []snapshot.Step{
		toolStep("read_file", map[string]interface{}{"path": "/foo.go"}),
		toolStep("write_file", map[string]interface{}{"path": "/bar.go", "content": "hello"}),
	}
	b := []snapshot.Step{
		toolStep("read_file", map[string]interface{}{"path": "/baz.go"}),
		toolStep("write_file", map[string]interface{}{"path": "/bar.go", "content": "world"}),
	}

	result := CompareTools(a, b)

	// Same tool sequence, so edit distance is 0 and sequence score is 0.
	// But args differ, so overall score should be > 0 but < 1.
	if result.Score <= 0.0 {
		t.Errorf("different args: expected score > 0, got %f", result.Score)
	}
	if result.Score >= 1.0 {
		t.Errorf("different args: expected score < 1, got %f", result.Score)
	}
	if result.EditDist != 0 {
		t.Errorf("different args: expected edit distance 0, got %d", result.EditDist)
	}
}

func TestCompareTools_AddedRemoved(t *testing.T) {
	a := []snapshot.Step{
		toolStep("read_file", nil),
		toolStep("search", nil),
	}
	b := []snapshot.Step{
		toolStep("read_file", nil),
		toolStep("deploy", nil),
	}

	result := CompareTools(a, b)

	if len(result.Added) != 1 || result.Added[0] != "deploy" {
		t.Errorf("added: expected [deploy], got %v", result.Added)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "search" {
		t.Errorf("removed: expected [search], got %v", result.Removed)
	}
}

func TestCompareTools_BothEmpty(t *testing.T) {
	result := CompareTools(nil, nil)

	if result.Score != 0.0 {
		t.Errorf("both empty: expected score 0.0, got %f", result.Score)
	}
	if len(result.Added) != 0 {
		t.Errorf("both empty: expected no added, got %v", result.Added)
	}
	if len(result.Removed) != 0 {
		t.Errorf("both empty: expected no removed, got %v", result.Removed)
	}
}

func TestCompareTools_OneEmptyOneNot(t *testing.T) {
	steps := []snapshot.Step{
		toolStep("read_file", nil),
		toolStep("write_file", nil),
	}

	result := CompareTools(nil, steps)

	if result.Score != 1.0 {
		t.Errorf("one empty: expected score 1.0, got %f", result.Score)
	}
	if len(result.Added) != 2 {
		t.Errorf("one empty: expected 2 added, got %v", result.Added)
	}
	if len(result.Removed) != 0 {
		t.Errorf("one empty: expected no removed, got %v", result.Removed)
	}

	// Reverse direction.
	result2 := CompareTools(steps, nil)

	if result2.Score != 1.0 {
		t.Errorf("one empty (reverse): expected score 1.0, got %f", result2.Score)
	}
	if len(result2.Removed) != 2 {
		t.Errorf("one empty (reverse): expected 2 removed, got %v", result2.Removed)
	}
	if len(result2.Added) != 0 {
		t.Errorf("one empty (reverse): expected no added, got %v", result2.Added)
	}
}

func TestCompareTools_MixedSteps(t *testing.T) {
	// Ensure non-tool-call steps are ignored.
	a := []snapshot.Step{
		textStep("thinking..."),
		toolStep("read_file", nil),
		textStep("analyzing..."),
		toolStep("write_file", nil),
	}
	b := []snapshot.Step{
		toolStep("read_file", nil),
		toolStep("write_file", nil),
	}

	result := CompareTools(a, b)

	if result.Score != 0.0 {
		t.Errorf("mixed steps: expected score 0.0, got %f", result.Score)
	}
}

func TestCompareTools_EmptyStepsNoToolCalls(t *testing.T) {
	// Steps exist but none have tool calls.
	a := []snapshot.Step{textStep("hello")}
	b := []snapshot.Step{textStep("world")}

	result := CompareTools(a, b)

	if result.Score != 0.0 {
		t.Errorf("no tool calls: expected score 0.0, got %f", result.Score)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b []string
		want int
	}{
		{nil, nil, 0},
		{[]string{"a"}, nil, 1},
		{nil, []string{"a"}, 1},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, 0},
		{[]string{"a", "b"}, []string{"b", "a"}, 2},
		{[]string{"a"}, []string{"b"}, 1},
		{[]string{"a", "b", "c"}, []string{"a", "c"}, 1},
	}

	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestJaccardArgs(t *testing.T) {
	// Identical args.
	a := map[string]interface{}{"path": "/foo", "mode": "r"}
	sim := jaccardArgs(a, a)
	if sim != 1.0 {
		t.Errorf("identical args: expected 1.0, got %f", sim)
	}

	// Completely different args.
	b := map[string]interface{}{"env": "prod", "count": 5.0}
	sim = jaccardArgs(a, b)
	if sim != 0.0 {
		t.Errorf("different args: expected 0.0, got %f", sim)
	}

	// Both empty.
	sim = jaccardArgs(map[string]interface{}{}, map[string]interface{}{})
	if sim != 1.0 {
		t.Errorf("both empty args: expected 1.0, got %f", sim)
	}

	// Partial overlap.
	c := map[string]interface{}{"path": "/foo", "mode": "w"}
	sim = jaccardArgs(a, c)
	expected := 1.0 / 3.0 // 1 match out of 3 unique pairs
	if math.Abs(sim-expected) > 0.001 {
		t.Errorf("partial overlap: expected ~%f, got %f", expected, sim)
	}
}

