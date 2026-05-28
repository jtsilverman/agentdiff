/* docs.jsx — developer reference page
   Distinct from /about (visitor tour). This is the manual: concept glossary,
   API contract, data flow, embed-cache mechanism, interpretation guide.
*/

const { useState: useStateDX, useEffect: useEffectDX, useRef: useRefDX, useMemo: useMemoDX } = React;

// ─── Glossary ───
const CONCEPTS = [
  { id: "trace", title: "Trace", body: "A recording of a single agent run on a task. Includes the prompt, every assistant chain-of-thought step, every tool call, every tool result, and the final answer. The atomic unit of analysis.", example: "t-7a4f — Run 1, 9 steps, succeeded" },
  { id: "baseline", title: "Baseline", body: "A group of traces all running the same task. The unit of comparison: you don't compare two single runs, you compare distributions of runs. A baseline gives you statistical power.", example: "baseline:rate-limit — 5 runs of \"add rate limiting to a Flask endpoint\"" },
  { id: "strategy", title: "Strategy", body: "A cluster of traces within a baseline that took the same tool-call shape. Computed by tokenizing each trace's tool sequence and clustering. The label is auto-generated from the most representative tool path.", example: "Strategy A · flask-limiter (3 of 5 runs) · Strategy B · hand-rolled middleware (1) · Strategy C · redis lua (1)" },
  { id: "drift", title: "Drift", body: "When a new trace lands outside the baseline's existing strategy clusters, or shifts the cluster distribution. Drift is signed: better (additive), worse (regression), or sideways (variance). Detected per-baseline.", example: "v1.3 prompt drift detected · cluster mass shifted 40% toward strategy C · 1 regression" },
  { id: "pathgraph", title: "Path graph", body: "A network visualization of every trace in a baseline, overlaid. Nodes are tool calls; edges are transitions between calls. Edge thickness is run count through that transition. Node size is call frequency.", example: "5 nodes, 7 edges · widest edge: search → install (4 runs)" },
  { id: "overlay", title: "Overlay", body: "A one-trace-vs-baseline view: layer a single run on the baseline's path graph, dim the rest. Used to see where one trace agreed with the herd and where it broke off.", example: "Overlay t-d57e on baseline:rate-limit → diverges at step 5" },
  { id: "counterfactual", title: "Counterfactual replay", body: "\"What if the agent had gone left at step 5?\" Pick a trace, pick a divergence step, force a different first decision, then re-run from that point with the same context. The result shows how the rest of the trace would have played out.", example: "POST /api/counterfactual { trace_id, fork_step, alternative }" },
  { id: "edit-prompt", title: "Edit-prompt rewrite", body: "\"What if the prompt had said X instead of Y?\" Rewrite the system or user prompt; the agent is re-run 5 times with the new prompt; the distribution shift is reported.", example: "POST /api/edit-prompt { baseline_id, new_prompt }" },
  { id: "similarity", title: "Similarity", body: "Vector similarity over trace embeddings. Used to surface baselines with similar shape, failure mode, or toolchain. Embeddings are pre-computed for known traces and cached.", example: "Top-k cosine over the cached embedding for the active trace" },
];

// ─── API endpoints ───
const ENDPOINTS = [
  {
    id: "list-baselines",
    method: "GET",
    path: "/api/baselines",
    purpose: "List every baseline in the corpus.",
    request: null,
    response: `[
  {
    "id": "rate-limit",
    "task": "Add rate limiting to a Flask endpoint",
    "runs": 5,
    "drift": "variance",
    "created_at": "2026-05-24T14:22Z"
  },
  ...
]`,
    curl: `curl https://agentdiff.dev/api/baselines`,
  },
  {
    id: "get-baseline",
    method: "GET",
    path: "/api/baselines/:id",
    purpose: "Fetch one baseline with its trace list, summary stats, and prompt.",
    request: null,
    response: `{
  "id": "rate-limit",
  "task": "Add rate limiting to a Flask endpoint",
  "prompt": "Add per-IP rate limiting (100 req/min)...",
  "traces": [{"id": "t-7a4f", "name": "Run 1", "outcome": "succeeded", ...}],
  "summary": {"strategies": 3, "avg_steps": 11.2}
}`,
    curl: `curl https://agentdiff.dev/api/baselines/rate-limit`,
  },
  {
    id: "get-clusters",
    method: "GET",
    path: "/api/clusters/:baseline_id",
    purpose: "Get the auto-clustered strategies for a baseline.",
    request: null,
    response: `{
  "baseline_id": "rate-limit",
  "clusters": [
    {"label": "flask-limiter",  "trace_ids": ["t-7a4f","t-9b30","t-411a"], "centroid_path": [...]},
    {"label": "hand-rolled",    "trace_ids": ["t-2c81"], "centroid_path": [...]},
    {"label": "redis lua",      "trace_ids": ["t-d57e"], "centroid_path": [...]}
  ]
}`,
    curl: `curl https://agentdiff.dev/api/clusters/rate-limit`,
  },
  {
    id: "get-graph",
    method: "GET",
    path: "/api/graphs/:baseline_id",
    purpose: "Path-graph nodes + edges for a baseline (used to render the network viz).",
    request: null,
    response: `{
  "nodes": [{"id": "read_file", "label": "read_file:app.py", "freq": 5}, ...],
  "edges": [{"from": "read_file", "to": "search", "count": 4, "color": "ok"}, ...]
}`,
    curl: `curl https://agentdiff.dev/api/graphs/rate-limit`,
  },
  {
    id: "post-counterfactual",
    method: "POST",
    path: "/api/counterfactual",
    purpose: "Replay a trace from a chosen divergence step with a forced alternative decision.",
    request: `{
  "trace_id": "t-7a4f",
  "fork_step": 4,
  "alternative": "use Redis directly"
}`,
    response: `{
  "result_trace_id": "cf-9e2a",
  "outcome": "succeeded",
  "delta_steps": 2,
  "path": ["read_file", "edit_file", "shell", "done"]
}`,
    curl: `curl -X POST https://agentdiff.dev/api/counterfactual \\
  -H "Content-Type: application/json" \\
  -d '{"trace_id":"t-7a4f","fork_step":4,"alternative":"use Redis directly"}'`,
  },
  {
    id: "post-edit-prompt",
    method: "POST",
    path: "/api/edit-prompt",
    purpose: "Rewrite a baseline's prompt and re-run 5 traces; returns the new distribution.",
    request: `{
  "baseline_id": "rate-limit",
  "new_prompt": "Add rate limiting using Redis."
}`,
    response: `{
  "new_baseline_id": "rate-limit-ep-2",
  "distribution": [
    {"strategy": "flask-limiter", "count": 1},
    {"strategy": "redis lua",     "count": 2},
    {"strategy": "hand-rolled",   "count": 2}
  ],
  "delta": "+1.4 avg steps, variance up"
}`,
    curl: `curl -X POST https://agentdiff.dev/api/edit-prompt \\
  -H "Content-Type: application/json" \\
  -d '{"baseline_id":"rate-limit","new_prompt":"..."}'`,
  },
  {
    id: "get-similar",
    method: "GET",
    path: "/api/similar/:trace_id?k=5",
    purpose: "Top-k baselines similar in shape, failure mode, or toolchain.",
    request: null,
    response: `[
  {"baseline_id": "api-key-rotation", "score": 0.94, "reason": "same shape — library-vs-handroll"},
  {"baseline_id": "cors-debug",       "score": 0.88, "reason": "same failure surface"},
  ...
]`,
    curl: `curl https://agentdiff.dev/api/similar/t-7a4f?k=5`,
  },
];

// ─── Interpretation guide ───
const INTERPRETS = [
  { title: "Path graph — edge thickness", body: "How many runs took that transition. A thick edge means consensus; a thin edge means a road less traveled.", chartHint: "thickness = run count" },
  { title: "Path graph — node size", body: "How often that tool got called across all runs in the baseline. A big node means the agent kept returning to it.", chartHint: "size = step count" },
  { title: "Path graph — node color", body: "Cluster membership. Same color = same strategy. Hover for the cluster label.", chartHint: "color = strategy" },
  { title: "Strategy cluster — label", body: "Auto-generated from the most representative tool path in the cluster. Labels are not human-curated; treat them as descriptive, not authoritative.", chartHint: "k-means on tool-sequence vectors" },
  { title: "Drift badge — variance", body: "Distribution shifted but no run regressed. Often expected when the prompt under-specifies a library choice.", chartHint: "color: warn" },
  { title: "Drift badge — regression", body: "A new run failed where the baseline succeeded. Worth opening immediately — bisect by prompt or model version.", chartHint: "color: bad" },
  { title: "Drift badge — additive", body: "A new run succeeded via a path no baseline run took. Possibly a better strategy worth promoting to canonical.", chartHint: "color: novel" },
];

// ─── TOC structure ───
const TOC = [
  { id: "concepts", label: "Concepts", num: "01" },
  { id: "api", label: "API reference", num: "02" },
  { id: "data-flow", label: "Data flow", num: "03" },
  { id: "embed-cache", label: "Embed cache", num: "04" },
  { id: "interpret", label: "Interpretation", num: "05" },
];

// ─── Components ───

const MethodPill = ({ method }) => {
  const cls = {
    GET: "ok",
    POST: "novel",
    PUT: "warn",
    DELETE: "bad",
  }[method] || "neutral";
  return <span className={"badge " + cls + " mono dx-method"}>{method}</span>;
};

const CodeBlock = ({ children, lang }) => (
  <div className="dx-code">
    {lang && <div className="dx-code-lang mono">{lang}</div>}
    <pre className="mono">{children}</pre>
  </div>
);

const EndpointCard = ({ ep }) => (
  <article id={ep.id} className="dx-endpoint">
    <header className="dx-endpoint-head">
      <MethodPill method={ep.method}/>
      <code className="mono dx-endpoint-path">{ep.path}</code>
      <a className="dx-anchor mono" href={"#" + ep.id} aria-label="anchor">#</a>
    </header>
    <p className="dx-endpoint-purpose">{ep.purpose}</p>

    {ep.request && (
      <div className="dx-endpoint-section">
        <span className="eyebrow">request body</span>
        <CodeBlock lang="json">{ep.request}</CodeBlock>
      </div>
    )}

    <div className="dx-endpoint-section">
      <span className="eyebrow">response</span>
      <CodeBlock lang="json">{ep.response}</CodeBlock>
    </div>

    <div className="dx-endpoint-section">
      <span className="eyebrow">example</span>
      <CodeBlock lang="bash">{ep.curl}</CodeBlock>
    </div>
  </article>
);

const ConceptItem = ({ c }) => (
  <article id={c.id} className="dx-concept">
    <header className="dx-concept-head">
      <h3>{c.title}</h3>
      <a className="dx-anchor mono" href={"#" + c.id} aria-label="anchor">#</a>
    </header>
    <p className="dx-concept-body">{c.body}</p>
    <div className="dx-concept-example mono">
      <span className="dx-example-tag">e.g.</span>
      <span>{c.example}</span>
    </div>
  </article>
);

// ─── Main ───

const Docs = () => {
  const [activeId, setActiveId] = useStateDX("concepts");
  const railRef = useRefDX(null);

  // Track which section is active via scroll position
  useEffectDX(() => {
    let lastScrollY = -1;
    let lastId = "";

    const compute = () => {
      const sy = window.scrollY;
      if (sy === lastScrollY) return;
      lastScrollY = sy;
      const sections = document.querySelectorAll("[data-section-id]");
      const threshold = 140;
      let current = "concepts";
      sections.forEach((el) => {
        const r = el.getBoundingClientRect();
        if (r.top - threshold <= 0) {
          current = el.getAttribute("data-section-id");
        }
      });
      if (current !== lastId) {
        lastId = current;
        setActiveId(current);
      }
    };

    const onScroll = () => compute();
    window.addEventListener("scroll", onScroll, { passive: true });

    const obs = new IntersectionObserver(() => compute(), {
      rootMargin: "0px 0px -50% 0px",
      threshold: [0, 0.5, 1],
    });
    document.querySelectorAll("[data-section-id]").forEach(el => obs.observe(el));

    const poll = setInterval(compute, 60);
    compute();

    return () => {
      window.removeEventListener("scroll", onScroll);
      obs.disconnect();
      clearInterval(poll);
    };
  }, []);

  const jumpTo = (id) => {
    const el = document.getElementById(id);
    if (el) {
      const top = el.getBoundingClientRect().top + window.scrollY - 80;
      window.scrollTo({ top, behavior: "smooth" });
    }
  };

  return (
    <>
      <Nav active="docs"/>

      <main>
        {/* Hero */}
        <header className="tr-hero dx-hero" data-screen-label="01 Docs hero">
          <div className="container">
            <div className="bl-breadcrumb mono">
              <a href="index.html" className="bl-crumb">home</a>
              <span className="dim">/</span>
              <span className="bl-crumb-current">docs</span>
            </div>
            <h1 className="tr-hero-title">Reference</h1>
            <p className="tr-hero-sub">
              The manual. Concept definitions, the Go backend API contract, how traces flow through the system, and how to read each visualization.
            </p>
            <div className="dx-hero-meta mono">
              <span className="badge accent">v0.4.1</span>
              <span className="dim">·</span>
              <span>OpenAPI · </span>
              <a href="#api">/api/openapi.json</a>
              <span className="dim">·</span>
              <span>last updated 2026-05-27</span>
            </div>
          </div>
        </header>

        <div className="container dx-body">
          {/* Sticky TOC */}
          <aside className="dx-rail">
            <div className="dx-rail-inner" ref={railRef}>
              <span className="eyebrow">on this page</span>
              <ol className="dx-toc">
                {TOC.map(t => (
                  <li key={t.id} className={activeId === t.id ? "active" : ""}>
                    <a href={"#" + t.id} onClick={(e) => { e.preventDefault(); jumpTo(t.id); }}>
                      <span className="dx-toc-num mono">{t.num}</span>
                      <span>{t.label}</span>
                    </a>
                  </li>
                ))}
              </ol>

              <div className="dx-rail-callout">
                <span className="eyebrow">looking for the tour?</span>
                <a href="about.html" className="btn ghost">
                  Go to /about <Icon.ArrowRight/>
                </a>
              </div>
            </div>
          </aside>

          {/* Main content */}
          <section className="dx-content">
            {/* ─── 01 CONCEPTS ─── */}
            <section id="concepts" data-section-id="concepts" className="dx-section">
              <header className="dx-section-head">
                <span className="eyebrow">01 · glossary</span>
                <h2>Concepts</h2>
                <p>Nine terms. Read these once; the rest of the docs assumes them.</p>
              </header>
              <div className="dx-concepts">
                {CONCEPTS.map(c => <ConceptItem key={c.id} c={c}/>)}
              </div>
            </section>

            {/* ─── 02 API ─── */}
            <section id="api" data-section-id="api" className="dx-section">
              <header className="dx-section-head">
                <span className="eyebrow">02 · http api</span>
                <h2>API reference</h2>
                <p>The Go backend exposes seven endpoints. All paths are relative to <code className="mono">/api</code>; all bodies are JSON.</p>
              </header>
              <div className="dx-endpoints">
                {ENDPOINTS.map(ep => <EndpointCard key={ep.id} ep={ep}/>)}
              </div>
            </section>

            {/* ─── 03 DATA FLOW ─── */}
            <section id="data-flow" data-section-id="data-flow" className="dx-section">
              <header className="dx-section-head">
                <span className="eyebrow">03 · pipeline</span>
                <h2>Data flow</h2>
                <p>How a recorded trace becomes a visualization. Four stages, all in-process.</p>
              </header>

              <div className="dx-flow-diagram mono">
                <pre>{`  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
  │   traces    │ ──▶ │  embeddings │ ──▶ │  clustering │ ──▶ │ visualization│
  │  .jsonl     │     │  (LLM)      │     │  (k-means)  │     │  (graph + UI)│
  └─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
        ▲                    ▲
        │                    │
  user upload         seed-cache.json
  (POST /api/         (//go:embed, cold-
   traces)            start = 0ms)`}</pre>
              </div>

              <ol className="dx-flow-steps">
                <li>
                  <span className="dx-flow-step-num mono">01</span>
                  <div>
                    <h4>Traces in</h4>
                    <p>Trace recordings (JSONL, one step per line) are written to the local store. Either uploaded via <code className="mono">POST /api/traces</code> or watched from a directory.</p>
                  </div>
                </li>
                <li>
                  <span className="dx-flow-step-num mono">02</span>
                  <div>
                    <h4>Embed</h4>
                    <p>Each trace's tool-call sequence is tokenized and embedded via an LLM call. Embeddings are cached in <code className="mono">embeddings/</code>; the binary also ships with <code className="mono">seed-cache.json</code> for the demo corpus.</p>
                  </div>
                </li>
                <li>
                  <span className="dx-flow-step-num mono">03</span>
                  <div>
                    <h4>Cluster</h4>
                    <p>Per baseline, k-means clusters trace embeddings into strategies. <code className="mono">k</code> is auto-tuned from silhouette score, capped at 5 for legibility.</p>
                  </div>
                </li>
                <li>
                  <span className="dx-flow-step-num mono">04</span>
                  <div>
                    <h4>Visualize</h4>
                    <p>Nodes + edges + cluster colors are joined into the path-graph payload returned by <code className="mono">GET /api/graphs/:id</code>. The UI is pure HTML/CSS/SVG — no chart library.</p>
                  </div>
                </li>
              </ol>
            </section>

            {/* ─── 04 EMBED CACHE ─── */}
            <section id="embed-cache" data-section-id="embed-cache" className="dx-section">
              <header className="dx-section-head">
                <span className="eyebrow">04 · build mechanism</span>
                <h2>Embed cache</h2>
                <p>How agentdiff ships with zero cold-start cost.</p>
              </header>

              <div className="dx-callout">
                <span className="eyebrow accent">why this matters</span>
                <p>Embedding a fresh corpus takes ~30–60s of LLM round trips. For the demo, we pre-compute embeddings and bake them into the binary at build time via <code className="mono">//go:embed</code>. The first user request hits an in-memory map; no network calls, no API keys required, no waiting.</p>
              </div>

              <CodeBlock lang="go">{`//go:embed seed-cache.json
var seedCacheBytes []byte

var seedCache map[string]Embedding

func init() {
    if err := json.Unmarshal(seedCacheBytes, &seedCache); err != nil {
        panic(fmt.Errorf("seed-cache.json malformed: %w", err))
    }
}

func GetEmbedding(traceID string) (Embedding, bool) {
    if e, ok := seedCache[traceID]; ok {
        return e, true            // hit · 0ms
    }
    return computeAndCache(traceID) // miss · ~200–600ms
}`}</CodeBlock>

              <div className="dx-callout">
                <span className="eyebrow">in production</span>
                <p>For workspaces with their own traces, <code className="mono">seed-cache.json</code> is regenerated nightly via a CI step (<code className="mono">go run ./cmd/seed-cache</code>) and committed to the binary. Live miss-path traces are cached in a side store and merged into the next seed build.</p>
              </div>
            </section>

            {/* ─── 05 INTERPRETATION ─── */}
            <section id="interpret" data-section-id="interpret" className="dx-section">
              <header className="dx-section-head">
                <span className="eyebrow">05 · how to read</span>
                <h2>Interpretation</h2>
                <p>What each visual element actually encodes. Read once; trust the visuals after.</p>
              </header>

              <div className="dx-interpret-grid">
                {INTERPRETS.map((i, idx) => (
                  <div key={idx} className="dx-interpret">
                    <h4>{i.title}</h4>
                    <p>{i.body}</p>
                    <span className="dx-interpret-tag mono">{i.chartHint}</span>
                  </div>
                ))}
              </div>
            </section>

            <footer className="dx-foot">
              <span className="mono dim">end of reference</span>
              <a href="about.html" className="btn">
                <Icon.ArrowRight style={{transform: "rotate(180deg)"}}/> Back to the tour
              </a>
            </footer>
          </section>
        </div>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<Docs/>);
