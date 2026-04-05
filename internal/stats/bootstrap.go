package stats

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/jtsilverman/agentdiff/internal/diff"
)

// BootstrapResult holds the mean and confidence interval from bootstrap resampling.
type BootstrapResult struct {
	Mean       float64 `json:"mean"`
	Lower      float64 `json:"lower"`      // lower percentile bound
	Upper      float64 `json:"upper"`      // upper percentile bound
	SampleSize int     `json:"sample_size"`
}

// BaselineStats holds bootstrap results for each diff component.
type BaselineStats struct {
	ToolScore BootstrapResult `json:"tool_score"`
	TextScore BootstrapResult `json:"text_score"`
	StepDelta BootstrapResult `json:"step_delta"`
}

// Bootstrap performs bootstrap resampling to estimate the confidence interval
// of the sample mean. confidence must be in (0.0, 1.0). B is the number of
// bootstrap resamples. seed controls the random number generator for
// reproducibility.
func Bootstrap(samples []float64, confidence float64, B int, seed int64) (BootstrapResult, error) {
	if confidence <= 0.0 || confidence >= 1.0 {
		return BootstrapResult{}, fmt.Errorf("confidence must be in (0.0, 1.0), got %f", confidence)
	}

	n := len(samples)
	if n == 0 {
		return BootstrapResult{}, nil
	}

	mean := sampleMean(samples)

	if n == 1 {
		return BootstrapResult{
			Mean:       mean,
			Lower:      mean,
			Upper:      mean,
			SampleSize: 1,
		}, nil
	}

	rng := rand.New(rand.NewSource(seed))
	means := make([]float64, B)

	for i := 0; i < B; i++ {
		sum := 0.0
		for j := 0; j < n; j++ {
			sum += samples[rng.Intn(n)]
		}
		means[i] = sum / float64(n)
	}

	sort.Float64s(means)

	alpha := 1.0 - confidence
	lowerIdx := int(math.Floor(alpha / 2.0 * float64(B)))
	upperIdx := int(math.Floor((1.0 - alpha/2.0) * float64(B)))

	if lowerIdx < 0 {
		lowerIdx = 0
	}
	if upperIdx >= B {
		upperIdx = B - 1
	}

	return BootstrapResult{
		Mean:       mean,
		Lower:      means[lowerIdx],
		Upper:      means[upperIdx],
		SampleSize: n,
	}, nil
}

// ComputeBaselineStats computes bootstrap statistics for each diff component
// from a slice of DiffResults.
func ComputeBaselineStats(diffs []diff.DiffResult, confidence float64) BaselineStats {
	toolScores := make([]float64, len(diffs))
	textScores := make([]float64, len(diffs))
	stepDeltas := make([]float64, len(diffs))

	for i, d := range diffs {
		toolScores[i] = d.ToolDiff.Score
		textScores[i] = d.TextDiff.Score
		stepDeltas[i] = float64(d.StepsDiff.Delta)
	}

	seed := int64(len(diffs))
	const B = 10000

	toolResult, _ := Bootstrap(toolScores, confidence, B, seed)
	textResult, _ := Bootstrap(textScores, confidence, B, seed+1)
	stepResult, _ := Bootstrap(stepDeltas, confidence, B, seed+2)

	return BaselineStats{
		ToolScore: toolResult,
		TextScore: textResult,
		StepDelta: stepResult,
	}
}

func sampleMean(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range samples {
		sum += v
	}
	return sum / float64(len(samples))
}
