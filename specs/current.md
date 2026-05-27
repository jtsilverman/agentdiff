# Spec: agentdiff-redesign

**Status:** Locked
**Created:** 2026-05-27
**Locked:** 2026-05-27
**Branch:** feat/hosted-deploy
**Worktree:** /Users/admin/Documents/projects/agentdiff/.worktrees/hosted-deploy

## Context

agentdiff hosted demo (https://agentdiff.vercel.app) is technically
complete but visually unfinished. A Claude Design handoff bundle was
produced (see `_design/` in this worktree). The home page has been
translated into the Next.js frontend; baseline detail and `/about` are
still on the old Tremor visuals.

Audience target (per Jake): the design must read to AI engineers *and*
the non-tech-native stakeholders they pitch to (Tenex forward-deployed
engineer shape). Hero language and example tasks are written for both —
"Add a 'Buy Now' button to the pricing page" / "Diagnose yesterday's
site outage" / "Build a weekly report on customer signups" instead of
flask-limiter / JWT migration / flaky CI test.

## Requirements

- Translate the Claude Design bundle's `baseline.html` design into the
  existing `/baselines/[id]` Next.js route.
- Create `/about` route from the bundle's `about.html` design.
- Keep all three pages visually coherent (same nav, footer, design
  tokens defined in `web/frontend/src/app/globals.css`).
- Cards on home (already shipped) deep-link to seeded baselines by name
  match (`variance`, `regression`, `novel`). Baseline detail must work
  for those targets.

## Constraints

- Next.js 14.2.29 App Router, React 18.3, Tailwind 3.4, Tremor 3.18.
- Geist + Geist Mono fonts via the `geist` npm package (NOT
  `next/font/google` — Geist isn't there in Next 14). Wired through
  `--font-geist-sans` / `--font-geist-mono` CSS vars in
  `web/frontend/src/app/globals.css`.
- The repo deploys to Vercel preview on push to `feat/hosted-deploy`.
  Backend (Go) is on Fly.io; frontend talks to it via
  `NEXT_PUBLIC_API_URL`. Don't touch the API surface unless required.
- The seeded baselines on the backend are still the old shape
  (`seed-tool-order-stable`, `seed-tool-order-variance`,
  `seed-prompt-regression`, `seed-novel-tool-discovery`,
  `seed-noise-and-strategies`). The home page links by name-substring
  match.

## Non-goals

- Rewriting the seeded backend scenarios. The plan file mentioned this
  (3 task-driven scenarios with realistic prose) but it's out of scope
  for this spec. Track as follow-up.
- The Tweaks panel (density / accent / background tone live toggles
  from the design bundle). Nice-to-have, not required.
- Mobile-first polish. Design is desktop-first; mobile should
  degrade gracefully via the existing media queries in `globals.css`.

## Interfaces / data model

Frontend routes touched:
- `web/frontend/src/app/baselines/[id]/page.tsx` — replace Tremor
  layout with the design's baseline page.
- `web/frontend/src/app/about/page.tsx` — new file (route doesn't exist
  yet).
- `web/frontend/src/app/traces/...` and `diff/...` — out of scope this
  spec but may need light touch-up to stop looking broken after the
  layout change (no sidebar anymore).

Existing API client (`web/frontend/src/lib/api.ts`) — no changes
expected. The baseline page already uses `getCluster`, `getGraph`,
`getOverlay`, `runCounterfactual`, `editPrompt`, `getSimilar`.

Existing components to reuse where the design allows:
- `MetadataBadges.tsx`, `StepList.tsx`, `Transcript.tsx`,
  `TriagePanel.tsx`, `PathGraph.tsx`, `CounterfactualGraph.tsx`,
  `StrategyCluster.tsx`, `DriftBadge.tsx`. (These are what got uploaded
  into Claude Design as `_design/.../uploads/`.)

## Acceptance criteria

- `/about` route exists and renders without errors.
- `/baselines/<id>` for each seeded baseline renders the new visual:
  task header + callout, trace list with outcome/step metadata, path
  graph card, three money-feature action buttons (counterfactual,
  edit-prompt, similar) wired to working modals or existing
  flows.
- Hero, nav, and footer match the design's typography, palette, and
  spacing across home, baseline, and about.
- `npx tsc --noEmit` clean. `npx vitest run` green (existing tests
  must not break; new tests for new components TDD-style).
- Vercel preview rebuilds clean and the three pages look right when
  inspected by eye.

## Test strategy

- Unit tests under `web/frontend/src/app/__tests__/` for the
  `/about` page (renders concept cards, sections) and the new
  `/baselines/[id]` page (renders task callout, links to overlay
  view, etc.).
- TypeScript + lint clean.
- Manual visual check via Vercel preview URL after each chunk's push.
  (Local dev hangs in Jake's shell due to Watchpack EINTR — push to
  Vercel for visuals.)

## Execution boundaries

- In scope: `web/frontend/src/app/baselines/[id]/page.tsx`,
  `web/frontend/src/app/about/page.tsx` (new), any new components
  under `web/frontend/src/components/baseline/` and
  `web/frontend/src/components/about/`, design tokens in
  `web/frontend/src/app/globals.css`.
- Touch lightly if visually broken: `web/frontend/src/app/traces/...`
  and `web/frontend/src/app/diff/...`.
- Out of bounds: anything under `web/api/` (Go backend), seed data,
  embed cache.

## Chunk decomposition

**Chunk 1: /about page** — translate `_design/agent-diff-visualization/project/about.{html,jsx,css}` into `web/frontend/src/app/about/page.tsx` and supporting components.
- Acceptance: route `/about` renders concept grid (8 concepts), page tour, money features, footer CTA; matches design typography/palette.
- Tier: B.
- Caveat: pure prose page, no API calls.

**Chunk 2: Baseline detail visual shell** — replace top of `baselines/[id]/page.tsx` with the design's task header + callout + trace list.
- Acceptance: task description as callout, trace rows showing outcome/steps/key-decision badges, no regressions on existing data fetching.
- Tier: B.
- Caveat: baseline data shape doesn't yet include task description or per-trace outcome labels (planned in non-goal followup). Use placeholder copy keyed off baseline name until the seed work ships.

**Chunk 3: Baseline path graph card** — port the design's path graph treatment + legend to the baseline detail page.
- Acceptance: path graph renders inside a styled card with the legend ("edge thickness = run count" etc.) below.
- Tier: B.

**Chunk 4: Money-feature action row + modals** — replace existing scattered buttons with the design's three-button row (counterfactual / edit-prompt / similar) and modal flows.
- Acceptance: each button opens a working modal that invokes the existing API (`runCounterfactual`, `editPrompt`, `getSimilar`); existing tests still pass.
- Tier: B.

**Chunk 5: Inner pages tidy** — light visual fixes for `/traces`, `/diff`, and trace detail so they don't look broken after the layout/sidebar change shipped on home.
- Acceptance: each page renders in a sensible container, uses the new nav, no obvious style regressions.
- Tier: C.

## Testing chunks

| Chunk | Scenario | Test type |
|---|---|---|
| 1 | `/about` route renders concept grid + sections | unit (vitest + RTL) |
| 2 | baseline detail shows task callout + outcome badges | unit |
| 3 | baseline path graph renders with legend present | unit |
| 4 | each money-feature button opens its modal | unit + manual |
| 5 | each inner page renders without overflow / broken style | manual via Vercel preview |

## Current chunk

Chunk 1 — `/about` page.

## Completed chunks

- **Chunk 0: Home page redesign** — Shipped 2026-05-27 in commit
  `7b7b43d feat(home): redesign landing page with Geist + dark theme`.
  Replaced Tremor baseline grid with hero + 3 example cards (variance
  / regression / novel) using Claude Design output. Geist fonts via
  `geist` npm package. Sidebar dropped from layout; full-width
  content. Home tests rewritten (4/4 green).

## Drift queue (drained at ship)

(empty)

## Open questions

- Should baseline detail show a *generated* task description per
  baseline (LLM-generated from the trace data), or wait for the
  follow-up seed work to provide one explicitly? Recommendation:
  hardcode by baseline name for now; defer LLM generation.
- The design's "Tweaks panel" (density toggle etc.) is not in scope
  but is visually present in the screenshots. Confirm we drop it
  entirely.

## Follow-up (post-ship, separate spec)

- Rewrite seeded backend scenarios (`web/api/seed/seed.go`) into the 3
  task-driven scenarios from the plan file
  (`~/.claude/plans/we-will-plan-here-parallel-sparrow.md`). Add
  `task` + `outcome` trace metadata, baseline `description` column,
  regenerate `seed-cache.json`.
- Mobile pass.
- Tweaks panel (live density / accent toggles).

## Design handoff bundle

The Claude Design output lives at `_design/agent-diff-visualization/`
in this worktree. Key files:
- `README.md` — Claude Design's instructions for the coding agent.
- `chats/chat1.md` — the design conversation transcript.
- `project/about.html`, `about.jsx`, `about.css` — Chunk 1 source.
- `project/baseline.html`, `baseline.jsx`, `baseline-data.jsx`,
  `baseline-graph.jsx`, `baseline-modals.jsx`, `baseline.css` —
  Chunks 2-4 source.
- `project/styles.css`, `density.css` — shared tokens (already
  partially ported into `web/frontend/src/app/globals.css`).
- `project/shared.jsx` — shared Nav/Footer/Icons (already ported to
  `web/frontend/src/components/{SiteNav,SiteFooter,Icons}.tsx`).
- `project/screenshots/` — what the design looks like rendered.
