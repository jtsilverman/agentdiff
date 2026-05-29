# AgentDiff Web

Git diff for agent behavior — in a browser. Multi-run comparison dashboard with strategy clustering and drift detection.

## Prerequisites

- Go 1.24+ with CGo enabled (C compiler required for SQLite)

## Development

The Go API serves both the static site (`web/site/`) at `/` and the API at `/api/*` from a single port.

```bash
cd web
make dev
```

Or directly:

```bash
cd web/api && go run . -port 8080 -site ../site
# open http://localhost:8080
```

## Testing

```bash
make test
```

## Deploy

**Frontend (Vercel):**
- Connect repo, set root directory to `web/site`
- Framework preset: Other (no framework, no build)
- `web/site/vercel.json` pins these settings

**API (Railway):**
- Push to GitHub, connect repo in Railway
- Set root directory to repo root (Dockerfile at `web/api/Dockerfile`)
- Add persistent volume mounted at `/data`

## Architecture

- **Go API** (`web/api/`): Chi router wrapping AgentDiff internal packages. SQLite storage. Serves `web/site/` statically at `/`, API at `/api/*`.
- **Static site** (`web/site/`): Plain HTML/CSS/JS — home, demo, install, about. Path graph + scrubber + diff + counterfactual + edit-prompt views run in the browser against canned data in `assets/data.js`.
- **CLI** (unchanged): Existing `agentdiff` CLI works independently.
