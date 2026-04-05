package cmd

import (
	"fmt"
	"os"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/report"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	thresholdTool  float64
	thresholdText  float64
	thresholdSteps int
)

var diffCmd = &cobra.Command{
	Use:   "diff <snapshot-a> <snapshot-b>",
	Short: "Compare two snapshots and detect regressions",
	Long: `Diff loads two snapshots by name or ID prefix, compares them using
configured thresholds, and reports the result. Exits with code 1 if a
regression is detected.`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().Float64Var(&thresholdTool, "threshold-tool", 0.3, "tool score threshold for regression")
	diffCmd.Flags().Float64Var(&thresholdText, "threshold-text", 0.5, "text score threshold for regression")
	diffCmd.Flags().IntVar(&thresholdSteps, "threshold-steps", 5, "step delta threshold for regression")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Load config and apply CLI overrides.
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cmd.Flags().Changed("threshold-tool") {
		cfg.Thresholds.ToolScore = thresholdTool
	}
	if cmd.Flags().Changed("threshold-text") {
		cfg.Thresholds.TextScore = thresholdText
	}
	if cmd.Flags().Changed("threshold-steps") {
		cfg.Thresholds.StepDelta = thresholdSteps
	}

	// Load snapshots.
	store := snapshot.NewStore(cwd)

	snapA, err := store.Load(args[0])
	if err != nil {
		return fmt.Errorf("load snapshot %q: %w", args[0], err)
	}

	snapB, err := store.Load(args[1])
	if err != nil {
		return fmt.Errorf("load snapshot %q: %w", args[1], err)
	}

	// Compare.
	result := diff.Compare(snapA, snapB, cfg)

	// Output.
	if jsonOutput {
		if err := report.JSON(result, os.Stdout); err != nil {
			return fmt.Errorf("write JSON report: %w", err)
		}
	} else {
		if err := report.Terminal(result, os.Stdout); err != nil {
			return fmt.Errorf("write terminal report: %w", err)
		}
	}

	// Exit 1 on regression.
	if result.Overall == diff.VerdictRegression {
		os.Exit(1)
	}

	return nil
}
