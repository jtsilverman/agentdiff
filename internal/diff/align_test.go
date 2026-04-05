package diff

import (
	"testing"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
)

func TestAlign_IdenticalSequences(t *testing.T) {
	seq := []string{"Read", "Write", "Bash"}
	r := Align(seq, seq)

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}
	for i, p := range r.Pairs {
		if p.Op != AlignMatch {
			t.Errorf("pair %d: expected AlignMatch, got %d", i, p.Op)
		}
		if p.IndexA != i || p.IndexB != i {
			t.Errorf("pair %d: expected indices (%d,%d), got (%d,%d)", i, i, i, p.IndexA, p.IndexB)
		}
	}
	if r.FirstDivergence != -1 {
		t.Errorf("expected FirstDivergence=-1, got %d", r.FirstDivergence)
	}
	if r.Diverged {
		t.Error("expected Diverged=false")
	}
}

func TestAlign_SingleInsertion(t *testing.T) {
	a := []string{"Read", "Write"}
	b := []string{"Read", "Bash", "Write"}
	r := Align(a, b)

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}

	// Read matches, Bash inserted, Write matches.
	expected := []struct {
		op    AlignOp
		toolA string
		toolB string
	}{
		{AlignMatch, "Read", "Read"},
		{AlignInsert, "", "Bash"},
		{AlignMatch, "Write", "Write"},
	}

	for i, e := range expected {
		if r.Pairs[i].Op != e.op {
			t.Errorf("pair %d: expected op %d, got %d", i, e.op, r.Pairs[i].Op)
		}
		if r.Pairs[i].ToolA != e.toolA {
			t.Errorf("pair %d: expected toolA=%q, got %q", i, e.toolA, r.Pairs[i].ToolA)
		}
		if r.Pairs[i].ToolB != e.toolB {
			t.Errorf("pair %d: expected toolB=%q, got %q", i, e.toolB, r.Pairs[i].ToolB)
		}
	}

	if r.Pairs[1].IndexA != -1 {
		t.Errorf("insert pair should have IndexA=-1, got %d", r.Pairs[1].IndexA)
	}
	if r.FirstDivergence != 1 {
		t.Errorf("expected FirstDivergence=1, got %d", r.FirstDivergence)
	}
}

func TestAlign_SingleDeletion(t *testing.T) {
	a := []string{"Read", "Bash", "Write"}
	b := []string{"Read", "Write"}
	r := Align(a, b)

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}

	if r.Pairs[0].Op != AlignMatch || r.Pairs[0].ToolA != "Read" {
		t.Errorf("pair 0: expected Match(Read), got op=%d tool=%q", r.Pairs[0].Op, r.Pairs[0].ToolA)
	}
	if r.Pairs[1].Op != AlignDelete || r.Pairs[1].ToolA != "Bash" {
		t.Errorf("pair 1: expected Delete(Bash), got op=%d tool=%q", r.Pairs[1].Op, r.Pairs[1].ToolA)
	}
	if r.Pairs[1].IndexB != -1 {
		t.Errorf("delete pair should have IndexB=-1, got %d", r.Pairs[1].IndexB)
	}
	if r.Pairs[2].Op != AlignMatch || r.Pairs[2].ToolA != "Write" {
		t.Errorf("pair 2: expected Match(Write), got op=%d tool=%q", r.Pairs[2].Op, r.Pairs[2].ToolA)
	}
}

func TestAlign_Substitution(t *testing.T) {
	a := []string{"Read", "Bash", "Write"}
	b := []string{"Read", "Grep", "Write"}
	r := Align(a, b)

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}
	if r.Pairs[1].Op != AlignSubst {
		t.Errorf("pair 1: expected AlignSubst, got %d", r.Pairs[1].Op)
	}
	if r.Pairs[1].ToolA != "Bash" || r.Pairs[1].ToolB != "Grep" {
		t.Errorf("pair 1: expected Bash/Grep, got %s/%s", r.Pairs[1].ToolA, r.Pairs[1].ToolB)
	}
	if r.FirstDivergence != 1 {
		t.Errorf("expected FirstDivergence=1, got %d", r.FirstDivergence)
	}
}

func TestAlign_CompleteDivergence(t *testing.T) {
	a := []string{"A", "B", "C", "D", "E"}
	b := []string{"V", "W", "X", "Y", "Z"}
	r := Align(a, b)

	if !r.Diverged {
		t.Error("expected Diverged=true for completely different sequences")
	}
	if r.FirstDivergence != 0 {
		t.Errorf("expected FirstDivergence=0, got %d", r.FirstDivergence)
	}
}

func TestAlign_EmptyBothSequences(t *testing.T) {
	r := Align([]string{}, []string{})
	if len(r.Pairs) != 0 {
		t.Errorf("expected 0 pairs, got %d", len(r.Pairs))
	}
	if r.FirstDivergence != -1 {
		t.Errorf("expected FirstDivergence=-1, got %d", r.FirstDivergence)
	}
	if r.Diverged {
		t.Error("expected Diverged=false for two empty sequences")
	}
}

func TestAlign_EmptyOneSequence(t *testing.T) {
	// A empty, B has elements.
	r := Align([]string{}, []string{"Read", "Write"})
	if len(r.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(r.Pairs))
	}
	for _, p := range r.Pairs {
		if p.Op != AlignInsert {
			t.Errorf("expected AlignInsert, got %d", p.Op)
		}
		if p.IndexA != -1 {
			t.Errorf("expected IndexA=-1, got %d", p.IndexA)
		}
	}
	if r.FirstDivergence != 0 {
		t.Errorf("expected FirstDivergence=0, got %d", r.FirstDivergence)
	}

	// B empty, A has elements.
	r2 := Align([]string{"Read", "Write"}, []string{})
	if len(r2.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(r2.Pairs))
	}
	for _, p := range r2.Pairs {
		if p.Op != AlignDelete {
			t.Errorf("expected AlignDelete, got %d", p.Op)
		}
		if p.IndexB != -1 {
			t.Errorf("expected IndexB=-1, got %d", p.IndexB)
		}
	}
}

func TestAlign_MixedOperations(t *testing.T) {
	a := []string{"A", "B", "C"}
	b := []string{"A", "X", "B"}
	r := Align(a, b)

	// Optimal: Match(A), Subst(B->X), Match(C->B)... no.
	// Actually: A=A(match), B vs X (subst or del+ins), C vs B...
	// DP: cost of matching A,A=0. Then B!=X cost=1 sub, B del + X ins = 2, etc.
	// Optimal alignment: A-A(match), sub(B,X), sub(C,B) = cost 2
	// OR: A-A(match), del(B), match(C... no C!=B), ins(X)... that's worse.
	// Let me think: dp[3][3] path.
	// Actually the best is: A-A(0), B-X(1), C-B(1) = 2 total. All same length.
	// But B appears in both, so maybe: A-A(0), del(B), ins(X), C-? no that doesn't work either.
	// With seqA=[A,B,C] seqB=[A,X,B]:
	// Match A-A, then remaining [B,C] vs [X,B].
	// [B,C] vs [X,B]: sub(B,X)+sub(C,B)=2, or del(B)+match(C..no)+ins(X)=bad
	// Or: ins(X)+match(B,B)+del(C) = 3. So 2 subs wins.

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}
	if r.Pairs[0].Op != AlignMatch {
		t.Errorf("pair 0: expected Match, got %d", r.Pairs[0].Op)
	}
	// Pairs 1 and 2 should both be substitutions.
	if r.Pairs[1].Op != AlignSubst {
		t.Errorf("pair 1: expected Subst, got %d", r.Pairs[1].Op)
	}
	if r.Pairs[2].Op != AlignSubst {
		t.Errorf("pair 2: expected Subst, got %d", r.Pairs[2].Op)
	}
}

func TestAlign_TieBreakingDeterminism(t *testing.T) {
	// ["A","B"] vs ["B","A"]: cost is 2 either way (2 subs, or del+ins combos).
	// With tie-breaking: match > substitute > delete > insert.
	// Diagonal (sub) preferred over delete/insert.
	r1 := Align([]string{"A", "B"}, []string{"B", "A"})
	r2 := Align([]string{"A", "B"}, []string{"B", "A"})

	if len(r1.Pairs) != len(r2.Pairs) {
		t.Fatalf("non-deterministic pair count: %d vs %d", len(r1.Pairs), len(r2.Pairs))
	}
	for i := range r1.Pairs {
		if r1.Pairs[i].Op != r2.Pairs[i].Op {
			t.Errorf("pair %d: non-deterministic op %d vs %d", i, r1.Pairs[i].Op, r2.Pairs[i].Op)
		}
		if r1.Pairs[i].IndexA != r2.Pairs[i].IndexA || r1.Pairs[i].IndexB != r2.Pairs[i].IndexB {
			t.Errorf("pair %d: non-deterministic indices", i)
		}
	}

	// Both should be substitutions (diagonal preferred over delete+insert).
	if len(r1.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(r1.Pairs))
	}
	for i, p := range r1.Pairs {
		if p.Op != AlignSubst {
			t.Errorf("pair %d: expected Subst (tie-break prefers diagonal), got %d", i, p.Op)
		}
	}
}

func TestCollapseRetries_SameArgs(t *testing.T) {
	args := map[string]interface{}{"path": "/foo/bar.go"}
	steps := []snapshot.Step{
		toolStep("Read", args),
		toolStep("Read", args),
		toolStep("Read", args),
		toolStep("Write", map[string]interface{}{"path": "/out.go"}),
	}

	collapsed, remap, groups := CollapseRetries(steps)

	if len(collapsed) != 2 {
		t.Fatalf("expected 2 collapsed steps, got %d", len(collapsed))
	}
	if collapsed[0].ToolCall.Name != "Read" {
		t.Errorf("expected first collapsed to be Read, got %s", collapsed[0].ToolCall.Name)
	}
	if collapsed[1].ToolCall.Name != "Write" {
		t.Errorf("expected second collapsed to be Write, got %s", collapsed[1].ToolCall.Name)
	}

	// Remap: collapsed[0] -> original 0, collapsed[1] -> original 3.
	if len(remap) != 2 {
		t.Fatalf("expected remap len 2, got %d", len(remap))
	}
	if remap[0] != 0 {
		t.Errorf("remap[0]: expected 0, got %d", remap[0])
	}
	if remap[1] != 3 {
		t.Errorf("remap[1]: expected 3, got %d", remap[1])
	}

	// One group for the 3 Reads.
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].ToolName != "Read" {
		t.Errorf("group tool: expected Read, got %s", groups[0].ToolName)
	}
	if groups[0].CountA != 3 {
		t.Errorf("group count: expected 3, got %d", groups[0].CountA)
	}
	if groups[0].StartA != 0 {
		t.Errorf("group startA: expected 0, got %d", groups[0].StartA)
	}
}

func TestCollapseRetries_DifferentArgs(t *testing.T) {
	steps := []snapshot.Step{
		toolStep("Read", map[string]interface{}{"path": "/a.go"}),
		toolStep("Read", map[string]interface{}{"path": "/b.go"}),
		toolStep("Read", map[string]interface{}{"path": "/c.go"}),
	}

	collapsed, remap, groups := CollapseRetries(steps)

	// Different args: no collapse.
	if len(collapsed) != 3 {
		t.Fatalf("expected 3 collapsed steps (no collapse), got %d", len(collapsed))
	}
	if len(remap) != 3 {
		t.Fatalf("expected remap len 3, got %d", len(remap))
	}
	for i := 0; i < 3; i++ {
		if remap[i] != i {
			t.Errorf("remap[%d]: expected %d, got %d", i, i, remap[i])
		}
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestCollapseRetries_RemapCorrectness(t *testing.T) {
	args := map[string]interface{}{"cmd": "ls"}
	steps := []snapshot.Step{
		toolStep("Bash", args),
		toolStep("Bash", args),
		toolStep("Read", map[string]interface{}{"path": "/x"}),
		toolStep("Write", map[string]interface{}{"path": "/y"}),
		toolStep("Write", map[string]interface{}{"path": "/y"}),
	}

	collapsed, remap, groups := CollapseRetries(steps)

	// Bash(0,1) -> collapsed[0]=0, Read(2) -> collapsed[1]=2, Write(3,4) -> collapsed[2]=3
	if len(collapsed) != 3 {
		t.Fatalf("expected 3 collapsed, got %d", len(collapsed))
	}
	expectedRemap := []int{0, 2, 3}
	for i, exp := range expectedRemap {
		if remap[i] != exp {
			t.Errorf("remap[%d]: expected %d, got %d", i, exp, remap[i])
		}
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].ToolName != "Bash" || groups[0].CountA != 2 || groups[0].StartA != 0 {
		t.Errorf("group 0: unexpected %+v", groups[0])
	}
	if groups[1].ToolName != "Write" || groups[1].CountA != 2 || groups[1].StartA != 3 {
		t.Errorf("group 1: unexpected %+v", groups[1])
	}
}

func TestCollapseRetries_EmptySteps(t *testing.T) {
	collapsed, remap, groups := CollapseRetries([]snapshot.Step{})
	if len(collapsed) != 0 {
		t.Errorf("expected 0 collapsed, got %d", len(collapsed))
	}
	if len(remap) != 0 {
		t.Errorf("expected 0 remap, got %d", len(remap))
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestCollapseRetries_NonToolSteps(t *testing.T) {
	steps := []snapshot.Step{
		textStep("hello"),
		toolStep("Read", map[string]interface{}{"path": "/a"}),
		textStep("world"),
	}

	collapsed, remap, _ := CollapseRetries(steps)
	if len(collapsed) != 3 {
		t.Fatalf("expected 3 collapsed, got %d", len(collapsed))
	}
	if remap[0] != 0 || remap[1] != 1 || remap[2] != 2 {
		t.Errorf("remap incorrect: %v", remap)
	}
}

func TestAlign_DivergenceThreshold(t *testing.T) {
	// 5 elements, need >= 1 match for 20% (1/5=0.2, not < 0.2).
	// With exactly 1 match out of 5: ratio=0.2, NOT diverged.
	a := []string{"X", "B", "C", "D", "E"}
	b := []string{"X", "W", "Y", "Z", "Q"}
	r := Align(a, b)
	if r.Diverged {
		t.Error("1/5 match ratio = 0.2, should NOT be diverged (< 0.2 required)")
	}

	// 0 matches out of 5: diverged.
	a2 := []string{"A", "B", "C", "D", "E"}
	b2 := []string{"V", "W", "X", "Y", "Z"}
	r2 := Align(a2, b2)
	if !r2.Diverged {
		t.Error("0/5 match ratio = 0.0, should be diverged")
	}
}
