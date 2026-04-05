package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/internal/stats"
	"github.com/spf13/cobra"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Manage statistical baselines for regression detection",
	Long: `Baseline commands let you record snapshots into named baselines,
compare new snapshots against baseline statistics, and list all baselines.`,
}

var baselineRecordCmd = &cobra.Command{
	Use:   "record <name> <snapshot>",
	Short: "Add a snapshot to a baseline",
	Args:  cobra.ExactArgs(2),
	RunE:  runBaselineRecord,
}

var baselineCompareCmd = &cobra.Command{
	Use:   "compare <name> <snapshot>",
	Short: "Compare a snapshot against a baseline using statistical thresholds",
	Args:  cobra.ExactArgs(2),
	RunE:  runBaselineCompare,
}

var baselineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all baselines",
	RunE:  runBaselineList,
}

func init() {
	baselineCmd.AddCommand(baselineRecordCmd)
	baselineCmd.AddCommand(baselineCompareCmd)
	baselineCmd.AddCommand(baselineListCmd)
	rootCmd.AddCommand(baselineCmd)
}

func runBaselineRecord(cmd *cobra.Command, args []string) error {
	name := args[0]
	snapRef := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	store := snapshot.NewStore(cwd)
	snap, err := store.Load(snapRef)
	if err != nil {
		return fmt.Errorf("load snapshot %q: %w", snapRef, err)
	}

	bs := snapshot.NewBaselineStore(cwd)
	if err := bs.AddSnapshot(name, snap); err != nil {
		return fmt.Errorf("add snapshot to baseline: %w", err)
	}

	// Reload to get accurate count.
	baseline, err := bs.Load(name)
	if err != nil {
		return fmt.Errorf("reload baseline: %w", err)
	}

	fmt.Printf("Added %s to baseline %s (%d snapshots total)\n", snap.Name, name, len(baseline.Snapshots))
	return nil
}

func runBaselineCompare(cmd *cobra.Command, args []string) error {
	name := args[0]
	snapRef := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	store := snapshot.NewStore(cwd)
	current, err := store.Load(snapRef)
	if err != nil {
		return fmt.Errorf("load snapshot %q: %w", snapRef, err)
	}

	bs := snapshot.NewBaselineStore(cwd)
	baseline, err := bs.Load(name)
	if err != nil {
		return fmt.Errorf("load baseline %q: %w", name, err)
	}

	if len(baseline.Snapshots) < 2 {
		return fmt.Errorf("baseline %q needs at least 2 snapshots (has %d)", name, len(baseline.Snapshots))
	}

	// Compute inter-baseline diffs to establish normal variance.
	// Each consecutive pair of baseline snapshots is compared.
	var baselineDiffs []diff.DiffResult
	for i := 1; i < len(baseline.Snapshots); i++ {
		d := diff.Compare(baseline.Snapshots[i-1], baseline.Snapshots[i], cfg)
		baselineDiffs = append(baselineDiffs, d)
	}

	// Compute baseline stats from inter-baseline diffs.
	confidence := cfg.Baseline.Confidence
	bstats := stats.ComputeBaselineStats(baselineDiffs, confidence)
	weights := stats.ComputeWeights(bstats, cfg.Thresholds)

	// Diff current snapshot against the first baseline snapshot as the reference.
	currentDiff := diff.Compare(baseline.Snapshots[0], current, cfg)

	// Check if the current diff exceeds the baseline norms.
	isReg, reason := stats.IsRegression(currentDiff, bstats, weights)

	if jsonOutput {
		output := baselineCompareJSON{
			Baseline:   name,
			Snapshot:   current.Name,
			Stats:      bstats,
			Weights:    weights,
			Regression: isReg,
			Reason:     reason,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			return fmt.Errorf("write JSON: %w", err)
		}
	} else {
		fmt.Printf("Baseline: %s (%d snapshots)\n", name, len(baseline.Snapshots))
		fmt.Printf("Current:  %s\n\n", current.Name)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "COMPONENT\tMEAN\tCI LOWER\tCI UPPER\tCURRENT\tTHRESHOLD\tSTATUS")

		currentVals := map[string]float64{
			"tool_score": currentDiff.ToolDiff.Score,
			"text_score": currentDiff.TextDiff.Score,
			"step_delta": float64(currentDiff.StepsDiff.Delta),
		}
		ciMap := map[string]stats.BootstrapResult{
			"tool_score": bstats.ToolScore,
			"text_score": bstats.TextScore,
			"step_delta": bstats.StepDelta,
		}

		for _, cw := range weights {
			ci := ciMap[cw.Name]
			val := currentVals[cw.Name]
			status := "PASS"
			if val > cw.Threshold && val > ci.Upper {
				status = "FAIL"
			}
			fmt.Fprintf(w, "%s\t%.3f\t%.3f\t%.3f\t%.3f\t%.3f\t%s\n",
				cw.Name, ci.Mean, ci.Lower, ci.Upper, val, cw.Threshold, status)
		}
		w.Flush()

		fmt.Println()
		if isReg {
			fmt.Printf("REGRESSION: %s\n", reason)
		} else {
			fmt.Println("PASS: no regression detected")
		}
	}

	if isReg {
		return ErrRegression
	}
	return nil
}

type baselineCompareJSON struct {
	Baseline   string                 `json:"baseline"`
	Snapshot   string                 `json:"snapshot"`
	Stats      stats.BaselineStats    `json:"stats"`
	Weights    []stats.ComponentWeight `json:"weights"`
	Regression bool                   `json:"regression"`
	Reason     string                 `json:"reason"`
}

func runBaselineList(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	bs := snapshot.NewBaselineStore(cwd)
	baselines, err := bs.List()
	if err != nil {
		return fmt.Errorf("list baselines: %w", err)
	}

	if len(baselines) == 0 {
		fmt.Println("No baselines recorded.")
		return nil
	}

	if jsonOutput {
		type baselineInfo struct {
			Name      string `json:"name"`
			Snapshots int    `json:"snapshots"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		var infos []baselineInfo
		for _, b := range baselines {
			infos = append(infos, baselineInfo{
				Name:      b.Name,
				Snapshots: len(b.Snapshots),
				CreatedAt: b.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt: b.UpdatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(infos); err != nil {
			return fmt.Errorf("write JSON: %w", err)
		}
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSNAPSHOTS\tCREATED\tUPDATED")
	for _, b := range baselines {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			b.Name,
			len(b.Snapshots),
			b.CreatedAt.Format("2006-01-02 15:04:05"),
			b.UpdatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	return w.Flush()
}
