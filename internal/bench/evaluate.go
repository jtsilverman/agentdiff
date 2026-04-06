package bench

import (
	"math"
	"math/rand/v2"
	"sort"

	"github.com/jtsilverman/agentdiff/internal/cluster"
	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// DetectionResult holds confusion matrix metrics for regression detection.
type DetectionResult struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	Accuracy  float64 `json:"accuracy"`
	TP        int     `json:"true_positives"`
	FP        int     `json:"false_positives"`
	FN        int     `json:"false_negatives"`
	TN        int     `json:"true_negatives"`
}

// ROCPoint represents a single point on an ROC curve.
type ROCPoint struct {
	Threshold float64 `json:"threshold"`
	TPR       float64 `json:"tpr"`
	FPR       float64 `json:"fpr"`
	F1        float64 `json:"f1"`
}

// ThresholdResult holds the result of sweeping a threshold dimension.
type ThresholdResult struct {
	Dimension    string     `json:"dimension"`
	ROC          []ROCPoint `json:"roc"`
	AUC          float64    `json:"auc"`
	OptimalPoint ROCPoint   `json:"optimal"`
}

// ClusterResult holds evaluation metrics for clustering quality.
type ClusterResult struct {
	ARI           float64 `json:"ari"`
	NumClusters   int     `json:"num_clusters"`
	NumStrategies int     `json:"num_strategies"`
	NoiseCount    int     `json:"noise_count"`
}

// CrossValResult holds cross-validation results.
type CrossValResult struct {
	Folds       int     `json:"folds"`
	MeanF1      float64 `json:"mean_f1"`
	StdF1       float64 `json:"std_f1"`
	OptimalTool float64 `json:"optimal_tool_threshold"`
	OptimalText float64 `json:"optimal_text_threshold"`
	OptimalStep int     `json:"optimal_step_threshold"`
}

// EvaluateDetection runs diff.Compare on each pair and computes confusion matrix metrics.
func EvaluateDetection(pairs []LabeledPair, cfg config.Config) DetectionResult {
	var tp, fp, fn, tn int

	for _, p := range pairs {
		result := diff.Compare(p.Baseline, p.Candidate, cfg)
		predicted := result.Overall == diff.VerdictRegression

		switch {
		case predicted && p.IsRegression:
			tp++
		case predicted && !p.IsRegression:
			fp++
		case !predicted && p.IsRegression:
			fn++
		default:
			tn++
		}
	}

	var precision, recall, f1 float64
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	total := tp + fp + fn + tn
	var accuracy float64
	if total > 0 {
		accuracy = float64(tp+tn) / float64(total)
	}

	return DetectionResult{
		Precision: precision,
		Recall:    recall,
		F1:        f1,
		Accuracy:  accuracy,
		TP:        tp,
		FP:        fp,
		FN:        fn,
		TN:        tn,
	}
}

// SweepThreshold sweeps a threshold dimension and computes ROC points.
func SweepThreshold(pairs []LabeledPair, dimension string, min, max, step float64) ThresholdResult {
	var points []ROCPoint

	for threshold := min; threshold <= max+step/2; threshold += step {
		var tp, fp, fn, tn int

		for _, p := range pairs {
			var predicted bool

			switch dimension {
			case "tool":
				toolResult, _ := diff.CompareToolsWithDiagnostics(p.Baseline.Steps, p.Candidate.Steps)
				predicted = toolResult.Score > threshold
			case "text":
				textResult := diff.CompareText(p.Baseline.Steps, p.Candidate.Steps)
				predicted = textResult.Score > threshold
			case "step":
				stepResult := diff.CompareSteps(p.Baseline.Steps, p.Candidate.Steps)
				predicted = stepResult.Delta > int(threshold)
			}

			switch {
			case predicted && p.IsRegression:
				tp++
			case predicted && !p.IsRegression:
				fp++
			case !predicted && p.IsRegression:
				fn++
			default:
				tn++
			}
		}

		var tpr, fpr, f1 float64
		if tp+fn > 0 {
			tpr = float64(tp) / float64(tp+fn)
		}
		if fp+tn > 0 {
			fpr = float64(fp) / float64(fp+tn)
		}
		precision := 0.0
		if tp+fp > 0 {
			precision = float64(tp) / float64(tp+fp)
		}
		if precision+tpr > 0 {
			f1 = 2 * precision * tpr / (precision + tpr)
		}

		points = append(points, ROCPoint{
			Threshold: threshold,
			TPR:       tpr,
			FPR:       fpr,
			F1:        f1,
		})
	}

	// Find optimal point (max F1).
	optimal := ROCPoint{}
	for _, pt := range points {
		if pt.F1 > optimal.F1 {
			optimal = pt
		}
	}

	auc := ComputeAUC(points)

	return ThresholdResult{
		Dimension:    dimension,
		ROC:          points,
		AUC:          auc,
		OptimalPoint: optimal,
	}
}

// ComputeAUC computes the area under the ROC curve using the trapezoidal rule.
func ComputeAUC(points []ROCPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	// Sort by FPR ascending.
	sorted := make([]ROCPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].FPR < sorted[j].FPR
	})

	auc := 0.0
	for i := 0; i < len(sorted)-1; i++ {
		auc += (sorted[i+1].FPR - sorted[i].FPR) * (sorted[i+1].TPR + sorted[i].TPR) / 2
	}
	return auc
}

// EvaluateClustering runs DBSCAN clustering and evaluates against ground truth strategy labels.
func EvaluateClustering(traces []LabeledTrace, epsilon float64, minPts int) ClusterResult {
	snapshots := make([]snapshot.Snapshot, len(traces))
	for i, lt := range traces {
		snapshots[i] = lt.Trace
	}

	baseline := snapshot.Baseline{
		Name:      "bench",
		Snapshots: snapshots,
	}

	report, err := cluster.ClusterBaseline(baseline, epsilon, minPts)
	if err != nil {
		return ClusterResult{
			ARI:           0.0,
			NumClusters:   0,
			NumStrategies: countUnique(traces),
			NoiseCount:    len(traces),
		}
	}

	// Build name-to-cluster mapping.
	nameToCluster := make(map[string]int)
	for _, strat := range report.Strategies {
		for _, member := range strat.Members {
			nameToCluster[member] = strat.ID
		}
	}
	for _, noiseName := range report.Noise {
		nameToCluster[noiseName] = -1
	}

	// Build predicted and truth label arrays.
	predicted := make([]int, len(traces))
	truth := make([]int, len(traces))
	for i, lt := range traces {
		if clusterID, ok := nameToCluster[lt.Trace.Name]; ok {
			predicted[i] = clusterID
		} else {
			predicted[i] = -1
		}
		truth[i] = lt.StrategyID
	}

	ari := AdjustedRandIndex(predicted, truth)

	return ClusterResult{
		ARI:           ari,
		NumClusters:   len(report.Strategies),
		NumStrategies: countUnique(traces),
		NoiseCount:    len(report.Noise),
	}
}

// countUnique returns the number of unique strategy IDs in a trace set.
func countUnique(traces []LabeledTrace) int {
	seen := make(map[int]bool)
	for _, lt := range traces {
		seen[lt.StrategyID] = true
	}
	return len(seen)
}

// choose2 computes n*(n-1)/2.
func choose2(n int) float64 {
	return float64(n) * float64(n-1) / 2.0
}

// AdjustedRandIndex computes the ARI between two label vectors.
func AdjustedRandIndex(predicted, truth []int) float64 {
	n := len(predicted)
	if n == 0 {
		return 0.0
	}

	// Edge case: all same cluster.
	allSamePred := true
	allSameTruth := true
	for i := 1; i < n; i++ {
		if predicted[i] != predicted[0] {
			allSamePred = false
		}
		if truth[i] != truth[0] {
			allSameTruth = false
		}
	}
	if allSamePred && allSameTruth {
		return 1.0
	}
	if allSamePred || allSameTruth {
		return 0.0
	}

	// All singletons check.
	predSet := make(map[int]bool)
	for _, v := range predicted {
		predSet[v] = true
	}
	truthSet := make(map[int]bool)
	for _, v := range truth {
		truthSet[v] = true
	}
	if len(predSet) == n && len(truthSet) == n {
		return 1.0
	}
	if len(predSet) == n || len(truthSet) == n {
		return 0.0
	}

	// Build contingency table.
	contingency := make(map[int]map[int]int)
	for i := 0; i < n; i++ {
		p := predicted[i]
		t := truth[i]
		if contingency[p] == nil {
			contingency[p] = make(map[int]int)
		}
		contingency[p][t]++
	}

	// Row sums (predicted clusters).
	rowSums := make(map[int]int)
	for p, cols := range contingency {
		for _, count := range cols {
			rowSums[p] += count
		}
	}

	// Col sums (truth clusters).
	colSums := make(map[int]int)
	for _, cols := range contingency {
		for t, count := range cols {
			colSums[t] += count
		}
	}

	// Index = sum of nij choose 2.
	index := 0.0
	for _, cols := range contingency {
		for _, nij := range cols {
			index += choose2(nij)
		}
	}

	// sum(ai choose 2).
	sumA := 0.0
	for _, ai := range rowSums {
		sumA += choose2(ai)
	}

	// sum(bj choose 2).
	sumB := 0.0
	for _, bj := range colSums {
		sumB += choose2(bj)
	}

	nC2 := choose2(n)

	expected := sumA * sumB / nC2
	maxIndex := (sumA + sumB) / 2.0

	if maxIndex == expected {
		if index == expected {
			return 1.0
		}
		return 0.0
	}

	return (index - expected) / (maxIndex - expected)
}

// CrossValidate runs stratified K-fold cross-validation on labeled pairs.
func CrossValidate(pairs []LabeledPair, folds int, rng *rand.Rand) CrossValResult {
	if folds < 2 {
		folds = 2
	}

	// Stratified split: group by (IsRegression, MutationType), then by Baseline.ID.
	type stratumKey struct {
		IsRegression bool
		MutationType MutationType
	}

	// Group pairs by stratum, then by baseline ID within each stratum.
	type baselineGroup struct {
		baselineID string
		indices    []int // indices into pairs
	}

	strata := make(map[stratumKey]map[string]*baselineGroup)
	for i, p := range pairs {
		sk := stratumKey{IsRegression: p.IsRegression, MutationType: p.MutationType}
		if strata[sk] == nil {
			strata[sk] = make(map[string]*baselineGroup)
		}
		bg := strata[sk][p.Baseline.ID]
		if bg == nil {
			bg = &baselineGroup{baselineID: p.Baseline.ID}
			strata[sk][p.Baseline.ID] = bg
		}
		bg.indices = append(bg.indices, i)
	}

	// Build fold assignments: distribute baseline-groups round-robin across folds.
	foldAssignment := make([]int, len(pairs))
	foldCounter := 0
	for _, baselineGroups := range strata {
		// Collect groups into a slice for deterministic iteration.
		groups := make([]*baselineGroup, 0, len(baselineGroups))
		for _, bg := range baselineGroups {
			groups = append(groups, bg)
		}
		// Shuffle within stratum for randomness.
		rng.Shuffle(len(groups), func(i, j int) { groups[i], groups[j] = groups[j], groups[i] })

		for _, bg := range groups {
			fold := foldCounter % folds
			for _, idx := range bg.indices {
				foldAssignment[idx] = fold
			}
			foldCounter++
		}
	}

	f1s := make([]float64, folds)
	optTools := make([]float64, folds)
	optTexts := make([]float64, folds)
	optSteps := make([]int, folds)

	for fold := 0; fold < folds; fold++ {
		var train, test []LabeledPair
		for i, p := range pairs {
			if foldAssignment[i] == fold {
				test = append(test, p)
			} else {
				train = append(train, p)
			}
		}

		if len(test) == 0 || len(train) == 0 {
			continue
		}

		// Sweep on train set to find optimal thresholds.
		toolResult := SweepThreshold(train, "tool", 0, 1, 0.05)
		textResult := SweepThreshold(train, "text", 0, 1, 0.05)
		stepResult := SweepThreshold(train, "step", 0, 20, 1)

		optTools[fold] = toolResult.OptimalPoint.Threshold
		optTexts[fold] = textResult.OptimalPoint.Threshold
		optSteps[fold] = int(stepResult.OptimalPoint.Threshold)

		// Evaluate on test set using optimal thresholds.
		testCfg := config.Config{
			Thresholds: config.Thresholds{
				ToolScore: optTools[fold],
				TextScore: optTexts[fold],
				StepDelta: optSteps[fold],
			},
		}
		det := EvaluateDetection(test, testCfg)
		f1s[fold] = det.F1
	}

	// Compute mean and std of F1.
	meanF1 := 0.0
	for _, f := range f1s {
		meanF1 += f
	}
	meanF1 /= float64(folds)

	variance := 0.0
	for _, f := range f1s {
		d := f - meanF1
		variance += d * d
	}
	variance /= float64(folds)
	stdF1 := math.Sqrt(variance)

	// Average optimal thresholds.
	avgTool := 0.0
	avgText := 0.0
	avgStep := 0.0
	for i := 0; i < folds; i++ {
		avgTool += optTools[i]
		avgText += optTexts[i]
		avgStep += float64(optSteps[i])
	}
	avgTool /= float64(folds)
	avgText /= float64(folds)
	avgStep /= float64(folds)

	return CrossValResult{
		Folds:       folds,
		MeanF1:      meanF1,
		StdF1:       stdF1,
		OptimalTool: avgTool,
		OptimalText: avgText,
		OptimalStep: int(math.Round(avgStep)),
	}
}
