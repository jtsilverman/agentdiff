package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestReportVerboseOutput(t *testing.T) {
	workDir := makeWorkDir(t)

	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "reading file", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package main"}},
	}
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "reading file", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "b.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package main"}},
	}

	saveTestSnapshot(t, workDir, "rpt-a", stepsA)
	saveTestSnapshot(t, workDir, "rpt-b", stepsB)

	stdout, _, _ := runAgentDiff(t, workDir, "report", "rpt-a", "rpt-b")

	// Verbose report should include step-level detail.
	if stdout == "" {
		t.Fatal("expected non-empty report output")
	}
	// Should contain something about the snapshots being compared.
	if !strings.Contains(stdout, "read_file") {
		t.Fatalf("expected report to mention 'read_file' tool, got: %s", stdout)
	}
}

func TestReportRegressionExitCode(t *testing.T) {
	workDir := makeWorkDir(t)

	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "totally_different_tool", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "totally_different_tool", Output: "ok"}},
	}

	saveTestSnapshot(t, workDir, "reg-a", stepsA)
	saveTestSnapshot(t, workDir, "reg-b", stepsB)

	_, _, exitCode := runAgentDiff(t, workDir, "report", "reg-a", "reg-b")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 for regression, got %d", exitCode)
	}
}

func TestReportJSON(t *testing.T) {
	workDir := makeWorkDir(t)

	steps := []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "ok"}},
	}
	saveTestSnapshot(t, workDir, "rpt-json-a", steps)
	saveTestSnapshot(t, workDir, "rpt-json-b", steps)

	stdout, _, exitCode := runAgentDiff(t, workDir, "report", "--json", "rpt-json-a", "rpt-json-b")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}
}

func TestReportMissingSnapshot(t *testing.T) {
	workDir := makeWorkDir(t)

	saveTestSnapshot(t, workDir, "rpt-exists", []snapshot.Step{
		{Role: "assistant", Content: "a"},
	})

	_, _, exitCode := runAgentDiff(t, workDir, "report", "rpt-exists", "rpt-ghost")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for missing snapshot")
	}
}

func TestReportNoArgs(t *testing.T) {
	workDir := makeWorkDir(t)
	_, _, exitCode := runAgentDiff(t, workDir, "report")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit when report called with no args")
	}
}
