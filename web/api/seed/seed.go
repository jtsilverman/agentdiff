// Package seed populates the database with canned demo scenarios on first boot.
// Baselines carry short engineering-style names (api-endpoint-rename /
// auth-migration-md5-to-bcrypt / new-endpoint-with-tests) so they read as real
// project state rather than fixture data; the names also drive the frontend's
// matchHint substring lookup for the landing-page demo row.
//
// Each scenario is task-driven: the human-readable task statement appears as
// the user-prompt in every trace and is also surfaced via trace metadata
// (`task` key, same value across all traces in the scenario). Each individual
// trace carries an `outcome` metadata key (`succeeded` / `regressed` /
// `variance` / `additive`) the frontend renders as a per-trace badge, plus a
// `key_decision` metadata key (one short sentence describing the trace's
// load-bearing choice) the redesigned trace-detail page renders as a callout.
package seed

import (
	"fmt"

	"github.com/jtsilverman/agentdiff/internal/snapshot"
	"github.com/jtsilverman/agentdiff/web/api/db"
)

// scenario is one demo baseline: a human-readable name + description + the
// shared task statement + N traces, each with its own tool-call sequence and
// per-trace outcome label. Steps are built inline (not from JSONL fixtures)
// because the recipes are short and read more clearly as Go literals.
type scenario struct {
	name        string
	description string
	task        string
	traces      []traceSpec
}

// traceSpec is one run within a scenario. toolCalls are paired with realistic
// args + short result snippets so the rendered trace reads as a plausible
// agent invocation rather than the prior "Demo prompt for read_file" filler.
// keyDecision is one short sentence describing this trace's load-bearing
// choice; it lands in metadata and powers the trace-detail page's
// key-decision callout.
type traceSpec struct {
	name        string
	outcome     string
	keyDecision string
	toolCalls   []toolCall
}

type toolCall struct {
	name   string
	args   map[string]interface{}
	output string
}

func scenarios() []scenario {
	return []scenario{
		// Variance scenario: same task, agent picks one of three valid strategies
		// (grep-first, search-first, assume-known-path). All five runs succeed;
		// the demo value is the cluster + path-graph showing three coexisting
		// strategies for the same prompt.
		{
			name:        "api-endpoint-rename",
			description: "Rename the /users endpoint to /customers across the codebase. Five agents pick three different strategies — all valid, none regressed.",
			task:        "Rename the /users endpoint to /customers across the codebase, updating all call sites.",
			traces: []traceSpec{
				{
					name:        "run-1",
					outcome:     "variance",
					keyDecision: "Ran grep before touching any file, anchoring the edit set on the literal /users string.",
					toolCalls: []toolCall{
						{name: "grep", args: map[string]interface{}{"pattern": "/users", "path": "."}, output: "src/routes/api.go:23\nsrc/handlers/auth.go:41\nweb/client/api.ts:12"},
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/api.go"}, output: "router.GET(\"/users\", ListUsers)\nrouter.POST(\"/users\", CreateUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/api.go", "edit": "/users -> /customers"}, output: "wrote 2 changes"},
						{name: "write_file", args: map[string]interface{}{"path": "web/client/api.ts", "edit": "/users -> /customers"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "run-2",
					outcome:     "variance",
					keyDecision: "Repeated the grep-first sequence verbatim — within-strategy reproducibility check.",
					toolCalls: []toolCall{
						{name: "grep", args: map[string]interface{}{"pattern": "/users", "path": "."}, output: "src/routes/api.go:23\nsrc/handlers/auth.go:41\nweb/client/api.ts:12"},
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/api.go"}, output: "router.GET(\"/users\", ListUsers)\nrouter.POST(\"/users\", CreateUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/api.go", "edit": "/users -> /customers"}, output: "wrote 2 changes"},
						{name: "write_file", args: map[string]interface{}{"path": "web/client/api.ts", "edit": "/users -> /customers"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "run-3",
					outcome:     "variance",
					keyDecision: "Chose the higher-level search tool over grep, treating the task as finding route definitions rather than a literal string sweep.",
					toolCalls: []toolCall{
						{name: "search", args: map[string]interface{}{"query": "endpoint definitions"}, output: "3 files match: src/routes/api.go, src/handlers/auth.go, web/client/api.ts"},
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/api.go"}, output: "router.GET(\"/users\", ListUsers)\nrouter.POST(\"/users\", CreateUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/api.go", "edit": "/users -> /customers"}, output: "wrote 2 changes"},
					},
				},
				{
					name:        "run-4",
					outcome:     "variance",
					keyDecision: "Repeated the search-first sequence — same exploration depth as run-3.",
					toolCalls: []toolCall{
						{name: "search", args: map[string]interface{}{"query": "endpoint definitions"}, output: "3 files match: src/routes/api.go, src/handlers/auth.go, web/client/api.ts"},
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/api.go"}, output: "router.GET(\"/users\", ListUsers)\nrouter.POST(\"/users\", CreateUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/api.go", "edit": "/users -> /customers"}, output: "wrote 2 changes"},
					},
				},
				{
					name:        "run-5",
					outcome:     "variance",
					keyDecision: "Skipped all discovery — assumed the route file and client file locations and edited both directly.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/api.go"}, output: "router.GET(\"/users\", ListUsers)\nrouter.POST(\"/users\", CreateUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/api.go", "edit": "/users -> /customers"}, output: "wrote 2 changes"},
						{name: "write_file", args: map[string]interface{}{"path": "web/client/api.ts", "edit": "/users -> /customers"}, output: "wrote 1 change"},
					},
				},
			},
		},

		// Regression scenario: a "before" path used search to enumerate call
		// sites and update them comprehensively; the "after" path skips the
		// search and only updates the file it directly knows about, silently
		// missing other call sites. Demonstrates regression detection via
		// before/after divergence in tool sequence.
		{
			name:        "auth-migration-md5-to-bcrypt",
			description: "Refactor user-auth from MD5 to bcrypt. Four runs handle the migration carefully; one shortcuts after a prompt change and misses call sites.",
			task:        "Refactor the user-auth module from MD5 password hashing to bcrypt, updating every call site.",
			traces: []traceSpec{
				{
					name:        "before-1",
					outcome:     "succeeded",
					keyDecision: "Searched for every MD5 call site before editing, surfacing verify.go and migration/users.go alongside the pointed-at module.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/passwords.go"}, output: "func Hash(pw string) string { return md5.Sum([]byte(pw)) }"},
						{name: "search", args: map[string]interface{}{"query": "md5.Sum|MD5", "path": "src/"}, output: "src/auth/passwords.go:8\nsrc/auth/verify.go:14\nsrc/migration/users.go:22"},
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/verify.go"}, output: "expected := md5.Sum([]byte(input))"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/passwords.go", "edit": "md5.Sum -> bcrypt.GenerateFromPassword"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/verify.go", "edit": "md5.Sum -> bcrypt.CompareHashAndPassword"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "before-2",
					outcome:     "succeeded",
					keyDecision: "Reproduced the comprehensive discovery-first migration — same shape as before-1.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/passwords.go"}, output: "func Hash(pw string) string { return md5.Sum([]byte(pw)) }"},
						{name: "search", args: map[string]interface{}{"query": "md5.Sum|MD5", "path": "src/"}, output: "src/auth/passwords.go:8\nsrc/auth/verify.go:14\nsrc/migration/users.go:22"},
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/verify.go"}, output: "expected := md5.Sum([]byte(input))"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/passwords.go", "edit": "md5.Sum -> bcrypt.GenerateFromPassword"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/verify.go", "edit": "md5.Sum -> bcrypt.CompareHashAndPassword"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "before-3",
					outcome:     "succeeded",
					keyDecision: "Reproduced the discovery-first migration a third time — the pre-change prompt elicits this shape reliably.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/passwords.go"}, output: "func Hash(pw string) string { return md5.Sum([]byte(pw)) }"},
						{name: "search", args: map[string]interface{}{"query": "md5.Sum|MD5", "path": "src/"}, output: "src/auth/passwords.go:8\nsrc/auth/verify.go:14\nsrc/migration/users.go:22"},
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/verify.go"}, output: "expected := md5.Sum([]byte(input))"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/passwords.go", "edit": "md5.Sum -> bcrypt.GenerateFromPassword"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/verify.go", "edit": "md5.Sum -> bcrypt.CompareHashAndPassword"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "before-4",
					outcome:     "succeeded",
					keyDecision: "Fourth pre-change run with the same comprehensive shape — establishes the baseline against which the after-prompt regression is measured.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/passwords.go"}, output: "func Hash(pw string) string { return md5.Sum([]byte(pw)) }"},
						{name: "search", args: map[string]interface{}{"query": "md5.Sum|MD5", "path": "src/"}, output: "src/auth/passwords.go:8\nsrc/auth/verify.go:14\nsrc/migration/users.go:22"},
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/verify.go"}, output: "expected := md5.Sum([]byte(input))"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/passwords.go", "edit": "md5.Sum -> bcrypt.GenerateFromPassword"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/verify.go", "edit": "md5.Sum -> bcrypt.CompareHashAndPassword"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "after-prompt-change",
					outcome:     "regressed",
					keyDecision: "Skipped the call-site search — only the file the prompt named was edited, silently leaving verify.go and migration/users.go on MD5.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/auth/passwords.go"}, output: "func Hash(pw string) string { return md5.Sum([]byte(pw)) }"},
						{name: "write_file", args: map[string]interface{}{"path": "src/auth/passwords.go", "edit": "md5.Sum -> bcrypt.GenerateFromPassword"}, output: "wrote 1 change"},
					},
				},
			},
		},

		// Novel-strategy scenario: 3 baseline runs add the endpoint directly;
		// 2 later runs adopt a test-first approach, writing the test file
		// alongside the route. The new pattern is additive — the original
		// route-only path still works — so outcome is `additive`, not
		// `regressed`. Natural counterfactual target: "what would run-1 look
		// like if it also wrote a test?"
		{
			name:        "new-endpoint-with-tests",
			description: "Add a new GET /users/:id/preferences endpoint. Three runs add the route directly; two later runs also write a test file — additive behavior, not a regression.",
			task:        "Add a new GET /users/:id/preferences endpoint to the user service.",
			traces: []traceSpec{
				{
					name:        "run-1",
					outcome:     "variance",
					keyDecision: "Added the new route in a single write to the routes file — no companion test.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/users.go"}, output: "router.GET(\"/users/:id\", GetUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users.go", "edit": "+ router.GET(\"/users/:id/preferences\", GetPreferences)"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "run-2",
					outcome:     "variance",
					keyDecision: "Repeated the route-only addition — within-cohort reproducibility check.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/users.go"}, output: "router.GET(\"/users/:id\", GetUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users.go", "edit": "+ router.GET(\"/users/:id/preferences\", GetPreferences)"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "run-3",
					outcome:     "variance",
					keyDecision: "Third route-only run — establishes the pre-adoption baseline against which the test-alongside runs are compared.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/users.go"}, output: "router.GET(\"/users/:id\", GetUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users.go", "edit": "+ router.GET(\"/users/:id/preferences\", GetPreferences)"}, output: "wrote 1 change"},
					},
				},
				{
					name:        "run-4",
					outcome:     "additive",
					keyDecision: "Added the route and created a companion test file in the same run — layered a test-first habit on top of the basic route addition.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/users.go"}, output: "router.GET(\"/users/:id\", GetUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users.go", "edit": "+ router.GET(\"/users/:id/preferences\", GetPreferences)"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users_test.go", "edit": "+ TestGetPreferences"}, output: "created 1 file"},
					},
				},
				{
					name:        "run-5",
					outcome:     "additive",
					keyDecision: "Repeated the route-plus-test pattern — the new shape is stable across multiple invocations once adopted.",
					toolCalls: []toolCall{
						{name: "read_file", args: map[string]interface{}{"path": "src/routes/users.go"}, output: "router.GET(\"/users/:id\", GetUser)"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users.go", "edit": "+ router.GET(\"/users/:id/preferences\", GetPreferences)"}, output: "wrote 1 change"},
						{name: "write_file", args: map[string]interface{}{"path": "src/routes/users_test.go", "edit": "+ TestGetPreferences"}, output: "created 1 file"},
					},
				},
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
				map[string]string{
					"scenario":     s.name,
					"task":         s.task,
					"outcome":      ts.outcome,
					"key_decision": ts.keyDecision,
				},
			)
			if err != nil {
				return fmt.Errorf("create trace %s/%s: %w", s.name, ts.name, err)
			}
			if err := database.InsertSnapshots(trace.ID, buildSteps(s.task, ts.toolCalls)); err != nil {
				return fmt.Errorf("insert snapshots %s/%s: %w", s.name, ts.name, err)
			}
			traceIDs = append(traceIDs, trace.ID)
		}
		if _, err := database.CreateBaseline(s.name, s.description, traceIDs); err != nil {
			return fmt.Errorf("create baseline %s: %w", s.name, err)
		}
	}
	return nil
}

// buildSteps produces a user-prompt + alternating tool-call / tool-result chain.
// The user prompt is the scenario's task statement (so trace transcripts read
// as plausible agent invocations). Each tool-call carries its real args + a
// short canned output snippet — long enough to read as a real result, short
// enough not to drown the path graph.
func buildSteps(task string, calls []toolCall) []snapshot.Step {
	steps := []snapshot.Step{
		{Role: "user", Content: task},
	}
	for _, c := range calls {
		steps = append(steps,
			snapshot.Step{
				Role: "assistant",
				ToolCall: &snapshot.ToolCall{
					Name: c.name,
					Args: c.args,
				},
			},
			snapshot.Step{
				Role: "tool_result",
				ToolResult: &snapshot.ToolResult{
					Name:   c.name,
					Output: c.output,
				},
			},
		)
	}
	return steps
}
