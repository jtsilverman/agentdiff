package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.ToolScore != 0.3 {
		t.Errorf("tool_score: got %v, want 0.3", cfg.Thresholds.ToolScore)
	}
	if cfg.Thresholds.TextScore != 0.5 {
		t.Errorf("text_score: got %v, want 0.5", cfg.Thresholds.TextScore)
	}
	if cfg.Thresholds.StepDelta != 5 {
		t.Errorf("step_delta: got %v, want 5", cfg.Thresholds.StepDelta)
	}
	// Check new defaults.
	if cfg.CI.BaselinePath != "" {
		t.Errorf("ci.baseline_path: got %q, want empty", cfg.CI.BaselinePath)
	}
	if cfg.CI.FailOnStyleDrift != false {
		t.Errorf("ci.fail_on_style_drift: got %v, want false", cfg.CI.FailOnStyleDrift)
	}
	if cfg.Baseline.Runs != 5 {
		t.Errorf("baseline.runs: got %v, want 5", cfg.Baseline.Runs)
	}
	if cfg.Baseline.Confidence != 0.95 {
		t.Errorf("baseline.confidence: got %v, want 0.95", cfg.Baseline.Confidence)
	}
}

func TestLoadPartialFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("thresholds:\n  tool_score: 0.8\n")
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.ToolScore != 0.8 {
		t.Errorf("tool_score: got %v, want 0.8", cfg.Thresholds.ToolScore)
	}
	// Unset fields should keep defaults.
	if cfg.Thresholds.TextScore != 0.5 {
		t.Errorf("text_score: got %v, want 0.5 (default)", cfg.Thresholds.TextScore)
	}
	if cfg.Thresholds.StepDelta != 5 {
		t.Errorf("step_delta: got %v, want 5 (default)", cfg.Thresholds.StepDelta)
	}
}

func TestLoadFullFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`thresholds:
  tool_score: 0.1
  text_score: 0.2
  step_delta: 10
`)
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.ToolScore != 0.1 {
		t.Errorf("tool_score: got %v, want 0.1", cfg.Thresholds.ToolScore)
	}
	if cfg.Thresholds.TextScore != 0.2 {
		t.Errorf("text_score: got %v, want 0.2", cfg.Thresholds.TextScore)
	}
	if cfg.Thresholds.StepDelta != 10 {
		t.Errorf("step_delta: got %v, want 10", cfg.Thresholds.StepDelta)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := []byte("{{invalid yaml")
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadCIConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`ci:
  baseline_path: baselines/
  fail_on_style_drift: true
`)
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CI.BaselinePath != "baselines/" {
		t.Errorf("ci.baseline_path: got %q, want %q", cfg.CI.BaselinePath, "baselines/")
	}
	if cfg.CI.FailOnStyleDrift != true {
		t.Errorf("ci.fail_on_style_drift: got %v, want true", cfg.CI.FailOnStyleDrift)
	}
	// Thresholds should keep defaults.
	if cfg.Thresholds.ToolScore != 0.3 {
		t.Errorf("tool_score: got %v, want 0.3 (default)", cfg.Thresholds.ToolScore)
	}
}

func TestLoadBaselineConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`baseline:
  runs: 10
  confidence: 0.99
`)
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Baseline.Runs != 10 {
		t.Errorf("baseline.runs: got %v, want 10", cfg.Baseline.Runs)
	}
	if cfg.Baseline.Confidence != 0.99 {
		t.Errorf("baseline.confidence: got %v, want 0.99", cfg.Baseline.Confidence)
	}
}

func TestLoadBaselineConfidenceInvalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"zero", "baseline:\n  confidence: 0.0\n"},
		{"one", "baseline:\n  confidence: 1.0\n"},
		{"negative", "baseline:\n  confidence: -0.5\n"},
		{"above_one", "baseline:\n  confidence: 1.5\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(dir)
			if err == nil {
				t.Error("expected error for invalid confidence, got nil")
			}
		})
	}
}

func TestLoadAllSections(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`thresholds:
  tool_score: 0.5
ci:
  baseline_path: /tmp/baselines
  fail_on_style_drift: true
baseline:
  runs: 20
  confidence: 0.9
`)
	if err := os.WriteFile(filepath.Join(dir, ".agentdiff.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Thresholds.ToolScore != 0.5 {
		t.Errorf("tool_score: got %v, want 0.5", cfg.Thresholds.ToolScore)
	}
	if cfg.CI.BaselinePath != "/tmp/baselines" {
		t.Errorf("ci.baseline_path: got %q, want %q", cfg.CI.BaselinePath, "/tmp/baselines")
	}
	if cfg.CI.FailOnStyleDrift != true {
		t.Errorf("ci.fail_on_style_drift: got %v, want true", cfg.CI.FailOnStyleDrift)
	}
	if cfg.Baseline.Runs != 20 {
		t.Errorf("baseline.runs: got %v, want 20", cfg.Baseline.Runs)
	}
	if cfg.Baseline.Confidence != 0.9 {
		t.Errorf("baseline.confidence: got %v, want 0.9", cfg.Baseline.Confidence)
	}
}
