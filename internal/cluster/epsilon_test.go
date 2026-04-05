package cluster

import (
	"testing"
)

func TestAutoEpsilon_TooFewPoints(t *testing.T) {
	// 2 points: should error.
	dm := [][]int{
		{0, 5},
		{5, 0},
	}
	_, err := AutoEpsilon(dm, 1)
	if err == nil {
		t.Fatal("expected error for 2 points, got nil")
	}
	if err.Error() != "need at least 3 points for auto epsilon" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	// 0 points.
	_, err = AutoEpsilon(nil, 1)
	if err == nil {
		t.Fatal("expected error for 0 points, got nil")
	}
}

func TestAutoEpsilon_ClearElbow(t *testing.T) {
	// Two tight clusters far apart.
	// Cluster A: indices 0,1,2 with sequences like ["a","b"] variations (edit dist ~1).
	// Cluster B: indices 3,4,5 with sequences like ["x","y","z"] variations (edit dist ~1).
	// Inter-cluster distance ~3.
	sequences := [][]string{
		{"a", "b"},
		{"a", "c"},
		{"a", "b", "c"},
		{"x", "y", "z"},
		{"x", "y", "w"},
		{"x", "y", "z", "w"},
	}
	dm := DistanceMatrix(sequences, levenshtein)

	eps, err := AutoEpsilon(dm, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Epsilon should be between intra-cluster max (~1-2) and inter-cluster min (~2-3).
	// The elbow should pick a value that separates them.
	if eps < 1 || eps > 3 {
		t.Errorf("epsilon = %f, want between 1 and 3 (inclusive)", eps)
	}
}

func TestAutoEpsilon_AllSame(t *testing.T) {
	// All points identical: all distances are 0 except diagonal (already 0).
	// k-distances all 0, should return 1.0 (minimum epsilon for zero case).
	dm := [][]int{
		{0, 0, 0},
		{0, 0, 0},
		{0, 0, 0},
	}
	eps, err := AutoEpsilon(dm, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eps != 1.0 {
		t.Errorf("epsilon = %f, want 1.0 (min epsilon for all-zero distances)", eps)
	}
}

func TestAutoEpsilon_AllEqualNonZero(t *testing.T) {
	// All distances equal (non-zero): equilateral triangle.
	dm := [][]int{
		{0, 5, 5},
		{5, 0, 5},
		{5, 5, 0},
	}
	eps, err := AutoEpsilon(dm, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eps != 5.0 {
		t.Errorf("epsilon = %f, want 5.0 (all equidistant)", eps)
	}
}

func TestAutoEpsilon_LinearGradient(t *testing.T) {
	// Points on a line: 0, 10, 20, 30, 40.
	// Distances are absolute differences.
	dm := [][]int{
		{0, 10, 20, 30, 40},
		{10, 0, 10, 20, 30},
		{20, 10, 0, 10, 20},
		{30, 20, 10, 0, 10},
		{40, 30, 20, 10, 0},
	}
	// With minPts=1, k-distances (nearest neighbor): [10, 10, 10, 10, 10].
	// All equal -> returns 10.
	eps, err := AutoEpsilon(dm, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eps != 10.0 {
		t.Errorf("epsilon = %f, want 10.0", eps)
	}

	// With minPts=2, k-distances (2nd nearest): [20, 10, 10, 10, 20].
	// Sorted: [10, 10, 10, 20, 20].
	// d2: [10-20+10=0, 20-20+10=10, 20-40+10=-10] -- not all zero, not all equal.
	// Max d2 at index 1 -> kDistances[2] = 10.
	eps, err = AutoEpsilon(dm, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eps != 10.0 {
		t.Errorf("epsilon = %f, want 10.0", eps)
	}
}

func TestAutoEpsilon_LinearSecondDerivative(t *testing.T) {
	// k-distances that form a perfect linear ramp: second derivative all zeros.
	// Use 5 points with carefully chosen distances so k-distances = [1,2,3,4,5].
	// Direct matrix construction.
	dm := [][]int{
		{0, 1, 2, 3, 4},
		{1, 0, 1, 2, 3},
		{2, 1, 0, 1, 2},
		{3, 2, 1, 0, 1},
		{4, 3, 2, 1, 0},
	}
	// minPts=2: 2nd nearest neighbor distances.
	// Point 0: dists=[1,2,3,4], 2nd=2
	// Point 1: dists=[1,1,2,3], 2nd=1
	// Point 2: dists=[1,1,2,2], 2nd=1
	// Point 3: dists=[1,1,2,3], 2nd=1
	// Point 4: dists=[1,2,3,4], 2nd=2
	// Sorted: [1,1,1,2,2]
	// d2: [1-2+1=0, 2-2+1=1, 2-4+1=-1] -- max at index 1, eps=kd[2]=1.
	eps, err := AutoEpsilon(dm, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eps != 1.0 {
		t.Errorf("epsilon = %f, want 1.0", eps)
	}
}

func TestAutoEpsilon_RoundTrip(t *testing.T) {
	// Verify that auto-epsilon produces reasonable clustering when fed to DBSCAN.
	// Two clear clusters + one outlier.
	sequences := [][]string{
		{"a", "b"},
		{"a", "c"},
		{"a", "b", "c"},
		{"x", "y", "z"},
		{"x", "y", "w"},
		{"x", "y", "z", "w"},
		{"q", "r", "s", "t", "u", "v"}, // outlier
	}
	dm := DistanceMatrix(sequences, levenshtein)
	minPts := 2

	eps, err := AutoEpsilon(dm, minPts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := DBSCAN(dm, eps, minPts)

	// Should produce at least 2 clusters.
	if len(result.Clusters) < 2 {
		t.Errorf("expected at least 2 clusters, got %d (eps=%f)", len(result.Clusters), eps)
	}

	// The outlier (index 6) should be noise or in its own cluster.
	inMainCluster := false
	for _, c := range result.Clusters {
		if len(c.Members) >= 2 {
			for _, m := range c.Members {
				if m == 6 {
					inMainCluster = true
				}
			}
		}
	}
	if inMainCluster {
		t.Error("outlier (index 6) should not be in a main cluster")
	}
}
