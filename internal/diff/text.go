package diff

import (
	"strings"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// CompareText computes a TextDiffResult by comparing assistant-role text content
// between two step sequences using bigram Jaccard similarity.
func CompareText(a, b []snapshot.Step) TextDiffResult {
	textA := extractAssistantText(a)
	textB := extractAssistantText(b)

	// Both empty: identical.
	if textA == "" && textB == "" {
		return TextDiffResult{Similarity: 1.0, Score: 0.0}
	}

	// One empty: completely different.
	if textA == "" || textB == "" {
		return TextDiffResult{Similarity: 0.0, Score: 1.0}
	}

	tokensA := strings.Fields(textA)
	tokensB := strings.Fields(textB)

	bigramsA := buildBigrams(tokensA)
	bigramsB := buildBigrams(tokensB)

	// If both have fewer than 2 tokens, no bigrams exist.
	// Fall back to exact token match.
	if len(bigramsA) == 0 && len(bigramsB) == 0 {
		// Single-token case: compare directly.
		if textA == textB {
			return TextDiffResult{Similarity: 1.0, Score: 0.0}
		}
		return TextDiffResult{Similarity: 0.0, Score: 1.0}
	}

	similarity := jaccardBigrams(bigramsA, bigramsB)

	return TextDiffResult{
		Similarity: similarity,
		Score:      1.0 - similarity,
	}
}

// CompareSteps computes a StepsDiffResult by counting steps in each slice.
func CompareSteps(a, b []snapshot.Step) StepsDiffResult {
	countA := len(a)
	countB := len(b)
	delta := countA - countB
	if delta < 0 {
		delta = -delta
	}
	return StepsDiffResult{
		CountA: countA,
		CountB: countB,
		Delta:  delta,
	}
}

// Compare runs all comparisons and produces a DiffResult with an overall verdict.
func Compare(a, b snapshot.Snapshot, cfg config.Config) DiffResult {
	toolDiff := CompareTools(a.Steps, b.Steps)
	textDiff := CompareText(a.Steps, b.Steps)
	stepsDiff := CompareSteps(a.Steps, b.Steps)

	var overall Verdict
	switch {
	case toolDiff.Score > cfg.Thresholds.ToolScore ||
		textDiff.Score > cfg.Thresholds.TextScore ||
		stepsDiff.Delta > cfg.Thresholds.StepDelta:
		overall = VerdictRegression
	case toolDiff.Score > 0 || textDiff.Score > 0 || stepsDiff.Delta > 0:
		overall = VerdictChanged
	default:
		overall = VerdictPass
	}

	return DiffResult{
		Snapshot1: a.ID,
		Snapshot2: b.ID,
		Overall:   overall,
		ToolDiff:  toolDiff,
		TextDiff:  textDiff,
		StepsDiff: stepsDiff,
	}
}

// extractAssistantText concatenates all content from assistant-role steps.
func extractAssistantText(steps []snapshot.Step) string {
	var parts []string
	for _, s := range steps {
		if s.Role == "assistant" && s.Content != "" {
			parts = append(parts, s.Content)
		}
	}
	return strings.Join(parts, " ")
}

// buildBigrams returns a set of consecutive token pairs from a token slice.
func buildBigrams(tokens []string) map[string]bool {
	if len(tokens) < 2 {
		return nil
	}
	bigrams := make(map[string]bool, len(tokens)-1)
	for i := 0; i < len(tokens)-1; i++ {
		bigrams[tokens[i]+" "+tokens[i+1]] = true
	}
	return bigrams
}

// jaccardBigrams computes Jaccard similarity between two bigram sets.
func jaccardBigrams(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}

	union := len(a)
	for k := range b {
		if !a[k] {
			union++
		}
	}

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}
