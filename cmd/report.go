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

var reportCmd = &cobra.Command{
	Use:   "report <snapshot-a> <snapshot-b>",
	Short: "Generate a detailed report comparing two snapshots",
	Long: `Report loads two snapshots by name or ID prefix and generates a detailed
comparison report. Functionally identical to diff with verbose output.
Exits with code 1 if a regression is detected.`,
	Args: cobra.ExactArgs(2),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Load config.
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
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

	// Output (verbose: per-step breakdown, arg-level changes, text excerpts).
	if jsonOutput {
		if err := report.JSON(result, os.Stdout); err != nil {
			return fmt.Errorf("write JSON report: %w", err)
		}
	} else {
		if err := report.TerminalVerbose(result, snapA, snapB, os.Stdout); err != nil {
			return fmt.Errorf("write terminal report: %w", err)
		}
	}

	// Exit 1 on regression.
	if result.Overall == diff.VerdictRegression {
		return ErrRegression
	}

	return nil
}
