package render

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

// TestSVG_BaselineOnly_ContainsNodeLabels asserts that an SVG render of a small
// 3-node, 2-edge baseline graph includes the displayed label text for every node.
// First failing test for the render package; all later tests assume the SVG path
// works before exercising overlays or PNG rasterization.
func TestSVG_BaselineOnly_ContainsNodeLabels(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "bash", Label: "bash"},
			{ID: "grep", Label: "grep"},
			{ID: "write", Label: "write"},
		},
		Edges: []Edge{
			{From: "bash", To: "grep", Weight: 1.0},
			{From: "grep", To: "write", Weight: 1.0},
		},
	}

	svg := SVG(in)

	if !strings.HasPrefix(svg, "<svg") {
		t.Fatalf("expected SVG output to start with <svg, got: %q", svg[:min(80, len(svg))])
	}
	for _, label := range []string{"bash", "grep", "write"} {
		if !strings.Contains(svg, label) {
			t.Errorf("expected SVG to contain node label %q, but it did not. SVG was:\n%s", label, svg)
		}
	}
}

// TestSVG_WithOverlay_DivergenceNodeMarked asserts that a node listed in the
// overlay's DivergenceNodes carries a divergence marker (data attribute) and
// that an arrow element annotates that node. This is the visual contract the
// acceptance criterion ("PR comment includes a rendered PNG showing the
// divergence") depends on.
func TestSVG_WithOverlay_DivergenceNodeMarked(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "bash", Label: "bash"},
			{ID: "grep", Label: "grep"},
			{ID: "write", Label: "write"},
		},
		Edges: []Edge{
			{From: "bash", To: "grep", Weight: 1.0},
			{From: "grep", To: "write", Weight: 1.0},
		},
		Overlay: &Overlay{
			MatchedNodes:    []string{"bash", "grep"},
			MatchedEdges:    []string{"bash->grep"},
			DivergenceNodes: []string{"grep"},
		},
	}

	svg := SVG(in)

	// The grep node group must carry the divergence marker.
	if !strings.Contains(svg, `data-id="grep"`) {
		t.Fatalf("expected grep node group in SVG")
	}
	if !strings.Contains(svg, `data-divergence="true"`) {
		t.Errorf("expected SVG to mark divergence node with data-divergence=true. SVG was:\n%s", svg)
	}
	// An arrow element must reference the divergence node.
	if !strings.Contains(svg, `data-arrow-target="grep"`) {
		t.Errorf("expected SVG to include arrow annotation targeting grep. SVG was:\n%s", svg)
	}
}

// TestPNG_DecodesAsValidPNG confirms the rasterizer produces a valid PNG with
// the expected pixel dimensions. The acceptance criterion requires an embedded
// PNG in the PR comment; this is the proof that the bytes we emit are a real
// decodable image, not just a happy-path placeholder.
func TestPNG_DecodesAsValidPNG(t *testing.T) {
	in := Input{
		Nodes: []Node{
			{ID: "bash", Label: "bash"},
			{ID: "grep", Label: "grep"},
			{ID: "write", Label: "write"},
		},
		Edges: []Edge{
			{From: "bash", To: "grep", Weight: 1.0},
			{From: "grep", To: "write", Weight: 1.0},
		},
		Overlay: &Overlay{
			MatchedNodes:    []string{"bash", "grep"},
			MatchedEdges:    []string{"bash->grep"},
			DivergenceNodes: []string{"grep"},
		},
	}

	pngBytes, err := PNG(in)
	if err != nil {
		t.Fatalf("PNG returned error: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatalf("PNG returned empty bytes")
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("PNG.Decode failed: %v", err)
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("decoded PNG has zero-or-negative dimensions: %dx%d", b.Dx(), b.Dy())
	}
	// SVG dimensions for this 3-node input are width=400, height=100; the PNG
	// renders at the same logical size.
	if b.Dx() != 400 {
		t.Errorf("expected PNG width 400, got %d", b.Dx())
	}
	if b.Dy() != 100 {
		t.Errorf("expected PNG height 100, got %d", b.Dy())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
