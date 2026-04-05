package report

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jtsilverman/agentdiff/internal/diff"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBold   = "\033[1m"
)

// Terminal writes a human-readable diff report to w.
// If the NO_COLOR environment variable is set, ANSI codes are omitted.
func Terminal(result diff.DiffResult, w io.Writer) error {
	noColor := os.Getenv("NO_COLOR") != ""

	green := colorGreen
	red := colorRed
	yellow := colorYellow
	bold := colorBold
	reset := colorReset
	if noColor {
		green = ""
		red = ""
		yellow = ""
		bold = ""
		reset = ""
	}

	// Header
	fmt.Fprintf(w, "Comparing: %s vs %s\n", result.Snapshot1, result.Snapshot2)
	fmt.Fprintln(w, strings.Repeat("-", 50))

	// Tool Changes
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Tool Changes:")
	for _, name := range result.ToolDiff.Added {
		fmt.Fprintf(w, "  %s+ %s%s\n", green, name, reset)
	}
	for _, name := range result.ToolDiff.Removed {
		fmt.Fprintf(w, "  %s- %s%s\n", red, name, reset)
	}
	if result.ToolDiff.Reordered {
		fmt.Fprintf(w, "  %s\u26a0 Tools reordered%s\n", yellow, reset)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  Tool score:\t%.2f\t(edit distance: %d)\n", result.ToolDiff.Score, result.ToolDiff.EditDist)
	tw.Flush()

	// Text Similarity
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Text Similarity:")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "  Text similarity:\t%.1f%%\n", result.TextDiff.Similarity*100)
	fmt.Fprintf(tw, "  Text score:\t%.2f\n", result.TextDiff.Score)
	tw.Flush()

	// Steps
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Steps:")
	delta := result.StepsDiff.Delta
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	fmt.Fprintf(w, "  Steps: %d \u2192 %d (delta: %s%d)\n", result.StepsDiff.CountA, result.StepsDiff.CountB, sign, delta)

	// Verdict
	fmt.Fprintln(w, "")
	var verdictStr string
	switch result.Overall {
	case diff.VerdictPass:
		verdictStr = fmt.Sprintf("%s%sPASS%s", bold, green, reset)
	case diff.VerdictChanged:
		verdictStr = fmt.Sprintf("%s%sCHANGED%s", bold, yellow, reset)
	case diff.VerdictRegression:
		verdictStr = fmt.Sprintf("%s%sREGRESSION%s", bold, red, reset)
	default:
		verdictStr = string(result.Overall)
	}
	fmt.Fprintf(w, "Verdict: %s\n", verdictStr)

	return nil
}
