package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
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

func makeSnapshots() (snapshot.Snapshot, snapshot.Snapshot) {
	snapA := snapshot.Snapshot{
		ID:   "snap-a",
		Name: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Read", Args: map[string]interface{}{"path": "/a"}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Edit", Args: map[string]interface{}{"path": "/b"}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Bash", Args: map[string]interface{}{"cmd": "ls"}}},
		},
	}
	snapB := snapshot.Snapshot{
		ID:   "snap-b",
		Name: "snap-b",
		Steps: []snapshot.Step{
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Read", Args: map[string]interface{}{"path": "/a"}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Write", Args: map[string]interface{}{"path": "/c"}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Edit", Args: map[string]interface{}{"path": "/b"}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Bash", Args: map[string]interface{}{"cmd": "ls"}}},
		},
	}
	return snapA, snapB
}

func TestTerminalVerboseWithDiagnostics(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	snapA, snapB := makeSnapshots()
	result := makeResult(diff.VerdictChanged)
	result.Diagnostics = &diff.Diagnostics{
		Alignment: []diff.AlignedPair{
			{IndexA: 0, IndexB: 0, Op: diff.AlignMatch, ToolA: "Read", ToolB: "Read"},
			{IndexA: -1, IndexB: 1, Op: diff.AlignInsert, ToolB: "Write"},
			{IndexA: 1, IndexB: 2, Op: diff.AlignMatch, ToolA: "Edit", ToolB: "Edit"},
			{IndexA: 2, IndexB: 3, Op: diff.AlignMatch, ToolA: "Bash", ToolB: "Bash"},
		},
		FirstDivergence: 1,
		Diverged:        false,
		RemapA:          []int{0, 1, 2},
		RemapB:          []int{0, 1, 2, 3},
	}

	var buf bytes.Buffer
	if err := TerminalVerbose(result, snapA, snapB, &buf); err != nil {
		t.Fatalf("TerminalVerbose returned error: %v", err)
	}

	out := buf.String()

	// Should have alignment-aware markers.
	checks := []string{
		"Per-Step Breakdown",
		"First divergence at aligned step 1",
		"+ [B only] step 2: Write",   // Insert marker (remapB[1]=1, +1 for display)
		"[A step 1 / B step 1] Read", // Match marker
		"[A step 2 / B step 3] Edit", // Match marker
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}

	// Should NOT have positional "Step N:" markers.
	if strings.Contains(out, "\nStep 1:\n") {
		t.Error("alignment mode should not use positional 'Step N:' headers")
	}
}

func TestTerminalVerboseWithDiagnosticsSubstAndDelete(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	snapA := snapshot.Snapshot{
		ID:   "snap-a",
		Name: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Read", Args: map[string]interface{}{}}},
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Bash", Args: map[string]interface{}{}}},
		},
	}
	snapB := snapshot.Snapshot{
		ID:   "snap-b",
		Name: "snap-b",
		Steps: []snapshot.Step{
			{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "Write", Args: map[string]interface{}{}}},
		},
	}

	result := makeResult(diff.VerdictChanged)
	result.Diagnostics = &diff.Diagnostics{
		Alignment: []diff.AlignedPair{
			{IndexA: 0, IndexB: 0, Op: diff.AlignSubst, ToolA: "Read", ToolB: "Write"},
			{IndexA: 1, IndexB: -1, Op: diff.AlignDelete, ToolA: "Bash"},
		},
		FirstDivergence: 0,
		Diverged:        true,
		RemapA:          []int{0, 1},
		RemapB:          []int{0},
	}

	var buf bytes.Buffer
	if err := TerminalVerbose(result, snapA, snapB, &buf); err != nil {
		t.Fatalf("TerminalVerbose returned error: %v", err)
	}

	out := buf.String()

	checks := []string{
		"WARNING: Traces diverged (different strategies). Alignment unreliable.",
		"First divergence at aligned step 0",
		"- [A step 1] Read",
		"+ [B step 1] Write",
		"- [A only] step 2: Bash",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestTerminalVerboseWithRetryGroups(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	snapA, snapB := makeSnapshots()
	result := makeResult(diff.VerdictPass)
	result.Diagnostics = &diff.Diagnostics{
		Alignment: []diff.AlignedPair{
			{IndexA: 0, IndexB: 0, Op: diff.AlignMatch, ToolA: "Read", ToolB: "Read"},
		},
		FirstDivergence: -1,
		Diverged:        false,
		RetryGroups: []diff.RetryGroup{
			{ToolName: "Read", CountA: 3, CountB: 2, StartA: 0, StartB: 0},
		},
		RemapA: []int{0, 1, 2},
		RemapB: []int{0, 1, 2, 3},
	}

	var buf bytes.Buffer
	if err := TerminalVerbose(result, snapA, snapB, &buf); err != nil {
		t.Fatalf("TerminalVerbose returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Retries: Read x3 (A) vs Read x2 (B)") {
		t.Errorf("output missing retry group line\nfull output:\n%s", out)
	}
}

func TestTerminalVerboseNilDiagnosticsFallback(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	snapA, snapB := makeSnapshots()
	result := makeResult(diff.VerdictPass)
	// No diagnostics -- should use positional fallback.

	var buf bytes.Buffer
	if err := TerminalVerbose(result, snapA, snapB, &buf); err != nil {
		t.Fatalf("TerminalVerbose returned error: %v", err)
	}

	out := buf.String()

	// Positional mode should have "Step N:" headers.
	if !strings.Contains(out, "\nStep 1:\n") {
		t.Errorf("fallback mode should have positional 'Step 1:' header\nfull output:\n%s", out)
	}
	if !strings.Contains(out, "\nStep 2:\n") {
		t.Errorf("fallback mode should have positional 'Step 2:' header\nfull output:\n%s", out)
	}

	// Should NOT have alignment markers.
	if strings.Contains(out, "Aligned step") {
		t.Error("fallback mode should not contain 'Aligned step'")
	}
}
