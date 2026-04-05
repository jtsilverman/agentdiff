package diff

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// AlignOp represents the type of alignment operation.
type AlignOp int

const (
	AlignMatch  AlignOp = iota // same tool at both positions
	AlignSubst                 // different tool
	AlignInsert                // tool in B only (not in A)
	AlignDelete                // tool in A only (not in B)
)

// AlignedPair represents one pair in the alignment.
type AlignedPair struct {
	IndexA int     `json:"index_a"` // -1 if insert
	IndexB int     `json:"index_b"` // -1 if delete
	Op     AlignOp `json:"op"`
	ToolA  string  `json:"tool_a,omitempty"`
	ToolB  string  `json:"tool_b,omitempty"`
}

// AlignResult holds the full alignment output.
type AlignResult struct {
	Pairs           []AlignedPair
	FirstDivergence int // index into Pairs, -1 if identical
	Diverged        bool
	RetryGroups     []RetryGroup
	RemapA          []int // remapA[collapsedIdx] = original step index
	RemapB          []int // remapB[collapsedIdx] = original step index
}

// RetryGroup records a collapsed run of retried tool calls.
type RetryGroup struct {
	ToolName string `json:"tool_name"`
	CountA   int    `json:"count_a"`
	CountB   int    `json:"count_b"`
	StartA   int    `json:"start_a"`
	StartB   int    `json:"start_b"`
}

// Align computes Levenshtein edit-distance alignment between two tool-name sequences.
// It builds a full DP matrix and backtraces to produce AlignedPair slices.
// Tie-breaking preference: match > substitute > delete > insert.
func Align(seqA, seqB []string) AlignResult {
	// Truncate to last MaxToolCalls if needed.
	if len(seqA) > MaxToolCalls {
		fmt.Fprintf(os.Stderr, "Warning: truncated %d steps to %d for alignment. Use --max-steps to increase.\n", len(seqA), MaxToolCalls)
		seqA = seqA[len(seqA)-MaxToolCalls:]
	}
	if len(seqB) > MaxToolCalls {
		fmt.Fprintf(os.Stderr, "Warning: truncated %d steps to %d for alignment. Use --max-steps to increase.\n", len(seqB), MaxToolCalls)
		seqB = seqB[len(seqB)-MaxToolCalls:]
	}

	la, lb := len(seqA), len(seqB)

	// Build full DP matrix.
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if seqA[i-1] == seqB[j-1] {
				cost = 0
			}
			sub := dp[i-1][j-1] + cost
			del := dp[i-1][j] + 1
			ins := dp[i][j-1] + 1

			min := sub
			if del < min {
				min = del
			}
			if ins < min {
				min = ins
			}
			dp[i][j] = min
		}
	}

	// Backtrace from [la][lb] to [0][0].
	var pairs []AlignedPair
	i, j := la, lb
	for i > 0 || j > 0 {
		var pair AlignedPair

		if i > 0 && j > 0 {
			cost := 1
			if seqA[i-1] == seqB[j-1] {
				cost = 0
			}
			diagVal := dp[i-1][j-1] + cost
			delVal := dp[i-1][j] + 1
			insVal := dp[i][j-1] + 1

			// Tie-breaking: match > substitute > delete > insert.
			// Match/substitute both come from diagonal.
			if cost == 0 && diagVal == dp[i][j] {
				// Match
				pair = AlignedPair{IndexA: i - 1, IndexB: j - 1, Op: AlignMatch, ToolA: seqA[i-1], ToolB: seqB[j-1]}
				i--
				j--
			} else if cost == 1 && diagVal == dp[i][j] {
				// Substitute (prefer over delete/insert)
				pair = AlignedPair{IndexA: i - 1, IndexB: j - 1, Op: AlignSubst, ToolA: seqA[i-1], ToolB: seqB[j-1]}
				i--
				j--
			} else if delVal == dp[i][j] {
				// Delete
				pair = AlignedPair{IndexA: i - 1, IndexB: -1, Op: AlignDelete, ToolA: seqA[i-1]}
				i--
			} else if insVal == dp[i][j] {
				// Insert
				pair = AlignedPair{IndexA: -1, IndexB: j - 1, Op: AlignInsert, ToolB: seqB[j-1]}
				j--
			}
		} else if i > 0 {
			pair = AlignedPair{IndexA: i - 1, IndexB: -1, Op: AlignDelete, ToolA: seqA[i-1]}
			i--
		} else {
			pair = AlignedPair{IndexA: -1, IndexB: j - 1, Op: AlignInsert, ToolB: seqB[j-1]}
			j--
		}

		pairs = append(pairs, pair)
	}

	// Reverse to get forward order.
	for l, r := 0, len(pairs)-1; l < r; l, r = l+1, r-1 {
		pairs[l], pairs[r] = pairs[r], pairs[l]
	}

	// Compute FirstDivergence.
	firstDiv := -1
	matchCount := 0
	for idx, p := range pairs {
		if p.Op == AlignMatch {
			matchCount++
		} else if firstDiv == -1 {
			firstDiv = idx
		}
	}

	// Compute Diverged: matchCount / max(len(seqA), len(seqB)) < 0.2.
	diverged := false
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	if maxLen > 0 {
		ratio := float64(matchCount) / float64(maxLen)
		diverged = ratio < 0.2
	}

	if pairs == nil {
		pairs = []AlignedPair{}
	}

	return AlignResult{
		Pairs:           pairs,
		FirstDivergence: firstDiv,
		Diverged:        diverged,
	}
}

// CollapseRetries collapses consecutive retry steps (same tool name AND same
// arguments) into a single representative. Returns the collapsed steps, a remap
// from collapsed index to original step index, and retry groups.
func CollapseRetries(steps []snapshot.Step) ([]snapshot.Step, []int, []RetryGroup) {
	if len(steps) == 0 {
		return []snapshot.Step{}, []int{}, []RetryGroup{}
	}

	var collapsed []snapshot.Step
	var remap []int
	var groups []RetryGroup

	i := 0
	for i < len(steps) {
		// Check if this step has a tool call.
		if steps[i].ToolCall == nil {
			collapsed = append(collapsed, steps[i])
			remap = append(remap, i)
			i++
			continue
		}

		// Find the run of consecutive steps with same tool name and same args.
		runStart := i
		runName := steps[i].ToolCall.Name
		runArgsKey := canonicalArgs(steps[i].ToolCall.Args)
		count := 1
		for i+count < len(steps) &&
			steps[i+count].ToolCall != nil &&
			steps[i+count].ToolCall.Name == runName &&
			canonicalArgs(steps[i+count].ToolCall.Args) == runArgsKey {
			count++
		}

		// Keep first representative.
		collapsed = append(collapsed, steps[runStart])
		remap = append(remap, runStart)

		// Record group if this was a retry (2+ consecutive).
		if count >= 2 {
			groups = append(groups, RetryGroup{
				ToolName: runName,
				CountA:   count,
				CountB:   0,
				StartA:   runStart,
				StartB:   -1,
			})
		}

		i += count
	}

	if groups == nil {
		groups = []RetryGroup{}
	}

	return collapsed, remap, groups
}

// canonicalArgs serializes a map to a deterministic JSON string for comparison.
func canonicalArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]interface{}, 0, len(keys)*2)
	m := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		m[k] = args[k]
	}

	// Use json.Marshal on the map, but Go's encoding/json sorts keys by default.
	_ = ordered
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(b)
}
