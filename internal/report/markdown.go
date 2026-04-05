package report

import (
	"fmt"
	"io"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
)

// Markdown writes a single diff result as a markdown report to w.
func Markdown(result diff.DiffResult, cfg config.Config, w io.Writer) error {
	fmt.Fprintln(w, "## AgentDiff Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Comparing **%s** vs **%s**\n", result.Snapshot1, result.Snapshot2)
	fmt.Fprintln(w)

	writeMetricTable(result, cfg, w)
	writeDiagnostics(result, w)
	writeVerdict(result, w)

	return nil
}

// CIMarkdown writes multiple diff results as a combined markdown report to w.
func CIMarkdown(results []diff.DiffResult, cfg config.Config, w io.Writer) error {
	fmt.Fprintln(w, "## AgentDiff CI Report")
	fmt.Fprintln(w)

	for _, result := range results {
		fmt.Fprintf(w, "### %s vs %s\n", result.Snapshot1, result.Snapshot2)
		fmt.Fprintln(w)

		writeMetricTable(result, cfg, w)
		writeDiagnostics(result, w)
		writeVerdict(result, w)
	}

	return nil
}

func writeMetricTable(result diff.DiffResult, cfg config.Config, w io.Writer) {
	fmt.Fprintln(w, "| Metric | Score | Threshold | Status |")
	fmt.Fprintln(w, "|--------|-------|-----------|--------|")

	toolStatus := statusIcon(result.ToolDiff.Score <= cfg.Thresholds.ToolScore, result.Overall)
	fmt.Fprintf(w, "| Tool Score | %.2f | %.2f | %s |\n",
		result.ToolDiff.Score, cfg.Thresholds.ToolScore, toolStatus)

	textStatus := statusIcon(result.TextDiff.Score <= cfg.Thresholds.TextScore, result.Overall)
	fmt.Fprintf(w, "| Text Score | %.2f | %.2f | %s |\n",
		result.TextDiff.Score, cfg.Thresholds.TextScore, textStatus)

	stepOk := result.StepsDiff.Delta <= cfg.Thresholds.StepDelta && result.StepsDiff.Delta >= -cfg.Thresholds.StepDelta
	stepStatus := statusIcon(stepOk, result.Overall)
	fmt.Fprintf(w, "| Step Delta | %d | %d | %s |\n",
		result.StepsDiff.Delta, cfg.Thresholds.StepDelta, stepStatus)

	fmt.Fprintln(w)
}

func writeDiagnostics(result diff.DiffResult, w io.Writer) {
	if result.Diagnostics == nil {
		return
	}

	diag := result.Diagnostics

	fmt.Fprintln(w, "<details>")
	fmt.Fprintln(w, "<summary>Alignment Details</summary>")
	fmt.Fprintln(w)

	// Per-step alignment breakdown.
	if len(diag.Alignment) > 0 {
		fmt.Fprintln(w, "| # | Op | Tool A | Tool B |")
		fmt.Fprintln(w, "|---|-----|--------|--------|")
		for i, pair := range diag.Alignment {
			opStr := alignOpString(pair.Op)
			fmt.Fprintf(w, "| %d | %s | %s | %s |\n",
				i+1, opStr, pair.ToolA, pair.ToolB)
		}
		fmt.Fprintln(w)
	}

	// Diagnostics summary.
	if diag.Diverged {
		fmt.Fprintf(w, "First divergence at alignment step **%d**.\n", diag.FirstDivergence+1)
		fmt.Fprintln(w)
	}

	if len(diag.RetryGroups) > 0 {
		fmt.Fprintln(w, "**Retry Groups:**")
		for _, rg := range diag.RetryGroups {
			fmt.Fprintf(w, "- `%s`: %d in A, %d in B\n", rg.ToolName, rg.CountA, rg.CountB)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "</details>")
	fmt.Fprintln(w)
}

func writeVerdict(result diff.DiffResult, w io.Writer) {
	var icon string
	switch result.Overall {
	case diff.VerdictPass:
		icon = "PASS"
	case diff.VerdictChanged:
		icon = "WARNING: CHANGED"
	case diff.VerdictRegression:
		icon = "FAIL: REGRESSION"
	default:
		icon = string(result.Overall)
	}
	fmt.Fprintf(w, "**Verdict: %s**\n", icon)
	fmt.Fprintln(w)
}

func statusIcon(withinThreshold bool, overall diff.Verdict) string {
	if withinThreshold {
		return "ok"
	}
	if overall == diff.VerdictRegression {
		return "FAIL"
	}
	return "WARNING"
}

func alignOpString(op diff.AlignOp) string {
	switch op {
	case diff.AlignMatch:
		return "match"
	case diff.AlignSubst:
		return "subst"
	case diff.AlignInsert:
		return "insert"
	case diff.AlignDelete:
		return "delete"
	default:
		return "?"
	}
}
