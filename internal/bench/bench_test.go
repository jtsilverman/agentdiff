package bench

import (
	"encoding/json"
	"testing"
)

func TestRunBench(t *testing.T) {
	result := Run(42, false)

	// Detection metrics in [0,1].
	if result.Detection.Precision < 0 || result.Detection.Precision > 1 {
		t.Errorf("Precision out of range: %f", result.Detection.Precision)
	}
	if result.Detection.Recall < 0 || result.Detection.Recall > 1 {
		t.Errorf("Recall out of range: %f", result.Detection.Recall)
	}
	if result.Detection.F1 < 0 || result.Detection.F1 > 1 {
		t.Errorf("F1 out of range: %f", result.Detection.F1)
	}
	if result.Detection.Accuracy < 0 || result.Detection.Accuracy > 1 {
		t.Errorf("Accuracy out of range: %f", result.Detection.Accuracy)
	}

	// Threshold AUCs in [0,1].
	if result.ToolThreshold.AUC < 0 || result.ToolThreshold.AUC > 1 {
		t.Errorf("Tool AUC out of range: %f", result.ToolThreshold.AUC)
	}
	if result.TextThreshold.AUC < 0 || result.TextThreshold.AUC > 1 {
		t.Errorf("Text AUC out of range: %f", result.TextThreshold.AUC)
	}
	if result.StepThreshold.AUC < 0 || result.StepThreshold.AUC > 1 {
		t.Errorf("Step AUC out of range: %f", result.StepThreshold.AUC)
	}

	// Clustering ARI >= -0.5.
	if result.Clustering.ARI < -0.5 {
		t.Errorf("ARI too low: %f", result.Clustering.ARI)
	}

	// Cross-validation metrics.
	if result.CrossVal.MeanF1 < 0 || result.CrossVal.MeanF1 > 1 {
		t.Errorf("MeanF1 out of range: %f", result.CrossVal.MeanF1)
	}
	if result.CrossVal.StdF1 < 0 {
		t.Errorf("StdF1 negative: %f", result.CrossVal.StdF1)
	}

	// FormatTable returns non-empty string.
	table := FormatTable(result, false)
	if len(table) == 0 {
		t.Error("FormatTable returned empty string")
	}

	tableVerbose := FormatTable(result, true)
	if len(tableVerbose) == 0 {
		t.Error("FormatTable verbose returned empty string")
	}

	// FormatJSON returns valid JSON.
	data, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}
	var parsed BenchResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("FormatJSON output is not valid JSON: %v", err)
	}
	if parsed.Seed != 42 {
		t.Errorf("JSON round-trip seed mismatch: got %d, want 42", parsed.Seed)
	}
}
