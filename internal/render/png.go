package render

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var (
	bgColor        = color.RGBA{0xff, 0xff, 0xff, 0xff}
	nodeStroke     = color.RGBA{0x33, 0x33, 0x33, 0xff}
	divergentColor = color.RGBA{0xdd, 0x33, 0x33, 0xff}
	edgeColor      = color.RGBA{0x88, 0x88, 0x88, 0xff}
	textColor      = color.RGBA{0x11, 0x11, 0x11, 0xff}
)

// PNG renders the input graph as a PNG image. Layout matches SVG (same node
// pitch and dimensions) so the two surfaces stay visually consistent.
func PNG(in Input) ([]byte, error) {
	count := len(in.Nodes)
	width := marginX*2 + count*nodeWidth + max(0, count-1)*gapX
	height := marginY*2 + nodeHeight

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	positions := make(map[string]int)
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
		stroke := nodeStroke
		thickness := 2
		if divergenceSet[n.ID] {
			stroke = divergentColor
			thickness = 3
		}
		drawRectOutline(img, image.Rect(x, marginY, x+nodeWidth, marginY+nodeHeight), stroke, thickness)
		drawCenteredText(img, n.Label, cx, cy, textColor)
	}

	for _, e := range in.Edges {
		fromX, okFrom := positions[e.From]
		toX, okTo := positions[e.To]
		if !okFrom || !okTo {
			continue
		}
		y := marginY + nodeHeight/2
		drawHLine(img, fromX+nodeWidth/2, toX-nodeWidth/2, y, edgeColor)
	}

	if in.Overlay != nil {
		for _, id := range in.Overlay.DivergenceNodes {
			cx, ok := positions[id]
			if !ok {
				continue
			}
			drawDownArrow(img, cx, marginY-2, divergentColor)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawRectOutline(img *image.RGBA, r image.Rectangle, col color.Color, thickness int) {
	for t := 0; t < thickness; t++ {
		drawHLine(img, r.Min.X+t, r.Max.X-1-t, r.Min.Y+t, col)
		drawHLine(img, r.Min.X+t, r.Max.X-1-t, r.Max.Y-1-t, col)
		drawVLine(img, r.Min.X+t, r.Min.Y+t, r.Max.Y-1-t, col)
		drawVLine(img, r.Max.X-1-t, r.Min.Y+t, r.Max.Y-1-t, col)
	}
}

func drawHLine(img *image.RGBA, x1, x2, y int, col color.Color) {
	if x2 < x1 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		img.Set(x, y, col)
	}
}

func drawVLine(img *image.RGBA, x, y1, y2 int, col color.Color) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		img.Set(x, y, col)
	}
}

// drawDownArrow draws a small downward-pointing triangle with its tip at (cx, tipY).
func drawDownArrow(img *image.RGBA, cx, tipY int, col color.Color) {
	const half = 6
	const height = 12
	topY := tipY - height
	for dy := 0; dy < height; dy++ {
		// Width shrinks linearly from `half*2` at top to 1 at tip.
		w := half - (dy*half)/height
		y := topY + dy
		for x := cx - w; x <= cx+w; x++ {
			img.Set(x, y, col)
		}
	}
}

func drawCenteredText(img *image.RGBA, s string, cx, cy int, col color.Color) {
	face := basicfont.Face7x13
	// basicfont.Face7x13: 7px advance per char, 13px height. Center horizontally
	// by offsetting half the pixel width; vertically by half the cap height.
	width := len(s) * 7
	baseline := cy + 4
	x := cx - width/2
	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{C: col},
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(s)
}
