package stats

import (
	"fmt"
	"math"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
)

// ComponentWeight holds the computed weight and adjusted threshold for a diff component.
type ComponentWeight struct {
	Name      string  `json:"name"`
	CV        float64 `json:"cv"`
	Weight    float64 `json:"weight"`
	Threshold float64 `json:"threshold"`
}

// ComputeWeights calculates per-component weights and adjusted thresholds
// based on coefficient of variation (CV = std/mean). Low CV (< 0.1) tightens
// the threshold by 20%; high CV (> 0.5) relaxes it by 30%.
func ComputeWeights(stats BaselineStats, thresholds config.Thresholds) []ComponentWeight {
	components := []struct {
		name      string
		result    BootstrapResult
		threshold float64
	}{
		{"tool_score", stats.ToolScore, thresholds.ToolScore},
		{"text_score", stats.TextScore, thresholds.TextScore},
		{"step_delta", stats.StepDelta, float64(thresholds.StepDelta)},
	}

	weights := make([]ComponentWeight, len(components))
	totalCV := 0.0

	for i, c := range components {
		cv := computeCV(c.result)
		adjustedThreshold := c.threshold

		switch {
		case cv < 0.1:
			adjustedThreshold *= 0.8
		case cv > 0.5:
			adjustedThreshold *= 1.3
		}

		weights[i] = ComponentWeight{
			Name:      c.name,
			CV:        cv,
			Threshold: adjustedThreshold,
		}
		totalCV += cv
	}

	// Normalize weights to sum to 1.0.
	if totalCV == 0 {
		// Equal weights when all CVs are zero.
		for i := range weights {
			weights[i].Weight = 1.0 / float64(len(weights))
		}
	} else {
		for i := range weights {
			weights[i].Weight = weights[i].CV / totalCV
		}
	}

	return weights
}

// IsRegression checks whether the current diff result represents a regression
// relative to the baseline statistics and adjusted thresholds.
func IsRegression(current diff.DiffResult, stats BaselineStats, weights []ComponentWeight) (bool, string) {
	values := map[string]float64{
		"tool_score": current.ToolDiff.Score,
		"text_score": current.TextDiff.Score,
		"step_delta": float64(current.StepsDiff.Delta),
	}

	uppers := map[string]float64{
		"tool_score": stats.ToolScore.Upper,
		"text_score": stats.TextScore.Upper,
		"step_delta": stats.StepDelta.Upper,
	}

	configuredThresholds := map[string]float64{}
	for _, w := range weights {
		configuredThresholds[w.Name] = 0
	}

	for _, w := range weights {
		val := values[w.Name]
		upper := uppers[w.Name]

		if val > w.Threshold && val > upper {
			// Determine the configured (unadjusted) threshold for the reason string.
			cvLabel := cvCategory(w.CV)
			return true, fmt.Sprintf(
				"%s %.2f > effective threshold %.2f (configured %.2f, %s CV=%.2f)",
				w.Name, val, w.Threshold, configuredThreshold(w), cvLabel, w.CV,
			)
		}
	}

	return false, ""
}

// configuredThreshold reverse-computes the original threshold from the adjusted one.
func configuredThreshold(w ComponentWeight) float64 {
	switch {
	case w.CV < 0.1:
		return w.Threshold / 0.8
	case w.CV > 0.5:
		return w.Threshold / 1.3
	default:
		return w.Threshold
	}
}

// cvCategory returns a human-readable label for the CV-based adjustment.
func cvCategory(cv float64) string {
	switch {
	case cv < 0.1:
		return "tightened due to low variance"
	case cv > 0.5:
		return "relaxed due to high variance"
	default:
		return "normal variance"
	}
}

// computeCV estimates the coefficient of variation from a BootstrapResult.
// Uses the CI width as a proxy for standard deviation.
func computeCV(r BootstrapResult) float64 {
	if r.Mean == 0 || r.SampleSize < 2 {
		return 0
	}
	// Approximate std from the CI: for a 95% CI, width ~ 4*std/sqrt(n).
	// But since we have the bootstrap CI directly, use half-width / 1.96 * sqrt(n)
	// as an approximation of the population std.
	halfWidth := (r.Upper - r.Lower) / 2.0
	if halfWidth <= 0 {
		return 0
	}
	// For a normal distribution, 95% CI half-width ~ 1.96 * std / sqrt(n).
	// So std ~ halfWidth * sqrt(n) / 1.96.
	approxStd := halfWidth * math.Sqrt(float64(r.SampleSize)) / 1.96
	return approxStd / math.Abs(r.Mean)
}
