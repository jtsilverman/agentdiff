package cluster

import "sort"

// Cluster represents a group of similar sequences found by DBSCAN.
type Cluster struct {
	ID       int   // cluster index, -1 for noise
	Members  []int // indices into input slice
	Exemplar int   // most central member (min avg distance to others)
}

// DBSCANResult holds the output of a DBSCAN clustering run.
type DBSCANResult struct {
	Clusters []Cluster
	Noise    []int   // indices of noise points
	Epsilon  float64 // epsilon used (may be auto-selected)
	MinPts   int
}

// DistanceMatrix builds a symmetric N x N distance matrix from sequences
// using the provided distance function.
func DistanceMatrix(sequences [][]string, distFn func(a, b []string) int) [][]int {
	n := len(sequences)
	matrix := make([][]int, n)
	for i := range matrix {
		matrix[i] = make([]int, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			d := distFn(sequences[i], sequences[j])
			matrix[i][j] = d
			matrix[j][i] = d
		}
	}
	return matrix
}

// DBSCAN performs density-based spatial clustering on a precomputed distance matrix.
//
// Points with fewer than minPts neighbors within epsilon distance are initially
// classified as noise. Border points (noise points found in a core point's
// neighborhood during expansion) are added to the expanding cluster.
func DBSCAN(distMatrix [][]int, epsilon float64, minPts int) DBSCANResult {
	n := len(distMatrix)
	if n == 0 {
		return DBSCANResult{Epsilon: epsilon, MinPts: minPts}
	}

	const (
		unvisited = 0
		visited   = 1
	)

	state := make([]int, n)      // visited state
	clusterID := make([]int, n)  // assigned cluster, -1 = noise/unassigned
	for i := range clusterID {
		clusterID[i] = -1
	}

	currentCluster := 0

	regionQuery := func(p int) []int {
		var neighbors []int
		for q := 0; q < n; q++ {
			if float64(distMatrix[p][q]) <= epsilon {
				neighbors = append(neighbors, q)
			}
		}
		return neighbors
	}

	for i := 0; i < n; i++ {
		if state[i] == visited {
			continue
		}
		state[i] = visited
		neighbors := regionQuery(i)

		if len(neighbors) < minPts {
			// Mark as noise for now; may be reclaimed as border point later.
			continue
		}

		// Core point: start a new cluster.
		cID := currentCluster
		currentCluster++
		clusterID[i] = cID

		// Use a queue to expand the cluster.
		queue := make([]int, len(neighbors))
		copy(queue, neighbors)

		for len(queue) > 0 {
			q := queue[0]
			queue = queue[1:]

			if state[q] != visited {
				state[q] = visited
				qNeighbors := regionQuery(q)
				if len(qNeighbors) >= minPts {
					queue = append(queue, qNeighbors...)
				}
			}

			// If q is not yet in any cluster (noise or unassigned), add to this cluster.
			if clusterID[q] == -1 {
				clusterID[q] = cID
			}
		}
	}

	// Build result.
	clusterMap := make(map[int][]int) // cluster ID -> member indices
	var noise []int
	for i, c := range clusterID {
		if c == -1 {
			noise = append(noise, i)
		} else {
			clusterMap[c] = append(clusterMap[c], i)
		}
	}

	clusters := make([]Cluster, 0, len(clusterMap))
	for id := 0; id < currentCluster; id++ {
		members, ok := clusterMap[id]
		if !ok {
			continue
		}
		sort.Ints(members)
		exemplar := findExemplar(members, distMatrix)
		clusters = append(clusters, Cluster{
			ID:       id,
			Members:  members,
			Exemplar: exemplar,
		})
	}

	sort.Ints(noise)

	return DBSCANResult{
		Clusters: clusters,
		Noise:    noise,
		Epsilon:  epsilon,
		MinPts:   minPts,
	}
}

// findExemplar returns the member index with the minimum average distance
// to all other members in the cluster.
func findExemplar(members []int, distMatrix [][]int) int {
	if len(members) <= 1 {
		return members[0]
	}

	bestIdx := members[0]
	bestAvg := -1.0

	for _, i := range members {
		sum := 0
		for _, j := range members {
			sum += distMatrix[i][j]
		}
		avg := float64(sum) / float64(len(members))
		if bestAvg < 0 || avg < bestAvg {
			bestAvg = avg
			bestIdx = i
		}
	}

	return bestIdx
}
