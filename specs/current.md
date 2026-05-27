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

**Chunk 2: Seed-data rewrite — backend** — rewrite `scenarios()` in `web/api/seed/seed.go` with 3 task-driven scenarios; add `description` column via `migrateAddColumn`; regenerate `seed-cache.json` via `cache_gen_test.go`.
- Acceptance: `go test ./web/api/...` green; `curl /api/baselines` returns 3 baselines each with `description`; each trace has `task` + `outcome` keys.
- Tier: B.
- Caveat: regenerating seed-cache.json is mandatory; skipping it means stale cache silently serves old LLM/embedding data.

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

**Chunk 8: /diff page port** — port CD's `diff.{html,jsx,css}` (Prompt 2) into `web/frontend/src/app/diff/page.tsx`. Two-trace side-by-side with divergence highlighting.
- Acceptance: route accepts two trace IDs, renders side-by-side compare, divergence rows visually emphasized, no Tremor leftovers.
- Tier: B.
- Caveat: blocks until `_design/project/diff.*` exists.

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

## Current chunk

Chunk 2 — Seed-data rewrite (backend).

## Completed chunks

- **Chunk 0: Home page redesign** — Shipped 2026-05-27 in commit
  `7b7b43d feat(home): redesign landing page with Geist + dark theme`.
  Replaced Tremor baseline grid with hero + 3 example cards (variance
  / regression / novel) using Claude Design output. Geist fonts via
  `geist` npm package. Sidebar dropped from layout; full-width
  content. Home tests rewritten (4/4 green).

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
  or a keyboard shortcut? **Recommendation:** floating button (matches
  the design's `tweaks-panel.jsx` posture). Confirm at Chunk 12
  kickoff.
- FDE contact CTA target: email, Calendly, LinkedIn, or all three?
  Need Jake's pick at Chunk 13 kickoff (introduce-me port). Chunk 14
  copy pass propagates the same choice across home / about / footer.
- Introduce-me placement (nav chip / floating badge / footer block):
  defer to Claude Design's recommendation (Prompt 6). Re-confirm at
  Chunk 13 kickoff against the CD output.

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
