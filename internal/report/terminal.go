package report

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
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

// TerminalVerbose writes a detailed diff report with per-step breakdown,
// arg-level changes, and text excerpts.
func TerminalVerbose(result diff.DiffResult, snapA, snapB snapshot.Snapshot, w io.Writer) error {
	// Print the standard summary first.
	if err := Terminal(result, w); err != nil {
		return err
	}

	noColor := os.Getenv("NO_COLOR") != ""
	bold := colorBold
	reset := colorReset
	green := colorGreen
	red := colorRed
	yellow := colorYellow
	if noColor {
		bold = ""
		reset = ""
		green = ""
		red = ""
		yellow = ""
	}

	// Per-step breakdown.
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%sPer-Step Breakdown%s\n", bold, reset)
	fmt.Fprintln(w, strings.Repeat("-", 50))

	if result.Diagnostics != nil {
		printAlignedSteps(w, result.Diagnostics, snapA, snapB, bold, reset, green, red, yellow, noColor)
	} else {
		printPositionalSteps(w, snapA, snapB, green, red, reset, noColor)
	}

	return nil
}

// printAlignedSteps renders per-step breakdown using alignment diagnostics.
func printAlignedSteps(w io.Writer, diag *diff.Diagnostics, snapA, snapB snapshot.Snapshot, bold, reset, green, red, yellow string, noColor bool) {
	// Divergence warning.
	if diag.Diverged {
		fmt.Fprintf(w, "\n%s%sWARNING: Traces diverged (different strategies). Alignment unreliable.%s\n", bold, yellow, reset)
	}

	// First divergence.
	if diag.FirstDivergence >= 0 {
		fmt.Fprintf(w, "\nFirst divergence at aligned step %d\n", diag.FirstDivergence)
	}

	// Retry groups.
	if len(diag.RetryGroups) > 0 {
		fmt.Fprintln(w, "")
		for _, rg := range diag.RetryGroups {
			fmt.Fprintf(w, "Retries: %s x%d (A) vs %s x%d (B)\n", rg.ToolName, rg.CountA, rg.ToolName, rg.CountB)
		}
	}

	// Walk aligned pairs.
	for i, pair := range diag.Alignment {
		fmt.Fprintf(w, "\nAligned step %d:\n", i)

		switch pair.Op {
		case diff.AlignMatch:
			origA := diag.RemapA[pair.IndexA]
			origB := diag.RemapB[pair.IndexB]
			stepA := &snapA.Steps[origA]
			stepB := &snapB.Steps[origB]

			fmt.Fprintf(w, "  [A step %d / B step %d] %s\n", origA+1, origB+1, pair.ToolA)

			// Compare args if both have tool calls.
			if stepA.ToolCall != nil && stepB.ToolCall != nil {
				printArgDiff(w, stepA.ToolCall.Args, stepB.ToolCall.Args, noColor)
			}

			// Text content excerpts.
			if stepA.Content != "" || stepB.Content != "" {
				if stepA.Content == stepB.Content {
					fmt.Fprintf(w, "    text: %s\n", truncate(stepA.Content, 120))
				} else {
					fmt.Fprintf(w, "    %s- %s%s\n", red, truncate(stepA.Content, 120), reset)
					fmt.Fprintf(w, "    %s+ %s%s\n", green, truncate(stepB.Content, 120), reset)
				}
			}

		case diff.AlignSubst:
			origA := diag.RemapA[pair.IndexA]
			origB := diag.RemapB[pair.IndexB]
			fmt.Fprintf(w, "  %s- [A step %d] %s%s\n", red, origA+1, pair.ToolA, reset)
			fmt.Fprintf(w, "  %s+ [B step %d] %s%s\n", green, origB+1, pair.ToolB, reset)

		case diff.AlignInsert:
			origB := diag.RemapB[pair.IndexB]
			fmt.Fprintf(w, "  %s+ [B only] step %d: %s%s\n", green, origB+1, pair.ToolB, reset)

		case diff.AlignDelete:
			origA := diag.RemapA[pair.IndexA]
			fmt.Fprintf(w, "  %s- [A only] step %d: %s%s\n", red, origA+1, pair.ToolA, reset)
		}
	}
}

// printPositionalSteps renders the legacy positional per-step breakdown.
func printPositionalSteps(w io.Writer, snapA, snapB snapshot.Snapshot, green, red, reset string, noColor bool) {
	maxSteps := len(snapA.Steps)
	if len(snapB.Steps) > maxSteps {
		maxSteps = len(snapB.Steps)
	}

	for i := 0; i < maxSteps; i++ {
		var stepA, stepB *snapshot.Step
		if i < len(snapA.Steps) {
			stepA = &snapA.Steps[i]
		}
		if i < len(snapB.Steps) {
			stepB = &snapB.Steps[i]
		}

		fmt.Fprintf(w, "\nStep %d:\n", i+1)

		if stepA == nil {
			fmt.Fprintf(w, "  %s+ [B] %s: %s%s\n", green, stepB.Role, truncate(stepSummary(stepB), 80), reset)
			continue
		}
		if stepB == nil {
			fmt.Fprintf(w, "  %s- [A] %s: %s%s\n", red, stepA.Role, truncate(stepSummary(stepA), 80), reset)
			continue
		}

		if stepA.Role != stepB.Role {
			fmt.Fprintf(w, "  %s- [A] %s: %s%s\n", red, stepA.Role, truncate(stepSummary(stepA), 80), reset)
			fmt.Fprintf(w, "  %s+ [B] %s: %s%s\n", green, stepB.Role, truncate(stepSummary(stepB), 80), reset)
			continue
		}

		// Same role — show differences.
		fmt.Fprintf(w, "  [%s]\n", stepA.Role)

		// Tool call arg-level changes.
		if stepA.ToolCall != nil && stepB.ToolCall != nil {
			if stepA.ToolCall.Name != stepB.ToolCall.Name {
				fmt.Fprintf(w, "    %stool: %s%s -> %s%s%s\n", red, stepA.ToolCall.Name, reset, green, stepB.ToolCall.Name, reset)
			} else {
				fmt.Fprintf(w, "    tool: %s\n", stepA.ToolCall.Name)
				printArgDiff(w, stepA.ToolCall.Args, stepB.ToolCall.Args, noColor)
			}
		}

		// Text content excerpts.
		if stepA.Content != "" || stepB.Content != "" {
			if stepA.Content == stepB.Content {
				fmt.Fprintf(w, "    text: %s\n", truncate(stepA.Content, 120))
			} else {
				fmt.Fprintf(w, "    %s- %s%s\n", red, truncate(stepA.Content, 120), reset)
				fmt.Fprintf(w, "    %s+ %s%s\n", green, truncate(stepB.Content, 120), reset)
			}
		}
	}
}

func stepSummary(s *snapshot.Step) string {
	if s.ToolCall != nil {
		return "tool_call:" + s.ToolCall.Name
	}
	if s.ToolResult != nil {
		return "tool_result:" + s.ToolResult.Name
	}
	return s.Content
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printArgDiff(w io.Writer, argsA, argsB map[string]interface{}, noColor bool) {
	red := colorRed
	green := colorGreen
	reset := colorReset
	if noColor {
		red = ""
		green = ""
		reset = ""
	}

	allKeys := map[string]bool{}
	for k := range argsA {
		allKeys[k] = true
	}
	for k := range argsB {
		allKeys[k] = true
	}

	for k := range allKeys {
		vA, okA := argsA[k]
		vB, okB := argsB[k]

		if !okA {
			fmt.Fprintf(w, "      %s+ %s: %v%s\n", green, k, vB, reset)
		} else if !okB {
			fmt.Fprintf(w, "      %s- %s: %v%s\n", red, k, vA, reset)
		} else if fmt.Sprintf("%v", vA) != fmt.Sprintf("%v", vB) {
			fmt.Fprintf(w, "      %s- %s: %v%s\n", red, k, vA, reset)
			fmt.Fprintf(w, "      %s+ %s: %v%s\n", green, k, vB, reset)
		}
	}
}
