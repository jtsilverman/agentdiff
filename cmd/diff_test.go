package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestDiffThresholdTextFlag(t *testing.T) {
	workDir := makeWorkDir(t)

	// Identical tool sequence, same step count, but different assistant text content.
	// tool_diff.score=0, steps_diff.delta=0, text_diff.score=1.
	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "reading alpha", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "writing beta completely different", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}

	saveTestSnapshot(t, workDir, "thr-a", stepsA)
	saveTestSnapshot(t, workDir, "thr-b", stepsB)

	// Default threshold-text is 0.5. Text score=1 exceeds it → regression.
	_, _, exitCode := runAgentDiff(t, workDir, "diff", "thr-a", "thr-b")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 with default text threshold, got %d", exitCode)
	}

	// Override text threshold to 1.0 (score must be strictly > threshold).
	// text score=1.0 is NOT > 1.0, so it should pass.
	_, _, exitCode = runAgentDiff(t, workDir, "diff",
		"--threshold-text", "1.0",
		"thr-a", "thr-b")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 with threshold-text=1.0, got %d", exitCode)
	}
}

func TestDiffThresholdStepsFlag(t *testing.T) {
	workDir := makeWorkDir(t)

	// Same tool repeated, identical text. Only difference is step count.
	// tool_diff.score will be nonzero due to edit distance. Use identical text to keep text_diff.score=0.
	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}

	saveTestSnapshot(t, workDir, "steps-a", stepsA)
	saveTestSnapshot(t, workDir, "steps-b", stepsB)

	// With all thresholds set very permissively, should pass.
	_, _, exitCode := runAgentDiff(t, workDir, "diff",
		"--threshold-steps", "100", "--threshold-tool", "1.0", "--threshold-text", "1.0",
		"steps-a", "steps-b")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 with permissive thresholds, got %d", exitCode)
	}

	// With step threshold=0 (delta=12 > 0), should regress.
	_, _, exitCode = runAgentDiff(t, workDir, "diff",
		"--threshold-steps", "0", "--threshold-tool", "1.0", "--threshold-text", "1.0",
		"steps-a", "steps-b")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 with step threshold=0, got %d", exitCode)
	}
}

func TestDiffMissingSnapshot(t *testing.T) {
	workDir := makeWorkDir(t)

	saveTestSnapshot(t, workDir, "exists", []snapshot.Step{
		{Role: "assistant", Content: "a"},
	})

	_, stderr, exitCode := runAgentDiff(t, workDir, "diff", "exists", "nonexistent")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for missing snapshot")
	}
	if !strings.Contains(stderr, "nonexistent") {
		t.Fatalf("expected error to mention 'nonexistent', got: %s", stderr)
	}
}

func TestDiffNoArgs(t *testing.T) {
	workDir := makeWorkDir(t)
	_, _, exitCode := runAgentDiff(t, workDir, "diff")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit when diff called with no args")
	}
}

func TestDiffJSONContainsExpectedFields(t *testing.T) {
	workDir := makeWorkDir(t)

	steps := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}
	saveTestSnapshot(t, workDir, "json-fa", steps)
	saveTestSnapshot(t, workDir, "json-fb", steps)

	stdout, _, exitCode := runAgentDiff(t, workDir, "diff", "--json", "json-fa", "json-fb")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	for _, field := range []string{"overall", "tool_diff", "text_diff", "steps_diff", "diagnostics"} {
		if _, ok := result[field]; !ok {
			t.Errorf("expected JSON to contain %q field", field)
		}
	}
}
