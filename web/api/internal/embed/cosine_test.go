package embed

import (
	"math"
	"testing"
)

func TestCosine_IdenticalVectors(t *testing.T) {
	a := []float32{1, 2, 3, 4}
	b := []float32{1, 2, 3, 4}
	score, err := Cosine(a, b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if math.Abs(score-1.0) > 1e-6 {
		t.Errorf("identical vectors: expected ~1.0, got %v", score)
	}
}

func TestCosine_OrthogonalVectors(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	score, err := Cosine(a, b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if math.Abs(score) > 1e-6 {
		t.Errorf("orthogonal vectors: expected ~0.0, got %v", score)
	}
}

func TestCosine_OppositeVectors(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{-1, -2, -3}
	score, err := Cosine(a, b)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if math.Abs(score-(-1.0)) > 1e-6 {
		t.Errorf("opposite vectors: expected ~-1.0, got %v", score)
	}
}

func TestCosine_LengthMismatch(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2}
	_, err := Cosine(a, b)
	if err == nil {
		t.Errorf("expected error on length mismatch, got nil")
	}
}

func TestCosine_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	score, err := Cosine(a, b)
	if err != nil {
		t.Fatalf("expected no error on zero vector, got %v", err)
	}
	// Convention: cos(0, x) = 0 (no signal from a zero vector).
	if math.Abs(score) > 1e-6 {
		t.Errorf("zero-vector cosine: expected 0, got %v", score)
	}
}

func TestCosine_RankingOrder(t *testing.T) {
	// Acceptance scenario from prework: two related vectors should outrank
	// an unrelated vector against the same reference.
	ref := []float32{1, 1, 0, 0}
	related := []float32{0.9, 1.1, 0.1, 0}
	unrelated := []float32{0, 0, 1, 1}

	scoreRelated, err1 := Cosine(ref, related)
	scoreUnrelated, err2 := Cosine(ref, unrelated)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v / %v", err1, err2)
	}
	if scoreRelated <= scoreUnrelated {
		t.Errorf("expected scoreRelated (%v) > scoreUnrelated (%v)", scoreRelated, scoreUnrelated)
	}
}
