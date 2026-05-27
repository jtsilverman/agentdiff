# Claude Design prompts — agentdiff expansion

Six prompts for the six remaining surfaces. Run them in the **existing**
Claude Design session (the one that produced home/about/baseline/tweaks)
so the design system stays coherent. Paste results back into the
worktree's `_design/project/` folder; the spec's port chunks (chunks
7-13 in `specs/current.md`) will pick them up.

Order is independent — you can run all six in parallel or one at a
time. The Next.js port chunks fire as results arrive.

---

## Prompt 1 — /traces page (overview of all traces across baselines)

> Same project, same design system. Add a new page: `/traces` — the
> "everything" view, a single scannable list of every trace across
> every baseline.
>
> **Purpose:** an FDE-positioned visitor lands here when they want to
> browse the corpus instead of starting from a baseline. They scan for
> interesting traces, filter by outcome or task, click into individual
> trace details.
>
> **Data shape (from the existing Go API):** array of objects with
> `{trace_id, baseline_id, baseline_name, prompt, tool_sequence (array
> of step names), metadata: {task, outcome, key_decision?, ...}}`.
> Outcomes are one of `succeeded | regressed | variance | additive`.
>
> **Components you can reuse (already in `_design/project/uploads/`):**
> `MetadataBadges.tsx` (key-value badge row), `StepList.tsx` (renders
> a tool sequence), `DriftBadge.tsx`. Don't redesign these.
>
> **What to design:** the page shell + the trace-list treatment.
> Filter chips at the top (by baseline / by outcome / by task). Each
> trace row should be scannable in one beat — show baseline name,
> outcome badge, tool sequence (compressed), task summary. Density
> matches the existing baseline page.
>
> **Don't:** redesign nav or footer (use the existing `shared.jsx`).
> Don't reintroduce Tremor. Keep the dark theme + Geist fonts + the
> tokens already in `styles.css`.
>
> **Deliver:** `traces.html`, `traces.jsx`, `traces.css` in the same
> `project/` folder as the existing pages. Include 2-3 screenshots of
> the page populated with realistic synthetic data showing the variety
> of outcomes.

---

## Prompt 2 — /diff page (side-by-side trace compare)

> Same project, same design system. Add a new page: `/diff` — pick two
> traces, see where they diverge.
>
> **Purpose:** the visitor has spotted two interesting traces (either
> from `/traces` or from a baseline page) and wants to see them
> side-by-side. The page emphasizes divergence points: different tool
> called at step N, different output content, different outcome.
>
> **Data shape:** two trace objects (same structure as in Prompt 1
> above, plus a full `transcript` field — array of `{role,
> tool_calls[], content}` objects). The two traces may or may not
> share a baseline.
>
> **Components you can reuse:** `Transcript.tsx` (full message view),
> `StepList.tsx`, `MetadataBadges.tsx`. Don't redesign them.
>
> **What to design:** the side-by-side layout. Two columns of equal
> width on desktop; emphasize divergence visually (highlighted rows
> where the two traces differ, muted rows where they match). A header
> strip showing both trace IDs + baseline + outcome. A switcher at the
> top to swap either side.
>
> **Don't:** redesign nav or footer; keep dark theme + Geist + existing
> tokens. Don't introduce a new diff library — use plain HTML/CSS
> highlighting.
>
> **Deliver:** `diff.html`, `diff.jsx`, `diff.css`. Include 2
> screenshots: one with two traces sharing a baseline (subtle
> divergence), one with two traces from different baselines (heavy
> divergence).

---

## Prompt 3 — Trace detail page (individual trace deep-dive)

> Same project, same design system. Redesign the individual trace view
> at `/traces/[id]` — currently on old Tremor visuals, needs full
> dark-theme treatment.
>
> **Purpose:** the visitor clicked into one specific trace from `/traces`
> or a baseline page. They want the full story: every tool call, every
> message, every metadata field, in one continuous deep-dive view.
> Think "stack trace viewer for AI agents."
>
> **Data shape:** one trace object with `{trace_id, baseline_id,
> prompt, tool_sequence, transcript (full message array), metadata
> (task, outcome, key_decision, all custom keys)}`.
>
> **Components you can reuse:** `Transcript.tsx`, `StepList.tsx`,
> `MetadataBadges.tsx`. These already work, don't redesign.
>
> **What to design:** the page shell + how the three components stack.
> A header section with task description + outcome badge + key metadata.
> A step-list overview (compressed, clickable to jump). The full
> transcript below, with anchored sections for each tool call. Sticky
> step-list as the visitor scrolls the transcript would be excellent.
>
> **Don't:** redesign nav/footer/dark theme; don't reintroduce Tremor.
>
> **Deliver:** `trace-detail.html`, `trace-detail.jsx`,
> `trace-detail.css`. Include 2 screenshots — one short trace, one
> long with sticky-step-list demonstrating the scroll behavior.

---

## Prompt 4 — /docs page (developer reference)

> Same project, same design system. Add a new page: `/docs` — the
> developer reference. Distinct from `/about` (which is the visitor
> tour); this page is the manual.
>
> **Purpose:** an engineer landed on the site, decided to use it or
> integrate with it, needs reference material. Concept definitions,
> API contract, how data flows through the system, how to interpret
> each visualization.
>
> **Sections to cover:**
> 1. **Concept glossary** — trace, baseline, strategy, drift, path
>    graph, overlay, counterfactual, edit-prompt, similarity. One
>    paragraph + one inline example each.
> 2. **API reference** — the Go backend endpoints (`GET /api/baselines`,
>    `GET /api/baselines/:id`, `GET /api/clusters/:id`, `GET
>    /api/graphs/:id`, `POST /api/counterfactual`, `POST
>    /api/edit-prompt`, `GET /api/similar`). Each with method, path,
>    request shape, response shape, one curl example.
> 3. **Data flow** — your traces → embeddings → clustering →
>    visualization. Include a small mermaid or ASCII diagram.
> 4. **Embed cache mechanism** — `seed-cache.json` is `//go:embed`-ed
>    into the binary at build time. Pre-computed LLM/embedding results
>    ship with the binary so the demo has zero cold-start cost. Mention
>    this — it's an interesting technical choice that signals depth.
> 5. **Interpretation guide** — what each visualization actually means.
>    Path graph: edge thickness = run count, node size = step count.
>    Strategy cluster: traces grouped by tool-sequence similarity.
>    Drift badge: signal that an "after" trace diverged from a stable
>    pattern.
>
> **Tone:** technical, scannable, jump-linkable. Each section heading
> should be a sticky anchor. Code blocks for API examples (use Geist
> Mono). No marketing copy — this is the manual, not the pitch.
>
> **Don't:** redesign nav/footer. Don't reintroduce Tremor. No API
> calls — this is pure prose.
>
> **Deliver:** `docs.html`, `docs.jsx`, `docs.css`. Screenshots: 2-3
> showing different sections, including one with the sticky-anchor
> sidebar visible and one zoomed on a code block.

---

## Prompt 5 — /changelog page (shipped features timeline)

> Same project, same design system. Add a new page: `/changelog` — a
> visible log of what's shipped, when, and why it matters.
>
> **Purpose:** signals momentum + technical depth to a visitor.
> Doubles as a "what's new" surface for return visitors. FDE
> positioning: showing a track record of shipped work is part of the
> pitch.
>
> **Format:** reverse-chronological list of entries. Each entry has:
> date, short title, 1-2 sentence description, optional tag (feature
> / fix / docs / infra). Group entries under month headings.
>
> **Example content (to populate the page):**
> - 2026-05-27 · **Site redesign** · Full visual rebuild on Geist +
>   dark theme; new /about, /docs, /traces, /diff pages; live tweaks
>   panel. (feature)
> - 2026-05-27 · **Task-driven seed scenarios** · Replaced 5 abstract
>   baselines with 3 realistic scenarios carrying task descriptions
>   and per-trace outcome labels. (feature)
> - (earlier entries — make them up plausibly based on what a hosted
>   AI-trace-analysis demo would have shipped over the prior weeks:
>   counterfactual replay, edit-prompt rewrites, embeddings similarity,
>   path graph heatmaps, AI triage, Fly+Vercel hosted deploy)
>
> **Tone:** factual, dated, no marketing puffery. The "why it matters"
> sentence is the interesting part — what changed for the user.
>
> **Don't:** redesign nav/footer. Don't reintroduce Tremor. Pure
> prose, no API.
>
> **Deliver:** `changelog.html`, `changelog.jsx`, `changelog.css`.
> Screenshots: 1-2 showing the timeline density and tag styling.

---

## Prompt 6 — Introduce-me persistent surface

> Same project, same design system. Design a persistent surface that
> introduces the maker (Jake Silverman) and signals his availability
> as a forward-deployed engineer. Visible on every page; not a
> separate route.
>
> **Purpose:** this whole site is Jake's FDE (forward-deployed
> engineer) portfolio piece. A visitor lands, browses, and at some
> point notices that there's a real person behind it who is available
> to do this kind of work for them. The surface needs to be present
> enough to register without competing with the page content.
>
> **Placement — your call from these three options (pick the best one
> and ship that; don't ship all three):**
> - A nav-resident chip in the top-right (photo + name + small CTA).
> - A floating bottom-corner badge that expands on hover into a small
>   bio card.
> - A footer block ("Built by Jake Silverman · for hire as FDE ·
>   contact") that's always visible at the bottom of every page.
>
> Pick whichever fits the design system best. Justify the choice in
> a 2-sentence note alongside the deliverable.
>
> **Content:** name (Jake Silverman), one-line role descriptor
> (something like "forward-deployed engineer — drop me into your
> systems, ship AI improvements"), one CTA (link target TBD by Jake at
> port time — could be email, Calendly, or LinkedIn).
>
> **Tone:** confident not braggy. Visible but not intrusive. Should
> read to a non-technical CEO as "this person is hireable" and to a
> technical CTO as "this person ships."
>
> **Don't:** make it a popup, a modal, or a banner that has to be
> dismissed. Don't overcrowd existing nav. Don't use stock photography.
>
> **Deliver:** `introduce-me.html`, `introduce-me.jsx`,
> `introduce-me.css` showing the chosen placement integrated with the
> existing nav/footer/page shell. Include the 2-sentence justification
> for the placement choice. Screenshots: the surface visible on home,
> on a baseline detail page, and on /about so we can see how it
> coexists with each page's content.
