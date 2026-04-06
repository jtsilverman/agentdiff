package bench

import (
	"math/rand/v2"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/config"
)

func TestEvaluateDetection(t *testing.T) {
	cfg := DefaultConfig()
	pairs := GenerateLabeledPairs(cfg)

	result := EvaluateDetection(pairs, defaultDiffConfig())

	if result.Precision < 0 || result.Precision > 1 {
		t.Errorf("precision out of range [0,1]: %f", result.Precision)
	}
	if result.Recall < 0 || result.Recall > 1 {
		t.Errorf("recall out of range [0,1]: %f", result.Recall)
	}
	if result.F1 < 0 || result.F1 > 1 {
		t.Errorf("F1 out of range [0,1]: %f", result.F1)
	}
	if result.Accuracy < 0 || result.Accuracy > 1 {
		t.Errorf("accuracy out of range [0,1]: %f", result.Accuracy)
	}

	total := result.TP + result.FP + result.FN + result.TN
	if total != len(pairs) {
		t.Errorf("confusion matrix total %d != pair count %d", total, len(pairs))
	}

	t.Logf("Detection: P=%.3f R=%.3f F1=%.3f Acc=%.3f (TP=%d FP=%d FN=%d TN=%d)",
		result.Precision, result.Recall, result.F1, result.Accuracy,
		result.TP, result.FP, result.FN, result.TN)
}

func TestEvaluateSweepThreshold(t *testing.T) {
	cfg := DefaultConfig()
	pairs := GenerateLabeledPairs(cfg)

	result := SweepThreshold(pairs, "tool", 0, 1, 0.05)

	if len(result.ROC) < 2 {
		t.Errorf("expected multiple ROC points, got %d", len(result.ROC))
	}
	if result.AUC < 0 || result.AUC > 1 {
		t.Errorf("AUC out of range [0,1]: %f", result.AUC)
	}
	if result.Dimension != "tool" {
		t.Errorf("expected dimension 'tool', got %s", result.Dimension)
	}

	// Verify optimal point has max F1.
	for _, pt := range result.ROC {
		if pt.F1 > result.OptimalPoint.F1+1e-9 {
			t.Errorf("found ROC point with F1=%.4f > optimal F1=%.4f", pt.F1, result.OptimalPoint.F1)
		}
	}

	t.Logf("Tool sweep: AUC=%.3f OptimalThreshold=%.3f OptimalF1=%.3f",
		result.AUC, result.OptimalPoint.Threshold, result.OptimalPoint.F1)
}

func TestEvaluateAUC(t *testing.T) {
	// Perfect classifier: TPR=1 at all FPR values.
	perfect := []ROCPoint{
		{FPR: 0.0, TPR: 1.0},
		{FPR: 0.5, TPR: 1.0},
		{FPR: 1.0, TPR: 1.0},
	}
	auc := ComputeAUC(perfect)
	if auc < 0.99 || auc > 1.01 {
		t.Errorf("perfect classifier AUC should be ~1.0, got %f", auc)
	}

	// Random classifier: TPR = FPR (diagonal).
	random := []ROCPoint{
		{FPR: 0.0, TPR: 0.0},
		{FPR: 0.25, TPR: 0.25},
		{FPR: 0.5, TPR: 0.5},
		{FPR: 0.75, TPR: 0.75},
		{FPR: 1.0, TPR: 1.0},
	}
	auc = ComputeAUC(random)
	if auc < 0.49 || auc > 0.51 {
		t.Errorf("random classifier AUC should be ~0.5, got %f", auc)
	}

	// Single point: AUC = 0.
	single := []ROCPoint{{FPR: 0.5, TPR: 0.5}}
	auc = ComputeAUC(single)
	if auc != 0.0 {
		t.Errorf("single point AUC should be 0.0, got %f", auc)
	}
}

func TestEvaluateARI(t *testing.T) {
	// Perfect match.
	pred := []int{0, 0, 1, 1, 2, 2}
	truth := []int{0, 0, 1, 1, 2, 2}
	ari := AdjustedRandIndex(pred, truth)
	if ari < 0.99 {
		t.Errorf("perfect match ARI should be ~1.0, got %f", ari)
	}

	// All same cluster, truth has multiple clusters.
	pred = []int{0, 0, 0, 0, 0, 0}
	truth = []int{0, 0, 1, 1, 2, 2}
	ari = AdjustedRandIndex(pred, truth)
	if ari > 0.01 || ari < -0.01 {
		t.Errorf("all-same-pred ARI should be ~0.0, got %f", ari)
	}

	// All same cluster, truth also all same.
	pred = []int{5, 5, 5, 5}
	truth = []int{3, 3, 3, 3}
	ari = AdjustedRandIndex(pred, truth)
	if ari != 1.0 {
		t.Errorf("both all-same ARI should be 1.0, got %f", ari)
	}

	// Permuted labels should still be perfect.
	pred = []int{1, 1, 2, 2, 0, 0}
	truth = []int{0, 0, 1, 1, 2, 2}
	ari = AdjustedRandIndex(pred, truth)
	if ari < 0.99 {
		t.Errorf("permuted labels ARI should be ~1.0, got %f", ari)
	}

	// Both all singletons (identical partitions).
	pred = []int{0, 1, 2, 3}
	truth = []int{4, 5, 6, 7}
	ari = AdjustedRandIndex(pred, truth)
	if ari != 1.0 {
		t.Errorf("both all-singletons ARI should be 1.0, got %f", ari)
	}

	// Only predicted is singletons, truth has clusters.
	pred = []int{0, 1, 2, 3, 4, 5}
	truth = []int{0, 0, 1, 1, 2, 2}
	ari = AdjustedRandIndex(pred, truth)
	if ari > 0.01 {
		t.Errorf("pred-singletons ARI should be ~0.0, got %f", ari)
	}

	// Empty.
	ari = AdjustedRandIndex([]int{}, []int{})
	if ari != 0.0 {
		t.Errorf("empty ARI should be 0.0, got %f", ari)
	}
}

func TestEvaluateClustering(t *testing.T) {
	cfg := DefaultConfig()
	traces := GenerateStrategyTraces(cfg)

	result := EvaluateClustering(traces, 0, 2)

	if result.ARI < -0.5 {
		t.Errorf("ARI too low: %f", result.ARI)
	}
	if result.NumStrategies != cfg.NumStrategies {
		t.Errorf("expected %d strategies, got %d", cfg.NumStrategies, result.NumStrategies)
	}

	t.Logf("Clustering: ARI=%.3f Clusters=%d Strategies=%d Noise=%d",
		result.ARI, result.NumClusters, result.NumStrategies, result.NoiseCount)
}

func TestEvaluateCrossValidate(t *testing.T) {
	cfg := DefaultConfig()
	pairs := GenerateLabeledPairs(cfg)
	rng := rand.New(rand.NewPCG(42, 0))

	result := CrossValidate(pairs, 5, rng)

	if result.Folds != 5 {
		t.Errorf("expected 5 folds, got %d", result.Folds)
	}
	if result.MeanF1 < 0 || result.MeanF1 > 1 {
		t.Errorf("MeanF1 out of range [0,1]: %f", result.MeanF1)
	}
	if result.StdF1 < 0 {
		t.Errorf("StdF1 should be >= 0, got %f", result.StdF1)
	}

	t.Logf("CrossVal: MeanF1=%.3f StdF1=%.3f OptTool=%.3f OptText=%.3f OptStep=%d",
		result.MeanF1, result.StdF1, result.OptimalTool, result.OptimalText, result.OptimalStep)
}

// defaultDiffConfig returns the default config for diff.Compare.
func defaultDiffConfig() config.Config {
	return config.DefaultConfig()
}
