package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jtsilverman/agentdiff/internal/cluster"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

var binPath string

func TestMain(m *testing.M) {
	// Build binary to a temp directory.
	tmpDir, err := os.MkdirTemp("", "agentdiff-integration-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}

	binPath = filepath.Join(tmpDir, "agentdiff")
	build := exec.Command(filepath.Join(os.Getenv("HOME"), "go-sdk", "go", "bin", "go"), "build", "-o", binPath, ".")
	build.Dir = filepath.Join(projectRoot())
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.RemoveAll(tmpDir)
		panic("build binary: " + err.Error())
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// projectRoot returns the absolute path to the agentdiff project root.
func projectRoot() string {
	// cmd/integration_test.go is in cmd/, so project root is one level up.
	// But since tests run with cwd set to the package dir, we use a fixed approach.
	dir, err := filepath.Abs(filepath.Join("..", "."))
	if err != nil {
		panic("resolve project root: " + err.Error())
	}
	return dir
}

// testdataFile returns the absolute path to a testdata file.
func testdataFile(name string) string {
	return filepath.Join(projectRoot(), "testdata", name)
}

// makeWorkDir creates a unique temp directory for a test to use as its working directory.
func makeWorkDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "agentdiff-test-*")
	if err != nil {
		t.Fatalf("create work dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// runAgentDiff executes the agentdiff binary with the given args in the given working directory.
// Returns stdout, stderr, and exit code.
func runAgentDiff(t *testing.T, workDir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = workDir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run agentdiff %v: %v", args, err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

func TestIntegrationRecordClaude(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, _, exitCode := runAgentDiff(t, workDir,
		"record", "--name", "claude-test", testdataFile("claude_trace.jsonl"))

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "Recorded snapshot") {
		t.Fatalf("expected output to contain 'Recorded snapshot', got: %s", stdout)
	}
}

func TestIntegrationRecordOpenAI(t *testing.T) {
	workDir := makeWorkDir(t)

	_, _, exitCode := runAgentDiff(t, workDir,
		"record", "--name", "openai-test", testdataFile("openai_trace.json"))

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
}

func TestIntegrationList(t *testing.T) {
	workDir := makeWorkDir(t)

	// Record two snapshots.
	runAgentDiff(t, workDir, "record", "--name", "snap-alpha", testdataFile("claude_trace.jsonl"))
	runAgentDiff(t, workDir, "record", "--name", "snap-beta", testdataFile("openai_trace.json"))

	stdout, _, exitCode := runAgentDiff(t, workDir, "list")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}

	if !strings.Contains(stdout, "snap-alpha") {
		t.Fatalf("expected output to contain 'snap-alpha', got: %s", stdout)
	}
	if !strings.Contains(stdout, "snap-beta") {
		t.Fatalf("expected output to contain 'snap-beta', got: %s", stdout)
	}
}

func TestIntegrationDiffIdentical(t *testing.T) {
	workDir := makeWorkDir(t)

	// Record the same trace twice with different names.
	runAgentDiff(t, workDir, "record", "--name", "baseline", testdataFile("claude_trace.jsonl"))
	runAgentDiff(t, workDir, "record", "--name", "repeat", testdataFile("claude_trace.jsonl"))

	stdout, _, exitCode := runAgentDiff(t, workDir, "diff", "baseline", "repeat")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (PASS), got %d; output: %s", exitCode, stdout)
	}
}

func TestIntegrationDiffRegression(t *testing.T) {
	workDir := makeWorkDir(t)

	// Record Claude and OpenAI traces (different tools = regression).
	runAgentDiff(t, workDir, "record", "--name", "run-claude", testdataFile("claude_trace.jsonl"))
	runAgentDiff(t, workDir, "record", "--name", "run-openai", testdataFile("openai_trace.json"))

	_, _, exitCode := runAgentDiff(t, workDir, "diff", "run-claude", "run-openai")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 (REGRESSION), got %d", exitCode)
	}
}

func TestIntegrationDiffJSON(t *testing.T) {
	workDir := makeWorkDir(t)

	// Record two different traces.
	runAgentDiff(t, workDir, "record", "--name", "json-a", testdataFile("claude_trace.jsonl"))
	runAgentDiff(t, workDir, "record", "--name", "json-b", testdataFile("openai_trace.json"))

	stdout, _, _ := runAgentDiff(t, workDir, "diff", "--json", "json-a", "json-b")

	// Verify output is valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("expected valid JSON output, got parse error: %v\noutput: %s", err, stdout)
	}
}

// saveTestSnapshot is a helper that creates and saves a snapshot to the given workDir.
func saveTestSnapshot(t *testing.T, workDir, name string, steps []snapshot.Step) {
	t.Helper()
	store := snapshot.NewStore(workDir)
	_, err := store.Save(snapshot.Snapshot{
		Name:      name,
		Source:    "test",
		Timestamp: time.Now(),
		Steps:     steps,
	})
	if err != nil {
		t.Fatalf("save snapshot %q: %v", name, err)
	}
}

func TestIntegrationAlignment(t *testing.T) {
	workDir := makeWorkDir(t)

	// Snapshot A: read_file, write_file, run_test
	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "reading", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "contents"}},
		{Role: "assistant", Content: "writing", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "ok"}},
		{Role: "assistant", Content: "testing", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: map[string]interface{}{"cmd": "go test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "run_test", Output: "PASS"}},
	}

	// Snapshot B: read_file, search (INSERTED), write_file, run_test
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "reading", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "contents"}},
		{Role: "assistant", Content: "searching", ToolCall: &snapshot.ToolCall{Name: "search", Args: map[string]interface{}{"query": "foo"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "search", Output: "found"}},
		{Role: "assistant", Content: "writing", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "ok"}},
		{Role: "assistant", Content: "testing", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: map[string]interface{}{"cmd": "go test"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "run_test", Output: "PASS"}},
	}

	saveTestSnapshot(t, workDir, "align-a", stepsA)
	saveTestSnapshot(t, workDir, "align-b", stepsB)

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "diff", "--json", "align-a", "align-b")
	_ = stderr

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse JSON: %v\nstdout: %s", err, stdout)
	}

	diag, ok := result["diagnostics"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected diagnostics object, got: %v", result["diagnostics"])
	}

	// Verify alignment contains an insert op (op=2).
	alignment, ok := diag["alignment"].([]interface{})
	if !ok || len(alignment) == 0 {
		t.Fatalf("expected non-empty alignment array, got: %v", diag["alignment"])
	}

	hasInsert := false
	for _, entry := range alignment {
		pair, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if op, ok := pair["op"].(float64); ok && int(op) == 2 {
			hasInsert = true
			break
		}
	}
	if !hasInsert {
		t.Fatalf("expected alignment to contain an insert op (op=2), got: %v", alignment)
	}

	// Verify first_divergence >= 0.
	firstDiv, ok := diag["first_divergence"].(float64)
	if !ok || int(firstDiv) < 0 {
		t.Fatalf("expected first_divergence >= 0, got: %v", diag["first_divergence"])
	}

	// Exit code should be 1 (regression) since there's a tool difference.
	_ = exitCode
}

func TestIntegrationRetryCollapse(t *testing.T) {
	workDir := makeWorkDir(t)

	// Both snapshots have 3 consecutive tool call steps with same name AND same args
	// (retry sequence). Steps must be consecutive with ToolCall set for CollapseRetries
	// to detect the run.
	retryArgs := map[string]interface{}{"cmd": "npm test"}

	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "try 1", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
		{Role: "assistant", Content: "try 2", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
		{Role: "assistant", Content: "try 3", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
	}

	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "try 1", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
		{Role: "assistant", Content: "try 2", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
		{Role: "assistant", Content: "try 3", ToolCall: &snapshot.ToolCall{Name: "run_test", Args: retryArgs}},
	}

	saveTestSnapshot(t, workDir, "retry-a", stepsA)
	saveTestSnapshot(t, workDir, "retry-b", stepsB)

	stdout, _, _ := runAgentDiff(t, workDir, "diff", "--json", "retry-a", "retry-b")

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse JSON: %v\nstdout: %s", err, stdout)
	}

	diag, ok := result["diagnostics"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected diagnostics object, got: %v", result["diagnostics"])
	}

	retryGroups, ok := diag["retry_groups"].([]interface{})
	if !ok || len(retryGroups) == 0 {
		t.Fatalf("expected non-empty retry_groups array, got: %v", diag["retry_groups"])
	}
}

func TestIntegrationCI(t *testing.T) {
	workDir := makeWorkDir(t)

	// Create identical steps for baseline and current snapshots.
	steps := []snapshot.Step{
		{Role: "assistant", Content: "reading", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "main.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package main"}},
	}

	// Save current snapshot.
	saveTestSnapshot(t, workDir, "ci-snap", steps)

	// Create baseline with the same snapshot (identical = should pass).
	bs := snapshot.NewBaselineStore(workDir)
	baselineSnap := snapshot.Snapshot{
		Name:      "ci-snap",
		Source:    "test",
		Timestamp: time.Now(),
		Steps:     steps,
	}
	baseline := snapshot.Baseline{
		Name:      "ci-baseline",
		Snapshots: []snapshot.Snapshot{baselineSnap},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := bs.Save(baseline); err != nil {
		t.Fatalf("save baseline: %v", err)
	}

	// Write .agentdiff.yaml with ci.baseline_path pointing to the baseline file.
	baselinePath := filepath.Join(workDir, ".agentdiff", "baselines", "ci-baseline.json.gz")
	yamlContent := "ci:\n  baseline_path: " + baselinePath + "\n"
	if err := os.WriteFile(filepath.Join(workDir, ".agentdiff.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	reportPath := filepath.Join(workDir, "report.md")
	_, stderr, exitCode := runAgentDiff(t, workDir, "ci", "--output", reportPath)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for identical baseline/current, got %d; stderr: %s", exitCode, stderr)
	}

	// Verify report.md exists and contains markdown table headers.
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "| Metric |") {
		t.Fatalf("expected report to contain '| Metric |', got:\n%s", reportStr)
	}
}

func TestIntegrationBaselineRoundTrip(t *testing.T) {
	workDir := makeWorkDir(t)

	// Create two slightly different snapshots.
	stepsA := []snapshot.Step{
		{Role: "assistant", Content: "reading", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package a"}},
		{Role: "assistant", Content: "writing", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "a.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "ok"}},
	}
	stepsB := []snapshot.Step{
		{Role: "assistant", Content: "reading", ToolCall: &snapshot.ToolCall{Name: "read_file", Args: map[string]interface{}{"path": "b.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "read_file", Output: "package b"}},
		{Role: "assistant", Content: "writing", ToolCall: &snapshot.ToolCall{Name: "write_file", Args: map[string]interface{}{"path": "b.go"}}},
		{Role: "tool", ToolResult: &snapshot.ToolResult{Name: "write_file", Output: "ok"}},
	}

	// Save both snapshots via store (so baseline record can find them).
	saveTestSnapshot(t, workDir, "snap-a", stepsA)
	saveTestSnapshot(t, workDir, "snap-b", stepsB)

	// Record both into a baseline via CLI.
	_, stderr1, exit1 := runAgentDiff(t, workDir, "baseline", "record", "test-bl", "snap-a")
	if exit1 != 0 {
		t.Fatalf("baseline record snap-a failed (exit %d): %s", exit1, stderr1)
	}

	_, stderr2, exit2 := runAgentDiff(t, workDir, "baseline", "record", "test-bl", "snap-b")
	if exit2 != 0 {
		t.Fatalf("baseline record snap-b failed (exit %d): %s", exit2, stderr2)
	}

	// Compare snap-a against baseline with JSON output.
	stdout, stderr3, _ := runAgentDiff(t, workDir, "baseline", "compare", "test-bl", "snap-a", "--json")

	if stdout == "" {
		t.Fatalf("expected JSON output, got empty; stderr: %s", stderr3)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse JSON: %v\nstdout: %s", err, stdout)
	}

	// Verify stats contain tool_score with mean, lower, upper.
	statsObj, ok := result["stats"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected stats object, got: %v", result["stats"])
	}

	toolScore, ok := statsObj["tool_score"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected tool_score object, got: %v", statsObj["tool_score"])
	}

	for _, field := range []string{"mean", "lower", "upper"} {
		if _, ok := toolScore[field]; !ok {
			t.Fatalf("expected tool_score to have %q field, got: %v", field, toolScore)
		}
	}
}

// makeClusterSteps creates steps with the given tool sequence.
func makeClusterSteps(tools []string) []snapshot.Step {
	var steps []snapshot.Step
	for _, tool := range tools {
		steps = append(steps,
			snapshot.Step{
				Role:    "assistant",
				Content: "calling " + tool,
				ToolCall: &snapshot.ToolCall{
					Name: tool,
					Args: map[string]interface{}{"q": tool},
				},
			},
			snapshot.Step{
				Role:       "tool",
				ToolResult: &snapshot.ToolResult{Name: tool, Output: "done"},
			},
		)
	}
	return steps
}

// setupClusterBaseline creates 6 snapshots (3 search+summarize, 3 lookup+answer) and
// records them into a baseline named baselineName. Returns the workDir.
func setupClusterBaseline(t *testing.T, baselineName string) string {
	t.Helper()
	workDir := makeWorkDir(t)

	store := snapshot.NewStore(workDir)
	bs := snapshot.NewBaselineStore(workDir)

	// Group A: search+summarize variants (one has an extra "refine" to create intra-cluster variance).
	seqsA := [][]string{
		{"search", "summarize"},
		{"search", "summarize"},
		{"search", "summarize", "refine"},
	}
	// Group B: lookup+answer variants (one has an extra "format").
	seqsB := [][]string{
		{"lookup", "answer"},
		{"lookup", "answer"},
		{"lookup", "answer", "format"},
	}

	var allSnaps []snapshot.Snapshot
	for i := 0; i < 3; i++ {
		snapA := snapshot.Snapshot{
			Name:      fmt.Sprintf("search-%d", i),
			Source:    "test",
			Timestamp: time.Now(),
			Steps:     makeClusterSteps(seqsA[i]),
		}
		saved, err := store.Save(snapA)
		if err != nil {
			t.Fatalf("save snapshot search-%d: %v", i, err)
		}
		allSnaps = append(allSnaps, saved)

		snapB := snapshot.Snapshot{
			Name:      fmt.Sprintf("lookup-%d", i),
			Source:    "test",
			Timestamp: time.Now(),
			Steps:     makeClusterSteps(seqsB[i]),
		}
		saved, err = store.Save(snapB)
		if err != nil {
			t.Fatalf("save snapshot lookup-%d: %v", i, err)
		}
		allSnaps = append(allSnaps, saved)
	}

	// Build the baseline directly.
	baseline := snapshot.Baseline{
		Name:      baselineName,
		Snapshots: allSnaps,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := bs.Save(baseline); err != nil {
		t.Fatalf("save baseline: %v", err)
	}

	return workDir
}

func TestIntegrationV3Cluster(t *testing.T) {
	workDir := setupClusterBaseline(t, "test-strategies")

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "cluster", "test-strategies", "--json")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
	}

	var report cluster.StrategyReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("parse JSON: %v\nstdout: %s", err, stdout)
	}

	if len(report.Strategies) != 2 {
		t.Fatalf("expected 2 strategies, got %d; report: %+v", len(report.Strategies), report)
	}

	for _, s := range report.Strategies {
		if s.Count != 3 {
			t.Fatalf("expected strategy %d to have 3 members, got %d", s.ID, s.Count)
		}
	}
}

func TestIntegrationV3ClusterCompare(t *testing.T) {
	workDir := setupClusterBaseline(t, "test-strategies")

	// Create a new snapshot with tools [search, summarize, cite] -- close to search+summarize.
	saveTestSnapshot(t, workDir, "new-snap", makeClusterSteps([]string{"search", "summarize", "cite"}))

	stdout, stderr, exitCode := runAgentDiff(t, workDir, "cluster", "compare", "test-strategies", "new-snap", "--json")
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (matched), got %d; stderr: %s; stdout: %s", exitCode, stderr, stdout)
	}

	var result cluster.MatchResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("parse JSON: %v\nstdout: %s", err, stdout)
	}

	if !result.Matched {
		t.Fatalf("expected Matched=true, got false; result: %+v", result)
	}
}

func TestIntegrationV3Record(t *testing.T) {
	// Test agents_sdk adapter.
	t.Run("agents_sdk", func(t *testing.T) {
		workDir := makeWorkDir(t)
		stdout, stderr, exitCode := runAgentDiff(t, workDir,
			"record", "--name", "agents-sdk-test", testdataFile("agents_sdk_trace.json"))
		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Recorded snapshot") {
			t.Fatalf("expected 'Recorded snapshot' in output, got: %s", stdout)
		}
	})

	// Test langchain adapter.
	t.Run("langchain", func(t *testing.T) {
		workDir := makeWorkDir(t)
		stdout, stderr, exitCode := runAgentDiff(t, workDir,
			"record", "--name", "langchain-test", testdataFile("langchain_callbacks.jsonl"))
		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Recorded snapshot") {
			t.Fatalf("expected 'Recorded snapshot' in output, got: %s", stdout)
		}
	})

	// Test generic adapter.
	t.Run("generic", func(t *testing.T) {
		workDir := makeWorkDir(t)

		// Write .agentdiff.yaml with generic adapter config.
		yamlContent := `adapter:
  generic:
    role_field: "kind"
    role_map:
      msg: "assistant"
      fn_call: "tool_call"
      fn_output: "tool_result"
    tool_name_field: "action.fn"
    tool_args_field: "action.params"
    tool_output_field: "result"
    content_field: "body"
`
		if err := os.WriteFile(filepath.Join(workDir, ".agentdiff.yaml"), []byte(yamlContent), 0644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		stdout, stderr, exitCode := runAgentDiff(t, workDir,
			"record", "--name", "generic-test", "--adapter", "generic", testdataFile("generic_trace.jsonl"))
		if exitCode != 0 {
			t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
		}
		if !strings.Contains(stdout, "Recorded snapshot") {
			t.Fatalf("expected 'Recorded snapshot' in output, got: %s", stdout)
		}
	})
}
