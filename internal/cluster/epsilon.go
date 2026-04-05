package cluster

import (
	"errors"
	"sort"
)

// AutoEpsilon selects an epsilon value for DBSCAN by finding the "elbow" in
// the sorted k-distance graph using the discrete second derivative.
//
// distMatrix must be a symmetric N x N distance matrix. minPts determines k
// (the k-th nearest neighbor distance for each point).
func AutoEpsilon(distMatrix [][]int, minPts int) (float64, error) {
	n := len(distMatrix)
	if n < 3 {
		return 0, errors.New("need at least 3 points for auto epsilon")
	}

	// For each point, collect its k-distance (distance to the k-th nearest neighbor).
	k := minPts
	kDistances := make([]int, n)
	for i := 0; i < n; i++ {
		dists := make([]int, 0, n-1)
		for j := 0; j < n; j++ {
			if j != i {
				dists = append(dists, distMatrix[i][j])
			}
		}
		sort.Ints(dists)
		idx := k - 1
		if idx >= len(dists) {
			idx = len(dists) - 1
		}
		kDistances[i] = dists[idx]
	}

	// Sort k-distances ascending.
	sort.Ints(kDistances)

	// Check if all k-distances are equal.
	allEqual := true
	for i := 1; i < len(kDistances); i++ {
		if kDistances[i] != kDistances[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		eps := float64(kDistances[0])
		if eps == 0 {
			return 1.0, nil
		}
		return eps, nil
	}

	// Compute discrete second derivative: d2[i] = kd[i+1] - 2*kd[i] + kd[i-1]
	// for i in [1, len-2].
	d2 := make([]int, len(kDistances)-2)
	for i := 1; i < len(kDistances)-1; i++ {
		d2[i-1] = kDistances[i+1] - 2*kDistances[i] + kDistances[i-1]
	}

	// Check if all second derivatives are zero (linear).
	allZero := true
	for _, v := range d2 {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		// Return median k-distance.
		mid := len(kDistances) / 2
		var eps float64
		if len(kDistances)%2 == 0 {
			eps = float64(kDistances[mid-1]+kDistances[mid]) / 2.0
		} else {
			eps = float64(kDistances[mid])
		}
		if eps == 0 {
			return 1.0, nil
		}
		return eps, nil
	}

	// Find index of maximum second derivative. The corresponding k-distance
	// (offset by 1 because d2 starts at index 1 of kDistances) is epsilon.
	maxIdx := 0
	for i := 1; i < len(d2); i++ {
		if d2[i] > d2[maxIdx] {
			maxIdx = i
		}
	}

	// d2[maxIdx] corresponds to kDistances[maxIdx+1].
	eps := float64(kDistances[maxIdx+1])
	if eps == 0 {
		return 1.0, nil
	}
	return eps, nil
}
