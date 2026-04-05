package cmd_test

import (
	"testing"
)

func TestClusterCommand_NoArgs(t *testing.T) {
	workDir := makeWorkDir(t)
	_, _, exitCode := runAgentDiff(t, workDir, "cluster")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code when cluster is called with no args")
	}
}

func TestClusterCompareCommand_MissingArgs(t *testing.T) {
	workDir := makeWorkDir(t)
	_, _, exitCode := runAgentDiff(t, workDir, "cluster", "compare")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code when cluster compare is called with no args")
	}

	_, _, exitCode = runAgentDiff(t, workDir, "cluster", "compare", "only-one-arg")
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code when cluster compare is called with only one arg")
	}
}
