package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
