package diff

import (
	"math"
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestCompareText_IdenticalText(t *testing.T) {
	steps := []snapshot.Step{
		{Role: "assistant", Content: "The quick brown fox jumps over the lazy dog"},
	}

	result := CompareText(steps, steps)

	if result.Similarity != 1.0 {
		t.Errorf("identical text: expected similarity 1.0, got %f", result.Similarity)
	}
	if result.Score != 0.0 {
		t.Errorf("identical text: expected score 0.0, got %f", result.Score)
	}
}

func TestCompareText_CompletelyDifferent(t *testing.T) {
	a := []snapshot.Step{
		{Role: "assistant", Content: "alpha beta gamma delta epsilon"},
	}
	b := []snapshot.Step{
		{Role: "assistant", Content: "one two three four five six"},
	}

	result := CompareText(a, b)

	if result.Similarity != 0.0 {
		t.Errorf("completely different: expected similarity 0.0, got %f", result.Similarity)
	}
	if result.Score != 1.0 {
		t.Errorf("completely different: expected score 1.0, got %f", result.Score)
	}
}

func TestCompareText_PartialOverlap(t *testing.T) {
	a := []snapshot.Step{
		{Role: "assistant", Content: "the quick brown fox jumps"},
	}
	b := []snapshot.Step{
		{Role: "assistant", Content: "the quick red cat jumps high"},
	}

	result := CompareText(a, b)

	// Should have some similarity but not 0 or 1.
	if result.Similarity <= 0.0 || result.Similarity >= 1.0 {
		t.Errorf("partial overlap: expected 0 < similarity < 1, got %f", result.Similarity)
	}
	if result.Score <= 0.0 || result.Score >= 1.0 {
		t.Errorf("partial overlap: expected 0 < score < 1, got %f", result.Score)
	}
	// Score + Similarity should equal 1.0.
	if math.Abs((result.Score+result.Similarity)-1.0) > 0.0001 {
		t.Errorf("partial overlap: score + similarity should equal 1.0, got %f", result.Score+result.Similarity)
	}
}

func TestCompareText_BothEmpty(t *testing.T) {
	// No assistant steps at all.
	a := []snapshot.Step{
		{Role: "user", Content: "hello"},
	}
	b := []snapshot.Step{
		{Role: "user", Content: "world"},
	}

	result := CompareText(a, b)

	if result.Similarity != 1.0 {
		t.Errorf("both empty: expected similarity 1.0, got %f", result.Similarity)
	}
	if result.Score != 0.0 {
		t.Errorf("both empty: expected score 0.0, got %f", result.Score)
	}
}

func TestCompareText_OneEmpty(t *testing.T) {
	a := []snapshot.Step{
		{Role: "assistant", Content: "some text here"},
	}
	b := []snapshot.Step{
		{Role: "user", Content: "no assistant content"},
	}

	result := CompareText(a, b)

	if result.Similarity != 0.0 {
		t.Errorf("one empty: expected similarity 0.0, got %f", result.Similarity)
	}
	if result.Score != 1.0 {
		t.Errorf("one empty: expected score 1.0, got %f", result.Score)
	}

	// Reverse direction.
	result2 := CompareText(b, a)

	if result2.Similarity != 0.0 {
		t.Errorf("one empty (reverse): expected similarity 0.0, got %f", result2.Similarity)
	}
	if result2.Score != 1.0 {
		t.Errorf("one empty (reverse): expected score 1.0, got %f", result2.Score)
	}
}

func TestCompareText_MultipleAssistantSteps(t *testing.T) {
	a := []snapshot.Step{
		{Role: "assistant", Content: "first part"},
		{Role: "user", Content: "ignored"},
		{Role: "assistant", Content: "second part"},
	}
	b := []snapshot.Step{
		{Role: "assistant", Content: "first part second part"},
	}

	result := CompareText(a, b)

	// Concatenated text should be very similar.
	if result.Similarity < 0.5 {
		t.Errorf("multi-step: expected high similarity, got %f", result.Similarity)
	}
}

func TestCompareText_IgnoresNonAssistantRoles(t *testing.T) {
	a := []snapshot.Step{
		{Role: "user", Content: "user text"},
		{Role: "assistant", Content: "hello world foo bar"},
		{Role: "tool_call", Content: "tool stuff"},
	}
	b := []snapshot.Step{
		{Role: "system", Content: "system text"},
		{Role: "assistant", Content: "hello world foo bar"},
	}

	result := CompareText(a, b)

	if result.Similarity != 1.0 {
		t.Errorf("non-assistant ignored: expected similarity 1.0, got %f", result.Similarity)
	}
}

func TestCompareSteps_Basic(t *testing.T) {
	a := []snapshot.Step{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
		{Role: "user", Content: "bye"},
	}
	b := []snapshot.Step{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}

	result := CompareSteps(a, b)

	if result.CountA != 3 {
		t.Errorf("expected CountA=3, got %d", result.CountA)
	}
	if result.CountB != 2 {
		t.Errorf("expected CountB=2, got %d", result.CountB)
	}
	if result.Delta != 1 {
		t.Errorf("expected Delta=1, got %d", result.Delta)
	}
}

func TestCompareSteps_Equal(t *testing.T) {
	steps := []snapshot.Step{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}

	result := CompareSteps(steps, steps)

	if result.Delta != 0 {
		t.Errorf("equal steps: expected Delta=0, got %d", result.Delta)
	}
}

func TestCompareSteps_BothEmpty(t *testing.T) {
	result := CompareSteps(nil, nil)

	if result.CountA != 0 || result.CountB != 0 || result.Delta != 0 {
		t.Errorf("both empty: expected all zeros, got %+v", result)
	}
}
