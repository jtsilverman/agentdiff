# AgentDiff

**Agent observability for the CLI generation.** Snapshot agent runs, see the path they took, get AI-explained diffs across changes, catch silent regressions before they ship.

AgentDiff started as a Go CLI (think `pytest` but for AI agents) and is now a three-surface platform:

- **Web dashboard** for visualizing agent behavior across runs (run locally — see below; hosted demo coming soon).
- **CLI** (`agentdiff record | diff | bench`) for local-first regression testing.
- **GitHub Action** for catching regressions on every PR with a sticky comment.

All three share the same trace format, baselines, and diffing engine.

## Try the Web Dashboard Locally (60 seconds)

The dashboard ships seeded with five canned scenarios so you can click around without uploading anything.

```bash
git clone https://github.com/jtsilverman/agentdiff
cd agentdiff

# Terminal 1: API + auto-seed
cd web/api && go run . -port 8080

# Terminal 2: frontend
cd web/frontend && npm install && npm run dev
# open http://localhost:3000
```

You'll land on a "Try these examples" row with the five seeded baselines (stable tool order, tool-order variance, prompt regression, novel-tool discovery, noise outlier). Click any baseline to see the path graph; click a trace in the panel to see overlay coloring and branch-confidence percentages.

For the AI features (triage + transcripts), set `ANTHROPIC_API_KEY=sk-...` before starting the API. Without a key, the endpoints still respond (with deterministic fallback) so the rest of the dashboard works.

## Problem

AI agents ship to production but there's no standard way to test whether a prompt, model, or config change caused a regression. Outputs are non-deterministic, so traditional assertion-based testing fails. AgentDiff captures the structural shape of agent behavior (which tools, in what order, branching where) and the textual shape (what the agent said) so you can compare runs the way you compare code: see the diff, explain the diff, decide if it's a regression.

## Web Dashboard Features

- **Path graph** — interactive directed graph of tool-call sequences across all traces in a baseline, with branch-point confidence percentages and overlay coloring when you click a specific trace.
- **Cost/latency heatmap** — toggle the path graph between overlay, cost, and latency modes. `GET /api/baselines/:id/graph` aggregates per-tool cost (tokens) and latency (ms) across baseline traces; nodes recolor on a hot-to-cold scale and surface the value inline. Old traces without usage metadata render gray, not error. Claude Code and OpenAI adapters parse usage out of stream-json / ChatCompletion responses.
- **Replay scrubber** — drag a slider across any trace's step timeline; the side panel shows the role, full context window, tool call args, and tool result at each position. No backend changes; the frontend consumes the existing `GET /api/traces/:id` step list. Empty traces render an empty-state message instead of crashing.
- **Counterfactual replay** — pick any step on a trace, edit its input, click "What if?". `POST /api/traces/:id/counterfactual` re-runs the agent from that step with the modified input and returns the new trace plus an `{original_path, new_path, divergence_step}` comparison. The frontend renders both paths overlaid on a fork-graph: shared prefix, then original vs counterfactual branches with the divergence step highlighted.
- **Inline prompt editor** — sister feature to counterfactual replay: pick a step, rewrite its prompt, click "Re-run from here". `POST /api/traces/:id/edit-prompt` re-runs the agent with the rewritten prompt and returns the same fork-graph shape (shared prefix → original vs edited branches with the divergence step). Persists the link in `prompt_edit_runs` so you can trace prompt-change consequences after the fact.
- **Audit-to-test (Promote to baseline)** — `POST /api/traces/:id/promote` turns any one-off trace into a brand-new single-trace baseline in one click, so the next uploaded run gets diffed against it. The trace detail page surfaces a "Promote to baseline" button that defaults the name to `promoted-<trace-name>` and pushes you to the new baseline on success.
- **AI triage** — `GET /api/diff/:idA/:idB/triage` sends two trace step-sequences to Claude with a structured prompt and returns `{summary, classification, likely_cause}` where classification is one of `regression | variance | additive`. Cached on a hash of the canonical step content so repeat asks return in <100ms with zero LLM cost.
- **Annotated transcripts** — `GET /api/traces/:id/transcript` returns a one-paragraph plain-English summary of what the agent did, plus a short list of key decisions it made. Same caching shape as triage.
- **Similar traces (embeddings search)** — `GET /api/traces/:id/similar` returns the top-5 most semantically similar traces by cosine distance over Voyage AI embeddings (`voyage-3-lite`). Embeddings are generated in a background goroutine on trace insert (non-blocking, 30s timeout) and cached in a `trace_embeddings` table. Traces without embeddings are skipped naturally; an unembedded source returns an empty match list rather than 500.
- **Pre-seeded examples** — five canned scenarios (stable tool order, tool-order variance, prompt regression, novel-tool discovery, noisy outlier) so a stranger can click around and understand the product in 30 seconds.
- **Side-by-side text diff** — secondary tab on `/diff/:idA/:idB` for when you want the raw aligned step-by-step view.
- **Drag-drop trace upload** — drop a `.jsonl` (Claude Code or OpenAI format) and it parses, persists, and is immediately diffable.

The backend is a Go + Chi API talking to SQLite (WAL mode, foreign keys on). The frontend is Next.js 14 (App Router) + Tailwind + Tremor + React Flow + dagre. Hosted deploy (Railway + Vercel) is wired in `web/api/railway.json` + `web/frontend/next.config.js`; a hosted demo URL will land here once provisioning completes.

## CLI Quick Start

```bash
go install github.com/jtsilverman/agentdiff@latest

# Record a baseline
agentdiff record --name baseline trace.jsonl

# Make changes, record again
agentdiff record --name after-change trace.jsonl

# Diff
agentdiff diff baseline after-change
```

## Supported Trace Formats

- **Claude Code** — JSONL conversation traces and `stream-json` format
- **OpenAI** — Chat completions messages array (direct or API response wrapper)
- Auto-detection (default)

Adding a new agent framework is roughly ~100 lines via the adapter interface.

## How the Diff Works

AgentDiff compares two traces on two independent dimensions:

1. **Structural (tool calls)** — Levenshtein edit distance on the ordered sequence of tool names. Catches: different tools used, different order, different arguments.
2. **Textual (output content)** — Jaccard similarity on bigram token sets. Robust to rephrasing, catches topical drift.

Configurable thresholds determine when a difference is a regression versus expected variation.

## Configuration

Create `.agentdiff.yaml`:
```yaml
thresholds:
  tool_score: 0.3    # tool diff above this = regression
  text_score: 0.5    # text diff above this = regression
  step_delta: 5      # step count change above this = regression
```

## GitHub Action

Add to your workflow to catch agent regressions on every PR:

```yaml
- uses: jtsilverman/agentdiff@v0
  with:
    baseline_path: ".agentdiff/baselines/main.json.gz"  # optional
    threshold_tool: "0.3"   # tool diff threshold (0.0-1.0)
    threshold_text: "0.5"   # text diff threshold (0.0-1.0)
    threshold_steps: "5"    # step count delta threshold
    fail_on_style_drift: "false"  # fail on text-only regressions
```

On pull requests, posts a sticky comment with the regression report and a rendered PNG of the baseline path graph with the divergence step annotated (red-outlined node + arrow). The PNG is rendered server-side in pure Go (no cgo, no Chrome), committed to an orphan `agentdiff-images` branch in the consumer's repo, and embedded in the comment via raw.githubusercontent. Exits 1 if regressions are detected.

## CI Usage (Manual)

```yaml
- name: Check for agent regressions
  run: |
    agentdiff record --name baseline golden/trace.jsonl
    agentdiff record --name current current/trace.jsonl
    agentdiff diff baseline current  # exits 1 on regression
```

## Bench Suite

Empirical validation of AgentDiff's regression detection using synthetic traces and mutation testing.

```bash
# Run bench with table output
agentdiff bench

# JSON output for CI/plotting
agentdiff bench --json

# Custom seed, save to file
agentdiff bench --seed 123 --output results.json
```

Evaluates four dimensions:
- **Regression detection** — precision, recall, F1 on 90 labeled trace pairs (60 mutated + 30 natural variance)
- **Threshold calibration** — ROC curves + AUC across tool, text, and step dimensions; identifies optimal operating points
- **Clustering quality** — Adjusted Rand Index on DBSCAN clustering of strategy-labeled traces
- **Cross-validation** — 5-fold stratified validation with mean/std F1 and averaged optimal thresholds

Deterministic (same seed = same output), runs in under 2 seconds.

## Roadmap

- Hosted demo at a public URL (Railway API + Vercel frontend, provisioning the deploy configs already in this repo).
- Multi-model comparison (run the same prompt against Claude / GPT / Gemini, overlay all three on the path graph).
- Authentication and multi-tenancy.
- OpenTelemetry ingestion and live WebSocket streaming of in-progress runs.

PRs welcome.

## Architecture Notes

- **CLI core** (`internal/`): trace parsing, snapshot model, Levenshtein + Jaccard diff, DBSCAN strategy clustering, bench harness. Pure Go, no network.
- **Web API** (`web/api/`): Chi router, SQLite storage (WAL + FK on), handlers reuse the CLI's diff/cluster packages directly. LLM-backed endpoints (triage, transcript, counterfactual, edit-prompt, embeddings) use a DI seam (`Triager` / `Summarizer` / `Counterfactualer` / `EditPrompter` / `Embedder` interfaces) so unit tests inject fakes; production binds to an `Anthropic*` implementation (or `Voyage*` for embeddings) that handles cached system prompts, tolerant JSON parsing, and deterministic fallback on any failure. New tables in the schema: `triage_cache`, `transcripts`, `counterfactual_runs`, `prompt_edit_runs`, `trace_embeddings`.
- **Frontend** (`web/frontend/`): Next.js 14, Tailwind, Tremor for primitives, React Flow + dagre for the path graph, Vitest + RTL for tests. Zero runtime LLM calls from the browser.

## License

MIT.
