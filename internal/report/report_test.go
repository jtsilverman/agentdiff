package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/diff"
)

func makeResult(verdict diff.Verdict) diff.DiffResult {
	return diff.DiffResult{
		Snapshot1: "snap-a",
		Snapshot2: "snap-b",
		Overall:   verdict,
		ToolDiff: diff.ToolDiffResult{
			Added:     []string{"bash", "read"},
			Removed:   []string{"write"},
			Reordered: true,
			EditDist:  3,
			Score:     0.42,
		},
		TextDiff: diff.TextDiffResult{
			Similarity: 0.856,
			Score:      0.14,
		},
		StepsDiff: diff.StepsDiffResult{
			CountA: 10,
			CountB: 12,
			Delta:  2,
		},
	}
}

func TestTerminalContainsExpectedSections(t *testing.T) {
	var buf bytes.Buffer
	result := makeResult(diff.VerdictPass)

	if err := Terminal(result, &buf); err != nil {
		t.Fatalf("Terminal returned error: %v", err)
	}

	out := buf.String()

	checks := []string{
		"Comparing: snap-a vs snap-b",
		"+ bash",
		"+ read",
		"- write",
		"Tools reordered",
		"0.42",
		"edit distance: 3",
		"85.6%",
		"0.14",
		"10",
		"12",
		"PASS",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestTerminalRegressionVerdict(t *testing.T) {
	var buf bytes.Buffer
	result := makeResult(diff.VerdictRegression)

	if err := Terminal(result, &buf); err != nil {
		t.Fatalf("Terminal returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "REGRESSION") {
		t.Error("output missing REGRESSION verdict")
	}
}

func TestTerminalChangedVerdict(t *testing.T) {
	var buf bytes.Buffer
	result := makeResult(diff.VerdictChanged)

	if err := Terminal(result, &buf); err != nil {
		t.Fatalf("Terminal returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "CHANGED") {
		t.Error("output missing CHANGED verdict")
	}
}

func TestTerminalNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	var buf bytes.Buffer
	result := makeResult(diff.VerdictRegression)

	if err := Terminal(result, &buf); err != nil {
		t.Fatalf("Terminal returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\033[") {
		t.Error("output contains ANSI escape codes with NO_COLOR set")
	}

	// Content should still be present.
	if !strings.Contains(out, "REGRESSION") {
		t.Error("output missing REGRESSION verdict with NO_COLOR")
	}
	if !strings.Contains(out, "Comparing:") {
		t.Error("output missing header with NO_COLOR")
	}
}

func TestTerminalNegativeDelta(t *testing.T) {
	var buf bytes.Buffer
	result := makeResult(diff.VerdictPass)
	result.StepsDiff.Delta = -3
	result.StepsDiff.CountB = 7

	if err := Terminal(result, &buf); err != nil {
		t.Fatalf("Terminal returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "delta: -3") {
		t.Error("negative delta should not have + prefix")
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	result := makeResult(diff.VerdictPass)

	if err := JSON(result, &buf); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Error("JSON output should end with newline")
	}

	// Verify indentation.
	if !strings.Contains(out, "  ") {
		t.Error("JSON output should be indented")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	original := makeResult(diff.VerdictRegression)

	if err := JSON(original, &buf); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	var decoded diff.DiffResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}

	if decoded.Overall != original.Overall {
		t.Errorf("verdict mismatch: got %q, want %q", decoded.Overall, original.Overall)
	}
	if decoded.Snapshot1 != original.Snapshot1 {
		t.Errorf("snapshot1 mismatch: got %q, want %q", decoded.Snapshot1, original.Snapshot1)
	}
	if decoded.ToolDiff.Score != original.ToolDiff.Score {
		t.Errorf("tool score mismatch: got %f, want %f", decoded.ToolDiff.Score, original.ToolDiff.Score)
	}
	if decoded.TextDiff.Similarity != original.TextDiff.Similarity {
		t.Errorf("similarity mismatch: got %f, want %f", decoded.TextDiff.Similarity, original.TextDiff.Similarity)
	}
	if decoded.StepsDiff.Delta != original.StepsDiff.Delta {
		t.Errorf("delta mismatch: got %d, want %d", decoded.StepsDiff.Delta, original.StepsDiff.Delta)
	}
	if len(decoded.ToolDiff.Added) != len(original.ToolDiff.Added) {
		t.Errorf("added tools count mismatch: got %d, want %d", len(decoded.ToolDiff.Added), len(original.ToolDiff.Added))
	}
}
