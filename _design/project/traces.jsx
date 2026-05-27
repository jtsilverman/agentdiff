/* traces.jsx — /traces corpus browser
   Sibling of baseline detail TraceList. Same row anatomy, same outcome palette,
   same .seg filter chrome. Primary column flips: baseline_name is now a real
   column instead of being implied by page context.
*/

const { useState: useStateT, useMemo: useMemoT, useEffect: useEffectT } = React;

// ─── API outcome → existing OutcomeBadge kind ───
const OUTCOME_KIND = {
  succeeded: "ok",
  variance:  "warn",
  regressed: "bad",
  additive:  "novel",
};
// The label we show users matches the API string (matches filter chip names)
const OUTCOME_LABEL = {
  succeeded: "succeeded",
  variance:  "variance",
  regressed: "regressed",
  additive:  "additive",
};

// ─── Synthetic corpus ───
// Schema mirrors GET /api/traces: {id, name, adapter, step_count, metadata, created_at}
// + the additive baseline_name field the spec said WILL ship (designed for it).
const NOW = Date.parse("2026-05-27T16:00:00Z");
const ago = (mins) => new Date(NOW - mins * 60_000).toISOString();

const TRACES = [
  // rate-limit baseline
  { id: "t-7a4f", name: "Run 1",            baseline_name: "rate-limit",     baseline_id: "rate-limit",     adapter: "claude-code",       step_count: 9,  metadata: { task: "Add rate limiting to a Flask endpoint", outcome: "succeeded", key_decision: "Chose flask-limiter — used decorator pattern, no Redis" }, created_at: ago(35) },
  { id: "t-2c81", name: "Run 2",            baseline_name: "rate-limit",     baseline_id: "rate-limit",     adapter: "claude-code",       step_count: 13, metadata: { task: "Add rate limiting to a Flask endpoint", outcome: "succeeded", key_decision: "Built before_request middleware backed by Redis INCR" }, created_at: ago(40) },
  { id: "t-9b30", name: "Run 3",            baseline_name: "rate-limit",     baseline_id: "rate-limit",     adapter: "claude-code",       step_count: 10, metadata: { task: "Add rate limiting to a Flask endpoint", outcome: "succeeded", key_decision: "flask-limiter again, in-memory backend" }, created_at: ago(44) },
  { id: "t-d57e", name: "Run 4",            baseline_name: "rate-limit",     baseline_id: "rate-limit",     adapter: "claude-code",       step_count: 14, metadata: { task: "Add rate limiting to a Flask endpoint", outcome: "additive",  key_decision: "Wrote a Lua atomic counter, skipped Python middleware" }, created_at: ago(48) },
  { id: "t-411a", name: "Run 5",            baseline_name: "rate-limit",     baseline_id: "rate-limit",     adapter: "claude-code",       step_count: 10, metadata: { task: "Add rate limiting to a Flask endpoint", outcome: "succeeded", key_decision: "flask-limiter, simpler config than Run 1" }, created_at: ago(53) },

  // auth-migration baseline
  { id: "t-c1b2", name: "v1.2 · Run 1",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 16, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "succeeded", key_decision: "express-session + connect-redis (baseline pattern)" }, created_at: ago(160) },
  { id: "t-44d8", name: "v1.2 · Run 2",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 17, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "succeeded", key_decision: "Same pattern, explicit session.touch() in middleware" }, created_at: ago(168) },
  { id: "t-9aef", name: "v1.2 · Run 3",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 18, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "succeeded", key_decision: "Same pattern, refactored middleware order" }, created_at: ago(174) },
  { id: "t-0f30", name: "v1.2 · Run 4",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 16, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "succeeded", key_decision: "Same pattern, clean" }, created_at: ago(186) },
  { id: "t-bc91", name: "v1.3 · Run 1",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 19, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "variance",  key_decision: "Added a refresh loop — slower but passes" }, created_at: ago(80) },
  { id: "t-bc92", name: "v1.3 · Run 2",     baseline_name: "auth-migration", baseline_id: "auth-migration", adapter: "cursor",            step_count: 22, metadata: { task: "Migrate auth from JWT to session cookies", outcome: "regressed", key_decision: "Refresh loop deadlocked when session was read mid-rotate" }, created_at: ago(72) },

  // flaky-test baseline
  { id: "t-aa11", name: "Run 1",            baseline_name: "flaky-test",     baseline_id: "flaky-test",     adapter: "claude-code",       step_count: 24, metadata: { task: "Debug intermittent test failure in CI", outcome: "variance",  key_decision: "Added mutex around user_count() — still flakes" }, created_at: ago(720) },
  { id: "t-aa12", name: "Run 2",            baseline_name: "flaky-test",     baseline_id: "flaky-test",     adapter: "claude-code",       step_count: 22, metadata: { task: "Debug intermittent test failure in CI", outcome: "variance",  key_decision: "Added retry-on-failure (masks the bug)" }, created_at: ago(740) },
  { id: "t-aa13", name: "Run 3",            baseline_name: "flaky-test",     baseline_id: "flaky-test",     adapter: "claude-code",       step_count: 25, metadata: { task: "Debug intermittent test failure in CI", outcome: "variance",  key_decision: "Added DB transaction wrapping — wrong root cause" }, created_at: ago(750) },
  { id: "t-aa14", name: "Run 4",            baseline_name: "flaky-test",     baseline_id: "flaky-test",     adapter: "claude-code",       step_count: 21, metadata: { task: "Debug intermittent test failure in CI", outcome: "additive",  key_decision: "Noticed 2y CI clock skew, fixed timestamp parser" }, created_at: ago(760) },
  { id: "t-aa15", name: "Run 5",            baseline_name: "flaky-test",     baseline_id: "flaky-test",     adapter: "claude-code",       step_count: 25, metadata: { task: "Debug intermittent test failure in CI", outcome: "variance",  key_decision: "Promise.race await — still flakes" }, created_at: ago(770) },

  // more baselines for browsing variety
  { id: "t-key1", name: "Run 1",            baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code",   step_count: 12, metadata: { task: "Rotate API keys without downtime", outcome: "succeeded", key_decision: "Dual-write window, retire old key after 24h" }, created_at: ago(2880) },
  { id: "t-key2", name: "Run 2",            baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code",   step_count: 15, metadata: { task: "Rotate API keys without downtime", outcome: "succeeded", key_decision: "Same dual-write, added healthcheck on new key" }, created_at: ago(2900) },
  { id: "t-key3", name: "Run 3",            baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code",   step_count: 11, metadata: { task: "Rotate API keys without downtime", outcome: "succeeded", key_decision: "Skipped healthcheck, monitored 5xx rate instead" }, created_at: ago(2940) },
  { id: "t-key4", name: "Run 4",            baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code",   step_count: 19, metadata: { task: "Rotate API keys without downtime", outcome: "additive",  key_decision: "Wrote a key-rotation runbook, then automated it" }, created_at: ago(3000) },

  { id: "t-cors1", name: "Run 1",           baseline_name: "cors-debug",     baseline_id: "cors-debug",     adapter: "cursor",            step_count: 8,  metadata: { task: "Debug CORS preflight on /api/upload", outcome: "succeeded", key_decision: "Added OPTIONS handler with explicit Access-Control-Allow-Headers" }, created_at: ago(8640) },
  { id: "t-cors2", name: "Run 2",           baseline_name: "cors-debug",     baseline_id: "cors-debug",     adapter: "cursor",            step_count: 11, metadata: { task: "Debug CORS preflight on /api/upload", outcome: "succeeded", key_decision: "Same as Run 1, also fixed Access-Control-Max-Age" }, created_at: ago(8700) },
  { id: "t-cors3", name: "Run 3",           baseline_name: "cors-debug",     baseline_id: "cors-debug",     adapter: "cursor",            step_count: 14, metadata: { task: "Debug CORS preflight on /api/upload", outcome: "regressed", key_decision: "Wildcard origin caused credentials check to fail" }, created_at: ago(8760) },
  { id: "t-cors4", name: "Run 4",           baseline_name: "cors-debug",     baseline_id: "cors-debug",     adapter: "cursor",            step_count: 9,  metadata: { task: "Debug CORS preflight on /api/upload", outcome: "succeeded", key_decision: "Reverted wildcard, allow-list of trusted origins" }, created_at: ago(8820) },
  { id: "t-cors5", name: "Run 5",           baseline_name: "cors-debug",     baseline_id: "cors-debug",     adapter: "cursor",            step_count: 10, metadata: { task: "Debug CORS preflight on /api/upload", outcome: "succeeded", key_decision: "Same, with origin extracted to env var" }, created_at: ago(8880) },

  { id: "t-cel1", name: "Run 1",            baseline_name: "celery-retry",   baseline_id: "celery-retry",   adapter: "claude-code",       step_count: 17, metadata: { task: "Make a Celery task idempotent", outcome: "succeeded", key_decision: "Idempotency key in Redis, 24h TTL" }, created_at: ago(14400) },
  { id: "t-cel2", name: "Run 2",            baseline_name: "celery-retry",   baseline_id: "celery-retry",   adapter: "claude-code",       step_count: 21, metadata: { task: "Make a Celery task idempotent", outcome: "additive",  key_decision: "Idempotency via DB unique constraint, no Redis" }, created_at: ago(14500) },
  { id: "t-cel3", name: "Run 3",            baseline_name: "celery-retry",   baseline_id: "celery-retry",   adapter: "claude-code",       step_count: 18, metadata: { task: "Make a Celery task idempotent", outcome: "succeeded", key_decision: "Redis lock + DB unique constraint (belt + suspenders)" }, created_at: ago(14600) },

  { id: "t-pg1", name: "Run 1",             baseline_name: "redis-cache",    baseline_id: "redis-cache",    adapter: "claude-code",       step_count: 13, metadata: { task: "Add Redis-backed caching to /products", outcome: "succeeded", key_decision: "Cache-aside pattern, 5min TTL" }, created_at: ago(20000) },
  { id: "t-pg2", name: "Run 2",             baseline_name: "redis-cache",    baseline_id: "redis-cache",    adapter: "claude-code",       step_count: 16, metadata: { task: "Add Redis-backed caching to /products", outcome: "regressed", key_decision: "Cache invalidation missed on PUT — stale reads for 5min" }, created_at: ago(20100) },
];

// ─── Helpers ───
function relativeTime(iso) {
  const t = new Date(iso).getTime();
  const diff = (NOW - t) / 1000; // seconds
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 7) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / (86400 * 7))}w ago`;
  return `${Math.floor(diff / (86400 * 30))}mo ago`;
}

function formatStamp(iso) {
  const d = new Date(iso);
  const mins = String(d.getUTCMinutes()).padStart(2, "0");
  const hours = String(d.getUTCHours()).padStart(2, "0");
  const day = String(d.getUTCDate()).padStart(2, "0");
  const month = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"][d.getUTCMonth()];
  return `${month} ${day} · ${hours}:${mins}`;
}

// ─── Filter bar ───
const OUTCOMES = ["succeeded", "variance", "regressed", "additive"];

const FilterBar = ({ baselines, baseline, setBaseline, outcomes, toggleOutcome, query, setQuery, resultCount, total, clearAll }) => {
  const isFiltering = baseline !== "all" || outcomes.size > 0 || query.trim() !== "";

  return (
    <div className="tr-filters">
      <div className="tr-filter-row">
        <div className="tr-filter-group">
          <label className="tr-filter-label mono">baseline</label>
          <select className="select" value={baseline} onChange={e => setBaseline(e.target.value)}>
            <option value="all">all baselines · {baselines.length}</option>
            {baselines.map(b => (
              <option key={b.id} value={b.id}>{b.name} · {b.count}</option>
            ))}
          </select>
        </div>

        <div className="tr-filter-group">
          <label className="tr-filter-label mono">outcome</label>
          <div className="seg">
            <button className={"seg-btn" + (outcomes.size === 0 ? " active" : "")} onClick={() => clearAll("outcomes")}>all</button>
            {OUTCOMES.map(o => (
              <button key={o} className={"seg-btn" + (outcomes.has(o) ? " active" : "")} onClick={() => toggleOutcome(o)}>{o}</button>
            ))}
          </div>
        </div>

        <div className="tr-filter-group tr-filter-search">
          <label className="tr-filter-label mono">task</label>
          <div className="tr-search">
            <Icon.Search/>
            <input
              type="text"
              className="tr-search-input"
              placeholder="filter by task name or key decision…"
              value={query}
              onChange={e => setQuery(e.target.value)}
            />
            {query && (
              <button className="tr-search-clear" onClick={() => setQuery("")}><Icon.Close/></button>
            )}
          </div>
        </div>
      </div>

      <div className="tr-filter-summary">
        <span className="mono dim">
          {resultCount === total
            ? <>showing <span style={{color: "var(--fg)"}}>{total}</span> traces</>
            : <>showing <span style={{color: "var(--fg)"}}>{resultCount}</span> of {total} traces</>
          }
        </span>
        {isFiltering && (
          <button className="btn ghost" onClick={() => clearAll("everything")} style={{padding: "4px 10px", fontSize: 12}}>
            <Icon.Close/> clear filters
          </button>
        )}
      </div>
    </div>
  );
};

// ─── Trace row ───
const TraceRow = ({ t, query }) => {
  const out = t.metadata.outcome;
  const summary = t.metadata.key_decision || t.metadata.task;

  // Highlight matches in task / decision text
  const renderHighlight = (text) => {
    if (!query.trim()) return text;
    const re = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`, "ig");
    const parts = text.split(re);
    return parts.map((p, i) => re.test(p)
      ? <mark key={i}>{p}</mark>
      : <React.Fragment key={i}>{p}</React.Fragment>
    );
  };

  return (
    <a href="#" className="tl-row tr-row" onClick={e => e.preventDefault()}>
      <span className="tl-cell tr-cell-id">
        <span className="tl-trace-name">{t.name}</span>
        <span className="mono dim" style={{fontSize: 11}}>{t.id}</span>
      </span>

      <span className="tl-cell tr-cell-baseline">
        <span className="tr-baseline-link mono">{t.baseline_name}</span>
      </span>

      <span className="tl-cell tr-cell-out">
        <OutcomeBadge kind={OUTCOME_KIND[out]} label={OUTCOME_LABEL[out]}/>
      </span>

      <span className="tl-cell tr-cell-steps mono">{t.step_count}</span>

      <span className="tl-cell tr-cell-task">
        <div className="tr-task-text">{renderHighlight(t.metadata.task)}</div>
        {t.metadata.key_decision && (
          <div className="tr-decision-text mono">{renderHighlight(t.metadata.key_decision)}</div>
        )}
      </span>

      <span className="tl-cell tr-cell-adapter">
        <span className="tr-adapter mono">{t.adapter}</span>
      </span>

      <span className="tl-cell tr-cell-time mono" title={t.created_at}>
        {relativeTime(t.created_at)}
      </span>

      <span className="tl-cell tl-cell-arrow"><Icon.ChevronRight/></span>
    </a>
  );
};

// ─── Empty state ───
const EmptyState = ({ onClear }) => (
  <div className="tr-empty">
    <div className="tr-empty-icon">
      <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
        <circle cx="18" cy="18" r="11" stroke="var(--border-hi)" strokeWidth="1.4" strokeDasharray="3 3"/>
        <path d="M26 26 L 34 34" stroke="var(--border-hi)" strokeWidth="1.4" strokeLinecap="round"/>
        <circle cx="18" cy="18" r="3" fill="var(--accent)" opacity="0.4"/>
      </svg>
    </div>
    <h4>No traces match these filters.</h4>
    <p>Try widening the baseline, clearing the outcome chips, or shortening your search.</p>
    <button className="btn" onClick={onClear}>Clear all filters</button>
  </div>
);

// ─── Sort menu ───
const SORTS = [
  { id: "recent", label: "newest first" },
  { id: "oldest", label: "oldest first" },
  { id: "steps_desc", label: "most steps" },
  { id: "steps_asc", label: "fewest steps" },
  { id: "outcome", label: "outcome" },
  { id: "baseline", label: "baseline name" },
];

// ─── Main ───
const Traces = () => {
  const [baseline, setBaseline]   = useStateT("all");
  const [outcomes, setOutcomes]   = useStateT(new Set()); // Set of outcome strings
  const [query, setQuery]         = useStateT("");
  const [sort, setSort]           = useStateT("recent");

  // Compute baseline list from corpus
  const baselines = useMemoT(() => {
    const counts = {};
    TRACES.forEach(t => { counts[t.baseline_id] = (counts[t.baseline_id] || 0) + 1; });
    const order = [];
    TRACES.forEach(t => {
      if (!order.find(b => b.id === t.baseline_id)) {
        order.push({ id: t.baseline_id, name: t.baseline_name, count: counts[t.baseline_id] });
      }
    });
    return order;
  }, []);

  // Filter + sort
  const filtered = useMemoT(() => {
    let rows = TRACES;
    if (baseline !== "all") rows = rows.filter(t => t.baseline_id === baseline);
    if (outcomes.size > 0) rows = rows.filter(t => outcomes.has(t.metadata.outcome));
    if (query.trim()) {
      const q = query.toLowerCase();
      rows = rows.filter(t =>
        t.metadata.task.toLowerCase().includes(q) ||
        (t.metadata.key_decision || "").toLowerCase().includes(q) ||
        t.name.toLowerCase().includes(q) ||
        t.baseline_name.toLowerCase().includes(q)
      );
    }

    rows = [...rows];
    if (sort === "recent")      rows.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    else if (sort === "oldest") rows.sort((a, b) => new Date(a.created_at) - new Date(b.created_at));
    else if (sort === "steps_desc") rows.sort((a, b) => b.step_count - a.step_count);
    else if (sort === "steps_asc")  rows.sort((a, b) => a.step_count - b.step_count);
    else if (sort === "outcome")    rows.sort((a, b) => a.metadata.outcome.localeCompare(b.metadata.outcome));
    else if (sort === "baseline")   rows.sort((a, b) => a.baseline_name.localeCompare(b.baseline_name));
    return rows;
  }, [baseline, outcomes, query, sort]);

  const toggleOutcome = (o) => {
    const next = new Set(outcomes);
    if (next.has(o)) next.delete(o); else next.add(o);
    setOutcomes(next);
  };
  const clearAll = (what) => {
    if (what === "outcomes") setOutcomes(new Set());
    else {
      setBaseline("all"); setOutcomes(new Set()); setQuery("");
    }
  };

  // Adapter pill counts for the header strip
  const adapterCounts = useMemoT(() => {
    const c = {};
    TRACES.forEach(t => { c[t.adapter] = (c[t.adapter] || 0) + 1; });
    return c;
  }, []);
  const outcomeCounts = useMemoT(() => {
    const c = { succeeded: 0, variance: 0, regressed: 0, additive: 0 };
    TRACES.forEach(t => { c[t.metadata.outcome]++; });
    return c;
  }, []);

  return (
    <>
      <Nav active="traces"/>

      <main>
        {/* HERO STRIP — same anatomy as baseline header */}
        <header className="tr-hero" data-screen-label="01 Traces hero">
          <div className="container">
            <div className="tr-hero-grid">
              <div>
                <div className="bl-breadcrumb mono">
                  <a href="index.html" className="bl-crumb">home</a>
                  <span className="dim">/</span>
                  <span className="bl-crumb-current">traces</span>
                </div>
                <h1 className="tr-hero-title">Traces</h1>
                <p className="tr-hero-sub">
                  Every recording across every baseline, in one place. Filter, sort, click in.
                </p>
              </div>

              <aside className="tr-hero-stats">
                <div className="bl-stat">
                  <span className="mono dim">traces</span>
                  <span className="bl-stat-val">{TRACES.length}</span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">baselines</span>
                  <span className="bl-stat-val">{baselines.length}</span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">regressed</span>
                  <span className="bl-stat-val" style={{color: outcomeCounts.regressed > 0 ? "var(--bad)" : "var(--fg)"}}>{outcomeCounts.regressed}</span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">additive</span>
                  <span className="bl-stat-val" style={{color: outcomeCounts.additive > 0 ? "var(--novel)" : "var(--fg)"}}>{outcomeCounts.additive}</span>
                </div>
              </aside>
            </div>

            <div className="tr-hero-adapters">
              <span className="eyebrow">recorded via</span>
              {Object.entries(adapterCounts).map(([a, n]) => (
                <span key={a} className="badge neutral mono">{a} · {n}</span>
              ))}
            </div>
          </div>
        </header>

        <div className="container">
          {/* FILTERS */}
          <FilterBar
            baselines={baselines}
            baseline={baseline}
            setBaseline={setBaseline}
            outcomes={outcomes}
            toggleOutcome={toggleOutcome}
            query={query}
            setQuery={setQuery}
            resultCount={filtered.length}
            total={TRACES.length}
            clearAll={clearAll}
          />

          {/* SORT */}
          <div className="tr-list-head">
            <div className="row" style={{gap: 10}}>
              <span className="eyebrow">trace list</span>
              <span className="mono dim" style={{fontSize: 11}}>{filtered.length} rows</span>
            </div>
            <select className="select" value={sort} onChange={e => setSort(e.target.value)}>
              {SORTS.map(s => <option key={s.id} value={s.id}>sort: {s.label}</option>)}
            </select>
          </div>

          {/* TABLE */}
          {filtered.length === 0 ? (
            <EmptyState onClear={() => clearAll("everything")}/>
          ) : (
            <div className="tl-table tr-table">
              <div className="tl-row tl-head tr-head mono">
                <span className="tl-cell tr-cell-id">trace</span>
                <span className="tl-cell tr-cell-baseline">baseline</span>
                <span className="tl-cell tr-cell-out">outcome</span>
                <span className="tl-cell tr-cell-steps">steps</span>
                <span className="tl-cell tr-cell-task">task · key decision</span>
                <span className="tl-cell tr-cell-adapter">adapter</span>
                <span className="tl-cell tr-cell-time">recorded</span>
                <span className="tl-cell tl-cell-arrow"></span>
              </div>

              {filtered.map(t => <TraceRow key={t.id} t={t} query={query}/>)}
            </div>
          )}

          {/* FOOT */}
          <div className="tr-foot">
            <span className="mono dim" style={{fontSize: 11}}>
              api · GET /api/traces · {TRACES.length} records
            </span>
            <a href="index.html" className="btn ghost">
              <Icon.ArrowRight style={{transform: "rotate(180deg)"}}/> Back to home
            </a>
          </div>
        </div>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<Traces/>);
