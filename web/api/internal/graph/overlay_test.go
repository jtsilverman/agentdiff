package graph

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func sortedStrings(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func TestOverlay_MatchingTrace(t *testing.T) {
	// Baseline: all traces follow search -> filter -> summarize.
	baseline := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
	}
	g := Aggregate(baseline)
	target := toolCallSteps("search", "filter", "summarize")

	r := Overlay(g, target)

	wantNodes := []string{"filter", "search", "summarize"}
	if !reflect.DeepEqual(sortedStrings(r.MatchedNodeIDs), wantNodes) {
		t.Errorf("MatchedNodeIDs = %v, want %v", r.MatchedNodeIDs, wantNodes)
	}
	if len(r.MatchedEdgeIDs) != 2 {
		t.Errorf("MatchedEdgeIDs = %v, want 2 edges", r.MatchedEdgeIDs)
	}
	if len(r.DivergencePoints) != 0 {
		t.Errorf("DivergencePoints = %v, want 0", r.DivergencePoints)
	}
}

func TestOverlay_BranchDivergentTrace(t *testing.T) {
	// Baseline: 4 traces take filter->summarize, 1 trace takes filter->translate.
	// The trace under overlay takes the minority path (filter->translate).
	baseline := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "translate"),
	}
	g := Aggregate(baseline)
	target := toolCallSteps("search", "filter", "translate")

	r := Overlay(g, target)

	// All three nodes exist in the graph; trace matched all of them.
	if got := sortedStrings(r.MatchedNodeIDs); !reflect.DeepEqual(got, []string{"filter", "search", "translate"}) {
		t.Errorf("MatchedNodeIDs = %v, want [filter search translate]", got)
	}
	// Both edges exist in the graph (search->filter and filter->translate).
	if len(r.MatchedEdgeIDs) != 2 {
		t.Errorf("MatchedEdgeIDs = %v, want 2 edges", r.MatchedEdgeIDs)
	}
	// Divergence at filter: majority went to summarize (4/5=0.8), this trace chose translate (1/5=0.2).
	if len(r.DivergencePoints) != 1 {
		t.Fatalf("DivergencePoints = %v, want 1", r.DivergencePoints)
	}
	d := r.DivergencePoints[0]
	if d.NodeID != "filter" {
		t.Errorf("DivergencePoints[0].NodeID = %q, want %q", d.NodeID, "filter")
	}
	if d.ThisTraceChose != "translate" {
		t.Errorf("DivergencePoints[0].ThisTraceChose = %q, want %q", d.ThisTraceChose, "translate")
	}
	if d.BaselinePct != 0.2 {
		t.Errorf("DivergencePoints[0].BaselinePct = %f, want 0.2", d.BaselinePct)
	}
}

func TestOverlay_NovelTrace(t *testing.T) {
	// Baseline: search -> filter -> summarize.
	// Target trace introduces a novel "unknown" tool after search.
	baseline := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
	}
	g := Aggregate(baseline)
	target := toolCallSteps("search", "unknown")

	r := Overlay(g, target)

	// Only "search" matches; "unknown" is novel.
	if got := sortedStrings(r.MatchedNodeIDs); !reflect.DeepEqual(got, []string{"search"}) {
		t.Errorf("MatchedNodeIDs = %v, want [search]", got)
	}
	// No edges match (search->unknown does not exist in baseline).
	if len(r.MatchedEdgeIDs) != 0 {
		t.Errorf("MatchedEdgeIDs = %v, want 0", r.MatchedEdgeIDs)
	}
	// Divergence at search: trace chose "unknown" which doesn't exist in baseline (pct=0).
	if len(r.DivergencePoints) != 1 {
		t.Fatalf("DivergencePoints = %v, want 1", r.DivergencePoints)
	}
	d := r.DivergencePoints[0]
	if d.NodeID != "search" || d.ThisTraceChose != "unknown" || d.BaselinePct != 0.0 {
		t.Errorf("DivergencePoint = %+v, want {search, unknown, 0.0}", d)
	}
}

func TestOverlay_EmptyTrace(t *testing.T) {
	baseline := [][]snapshot.Step{
		toolCallSteps("search", "filter"),
	}
	g := Aggregate(baseline)

	r := Overlay(g, []snapshot.Step{})

	if len(r.MatchedNodeIDs) != 0 {
		t.Errorf("MatchedNodeIDs = %v, want 0", r.MatchedNodeIDs)
	}
	if len(r.MatchedEdgeIDs) != 0 {
		t.Errorf("MatchedEdgeIDs = %v, want 0", r.MatchedEdgeIDs)
	}
	if len(r.DivergencePoints) != 0 {
		t.Errorf("DivergencePoints = %v, want 0", r.DivergencePoints)
	}
}

func TestOverlay_MatchedEdgeIDFormat(t *testing.T) {
	// Confirms the edge ID format is "from->to".
	baseline := [][]snapshot.Step{
		toolCallSteps("a", "b"),
	}
	g := Aggregate(baseline)
	r := Overlay(g, toolCallSteps("a", "b"))
	if len(r.MatchedEdgeIDs) != 1 || r.MatchedEdgeIDs[0] != "a->b" {
		t.Errorf("MatchedEdgeIDs = %v, want [a->b]", r.MatchedEdgeIDs)
	}
}
