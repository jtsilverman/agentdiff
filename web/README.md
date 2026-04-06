# AgentDiff Web

Git diff for agent behavior — in a browser. Multi-run comparison dashboard with strategy clustering and drift detection.

## Prerequisites

- Go 1.24+ with CGo enabled (C compiler required for SQLite)
- Node.js 18+
- npm

## Development

Start both API and frontend:

```bash
cd web
make dev
```

Or separately:

```bash
# Terminal 1: Go API on :8080
make dev-api

# Terminal 2: Next.js on :3000
make dev-frontend
```

## Testing

```bash
make test
```

## Deploy

**API (Railway):**
- Push to GitHub, connect repo in Railway
- Set root directory to repo root (Dockerfile references `web/api/`)
- Add persistent volume mounted at `/data`

**Frontend (Vercel):**
- Connect repo, set root directory to `web/frontend`
- Set `NEXT_PUBLIC_API_URL` to your Railway API URL
- Framework preset: Next.js

## Architecture

- **Go API** (`web/api/`): Chi router wrapping existing AgentDiff internal packages. SQLite storage.
- **Next.js Frontend** (`web/frontend/`): App Router + Tailwind + Tremor. Comparison-first dashboard.
- **CLI** (unchanged): Existing `agentdiff` CLI works independently.
