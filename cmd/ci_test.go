package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// setupCIWorkDir creates a temp directory with baseline and current snapshots for CI tests.
// Returns the work directory path.
func setupCIWorkDir(t *testing.T, baselineSnaps []snapshot.Snapshot, currentSnaps []snapshot.Snapshot, configYAML string) string {
	t.Helper()
	workDir := makeWorkDir(t)

	// Write config.
	if configYAML != "" {
		if err := os.WriteFile(filepath.Join(workDir, ".agentdiff.yaml"), []byte(configYAML), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}

	// Save baseline using BaselineStore.
	bs := snapshot.NewBaselineStore(workDir)
	baseline := snapshot.Baseline{
		Name:      "main",
		Snapshots: baselineSnaps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := bs.Save(baseline); err != nil {
		t.Fatalf("save baseline: %v", err)
	}

	// Save current snapshots using Store.
	store := snapshot.NewStore(workDir)
	for _, s := range currentSnaps {
		if _, err := store.Save(s); err != nil {
			t.Fatalf("save snapshot %q: %v", s.Name, err)
		}
	}

	return workDir
}

func makeBaselineConfig(baselinePath string) string {
	return "ci:\n  baseline_path: " + baselinePath + "\n"
}

func makeBaselineConfigWithDrift(baselinePath string, failOnDrift bool) string {
	driftStr := "false"
	if failOnDrift {
		driftStr = "true"
	}
	return "ci:\n  baseline_path: " + baselinePath + "\n  fail_on_style_drift: " + driftStr + "\n"
}

// identicalSteps returns steps that produce identical tool and text comparisons.
func identicalSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world", ToolCall: &snapshot.ToolCall{Name: "search", Args: map[string]interface{}{"q": "test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "search", Output: "result"}},
	}
}

// differentToolSteps returns steps with different tools (causes tool regression).
func differentToolSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world", ToolCall: &snapshot.ToolCall{Name: "fetch", Args: map[string]interface{}{"url": "http://example.com"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "fetch", Output: "page"}},
		{Role: "assistant", Content: "extra step", ToolCall: &snapshot.ToolCall{Name: "parse", Args: map[string]interface{}{}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "parse", Output: "done"}},
		{Role: "assistant", Content: "more steps"},
		{Role: "assistant", Content: "even more steps"},
		{Role: "assistant", Content: "and more"},
		{Role: "assistant", Content: "keep going"},
		{Role: "assistant", Content: "still more"},
	}
}

// differentTextSteps returns steps with same tools but different text (style drift only).
func differentTextSteps() []snapshot.Step {
	return []snapshot.Step{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "completely different response text that shares no words with the original baseline text whatsoever", ToolCall: &snapshot.ToolCall{Name: "search", Args: map[string]interface{}{"q": "test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "search", Output: "result"}},
	}
}

func TestCIPass(t *testing.T) {
	// Both baseline and current have identical steps -> exit 0.
	steps := identicalSteps()

	baseSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     steps,
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     steps,
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig(".agentdiff/baselines/main.json.gz"))

	stdout, _, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (PASS), got %d\nstdout: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "AgentDiff CI Report") {
		t.Fatalf("expected CI report in stdout, got: %s", stdout)
	}
}

func TestCIFunctionalRegression(t *testing.T) {
	// Baseline has one set of tools, current has different tools -> exit 1.
	baseSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     identicalSteps(),
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     differentToolSteps(),
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig(".agentdiff/baselines/main.json.gz"))

	stdout, _, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode != 1 {
		t.Fatalf("expected exit 1 (REGRESSION), got %d\nstdout: %s", exitCode, stdout)
	}
	// Report should still be rendered before exit.
	if !strings.Contains(stdout, "AgentDiff CI Report") {
		t.Fatalf("expected CI report in stdout even on regression, got: %s", stdout)
	}
}

func TestCIStyleDrift(t *testing.T) {
	// Same tools, different text, fail_on_style_drift=false -> exit 2.
	baseSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     identicalSteps(),
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     differentTextSteps(),
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfigWithDrift(".agentdiff/baselines/main.json.gz", false))

	stdout, _, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode != 2 {
		t.Fatalf("expected exit 2 (STYLE DRIFT), got %d\nstdout: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "AgentDiff CI Report") {
		t.Fatalf("expected CI report in stdout, got: %s", stdout)
	}
}

func TestCIMissingBaselinePath(t *testing.T) {
	workDir := makeWorkDir(t)

	// No config file, no --baseline flag -> error about missing baseline path.
	_, stderr, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode == 0 {
		t.Fatal("expected non-zero exit for missing baseline path")
	}
	if !strings.Contains(stderr, "baseline") {
		t.Fatalf("expected error about baseline in stderr, got: %s", stderr)
	}
}

func TestCIMissingBaselineFile(t *testing.T) {
	workDir := makeWorkDir(t)

	// Config points to a baseline file that doesn't exist.
	cfg := makeBaselineConfig(".agentdiff/baselines/nonexistent.json.gz")
	if err := os.WriteFile(filepath.Join(workDir, ".agentdiff.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, stderr, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode == 0 {
		t.Fatal("expected non-zero exit for missing baseline file")
	}
	if !strings.Contains(stderr, "baseline") {
		t.Fatalf("expected error about baseline in stderr, got: %s", stderr)
	}
}

func TestCIBaselineFlagOverride(t *testing.T) {
	// Config has wrong path, but --baseline flag points to correct file.
	steps := identicalSteps()
	baseSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     steps,
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     steps,
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig("wrong/path/baseline.json.gz"))

	// Use --baseline to point to the correct location.
	correctPath := filepath.Join(workDir, ".agentdiff", "baselines", "main.json.gz")
	stdout, _, exitCode := runAgentDiff(t, workDir, "ci", "--baseline", correctPath)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (PASS) with --baseline override, got %d\nstdout: %s", exitCode, stdout)
	}
}

func TestCIOutputFile(t *testing.T) {
	steps := identicalSteps()
	baseSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     steps,
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     steps,
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig(".agentdiff/baselines/main.json.gz"))

	outFile := filepath.Join(workDir, "report.md")
	stdout, _, exitCode := runAgentDiff(t, workDir, "ci", "--output", outFile)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout: %s", exitCode, stdout)
	}

	// stdout should be empty (report goes to file).
	if strings.Contains(stdout, "AgentDiff CI Report") {
		t.Fatal("report should go to file, not stdout, when --output is set")
	}

	// File should contain the report.
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), "AgentDiff CI Report") {
		t.Fatalf("expected report in output file, got: %s", string(data))
	}
}

func TestCINoBaselineMatch(t *testing.T) {
	// Baseline has different snapshot names than current -> warning, no results.
	baseSnap := snapshot.Snapshot{
		Name:      "baseline-only",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     identicalSteps(),
	}
	currentSnap := snapshot.Snapshot{
		Name:      "current-only",
		Timestamp: time.Now(),
		Steps:     identicalSteps(),
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{baseSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig(".agentdiff/baselines/main.json.gz"))

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (no matches = no regressions), got %d", exitCode)
	}
	if !strings.Contains(stderr, "warning") {
		t.Fatalf("expected warning about no baseline match in stderr, got: %s", stderr)
	}
	if !strings.Contains(stdout, "AgentDiff CI Report") {
		t.Fatalf("expected CI report header, got: %s", stdout)
	}
}

func TestCILatestBaselineSnapshot(t *testing.T) {
	// Baseline has two snapshots with same name, different timestamps.
	// Should use the latest one.
	steps := identicalSteps()
	oldSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-2 * time.Hour),
		Steps:     differentToolSteps(), // Old: different tools (would regress).
	}
	newSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now().Add(-time.Hour),
		Steps:     steps, // New: identical (should pass).
	}
	currentSnap := snapshot.Snapshot{
		Name:      "agent-test",
		Timestamp: time.Now(),
		Steps:     steps,
	}

	workDir := setupCIWorkDir(t, []snapshot.Snapshot{oldSnap, newSnap}, []snapshot.Snapshot{currentSnap},
		makeBaselineConfig(".agentdiff/baselines/main.json.gz"))

	stdout, _, exitCode := runAgentDiff(t, workDir, "ci")

	if exitCode != 0 {
		t.Fatalf("expected exit 0 (latest baseline matches), got %d\nstdout: %s", exitCode, stdout)
	}
}
