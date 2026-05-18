package graph

import (
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func toolCallSteps(names ...string) []snapshot.Step {
	steps := make([]snapshot.Step, 0, len(names))
	for _, n := range names {
		steps = append(steps, snapshot.Step{
			Role:     "tool_call",
			ToolCall: &snapshot.ToolCall{Name: n, Args: map[string]interface{}{}},
		})
	}
	return steps
}

func findNode(g PathGraph, id string) (Node, bool) {
	for _, n := range g.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

func findEdge(g PathGraph, from, to string) (Edge, bool) {
	for _, e := range g.Edges {
		if e.From == from && e.To == to {
			return e, true
		}
	}
	return Edge{}, false
}

func TestAggregate_SingleTrace(t *testing.T) {
	traces := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
	}
	g := Aggregate(traces)

	if g.Stats.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", g.Stats.TotalRuns)
	}
	if g.Stats.BranchPoints != 0 {
		t.Errorf("BranchPoints = %d, want 0", g.Stats.BranchPoints)
	}
	if len(g.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(g.Edges))
	}
	for _, name := range []string{"search", "filter", "summarize"} {
		n, ok := findNode(g, name)
		if !ok {
			t.Errorf("node %q missing", name)
			continue
		}
		if n.Count != 1 {
			t.Errorf("node %q count = %d, want 1", name, n.Count)
		}
	}
	if e, ok := findEdge(g, "search", "filter"); !ok || e.Count != 1 || e.Weight != 1.0 {
		t.Errorf("edge search->filter: %+v, ok=%v; want count=1 weight=1.0", e, ok)
	}
}

func TestAggregate_IdenticalTraces(t *testing.T) {
	traces := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
	}
	g := Aggregate(traces)

	if g.Stats.TotalRuns != 3 {
		t.Errorf("TotalRuns = %d, want 3", g.Stats.TotalRuns)
	}
	if g.Stats.BranchPoints != 0 {
		t.Errorf("BranchPoints = %d, want 0", g.Stats.BranchPoints)
	}
	if len(g.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(g.Nodes))
	}
	if len(g.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(g.Edges))
	}
	// Each tool occurs once per trace * 3 traces.
	for _, name := range []string{"search", "filter", "summarize"} {
		n, _ := findNode(g, name)
		if n.Count != 3 {
			t.Errorf("node %q count = %d, want 3", name, n.Count)
		}
	}
	for _, pair := range [][2]string{{"search", "filter"}, {"filter", "summarize"}} {
		e, _ := findEdge(g, pair[0], pair[1])
		if e.Count != 3 {
			t.Errorf("edge %s->%s count = %d, want 3", pair[0], pair[1], e.Count)
		}
		if e.Weight != 1.0 {
			t.Errorf("edge %s->%s weight = %f, want 1.0", pair[0], pair[1], e.Weight)
		}
	}
}

func TestAggregate_DivergentTraces(t *testing.T) {
	traces := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "translate"),
	}
	g := Aggregate(traces)

	if g.Stats.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", g.Stats.TotalRuns)
	}
	if g.Stats.BranchPoints != 1 {
		t.Errorf("BranchPoints = %d, want 1 (filter branches)", g.Stats.BranchPoints)
	}
	if len(g.Nodes) != 4 {
		t.Fatalf("len(Nodes) = %d, want 4", len(g.Nodes))
	}
	if len(g.Edges) != 3 {
		t.Fatalf("len(Edges) = %d, want 3", len(g.Edges))
	}

	// search -> filter occurs in both traces.
	if e, _ := findEdge(g, "search", "filter"); e.Count != 2 || e.Weight != 1.0 {
		t.Errorf("edge search->filter: count=%d weight=%f; want 2/1.0", e.Count, e.Weight)
	}
	// filter -> summarize and filter -> translate each occur once; weight 0.5.
	for _, target := range []string{"summarize", "translate"} {
		e, _ := findEdge(g, "filter", target)
		if e.Count != 1 {
			t.Errorf("edge filter->%s count = %d, want 1", target, e.Count)
		}
		if e.Weight != 0.5 {
			t.Errorf("edge filter->%s weight = %f, want 0.5", target, e.Weight)
		}
	}
}

func TestAggregate_NovelTrace_NoToolCalls(t *testing.T) {
	traces := [][]snapshot.Step{
		{{Role: "assistant", Content: "just talking, no tools"}},
	}
	g := Aggregate(traces)

	if g.Stats.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", g.Stats.TotalRuns)
	}
	if len(g.Nodes) != 0 {
		t.Errorf("len(Nodes) = %d, want 0", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("len(Edges) = %d, want 0", len(g.Edges))
	}
	if g.Stats.BranchPoints != 0 {
		t.Errorf("BranchPoints = %d, want 0", g.Stats.BranchPoints)
	}
}

func TestAggregate_RepeatedToolInSingleTrace(t *testing.T) {
	// A trace can call the same tool twice (e.g., search -> filter -> search).
	// Node 'search' should have count=2 (visited twice), edge search->filter count=1,
	// edge filter->search count=1.
	traces := [][]snapshot.Step{
		toolCallSteps("search", "filter", "search"),
	}
	g := Aggregate(traces)

	if len(g.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2 (search, filter)", len(g.Nodes))
	}
	if n, _ := findNode(g, "search"); n.Count != 2 {
		t.Errorf("node search count = %d, want 2", n.Count)
	}
	if n, _ := findNode(g, "filter"); n.Count != 1 {
		t.Errorf("node filter count = %d, want 1", n.Count)
	}
	if e, _ := findEdge(g, "search", "filter"); e.Count != 1 {
		t.Errorf("edge search->filter count = %d, want 1", e.Count)
	}
	if e, _ := findEdge(g, "filter", "search"); e.Count != 1 {
		t.Errorf("edge filter->search count = %d, want 1", e.Count)
	}
}

func TestAggregate_FivetraceSyntheticBaseline(t *testing.T) {
	// Per spec's testing chunk row: "synthetic 5-trace baseline returns expected node count + edge weights".
	// 3 traces go search->filter->summarize, 2 traces go search->lookup->answer.
	traces := [][]snapshot.Step{
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "filter", "summarize"),
		toolCallSteps("search", "lookup", "answer"),
		toolCallSteps("search", "lookup", "answer"),
	}
	g := Aggregate(traces)

	if g.Stats.TotalRuns != 5 {
		t.Errorf("TotalRuns = %d, want 5", g.Stats.TotalRuns)
	}
	if g.Stats.BranchPoints != 1 {
		t.Errorf("BranchPoints = %d, want 1 (search)", g.Stats.BranchPoints)
	}
	// Unique tools: search, filter, summarize, lookup, answer = 5.
	if len(g.Nodes) != 5 {
		t.Fatalf("len(Nodes) = %d, want 5", len(g.Nodes))
	}
	if n, _ := findNode(g, "search"); n.Count != 5 {
		t.Errorf("node search count = %d, want 5", n.Count)
	}
	if e, _ := findEdge(g, "search", "filter"); e.Count != 3 || e.Weight != 0.6 {
		t.Errorf("edge search->filter: count=%d weight=%f; want 3/0.6", e.Count, e.Weight)
	}
	if e, _ := findEdge(g, "search", "lookup"); e.Count != 2 || e.Weight != 0.4 {
		t.Errorf("edge search->lookup: count=%d weight=%f; want 2/0.4", e.Count, e.Weight)
	}
}
