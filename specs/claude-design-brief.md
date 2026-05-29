# Claude Design brief — agentdiff v2

## What we're building

**agentdiff** is a trace recorder + regression detector for AI agents. You give it your agent's runs (LangChain, LlamaIndex, raw JSON), it clusters them into "baselines" (what good looks like for a task), and catches regressions when new runs diverge — without writing brittle test assertions.

Hosted public demo with seeded data. No auth. Engineers install the CLI to send their own traces.

## Who this site is for

**Primary viewer:** hiring manager / CEO / FDE-curious person landing cold from a LinkedIn post or referral. Never heard of agentdiff. Their question: "what do I think of the person who built this?"

**Secondary viewer:** AI engineers evaluating whether the tool is worth installing.

The site is **portfolio-first**: the product IS the artifact on display; Jake is the author by association, not the headline. The viewer lands, sees the product working, wants to try it, and infers Jake's skill from the work.

## Features the site needs to show

- **Path graph** — visualize all of an agent's runs on a task as a network of tool calls. Nodes = tool calls, edges = transitions, thickness = run count. The signature viz.
- **Trace transcript with scrubber** — single-run replay. Step list on the left rail, transcript (context / reasoning / tool calls / output) on the right.
- **Diff view** — two traces side-by-side, step-aligned with insert/delete/substitute operations + an AI triage panel classifying the diff as regression / variance / additive.
- **Counterfactual replay** — re-run the agent from a chosen step with modified inputs; overlay original vs counterfactual paths.
- **Edit-prompt rewrite** — rewrite a prompt at a chosen step and re-run N times; see how the distribution of behavior shifts.
- **Similarity search** — find traces semantically similar to a given trace.
- **Promote-to-baseline** — turn a trace into a new single-trace baseline.
- **CLI + adapters** — `npm i -g agentdiff`, with LangChain / LlamaIndex / raw JSON adapters.
- **Drag-drop trace upload** in the browser.

## The goal

A portfolio-first showcase. Not a tool gallery, not a docs site. The viewer lands, sees the product working, wants to try it. They infer Jake's craft from the artifact — no "hire me" pitch needed.

## Visual reference: tenex.co

We want agentdiff to feel like **[tenex.co](https://www.tenex.co/)**. Specifically:

- **Near-black charcoal foundation** (~#1a1a1a) with white text. Minimal color saturation — no bright hues.
- **White CTAs with subtle arrow icons** (e.g. "Get started →"). No neon accents.
- **Luxurious whitespace.** Sections breathe; ample margins; each concept gets its own pocket of space.
- **Massive bold headlines** (48–72px) with refined body text underneath. Generous line-height.
- **Hero with motion.** Tenex uses an animated GIF (VR character). For agentdiff, the analogue is a looping animation of the product working — animated path graph, scrubber playback, or live diff.
- **Three-column feature cards** for breaking down distinct capabilities.
- **Testimonial-style quote blocks** (even without real testimonials yet, the format is useful for surfacing a key claim or one-liner).
- **FAQ accordion** as a clean way to surface concept Q&A without a wall of text.
- **Sans-serif throughout.** Modern, geometric, clean. Heavy weights for headlines, regular for body.
- **Tone:** technically confident yet approachable. "Startup speed meets enterprise gravitas." Human warmth, not enterprise stiffness.

The current site is dark but **cramped and tool-y**. Tenex.co is dark but **premium, minimal, and showcase-y.** That's the shift.

## Hard constraints

- **Portfolio showcase, not a tool gallery.** One killer flow beats six sibling feature pages.
- **Show the product, don't describe it.** The hero needs the product visibly working, not a schematic.
- **No modal-buried features.** If a feature is worth showing, it lives in the page flow with a clear label.
- **Single vocabulary.** Pick one term per concept; kill duplicates (e.g. "strategies cluster" vs "path graph" — pick one).
- **No 6-minute meandering tour.** Concepts explained crisply, in context.
- **Page count: aim for ~5.** Roughly: home + product explainer + demo + install/use + about-Jake. Final IA is your call — propose what works for the showcase shape.

## What to kill from the current site

- The **Tweaks panel** (density/accent/bgTone toggles in localStorage).
- The **"introduce-me" footer surface** — folds into the Jake/meta page.
- The **`/about` 6-minute tour** — keep the concept content, kill the meandering structure.
- The **auto-redirecting `/diff` landing**.
- The legacy **Strategies cluster view** alongside the path graph — pick one canonical viz.

## Source content

Three demo stories in plain English at `data/seed-scenarios.md`. Each is one realistic task an AI agent runs in production, run multiple times, where agentdiff reveals something the output alone wouldn't show:

- **Variance** — a FinTech code-review agent reviews the same security-relevant PR five times. Three different strategies all catch the same critical issue (missing CSRF check). Demonstrates: variation between runs can be normal, not regression.
- **Regression** — a SaaS support-triage agent classifies a duplicate-charge ticket. Four runs fetch the order history first and route to fraud-risk; one run after a prompt simplification skips the lookup and misclassifies as a billing question. Demonstrates: a silent regression caught by the missing step.
- **Additive** — a B2B docs-generation agent runs against a new service. Three runs write the API docs; two later runs (after a model upgrade) also query the internal wiki and link related pages. Demonstrates: new behavior that's strictly more helpful, not regressive.

The existing seeded scenarios in `seed-scenarios.go` (rename-URL, MD5-to-bcrypt, add-endpoint) are dev-trainee tasks and will be replaced during the port to match the three stories above. Design against the markdown, not the Go source.

## What success looks like

A cold viewer hands the URL to someone who has never seen the project. Within 5 minutes, without hunting through docs, they can:

- Explain in one sentence what agentdiff does.
- Locate install instructions.
- Find who built it.
- Name two distinct features of the product.

If any of those requires hunting, the IA isn't done.
