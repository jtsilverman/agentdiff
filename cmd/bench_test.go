package cmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchDefaultRun(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "bench")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
	}
	if stdout == "" {
		t.Fatal("expected non-empty table output")
	}
}

func TestBenchJSONOutput(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "bench", "--json")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Should have top-level result keys.
	if len(result) == 0 {
		t.Fatal("expected non-empty JSON result")
	}
}

func TestBenchSeedReproducibility(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout1, _, _ := runAgentDiff(t, workDir, "bench", "--json", "--seed", "123")
	stdout2, _, _ := runAgentDiff(t, workDir, "bench", "--json", "--seed", "123")

	if stdout1 != stdout2 {
		t.Fatal("expected identical output with same seed")
	}
}

func TestBenchDifferentSeeds(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout1, _, _ := runAgentDiff(t, workDir, "bench", "--json", "--seed", "1")
	stdout2, _, _ := runAgentDiff(t, workDir, "bench", "--json", "--seed", "999")

	if stdout1 == stdout2 {
		t.Fatal("expected different output with different seeds")
	}
}

func TestBenchOutputFile(t *testing.T) {
	workDir := makeWorkDir(t)
	outPath := filepath.Join(workDir, "bench-results.json")

	_, stderr, exitCode := runAgentDiff(t, workDir, "bench", "--output", outPath)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("output file is not valid JSON: %v", err)
	}
}

func TestBenchTableOutput(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, _, exitCode := runAgentDiff(t, workDir, "bench")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	// Table output should have recognizable structure.
	lower := strings.ToLower(stdout)
	hasMetricKeyword := strings.Contains(lower, "precision") ||
		strings.Contains(lower, "recall") ||
		strings.Contains(lower, "f1") ||
		strings.Contains(lower, "accuracy") ||
		strings.Contains(lower, "threshold") ||
		strings.Contains(lower, "score")
	if !hasMetricKeyword {
		t.Fatalf("expected table output to contain metric keywords, got:\n%s", stdout)
	}
}
