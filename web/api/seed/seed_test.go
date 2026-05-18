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

	if len(baselines) < 5 || len(baselines) > 8 {
		t.Fatalf("expected 5-8 seeded baselines, got %d", len(baselines))
	}

	for _, b := range baselines {
		if !strings.HasPrefix(b.Name, "seed-") {
			t.Errorf("baseline %q lacks seed- prefix", b.Name)
		}
		if b.TraceCount < 3 {
			t.Errorf("baseline %q has only %d traces, want >=3 for cluster/graph demo", b.Name, b.TraceCount)
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

	// Each scenario must contain at least 2 distinct tool sequences across its traces,
	// otherwise the path graph collapses to a single linear path and the cluster
	// view shows only one strategy — defeating the "demo of agent variance" purpose
	// for scenarios designed to show divergence. Exception: a deliberately stable
	// scenario (all-identical-paths) is allowed to have exactly 1 distinct sequence.
	stableScenarios := map[string]bool{
		"seed-tool-order-stable": true,
	}

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
		distinctCount := len(sequences)
		if stableScenarios[b.Name] {
			if distinctCount != 1 {
				t.Errorf("stable scenario %q expected exactly 1 distinct tool sequence, got %d", b.Name, distinctCount)
			}
		} else if distinctCount < 2 {
			t.Errorf("scenario %q has only %d distinct tool sequence; needs >=2 to demo divergence", b.Name, distinctCount)
		}
	}
}
