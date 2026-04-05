package cmd

import (
	"fmt"
	"os"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/spf13/cobra"
)

var (
	jsonOutput  bool
	maxStepsFlag int
)

// ErrRegression is returned when a diff detects a regression (exit code 1).
var ErrRegression = fmt.Errorf("regression detected")

var rootCmd = &cobra.Command{
	Use:   "agentdiff",
	Short: "pytest for AI agents -- snapshot behavior, diff across changes, catch regressions",
	Long: `AgentDiff records agent execution traces as structured snapshots,
then diffs snapshots across runs to surface behavioral regressions.

Works like pytest snapshot testing: record captures a baseline,
diff compares two snapshots, report summarizes changes in CI-friendly output.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().IntVar(&maxStepsFlag, "max-steps", 1000, "maximum tool call steps to align (truncates to last N)")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		diff.SetMaxToolCalls(maxStepsFlag)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if err == ErrRegression {
			os.Exit(1)
		}
		if err == ErrNewStrategy {
			os.Exit(1)
		}
		if err == ErrStyleDrift {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
