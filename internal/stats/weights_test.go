package stats

import (
	"math"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
)

func TestComputeWeights_LowCV(t *testing.T) {
	// All-same samples produce CV near 0 (< 0.1), so threshold should be tightened by 20%.
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.5, Lower: 0.5, Upper: 0.5, SampleSize: 10},
		TextScore: BootstrapResult{Mean: 0.5, Lower: 0.5, Upper: 0.5, SampleSize: 10},
		StepDelta: BootstrapResult{Mean: 5.0, Lower: 5.0, Upper: 5.0, SampleSize: 10},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	for _, w := range weights {
		if w.CV >= 0.1 {
			t.Errorf("%s: expected CV < 0.1, got %f", w.Name, w.CV)
		}
	}

	// tool_score threshold should be 0.3 * 0.8 = 0.24.
	toolWeight := findWeight(weights, "tool_score")
	if toolWeight == nil {
		t.Fatal("tool_score weight not found")
	}
	if math.Abs(toolWeight.Threshold-0.24) > 0.001 {
		t.Errorf("expected tool_score threshold 0.24 (0.3*0.8), got %f", toolWeight.Threshold)
	}

	// text_score threshold should be 0.5 * 0.8 = 0.40.
	textWeight := findWeight(weights, "text_score")
	if textWeight == nil {
		t.Fatal("text_score weight not found")
	}
	if math.Abs(textWeight.Threshold-0.40) > 0.001 {
		t.Errorf("expected text_score threshold 0.40 (0.5*0.8), got %f", textWeight.Threshold)
	}
}

func TestComputeWeights_HighCV(t *testing.T) {
	// Wide CI relative to mean produces high CV (> 0.5), so threshold should be relaxed by 30%.
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.3, Lower: 0.05, Upper: 0.55, SampleSize: 5},
		TextScore: BootstrapResult{Mean: 0.3, Lower: 0.05, Upper: 0.55, SampleSize: 5},
		StepDelta: BootstrapResult{Mean: 3.0, Lower: 0.5, Upper: 5.5, SampleSize: 5},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	toolWeight := findWeight(weights, "tool_score")
	if toolWeight == nil {
		t.Fatal("tool_score weight not found")
	}
	if toolWeight.CV <= 0.5 {
		t.Errorf("expected tool_score CV > 0.5, got %f", toolWeight.CV)
	}
	expectedThreshold := 0.3 * 1.3
	if math.Abs(toolWeight.Threshold-expectedThreshold) > 0.001 {
		t.Errorf("expected tool_score threshold %f (0.3*1.3), got %f", expectedThreshold, toolWeight.Threshold)
	}
}

func TestComputeWeights_NormalCV(t *testing.T) {
	// CV between 0.1 and 0.5: threshold should be unchanged.
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.5, Lower: 0.44, Upper: 0.56, SampleSize: 10},
		TextScore: BootstrapResult{Mean: 0.5, Lower: 0.44, Upper: 0.56, SampleSize: 10},
		StepDelta: BootstrapResult{Mean: 5.0, Lower: 4.4, Upper: 5.6, SampleSize: 10},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	for _, w := range weights {
		if w.CV < 0.1 || w.CV > 0.5 {
			t.Errorf("%s: expected normal CV (0.1-0.5), got %f", w.Name, w.CV)
		}
	}

	toolWeight := findWeight(weights, "tool_score")
	if toolWeight == nil {
		t.Fatal("tool_score weight not found")
	}
	if math.Abs(toolWeight.Threshold-0.3) > 0.001 {
		t.Errorf("expected tool_score threshold 0.3 (unchanged), got %f", toolWeight.Threshold)
	}
}

func TestComputeWeights_SumToOne(t *testing.T) {
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.3, Lower: 0.2, Upper: 0.4, SampleSize: 10},
		TextScore: BootstrapResult{Mean: 0.5, Lower: 0.3, Upper: 0.7, SampleSize: 10},
		StepDelta: BootstrapResult{Mean: 3.0, Lower: 1.0, Upper: 5.0, SampleSize: 10},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	sum := 0.0
	for _, w := range weights {
		sum += w.Weight
	}
	if math.Abs(sum-1.0) > 0.001 {
		t.Errorf("expected weights to sum to 1.0, got %f", sum)
	}
}

func TestComputeWeights_ZeroCVEqualWeights(t *testing.T) {
	// All CVs zero: weights should be equal (1/3 each).
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.5, Lower: 0.5, Upper: 0.5, SampleSize: 10},
		TextScore: BootstrapResult{Mean: 0.5, Lower: 0.5, Upper: 0.5, SampleSize: 10},
		StepDelta: BootstrapResult{Mean: 5.0, Lower: 5.0, Upper: 5.0, SampleSize: 10},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	for _, w := range weights {
		if math.Abs(w.Weight-1.0/3.0) > 0.001 {
			t.Errorf("%s: expected weight ~0.333, got %f", w.Name, w.Weight)
		}
	}
}

func TestIsRegression_AboveBothThresholds(t *testing.T) {
	// Use tight CI so CV is low/normal, and a current value clearly above both
	// the effective threshold and the upper CI bound.
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.1, Lower: 0.095, Upper: 0.105, SampleSize: 100},
		TextScore: BootstrapResult{Mean: 0.2, Lower: 0.19, Upper: 0.21, SampleSize: 100},
		StepDelta: BootstrapResult{Mean: 2.0, Lower: 1.9, Upper: 2.1, SampleSize: 100},
	}
	thresholds := config.Thresholds{ToolScore: 0.15, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	// tool_score CV is low, so threshold is tightened: 0.15 * 0.8 = 0.12.
	// Current tool_score 0.20 > 0.12 (effective threshold) AND > 0.105 (upper CI).
	current := diff.DiffResult{
		ToolDiff:  diff.ToolDiffResult{Score: 0.20},
		TextDiff:  diff.TextDiffResult{Score: 0.1},
		StepsDiff: diff.StepsDiffResult{Delta: 1},
	}

	isReg, reason := IsRegression(current, stats, weights)
	if !isReg {
		t.Error("expected regression, got false")
	}
	if reason == "" {
		t.Error("expected non-empty reason")
	}
	// Reason should mention the component name and thresholds.
	if len(reason) < 10 {
		t.Errorf("expected detailed reason, got: %s", reason)
	}
	t.Logf("Regression reason: %s", reason)
}

func TestIsRegression_BelowThresholds(t *testing.T) {
	stats := BaselineStats{
		ToolScore: BootstrapResult{Mean: 0.1, Lower: 0.05, Upper: 0.15, SampleSize: 10},
		TextScore: BootstrapResult{Mean: 0.2, Lower: 0.15, Upper: 0.25, SampleSize: 10},
		StepDelta: BootstrapResult{Mean: 2.0, Lower: 1.0, Upper: 3.0, SampleSize: 10},
	}
	thresholds := config.Thresholds{ToolScore: 0.3, TextScore: 0.5, StepDelta: 5}

	weights := ComputeWeights(stats, thresholds)

	// All values well below thresholds.
	current := diff.DiffResult{
		ToolDiff:  diff.ToolDiffResult{Score: 0.05},
		TextDiff:  diff.TextDiffResult{Score: 0.1},
		StepsDiff: diff.StepsDiffResult{Delta: 1},
	}

	isReg, reason := IsRegression(current, stats, weights)
	if isReg {
		t.Errorf("expected no regression, got true with reason: %s", reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason, got: %s", reason)
	}
}

func findWeight(weights []ComponentWeight, name string) *ComponentWeight {
	for _, w := range weights {
		if w.Name == name {
			return &w
		}
	}
	return nil
}
