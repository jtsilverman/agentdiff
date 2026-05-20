// Package graph builds aggregate path graphs from baseline traces and overlays
// individual traces against them. A node is a tool call by name; an edge is a
// directed transition from one tool call to the next within a trace. Edge weight
// is the fraction of outgoing transitions from the source node that took this edge.
package graph

import (
	"sort"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// Node represents one tool call in the aggregate path graph.
//
// CostTokens / LatencyMs are summed across every step that invoked this tool
// in the baseline's traces, ignoring steps whose adapter did not surface the
// underlying field. They stay nil when no step contributed any value, so the
// frontend heatmap can distinguish "unknown" from "zero."
type Node struct {
	ID         string `json:"id"`
	ToolName   string `json:"tool_name"`
	Count      int    `json:"count"`
	CostTokens *int   `json:"cost_tokens,omitempty"`
	LatencyMs  *int   `json:"latency_ms,omitempty"`
}

// Edge represents a directed transition between two tool calls.
// Weight is the share of outgoing transitions from From that took this edge
// (count / sum of outgoing edge counts from From), in [0.0, 1.0].
type Edge struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Count  int     `json:"count"`
	Weight float64 `json:"weight"`
}

// Stats summarizes the aggregate graph.
type Stats struct {
	TotalRuns    int `json:"total_runs"`
	BranchPoints int `json:"branch_points"`
}

// PathGraph is the aggregate path graph for a baseline.
type PathGraph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
	Stats Stats  `json:"stats"`
}

// Aggregate walks each trace's tool-call sequence and builds the path graph.
// Non-tool-call steps are skipped (matching diff.ExtractToolNames). Node ID is
// the tool name. Node.Count is total visits across all traces. Edge.Count is
// total adjacent transitions across all traces. Edge.Weight is normalized per
// source-node outgoing total.
func Aggregate(traces [][]snapshot.Step) PathGraph {
	nodeCounts := make(map[string]int)
	edgeCounts := make(map[string]map[string]int) // from -> to -> count

	// Per-tool sums. Tracked via *int so a tool with all-nil contributions
	// stays nil (distinguishable from a zero-cost tool).
	costSums := make(map[string]int)
	costSeen := make(map[string]bool)
	latencySums := make(map[string]int)
	latencySeen := make(map[string]bool)

	for _, steps := range traces {
		names := diff.ExtractToolNames(steps)
		for i, name := range names {
			nodeCounts[name]++
			if i > 0 {
				prev := names[i-1]
				if edgeCounts[prev] == nil {
					edgeCounts[prev] = make(map[string]int)
				}
				edgeCounts[prev][name]++
			}
		}
		// Walk steps again to accumulate per-tool cost/latency. We don't use
		// ExtractToolNames here because we need the original Step structs to
		// reach CostTokens / LatencyMs; tool_call steps carry the values.
		for _, s := range steps {
			if s.ToolCall == nil {
				continue
			}
			tool := s.ToolCall.Name
			if s.CostTokens != nil {
				costSums[tool] += *s.CostTokens
				costSeen[tool] = true
			}
			if s.LatencyMs != nil {
				latencySums[tool] += *s.LatencyMs
				latencySeen[tool] = true
			}
		}
	}

	nodes := make([]Node, 0, len(nodeCounts))
	for id, count := range nodeCounts {
		n := Node{ID: id, ToolName: id, Count: count}
		if costSeen[id] {
			v := costSums[id]
			n.CostTokens = &v
		}
		if latencySeen[id] {
			v := latencySums[id]
			n.LatencyMs = &v
		}
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	edges := make([]Edge, 0)
	branchPoints := 0
	for from, targets := range edgeCounts {
		if len(targets) > 1 {
			branchPoints++
		}
		var outgoingTotal int
		for _, c := range targets {
			outgoingTotal += c
		}
		for to, count := range targets {
			weight := 0.0
			if outgoingTotal > 0 {
				weight = float64(count) / float64(outgoingTotal)
			}
			edges = append(edges, Edge{From: from, To: to, Count: count, Weight: weight})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	return PathGraph{
		Nodes: nodes,
		Edges: edges,
		Stats: Stats{
			TotalRuns:    len(traces),
			BranchPoints: branchPoints,
		},
	}
}
