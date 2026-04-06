# AgentDiff Web

**Git diff for agent behavior -- in a browser.** Multi-run comparison dashboard with strategy clustering and drift detection. NOT a single-trace viewer.

## Problem

AgentDiff CLI produces powerful analysis (DBSCAN clustering, Levenshtein alignment, baseline comparison), but results are terminal-only. Teams need to:

1. Upload traces from CI or ad-hoc runs without touching the CLI
2. Visualize strategy clusters across dozens of runs at a glance
3. Diff two traces side-by-side with aligned tool sequences
4. Detect behavioral drift -- "this new run doesn't match any known strategy"

No existing tool does this. LangSmith and AgentOps are single-trace viewers with cost dashboards. Nobody clusters agent behavior or automates divergence root-cause across runs.

## Solution

A web dashboard that wraps the existing AgentDiff Go libraries (8,300 LOC) with a REST API and a comparison-first frontend.

**Architecture decisions:**
- **Go API imports existing `internal/` directly.** Zero duplication. The API is a thin HTTP wrapper around `adapter.Detect`, `cluster.ClusterBaseline`, `diff.CompareToolsWithDiagnostics`/`diff.Align`/`diff.ExtractToolNames`, and `cluster.CompareToCluster`. Note: `adapter.Detect` returns an `Adapter` interface, not a name string. Use a type switch (like `cmd/record.go:adapterSourceName`) to get the adapter name for storage.
- **SQLite replaces file-based storage for web.** The CLI's flat-file store works for local use. Web needs queryable storage: list traces, filter by adapter, join baselines to traces.
- **Next.js + Tremor for frontend.** Tremor provides bar charts, tables, badges out of the box. App Router for file-based routing. No custom charting library.
- **Comparison-first UI.** Landing page shows baselines and their strategy clusters. Single-trace detail is a secondary view reached by clicking into a cluster member. The default question is "how did these runs differ?" not "what did this run do?"
- **Single tenant, no auth.** MVP is self-hosted for one team. Auth is v2.
- **Same repo, new `web/` directory.** Keeps Go module unified so API imports work without replace directives.

## Scope

**Building:**
- Go REST API (`web/api/`) with Chi router, wrapping existing internal packages
- SQLite storage layer (`web/api/db/`) for traces, snapshots, baselines
- Trace upload endpoint with auto-detect (reuses `adapter.Detect`)
- Baseline management endpoints (create from trace IDs, list, detail)
- Strategy cluster endpoint (reuses `cluster.ClusterBaseline`)
- Sequence diff endpoint (reuses `diff.CompareToolsWithDiagnostics` and `diff.Align`)
- Drift detection endpoint (reuses `cluster.CompareToCluster`)
- Next.js 14 frontend (`web/frontend/`) with App Router + Tailwind + Tremor
- Baselines dashboard (landing page) with strategy cluster visualization
- Trace diff view with side-by-side aligned tool sequences
- Drift detection view showing match/no-match with confidence
- Trace upload UI with drag-and-drop JSONL

**Not building:**
- Authentication / multi-tenant
- Cost or token tracking
- Prompt playground or evals
- Live streaming / WebSocket ingestion
- Real-time trace collection (batch upload only)
- OTel ingestion
- CLI changes (existing CLI is untouched)

**Ship target:** GitHub (jtsilverman/agentdiff), tag v0.4.0

## Stack

| Layer | Tech | Why |
|-------|------|-----|
| API | Go 1.24, Chi router | Same module as CLI, direct import of `internal/` |
| Storage | SQLite via go-sqlite3 | Zero-ops, single file, good enough for single tenant |
| Frontend | Next.js 14.x (App Router, pinned) | File-based routing, RSC for initial loads |
| UI Components | Tailwind CSS + Tremor | Dashboard components (charts, tables, badges) out of the box |
| Deploy | Railway (Go API) + Vercel (Next.js) | Free tier for both, separate scaling |

New Go dependencies: `github.com/go-chi/chi/v5`, `github.com/mattn/go-sqlite3` (CGo).

## Architecture

### File Structure (new only)

```
agentdiff/
  web/
    api/
      main.go                 # API entry point, Chi router setup
      routes.go               # Route registration
      handlers/
        traces.go             # POST /api/traces, GET /api/traces, GET /api/traces/:id
        baselines.go          # POST /api/baselines, GET /api/baselines
        cluster.go            # GET /api/baselines/:id/cluster
        diff.go               # GET /api/diff/:idA/:idB
        compare.go            # POST /api/baselines/:id/compare
      db/
        sqlite.go             # Schema init, connection management
        traces.go             # Trace CRUD
        baselines.go          # Baseline CRUD, trace-baseline joins
        snapshots.go          # Snapshot storage (steps, tool calls)
      middleware/
        cors.go               # CORS for frontend
        logging.go            # Request logging
    frontend/
      package.json
      next.config.js
      tailwind.config.js
      tsconfig.json
      src/
        app/
          layout.tsx          # Root layout, nav sidebar
          page.tsx            # Landing: baselines + strategy clusters
          traces/
            page.tsx          # Trace list with upload
            [id]/
              page.tsx        # Trace detail (steps, metadata)
          baselines/
            [id]/
              page.tsx        # Baseline detail: strategy clusters
          diff/
            [idA]/
              [idB]/
                page.tsx      # Side-by-side diff view
        components/
          StrategyCluster.tsx # Cluster visualization (colored groups + noise)
          DiffView.tsx        # Side-by-side aligned tool sequences
          TraceUpload.tsx     # Drag-and-drop JSONL upload
          DriftBadge.tsx      # Match/no-match indicator with distance
          StepList.tsx        # Trace steps renderer
          Nav.tsx             # Sidebar navigation
        lib/
          api.ts              # Fetch wrapper for Go API
          types.ts            # TypeScript types matching API responses
```

### Database Schema (SQLite)

```sql
CREATE TABLE traces (
    id          TEXT PRIMARY KEY,  -- UUID
    name        TEXT NOT NULL,
    adapter     TEXT NOT NULL,     -- claude, openai, agents_sdk, langchain, generic
    source      TEXT NOT NULL DEFAULT '',
    metadata    TEXT,              -- JSON blob
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE snapshots (
    id          TEXT PRIMARY KEY,
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    step_index  INTEGER NOT NULL,
    role        TEXT NOT NULL,     -- user, assistant, tool_call, tool_result
    content     TEXT,
    tool_name   TEXT,
    tool_args   TEXT,              -- JSON
    tool_output TEXT,
    tool_is_error INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE baselines (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE baseline_traces (
    baseline_id TEXT NOT NULL REFERENCES baselines(id),
    trace_id    TEXT NOT NULL REFERENCES traces(id),
    PRIMARY KEY (baseline_id, trace_id)
);

CREATE INDEX idx_snapshots_trace ON snapshots(trace_id, step_index);
CREATE INDEX idx_baseline_traces_baseline ON baseline_traces(baseline_id);
```

### API Endpoints

```
POST   /api/traces
  Body: raw JSONL (Content-Type: application/jsonl or text/plain)
  Query: ?name=my-trace&adapter=auto (adapter optional, default auto-detect)
  Response: { "id": "uuid", "name": "...", "adapter": "claude", "step_count": 12 }
  Logic: adapter.Detect(body) -> parse -> store trace + snapshots in SQLite

GET    /api/traces
  Response: [{ "id", "name", "adapter", "step_count", "created_at" }]

GET    /api/traces/:id
  Response: { "id", "name", "adapter", "metadata", "steps": [...], "created_at" }
  Steps include role, content, tool_name, tool_args, tool_output.

POST   /api/baselines
  Body: { "name": "main-baseline", "trace_ids": ["uuid1", "uuid2", ...] }
  Response: { "id": "uuid", "name": "...", "trace_count": 5 }

GET    /api/baselines
  Response: [{ "id", "name", "trace_count", "created_at" }]

GET    /api/baselines/:id/cluster
  Query: ?epsilon=0&min_points=2 (optional, default auto)
  Response: {
    "baseline_name": "...",
    "snapshot_count": 10,
    "strategies": [{ "id": 0, "count": 4, "exemplar": "trace-name", "tool_sequence": [...], "members": [...] }],
    "noise": ["trace-name-7"],
    "epsilon": 3.5
  }
  Logic: Load traces -> build snapshot.Baseline -> cluster.ClusterBaseline()

GET    /api/diff/:idA/:idB
  Response: {
    "trace_a": { "id", "name" },
    "trace_b": { "id", "name" },
    "alignment": [{ "a_step": {...}|null, "b_step": {...}|null, "op": "match|insert|delete|substitute" }],
    "distance": 3,
    "summary": { "matches": 8, "insertions": 1, "deletions": 0, "substitutions": 2 }
  }
  Logic: Load both traces -> extract tool names via diff.ExtractToolNames(steps) -> diff.Align(seqA, seqB) for alignment pairs -> diff.Levenshtein(seqA, seqB) for distance -> map AlignedPair ops (AlignMatch=0, AlignSubst=1, AlignInsert=2, AlignDelete=3) to string labels -> join aligned pairs back to full step data for response

POST   /api/baselines/:id/compare
  Body: { "trace_id": "uuid" }
  Response: {
    "matched": true,
    "strategy_id": 0,
    "exemplar": "trace-name",
    "distance": 2,
    "max_intra_cluster_dist": 4
  }
  Logic: Load baseline + trace -> cluster.CompareToCluster()
```

### Frontend Pages

**Landing (`/`):** Baselines list with card per baseline. Each card shows: name, trace count, strategy count (fetched from cluster endpoint). Click a baseline to see its cluster detail. Prominent "Upload Traces" button in header.

**Baseline Detail (`/baselines/:id`):** Strategy cluster visualization. Each strategy is a colored group showing: member count, exemplar trace name, tool sequence as a horizontal pill chain. Noise traces listed separately in gray. Click any two members to jump to diff view. "Compare New Trace" button triggers drift check.

**Trace List (`/traces`):** Table of all uploaded traces. Columns: name, adapter, step count, date. Drag-and-drop upload zone at top. Bulk select traces to create a new baseline.

**Trace Detail (`/traces/:id`):** Step-by-step view. Each step shows role, content/tool info. Secondary view -- reached from clicking a trace name anywhere else in the UI.

**Diff View (`/diff/:idA/:idB`):** Two-column layout. Left = Trace A, Right = Trace B. Steps aligned vertically using Levenshtein alignment. Color coding: green = match, yellow = substitution, red = insertion/deletion. Step details expand on click.

## Tasks

### Task 1: SQLite Storage Layer

**Files:** `web/api/db/sqlite.go`, `web/api/db/traces.go`, `web/api/db/baselines.go`, `web/api/db/snapshots.go`

**Do:**
- Create `web/api/db/sqlite.go`: `NewDB(path string) (*DB, error)` opens SQLite connection, runs schema migration (CREATE TABLE IF NOT EXISTS for all 4 tables + indexes). `DB` struct wraps `*sql.DB`. `Close()` method.
- Create `web/api/db/traces.go`: `CreateTrace(name, adapter string, metadata map[string]string) (Trace, error)` generates UUID, inserts row. `ListTraces() ([]TraceSummary, error)` returns all traces with step count via subquery. `GetTrace(id string) (TraceDetail, error)` returns trace with all snapshots ordered by step_index.
- Create `web/api/db/snapshots.go`: `InsertSnapshots(traceID string, steps []snapshot.Step) error` batch-inserts steps. Maps `snapshot.Step` to snapshot table rows: role from step type, content from Content field, tool fields from ToolCall/ToolResult.
- Create `web/api/db/baselines.go`: `CreateBaseline(name string, traceIDs []string) (Baseline, error)` inserts baseline + join rows. `ListBaselines() ([]BaselineSummary, error)` with trace counts. `GetBaselineTraces(baselineID string) ([]TraceDetail, error)` loads all traces with their snapshots for a baseline.
- Define Go structs: `Trace`, `TraceSummary`, `TraceDetail`, `Baseline`, `BaselineSummary` in their respective files.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build ./web/api/db/`

**Dependencies:** none

### Task 2: API Server and Trace Handlers

**Files:** `web/api/main.go`, `web/api/routes.go`, `web/api/handlers/traces.go`, `web/api/middleware/cors.go`, `web/api/middleware/logging.go`

**Do:**
- Create `web/api/main.go`: parse `--port` flag (default 8080) and `--db` flag (default `agentdiff.db`). Init DB, init Chi router, register routes, start HTTP server.
- Create `web/api/middleware/cors.go`: allow all origins for MVP (`Access-Control-Allow-Origin: *`, allow POST/GET/OPTIONS, allow Content-Type header).
- Create `web/api/middleware/logging.go`: log method, path, status, duration to stdout.
- Create `web/api/routes.go`: `RegisterRoutes(r chi.Router, db *db.DB)` wires all handlers.
- Create `web/api/handlers/traces.go`:
  - `PostTrace(db)` handler: read raw body, get `name` and `adapter` from query params. If adapter is empty or "auto", call `adapter.Detect(body)` which returns an `Adapter` interface. Get adapter name string via type switch (match pattern from `cmd/record.go:adapterSourceName`: `*adapter.ClaudeAdapter` -> "claude", `*adapter.OpenAIAdapter` -> "openai", etc.). Parse with detected adapter via `detectedAdapter.Parse(body)` (method on the Adapter instance, not a package function). Store trace + snapshots via db. Return JSON with id, name, adapter, step_count.
  - `ListTraces(db)` handler: call db.ListTraces(), return JSON array.
  - `GetTrace(db)` handler: extract `:id` from URL, call db.GetTrace(), return JSON.
- Error responses as `{"error": "message"}` with appropriate HTTP status codes (400, 404, 500).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build ./web/api/`

**Dependencies:** Task 1

### Task 3: Baseline and Cluster Handlers

**Files:** `web/api/handlers/baselines.go`, `web/api/handlers/cluster.go`

**Do:**
- Create `web/api/handlers/baselines.go`:
  - `PostBaseline(db)`: parse JSON body `{"name": "...", "trace_ids": [...]}`. Validate name non-empty, at least 1 trace ID. Call db.CreateBaseline(). Return JSON with id, name, trace_count.
  - `ListBaselines(db)`: call db.ListBaselines(), return JSON array.
- Create `web/api/handlers/cluster.go`:
  - `GetCluster(db)`: extract `:id` from URL. Read `epsilon` and `min_points` from query params (default 0 and 2). Call db.GetBaselineTraces() to load all traces with snapshots. Convert db traces to `snapshot.Baseline` (build `snapshot.Snapshot` for each trace with its steps). Call `cluster.ClusterBaseline(baseline, epsilon, minPts)`. Return `StrategyReport` as JSON.
  - The conversion from DB model to `snapshot.Baseline` should be a helper function `toSnapshotBaseline(traces []TraceDetail) snapshot.Baseline` in this file.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build ./web/api/`

**Dependencies:** Task 1, Task 2

### Task 4: Diff and Compare Handlers

**Files:** `web/api/handlers/diff.go`, `web/api/handlers/compare.go`

**Do:**
- Create `web/api/handlers/diff.go`:
  - `GetDiff(db)`: extract `:idA` and `:idB` from URL. Load both traces via db.GetTrace(). Convert DB steps to `[]snapshot.Step`. Extract tool name sequences via `diff.ExtractToolNames(steps)`. Call `diff.Align(seqA, seqB)` for alignment (returns `AlignResult` with `Pairs []AlignedPair`). Call `diff.Levenshtein(seqA, seqB)` for edit distance. Map `AlignedPair.Op` int enum (AlignMatch=0, AlignSubst=1, AlignInsert=2, AlignDelete=3) to string labels. Join aligned pairs back to full step data using `IndexA`/`IndexB` (index into tool-call-only subsequences). For insert/delete ops, one side is null.
  - Define `DiffPair` struct: `AStep *StepJSON`, `BStep *StepJSON`, `Op string`.
  - Define `DiffResponse` struct with trace metadata, alignment array, distance, summary counts.
- Create `web/api/handlers/compare.go`:
  - `PostCompare(db)`: extract baseline `:id` from URL. Parse JSON body `{"trace_id": "..."}`. Load baseline traces and the comparison trace. Convert to snapshot types. Call `cluster.CompareToCluster()`. Return `MatchResult` as JSON.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build ./web/api/`

**Dependencies:** Task 1, Task 2

### Task 5: API Integration Tests

**Files:** `web/api/handlers/handlers_test.go`

**Do:**
- Create test file with `httptest.Server` wrapping the full Chi router with an in-memory SQLite DB (`:memory:`).
- `TestPostAndGetTrace`: POST a Claude JSONL trace (use content from `testdata/`), verify 200 response with correct adapter detection. GET the trace by ID, verify steps returned.
- `TestListTraces`: POST 2 traces, GET /api/traces, verify both returned.
- `TestBaselineAndCluster`: POST 6 traces (3 with tool sequence [search, summarize], 3 with [lookup, answer]). Create baseline from all 6. GET cluster endpoint, verify 2 strategies found.
- `TestDiff`: POST 2 different traces, GET diff, verify alignment returned with correct distance.
- `TestCompare`: create baseline, POST compare with a new trace, verify matched/unmatched response.
- `TestErrors`: POST trace with empty body (400), GET nonexistent trace (404), create baseline with bad trace IDs (400).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./web/api/handlers/ -v`

**Dependencies:** Task 2, Task 3, Task 4

### Task 6: Frontend Scaffold and Layout

**Files:** `web/frontend/package.json`, `web/frontend/next.config.js`, `web/frontend/tailwind.config.js`, `web/frontend/tsconfig.json`, `web/frontend/src/app/layout.tsx`, `web/frontend/src/components/Nav.tsx`, `web/frontend/src/lib/api.ts`, `web/frontend/src/lib/types.ts`

**Do:**
- Initialize Next.js 14 project in `web/frontend/`. Pin versions: `next@14.2`, `react@18`, `react-dom@18`, `@tremor/react@3`, `tailwindcss@3`, `postcss`, `autoprefixer`, `typescript`, `@types/react`, `@types/node`. Also install `eslint` and `eslint-config-next@14.2` for linting.
- Create `next.config.js`: configure API proxy rewrite from `/api/*` to `http://localhost:8080/api/*` for dev.
- Create `tailwind.config.js`: include `src/**/*.tsx`, extend with Tremor preset.
- Create `src/lib/types.ts`: TypeScript interfaces matching all API response types (TraceSummary, TraceDetail, Step, Baseline, BaselineSummary, StrategyReport, Strategy, DiffResponse, AlignedPair, MatchResult).
- Create `src/lib/api.ts`: fetch wrapper with base URL from env var `NEXT_PUBLIC_API_URL` (default `/api`). Functions: `uploadTrace()`, `listTraces()`, `getTrace()`, `createBaseline()`, `listBaselines()`, `getCluster()`, `getDiff()`, `compareTrace()`.
- Create `src/app/layout.tsx`: root layout with sidebar nav (Nav component) and main content area. Dark header with "AgentDiff" title.
- Create `src/components/Nav.tsx`: sidebar with links to `/` (Baselines), `/traces` (Traces). Active state highlighting.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff/web/frontend && npm install && npx tsc --noEmit && npm run build`

**Dependencies:** none

### Task 7: Baselines Dashboard (Landing Page)

**Files:** `web/frontend/src/app/page.tsx`, `web/frontend/src/components/StrategyCluster.tsx`, `web/frontend/src/components/DriftBadge.tsx`

**Do:**
- Create `src/app/page.tsx`: server component that fetches baselines list. Renders grid of baseline cards. Each card shows: name, trace count, and a "View Clusters" link. If no baselines exist, show empty state with prompt to upload traces first.
- Create `src/components/StrategyCluster.tsx`: client component. Props: `StrategyReport`. Renders each strategy as a colored card (Tremor Card) with: strategy ID badge, member count, exemplar name, tool sequence as horizontal pill chain (Tremor Badge for each tool). Noise traces rendered in a separate gray section. Use distinct Tremor colors for each strategy (blue, green, amber, purple, cycling).
- Create `src/components/DriftBadge.tsx`: props: `MatchResult`. Green badge "Matches Strategy N" if matched. Red badge "New Strategy Detected" if not matched. Shows distance value.
- Wire baseline detail page at `src/app/baselines/[id]/page.tsx`: fetches cluster data, renders StrategyCluster component. Includes "Compare Trace" button that opens a modal/dropdown to select a trace for comparison, shows DriftBadge with result.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff/web/frontend && npx tsc --noEmit`

**Dependencies:** Task 6 (Task 3 for API contract reference)

### Task 8: Trace Upload and Diff Views

**Files:** `web/frontend/src/app/traces/page.tsx`, `web/frontend/src/app/traces/[id]/page.tsx`, `web/frontend/src/app/diff/[idA]/[idB]/page.tsx`, `web/frontend/src/components/TraceUpload.tsx`, `web/frontend/src/components/DiffView.tsx`, `web/frontend/src/components/StepList.tsx`

**Do:**
- Create `src/components/TraceUpload.tsx`: client component. Drag-and-drop zone for JSONL files. On drop: read file, POST to `/api/traces` with file content as body and filename as name param. Show success/error state. Support optional name override input.
- Create `src/components/StepList.tsx`: renders array of steps. Each step shows role as colored label (user=blue, assistant=green, tool_call=amber, tool_result=purple), content or tool details. Collapsible tool args/output for long content.
- Create `src/app/traces/page.tsx`: Tremor Table of traces (name, adapter, step count, date). TraceUpload component above the table. Bulk select checkboxes + "Create Baseline" button that prompts for name and calls POST /api/baselines.
- Create `src/app/traces/[id]/page.tsx`: fetch trace detail, render with StepList. Link to diff this trace against another (dropdown to pick second trace).
- Create `src/components/DiffView.tsx`: client component. Props: `DiffResponse`. Two-column layout. Each row is an aligned pair. Color coding: green background for match, yellow for substitution (show both sides), red-left for deletion (empty right), red-right for insertion (empty left). Summary bar at top with match/insert/delete/sub counts.
- Create `src/app/diff/[idA]/[idB]/page.tsx`: fetch diff data, render DiffView.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff/web/frontend && npx tsc --noEmit`

**Dependencies:** Task 6 (Task 4 for API contract reference)

### Task 9: End-to-End Integration and Deploy Config

**Files:** `web/api/Dockerfile`, `web/frontend/.env.example`, `web/Makefile`, `web/README.md`

**Do:**
- Create `web/api/Dockerfile`: multi-stage build. Stage 1: Go build with CGo enabled (for go-sqlite3). Stage 2: minimal runtime with `agentdiff.db` volume mount. Expose port 8080.
- Create `web/frontend/.env.example`: `NEXT_PUBLIC_API_URL=http://localhost:8080/api`.
- Create `web/Makefile` with targets:
  - `dev-api`: `cd api && go run . --port 8080`
  - `dev-frontend`: `cd frontend && npm run dev`
  - `dev`: run both in parallel
  - `test`: `cd api && go test ./...`
  - `build-api`: `cd api && go build -o agentdiff-api .`
  - `build-frontend`: `cd frontend && npm run build`
- Create `web/README.md`: setup instructions (prerequisites, dev workflow, deploy to Railway + Vercel). Include screenshot placeholder for each main view.
- Verify full build: Go API compiles, frontend compiles, API tests pass.
- Add `railway.json` in `web/api/` for Railway deploy config (build command, start command).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build ./web/api/ && cd web/frontend && npm run build`

**Dependencies:** Task 5, Task 7, Task 8

## The One Hard Thing

**Strategy cluster visualization with real-time drift detection.**

The existing DBSCAN clustering and Levenshtein alignment are solved problems in the CLI. The hard thing for the web version is the bridge between algorithmic output and visual understanding:

1. **Cluster-to-visual mapping.** DBSCAN produces cluster IDs and member indices. The frontend must render these as visually distinct groups where the relationship between strategies is immediately obvious. This means: consistent color assignment per strategy, spatial grouping that reflects edit distance (close strategies appear near each other), and noise points visually separated but not hidden.

2. **Diff alignment rendering.** The Levenshtein alignment produces matched/inserted/deleted/substituted step pairs. Rendering this as a two-column diff that stays vertically aligned despite different step counts is a layout problem -- similar to GitHub's side-by-side diff but for structured agent steps instead of text lines. Empty cells for insertions/deletions must maintain row correspondence.

3. **Drift detection UX.** When a user uploads a new trace and compares against a baseline, the response must be instantly legible: "matches Strategy A (distance 2)" or "NEW STRATEGY -- closest to Strategy B (distance 7, threshold 4)." The confidence metric (distance vs. max intra-cluster distance) needs visual encoding that non-ML users can interpret.

Jake should be able to explain: why DBSCAN over k-means for agent strategies (unknown cluster count, non-spherical clusters), how Levenshtein alignment maps to the diff view (edit operations become visual rows), and why the comparison threshold uses max intra-cluster distance (it's the natural boundary of what the cluster considers "same behavior").

## Risks

1. **Becoming a single-trace viewer (high).** The biggest risk is that the trace detail page becomes the primary view and the dashboard devolves into "LangSmith but worse." Mitigation: landing page is baselines-only. Trace detail is deliberately minimal -- no fancy timeline, no token counts, no cost. The value proposition is comparison, not inspection.

2. **CGo dependency for SQLite (medium).** `go-sqlite3` requires CGo (`CGO_ENABLED=1`) and a C compiler (gcc/clang). On macOS, requires Xcode Command Line Tools. Complicates cross-compilation and Docker builds. Mitigation: the Dockerfile uses a CGo-enabled build stage with gcc. For Railway, ensure the build environment has gcc. If this becomes painful, swap to `modernc.org/sqlite` (pure Go) -- API-compatible drop-in.

3. **Snapshot conversion fidelity (medium).** Converting between the CLI's `snapshot.Step` model and the SQLite rows could lose information (e.g., nested tool args serialized as JSON strings). Mitigation: store tool_args and metadata as JSON text columns. Round-trip test in Task 5 verifies no data loss.

4. **Frontend complexity creep (medium).** Tremor components are opinionated. Custom visualizations (cluster spatial layout, diff alignment) may fight the library. Mitigation: use Tremor for standard UI (tables, cards, badges) and plain Tailwind for custom layouts (diff view, cluster groups). Don't force Tremor where it doesn't fit.

5. **CORS and deploy topology (low).** Frontend on Vercel calling API on Railway introduces CORS. Mitigation: explicit CORS middleware in Task 2. In production, set `Access-Control-Allow-Origin` to the Vercel domain instead of `*`.
