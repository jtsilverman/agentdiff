package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
)

func makeMarkdownResult(verdict diff.Verdict) diff.DiffResult {
	return diff.DiffResult{
		Snapshot1: "snap-a",
		Snapshot2: "snap-b",
		Overall:   verdict,
		ToolDiff: diff.ToolDiffResult{
			Added:     []string{"bash"},
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

func TestMarkdownContainsTableHeaders(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictPass)
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	checks := []string{
		"## AgentDiff Report",
		"| Metric | Score | Threshold | Status |",
		"| Tool Score |",
		"| Text Score |",
		"| Step Delta |",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestMarkdownContainsDetailsWhenDiagnosticsPresent(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictChanged)
	result.Diagnostics = &diff.Diagnostics{
		Alignment: []diff.AlignedPair{
			{IndexA: 0, IndexB: 0, Op: diff.AlignMatch, ToolA: "bash", ToolB: "bash"},
			{IndexA: 1, IndexB: 1, Op: diff.AlignSubst, ToolA: "read", ToolB: "write"},
		},
		FirstDivergence: 1,
		Diverged:        true,
		RetryGroups: []diff.RetryGroup{
			{ToolName: "bash", CountA: 3, CountB: 2, StartA: 0, StartB: 0},
		},
	}
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "<details>") {
		t.Error("output missing <details> tag")
	}
	if !strings.Contains(out, "</details>") {
		t.Error("output missing </details> tag")
	}
	if !strings.Contains(out, "match") {
		t.Error("output missing alignment op 'match'")
	}
	if !strings.Contains(out, "subst") {
		t.Error("output missing alignment op 'subst'")
	}
	if !strings.Contains(out, "First divergence") {
		t.Error("output missing first divergence info")
	}
	if !strings.Contains(out, "Retry Groups") {
		t.Error("output missing retry groups")
	}
}

func TestMarkdownPassStatus(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictPass)
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Verdict: PASS") {
		t.Error("output missing PASS verdict")
	}
}

func TestMarkdownRegressionStatus(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictRegression)
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "REGRESSION") {
		t.Error("output missing REGRESSION verdict")
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("output missing FAIL status for regression with score above threshold")
	}
}

func TestMarkdownChangedStatus(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictChanged)
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "CHANGED") {
		t.Error("output missing CHANGED verdict")
	}
	if !strings.Contains(out, "WARNING") {
		t.Error("output missing WARNING status for changed with score above threshold")
	}
}

func TestMarkdownNoDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	result := makeMarkdownResult(diff.VerdictPass)
	// Diagnostics is nil by default.
	cfg := config.DefaultConfig()

	if err := Markdown(result, cfg, &buf); err != nil {
		t.Fatalf("Markdown returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "<details>") {
		t.Error("output should not contain <details> when diagnostics are nil")
	}
}

func TestCIMarkdownMultipleResults(t *testing.T) {
	var buf bytes.Buffer
	results := []diff.DiffResult{
		makeMarkdownResult(diff.VerdictPass),
		makeMarkdownResult(diff.VerdictRegression),
	}
	results[1].Snapshot1 = "snap-c"
	results[1].Snapshot2 = "snap-d"
	cfg := config.DefaultConfig()

	if err := CIMarkdown(results, cfg, &buf); err != nil {
		t.Fatalf("CIMarkdown returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "## AgentDiff CI Report") {
		t.Error("output missing CI report header")
	}
	if !strings.Contains(out, "### snap-a vs snap-b") {
		t.Error("output missing first result sub-header")
	}
	if !strings.Contains(out, "### snap-c vs snap-d") {
		t.Error("output missing second result sub-header")
	}
	// Should have two metric tables.
	if strings.Count(out, "| Metric | Score | Threshold | Status |") != 2 {
		t.Error("expected two metric tables in CI report")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("output missing PASS verdict")
	}
	if !strings.Contains(out, "REGRESSION") {
		t.Error("output missing REGRESSION verdict")
	}
}
