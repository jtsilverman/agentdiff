package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "agentdiff",
	Short: "pytest for AI agents -- snapshot behavior, diff across changes, catch regressions",
	Long: `AgentDiff records agent execution traces as structured snapshots,
then diffs snapshots across runs to surface behavioral regressions.

Works like pytest snapshot testing: record captures a baseline,
diff compares two snapshots, report summarizes changes in CI-friendly output.`,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
