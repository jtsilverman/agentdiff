package cluster

import (
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func makeSteps(toolNames ...string) []snapshot.Step {
	var steps []snapshot.Step
	for _, name := range toolNames {
		steps = append(steps, snapshot.Step{
			Role:     "tool_call",
			ToolCall: &snapshot.ToolCall{Name: name, Args: map[string]interface{}{}},
		})
	}
	return steps
}

func makeSnapshot(name string, toolNames ...string) snapshot.Snapshot {
	return snapshot.Snapshot{
		Name:  name,
		Steps: makeSteps(toolNames...),
	}
}

func TestClusterBaseline_TwoStrategies(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "two-strategy",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("search-1", "search", "filter", "summarize"),
			makeSnapshot("search-2", "search", "filter", "summarize"),
			makeSnapshot("search-3", "search", "filter", "summarize"),
			makeSnapshot("direct-1", "lookup", "answer"),
			makeSnapshot("direct-2", "lookup", "answer"),
			makeSnapshot("direct-3", "lookup", "answer"),
		},
	}

	report, err := ClusterBaseline(baseline, 0, 2)
	if err != nil {
		t.Fatalf("ClusterBaseline: %v", err)
	}

	if report.BaselineName != "two-strategy" {
		t.Errorf("BaselineName = %q, want %q", report.BaselineName, "two-strategy")
	}
	if report.SnapshotCount != 6 {
		t.Errorf("SnapshotCount = %d, want 6", report.SnapshotCount)
	}
	if len(report.Strategies) != 2 {
		t.Fatalf("got %d strategies, want 2", len(report.Strategies))
	}
	if len(report.Noise) != 0 {
		t.Errorf("got %d noise points, want 0", len(report.Noise))
	}

	// Verify each strategy has 3 members.
	for _, s := range report.Strategies {
		if s.Count != 3 {
			t.Errorf("strategy %d has %d members, want 3", s.ID, s.Count)
		}
		if len(s.ToolSeq) == 0 {
			t.Errorf("strategy %d has empty tool sequence", s.ID)
		}
		if s.Exemplar == "" {
			t.Errorf("strategy %d has empty exemplar", s.ID)
		}
	}
}

func TestClusterBaseline_SingleStrategy(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "single",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("run-1", "fetch", "parse", "store"),
			makeSnapshot("run-2", "fetch", "parse", "store"),
			makeSnapshot("run-3", "fetch", "parse", "store"),
		},
	}

	report, err := ClusterBaseline(baseline, 0, 2)
	if err != nil {
		t.Fatalf("ClusterBaseline: %v", err)
	}

	if len(report.Strategies) != 1 {
		t.Fatalf("got %d strategies, want 1", len(report.Strategies))
	}
	if len(report.Noise) != 0 {
		t.Errorf("got %d noise points, want 0", len(report.Noise))
	}
	if report.Strategies[0].Count != 3 {
		t.Errorf("strategy has %d members, want 3", report.Strategies[0].Count)
	}
}

func TestClusterBaseline_EmptyBaseline(t *testing.T) {
	baseline := snapshot.Baseline{Name: "empty"}

	_, err := ClusterBaseline(baseline, 0, 2)
	if err == nil {
		t.Fatal("expected error for empty baseline")
	}
}

func TestClusterBaseline_TooFewSnapshots(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "small",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("only-one", "search"),
		},
	}

	_, err := ClusterBaseline(baseline, 0, 2)
	if err == nil {
		t.Fatal("expected error for baseline with 1 snapshot")
	}
}

func TestCompareToCluster_Matched(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "match-test",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("s1", "search", "filter", "summarize"),
			makeSnapshot("s2", "search", "filter", "summarize"),
			makeSnapshot("s3", "search", "filter", "summarize"),
			makeSnapshot("d1", "lookup", "answer"),
			makeSnapshot("d2", "lookup", "answer"),
			makeSnapshot("d3", "lookup", "answer"),
		},
	}

	// New snapshot that matches the search strategy.
	newSnap := makeSnapshot("new-search", "search", "filter", "summarize")

	result, err := CompareToCluster(baseline, newSnap, 0, 2)
	if err != nil {
		t.Fatalf("CompareToCluster: %v", err)
	}

	if !result.Matched {
		t.Errorf("expected Matched=true, got false (distance=%d, maxIntra=%d)",
			result.Distance, result.MaxIntraClusterDist)
	}
	if result.Distance != 0 {
		t.Errorf("expected Distance=0 for identical sequence, got %d", result.Distance)
	}
}

func TestCompareToCluster_NotMatched(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "nomatch-test",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("s1", "search", "filter", "summarize"),
			makeSnapshot("s2", "search", "filter", "summarize"),
			makeSnapshot("s3", "search", "filter", "summarize"),
			makeSnapshot("d1", "lookup", "answer"),
			makeSnapshot("d2", "lookup", "answer"),
			makeSnapshot("d3", "lookup", "answer"),
		},
	}

	// New snapshot with a completely different tool sequence.
	newSnap := makeSnapshot("alien", "deploy", "monitor", "alert", "rollback", "notify")

	result, err := CompareToCluster(baseline, newSnap, 0, 2)
	if err != nil {
		t.Fatalf("CompareToCluster: %v", err)
	}

	if result.Matched {
		t.Errorf("expected Matched=false, got true (distance=%d, maxIntra=%d)",
			result.Distance, result.MaxIntraClusterDist)
	}
	// Should still populate closest strategy info.
	if result.Exemplar == "" {
		t.Error("expected Exemplar to be populated even when not matched")
	}
}

func TestClusterBaseline_InvalidMinPts(t *testing.T) {
	baseline := snapshot.Baseline{
		Name: "test",
		Snapshots: []snapshot.Snapshot{
			makeSnapshot("a", "search", "summarize"),
			makeSnapshot("b", "search", "summarize"),
			makeSnapshot("c", "lookup", "answer"),
		},
	}

	// minPts = 0: should error.
	_, err := ClusterBaseline(baseline, 0, 0)
	if err == nil {
		t.Fatal("expected error for minPts=0, got nil")
	}

	// minPts = -1: should error.
	_, err = ClusterBaseline(baseline, 0, -1)
	if err == nil {
		t.Fatal("expected error for minPts=-1, got nil")
	}
}
