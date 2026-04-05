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
