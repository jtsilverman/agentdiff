# Three surfaces

Say it in one sentence: one Go module exposes the same diffing engine three ways, as a CLI, as an
HTTP API with a static dashboard, and as a GitHub Action.

Module `github.com/jtsilverman/agentdiff`, Go 1.25.0. Two direct dependencies, `spf13/cobra` and
`yaml.v3`. `mattn/go-sqlite3` means the web API needs CGo and a C compiler; the CLI does not.

## The engine

`internal/` is pure Go with no network calls. Every surface imports it.

| Package | Owns |
|---|---|
| `internal/adapter` | Parsing a trace file. Claude, Claude Code stream-json, OpenAI, OpenAI Agents SDK, LangChain, generic, plus auto-detection |
| `internal/snapshot` | The snapshot model, baselines, and the on-disk store |
| `internal/diff` | Levenshtein alignment on tool sequences, Jaccard on text, the regression verdict |
| `internal/cluster` | DBSCAN over traces to find behavioral strategies, with epsilon selection |
| `internal/graph` | Path-graph aggregation and per-trace overlay |
| `internal/render` | The PNG path graph, drawn with `image/draw` and `basicfont`. No cgo, no browser |
| `internal/report` | Terminal, markdown, and JSON report shapes |
| `internal/bench` | Synthetic trace generation, mutation, and the evaluation harness |
| `internal/config` | `.agentdiff.yaml` thresholds |
| `internal/stats` | Bootstrap resampling and weighting |

## Surface 1: the CLI

`main.go` is three lines; it calls `cmd.Execute()`. `cmd/` is the cobra tree.

| Command | Does |
|---|---|
| `record [trace-file]` | Parse a trace and save it as a snapshot. Reads stdin when the argument is `-` or absent |
| `diff <snapshot-a> <snapshot-b>` | Compare two snapshots. Exits 1 on a regression |
| `report <snapshot-a> <snapshot-b>` | Same comparison, report formatting |
| `baseline record \| compare \| list` | Manage a named baseline of many runs |
| `cluster <baseline-name>` | DBSCAN the baseline into strategies. `--epsilon 0` auto-selects by the elbow method in `internal/cluster/epsilon.go`. `--min-points 0` falls back to the config value, which defaults to `2` |
| `cluster compare <baseline-name> <snapshot>` | Score a snapshot against the clustered strategies. Exits 1 on a new strategy |
| `list` | List snapshots |
| `ci` | The Action's entry point. Writes a markdown report and a PNG |
| `bench` | The empirical validation suite |

Two persistent flags on the root: `--json` and `--max-steps` (default 1000, truncates to the last N
tool calls before alignment).

## Surface 2: the web API and the dashboard

One binary serves both. `web/api/main.go` opens SQLite, seeds it, builds the LLM clients, and starts
Chi. `RegisterRoutes` in `web/api/routes.go` mounts the API under `/api`, then
`r.Handle("/*", http.FileServer(...))` serves the static site at `/`. Chi matches `/api/*` first by
specificity.

| Path | Owns |
|---|---|
| `web/api/db/` | SQLite access, one file per table group |
| `web/api/handlers/` | One file per endpoint, plus an `*_anthropic.go` production client per LLM endpoint |
| `web/api/middleware/` | CORS, request logging, and the per-IP-per-day rate limit |
| `web/api/seed/` | The three canned baselines and their cached LLM responses (`seed-cache.json`) |
| `web/site/` | Plain HTML, CSS, and JS. Four pages: `index`, `demo`, `install`, `about` |

The routes:

```
POST   /api/traces                          GET /api/traces            GET /api/traces/{id}
GET    /api/traces/{id}/transcript          GET /api/traces/{id}/similar
POST   /api/traces/{id}/promote
POST   /api/traces/{id}/counterfactual      rate-limited
POST   /api/traces/{id}/edit-prompt         rate-limited
POST   /api/baselines                       GET /api/baselines
GET    /api/baselines/{id}/cluster          GET /api/baselines/{id}/graph
GET    /api/baselines/{id}/overlay/{traceID}
POST   /api/baselines/{id}/compare
GET    /api/diff/{idA}/{idB}                GET /api/diff/{idA}/{idB}/triage
```

Rate limiting wraps only the two endpoints that generate a fresh agent trace with no cache. Threshold
is 5 per IP per day. `DEMO_KILL_SWITCH=true` short-circuits both to a fixed payload with no LLM call.

Four environment variables shape the runtime: `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL`,
`VOYAGE_API_KEY`, `VOYAGE_MODEL`. Every one is optional. A missing key logs a warning and the
affected endpoints fall back. See `docs/decisions/llm-behind-an-interface.md`.

## Surface 3: the GitHub Action

`action.yml` is a composite action. It installs Go 1.24, `go install`s the CLI, writes a
`.agentdiff.yaml` from its five inputs when the repo has none, and runs
`agentdiff ci --output agentdiff-report.md --png-out agentdiff-graph.png`.

On a pull request it pushes the rendered PNG to an orphan `agentdiff-images` branch in the consuming
repo, then embeds it in the sticky comment through `raw.githubusercontent.com`. That push needs the
consuming workflow to grant `permissions: contents: write`. The step stashes the PNG and the report
in a temp directory before switching branches, so the branch switch cannot lose them.

## Where the root README drifts from the code

The root `README.md` is the product pitch. Four claims in it no longer match the tree:

| README says | The code says |
|---|---|
| The frontend is Next.js 14 + Tailwind + Tremor + React Flow + dagre in `web/frontend/` | That app was deleted in `f323144`. `web/site/` is plain HTML, CSS, and JS |
| Hosted deploy is Railway + Vercel, wired in `web/api/railway.json` and `web/frontend/next.config.js` | `fly.toml` deploys the API to Fly.io as `agentdiff-api`. `web/api/railway.json` still exists; `next.config.js` does not |
| The dashboard seeds five canned scenarios: stable tool order, tool-order variance, prompt regression, novel-tool discovery, noise outlier | `web/api/seed/seed.go` defines three: `api-endpoint-rename`, `auth-migration-md5-to-bcrypt`, `new-endpoint-with-tests`. `seed_test.go` asserts the count is 3 |
| "Hosted demo coming soon" | No URL is published in the README. `specs/archive/agentdiff-redesign-20260527.md:11` names `https://agentdiff.vercel.app` as technically complete, and `fly.toml` deploys the API as `agentdiff-api`. Neither is stated as the live demo |

`web/README.md` carries the same Railway and Vercel deploy section. `.gitignore` still lists
`web/frontend/node_modules/` and friends.
