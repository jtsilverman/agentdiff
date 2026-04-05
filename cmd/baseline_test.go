package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// makeTestSnapshot creates and saves a snapshot with the given name and steps in the workDir.
func makeTestSnapshot(t *testing.T, workDir, name string, steps []snapshot.Step) snapshot.Snapshot {
	t.Helper()
	store := snapshot.NewStore(workDir)
	snap := snapshot.Snapshot{
		Name:      name,
		Source:    "test",
		Timestamp: time.Now(),
		Metadata:  map[string]string{"test": "true"},
		Steps:     steps,
	}
	saved, err := store.Save(snap)
	if err != nil {
		t.Fatalf("save snapshot %q: %v", name, err)
	}
	return saved
}

// lowScoreSteps returns steps that produce a low diff score when compared to themselves.
func lowScoreSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hello there friend how are you doing today"},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "/tmp/test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "file contents"}},
	}
}

// variantSteps returns steps with a small variation (extra step) to produce slightly
// different diff scores against a target. The variant parameter controls which extra
// content is added.
func variantSteps(variant int) []snapshot.Step {
	base := []snapshot.Step{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hello there friend how are you doing today"},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "/tmp/test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "file contents"}},
	}
	// Add variant-specific extra steps so each baseline snapshot differs slightly.
	for i := 0; i < variant; i++ {
		base = append(base, snapshot.Step{
			Role:    "assistant",
			Content: fmt.Sprintf("extra thought number %d about the task", i),
		})
	}
	return base
}

// highScoreSteps returns steps that are very different from lowScoreSteps, producing
// high diff scores. Uses completely different tools and text.
func highScoreSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "completely different request"},
		{Role: "assistant", Content: "totally unrelated response with different words and meaning"},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "execute_command", Args: map[string]interface{}{"cmd": "ls"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "execute_command", Output: "output"}},
		{Role: "assistant", Content: "extra step not in baseline"},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "/tmp/out"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "done"}},
	}
}

// regressionBaselineSteps returns steps for building a baseline where each variant
// shares core tools but has varying step counts and slight tool differences.
// This produces varying diff scores when compared against the divergent snapshot,
// ensuring the CI has non-zero width.
func regressionBaselineSteps(variant int) []snapshot.Step {
	steps := []snapshot.Step{
		{Role: "user", Content: "analyze the codebase"},
		{Role: "assistant", Content: fmt.Sprintf("analyzing variant %d of the codebase now", variant)},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "/src/main.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package main"}},
	}
	// Add variant-specific tool calls to create different tool sequences and step counts.
	for i := 0; i < variant; i++ {
		toolName := fmt.Sprintf("helper_tool_%d", i)
		steps = append(steps,
			snapshot.Step{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: toolName, Args: map[string]interface{}{"n": i}}},
			snapshot.Step{Role: "tool", ToolResult: &snapshot.ToolResult{Name: toolName, Output: "ok"}},
		)
	}
	return steps
}

// regressionDivergentSteps returns steps that share some tools with regressionBaselineSteps
// but add many new tools and steps, producing scores above thresholds.
func regressionDivergentSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "analyze the codebase"},
		{Role: "assistant", Content: "I will do something entirely different from the baseline approach"},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "/src/main.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package main"}},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "execute_command", Args: map[string]interface{}{"cmd": "go test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "execute_command", Output: "PASS"}},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "/src/new.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "ok"}},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "execute_command", Args: map[string]interface{}{"cmd": "go build"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "execute_command", Output: "ok"}},
		{Role: "assistant", ToolCall: &snapshot.ToolCall{Name: "list_files", Args: map[string]interface{}{"dir": "/src"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "list_files", Output: "main.go new.go"}},
		{Role: "assistant", Content: "done with a very different workflow than expected"},
	}
}

func TestBaselineRecordCreatesNew(t *testing.T) {
	workDir := makeWorkDir(t)

	// Create a snapshot.
	makeTestSnapshot(t, workDir, "snap1", lowScoreSteps())

	stdout, _, exitCode := runAgentDiff(t, workDir, "baseline", "record", "my-baseline", "snap1")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "Added snap1 to baseline my-baseline") {
		t.Fatalf("unexpected output: %s", stdout)
	}
	if !strings.Contains(stdout, "1 snapshots total") {
		t.Fatalf("expected 1 snapshot total, got: %s", stdout)
	}

	// Verify file exists on disk.
	bsDir := filepath.Join(workDir, ".agentdiff", "baselines")
	if _, err := os.Stat(filepath.Join(bsDir, "my-baseline.json.gz")); err != nil {
		t.Fatalf("baseline file not found: %v", err)
	}
}

func TestBaselineRecordAppends(t *testing.T) {
	workDir := makeWorkDir(t)

	makeTestSnapshot(t, workDir, "snap1", lowScoreSteps())
	makeTestSnapshot(t, workDir, "snap2", lowScoreSteps())

	// Record first snapshot.
	runAgentDiff(t, workDir, "baseline", "record", "my-baseline", "snap1")

	// Record second snapshot.
	stdout, _, exitCode := runAgentDiff(t, workDir, "baseline", "record", "my-baseline", "snap2")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "2 snapshots total") {
		t.Fatalf("expected 2 snapshots total, got: %s", stdout)
	}
}

func TestBaselineCompareDetectsRegression(t *testing.T) {
	workDir := makeWorkDir(t)

	// Build baseline from similar snapshots (same tools, slight text variation).
	// All baseline snapshots have 4 steps and use read_file.
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("base%d", i)
		makeTestSnapshot(t, workDir, name, regressionBaselineSteps(i))
		runAgentDiff(t, workDir, "baseline", "record", "test-baseline", name)
	}

	// Create a divergent snapshot: shares some tools but adds many new ones and more steps.
	// This produces tool_score > 0.3 and step_delta > 5, exceeding default thresholds.
	makeTestSnapshot(t, workDir, "divergent", regressionDivergentSteps())

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "baseline", "compare", "test-baseline", "divergent")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 (regression), got %d; stdout: %s; stderr: %s", exitCode, stdout, stderr)
	}
}

func TestBaselineComparePassesSimilar(t *testing.T) {
	workDir := makeWorkDir(t)

	// Build baseline from identical snapshots.
	steps := lowScoreSteps()
	for i := 0; i < 3; i++ {
		makeTestSnapshot(t, workDir, "base"+string(rune('a'+i)), steps)
		runAgentDiff(t, workDir, "baseline", "record", "test-baseline", "base"+string(rune('a'+i)))
	}

	// Compare with the same kind of snapshot (identical steps = score 0).
	makeTestSnapshot(t, workDir, "current", steps)

	stdout, _, exitCode := runAgentDiff(t, workDir, "baseline", "compare", "test-baseline", "current")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (pass), got %d; output: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "PASS") {
		t.Fatalf("expected PASS in output, got: %s", stdout)
	}
}

func TestBaselineCompareJSON(t *testing.T) {
	workDir := makeWorkDir(t)

	steps := lowScoreSteps()
	for i := 0; i < 3; i++ {
		makeTestSnapshot(t, workDir, "base"+string(rune('a'+i)), steps)
		runAgentDiff(t, workDir, "baseline", "record", "test-baseline", "base"+string(rune('a'+i)))
	}
	makeTestSnapshot(t, workDir, "current", steps)

	stdout, _, exitCode := runAgentDiff(t, workDir, "--json", "baseline", "compare", "test-baseline", "current")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got parse error: %v\noutput: %s", err, stdout)
	}
	if _, ok := parsed["stats"]; !ok {
		t.Fatalf("expected 'stats' key in JSON output, got: %v", parsed)
	}
	if _, ok := parsed["regression"]; !ok {
		t.Fatalf("expected 'regression' key in JSON output, got: %v", parsed)
	}
}

func TestBaselineListShowsInfo(t *testing.T) {
	workDir := makeWorkDir(t)

	// Create baselines.
	makeTestSnapshot(t, workDir, "snap1", lowScoreSteps())
	makeTestSnapshot(t, workDir, "snap2", lowScoreSteps())
	runAgentDiff(t, workDir, "baseline", "record", "alpha", "snap1")
	runAgentDiff(t, workDir, "baseline", "record", "beta", "snap1")
	runAgentDiff(t, workDir, "baseline", "record", "beta", "snap2")

	stdout, _, exitCode := runAgentDiff(t, workDir, "baseline", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}

	if !strings.Contains(stdout, "alpha") {
		t.Fatalf("expected 'alpha' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "beta") {
		t.Fatalf("expected 'beta' in output, got: %s", stdout)
	}
}

func TestBaselineListJSON(t *testing.T) {
	workDir := makeWorkDir(t)

	makeTestSnapshot(t, workDir, "snap1", lowScoreSteps())
	runAgentDiff(t, workDir, "baseline", "record", "alpha", "snap1")

	stdout, _, exitCode := runAgentDiff(t, workDir, "--json", "baseline", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", exitCode, stdout)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("expected valid JSON array, got parse error: %v\noutput: %s", err, stdout)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(parsed))
	}
	if parsed[0]["name"] != "alpha" {
		t.Fatalf("expected name 'alpha', got: %v", parsed[0]["name"])
	}
}

func TestBaselineListEmpty(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, _, exitCode := runAgentDiff(t, workDir, "baseline", "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "No baselines recorded") {
		t.Fatalf("expected empty message, got: %s", stdout)
	}
}
