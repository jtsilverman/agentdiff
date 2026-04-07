package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestListEmpty(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, _, exitCode := runAgentDiff(t, workDir, "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "No snapshots recorded") {
		t.Fatalf("expected 'No snapshots recorded', got: %s", stdout)
	}
}

func TestListJSON(t *testing.T) {
	workDir := makeWorkDir(t)

	saveTestSnapshot(t, workDir, "json-list-a", []snapshot.Step{
		{Role: "assistant", Content: "a", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"p": "x"}}},
	})
	saveTestSnapshot(t, workDir, "json-list-b", []snapshot.Step{
		{Role: "assistant", Content: "b", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"p": "y"}}},
	})

	stdout, _, exitCode := runAgentDiff(t, workDir, "list", "--json")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	var snapshots []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &snapshots); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
}

func TestListTableHeaders(t *testing.T) {
	workDir := makeWorkDir(t)

	saveTestSnapshot(t, workDir, "hdr-snap", []snapshot.Step{
		{Role: "assistant", Content: "a"},
	})

	stdout, _, exitCode := runAgentDiff(t, workDir, "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	for _, col := range []string{"NAME", "ID", "SOURCE", "TIMESTAMP", "STEPS"} {
		if !strings.Contains(stdout, col) {
			t.Errorf("expected table header %q in output: %s", col, stdout)
		}
	}
}
