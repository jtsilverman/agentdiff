package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jtsilverman/agentdiff/internal/cluster"
	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/spf13/cobra"
)

// ErrNewStrategy is returned when a snapshot does not match any existing strategy (exit code 1).
var ErrNewStrategy = fmt.Errorf("new strategy detected")

var (
	clusterEpsilon  float64
	clusterMinPts   int
)

var clusterCmd = &cobra.Command{
	Use:   "cluster <baseline-name>",
	Short: "Cluster baseline snapshots into behavioral strategies",
	Long: `Cluster uses DBSCAN to group a baseline's snapshots by tool-call sequence similarity.
Each cluster represents a distinct behavioral strategy the agent uses.`,
	Args: cobra.ExactArgs(1),
	RunE: runClusterAnalyze,
}

var clusterCompareCmd = &cobra.Command{
	Use:   "compare <baseline-name> <snapshot>",
	Short: "Compare a snapshot against clustered baseline strategies",
	Args:  cobra.ExactArgs(2),
	RunE:  runClusterCompare,
}

func init() {
	clusterCmd.PersistentFlags().Float64Var(&clusterEpsilon, "epsilon", 0, "DBSCAN epsilon (0 = auto)")
	clusterCmd.PersistentFlags().IntVar(&clusterMinPts, "min-points", 0, "DBSCAN min points (0 = use config)")
	clusterCmd.AddCommand(clusterCompareCmd)
	rootCmd.AddCommand(clusterCmd)
}

func runClusterAnalyze(cmd *cobra.Command, args []string) error {
	baselineName := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	bs := snapshot.NewBaselineStore(cwd)
	baseline, err := bs.Load(baselineName)
	if err != nil {
		return fmt.Errorf("load baseline %q: %w", baselineName, err)
	}

	epsilon := clusterEpsilon
	if epsilon == 0 {
		epsilon = cfg.Cluster.Epsilon
	}

	minPts := clusterMinPts
	if minPts == 0 {
		minPts = cfg.Cluster.MinPoints
	}

	report, err := cluster.ClusterBaseline(baseline, epsilon, minPts)
	if err != nil {
		return fmt.Errorf("cluster baseline: %w", err)
	}

	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Terminal output.
	fmt.Printf("Baseline: %s (%d snapshots)\n\n", report.BaselineName, report.SnapshotCount)

	for _, s := range report.Strategies {
		fmt.Printf("Strategy %d (%d snapshots, exemplar: %s)\n  Tools: %s\n  Members: %s\n\n",
			s.ID, s.Count, s.Exemplar,
			strings.Join(s.ToolSeq, " \u2192 "),
			strings.Join(s.Members, ", "))
	}

	if len(report.Noise) > 0 {
		fmt.Printf("Noise (%d snapshots): %s\n\n", len(report.Noise), strings.Join(report.Noise, ", "))
	}

	fmt.Printf("Epsilon: %.2f\n", report.Epsilon)
	return nil
}

func runClusterCompare(cmd *cobra.Command, args []string) error {
	baselineName := args[0]
	snapRef := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	bs := snapshot.NewBaselineStore(cwd)
	baseline, err := bs.Load(baselineName)
	if err != nil {
		return fmt.Errorf("load baseline %q: %w", baselineName, err)
	}

	store := snapshot.NewStore(cwd)
	snap, err := store.Load(snapRef)
	if err != nil {
		return fmt.Errorf("load snapshot %q: %w", snapRef, err)
	}

	epsilon := clusterEpsilon
	if epsilon == 0 {
		epsilon = cfg.Cluster.Epsilon
	}

	minPts := clusterMinPts
	if minPts == 0 {
		minPts = cfg.Cluster.MinPoints
	}

	result, err := cluster.CompareToCluster(baseline, snap, epsilon, minPts)
	if err != nil {
		return fmt.Errorf("compare to cluster: %w", err)
	}

	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		if result.Matched {
			fmt.Printf("Matched strategy %d (exemplar: %s, distance: %d, threshold: %d)\n",
				result.StrategyID, result.Exemplar, result.Distance, result.MaxIntraClusterDist)
		} else {
			fmt.Printf("No matching strategy (closest: strategy %d, exemplar: %s, distance: %d, threshold: %d)\n",
				result.StrategyID, result.Exemplar, result.Distance, result.MaxIntraClusterDist)
		}
	}

	if !result.Matched {
		return ErrNewStrategy
	}
	return nil
}
