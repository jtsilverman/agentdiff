// Package render produces SVG and PNG images of path graphs for embedding in
// GitHub Action PR comments. The CLI side (cmd/agentdiff render-graph) wraps
// these primitives; the input shape is intentionally decoupled from the JSON
// types in internal/graph so render can stay free of API serialization
// concerns.
package render

import (
	"fmt"
	"strings"
)

// Node is one tool call in the path graph.
type Node struct {
	ID    string
	Label string
}

// Edge is a directed transition between two tool calls. Weight in [0.0, 1.0]
// represents the share of outgoing transitions from From; renderers may use it
// to vary stroke width.
type Edge struct {
	From   string
	To     string
	Weight float64
}

// Overlay describes which nodes and edges in the baseline graph were exercised
// by a single trace, and which source nodes the trace diverged at relative to
// the baseline majority path.
type Overlay struct {
	MatchedNodes    []string
	MatchedEdges    []string // canonical "from->to"
	DivergenceNodes []string
}

// Input is the full render input. A nil Overlay renders the baseline alone.
type Input struct {
	Nodes   []Node
	Edges   []Edge
	Overlay *Overlay
}

const (
	nodeWidth  = 100
	nodeHeight = 40
	gapX       = 40
	marginX    = 10
	marginY    = 30
)

// SVG returns the path graph as a standalone SVG document. Nodes are laid out
// left-to-right in input order at a fixed pitch; edges connect node centers.
func SVG(in Input) string {
	count := len(in.Nodes)
	width := marginX*2 + count*nodeWidth + max(0, count-1)*gapX
	height := marginY*2 + nodeHeight

	positions := make(map[string]int) // node ID -> center x
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		width, height, width, height)
	b.WriteString(`<style>
.node-box { fill: #fff; stroke: #333; stroke-width: 1.5; }
.node-box-divergent { fill: #fff; stroke: #d33; stroke-width: 2.5; }
.node-label { font-family: monospace; font-size: 14px; text-anchor: middle; dominant-baseline: middle; }
.edge { stroke: #888; stroke-width: 1.5; fill: none; }
</style>`)

	divergenceSet := make(map[string]bool)
	if in.Overlay != nil {
		for _, id := range in.Overlay.DivergenceNodes {
			divergenceSet[id] = true
		}
	}

	for i, n := range in.Nodes {
		x := marginX + i*(nodeWidth+gapX)
		cx := x + nodeWidth/2
		cy := marginY + nodeHeight/2
		positions[n.ID] = cx
		isDivergent := divergenceSet[n.ID]
		divAttr := ""
		boxClass := "node-box"
		if isDivergent {
			divAttr = ` data-divergence="true"`
			boxClass = "node-box node-box-divergent"
		}
		fmt.Fprintf(&b, `<g class="node" data-id="%s"%s>`, escape(n.ID), divAttr)
		fmt.Fprintf(&b, `<rect class="%s" x="%d" y="%d" width="%d" height="%d" rx="6"/>`,
			boxClass, x, marginY, nodeWidth, nodeHeight)
		fmt.Fprintf(&b, `<text class="node-label" x="%d" y="%d">%s</text>`,
			cx, cy, escape(n.Label))
		b.WriteString(`</g>`)
	}

	for _, e := range in.Edges {
		fromX, okFrom := positions[e.From]
		toX, okTo := positions[e.To]
		if !okFrom || !okTo {
			continue
		}
		y := marginY + nodeHeight/2
		fmt.Fprintf(&b, `<line class="edge" x1="%d" y1="%d" x2="%d" y2="%d"/>`,
			fromX+nodeWidth/2, y, toX-nodeWidth/2, y)
	}

	if in.Overlay != nil {
		for _, id := range in.Overlay.DivergenceNodes {
			cx, ok := positions[id]
			if !ok {
				continue
			}
			// Down-pointing arrow above the node box.
			tipY := marginY - 4
			topY := marginY - 18
			fmt.Fprintf(&b, `<polygon class="arrow" data-arrow-target="%s" points="%d,%d %d,%d %d,%d" fill="#d33" stroke="#a00" stroke-width="1"/>`,
				escape(id), cx-6, topY, cx+6, topY, cx, tipY)
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func escape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
