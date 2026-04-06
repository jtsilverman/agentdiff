package bench

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

// TraceConfig controls synthetic trace generation parameters.
type TraceConfig struct {
	Seed          int64
	NumTools      int      // tools per trace (default 8)
	NumTraces     int      // baseline traces to generate (default 50)
	NumStrategies int      // distinct behavioral strategies (default 4)
	ToolVocab     []string // pool of tool names
	TextVocab     []string // pool of text phrases
}

// MutationType identifies the kind of mutation applied to a trace.
type MutationType string

const (
	MutRemoval      MutationType = "removal"
	MutInsertion    MutationType = "insertion"
	MutReorder      MutationType = "reorder"
	MutSubstitution MutationType = "substitution"
	MutTextDrift    MutationType = "text_drift"
	MutCombined     MutationType = "combined"
)

// LabeledPair pairs a baseline snapshot with a (possibly mutated) candidate.
type LabeledPair struct {
	Baseline     snapshot.Snapshot
	Candidate    snapshot.Snapshot
	IsRegression bool
	MutationType MutationType // empty if not a regression
}

// LabeledTrace associates a trace with its behavioral strategy.
type LabeledTrace struct {
	Trace      snapshot.Snapshot
	StrategyID int
}

// DefaultConfig returns a TraceConfig with sensible defaults for benchmarking.
func DefaultConfig() TraceConfig {
	return TraceConfig{
		Seed:          42,
		NumTools:      8,
		NumTraces:     50,
		NumStrategies: 4,
		ToolVocab: []string{
			"read_file", "write_file", "search", "execute", "list_dir",
			"grep", "git_diff", "git_commit", "curl", "npm_install",
			"compile", "test_run", "lint", "format", "deploy",
			"db_query", "api_call", "parse_json", "encode_base64", "hash_sha256",
		},
		TextVocab: []string{
			"analyzing the code", "found a match", "checking dependencies",
			"reading configuration", "running tests", "deploying service",
			"compiling project", "linting source", "formatting output",
			"searching for pattern", "writing results", "updating state",
			"validating input", "parsing response", "encoding data",
			"fetching resource", "processing batch", "generating report",
			"cleaning up", "initializing environment", "loading module",
			"scanning directory", "resolving conflicts", "merging branches",
			"installing packages", "building artifacts", "publishing release",
			"monitoring logs", "debugging issue", "optimizing query",
		},
	}
}

// newRng creates a deterministic rng from a seed.
func newRng(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), 0))
}

// deepCopySnapshot returns a deep copy of a snapshot.
func deepCopySnapshot(s snapshot.Snapshot) snapshot.Snapshot {
	cp := snapshot.Snapshot{
		ID:        s.ID,
		Name:      s.Name,
		Source:    s.Source,
		Timestamp: s.Timestamp,
	}
	if s.Metadata != nil {
		cp.Metadata = make(map[string]string, len(s.Metadata))
		for k, v := range s.Metadata {
			cp.Metadata[k] = v
		}
	}
	cp.Steps = make([]snapshot.Step, len(s.Steps))
	for i, step := range s.Steps {
		cp.Steps[i] = deepCopyStep(step)
	}
	return cp
}

func deepCopyStep(s snapshot.Step) snapshot.Step {
	cp := snapshot.Step{
		Role:    s.Role,
		Content: s.Content,
	}
	if s.ToolCall != nil {
		tc := &snapshot.ToolCall{
			Name: s.ToolCall.Name,
		}
		if s.ToolCall.Args != nil {
			tc.Args = make(map[string]interface{}, len(s.ToolCall.Args))
			for k, v := range s.ToolCall.Args {
				tc.Args[k] = v
			}
		}
		cp.ToolCall = tc
	}
	if s.ToolResult != nil {
		tr := &snapshot.ToolResult{
			Name:    s.ToolResult.Name,
			Output:  s.ToolResult.Output,
			IsError: s.ToolResult.IsError,
		}
		cp.ToolResult = tr
	}
	return cp
}

// GenerateBaseline creates one trace with N tool call steps interleaved with assistant text steps.
func GenerateBaseline(cfg TraceConfig, rng *rand.Rand) snapshot.Snapshot {
	numTools := cfg.NumTools
	if numTools <= 0 {
		numTools = 8
	}

	steps := make([]snapshot.Step, 0, numTools*2)
	for i := 0; i < numTools; i++ {
		// Assistant text step
		text := cfg.TextVocab[rng.IntN(len(cfg.TextVocab))]
		steps = append(steps, snapshot.Step{
			Role:    "assistant",
			Content: text,
		})

		// Tool call step
		toolName := cfg.ToolVocab[rng.IntN(len(cfg.ToolVocab))]
		steps = append(steps, snapshot.Step{
			Role:    "assistant",
			Content: fmt.Sprintf("calling %s", toolName),
			ToolCall: &snapshot.ToolCall{
				Name: toolName,
				Args: map[string]interface{}{
					"path":  fmt.Sprintf("file_%d.go", rng.IntN(100)),
					"query": cfg.TextVocab[rng.IntN(len(cfg.TextVocab))],
				},
			},
		})

		// Tool result step
		steps = append(steps, snapshot.Step{
			Role:    "tool",
			Content: fmt.Sprintf("result from %s: ok", toolName),
			ToolResult: &snapshot.ToolResult{
				Name:    toolName,
				Output:  fmt.Sprintf("success: processed %d items", rng.IntN(100)),
				IsError: false,
			},
		})
	}

	return snapshot.Snapshot{
		ID:        fmt.Sprintf("trace-%d", rng.IntN(1000000)),
		Name:      "synthetic-trace",
		Source:    "bench-generator",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Metadata:  map[string]string{"generator": "bench", "tools": fmt.Sprintf("%d", numTools)},
		Steps:     steps,
	}
}

// GenerateStrategyTraces generates traces across distinct behavioral strategies.
func GenerateStrategyTraces(cfg TraceConfig) []LabeledTrace {
	rng := newRng(cfg.Seed)
	numTraces := cfg.NumTraces
	if numTraces <= 0 {
		numTraces = 50
	}
	numStrategies := cfg.NumStrategies
	if numStrategies <= 0 {
		numStrategies = 4
	}

	// Build strategy templates: each strategy uses a fixed tool sequence.
	templates := make([][]string, numStrategies)
	for s := 0; s < numStrategies; s++ {
		tmpl := make([]string, cfg.NumTools)
		for t := 0; t < cfg.NumTools; t++ {
			tmpl[t] = cfg.ToolVocab[(s*cfg.NumTools+t)%len(cfg.ToolVocab)]
		}
		templates[s] = tmpl
	}

	traces := make([]LabeledTrace, 0, numTraces)
	for i := 0; i < numTraces; i++ {
		strategyID := i % numStrategies
		tmpl := templates[strategyID]

		steps := make([]snapshot.Step, 0, len(tmpl)*3)
		for _, toolName := range tmpl {
			// Text with minor variance
			text := cfg.TextVocab[rng.IntN(len(cfg.TextVocab))]
			steps = append(steps, snapshot.Step{
				Role:    "assistant",
				Content: text,
			})
			steps = append(steps, snapshot.Step{
				Role:    "assistant",
				Content: fmt.Sprintf("calling %s", toolName),
				ToolCall: &snapshot.ToolCall{
					Name: toolName,
					Args: map[string]interface{}{
						"path":  fmt.Sprintf("file_%d.go", rng.IntN(100)),
						"query": cfg.TextVocab[rng.IntN(len(cfg.TextVocab))],
					},
				},
			})
			steps = append(steps, snapshot.Step{
				Role:    "tool",
				Content: fmt.Sprintf("result from %s", toolName),
				ToolResult: &snapshot.ToolResult{
					Name:   toolName,
					Output: fmt.Sprintf("ok: %d", rng.IntN(1000)),
				},
			})
		}

		snap := snapshot.Snapshot{
			ID:        fmt.Sprintf("strategy-%d-trace-%d", strategyID, i),
			Name:      fmt.Sprintf("strategy-%d", strategyID),
			Source:    "bench-generator",
			Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Metadata: map[string]string{
				"strategy": fmt.Sprintf("%d", strategyID),
			},
			Steps: steps,
		}
		traces = append(traces, LabeledTrace{
			Trace:      snap,
			StrategyID: strategyID,
		})
	}
	return traces
}

// toolCallIndices returns indices of steps that have a ToolCall.
func toolCallIndices(steps []snapshot.Step) []int {
	var indices []int
	for i, s := range steps {
		if s.ToolCall != nil {
			indices = append(indices, i)
		}
	}
	return indices
}

// textIndices returns indices of steps that are assistant text (no tool call/result).
func textIndices(steps []snapshot.Step) []int {
	var indices []int
	for i, s := range steps {
		if s.Role == "assistant" && s.ToolCall == nil && s.ToolResult == nil {
			indices = append(indices, i)
		}
	}
	return indices
}

// MutateRemoval drops 1-3 tool call steps from the snapshot.
func MutateRemoval(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := toolCallIndices(cp.Steps)
	if len(indices) == 0 {
		return cp
	}
	numRemove := 1 + rng.IntN(min(3, len(indices)))
	// Shuffle and pick first numRemove
	rng.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
	toRemove := make(map[int]bool, numRemove)
	for _, idx := range indices[:numRemove] {
		toRemove[idx] = true
	}
	newSteps := make([]snapshot.Step, 0, len(cp.Steps)-numRemove)
	for i, s := range cp.Steps {
		if !toRemove[i] {
			newSteps = append(newSteps, s)
		}
	}
	cp.Steps = newSteps
	return cp
}

// MutateInsertion inserts 1-3 new tool call steps at random positions.
func MutateInsertion(snap snapshot.Snapshot, rng *rand.Rand, vocab []string) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	numInsert := 1 + rng.IntN(3)
	for i := 0; i < numInsert; i++ {
		pos := rng.IntN(len(cp.Steps) + 1)
		toolName := vocab[rng.IntN(len(vocab))]
		newStep := snapshot.Step{
			Role:    "assistant",
			Content: fmt.Sprintf("inserting %s", toolName),
			ToolCall: &snapshot.ToolCall{
				Name: toolName,
				Args: map[string]interface{}{
					"path": fmt.Sprintf("inserted_%d.go", i),
				},
			},
		}
		cp.Steps = append(cp.Steps, snapshot.Step{}) // grow
		copy(cp.Steps[pos+1:], cp.Steps[pos:])
		cp.Steps[pos] = newStep
	}
	return cp
}

// MutateReorder swaps 2-4 adjacent tool call step pairs.
func MutateReorder(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := toolCallIndices(cp.Steps)
	if len(indices) < 2 {
		return cp
	}
	numSwaps := 2 + rng.IntN(min(3, len(indices)-1))
	for s := 0; s < numSwaps; s++ {
		idx := rng.IntN(len(indices) - 1)
		a, b := indices[idx], indices[idx+1]
		cp.Steps[a], cp.Steps[b] = cp.Steps[b], cp.Steps[a]
	}
	return cp
}

// MutateSubstitution replaces 1-3 tool names with different ones from vocab.
func MutateSubstitution(snap snapshot.Snapshot, rng *rand.Rand, vocab []string) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := toolCallIndices(cp.Steps)
	if len(indices) == 0 {
		return cp
	}
	numSub := 1 + rng.IntN(min(3, len(indices)))
	rng.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
	for _, idx := range indices[:numSub] {
		oldName := cp.Steps[idx].ToolCall.Name
		newName := vocab[rng.IntN(len(vocab))]
		// Ensure substitution is actually different
		for newName == oldName && len(vocab) > 1 {
			newName = vocab[rng.IntN(len(vocab))]
		}
		cp.Steps[idx].ToolCall.Name = newName
		cp.Steps[idx].Content = fmt.Sprintf("calling %s", newName)
	}
	return cp
}

// MutateTextDrift shuffles/replaces 30-60% of words in assistant text steps.
func MutateTextDrift(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := textIndices(cp.Steps)
	for _, idx := range indices {
		words := strings.Fields(cp.Steps[idx].Content)
		if len(words) == 0 {
			continue
		}
		driftPct := 0.3 + rng.Float64()*0.3 // 30-60%
		numDrift := max(1, int(float64(len(words))*driftPct))
		for d := 0; d < numDrift; d++ {
			pos := rng.IntN(len(words))
			// Replace with a random word
			words[pos] = fmt.Sprintf("drifted_%d", rng.IntN(10000))
		}
		cp.Steps[idx].Content = strings.Join(words, " ")
	}
	return cp
}

// MutateCombined applies 2-3 mutations from the other types.
func MutateCombined(snap snapshot.Snapshot, rng *rand.Rand, vocab []string) snapshot.Snapshot {
	mutators := []func(snapshot.Snapshot) snapshot.Snapshot{
		func(s snapshot.Snapshot) snapshot.Snapshot { return MutateRemoval(s, rng) },
		func(s snapshot.Snapshot) snapshot.Snapshot { return MutateInsertion(s, rng, vocab) },
		func(s snapshot.Snapshot) snapshot.Snapshot { return MutateReorder(s, rng) },
		func(s snapshot.Snapshot) snapshot.Snapshot { return MutateSubstitution(s, rng, vocab) },
		func(s snapshot.Snapshot) snapshot.Snapshot { return MutateTextDrift(s, rng) },
	}

	numMutations := 2 + rng.IntN(2) // 2 or 3
	// Shuffle mutators and apply first N
	rng.Shuffle(len(mutators), func(i, j int) { mutators[i], mutators[j] = mutators[j], mutators[i] })

	result := deepCopySnapshot(snap)
	for i := 0; i < numMutations; i++ {
		result = mutators[i](result)
	}
	return result
}

// VarianceArgs changes 1-2 arg values while keeping tool names identical.
func VarianceArgs(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := toolCallIndices(cp.Steps)
	if len(indices) == 0 {
		return cp
	}
	numChange := 1 + rng.IntN(min(2, len(indices)))
	rng.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
	for _, idx := range indices[:numChange] {
		if cp.Steps[idx].ToolCall.Args == nil {
			cp.Steps[idx].ToolCall.Args = make(map[string]interface{})
		}
		cp.Steps[idx].ToolCall.Args["path"] = fmt.Sprintf("variant_%d.go", rng.IntN(1000))
	}
	return cp
}

// VarianceText rephrases less than 15% of text content.
func VarianceText(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	indices := textIndices(cp.Steps)
	if len(indices) == 0 {
		return cp
	}
	// Change only 1 text step (to stay under 15%)
	idx := indices[rng.IntN(len(indices))]
	words := strings.Fields(cp.Steps[idx].Content)
	if len(words) > 0 {
		pos := rng.IntN(len(words))
		words[pos] = "rephrased"
		cp.Steps[idx].Content = strings.Join(words, " ")
	}
	return cp
}

// VarianceSteps adds or removes 1-2 non-tool assistant text steps.
func VarianceSteps(snap snapshot.Snapshot, rng *rand.Rand) snapshot.Snapshot {
	cp := deepCopySnapshot(snap)
	if rng.IntN(2) == 0 {
		// Add 1-2 text steps
		numAdd := 1 + rng.IntN(2)
		for i := 0; i < numAdd; i++ {
			pos := rng.IntN(len(cp.Steps) + 1)
			newStep := snapshot.Step{
				Role:    "assistant",
				Content: fmt.Sprintf("additional thought %d", rng.IntN(1000)),
			}
			cp.Steps = append(cp.Steps, snapshot.Step{})
			copy(cp.Steps[pos+1:], cp.Steps[pos:])
			cp.Steps[pos] = newStep
		}
	} else {
		// Remove 1-2 text-only steps
		indices := textIndices(cp.Steps)
		if len(indices) == 0 {
			return cp
		}
		numRemove := 1 + rng.IntN(min(2, len(indices)))
		rng.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
		toRemove := make(map[int]bool, numRemove)
		for _, idx := range indices[:numRemove] {
			toRemove[idx] = true
		}
		newSteps := make([]snapshot.Step, 0, len(cp.Steps))
		for i, s := range cp.Steps {
			if !toRemove[i] {
				newSteps = append(newSteps, s)
			}
		}
		cp.Steps = newSteps
	}
	return cp
}

// GenerateLabeledPairs generates 60 regression pairs (10 per mutation type)
// plus 30 normal-variance pairs (10 per variance type) for a total of 90 pairs.
func GenerateLabeledPairs(cfg TraceConfig) []LabeledPair {
	rng := newRng(cfg.Seed)
	pairs := make([]LabeledPair, 0, 90)

	mutTypes := []MutationType{MutRemoval, MutInsertion, MutReorder, MutSubstitution, MutTextDrift, MutCombined}

	// 60 regression pairs: 10 per mutation type
	for _, mt := range mutTypes {
		for i := 0; i < 10; i++ {
			base := GenerateBaseline(cfg, rng)
			base.ID = fmt.Sprintf("base-%d", len(pairs))

			var candidate snapshot.Snapshot
			switch mt {
			case MutRemoval:
				candidate = MutateRemoval(base, rng)
			case MutInsertion:
				candidate = MutateInsertion(base, rng, cfg.ToolVocab)
			case MutReorder:
				candidate = MutateReorder(base, rng)
			case MutSubstitution:
				candidate = MutateSubstitution(base, rng, cfg.ToolVocab)
			case MutTextDrift:
				candidate = MutateTextDrift(base, rng)
			case MutCombined:
				candidate = MutateCombined(base, rng, cfg.ToolVocab)
			}
			candidate.ID = base.ID + "-mut"

			pairs = append(pairs, LabeledPair{
				Baseline:     base,
				Candidate:    candidate,
				IsRegression: true,
				MutationType: mt,
			})
		}
	}

	// 30 normal-variance pairs: 10 per variance type
	type varianceFunc struct {
		name string
		fn   func(snapshot.Snapshot, *rand.Rand) snapshot.Snapshot
	}
	variances := []varianceFunc{
		{"args", func(s snapshot.Snapshot, r *rand.Rand) snapshot.Snapshot { return VarianceArgs(s, r) }},
		{"text", func(s snapshot.Snapshot, r *rand.Rand) snapshot.Snapshot { return VarianceText(s, r) }},
		{"steps", func(s snapshot.Snapshot, r *rand.Rand) snapshot.Snapshot { return VarianceSteps(s, r) }},
	}
	for _, vf := range variances {
		for i := 0; i < 10; i++ {
			base := GenerateBaseline(cfg, rng)
			base.ID = fmt.Sprintf("base-%d", len(pairs))

			candidate := vf.fn(base, rng)
			candidate.ID = base.ID + "-var-" + vf.name

			pairs = append(pairs, LabeledPair{
				Baseline:     base,
				Candidate:    candidate,
				IsRegression: false,
				MutationType: "",
			})
		}
	}

	return pairs
}
