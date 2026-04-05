package cmd

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/report"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/spf13/cobra"
)

// ErrStyleDrift is returned when only text regressions are detected (exit code 2).
var ErrStyleDrift = fmt.Errorf("style drift detected")

var (
	ciOutputFile  string
	ciBaselinePath string
)

var ciCmd = &cobra.Command{
	Use:   "ci",
	Short: "Compare current snapshots against a baseline for CI pipelines",
	Long: `CI loads a baseline file and current snapshots, matches them by name,
runs diff comparisons, and outputs a markdown report. Exit codes:
  0 = pass, 1 = functional regression, 2 = style drift only.`,
	RunE: runCI,
}

func init() {
	ciCmd.Flags().StringVar(&ciOutputFile, "output", "", "write markdown report to file instead of stdout")
	ciCmd.Flags().StringVar(&ciBaselinePath, "baseline", "", "override ci.baseline_path from config")
	rootCmd.AddCommand(ciCmd)
}

// loadBaselineFile reads a baseline directly from a gzipped JSON file path.
func loadBaselineFile(path string) (snapshot.Baseline, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot.Baseline{}, fmt.Errorf("baseline file not found: %s", path)
		}
		return snapshot.Baseline{}, fmt.Errorf("open baseline file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return snapshot.Baseline{}, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gr.Close()

	var b snapshot.Baseline
	if err := json.NewDecoder(gr).Decode(&b); err != nil {
		return snapshot.Baseline{}, fmt.Errorf("decode baseline: %w", err)
	}

	return b, nil
}

// latestByName returns a map from snapshot name to the most recent snapshot with that name.
func latestByName(snaps []snapshot.Snapshot) map[string]snapshot.Snapshot {
	result := make(map[string]snapshot.Snapshot)
	for _, s := range snaps {
		existing, ok := result[s.Name]
		if !ok || s.Timestamp.After(existing.Timestamp) {
			result[s.Name] = s
		}
	}
	return result
}

func runCI(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Load config.
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Resolve baseline path: flag overrides config.
	baselinePath := cfg.CI.BaselinePath
	if ciBaselinePath != "" {
		baselinePath = ciBaselinePath
	}
	if baselinePath == "" {
		return fmt.Errorf("no baseline path configured: set ci.baseline_path in .agentdiff.yaml or use --baseline flag")
	}

	// Resolve relative paths against cwd.
	if !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(cwd, baselinePath)
	}

	// Load baseline.
	baseline, err := loadBaselineFile(baselinePath)
	if err != nil {
		return fmt.Errorf("load baseline: %w", err)
	}

	// Build lookup of latest baseline snapshot per name.
	baselineByName := latestByName(baseline.Snapshots)

	// Load current snapshots.
	store := snapshot.NewStore(cwd)
	currentSnaps, err := store.List()
	if err != nil {
		return fmt.Errorf("load current snapshots: %w", err)
	}

	// Match and compare.
	var results []diff.DiffResult
	// Sort current snapshots by name for deterministic output.
	sort.Slice(currentSnaps, func(i, j int) bool {
		return currentSnaps[i].Name < currentSnaps[j].Name
	})

	for _, current := range currentSnaps {
		baseSnap, ok := baselineByName[current.Name]
		if !ok {
			fmt.Fprintf(os.Stderr, "warning: no baseline match for snapshot %q, skipping\n", current.Name)
			continue
		}
		result := diff.Compare(baseSnap, current, cfg)
		results = append(results, result)
	}

	// Render markdown report (always, before any non-zero exit).
	if ciOutputFile != "" {
		outPath := ciOutputFile
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(cwd, outPath)
		}
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		if err := report.CIMarkdown(results, cfg, f); err != nil {
			return fmt.Errorf("write CI report: %w", err)
		}
	} else {
		if err := report.CIMarkdown(results, cfg, os.Stdout); err != nil {
			return fmt.Errorf("write CI report: %w", err)
		}
	}

	// Determine exit code.
	hasToolRegression := false
	hasStepRegression := false
	hasTextOnlyRegression := false

	for _, r := range results {
		toolExceeded := r.ToolDiff.Score > cfg.Thresholds.ToolScore
		stepExceeded := r.StepsDiff.Delta > cfg.Thresholds.StepDelta
		textExceeded := r.TextDiff.Score > cfg.Thresholds.TextScore

		if toolExceeded {
			hasToolRegression = true
		}
		if stepExceeded {
			hasStepRegression = true
		}
		if textExceeded && !toolExceeded && !stepExceeded {
			hasTextOnlyRegression = true
		}
	}

	// Exit 1: tool or step regression.
	if hasToolRegression || hasStepRegression {
		return ErrRegression
	}

	// Exit 2: text-only regression with fail_on_style_drift disabled.
	if hasTextOnlyRegression && !cfg.CI.FailOnStyleDrift {
		return ErrStyleDrift
	}

	// If fail_on_style_drift is true, text regression is a hard failure.
	if hasTextOnlyRegression && cfg.CI.FailOnStyleDrift {
		return ErrRegression
	}

	return nil
}
