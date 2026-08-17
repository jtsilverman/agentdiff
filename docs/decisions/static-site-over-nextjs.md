# The dashboard is plain HTML served by the Go binary

**Picked:** `web/site/` is four static pages of HTML, CSS, and JS. `web/api/main.go` serves them at
`/` through `http.FileServer` behind a `-site` flag, with `/api/*` matched first by Chi specificity.
One binary, one port, no build step, no Node.

**Rejected:** the Next.js 14 app that used to live at `web/frontend/`, with Tailwind, Tremor, React
Flow, and dagre.

**Reason:** the site's job is to be seen. `specs/claude-design-brief.md` names the primary viewer as
someone landing cold from a link who has never heard of AgentDiff, and the site as portfolio-first.
A second runtime, a second deploy target, and an npm toolchain all sat between a visitor and the
product. The Next.js app was 46 components and roughly 23,000 lines, including a 3,716-line
`globals.css`.

**How it happened:** `894012b` added `web/site/` and the `-site` flag. `f323144` deleted
`web/frontend/` and dropped the npm prerequisites and the two-terminal dev flow from `README.md`,
`web/README.md`, and `web/Makefile`.

**What this constrains:**

- `web/site/` stays framework-free. The whole asset set is `data.js`, `demo.css`, `demo.js`,
  `glyphs.js`, `pathgraph.js`, `site.css`, and `site.js`. Adding a bundler reintroduces exactly what
  this decision removed.
- The path graph is hand-drawn in `pathgraph.js`. React Flow and dagre are gone.
- The dev flow is one command. `cd web && make dev`, or
  `cd web/api && go run . -port 8080 -site ../site`.
- `internal/render/png.go` draws the Action's path graph in pure Go with `image/draw` and
  `basicfont`. No cgo, no headless browser. The same no-extra-runtime rule holds on both surfaces.

**Left behind:** the root `README.md` still describes the Next.js stack and the Railway plus Vercel
deploy. `.gitignore` still carries a `web/frontend/` block. Both are stale, not load-bearing.

**What would reopen it:** a dashboard feature that plain JS cannot carry. Nothing on the roadmap
qualifies today.
