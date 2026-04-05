package diff

import (
	"fmt"
	"sort"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

const maxToolCalls = 1000

// CompareTools computes a ToolDiffResult between two step sequences.
// It extracts tool call names, computes edit distance, detects added/removed
// tools, checks for reordering, and scores argument similarity.
func CompareTools(a, b []snapshot.Step) ToolDiffResult {
	seqA := extractToolNames(a)
	seqB := extractToolNames(b)

	// Truncate to last 1000 if needed.
	if len(seqA) > maxToolCalls {
		seqA = seqA[len(seqA)-maxToolCalls:]
	}
	if len(seqB) > maxToolCalls {
		seqB = seqB[len(seqB)-maxToolCalls:]
	}

	// Both empty: identical.
	if len(seqA) == 0 && len(seqB) == 0 {
		return ToolDiffResult{
			Added:   []string{},
			Removed: []string{},
			Score:   0.0,
		}
	}

	// One empty, one not: completely different.
	if len(seqA) == 0 || len(seqB) == 0 {
		added, removed := setDifference(seqA, seqB)
		maxLen := len(seqA)
		if len(seqB) > maxLen {
			maxLen = len(seqB)
		}
		return ToolDiffResult{
			Added:    added,
			Removed:  removed,
			EditDist: maxLen,
			Score:    1.0,
		}
	}

	editDist := levenshtein(seqA, seqB)
	added, removed := setDifference(seqA, seqB)
	reordered := detectReordered(seqA, seqB)

	// Sequence score = edit_distance / max(len(a), len(b)).
	maxLen := len(seqA)
	if len(seqB) > maxLen {
		maxLen = len(seqB)
	}
	seqScore := float64(editDist) / float64(maxLen)
	if seqScore > 1.0 {
		seqScore = 1.0
	}

	// Argument score: compare args at aligned positions with matching names.
	argScore := computeArgScore(a, b, seqA, seqB)

	// Final score = weighted average.
	score := 0.6*seqScore + 0.4*argScore

	return ToolDiffResult{
		Added:     added,
		Removed:   removed,
		Reordered: reordered,
		EditDist:  editDist,
		Score:     score,
	}
}

// extractToolNames returns the ordered sequence of tool call names from steps.
func extractToolNames(steps []snapshot.Step) []string {
	var names []string
	for _, s := range steps {
		if s.ToolCall != nil {
			names = append(names, s.ToolCall.Name)
		}
	}
	return names
}

// levenshtein computes the edit distance between two string sequences.
func levenshtein(a, b []string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows to save memory.
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost

			min := ins
			if del < min {
				min = del
			}
			if sub < min {
				min = sub
			}
			curr[j] = min
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// setDifference computes added (in B but not A) and removed (in A but not B) tool names.
func setDifference(seqA, seqB []string) (added, removed []string) {
	setA := toSet(seqA)
	setB := toSet(seqB)

	for name := range setB {
		if !setA[name] {
			added = append(added, name)
		}
	}
	for name := range setA {
		if !setB[name] {
			removed = append(removed, name)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)

	if added == nil {
		added = []string{}
	}
	if removed == nil {
		removed = []string{}
	}

	return added, removed
}

// toSet converts a string slice to a set.
func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

// detectReordered returns true if both sequences have the same multiset of tool
// names but in a different order.
func detectReordered(seqA, seqB []string) bool {
	if len(seqA) != len(seqB) {
		return false
	}

	countsA := make(map[string]int)
	countsB := make(map[string]int)
	for _, n := range seqA {
		countsA[n]++
	}
	for _, n := range seqB {
		countsB[n]++
	}

	if len(countsA) != len(countsB) {
		return false
	}
	for k, v := range countsA {
		if countsB[k] != v {
			return false
		}
	}

	// Same multiset, but check if order differs.
	for i := range seqA {
		if seqA[i] != seqB[i] {
			return true
		}
	}
	return false
}

// computeArgScore compares arguments at aligned positions where tool names match.
// Returns 0.0 if all matching args are identical, 1.0 if completely different.
// If no aligned positions match by name, returns 0.0 (no arg signal).
func computeArgScore(a, b []snapshot.Step, seqA, seqB []string) float64 {
	toolCallsA := extractToolCalls(a)
	toolCallsB := extractToolCalls(b)

	// Truncate to match sequences.
	if len(toolCallsA) > maxToolCalls {
		toolCallsA = toolCallsA[len(toolCallsA)-maxToolCalls:]
	}
	if len(toolCallsB) > maxToolCalls {
		toolCallsB = toolCallsB[len(toolCallsB)-maxToolCalls:]
	}

	minLen := len(toolCallsA)
	if len(toolCallsB) < minLen {
		minLen = len(toolCallsB)
	}

	var totalSim float64
	var count int

	for i := 0; i < minLen; i++ {
		if toolCallsA[i].Name == toolCallsB[i].Name {
			sim := jaccardArgs(toolCallsA[i].Args, toolCallsB[i].Args)
			totalSim += sim
			count++
		}
	}

	if count == 0 {
		// No aligned positions with matching names: if sequences exist but
		// names never match, args are maximally different.
		if len(toolCallsA) > 0 || len(toolCallsB) > 0 {
			return 1.0
		}
		return 0.0
	}

	avgSim := totalSim / float64(count)
	// Invert: 1.0 similarity = 0.0 score (identical), 0.0 similarity = 1.0 score.
	return 1.0 - avgSim
}

// extractToolCalls returns ToolCall values from steps that have tool calls.
func extractToolCalls(steps []snapshot.Step) []snapshot.ToolCall {
	var calls []snapshot.ToolCall
	for _, s := range steps {
		if s.ToolCall != nil {
			calls = append(calls, *s.ToolCall)
		}
	}
	return calls
}

// jaccardArgs computes Jaccard similarity on JSON-serialized key-value pairs.
func jaccardArgs(a, b map[string]interface{}) float64 {
	setA := argPairSet(a)
	setB := argPairSet(b)

	if len(setA) == 0 && len(setB) == 0 {
		return 1.0
	}

	intersection := 0
	for k := range setA {
		if setB[k] {
			intersection++
		}
	}

	union := len(setA)
	for k := range setB {
		if !setA[k] {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// argPairSet serializes each key-value pair as "key=value" strings into a set.
func argPairSet(args map[string]interface{}) map[string]bool {
	s := make(map[string]bool, len(args))
	for k, v := range args {
		s[fmt.Sprintf("%s=%v", k, serializeValue(v))] = true
	}
	return s
}

// serializeValue converts a value to a stable string representation.
func serializeValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}


