package cmd_test

import (
	"os"
	"strings"
	"testing"
)

func TestRecordStdin(t *testing.T) {
	workDir := makeWorkDir(t)

	// Pipe a Claude trace via stdin.
	traceData, err := os.ReadFile(testdataFile("claude_trace.jsonl"))
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	cmd := runAgentDiffStdin(t, workDir, traceData, "record", "--name", "stdin-test", "-")
	if !strings.Contains(cmd, "Recorded snapshot") {
		t.Fatalf("expected 'Recorded snapshot', got: %s", cmd)
	}
}

func TestRecordAdapterFlag(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, _, exitCode := runAgentDiff(t, workDir,
		"record", "--name", "adapter-flag", "--adapter", "claude", testdataFile("claude_trace.jsonl"))
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "Recorded snapshot") {
		t.Fatalf("expected 'Recorded snapshot', got: %s", stdout)
	}
}

func TestRecordAutoName(t *testing.T) {
	workDir := makeWorkDir(t)

	// Record without --name flag. Should auto-generate name from filename.
	stdout, _, exitCode := runAgentDiff(t, workDir, "record", testdataFile("claude_trace.jsonl"))
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "claude_trace_") {
		t.Fatalf("expected auto-generated name containing 'claude_trace_', got: %s", stdout)
	}
}

func TestRecordMissingFile(t *testing.T) {
	workDir := makeWorkDir(t)

	_, stderr, exitCode := runAgentDiff(t, workDir, "record", "--name", "bad", "/nonexistent/file.jsonl")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for missing file")
	}
	_ = stderr
}

func TestRecordInvalidAdapter(t *testing.T) {
	workDir := makeWorkDir(t)

	_, _, exitCode := runAgentDiff(t, workDir,
		"record", "--name", "bad-adapter", "--adapter", "fake_adapter", testdataFile("claude_trace.jsonl"))
	if exitCode == 0 {
		t.Fatal("expected non-zero exit for invalid adapter name")
	}
}

func TestRecordClaudeCodeAdapter(t *testing.T) {
	workDir := makeWorkDir(t)

	stdout, stderr, exitCode := runAgentDiff(t, workDir,
		"record", "--name", "claudecode-test", testdataFile("claudecode_stream.jsonl"))
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "Recorded snapshot") {
		t.Fatalf("expected 'Recorded snapshot', got: %s", stdout)
	}
}
