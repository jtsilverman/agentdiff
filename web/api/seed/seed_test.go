package seed

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jtsilverman/agentdiff/web/api/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	dir := t.TempDir()
	database, err := db.NewDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestSeed_PopulatesScenariosOnEmptyDB(t *testing.T) {
	database := testDB(t)

	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	baselines, err := database.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}

	if len(baselines) != 3 {
		t.Fatalf("expected exactly 3 seeded baselines (variance / regression / novel-strategy), got %d", len(baselines))
	}

	for _, b := range baselines {
		if !strings.HasPrefix(b.Name, "seed-") {
			t.Errorf("baseline %q lacks seed- prefix", b.Name)
		}
		if b.TraceCount < 3 {
			t.Errorf("baseline %q has only %d traces, want >=3 for cluster/graph demo", b.Name, b.TraceCount)
		}
		if strings.TrimSpace(b.Description) == "" {
			t.Errorf("baseline %q has empty Description; every seeded baseline must carry a human-readable description", b.Name)
		}
	}
}

func TestSeed_Idempotent(t *testing.T) {
	database := testDB(t)

	if err := Seed(database); err != nil {
		t.Fatalf("first Seed: %v", err)
	}
	first, err := database.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines after first seed: %v", err)
	}

	if err := Seed(database); err != nil {
		t.Fatalf("second Seed: %v", err)
	}
	second, err := database.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines after second seed: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("Seed is not idempotent: first call produced %d baselines, second produced %d", len(first), len(second))
	}
}

func TestSeed_EachScenarioHasDistinctToolSequence(t *testing.T) {
	database := testDB(t)

	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	baselines, err := database.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}

	// Each of the 3 task-driven scenarios must contain at least 2 distinct tool
	// sequences across its traces, otherwise the path graph collapses to a single
	// linear path and the cluster view shows only one strategy — defeating the
	// "demo of agent variance / regression / novel-strategy" purpose. Unlike the
	// prior 5-scenario design there is no all-stable carve-out; all 3 scenarios
	// are deliberately divergent.
	for _, b := range baselines {
		traces, err := database.GetBaselineTraces(b.ID)
		if err != nil {
			t.Fatalf("GetBaselineTraces %s: %v", b.Name, err)
		}
		if len(traces) == 0 {
			t.Errorf("baseline %q has zero traces", b.Name)
			continue
		}
		sequences := make(map[string]bool)
		for _, td := range traces {
			var seq []string
			for _, step := range td.Steps {
				if step.ToolCall != nil {
					seq = append(seq, step.ToolCall.Name)
				}
			}
			sequences[strings.Join(seq, ",")] = true
		}
		if distinctCount := len(sequences); distinctCount < 2 {
			t.Errorf("scenario %q has only %d distinct tool sequence; needs >=2 to demo divergence", b.Name, distinctCount)
		}
	}
}

// TestSeed_TracesHaveTaskAndOutcomeMetadata enforces that every seeded trace
// carries the two metadata keys the redesigned frontend reads to render the
// per-trace badges: "task" (same per scenario, drives the baseline-detail
// header callout) and "outcome" (per-trace, drives the badge color/label).
// Missing either key silently breaks /baselines/[id] and home-card subtitles.
func TestSeed_TracesHaveTaskAndOutcomeMetadata(t *testing.T) {
	database := testDB(t)

	if err := Seed(database); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	baselines, err := database.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}

	for _, b := range baselines {
		traces, err := database.GetBaselineTraces(b.ID)
		if err != nil {
			t.Fatalf("GetBaselineTraces %s: %v", b.Name, err)
		}
		var sharedTask string
		for i, td := range traces {
			task, okTask := td.Metadata["task"]
			if !okTask || strings.TrimSpace(task) == "" {
				t.Errorf("trace %q under baseline %q is missing non-empty metadata.task", td.Name, b.Name)
			}
			outcome, okOutcome := td.Metadata["outcome"]
			if !okOutcome || strings.TrimSpace(outcome) == "" {
				t.Errorf("trace %q under baseline %q is missing non-empty metadata.outcome", td.Name, b.Name)
			}
			if i == 0 {
				sharedTask = task
			} else if task != sharedTask {
				t.Errorf("baseline %q trace %q has metadata.task = %q; expected all traces in one scenario to share the same task (first was %q)", b.Name, td.Name, task, sharedTask)
			}
		}
	}
}
