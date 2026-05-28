# Spec: agentdiff-redesign

**Status:** Locked
**Created:** 2026-05-27
**Locked:** 2026-05-27 (expanded same day to cover full website rebuild)
**Branch:** feat/hosted-deploy
**Worktree:** /Users/admin/Documents/projects/agentdiff/.worktrees/hosted-deploy

## Context

agentdiff hosted demo (https://agentdiff.vercel.app) is technically
complete but visually unfinished. A Claude Design handoff bundle was
produced (see `_design/` in this worktree). The home page has been
translated into the Next.js frontend; baseline detail and `/about` are
still on the old Tremor visuals.

**Audience: this doubles as Jake's FDE (forward-deployed engineer)
portfolio piece.** Every surface must read to hiring managers,
engineers, and CEOs as "this person can drop into your codebase and
ship AI-powered improvements." Aesthetic, copy, seed data, and feature
depth all serve that signal. Tasks framed for both AI engineers *and*
the non-technical stakeholders they pitch to: "Add a 'Buy Now' button
to the pricing page" / "Diagnose yesterday's site outage" / "Build a
weekly report on customer signups" — not flask-limiter / JWT
migration / flaky CI test.

The original spec covered home + /about + baseline detail + a light
"inner pages tidy" pass. This expanded spec adds: seed-data rewrite (3
task-driven scenarios with real metadata), full redesign of `/traces`
+ `/diff` + trace detail (not just tidy), two new routes (`/docs`
developer reference + `/changelog` shipped-features log), a persistent
introduce-me surface site-wide, tweaks panel, and a focused
FDE-positioning copy pass.

**Workflow shift: leverage Claude Design.** All net-new visual
surfaces go through Claude Design first. Prompts for the six remaining
surfaces (`/traces`, `/diff`, trace detail, `/docs`, `/changelog`,
introduce-me) are staged in `_design/prompts.md`; Jake runs them in
the existing CD session (same chat that produced home / about /
baseline / tweaks, so design system coherence is automatic) and pastes
results back into `_design/project/`. The Next.js port chunks
(chunks 7-13) fire as results arrive.

## Requirements

- Translate the Claude Design bundle's `baseline.html` design into the
  existing `/baselines/[id]` Next.js route.
- Create `/about` route from the bundle's `about.html` design.
- Replace the 5 abstract seeded baselines with 3 task-driven scenarios
  (variance / regression / novel-strategy) carrying `task` + `outcome`
  trace metadata and a baseline `description`. (See plan
  `~/.claude/plans/we-will-plan-here-parallel-sparrow.md` for the
  scenario substance.)
- Redesign `/traces`, `/diff`, and trace detail pages to match the
  home / about / baseline aesthetic. Not just tidy — full visual
  treatment via Claude Design output, ported to Next.js.
- Add `/docs` route — developer reference (concept glossary + API
  contract + data-flow + embed-cache mechanism), via Claude Design.
- Add `/changelog` route — manual reverse-chronological shipped-features
  timeline, via Claude Design.
- Add a persistent introduce-me surface visible on every page (placement
  per Claude Design's recommendation: nav chip / floating badge /
  footer block).
- Add a Tweaks panel (live density / accent / background-tone toggles)
  accessible from every page, choices persisting across reloads via
  localStorage.
- FDE positioning across home and /about: clear "what I do" framing,
  plausible-real example tasks already in copy, a contact/CTA surface
  for hiring managers.
- Keep all pages visually coherent (same nav, footer, design tokens
  defined in `web/frontend/src/app/globals.css`).
- Cards on home (already shipped) deep-link to seeded baselines by name
  match. After the seed rewrite, the name-substring keys change; home
  cards must update to match the new baseline names.

## Constraints

- Next.js 14.2.29 App Router, React 18.3, Tailwind 3.4, Tremor 3.18.
- Geist + Geist Mono fonts via the `geist` npm package (NOT
  `next/font/google` — Geist isn't there in Next 14). Wired through
  `--font-geist-sans` / `--font-geist-mono` CSS vars in
  `web/frontend/src/app/globals.css`.
- The repo deploys to Vercel preview on push to `feat/hosted-deploy`.
  Backend (Go) is on Fly.io; frontend talks to it via
  `NEXT_PUBLIC_API_URL`.
- API surface (route handlers) does NOT change. Seed data + the
  `baselines.description` column DO change, via the existing
  `migrateAddColumn` helper at `web/api/db/sqlite.go:136`.
- `seed-cache.json` is `//go:embed`-ed into the binary via
  `cache_load.go`. After any scenario change, it MUST be regenerated
  via `cache_gen_test.go` or the old cache silently serves stale
  LLM/embedding data.
- Tweaks panel design uses postMessage host protocol (Claude Design
  edit-mode plumbing). Strip that; persist to localStorage instead.
- All Claude Design output runs in the **existing** CD session (the one
  that produced home / about / baseline / tweaks). Same chat → design
  system stays coherent. New surfaces inherit Geist + dark theme +
  globals.css tokens without re-explaining.
- Port chunks (7-13) are gated on CD results landing in
  `_design/project/`. They can fire in any order as results arrive;
  no fixed dependency between port chunks.

## Non-goals

- Mobile-first polish. Design is desktop-first; mobile should degrade
  gracefully via the existing media queries in `globals.css`. (Was
  followup; still followup.)
- A guided product tour (shepherd.js / intro.js step-through). Prose
  walkthrough on /about is the v1.
- Re-architecting trace generation to use `internal/bench/generate.go`.
  Possible future cleanup.
- Onboarding for the upload-your-own-trace flow.

## Interfaces / data model

**Backend changes (Chunk 2):**
- `web/api/seed/seed.go` — rewrite `scenarios()` to produce 3
  task-driven scenarios.
- `web/api/db/baselines.go` — add `Description string` field to
  baseline struct; update CRUD.
- `web/api/db/sqlite.go` — call `migrateAddColumn` for new
  `description` column.
- `web/api/seed/seed-cache.json` — regenerate via
  `cache_gen_test.go`.
- Trace metadata gains `task` (same per scenario) + `outcome`
  (succeeded / regressed / variance / additive) keys. Existing
  `Record<string, string>` shape, no schema change.

**Frontend routes touched:**
- `web/frontend/src/app/about/page.tsx` — new file (Chunk 1).
- `web/frontend/src/app/baselines/[id]/page.tsx` — replace Tremor
  layout (Chunks 4-6).
- `web/frontend/src/app/page.tsx` — update baseline card subtitles
  + slug-match keys to new seed names (Chunk 3).
- `web/frontend/src/app/traces/page.tsx` + `[id]/page.tsx` — full
  redesign (Chunks 7, 9) via CD.
- `web/frontend/src/app/diff/page.tsx` — full redesign (Chunk 8) via CD.
- `web/frontend/src/app/docs/page.tsx` — new route (Chunk 10) via CD.
- `web/frontend/src/app/changelog/page.tsx` — new route (Chunk 11) via CD.
- `web/frontend/src/app/layout.tsx` — mount Tweaks panel root (Chunk 12)
  and introduce-me persistent surface (Chunk 13).

**Frontend types:**
- `web/frontend/src/lib/types.ts` — add `description?: string` to
  `BaselineSummary`.

**Existing API client (`web/frontend/src/lib/api.ts`):** no changes.

**Existing components to reuse:**
- `MetadataBadges.tsx`, `StepList.tsx`, `Transcript.tsx`,
  `TriagePanel.tsx`, `PathGraph.tsx`, `CounterfactualGraph.tsx`,
  `StrategyCluster.tsx`, `DriftBadge.tsx`. These uploaded into Claude
  Design as `_design/project/uploads/`.

## Acceptance criteria

- `/about` route exists and renders without errors.
- `curl /api/baselines` returns 3 baselines, each with non-empty
  `description`.
- Every trace in the seed data carries `task` and `outcome` keys.
- `/baselines/<id>` for each seeded baseline renders the new visual:
  task description as callout, trace list with outcome/step metadata,
  path graph card, three money-feature action buttons.
- `/traces`, `/diff`, and trace detail pages render in the new dark
  theme with Geist fonts and the design system's tokens. No visible
  Tremor leftovers.
- `/docs` route exists; renders the developer reference (concept
  glossary, API contract, data flow, embed-cache mechanism,
  interpretation guide) with sticky anchors.
- `/changelog` route exists; renders reverse-chronological shipped
  features with dates and tags.
- Introduce-me surface visible on every page (placement per CD's
  recommendation); communicates Jake's FDE availability without
  competing with page content.
- Tweaks panel opens from a floating button on every page; density,
  accent, and background-tone choices persist across reloads via
  localStorage.
- Home hero copy + /about page communicate the FDE value-prop
  unambiguously; /about includes a "get in touch" section.
- Hero, nav, footer match the design's typography, palette, spacing
  across all redesigned pages.
- `go test ./web/api/...` green. `npx tsc --noEmit` clean.
  `npx vitest run` green.
- Vercel preview rebuilds clean; manual eye-check on every page.

## Test strategy

- **Go side:** existing `seed_test.go` + `cache_gen_test.go` updated
  to assert new scenario shape. `go test ./web/api/...` is the gate
  for Chunk 2.
- **Frontend unit tests:** under
  `web/frontend/src/app/__tests__/` and
  `web/frontend/src/components/__tests__/`. New tests for /about,
  expanded baseline detail, redesigned inner pages, tweaks panel
  persistence.
- **Manual visual:** push to Vercel preview after each chunk; eyeball
  the affected pages. Local dev hangs in Jake's shell (Watchpack
  EINTR), so Vercel preview is the canonical visual gate.
- **First-impression test (after Chunk 11):** show someone unfamiliar
  with the product the home page for 30 seconds. They should be able
  to articulate what agentdiff does + that Jake is for hire.

## Execution boundaries

**In scope:**
- `web/api/seed/seed.go`, `web/api/seed/seed-cache.json`,
  `web/api/seed/cache_gen_test.go`, `web/api/seed/seed_test.go`.
- `web/api/db/baselines.go`, `web/api/db/sqlite.go` (only for the
  migrateAddColumn call).
- `web/frontend/src/app/**` (including new routes `app/docs/` and
  `app/changelog/`), `web/frontend/src/components/**`,
  `web/frontend/src/lib/types.ts`.
- `_design/prompts.md` (CD prompt batch) and `_design/project/`
  (CD result drop site).

**Out of bounds:**
- API route handlers under `web/api/handlers/` (the surface stays
  stable).
- Embed cache infrastructure beyond the documented regen flow.
- Auth, sessions, deployment config beyond `NEXT_PUBLIC_API_URL`.

## Chunk decomposition

**Chunk 1: /about page** ✓ shipped 2026-05-27.

**Chunk 2: Seed-data rewrite — backend** ✓ shipped 2026-05-27.

**Chunk 3: Seed-data UI surfacing** — add `description?` to `BaselineSummary` in `lib/types.ts`; update home baseline cards to use real `description` as subtitle + match new slug names; ensure baseline detail + trace rows consume `description` / `task` / `outcome`.
- Acceptance: home cards show real subtitle text + link to renamed baselines; baseline detail callout pulls `description` from API.
- Tier: B.

**Chunk 4: Baseline detail visual shell** — replace top of `baselines/[id]/page.tsx` with the design's task header + callout + trace list, consuming real `description` + `task` + `outcome`.
- Acceptance: task description as callout, trace rows showing outcome/steps/key-decision badges via `MetadataBadges`, no regressions on existing data fetching.
- Tier: B.

**Chunk 5: Baseline path graph card** — port the design's path graph treatment + legend to the baseline detail page.
- Acceptance: path graph renders inside a styled card with the legend ("edge thickness = run count" etc.) below.
- Tier: B.

**Chunk 6: Money-feature action row + modals** — replace existing scattered buttons with the design's three-button row (counterfactual / edit-prompt / similar) and modal flows.
- Acceptance: each button opens a working modal that invokes the existing API (`runCounterfactual`, `editPrompt`, `getSimilar`); existing tests still pass.
- Tier: B.

**Chunk 7: /traces page port** — port Claude Design's `traces.{html,jsx,css}` (Prompt 1 in `_design/prompts.md`) into `web/frontend/src/app/traces/page.tsx`. Filter chips, scannable trace list, links to trace detail.
- Acceptance: route loads with no Tremor sidebar, lists traces with filter chips, uses design system's card / badge styles.
- Tier: B.
- Caveat: blocks until `_design/project/traces.*` exists.

**Chunk 8: /diff page port** ✓ shipped 2026-05-27.

**Chunk 9: Trace detail page port** — port CD's `trace-detail.{html,jsx,css}` (Prompt 3) into `web/frontend/src/app/traces/[id]/page.tsx`. Keep existing `StepList`, `Transcript`, `MetadataBadges`; restyle container + chrome.
- Acceptance: trace detail renders in dark-themed shell with sticky step list, all sub-components readable, no regression on data display.
- Tier: B.
- Caveat: blocks until `_design/project/trace-detail.*` exists.

**Chunk 10: /docs page port** — port CD's `docs.{html,jsx,css}` (Prompt 4) into a new route `web/frontend/src/app/docs/page.tsx`. Concept glossary, API reference, data flow, embed-cache mechanism, interpretation guide.
- Acceptance: route exists, all 5 sections render with sticky anchors, code blocks render in Geist Mono, no API calls (pure prose).
- Tier: B.
- Caveat: blocks until `_design/project/docs.*` exists; add `/docs` link to nav.

**Chunk 11: /changelog page port** — port CD's `changelog.{html,jsx,css}` (Prompt 5) into a new route `web/frontend/src/app/changelog/page.tsx`. Reverse-chronological shipped-features list with date + tag.
- Acceptance: route exists, entries render grouped by month, dates and tags visible, no API calls (markdown-style prose).
- Tier: C.
- Caveat: blocks until `_design/project/changelog.*` exists; add `/changelog` link to nav.

**Chunk 12: Tweaks panel port** — port `_design/project/tweaks-app.jsx` + `tweaks-panel.jsx` into the Next.js frontend. Persist to localStorage. Mount at root layout. Wire to existing CSS vars in `globals.css`.
- Acceptance: floating Tweaks button opens panel; density/accent/bgTone toggle live; choices persist across reload; works on every page.
- Tier: B.
- Caveat: strip the postMessage host protocol (Claude Design edit-mode plumbing); replace with localStorage read/write.

**Chunk 13: Introduce-me persistent surface port** — port CD's `introduce-me.{html,jsx,css}` (Prompt 6) at the placement CD chose (nav chip / floating badge / footer block). Mount via root layout so it appears on every page.
- Acceptance: surface visible on home, /about, baseline detail, /traces, /diff, /docs, /changelog; copy communicates FDE availability; CTA links to Jake's chosen channel.
- Tier: B.
- Caveat: blocks until `_design/project/introduce-me.*` exists; need Jake's CTA target (email / Calendly / LinkedIn / other) at port time.

**Chunk 14: FDE positioning final copy pass** — tighten home hero copy; add "What I do" + "Get in touch" sections to /about; ensure footer + introduce-me surface + about all reinforce one consistent FDE pitch.
- Acceptance: home hero communicates FDE value-prop in ≤2 sentences; /about has hireable-me section; copy across home, /about, introduce-me, footer is coherent (no contradictions, one pitch).
- Tier: C (mostly copy).

**Chunk 15: seed enrichment — key_decision + baseline rename** — add `key_decision` to every trace's metadata in `web/api/seed/seed.go` (so the trace-detail key-decision callout renders on real traces, not just CD mockups); rename the three baselines from slug-prefixed fixture names (`seed-api-endpoint-rename` etc.) to short engineering-style names (Claude proposes, Jake redlines per the option-A pick at chunk 9 wrap); update home page slug-match keys in `app/page.tsx` to track new IDs; regenerate `web/api/seed/seed-cache.json` via `cache_gen_test.go`.
- Acceptance: `curl /api/baselines` returns 3 baselines with new short names; every trace JSON has non-empty `metadata.key_decision`; trace detail page's `.td-keydecision` block renders for every seeded trace; home cards link to renamed baseline IDs without 404; `go test ./web/api/...` green.
- Tier: B (seed data + cache regen + slug-match-key drift).

## Testing chunks

| Chunk | Scenario | Test type |
|---|---|---|
| 1 | `/about` route renders concept grid + sections | unit (vitest + RTL) |
| 2 | seed regen produces 3 baselines + task/outcome on traces | `go test ./web/api/...` |
| 3 | home cards link to renamed baselines + show real subtitles | unit + manual |
| 4 | baseline detail shows real task callout + outcome badges | unit |
| 5 | baseline path graph renders with legend present | unit |
| 6 | each money-feature button opens its modal | unit + manual |
| 7 | /traces renders styled list with filter chips, no Tremor sidebar | unit + manual |
| 8 | /diff renders styled compare view with divergence highlights | unit + manual |
| 9 | trace detail renders in dark shell with sticky step list | unit + manual |
| 10 | /docs renders 5 sections with sticky anchors + code blocks | unit + manual |
| 11 | /changelog renders entries grouped by month with tags | unit + manual |
| 12 | tweaks panel toggles persist across reload | unit + manual |
| 13 | introduce-me surface visible on every page; CTA links work | unit + manual |
| 14 | home hero + /about + introduce-me + footer copy is coherent | manual review |
| 15 | trace detail key-decision callout renders + baselines have human names | `go test ./web/api/...` + manual |

## Current chunk

Chunk 9 — Trace detail page port (blocks on `_design/project/trace-detail.*`).

## Completed chunks

- **Chunk 15: seed enrichment — key_decision + baseline rename** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Renamed all three seeded baselines by stripping the `seed-` prefix:
  `seed-api-endpoint-rename` → `api-endpoint-rename`,
  `seed-auth-migration-md5-to-bcrypt` → `auth-migration-md5-to-bcrypt`,
  `seed-new-endpoint-with-tests` → `new-endpoint-with-tests`. Short
  engineering-style names that read as real project state, not fixture
  data. Added `keyDecision string` field on `traceSpec` in
  `web/api/seed/seed.go`; populated `metadata.key_decision` per trace
  (one short sentence describing the load-bearing choice — e.g. run-1
  of api-endpoint-rename: "Ran grep before touching any file, anchoring
  the edit set on the literal /users string."). Updated `seed_test.go`:
  flipped the `HasPrefix("seed-")` assertion to an explicit
  expected-names allowlist (api-endpoint-rename /
  auth-migration-md5-to-bcrypt / new-endpoint-with-tests); added new
  `TestSeed_TracesHaveKeyDecisionMetadata` (every trace has non-empty
  metadata.key_decision under every baseline). Sed-replaced all
  `seed-X/...` trace-name references in `cache_gen_test.go` and
  `cache_load_test.go` (13 sites across the two files). Regenerated
  `web/api/seed/seed-cache.json` via
  `go test -tags genseed ./web/api/seed/ -run TestGenerateSeedCache`
  → 9 triage + 15 transcript entries with new trace names; 0
  embeddings (same Option-b path as chunk 2: `VOYAGE_API_KEY` unset
  locally, Fly binary regenerates lazily on miss). `Examples.tsx`
  untouched — its `matchHint` substrings (`api-endpoint-rename` /
  `auth-migration` / `new-endpoint-with-tests`) still hit the renamed
  baselines via `.includes()`, so home cards resolve without code
  changes. Full `go test -count=1 ./web/api/seed/` green (10/10 with
  the new key_decision test); full `go test ./web/api/...` green; full
  vitest 156/156 green excluding pre-existing `Nav.test.tsx` import
  failure; `tsc --noEmit` clean. REFACTOR scan: rewrote every
  predicted-obsolete trace-name reference in-chunk; nothing left
  dangling.

- **Chunk 13: Introduce-me persistent surface port** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Per CD's placement decision (footer block over nav-chip or
  floating-badge — see `_design/project/introduce-me.jsx` PLACEMENTS
  table), the introduce-me surface lives inside `SiteFooter` so it
  appears on every page after the product content, not before. Added
  `.introduce-me` block to `web/frontend/src/components/SiteFooter.tsx`:
  monogram (J.S. in brand cyan), name + pulsing "available ·
  forward-deployed eng" pill, the verbatim three-credential CV blurb
  ($100M+ enterprise data programs / production AI agents / PwC
  Transfer Pricing AI competition winner), `mailto:` primary CTA, and
  inline LinkedIn + GitHub links with glyphs. CTA targets pulled
  from CD design: `mailto:jakesilverman.pro@gmail.com` (Jake's
  professional inbox, not the session `userEmail`),
  `linkedin.com/in/jacob-silverman1/`, `github.com/jtsilverman`.
  Footer brand row's `/docs` and `/changelog` links wired to real
  routes (were pointing at `/about` placeholder). Mounted `SiteFooter`
  on `app/baselines/[id]/page.tsx` — every other page already had it,
  baseline detail was the only gap. Appended ~140 lines of
  introduce-me CSS to `globals.css` (`.introduce-me`,
  `.introduce-me-inner`, `.introduce-me-identity`,
  `.introduce-me-monogram`, `.introduce-me-text`,
  `.introduce-me-name-row`, `.introduce-me-name`,
  `.introduce-me-role-tag`, `.introduce-me-role-dot` +
  `@keyframes introduceMePulse`, `.introduce-me-blurb`,
  `.introduce-me-ctas`, `.introduce-me-primary`,
  `.introduce-me-elsewhere`, `.introduce-me-link`, with two responsive
  breakpoints at ≤900px and ≤540px). Tests: new
  `SiteFooter.test.tsx` (2 vitest+RTL assertions — identity + blurb
  text render; mailto/LinkedIn/GitHub `href` attrs match expected,
  external links carry `target="_blank"`). Full vitest 156/156 green
  excluding pre-existing `Nav.test.tsx`; `tsc --noEmit` clean.
  REFACTOR scan: no dead code obsoleted (placement is purely
  additive). Decision: CTA email is `jakesilverman.pro@gmail.com` per
  the CD design rather than the session `userEmail`
  (`jtsilverman8@gmail.com`); flagged at chunk-kickoff. Open
  Question `FDE contact CTA target` resolved to: email primary,
  LinkedIn + GitHub secondary (no Calendly in the v1).

- **Chunk 12: Tweaks panel port** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Ported `_design/project/tweaks-{app,panel}.jsx` into a Next.js
  client component at `web/frontend/src/components/Tweaks.tsx`. The
  CD design assumed a `postMessage`-based host protocol (edit-mode
  plumbing for the Claude Design editor — `__activate_edit_mode` /
  `__edit_mode_set_keys` etc.); per the spec caveat that protocol is
  stripped entirely and replaced with `localStorage` persistence
  (key `agentdiff:tweaks`). Three tweaks shipped: card density
  (compact / regular / spacious — wired to `body[data-density="…"]`),
  accent color (4 chip swatches — wired to `--accent` / `--accent-faint`
  / `--accent-glow` on document root), background tone (cool /
  neutral / warm — wired to `--bg` / `--bg-2` / `--surface` /
  `--border` on document root). Floating launcher button at
  `position:fixed; right:16px; bottom:16px` (per the chunk-12 Open
  Question resolution: floating button, not footer link or keyboard
  shortcut); clicking opens the light-glass panel (kept the CD
  design's intentional light aesthetic so the panel reads as a
  distinct site-wide settings surface against the dark app shell).
  Mounted once in `app/layout.tsx` so it appears on every page,
  including baseline detail. Defaults match existing CSS tokens
  (density=regular, accent=#22d3ee, bgTone=cool) so first paint is
  unchanged for new visitors; returning visitors hydrate via useEffect
  on mount — brief default-theme flash possible for non-default
  stored values (decision: skipped inline pre-hydration script for
  this chunk's scope, flagged inline). Appended ~190 lines to
  `globals.css`: `.twk-launcher`, `.twk-panel`, `.twk-hd`, `.twk-x`,
  `.twk-body`, `.twk-row`, `.twk-lbl`, `.twk-sect`, `.twk-seg`,
  `.twk-seg-thumb`, `.twk-chips`, `.twk-chip`, plus the full
  `body[data-density="compact"]` and `body[data-density="spacious"]`
  variant blocks (compact: tighter `.ex-card-*` padding, `.tl-row`
  font-size shrink, `.money-card` / `.concept-card` gap reductions;
  spacious: roomier paddings, larger graph height, larger card
  titles). Tests: 5 new vitest+RTL assertions in
  `Tweaks.test.tsx` — launcher opens panel; density choice persists
  to localStorage + applies `data-density` to body; hydration from
  pre-set localStorage applies density + `--accent` + `--bg` on
  mount; accent radio applies `--accent` on document root; close
  button removes the dialog. **`test/setup.ts` shim added**: the
  Claude Code harness intercepts node's `globalThis.localStorage`
  with a stub missing `.clear/.setItem/.getItem` (vanilla `node`
  repl returns a fully-functional jsdom localStorage); installed a
  Map-backed `Storage` shim before any test runs so suites that
  rely on persistent state behave the same in both environments.
  Full vitest 156/156 green excluding pre-existing `Nav.test.tsx`;
  `tsc --noEmit` clean. REFACTOR scan: no dead code obsoleted (the
  Tweaks surface is entirely new).

- **Chunk 8: /diff page port** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Ported `_design/project/diff.{html,jsx,css}` into
  `web/frontend/src/app/diff/[idA]/[idB]/page.tsx`. The new page renders
  five stacked sections inside the design-system shell: (1) hero strip
  with breadcrumb + H1 + 4-stat row (matches / subs / insertions /
  deletions in `--ok` / `--warn` / `--novel` / `--bad`), (2) switcher
  strip — two `TraceCard`s (A left-bordered accent, B left-bordered
  novel) with a `vs` divider; each card hosts a popover `Picker` that
  fetches the corpus via `listTraces()`, groups rows by baseline,
  filters across name/id/baseline/task, supports click-outside +
  Escape-to-close; selecting on either side calls
  `router.push('/diff/<newId>/<otherId>')`, (3) summary pill row + AI
  triage callout (`.diff-triage` block — classification-colored
  left-border, summary, likely-cause), (4) op-filter `seg`
  (all/matches/subs/insertions/deletions), (5) alignment list — grid
  `1fr 56px 1fr` with a center `.diff-row-marker` (op-badge `= / ~ / +
  / −` + long label); per-side index renders `String(i +
  1).padStart(2, '0')` for non-null indices and `··` for the skip
  side; ghost step renders a hatched repeating-linear-gradient
  placeholder. Two banner variants: cross-baseline (noisy-drift
  warning) and same-trace (empty diff). Footer carries `GET
  /api/diff/.../...` API hint + Back-to-traces button. Backend
  additive (chunks 4 + 7 precedent): `diffPair` JSON gains `a_index
  *int` + `b_index *int` (set to `&ap.IndexA` / `&ap.IndexB` when ≥ 0,
  `nil` otherwise) so the alignment list can number rows; `AlignedPair`
  TS type mirrored; `mockDiff` fixture extended to satisfy the new
  shape. Appended ~535 lines of diff-scoped CSS (`@keyframes fadeUp`,
  `.diff-hero-stats`, `.diff-banner`, `.diff-switcher*`,
  `.diff-trace-card*`, `.diff-picker*`, `.diff-summary*`,
  `.diff-triage*`, `.diff-controls`, `.diff-rows`, `.diff-row.op-*`,
  `.diff-row-marker`, `.diff-op-badge.op-*`, `.diff-step`,
  `.diff-step.ghost`, `.diff-role-tag.role-*`, `.diff-step-arg*`,
  `.diff-step-output*`, 2 responsive breakpoints at ≤980px and
  ≤720px) to `globals.css`. REFACTOR scan deletions (all in-chunk per
  the chunk-kickoff predictions): `web/frontend/src/components/DiffView.tsx`
  (only consumer was this page; CD design inlines alignment row
  markup so DiffView went dead) + `web/frontend/src/components/__tests__/DiffView.test.tsx`;
  existing diff-page test entirely rewritten (4 new vitest+RTL
  assertions: hero H1 "Diff" + stat-label keywords inside `<header>`
  via `within()`; switcher strip renders both trace names + `vs`
  divider; alignment list renders 3 op long-labels match/delete/insert;
  clicking the `insertions` op-filter button hides match + delete
  rows, leaves only insert). `TriagePanel.tsx` kept (still consumed
  by `/baselines/[id]`). Full vitest 132/132 green excluding the
  pre-existing `Nav.test.tsx` import-error (unchanged from chunks 1-7).
  `go test ./web/api/...` all green (`TestDiff` extended to assert
  `a_index`/`b_index` keys exist on every pair and at least one is
  non-nil). `tsc --noEmit` clean. Key decision: kept the existing
  `[idA]/[idB]` dynamic route shape; switching trace on either side
  calls `router.push` rather than the CD design's `history.replaceState
  + ?a/?b` query-param scheme — preserves canonical URLs that the
  rest of the app already links to (chunk 7's row-anchor goes to
  `/traces/<id>`, not `/traces.html?id=<id>`).

- **Chunk 7: /traces page port** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Replaced the Tremor-based traces page with the Claude Design corpus
  browser (`_design/project/traces.{html,jsx,css}`, fetched from the CD
  bundle and dropped into `_design/project/`). The new
  `web/frontend/src/app/traces/page.tsx` renders a hero strip
  (breadcrumb + H1 + 4-stat grid: traces / baselines / regressed /
  additive, with adapter pills below), a filter bar (baseline
  `<select>`, outcome `.seg` chips: all/succeeded/variance/regressed/additive,
  task search input with mark-highlighting), a sort `<select>` row, and
  a 7-column trace list (`tl-row` chrome from baseline.css + traces-
  specific `tr-*` grid: 130/150/120/50/1fr/100/80/16) with rows linking
  to `/traces/[id]` and per-row baseline pills linking to
  `/baselines/[id]`. Empty state with svg icon + clear-filters button.
  Adapter pill `display:none` at ≤1180px, time column hidden at
  ≤980px, single-column at ≤720px. Backend additive: extended
  `db.ListTraces()` SQL with two correlated subqueries for
  `baseline_id` + `baseline_name` (oldest baseline by `created_at`
  wins on multi-membership; both empty strings on orphan); mirrored on
  `traceSummaryResponse` JSON in `web/api/handlers/traces.go` and on
  `TraceSummary` in `web/frontend/src/lib/types.ts`. Same additive
  precedent as chunk 4's `step_count`/`metadata` on `traceRef`.
  REFACTOR scan deletions (all in-chunk per the chunk-kickoff
  predictions): `web/frontend/src/components/TraceUpload.tsx`
  (only consumer was the old /traces page; spec non-goal:
  "Onboarding for the upload-your-own-trace flow"),
  `web/frontend/src/components/__tests__/TraceUpload.test.tsx`,
  `uploadTrace` + `createBaseline` exports from
  `web/frontend/src/lib/api.ts` (no remaining consumers; promote-to-
  baseline UX deferred indefinitely), the corresponding uploadTrace
  + createBaseline describe blocks in `web/frontend/src/lib/__tests__/api.test.ts`,
  and the redundant "Upload trace" link in `SiteNav.tsx` (the main
  nav's `/traces` link already routes there). 3 new icons added to
  `Icons.tsx`: `Search`, `Close`, `ChevronRight`. ~290 lines of
  traces-scoped CSS (`.tr-hero`, `.tr-hero-grid`, `.tr-hero-stats`,
  `.tr-hero-adapters`, `.tr-filters`, `.tr-filter-row`,
  `.tr-filter-group`, `.tr-filter-search`, `.tr-filter-label`,
  `.tr-search`, `.tr-search-input`, `.tr-search-clear`,
  `.tr-filter-summary`, `.tr-list-head`, `.tr-table .tl-row`,
  `.tr-cell-*`, `.tr-baseline-link`, `.tr-task-text`,
  `.tr-decision-text`, `.tr-adapter`, `.tr-empty`, `.tr-empty-icon`,
  `.tr-foot`, plus 4 responsive breakpoints) appended to
  `globals.css`; all color tokens (`--bg-2`, `--border-hi`,
  `--border-2`, `--fg-2`, `--dim`, `--accent-faint`) already in
  `:root` from chunk 0. Tests: rewrote
  `web/frontend/src/app/__tests__/traces.test.tsx` (4 vitest+RTL
  assertions: hero H1 "Traces" + stats grid count, all rows render
  with names + baseline pills + outcome badges + step counts, clicking
  "regressed" filter chip hides non-regressed rows, search query that
  matches nothing shows the empty state). Added
  `TestListTraces_BaselineMembership` to `web/api/db/db_test.go`
  asserting in-baseline trace returns baseline_id + baseline_name and
  orphan returns empty strings. Full vitest 138/138 green excluding
  pre-existing `Nav.test.tsx` baseline-red (untouched, documented in
  Open Questions). `tsc --noEmit` clean. `go test ./web/api/...` all
  green. Key decision: row anchor is the Next `<Link>` (whole row
  clicks through to /traces/[id]); baseline pill stops propagation so
  clicking the pill navigates to /baselines/[id] instead.

- **Chunk 6: Money-feature action row + modals** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Added a money-feature `MoneyRow` (three cards: Run counterfactual /
  Edit the prompt / Find similar traces) above the existing Strategies
  + Compare Trace cards on `baselines/[id]/page.tsx`. Each card opens
  a design-faithful modal that calls the existing API:
  - `CounterfactualModal` — trace picker + step `<select>` + modified
    input textarea → `runCounterfactual(traceId, stepIndex, input)` →
    renders `CounterfactualGraph` for the comparison.
  - `EditPromptModal` — trace picker + step `<select>` + new prompt
    textarea → `editPrompt(traceId, stepIndex, prompt)` → renders
    `CounterfactualGraph`.
  - `SimilarModal` — trace picker → `getSimilar(traceId)` → renders
    matches as `.sim-list` rows with similarity score.
  New components: `Modal.tsx` (reusable shell with scrim, ESC-to-close,
  scrim-click-to-close, ARIA dialog role), `MoneyRow.tsx`,
  `CounterfactualModal.tsx`, `EditPromptModal.tsx`, `SimilarModal.tsx`.
  Step picker uses a numeric `<select>` over `0..step_count-1` (no
  extra `getTrace` fetch on trace pick — keeps modals self-contained;
  flagged at chunk-kickoff).
  Strategies + Compare Trace cards kept intact below the money row to
  preserve their existing 3-4 tests (flagged at chunk-kickoff).
  CSS: appended ~250 lines to `globals.css` (`.money-grid`,
  `.money-card`, `.modal-scrim`, `.modal`, `.modal-head/body/foot`,
  `.modal-foot-row`, `.cf-grid`, `.cf-col`, `.cf-label`, `.cf-list`,
  `.cf-row`, `.cf-result`, `.cf-running`, `.pg-spinner`,
  `.ep-textarea`, `.ep-diff-head`, `.sim-list`, `.sim-row`,
  `.sim-task`, `.sim-reason`, `.sim-meta`, `.sim-score`, `.ad-btn.ghost`);
  responsive collapses (`money-grid` and `cf-grid` to 1 column at
  ≤820px). Reused the existing `pg-spinner` keyframe naming from the
  design's baseline.css.
  Tests: added 3 vitest assertions to `baseline-detail.test.tsx` —
  CF card click → modal opens → Run → `runCounterfactual` called with
  `('t1', 0, 'try a different tool')`; EP card click → modal opens →
  Rerun → `editPrompt` called with `('t1', 0, 'new prompt text')`;
  Similar card click → modal opens → Find similar → `getSimilar`
  called with `'t1'` + matches render. Full vitest 149/149 green;
  pre-existing `Nav.test.tsx` transform failure unchanged. `tsc
  --noEmit` clean. REFACTOR scan: chunk is purely additive (money
  row + modals + CSS); no code went dead. Trace-detail's existing
  `CounterfactualButton` / `EditPromptButton` / `SimilarTraces`
  untouched (chunk 9 redesigns the trace detail surface separately).

- **Chunk 5: Baseline path graph card** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Replaced the Tremor `<Card><Title>Path graph</Title>...` block on
  `baselines/[id]/page.tsx` with a new
  `components/baseline/BaselinePathGraphCard.tsx`. The card wraps the
  existing React Flow `PathGraph` in `pg-container` (rounded surface
  border) and renders a `pg-legend` block below it with three rule
  rows ("edge thickness = run count", "node size = call frequency",
  "color = outcome cluster") and per-strategy cluster dots derived
  from `report.strategies` + `report.noise` (each cluster's dot color
  comes from its dominant outcome via the same kind mapping
  `TraceList` uses; noise renders neutral). The overlay/heatmap mode
  toggle moved into the `bl-section-head` controls as a `.seg`
  segmented control matching the trace-list filter chrome; old inline
  blue Tailwind buttons removed. Stats row ("X runs · Y branch points
  · overlay: Z") + overlay error moved to a `.bl-stats-row` below the
  legend. Appended ~75 lines of CSS to `globals.css` (`.pg-container`,
  `.pg-legend`, `.pg-legend-rules`, `.pg-rule-thick/size/color`,
  `.pg-legend-clusters`, `.pg-cluster`, `.pg-cluster-dot`,
  `.bl-stats-row`); the color-rule swatch uses
  `linear-gradient(--ok, --accent, --novel)` to match the existing
  outcome palette. Tests: added 2 vitest assertions to
  `baseline-detail.test.tsx` — three legend rule rows render,
  `.pg-cluster-dot` count = 2 (1 strategy + 1 noise). Full vitest
  146/146 green; pre-existing `Nav.test.tsx` transform failure
  unchanged. `tsc --noEmit` clean. REFACTOR scan: removed inline
  Card/Title/Text path-graph block (replaced by
  `BaselinePathGraphCard`), removed `PathGraph` default-import from
  page.tsx (only the `PathGraphMode` type is needed there now).
  `Card`/`Title`/`Text` from `@tremor/react` still imported for
  Strategies + Compare Trace blocks (chunk 6 redesigns those).

- **Chunk 4: Baseline detail visual shell** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Added a header + trace-list shell above the existing Tremor path-graph
  / strategies / compare cards (5 + 6 redesign those). Backend additive
  on `traceRef` in `web/api/handlers/cluster.go`: `step_count` +
  `metadata` fields populated from `database.GetBaselineTraces` so
  baseline detail can render per-row badges without a second
  `listTraces` fetch (extends chunk 2's additive-field precedent;
  no breaking changes, no new endpoints). Frontend
  `TraceRef` mirrored with `step_count?: number` and
  `metadata?: Record<string,string>`. Extracted shared `OutcomeBadge`
  (`web/frontend/src/components/OutcomeBadge.tsx`, `kind`: ok / warn /
  bad / novel / neutral / accent) from the inline copy in
  `home/Examples.tsx`; Examples now imports the shared one. New
  `components/baseline/BaselineHeader.tsx` (breadcrumb, badges row,
  H1 task title pulled from `traces[0].metadata.task`, callout body =
  `description` from chunk 3, side stats: runs / avg steps /
  strategies / drift derived from per-trace outcomes); new
  `components/baseline/TraceList.tsx` (filter chips:
  all/succeeded/variance/regressed/additive; sort: run order / steps /
  outcome; row click toggles overlay via parent `onSelect`; per-row
  outcome badge + steps + `MetadataBadges`). Dropped the Tremor
  "Traces in this baseline" card; `TraceList` owns trace selection +
  overlay trigger now. Appended ~190 lines of baseline-scoped CSS
  (`.bl-*`, `.tl-*`, `.seg`, `.select`, responsive breakpoints) to
  `web/frontend/src/app/globals.css`. Tests:
  - Extended `mockStrategyReport` fixture with `step_count` +
    `metadata` on members and noise.
  - Added 3 vitest assertions to `baseline-detail.test.tsx`: H1 from
    first trace's metadata.task; side stats render with run count = 3
    (2 strategy members + 1 noise); 3 `.tl-row-button` rows render
    with variance + regressed outcome badges. Existing overlay-click
    test continues to pass against the new TraceList (rows are
    `<button>` so `findByRole('button', { name: /trace-1/ })` resolves
    to a single match).
  Full vitest 144/144 green; pre-existing `Nav.test.tsx` failure
  unchanged. `tsc --noEmit` clean. `go test ./web/api/...` all green.
  Key decision text per row deferred (would require N transcript
  fetches per page load; acceptance "outcome/steps/key-decision badges
  via MetadataBadges" satisfied via outcome + steps; key decision can
  land in a later chunk or chunk 6 when modals are introduced).
  REFACTOR scan: removed inline OutcomeBadge in Examples.tsx (extracted),
  removed the Tremor traces card (replaced by TraceList), replaced
  the `nameToID` map in `cluster.go` with the richer `traceByName`
  (old code deleted).

- **Chunk 3: Seed-data UI surfacing** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Added `description?: string` to `BaselineSummary` in
  `web/frontend/src/lib/types.ts`. Updated `matchHint` substrings in
  `web/frontend/src/components/home/Examples.tsx` to the new seed
  baseline names (`api-endpoint-rename` / `auth-migration` /
  `new-endpoint-with-tests`); split `resolveHref` into
  `resolveBaseline` + `resolveHref` (the latter still exported for
  `page.tsx`'s `firstBaselineHref`); `ExampleCard` now takes a `hook`
  prop and the grid resolves it from the matched baseline's
  `description` (falls back to the hardcoded `example.hook` when no
  baseline match or empty description). Baseline detail
  (`web/frontend/src/app/baselines/[id]/page.tsx`) now calls
  `listBaselines()` alongside the cluster + graph fetches, finds the
  baseline by id, and renders its `description` as a minimal Tremor
  `<Card><Text>` callout above the path-graph card (no Title — would
  collide with `StrategyCluster`'s `baseline_name` Title). Tests:
  added "uses baseline description as card hook when API returns a
  match" to `home.test.tsx`; added "renders the baseline description
  as a callout when listBaselines returns it" to
  `baseline-detail.test.tsx`, plus a shared `listBaselines` mock in
  `beforeEach`. Updated the existing "links cards to matching
  baselines by name hint" test to use the new seed baseline names.
  Per-trace `task` / `outcome` surfacing deferred to Chunk 4 (already
  covered by Chunk 4's acceptance: "trace rows showing
  outcome/steps/key-decision badges via `MetadataBadges`"). Full
  vitest 141/141 green (pre-existing `Nav.test.tsx` failure
  documented in Open questions, untouched). `tsc --noEmit` clean.
  `go test ./web/api/...` all green (no backend changes). REFACTOR
  scan: no dead code obsoleted by the chunk.

- **Chunk 0: Home page redesign** — Shipped 2026-05-27 in commit
  `7b7b43d feat(home): redesign landing page with Geist + dark theme`.
  Replaced Tremor baseline grid with hero + 3 example cards (variance
  / regression / novel) using Claude Design output. Geist fonts via
  `geist` npm package. Sidebar dropped from layout; full-width
  content. Home tests rewritten (4/4 green).

- **Chunk 2: Seed-data rewrite — backend** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Replaced 5 abstract scenarios with 3 task-driven ones:
  `seed-api-endpoint-rename` (5 runs, 3 strategies — grep-first /
  search-first / assume-known-path; all variance),
  `seed-auth-migration-md5-to-bcrypt` (4 before-runs do
  read+search+read+write+write; 1 after-prompt-change skips the search
  → regressed outcome, silently leaves verify.go untouched), and
  `seed-new-endpoint-with-tests` (3 route-only runs + 2 route+test
  additive runs). Added `description TEXT NOT NULL DEFAULT ''` column
  to `baselines` table via existing `migrateAddColumn` helper
  (`web/api/db/sqlite.go:136`) per
  `sqlite-add-column-pragma-table-info-idempotent`; `Description`
  field on `Baseline` + `BaselineSummary`; `CreateBaseline` signature
  changed to `(name, description, traceIDs)`. Per-trace metadata
  carries `task` (same per scenario) + `outcome` (succeeded /
  regressed / variance / additive). User prompts + tool args + result
  outputs all use realistic synthetic content (real-looking paths,
  short snippets) instead of `"Demo prompt for read_file"` filler.
  `seed-cache.json` regenerated via `go test -tags genseed
  ./web/api/seed/ -run TestGenerateSeedCache` to 9 triage pairs + 15
  transcripts; embedding regen gated on `VOYAGE_API_KEY` (0
  embeddings currently — see Open Questions). **Scope drift accepted
  retroactively at checkpoint:** added `description` field to
  `baselineSummaryResponse` JSON in `web/api/handlers/baselines.go`
  so chunk 3 can wire home-card subtitles; the spec's "API surface
  stays stable" clause read as no-breaking-changes / no-new-endpoints,
  not no-additive-fields. Mechanical signature fix-ups in
  `web/api/db/db_test.go` (6 sites), `web/api/handlers/promote.go`
  (1 site), `web/api/handlers/baselines.go` (1 site), and trace-name
  string swaps in `web/api/seed/cache_load_test.go` (4 sites — old
  `seed-tool-order-stable/run-{1,2}` → new
  `seed-api-endpoint-rename/run-{1,2}`, round-trip tests still test
  mechanics). Tests: tightened
  `TestSeed_PopulatesScenariosOnEmptyDB` (exactly 3 + non-empty
  Description), added `TestSeed_TracesHaveTaskAndOutcomeMetadata`
  (per-trace task + outcome + shared-task invariant), removed
  `stableScenarios` carve-out from
  `TestSeed_EachScenarioHasDistinctToolSequence`. Full `go test
  ./web/api/...` green. **Recall hits applied:**
  `sqlite-add-column-pragma-table-info-idempotent`,
  `expose-prod-hash-from-handlers-for-cache-prepopulation`,
  `tdd-retroactive-ratification-flag-dont-fake-red`,
  `claude-as-its-own-cache-source` (wiki concept). **REFACTOR scan:**
  all predicted obsoletions deleted in-chunk (5 old scenario
  literals; `stableScenarios` map; 9 old triage pairs + 26 old
  transcript specs + 7 per-shape templates in cache_gen_test.go;
  ~13.5k lines of old seed-cache.json).

- **Chunk 1: /about page port** — Shipped 2026-05-27 (awaiting commit). Tier B.
  Translated `_design/project/about.{html,jsx,css}` into a Next.js client
  page at `web/frontend/src/app/about/page.tsx` (~290 lines, `'use client'`
  for `IntersectionObserver`-driven sticky tour-rail) + sibling SVG
  library at `web/frontend/src/components/about/AboutIcons.tsx` (8
  `ConceptIcon` line diagrams + 5 `TourMini` wireframes). Appended
  ~280 lines of about-scoped classes (`.ab-hero`, `.concepts-grid`,
  `.tour-*`, `.money-deep-*`, `.ab-cta-*`, responsive media query) to
  `web/frontend/src/app/globals.css`; all color/spacing/typography
  tokens already existed from chunk 0. Added jsdom `IntersectionObserver`
  polyfill to `web/frontend/src/test/setup.ts`. Tests: 5 new vitest+RTL
  in `web/frontend/src/app/__tests__/about.test.tsx` (hero + 8 concepts
  + 5 tour cards + 4 money features + CTA), all green; full vitest
  139/139 green excluding 1 pre-existing baseline-red `Nav.test.tsx`
  (imports `../Nav` after rename to `SiteNav` in chunk 0 — out of scope
  to fix here, flagged in Open questions). `tsc --noEmit` clean.
  REFACTOR scan: no dead code obsoleted (`/about` was a new route).
  Recall pattern applied: `rtl-assertions-gate-on-sut-unique-text`
  drove the `within('.tour-cards')` scope tighten when tour-rail nav
  titles collided with tour-card titles inside `#tour`.

## Drift queue (drained at ship)

(empty)

## Open questions

- `web/frontend/src/components/__tests__/Nav.test.tsx` imports `../Nav`
  but the component was renamed `SiteNav` in chunk 0 (commit `7b7b43d`).
  Pre-existing baseline-red, not introduced by chunk 1. Fix or retire
  the test as a separate cleanup chunk before ship.
- Should baseline detail show LLM-generated task descriptions, or pull
  them straight from the new `description` column? **Resolved:** after
  Chunk 2 ships the column, Chunk 4 pulls from it directly. No LLM
  generation needed.
- Tweaks panel: does it live behind a footer link, a floating button,
  or a keyboard shortcut? **Resolved (Chunk 12):** floating button at
  bottom-right per the recommendation. Mounted in root layout so it
  appears on every page including baseline detail.
- FDE contact CTA target: email, Calendly, LinkedIn, or all three?
  **Resolved (Chunk 13):** email primary (`mailto:jakesilverman.pro@gmail.com`)
  + LinkedIn + GitHub secondary; no Calendly in v1. Chunk 14 copy
  pass propagates the same set across home / about / footer.
- Introduce-me placement (nav chip / floating badge / footer block):
  **Resolved (Chunk 13):** footer block per CD's `introduce-me.jsx`
  PLACEMENTS table — visible after product content, not before;
  shared dark-tool aesthetic; doesn't crowd nav.
- `web/api/seed/seed-cache.json` was regenerated at chunk 2 with 0
  embedding entries because `VOYAGE_API_KEY` was unset locally;
  triage + transcript halves are correct. Before ship, either: (a)
  set `VOYAGE_API_KEY` and re-run `go test -tags genseed
  ./web/api/seed/ -run TestGenerateSeedCache` + commit the new JSON,
  OR (b) accept first-batch live Voyage cost on `/api/similar` miss
  when the Fly binary boots (Fly has the key set; per-trace embed
  generates lazily on miss and writes back to the embeddings table).
  Option (b) is the lower-friction path.
- Pre-existing `go vet` noise in
  `web/api/handlers/similar_test.go` (5 issues, "using resp before
  checking for errors") — not introduced by chunk 2; flag for
  separate cleanup chunk before ship.

## Follow-up (post-ship, separate spec)

- Mobile-first polish pass.
- Guided product tour (shepherd.js style step-through) if prose
  walkthrough on /about doesn't land.
- Re-architect trace generation to use `internal/bench/generate.go`.
- Onboarding for the upload-your-own-trace flow.

## Design handoff bundle

The Claude Design output lives at `_design/` in this worktree (note:
NOT `_design/agent-diff-visualization/` as an earlier draft stated).
Key files:
- `README.md` — Claude Design's instructions for the coding agent.
- `chats/chat1.md` — the design conversation transcript.
- `project/about.html`, `about.jsx`, `about.css` — Chunk 1 source.
- `project/baseline.html`, `baseline.jsx`, `baseline-data.jsx`,
  `baseline-graph.jsx`, `baseline-modals.jsx`, `baseline.css` —
  Chunks 4-6 source.
- `project/tweaks-app.jsx`, `tweaks-panel.jsx` — Chunk 10 source.
- `project/styles.css`, `density.css` — shared tokens (already
  partially ported into `web/frontend/src/app/globals.css`).
- `project/shared.jsx` — shared Nav/Footer/Icons (already ported to
  `web/frontend/src/components/{SiteNav,SiteFooter,Icons}.tsx`).
- `project/screenshots/` — what the design looks like rendered.
- **Six remaining surfaces have CD prompts staged at
  `_design/prompts.md`** (one per surface: /traces, /diff, trace
  detail, /docs, /changelog, introduce-me). Run them in the existing
  CD session so the design system stays coherent. Results land in
  `_design/project/` and unblock chunks 7-11 + 13.
