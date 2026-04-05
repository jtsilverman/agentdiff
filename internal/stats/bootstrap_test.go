package stats

import (
	"math"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/diff"
)

func TestBootstrap_AllSameSamples(t *testing.T) {
	samples := []float64{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
	result, err := Bootstrap(samples, 0.95, 10000, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Mean != 0.5 {
		t.Errorf("expected mean 0.5, got %f", result.Mean)
	}

	ciWidth := result.Upper - result.Lower
	if ciWidth > 0.001 {
		t.Errorf("expected CI width near 0 for constant samples, got %f", ciWidth)
	}

	if result.SampleSize != 10 {
		t.Errorf("expected sample size 10, got %d", result.SampleSize)
	}
}

func TestBootstrap_HighVarianceSamples(t *testing.T) {
	samples := []float64{0.0, 0.1, 0.9, 1.0, 0.0, 1.0, 0.05, 0.95, 0.5, 0.5}
	result, err := Bootstrap(samples, 0.95, 10000, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ciWidth := result.Upper - result.Lower
	if ciWidth < 0.1 {
		t.Errorf("expected wide CI for high-variance samples, got width %f", ciWidth)
	}
}

func TestBootstrap_CIContainsMean(t *testing.T) {
	// Statistical test: run 100 times with different seeds, verify containment > 90%.
	samples := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}
	contained := 0

	for seed := int64(0); seed < 100; seed++ {
		result, err := Bootstrap(samples, 0.95, 10000, seed)
		if err != nil {
			t.Fatalf("unexpected error at seed %d: %v", seed, err)
		}
		if result.Mean >= result.Lower && result.Mean <= result.Upper {
			contained++
		}
	}

	if contained < 90 {
		t.Errorf("expected mean contained in CI >90%% of the time, got %d/100", contained)
	}
}

func TestBootstrap_ConfidenceValidation(t *testing.T) {
	samples := []float64{1.0, 2.0, 3.0}

	_, err := Bootstrap(samples, 1.5, 1000, 42)
	if err == nil {
		t.Error("expected error for confidence=1.5, got nil")
	}

	_, err = Bootstrap(samples, -0.3, 1000, 42)
	if err == nil {
		t.Error("expected error for confidence=-0.3, got nil")
	}

	_, err = Bootstrap(samples, 0.0, 1000, 42)
	if err == nil {
		t.Error("expected error for confidence=0.0, got nil")
	}

	_, err = Bootstrap(samples, 1.0, 1000, 42)
	if err == nil {
		t.Error("expected error for confidence=1.0, got nil")
	}

	result, err := Bootstrap(samples, 0.95, 1000, 42)
	if err != nil {
		t.Errorf("expected no error for confidence=0.95, got %v", err)
	}
	if result.SampleSize != 3 {
		t.Errorf("expected sample size 3, got %d", result.SampleSize)
	}
}

func TestBootstrap_EmptySamples(t *testing.T) {
	result, err := Bootstrap([]float64{}, 0.95, 10000, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Mean != 0 || result.Lower != 0 || result.Upper != 0 {
		t.Errorf("expected zero result for empty samples, got mean=%f lower=%f upper=%f",
			result.Mean, result.Lower, result.Upper)
	}
	if result.SampleSize != 0 {
		t.Errorf("expected sample size 0, got %d", result.SampleSize)
	}
}

func TestBootstrap_SingleSample(t *testing.T) {
	result, err := Bootstrap([]float64{3.14}, 0.95, 10000, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Mean != 3.14 {
		t.Errorf("expected mean 3.14, got %f", result.Mean)
	}
	if result.Lower != 3.14 {
		t.Errorf("expected lower 3.14, got %f", result.Lower)
	}
	if result.Upper != 3.14 {
		t.Errorf("expected upper 3.14, got %f", result.Upper)
	}
	if result.SampleSize != 1 {
		t.Errorf("expected sample size 1, got %d", result.SampleSize)
	}
}

func TestComputeBaselineStats(t *testing.T) {
	diffs := []diff.DiffResult{
		{ToolDiff: diff.ToolDiffResult{Score: 0.1}, TextDiff: diff.TextDiffResult{Score: 0.2}, StepsDiff: diff.StepsDiffResult{Delta: 1}},
		{ToolDiff: diff.ToolDiffResult{Score: 0.15}, TextDiff: diff.TextDiffResult{Score: 0.25}, StepsDiff: diff.StepsDiffResult{Delta: 2}},
		{ToolDiff: diff.ToolDiffResult{Score: 0.12}, TextDiff: diff.TextDiffResult{Score: 0.22}, StepsDiff: diff.StepsDiffResult{Delta: 3}},
	}

	stats := ComputeBaselineStats(diffs, 0.95)

	// Verify means are approximately correct.
	expectedToolMean := (0.1 + 0.15 + 0.12) / 3.0
	if math.Abs(stats.ToolScore.Mean-expectedToolMean) > 0.001 {
		t.Errorf("expected tool score mean ~%f, got %f", expectedToolMean, stats.ToolScore.Mean)
	}

	expectedTextMean := (0.2 + 0.25 + 0.22) / 3.0
	if math.Abs(stats.TextScore.Mean-expectedTextMean) > 0.001 {
		t.Errorf("expected text score mean ~%f, got %f", expectedTextMean, stats.TextScore.Mean)
	}

	expectedStepMean := (1.0 + 2.0 + 3.0) / 3.0
	if math.Abs(stats.StepDelta.Mean-expectedStepMean) > 0.001 {
		t.Errorf("expected step delta mean ~%f, got %f", expectedStepMean, stats.StepDelta.Mean)
	}

	// CI should contain the mean.
	if stats.ToolScore.Lower > stats.ToolScore.Mean || stats.ToolScore.Upper < stats.ToolScore.Mean {
		t.Errorf("tool score CI [%f, %f] does not contain mean %f",
			stats.ToolScore.Lower, stats.ToolScore.Upper, stats.ToolScore.Mean)
	}
}
