package bench

import (
	"math/rand/v2"
	"testing"
)

func TestGenerateDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Seed != 42 {
		t.Errorf("expected seed 42, got %d", cfg.Seed)
	}
	if cfg.NumTools != 8 {
		t.Errorf("expected 8 tools, got %d", cfg.NumTools)
	}
	if cfg.NumTraces != 50 {
		t.Errorf("expected 50 traces, got %d", cfg.NumTraces)
	}
	if cfg.NumStrategies != 4 {
		t.Errorf("expected 4 strategies, got %d", cfg.NumStrategies)
	}
	if len(cfg.ToolVocab) != 20 {
		t.Errorf("expected 20 tool vocab items, got %d", len(cfg.ToolVocab))
	}
	if len(cfg.TextVocab) != 30 {
		t.Errorf("expected 30 text vocab items, got %d", len(cfg.TextVocab))
	}
}

func TestGenerateBaseline(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewPCG(42, 0))
	snap := GenerateBaseline(cfg, rng)

	if snap.ID == "" {
		t.Error("expected non-empty ID")
	}
	if snap.Source != "bench-generator" {
		t.Errorf("expected source bench-generator, got %s", snap.Source)
	}

	// Should have 3 steps per tool: text + tool call + tool result = 24 steps for 8 tools
	expectedSteps := cfg.NumTools * 3
	if len(snap.Steps) != expectedSteps {
		t.Errorf("expected %d steps, got %d", expectedSteps, len(snap.Steps))
	}

	// Verify we have tool calls and text
	hasToolCall := false
	hasText := false
	for _, s := range snap.Steps {
		if s.ToolCall != nil {
			hasToolCall = true
		}
		if s.Role == "assistant" && s.ToolCall == nil && s.ToolResult == nil {
			hasText = true
		}
	}
	if !hasToolCall {
		t.Error("expected at least one tool call step")
	}
	if !hasText {
		t.Error("expected at least one text step")
	}
}

func TestGenerateStrategyTraces(t *testing.T) {
	cfg := DefaultConfig()
	traces := GenerateStrategyTraces(cfg)

	if len(traces) != cfg.NumTraces {
		t.Errorf("expected %d traces, got %d", cfg.NumTraces, len(traces))
	}

	// Verify strategies are distinct by checking tool sequences
	strategyTools := make(map[int][]string)
	for _, lt := range traces {
		if _, exists := strategyTools[lt.StrategyID]; !exists {
			var tools []string
			for _, s := range lt.Trace.Steps {
				if s.ToolCall != nil {
					tools = append(tools, s.ToolCall.Name)
				}
			}
			strategyTools[lt.StrategyID] = tools
		}
	}

	if len(strategyTools) != cfg.NumStrategies {
		t.Errorf("expected %d distinct strategies, got %d", cfg.NumStrategies, len(strategyTools))
	}

	// Verify at least two strategies have different tool sequences
	strategies := make([][]string, 0)
	for _, tools := range strategyTools {
		strategies = append(strategies, tools)
	}
	allSame := true
	for i := 1; i < len(strategies); i++ {
		if len(strategies[i]) != len(strategies[0]) {
			allSame = false
			break
		}
		for j := range strategies[i] {
			if strategies[i][j] != strategies[0][j] {
				allSame = false
				break
			}
		}
		if !allSame {
			break
		}
	}
	if allSame {
		t.Error("expected strategies to have different tool sequences")
	}
}

func TestGenerateMutations(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewPCG(99, 0))
	base := GenerateBaseline(cfg, rng)

	tests := []struct {
		name string
		fn   func() string // returns a summary to compare
	}{
		{
			"removal",
			func() string {
				r := rand.New(rand.NewPCG(1, 0))
				m := MutateRemoval(base, r)
				if len(m.Steps) >= len(base.Steps) {
					t.Error("removal should reduce step count")
				}
				return "ok"
			},
		},
		{
			"insertion",
			func() string {
				r := rand.New(rand.NewPCG(2, 0))
				m := MutateInsertion(base, r, cfg.ToolVocab)
				if len(m.Steps) <= len(base.Steps) {
					t.Error("insertion should increase step count")
				}
				return "ok"
			},
		},
		{
			"reorder",
			func() string {
				r := rand.New(rand.NewPCG(3, 0))
				m := MutateReorder(base, r)
				if len(m.Steps) != len(base.Steps) {
					t.Error("reorder should keep same step count")
				}
				// Check that at least one step moved
				changed := false
				for i := range m.Steps {
					if m.Steps[i].Content != base.Steps[i].Content {
						changed = true
						break
					}
				}
				if !changed {
					t.Error("reorder should change step order")
				}
				return "ok"
			},
		},
		{
			"substitution",
			func() string {
				r := rand.New(rand.NewPCG(4, 0))
				m := MutateSubstitution(base, r, cfg.ToolVocab)
				// At least one tool name should differ
				changed := false
				baseTools := toolCallIndices(base.Steps)
				mutTools := toolCallIndices(m.Steps)
				if len(baseTools) != len(mutTools) {
					t.Error("substitution should keep same number of tool calls")
				}
				for i := range baseTools {
					if base.Steps[baseTools[i]].ToolCall.Name != m.Steps[mutTools[i]].ToolCall.Name {
						changed = true
						break
					}
				}
				if !changed {
					t.Error("substitution should change at least one tool name")
				}
				return "ok"
			},
		},
		{
			"text_drift",
			func() string {
				r := rand.New(rand.NewPCG(5, 0))
				m := MutateTextDrift(base, r)
				changed := false
				for i := range m.Steps {
					if m.Steps[i].Content != base.Steps[i].Content {
						changed = true
						break
					}
				}
				if !changed {
					t.Error("text_drift should change text content")
				}
				return "ok"
			},
		},
		{
			"combined",
			func() string {
				r := rand.New(rand.NewPCG(6, 0))
				m := MutateCombined(base, r, cfg.ToolVocab)
				// Combined should differ from base
				if len(m.Steps) == len(base.Steps) {
					// Step count same, check content
					allSame := true
					for i := range m.Steps {
						if m.Steps[i].Content != base.Steps[i].Content {
							allSame = false
							break
						}
					}
					if allSame {
						t.Error("combined mutation should produce a different snapshot")
					}
				}
				return "ok"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn()
		})
	}

	// Verify deep copy: mutations should not modify the original
	originalID := base.ID
	originalStepCount := len(base.Steps)
	r := rand.New(rand.NewPCG(7, 0))
	_ = MutateRemoval(base, r)
	if base.ID != originalID || len(base.Steps) != originalStepCount {
		t.Error("mutation modified original snapshot (deep copy failure)")
	}
}

func TestGenerateVariance(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewPCG(123, 0))
	base := GenerateBaseline(cfg, rng)

	t.Run("args", func(t *testing.T) {
		r := rand.New(rand.NewPCG(10, 0))
		v := VarianceArgs(base, r)
		// Tool names should be identical
		baseTools := toolCallIndices(base.Steps)
		varTools := toolCallIndices(v.Steps)
		if len(baseTools) != len(varTools) {
			t.Error("variance_args should keep same number of tool calls")
		}
		for i := range baseTools {
			if base.Steps[baseTools[i]].ToolCall.Name != v.Steps[varTools[i]].ToolCall.Name {
				t.Error("variance_args should not change tool names")
			}
		}
		// But at least one arg should differ
		argChanged := false
		for i := range baseTools {
			ba := base.Steps[baseTools[i]].ToolCall.Args
			va := v.Steps[varTools[i]].ToolCall.Args
			for k := range ba {
				if ba[k] != va[k] {
					argChanged = true
					break
				}
			}
		}
		if !argChanged {
			t.Error("variance_args should change at least one arg")
		}
	})

	t.Run("text", func(t *testing.T) {
		r := rand.New(rand.NewPCG(11, 0))
		v := VarianceText(base, r)
		changed := false
		for i := range v.Steps {
			if v.Steps[i].Content != base.Steps[i].Content {
				changed = true
				break
			}
		}
		if !changed {
			t.Error("variance_text should change some text")
		}
	})

	t.Run("steps", func(t *testing.T) {
		r := rand.New(rand.NewPCG(12, 0))
		v := VarianceSteps(base, r)
		if len(v.Steps) == len(base.Steps) {
			// It's possible but unlikely with our seed; just check it ran without error
			t.Log("variance_steps happened to keep same count (acceptable)")
		}
		// Tool calls should be preserved
		baseToolCount := len(toolCallIndices(base.Steps))
		varToolCount := len(toolCallIndices(v.Steps))
		if baseToolCount != varToolCount {
			t.Error("variance_steps should not affect tool call steps")
		}
	})
}

func TestGenerateLabeledPairs(t *testing.T) {
	cfg := DefaultConfig()
	pairs := GenerateLabeledPairs(cfg)

	if len(pairs) != 90 {
		t.Fatalf("expected 90 pairs, got %d", len(pairs))
	}

	regressionCount := 0
	normalCount := 0
	mutTypeCounts := make(map[MutationType]int)

	for _, p := range pairs {
		if p.IsRegression {
			regressionCount++
			mutTypeCounts[p.MutationType]++
			if p.MutationType == "" {
				t.Error("regression pair should have a mutation type")
			}
		} else {
			normalCount++
			if p.MutationType != "" {
				t.Error("normal pair should have empty mutation type")
			}
		}
		if p.Baseline.ID == "" {
			t.Error("baseline should have non-empty ID")
		}
		if p.Candidate.ID == "" {
			t.Error("candidate should have non-empty ID")
		}
	}

	if regressionCount != 60 {
		t.Errorf("expected 60 regression pairs, got %d", regressionCount)
	}
	if normalCount != 30 {
		t.Errorf("expected 30 normal pairs, got %d", normalCount)
	}

	// Verify 10 per mutation type
	expectedMutTypes := []MutationType{MutRemoval, MutInsertion, MutReorder, MutSubstitution, MutTextDrift, MutCombined}
	for _, mt := range expectedMutTypes {
		if mutTypeCounts[mt] != 10 {
			t.Errorf("expected 10 pairs for %s, got %d", mt, mutTypeCounts[mt])
		}
	}
}

func TestGenerateDeterminism(t *testing.T) {
	cfg := DefaultConfig()

	// Generate pairs twice with same config
	pairs1 := GenerateLabeledPairs(cfg)
	pairs2 := GenerateLabeledPairs(cfg)

	if len(pairs1) != len(pairs2) {
		t.Fatalf("different lengths: %d vs %d", len(pairs1), len(pairs2))
	}

	for i := range pairs1 {
		if pairs1[i].Baseline.ID != pairs2[i].Baseline.ID {
			t.Errorf("pair %d: baseline IDs differ: %s vs %s", i, pairs1[i].Baseline.ID, pairs2[i].Baseline.ID)
			break
		}
		if pairs1[i].IsRegression != pairs2[i].IsRegression {
			t.Errorf("pair %d: regression labels differ", i)
			break
		}
		if pairs1[i].MutationType != pairs2[i].MutationType {
			t.Errorf("pair %d: mutation types differ", i)
			break
		}
		if len(pairs1[i].Candidate.Steps) != len(pairs2[i].Candidate.Steps) {
			t.Errorf("pair %d: candidate step counts differ: %d vs %d", i, len(pairs1[i].Candidate.Steps), len(pairs2[i].Candidate.Steps))
			break
		}
	}

	// Also test strategy traces
	traces1 := GenerateStrategyTraces(cfg)
	traces2 := GenerateStrategyTraces(cfg)
	if len(traces1) != len(traces2) {
		t.Fatalf("strategy trace lengths differ: %d vs %d", len(traces1), len(traces2))
	}
	for i := range traces1 {
		if traces1[i].StrategyID != traces2[i].StrategyID {
			t.Errorf("trace %d: strategy IDs differ", i)
			break
		}
		if traces1[i].Trace.ID != traces2[i].Trace.ID {
			t.Errorf("trace %d: trace IDs differ", i)
			break
		}
	}
}
