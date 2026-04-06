package cmd

import (
	"fmt"
	"os"

	"github.com/jtsilverman/agentdiff/internal/bench"
	"github.com/spf13/cobra"
)

var (
	benchSeed    int64
	benchVerbose bool
	benchOutput  string
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Run empirical validation benchmarks",
	Long:  "Generates synthetic traces with controlled mutations and measures regression detection precision, recall, threshold calibration, and clustering quality.",
	RunE: func(cmd *cobra.Command, args []string) error {
		result := bench.Run(benchSeed, benchVerbose)

		if jsonOutput || benchOutput != "" {
			data, err := bench.FormatJSON(result)
			if err != nil {
				return fmt.Errorf("marshal results: %w", err)
			}
			if benchOutput != "" {
				if err := os.WriteFile(benchOutput, data, 0644); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			if jsonOutput {
				fmt.Println(string(data))
			}
		} else {
			fmt.Print(bench.FormatTable(result, benchVerbose))
		}
		return nil // exit code 0 always
	},
}

func init() {
	benchCmd.Flags().Int64Var(&benchSeed, "seed", 42, "random seed for reproducibility")
	benchCmd.Flags().BoolVar(&benchVerbose, "verbose", false, "show per-mutation-type breakdowns")
	benchCmd.Flags().StringVar(&benchOutput, "output", "", "write JSON results to file")
	rootCmd.AddCommand(benchCmd)
}
