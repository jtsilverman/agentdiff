# Go and test conventions

One answer per question. A reviewer can reject a change from this file alone.

## Layout

`internal/` is the engine and must stay free of network calls. `cmd/` is the cobra tree, one file per
command. `web/api/` is the only place that talks to SQLite or to an LLM provider. `web/site/` is
plain HTML, CSS, and JS with no build step and no framework.

A handler never reimplements a diff. `web/api/handlers/` imports the same `internal/diff`,
`internal/cluster`, and `internal/graph` packages the CLI uses.

## Where a test goes

Next to the code, same package, `<name>_test.go`. `internal/diff/tools.go` is tested by
`internal/diff/tools_test.go`. There is no separate test tree.

Cross-command tests live in `cmd/integration_test.go`. Shared handler helpers live in
`web/api/handlers/handlers_test.go`.

Standard library `testing` only. No assertion framework is in `go.mod`.

`testdata/` holds one trace fixture per supported format: `claude_trace.jsonl`,
`claudecode_stream.jsonl`, `claudecode_with_usage.jsonl`, `openai_trace.json`,
`openai_with_usage.json`, `agents_sdk_trace.json`, `langchain_callbacks.jsonl`, and
`generic_trace.jsonl`. A new adapter adds a fixture here.

## An LLM endpoint takes an interface

Every endpoint that calls a model declares its dependency as an interface in
`web/api/handlers/`, and the handler constructor takes it as a parameter:

| Interface | Endpoint |
|---|---|
| `Triager` | `GET /api/diff/{idA}/{idB}/triage` |
| `Summarizer` | `GET /api/traces/{id}/transcript` |
| `Counterfactualer` | `POST /api/traces/{id}/counterfactual` |
| `EditPrompter` | `POST /api/traces/{id}/edit-prompt` |
| `Embedder` | The background embedding on `POST /api/traces` |

The production implementation lives in a sibling `*_anthropic.go` file, or `embedder_voyage.go`.
`web/api/main.go` is the only place that binds one. A test injects a fake and never touches the
network. Do not call an HTTP client directly from a handler.

`Embedder` has nil-receiver semantics on purpose. A nil `Embedder` means embedding is off:
`POST /api/traces` still succeeds, no `trace_embeddings` row is written, and `/similar` returns an
empty match list. `main.go` assigns through a typed local to avoid Go's typed-nil-interface trap;
read the comment there before changing that block.

## Adding a trace format

Implement the adapter interface in `internal/adapter/<name>.go`, register it in `detect.go`, add a
fixture to `testdata/`, and add `internal/adapter/<name>_test.go`. The README puts the cost at
roughly 100 lines.

## Caching

Every LLM result caches on a hash of the canonical step content, so a repeat ask costs nothing.
The tables are `triage_cache`, `transcripts`, `counterfactual_runs`, `prompt_edit_runs`, and
`trace_embeddings`. `web/api/seed/seed-cache.json` pre-populates those caches for the three seeded
baselines, which is why the demo works with no API key.

## Determinism

`agentdiff bench` must produce identical output for identical seeds. It runs in under two seconds and
CI runs it on every push. A change that makes the bench non-deterministic breaks the build.

## Git

Conventional commits with a scope: `feat(traces):`, `fix(cluster):`, `chore(spec):`,
`feat(rate-limit):`. The scope is the surface or the feature, not the file. Branch per task, one
commit per logical unit, Jake merges the PR. Never commit to main. Stage with `git add <file>`.

## What never lands in the repo

`.gitignore` covers the built `agentdiff` binary, `.agentdiff/`, any `*.db`, and `.worktrees/`. It
still carries a `web/frontend/` block for the deleted Next.js app.
