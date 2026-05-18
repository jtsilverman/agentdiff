package graph

import (
	"sort"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// DivergencePoint describes a place in the trace where the chosen next-tool
// differs from the highest-weighted baseline edge at that source node.
// BaselinePct is the share of baseline outgoing transitions that took this
// trace's chosen edge (0.0 if the chosen edge isn't in the baseline at all).
type DivergencePoint struct {
	NodeID         string  `json:"node_id"`
	BaselinePct    float64 `json:"baseline_pct"`
	ThisTraceChose string  `json:"this_trace_chose"`
}

// OverlayResult is the per-trace overlay against an aggregate path graph.
// MatchedEdgeIDs use the "from->to" format.
type OverlayResult struct {
	MatchedNodeIDs   []string          `json:"matched_node_ids"`
	MatchedEdgeIDs   []string          `json:"matched_edge_ids"`
	DivergencePoints []DivergencePoint `json:"divergence_points"`
}

// edgeID returns the canonical string ID for an edge.
func edgeID(from, to string) string {
	return from + "->" + to
}

// Overlay computes which nodes and edges from the trace appear in the aggregate
// graph and where the trace diverged from the majority baseline path. A
// divergence point is recorded at each source node whose outgoing edge in the
// trace is not the highest-weighted outgoing edge in the aggregate.
func Overlay(g PathGraph, traceSteps []snapshot.Step) OverlayResult {
	// Index aggregate-graph nodes and edges for O(1) lookup.
	nodeSet := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeSet[n.ID] = true
	}
	edgeIndex := make(map[string]map[string]Edge) // from -> to -> Edge
	for _, e := range g.Edges {
		if edgeIndex[e.From] == nil {
			edgeIndex[e.From] = make(map[string]Edge)
		}
		edgeIndex[e.From][e.To] = e
	}

	names := diff.ExtractToolNames(traceSteps)

	matchedNodes := make(map[string]bool)
	matchedEdges := make(map[string]bool)
	divergences := make([]DivergencePoint, 0)

	for i, name := range names {
		if nodeSet[name] {
			matchedNodes[name] = true
		}
		if i == 0 {
			continue
		}
		prev := names[i-1]

		// Skip divergence detection if the source node isn't in the baseline
		// at all (no outgoing edges to compare against).
		outgoing, hasOutgoing := edgeIndex[prev]
		if !hasOutgoing {
			continue
		}

		// Find the top-weighted baseline edge from prev. Ties resolved by tool
		// name (alphabetical) for deterministic comparison.
		var topTarget string
		var topWeight float64
		targets := make([]string, 0, len(outgoing))
		for t := range outgoing {
			targets = append(targets, t)
		}
		sort.Strings(targets)
		for _, t := range targets {
			if outgoing[t].Weight > topWeight {
				topWeight = outgoing[t].Weight
				topTarget = t
			}
		}

		if e, ok := outgoing[name]; ok {
			matchedEdges[edgeID(prev, name)] = true
			if name != topTarget {
				divergences = append(divergences, DivergencePoint{
					NodeID:         prev,
					BaselinePct:    e.Weight,
					ThisTraceChose: name,
				})
			}
		} else {
			// Trace chose a novel target the baseline never took.
			divergences = append(divergences, DivergencePoint{
				NodeID:         prev,
				BaselinePct:    0.0,
				ThisTraceChose: name,
			})
		}
	}

	matchedNodeIDs := make([]string, 0, len(matchedNodes))
	for id := range matchedNodes {
		matchedNodeIDs = append(matchedNodeIDs, id)
	}
	sort.Strings(matchedNodeIDs)

	matchedEdgeIDs := make([]string, 0, len(matchedEdges))
	for id := range matchedEdges {
		matchedEdgeIDs = append(matchedEdgeIDs, id)
	}
	sort.Strings(matchedEdgeIDs)

	return OverlayResult{
		MatchedNodeIDs:   matchedNodeIDs,
		MatchedEdgeIDs:   matchedEdgeIDs,
		DivergencePoints: divergences,
	}
}
