# Build, test, deploy

## Prerequisites

- Go 1.25 or newer. `go.mod` declares `go 1.25.0`.
- A C compiler with CGo enabled. `mattn/go-sqlite3` needs it. The CLI alone builds without it.

## Run the tests

The whole suite, from the repo root:
```
go test ./...
```

One package:
```
go test ./internal/diff/
```

Handler tests only:
```
cd web && make test
```
That runs `go test ./handlers/ -v` inside `web/api`.

## Run the bench suite

```
go run . bench                          # table output
go run . bench --json                   # JSON for CI or plotting
go run . bench --seed 123 --output results.json
```

Same seed produces the same output. The run takes under two seconds. It reports regression-detection
precision, recall, and F1 over 90 labeled pairs; ROC curves and AUC for threshold calibration;
Adjusted Rand Index for clustering; and 5-fold stratified cross-validation.

## Build the CLI

```
go build -o agentdiff .
```

The built binary is gitignored.

## Run the dashboard locally

One binary serves the API and the static site.

```
cd web && make dev
```

Or directly:
```
cd web/api && go run . -port 8080 -site ../site
```

Open `http://localhost:8080`. The database seeds itself with three canned baselines, one per diff
classification. Nothing needs uploading.

| Baseline | Shows |
|---|---|
| `api-endpoint-rename` | Variance. Five agents rename `/users` to `/customers` by three different strategies, all valid |
| `auth-migration-md5-to-bcrypt` | Regression. Four runs migrate carefully; one shortcuts after a prompt change and misses call sites |
| `new-endpoint-with-tests` | Additive. Three runs add the route; two later runs also write a test file |

`web/api/seed/seed_test.go` pins the count at three. The root `README.md` claims five; it is stale.

Flags on the API binary:

| Flag | Default | Does |
|---|---|---|
| `-port` | `8080` | HTTP port |
| `-db` | `agentdiff.db` | SQLite path |
| `-site` | `../site` | Static site directory served at `/` |

Environment variables, all optional:

| Variable | Effect when unset |
|---|---|
| `ANTHROPIC_API_KEY` | Triage, transcripts, counterfactuals, and prompt-edit replays return deterministic fallbacks. The server logs a warning and starts |
| `ANTHROPIC_MODEL` | The client's own default model |
| `VOYAGE_API_KEY` | No embeddings are generated and `/similar` returns empty matches |
| `VOYAGE_MODEL` | The client's own default model |
| `DEMO_KILL_SWITCH` | Set to `true` to short-circuit counterfactual and edit-prompt to a fixed payload with no LLM call |

The three seeded baselines carry pre-cached LLM responses in `web/api/seed/seed-cache.json`, so the
demo reads the same with or without a key.

## Use the CLI

```
go install github.com/jtsilverman/agentdiff@latest

agentdiff record --name baseline trace.jsonl
agentdiff record --name after-change trace.jsonl
agentdiff diff baseline after-change        # exits 1 on a regression
```

`record` reads stdin when the file argument is `-` or absent. `--adapter` defaults to `auto`.

Thresholds come from `.agentdiff.yaml` in the working directory:
```yaml
thresholds:
  tool_score: 0.3
  text_score: 0.5
  step_delta: 5
```

## CI

`.github/workflows/ci.yml` runs on every push to main and every pull request against main. Three
steps on Go 1.24: `go test ./...`, `go run . bench`, `go build -o agentdiff .`.

The workflow pins Go 1.24 while `go.mod` declares 1.25.0 and the Dockerfile builds on
`golang:1.25-bookworm`. `action.yml` also pins 1.24.

## Deploy the API

`fly.toml` is the live config. App `agentdiff-api`, region `iad`, internal port 8080, HTTPS forced,
machines auto-stop and auto-start with `min_machines_running = 0`, a `agentdiff_data` volume mounted
at `/data`, and a `shared-cpu-1x` 512mb VM. The image builds from `web/api/Dockerfile`.

```
fly deploy
```

The Dockerfile is two stages: build on `golang:1.25-bookworm` with `CGO_ENABLED=1`, then run on
`debian:bookworm-slim` with `ca-certificates`. Its `CMD` is
`agentdiff-api --port 8080 --db /data/agentdiff.db`.

`web/api/railway.json` is a second deploy config for Railway, using the same Dockerfile with
`--port $PORT`. `web/README.md` documents the Railway and Vercel path. Fly is what `fly.toml`
describes; treat the Railway config as the older path until one of the two is deleted.

## Use the GitHub Action

```yaml
permissions:
  contents: write        # the Action pushes the PNG to an orphan branch

steps:
  - uses: jtsilverman/agentdiff@v0
    with:
      baseline_path: ".agentdiff/baselines/main.json.gz"
      threshold_tool: "0.3"
      threshold_text: "0.5"
      threshold_steps: "5"
      fail_on_style_drift: "false"
```

Every input is optional and carries the default shown. The Action writes `.agentdiff.yaml` only when
the repo has none, so a committed config wins. On a pull request it posts a sticky comment with the
regression report and the rendered path graph, and exits 1 when it finds a regression.

Without `contents: write`, the PNG push fails and the comment loses its image.
