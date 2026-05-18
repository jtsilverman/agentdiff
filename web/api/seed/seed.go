// Package seed populates the database with canned demo scenarios on first boot.
// Scenarios are named with a "seed-" prefix; the prefix doubles as the
// idempotency gate and as the frontend's filter for the landing-page demo row.
package seed

import (
	"fmt"
	"strings"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// scenario is one demo baseline: a human-readable name + N traces, each with
// its own tool-call sequence. Steps are built inline (not from JSONL fixtures)
// because the recipes are short and read more clearly as Go literals.
type scenario struct {
	name   string
	traces []traceSpec
}

type traceSpec struct {
	name      string
	toolNames []string
}

func scenarios() []scenario {
	return []scenario{
		{
			name: "seed-tool-order-stable",
			traces: []traceSpec{
				{name: "run-1", toolNames: []string{"read_file", "write_file"}},
				{name: "run-2", toolNames: []string{"read_file", "write_file"}},
				{name: "run-3", toolNames: []string{"read_file", "write_file"}},
				{name: "run-4", toolNames: []string{"read_file", "write_file"}},
				{name: "run-5", toolNames: []string{"read_file", "write_file"}},
			},
		},
		{
			name: "seed-tool-order-variance",
			traces: []traceSpec{
				{name: "run-1", toolNames: []string{"read_file", "write_file"}},
				{name: "run-2", toolNames: []string{"read_file", "write_file"}},
				{name: "run-3", toolNames: []string{"read_file", "write_file"}},
				{name: "run-4", toolNames: []string{"search", "read_file", "write_file"}},
				{name: "run-5", toolNames: []string{"search", "read_file", "write_file"}},
			},
		},
		{
			name: "seed-prompt-regression",
			traces: []traceSpec{
				{name: "before-1", toolNames: []string{"read_file", "search", "write_file"}},
				{name: "before-2", toolNames: []string{"read_file", "search", "write_file"}},
				{name: "before-3", toolNames: []string{"read_file", "search", "write_file"}},
				{name: "before-4", toolNames: []string{"read_file", "search", "write_file"}},
				{name: "after-prompt-change", toolNames: []string{"read_file", "write_file"}},
			},
		},
		{
			name: "seed-novel-tool-discovery",
			traces: []traceSpec{
				{name: "run-1", toolNames: []string{"read_file", "write_file"}},
				{name: "run-2", toolNames: []string{"read_file", "write_file"}},
				{name: "run-3", toolNames: []string{"read_file", "write_file"}},
				{name: "run-4", toolNames: []string{"read_file", "grep", "write_file"}},
				{name: "run-5", toolNames: []string{"read_file", "grep", "write_file"}},
			},
		},
		{
			name: "seed-noise-and-strategies",
			traces: []traceSpec{
				{name: "fetch-parse-store-1", toolNames: []string{"fetch", "parse", "store"}},
				{name: "fetch-parse-store-2", toolNames: []string{"fetch", "parse", "store"}},
				{name: "fetch-parse-store-3", toolNames: []string{"fetch", "parse", "store"}},
				{name: "fetch-store-1", toolNames: []string{"fetch", "store"}},
				{name: "fetch-store-2", toolNames: []string{"fetch", "store"}},
				{name: "outlier-bash-fetch", toolNames: []string{"bash", "fetch"}},
			},
		},
	}
}

// Seed inserts canned demo scenarios into the database. Safe to call repeatedly:
// scenarios are skipped per-name if a baseline with that name already exists.
// Existing tests against an empty DB stay green because main.go is the only
// caller; web/api/db tests never invoke Seed.
func Seed(database *db.DB) error {
	existing, err := database.ListBaselines()
	if err != nil {
		return fmt.Errorf("list baselines: %w", err)
	}
	have := make(map[string]bool, len(existing))
	for _, b := range existing {
		have[b.Name] = true
	}

	for _, s := range scenarios() {
		if have[s.name] {
			continue
		}
		traceIDs := make([]string, 0, len(s.traces))
		for _, ts := range s.traces {
			trace, err := database.CreateTrace(
				fmt.Sprintf("%s/%s", s.name, ts.name),
				"seed",
				map[string]string{"scenario": s.name},
			)
			if err != nil {
				return fmt.Errorf("create trace %s/%s: %w", s.name, ts.name, err)
			}
			if err := database.InsertSnapshots(trace.ID, buildSteps(ts.toolNames)); err != nil {
				return fmt.Errorf("insert snapshots %s/%s: %w", s.name, ts.name, err)
			}
			traceIDs = append(traceIDs, trace.ID)
		}
		if _, err := database.CreateBaseline(s.name, traceIDs); err != nil {
			return fmt.Errorf("create baseline %s: %w", s.name, err)
		}
	}
	return nil
}

// buildSteps produces a user-prompt + alternating tool-call / tool-result chain
// for the given tool names. Each tool is invoked once with a synthetic
// argument and returns a synthetic success output, enough for the cluster and
// path-graph algorithms to read a non-trivial sequence.
func buildSteps(toolNames []string) []snapshot.Step {
	steps := []snapshot.Step{
		{Role: "user", Content: "Demo prompt for " + strings.Join(toolNames, ", ")},
	}
	for _, name := range toolNames {
		steps = append(steps,
			snapshot.Step{
				Role: "assistant",
				ToolCall: &snapshot.ToolCall{
					Name: name,
					Args: map[string]interface{}{"target": "demo"},
				},
			},
			snapshot.Step{
				Role: "tool_result",
				ToolResult: &snapshot.ToolResult{
					Name:   name,
					Output: name + " ok",
				},
			},
		)
	}
	return steps
}
