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
- **AI triage** — `GET /api/diff/:idA/:idB/triage` sends two trace step-sequences to Claude with a structured prompt and returns `{summary, classification, likely_cause}` where classification is one of `regression | variance | additive`. Cached on a hash of the canonical step content so repeat asks return in <100ms with zero LLM cost.
- **Annotated transcripts** — `GET /api/traces/:id/transcript` returns a one-paragraph plain-English summary of what the agent did, plus a short list of key decisions it made. Same caching shape as triage.
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

On pull requests, posts a sticky comment with the regression report. Exits 1 if regressions are detected.

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

- Counterfactual replay (modify a mid-trace step, re-run, compare paths).
- Inline prompt editing (rewrite any step's prompt, re-run from there).
- Cost/latency heatmap on the path graph.
- Replay scrubber with full-context-window display per step.
- Embeddings-based similar-trace search.
- GitHub Action PR comments with rendered path-graph PNG.

These are tracked in the active spec; PRs welcome.

## Architecture Notes

- **CLI core** (`internal/`): trace parsing, snapshot model, Levenshtein + Jaccard diff, DBSCAN strategy clustering, bench harness. Pure Go, no network.
- **Web API** (`web/api/`): Chi router, SQLite storage (WAL + FK on), handlers reuse the CLI's diff/cluster packages directly. LLM-backed endpoints (triage, transcript) use a DI seam (`Triager` / `Summarizer` interfaces) so unit tests inject fakes; production binds to an `Anthropic*` implementation that handles cached system prompts, tolerant JSON parsing, and deterministic fallback on any failure.
- **Frontend** (`web/frontend/`): Next.js 14, Tailwind, Tremor for primitives, React Flow + dagre for the path graph, Vitest + RTL for tests. Zero runtime LLM calls from the browser.

## License

MIT.
