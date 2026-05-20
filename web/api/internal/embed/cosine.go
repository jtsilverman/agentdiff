// Package embed holds the vector-space helpers used by the similarity search
// (chunk 22). Decoupled from the handlers package so the math is testable in
// isolation and reusable from any caller that holds two float32 slices.
package embed

import (
	"fmt"
	"math"
)

// Cosine returns the cosine similarity of two equal-length float32 vectors.
// Range: -1.0 (opposite) to 1.0 (identical). Zero vectors return 0 by convention.
// Returns an error when the vector lengths differ.
func Cosine(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("cosine: vector length mismatch (%d vs %d)", len(a), len(b))
	}
	var dot, normA, normB float64
	for i := range a {
		fa := float64(a[i])
		fb := float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		// Zero vector carries no direction; convention is no similarity signal.
		return 0, nil
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}
