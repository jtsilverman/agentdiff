package diff

import (
	"testing"

	"github.com/jtsilverman/agentdiff/internal/config"
	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestCompare_VerdictPass(t *testing.T) {
	// Identical snapshots: all scores zero, delta zero.
	steps := []snapshot.Step{
		{Role: "assistant", Content: "hello world foo bar"},
		toolStep("read_file", map[string]interface{}{"path": "/foo.go"}),
	}
	a := snapshot.Snapshot{ID: "snap-1", Steps: steps}
	b := snapshot.Snapshot{ID: "snap-2", Steps: steps}

	result := Compare(a, b, config.DefaultConfig())

	if result.Overall != VerdictPass {
		t.Errorf("expected pass, got %s", result.Overall)
	}
	if result.Snapshot1 != "snap-1" {
		t.Errorf("expected snapshot_1=snap-1, got %s", result.Snapshot1)
	}
	if result.Snapshot2 != "snap-2" {
		t.Errorf("expected snapshot_2=snap-2, got %s", result.Snapshot2)
	}
}

func TestCompare_VerdictChanged(t *testing.T) {
	// Small differences that don't exceed any threshold.
	// Default thresholds: tool_score=0.3, text_score=0.5, step_delta=5.
	a := snapshot.Snapshot{
		ID: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "the quick brown fox jumps over the lazy dog today"},
			toolStep("read_file", map[string]interface{}{"path": "/foo.go"}),
			toolStep("write_file", map[string]interface{}{"path": "/bar.go"}),
		},
	}
	b := snapshot.Snapshot{
		ID: "snap-b",
		Steps: []snapshot.Step{
			// Slightly different text but high overlap in bigrams.
			{Role: "assistant", Content: "the quick brown fox jumps over the lazy cat today"},
			toolStep("read_file", map[string]interface{}{"path": "/foo.go"}),
			toolStep("write_file", map[string]interface{}{"path": "/bar.go"}),
			// One extra step: delta=1, under threshold of 5.
			{Role: "user", Content: "extra"},
		},
	}

	result := Compare(a, b, config.DefaultConfig())

	if result.Overall != VerdictChanged {
		t.Errorf("expected changed, got %s (toolScore=%f, textScore=%f, stepDelta=%d)",
			result.Overall, result.ToolDiff.Score, result.TextDiff.Score, result.StepsDiff.Delta)
	}
}

func TestCompare_VerdictRegression_ToolThreshold(t *testing.T) {
	// Completely different tool sequences: score will be 1.0, exceeds 0.3 threshold.
	a := snapshot.Snapshot{
		ID: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "same text here for both"},
			toolStep("read_file", map[string]interface{}{"path": "/a"}),
			toolStep("search", map[string]interface{}{"q": "x"}),
		},
	}
	b := snapshot.Snapshot{
		ID: "snap-b",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "same text here for both"},
			toolStep("deploy", map[string]interface{}{"env": "prod"}),
			toolStep("notify", map[string]interface{}{"msg": "done"}),
		},
	}

	result := Compare(a, b, config.DefaultConfig())

	if result.Overall != VerdictRegression {
		t.Errorf("expected regression (tool threshold), got %s (toolScore=%f)",
			result.Overall, result.ToolDiff.Score)
	}
}

func TestCompare_VerdictRegression_TextThreshold(t *testing.T) {
	// Completely different text: score will be 1.0, exceeds 0.5 threshold.
	a := snapshot.Snapshot{
		ID: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "alpha beta gamma delta epsilon zeta"},
		},
	}
	b := snapshot.Snapshot{
		ID: "snap-b",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "one two three four five six seven"},
		},
	}

	result := Compare(a, b, config.DefaultConfig())

	if result.Overall != VerdictRegression {
		t.Errorf("expected regression (text threshold), got %s (textScore=%f)",
			result.Overall, result.TextDiff.Score)
	}
}

func TestCompare_VerdictRegression_StepDeltaThreshold(t *testing.T) {
	// Same text and no tools, but step count differs by 6 (exceeds threshold of 5).
	aSteps := []snapshot.Step{
		{Role: "assistant", Content: "hello world foo bar"},
	}
	bSteps := make([]snapshot.Step, 0, 7)
	bSteps = append(bSteps, snapshot.Step{Role: "assistant", Content: "hello world foo bar"})
	for i := 0; i < 6; i++ {
		bSteps = append(bSteps, snapshot.Step{Role: "user", Content: "extra"})
	}

	a := snapshot.Snapshot{ID: "snap-a", Steps: aSteps}
	b := snapshot.Snapshot{ID: "snap-b", Steps: bSteps}

	result := Compare(a, b, config.DefaultConfig())

	if result.StepsDiff.Delta != 6 {
		t.Errorf("expected delta=6, got %d", result.StepsDiff.Delta)
	}
	if result.Overall != VerdictRegression {
		t.Errorf("expected regression (step delta), got %s (delta=%d)",
			result.Overall, result.StepsDiff.Delta)
	}
}

func TestCompare_CustomConfig(t *testing.T) {
	// Use very tight thresholds so even small changes trigger regression.
	cfg := config.Config{
		Thresholds: config.Thresholds{
			ToolScore: 0.01,
			TextScore: 0.01,
			StepDelta: 0,
		},
	}

	a := snapshot.Snapshot{
		ID: "snap-a",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "hello world foo bar baz"},
			toolStep("read_file", map[string]interface{}{"path": "/a"}),
		},
	}
	b := snapshot.Snapshot{
		ID: "snap-b",
		Steps: []snapshot.Step{
			{Role: "assistant", Content: "hello world foo bar baz"},
			toolStep("read_file", map[string]interface{}{"path": "/a"}),
			{Role: "user", Content: "extra step"},
		},
	}

	result := Compare(a, b, cfg)

	// Step delta is 1, exceeds threshold of 0.
	if result.Overall != VerdictRegression {
		t.Errorf("expected regression with tight config, got %s", result.Overall)
	}
}

func TestCompare_BothEmpty(t *testing.T) {
	a := snapshot.Snapshot{ID: "empty-a", Steps: nil}
	b := snapshot.Snapshot{ID: "empty-b", Steps: nil}

	result := Compare(a, b, config.DefaultConfig())

	if result.Overall != VerdictPass {
		t.Errorf("both empty: expected pass, got %s", result.Overall)
	}
}
