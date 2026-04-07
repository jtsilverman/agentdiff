package cluster

import (
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/diff"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// StrategyReport summarizes the clustering of a baseline into behavioral strategies.
type StrategyReport struct {
	BaselineName  string     `json:"baseline_name"`
	SnapshotCount int        `json:"snapshot_count"`
	Strategies    []Strategy `json:"strategies"`
	Noise         []string   `json:"noise"`
	Epsilon       float64    `json:"epsilon"`
}

// Strategy represents a cluster of snapshots that follow a similar tool-call pattern.
type Strategy struct {
	ID              int                       `json:"id"`
	Count           int                       `json:"count"`
	Exemplar        string                    `json:"exemplar"`
	ToolSeq         []string                  `json:"tool_sequence"`
	Members         []string                  `json:"members"`
	MetadataSummary map[string]map[string]int `json:"metadata_summary"`
}

// MatchResult describes whether a snapshot matches an existing strategy.
type MatchResult struct {
	Matched             bool   `json:"matched"`
	StrategyID          int    `json:"strategy_id"`
	Exemplar            string `json:"exemplar"`
	Distance            int    `json:"distance"`
	MaxIntraClusterDist int    `json:"max_intra_cluster_dist"`
}

// ClusterBaseline groups a baseline's snapshots into behavioral strategies using DBSCAN.
// If epsilon is 0, AutoEpsilon is used to select it. minPts controls the DBSCAN density threshold.
func ClusterBaseline(baseline snapshot.Baseline, epsilon float64, minPts int) (StrategyReport, error) {
	if minPts < 1 {
		return StrategyReport{}, fmt.Errorf("minPts must be at least 1 (got %d)", minPts)
	}
	minRequired := minPts + 1
	if minRequired < 3 {
		minRequired = 3
	}
	n := len(baseline.Snapshots)
	if n < minRequired {
		return StrategyReport{}, fmt.Errorf("baseline needs at least %d snapshots for clustering (has %d)", minRequired, n)
	}

	// Extract tool name sequences from each snapshot.
	seqs := make([][]string, n)
	for i, snap := range baseline.Snapshots {
		seqs[i] = diff.ExtractToolNames(snap.Steps)
	}

	// Build distance matrix.
	distMatrix := DistanceMatrix(seqs, func(a, b []string) int {
		return diff.Levenshtein(a, b)
	})

	// Auto-select epsilon if not provided.
	if epsilon == 0 {
		var err error
		epsilon, err = AutoEpsilon(distMatrix, minPts)
		if err != nil {
			return StrategyReport{}, fmt.Errorf("auto epsilon: %w", err)
		}
	}

	// Run DBSCAN.
	result := DBSCAN(distMatrix, epsilon, minPts)

	// Build strategies from clusters.
	strategies := make([]Strategy, len(result.Clusters))
	for i, c := range result.Clusters {
		members := make([]string, len(c.Members))
		metaSummary := make(map[string]map[string]int)
		for j, idx := range c.Members {
			members[j] = baseline.Snapshots[idx].Name
			for k, v := range baseline.Snapshots[idx].Metadata {
				if metaSummary[k] == nil {
					metaSummary[k] = make(map[string]int)
				}
				metaSummary[k][v]++
			}
		}
		strategies[i] = Strategy{
			ID:              c.ID,
			Count:           len(c.Members),
			Exemplar:        baseline.Snapshots[c.Exemplar].Name,
			ToolSeq:         seqs[c.Exemplar],
			Members:         members,
			MetadataSummary: metaSummary,
		}
	}

	// Collect noise snapshot names.
	noise := make([]string, len(result.Noise))
	for i, idx := range result.Noise {
		noise[i] = baseline.Snapshots[idx].Name
	}

	return StrategyReport{
		BaselineName:  baseline.Name,
		SnapshotCount: n,
		Strategies:    strategies,
		Noise:         noise,
		Epsilon:       epsilon,
	}, nil
}

// CompareToCluster clusters the baseline and checks whether snap matches any strategy.
// A snapshot matches a strategy if its distance to the exemplar is within the strategy's
// max intra-cluster distance.
func CompareToCluster(baseline snapshot.Baseline, snap snapshot.Snapshot, epsilon float64, minPts int) (MatchResult, error) {
	report, err := ClusterBaseline(baseline, epsilon, minPts)
	if err != nil {
		return MatchResult{}, err
	}

	newSeq := diff.ExtractToolNames(snap.Steps)

	// For each strategy, compute distance to exemplar and max intra-cluster distance.
	type candidate struct {
		strategyID   int
		exemplarName string
		dist         int
		maxIntraDist int
	}

	// Build a map from snapshot name to index for looking up sequences.
	nameToIdx := make(map[string]int, len(baseline.Snapshots))
	for i, s := range baseline.Snapshots {
		nameToIdx[s.Name] = i
	}

	// Extract sequences (recompute for member lookups).
	seqs := make([][]string, len(baseline.Snapshots))
	for i, s := range baseline.Snapshots {
		seqs[i] = diff.ExtractToolNames(s.Steps)
	}

	var candidates []candidate
	for _, strat := range report.Strategies {
		exemplarIdx := nameToIdx[strat.Exemplar]
		exemplarSeq := seqs[exemplarIdx]

		dist := diff.Levenshtein(newSeq, exemplarSeq)

		// Compute max intra-cluster distance (each member to exemplar).
		maxIntra := 0
		if len(strat.Members) <= 1 {
			maxIntra = int(report.Epsilon)
		} else {
			for _, memberName := range strat.Members {
				memberIdx := nameToIdx[memberName]
				d := diff.Levenshtein(seqs[memberIdx], exemplarSeq)
				if d > maxIntra {
					maxIntra = d
				}
			}
		}

		candidates = append(candidates, candidate{
			strategyID:   strat.ID,
			exemplarName: strat.Exemplar,
			dist:         dist,
			maxIntraDist: maxIntra,
		})
	}

	if len(candidates) == 0 {
		return MatchResult{Matched: false}, nil
	}

	// Find closest strategy.
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.dist < best.dist {
			best = c
		}
	}

	matched := best.dist <= best.maxIntraDist

	return MatchResult{
		Matched:             matched,
		StrategyID:          best.strategyID,
		Exemplar:            best.exemplarName,
		Distance:            best.dist,
		MaxIntraClusterDist: best.maxIntraDist,
	}, nil
}
