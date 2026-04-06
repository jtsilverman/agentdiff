package bench

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"text/tabwriter"

	"github.com/jtsilverman/agentdiff/internal/config"
)

// BenchResult holds all benchmark evaluation results.
type BenchResult struct {
	Seed          int64           `json:"seed"`
	Detection     DetectionResult `json:"detection"`
	ToolThreshold ThresholdResult `json:"tool_threshold"`
	TextThreshold ThresholdResult `json:"text_threshold"`
	StepThreshold ThresholdResult `json:"step_threshold"`
	Clustering    ClusterResult   `json:"clustering"`
	CrossVal      CrossValResult  `json:"cross_validation"`
}

// Run executes the full benchmark suite with a given seed.
func Run(seed int64, verbose bool) BenchResult {
	cfg := DefaultConfig()
	cfg.Seed = seed

	pairs := GenerateLabeledPairs(cfg)

	detection := EvaluateDetection(pairs, config.DefaultConfig())

	toolThreshold := SweepThreshold(pairs, "tool", 0.0, 1.0, 0.05)
	textThreshold := SweepThreshold(pairs, "text", 0.0, 1.0, 0.05)
	stepThreshold := SweepThreshold(pairs, "step", 0, 20, 1)

	strategyTraces := GenerateStrategyTraces(cfg)
	clustering := EvaluateClustering(strategyTraces, 0, 2)

	crossVal := CrossValidate(pairs, 5, rand.New(rand.NewPCG(uint64(seed), 0)))

	return BenchResult{
		Seed:          seed,
		Detection:     detection,
		ToolThreshold: toolThreshold,
		TextThreshold: textThreshold,
		StepThreshold: stepThreshold,
		Clustering:    clustering,
		CrossVal:      crossVal,
	}
}

// FormatTable renders benchmark results as a terminal table.
func FormatTable(result BenchResult, verbose bool) string {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	fmt.Fprintf(w, "AgentDiff Benchmark Results (seed=%d)\n", result.Seed)
	fmt.Fprintln(w, strings.Repeat("=", 60))

	// Regression Detection
	fmt.Fprintln(w, "\nRegression Detection")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "Precision\t%.4f\n", result.Detection.Precision)
	fmt.Fprintf(w, "Recall\t%.4f\n", result.Detection.Recall)
	fmt.Fprintf(w, "F1\t%.4f\n", result.Detection.F1)
	fmt.Fprintf(w, "Accuracy\t%.4f\n", result.Detection.Accuracy)
	fmt.Fprintf(w, "TP/FP/FN/TN\t%d / %d / %d / %d\n",
		result.Detection.TP, result.Detection.FP, result.Detection.FN, result.Detection.TN)

	// Threshold Calibration
	fmt.Fprintln(w, "\nThreshold Calibration")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "Dimension\tCurrent\tOptimal\tAUC\n")

	defaultCfg := config.DefaultConfig()
	fmt.Fprintf(w, "Tool\t%.2f\t%.2f\t%.4f\n",
		defaultCfg.Thresholds.ToolScore, result.ToolThreshold.OptimalPoint.Threshold, result.ToolThreshold.AUC)
	fmt.Fprintf(w, "Text\t%.2f\t%.2f\t%.4f\n",
		defaultCfg.Thresholds.TextScore, result.TextThreshold.OptimalPoint.Threshold, result.TextThreshold.AUC)
	fmt.Fprintf(w, "Step\t%d\t%.0f\t%.4f\n",
		defaultCfg.Thresholds.StepDelta, result.StepThreshold.OptimalPoint.Threshold, result.StepThreshold.AUC)

	// Clustering Quality
	fmt.Fprintln(w, "\nClustering Quality")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "ARI\t%.4f\n", result.Clustering.ARI)
	fmt.Fprintf(w, "Clusters found\t%d\n", result.Clustering.NumClusters)
	fmt.Fprintf(w, "Ground truth strategies\t%d\n", result.Clustering.NumStrategies)
	fmt.Fprintf(w, "Noise points\t%d\n", result.Clustering.NoiseCount)

	// Cross-Validation
	fmt.Fprintln(w, "\nCross-Validation (5-fold)")
	fmt.Fprintln(w, strings.Repeat("-", 40))
	fmt.Fprintf(w, "Mean F1\t%.4f +/- %.4f\n", result.CrossVal.MeanF1, result.CrossVal.StdF1)
	fmt.Fprintf(w, "Optimal tool threshold\t%.2f\n", result.CrossVal.OptimalTool)
	fmt.Fprintf(w, "Optimal text threshold\t%.2f\n", result.CrossVal.OptimalText)
	fmt.Fprintf(w, "Optimal step threshold\t%d\n", result.CrossVal.OptimalStep)

	if verbose {
		fmt.Fprintln(w, "\nVerbose mode: use --json for per-mutation breakdown")
	}

	w.Flush()
	return buf.String()
}

// FormatJSON returns the benchmark results as indented JSON.
func FormatJSON(result BenchResult) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}
