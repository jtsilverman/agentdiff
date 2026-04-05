package cluster

import (
	"reflect"
	"testing"
)

// levenshtein computes the edit distance between two string slices.
func levenshtein(a, b []string) int {
	la, lb := len(a), len(b)
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			dp[i][j] = min(dp[i-1][j]+1, min(dp[i][j-1]+1, dp[i-1][j-1]+cost))
		}
	}
	return dp[la][lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestDBSCAN_TwoClusters(t *testing.T) {
	sequences := [][]string{
		{"a", "b"},
		{"a", "b", "c"},
		{"x", "y", "z", "w", "v"},
		{"x", "y", "z", "w"},
	}
	dm := DistanceMatrix(sequences, levenshtein)
	result := DBSCAN(dm, 2, 2)

	if len(result.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(result.Clusters))
	}
	if len(result.Noise) != 0 {
		t.Fatalf("expected 0 noise points, got %d", len(result.Noise))
	}

	// Cluster 0 should have indices 0,1 and cluster 1 should have 2,3
	// (or vice versa depending on traversal order, but since we iterate 0..N
	// and 0,1 are close, cluster 0 = {0,1}).
	c0 := result.Clusters[0].Members
	c1 := result.Clusters[1].Members
	if !reflect.DeepEqual(c0, []int{0, 1}) {
		t.Errorf("cluster 0 members = %v, want [0 1]", c0)
	}
	if !reflect.DeepEqual(c1, []int{2, 3}) {
		t.Errorf("cluster 1 members = %v, want [2 3]", c1)
	}
}

func TestDBSCAN_AllSame(t *testing.T) {
	sequences := [][]string{
		{"a", "b"},
		{"a", "b"},
		{"a", "b"},
	}
	dm := DistanceMatrix(sequences, levenshtein)
	result := DBSCAN(dm, 0, 2)

	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}
	if !reflect.DeepEqual(result.Clusters[0].Members, []int{0, 1, 2}) {
		t.Errorf("cluster members = %v, want [0 1 2]", result.Clusters[0].Members)
	}
	if len(result.Noise) != 0 {
		t.Errorf("expected 0 noise, got %d", len(result.Noise))
	}
}

func TestDBSCAN_AllNoise(t *testing.T) {
	sequences := [][]string{
		{"a"},
		{"b"},
		{"c"},
	}
	dm := DistanceMatrix(sequences, levenshtein)
	result := DBSCAN(dm, 0, 2)

	if len(result.Clusters) != 0 {
		t.Fatalf("expected 0 clusters, got %d", len(result.Clusters))
	}
	if !reflect.DeepEqual(result.Noise, []int{0, 1, 2}) {
		t.Errorf("noise = %v, want [0 1 2]", result.Noise)
	}
}

func TestDBSCAN_Empty(t *testing.T) {
	dm := DistanceMatrix(nil, levenshtein)
	result := DBSCAN(dm, 1, 2)

	if len(result.Clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(result.Clusters))
	}
	if len(result.Noise) != 0 {
		t.Errorf("expected 0 noise, got %d", len(result.Noise))
	}
	if result.Epsilon != 1 {
		t.Errorf("epsilon = %f, want 1", result.Epsilon)
	}
}

func TestDBSCAN_SinglePoint_MinPts1(t *testing.T) {
	sequences := [][]string{{"a"}}
	dm := DistanceMatrix(sequences, levenshtein)
	result := DBSCAN(dm, 1, 1)

	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}
	if !reflect.DeepEqual(result.Clusters[0].Members, []int{0}) {
		t.Errorf("cluster members = %v, want [0]", result.Clusters[0].Members)
	}
}

func TestDBSCAN_SinglePoint_MinPts2(t *testing.T) {
	sequences := [][]string{{"a"}}
	dm := DistanceMatrix(sequences, levenshtein)
	result := DBSCAN(dm, 1, 2)

	if len(result.Clusters) != 0 {
		t.Fatalf("expected 0 clusters, got %d", len(result.Clusters))
	}
	if !reflect.DeepEqual(result.Noise, []int{0}) {
		t.Errorf("noise = %v, want [0]", result.Noise)
	}
}

func TestDBSCAN_ExemplarCorrectness(t *testing.T) {
	// Create 3 points: distances are
	//   0-1: 1, 0-2: 3, 1-2: 2
	// All in one cluster with epsilon=3, minPts=1.
	// Point 1 has avg dist (1+2)/3 = 1.0
	// Point 0 has avg dist (1+3)/3 = 1.33
	// Point 2 has avg dist (3+2)/3 = 1.67
	// Exemplar should be point 1.
	dm := [][]int{
		{0, 1, 3},
		{1, 0, 2},
		{3, 2, 0},
	}
	result := DBSCAN(dm, 3, 1)

	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}
	if result.Clusters[0].Exemplar != 1 {
		t.Errorf("exemplar = %d, want 1", result.Clusters[0].Exemplar)
	}
}

func TestDBSCAN_BorderPoints(t *testing.T) {
	// Point 0 and 1 are core (within eps of each other and of point 2).
	// Point 2 is within eps of point 1 but has < minPts neighbors at eps=1.
	// Point 2 should be added as a border point, not noise.
	dm := [][]int{
		{0, 1, 2},
		{1, 0, 1},
		{2, 1, 0},
	}
	result := DBSCAN(dm, 1, 2)

	if len(result.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(result.Clusters))
	}
	if !reflect.DeepEqual(result.Clusters[0].Members, []int{0, 1, 2}) {
		t.Errorf("cluster members = %v, want [0 1 2]", result.Clusters[0].Members)
	}
	if len(result.Noise) != 0 {
		t.Errorf("expected 0 noise, got %v", result.Noise)
	}
}

func TestDistanceMatrix_Symmetric(t *testing.T) {
	sequences := [][]string{
		{"a", "b"},
		{"c"},
		{"a", "b", "c"},
	}
	dm := DistanceMatrix(sequences, levenshtein)

	for i := 0; i < len(dm); i++ {
		if dm[i][i] != 0 {
			t.Errorf("diagonal dm[%d][%d] = %d, want 0", i, i, dm[i][i])
		}
		for j := i + 1; j < len(dm); j++ {
			if dm[i][j] != dm[j][i] {
				t.Errorf("dm[%d][%d]=%d != dm[%d][%d]=%d", i, j, dm[i][j], j, i, dm[j][i])
			}
		}
	}
}
