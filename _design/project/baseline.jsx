/* baseline.jsx — baseline detail page */

const { useState: useStateB, useMemo: useMemoB } = React;

const BaselineHeader = ({ baseline }) => (
  <header className="bl-header" data-screen-label="01 Baseline header">
    <div className="container">
      <div className="bl-breadcrumb mono">
        <a href="index.html" className="bl-crumb">home</a>
        <span className="dim">/</span>
        <span className="bl-crumb">baselines</span>
        <span className="dim">/</span>
        <span className="bl-crumb-current">{baseline.id}</span>
      </div>

      <div className="bl-head-grid">
        <div className="bl-head-main">
          <div className="row" style={{gap: 12, marginBottom: 18, flexWrap: "wrap"}}>
            <OutcomeBadge kind={baseline.outcomeKind} label={baseline.outcomeLabel}/>
            <span className="badge neutral mono">{baseline.traces.length} runs</span>
            <span className="badge neutral mono">{baseline.model}</span>
            <span className="badge neutral mono">avg {baseline.stepsAvg} steps</span>
            <span className="badge neutral mono">avg {baseline.durationAvg}</span>
          </div>

          <h1 className="bl-title">{baseline.task}</h1>

          <div className="bl-callout">
            <div className="bl-callout-head">
              <span className="eyebrow">task description</span>
              <span className="mono dim" style={{fontSize: 11}}>captured {baseline.captured}</span>
            </div>
            <div className="bl-callout-body">
              {baseline.description.map((p, i) => <p key={i}>{p}</p>)}
            </div>
            <div className="bl-callout-prompt">
              <span className="eyebrow">prompt</span>
              <pre className="mono">{baseline.prompt}</pre>
            </div>
          </div>
        </div>

        <aside className="bl-head-side">
          <div className="bl-stat">
            <span className="mono dim">runs</span>
            <span className="bl-stat-val">{baseline.traces.length}</span>
          </div>
          <div className="bl-stat">
            <span className="mono dim">avg steps</span>
            <span className="bl-stat-val">{baseline.stepsAvg}</span>
          </div>
          <div className="bl-stat">
            <span className="mono dim">strategies</span>
            <span className="bl-stat-val">{new Set(baseline.traces.map(t => t.strategy)).size}</span>
          </div>
          <div className="bl-stat">
            <span className="mono dim">drift</span>
            <span className="bl-stat-val" style={{color: baseline.outcomeKind === "bad" ? "var(--bad)" : baseline.outcomeKind === "novel" ? "var(--novel)" : "var(--warn)"}}>
              {baseline.outcomeKind === "bad" ? "high" : baseline.outcomeKind === "novel" ? "novel" : "med"}
            </span>
          </div>
        </aside>
      </div>
    </div>
  </header>
);

// Trace list
const TraceList = ({ baseline, filter, setFilter, sortBy, setSortBy }) => {
  const filtered = useMemoB(() => {
    let rows = baseline.traces;
    if (filter !== "all") {
      rows = rows.filter(t => t.outcome === filter);
    }
    if (sortBy === "steps") rows = [...rows].sort((a, b) => a.steps - b.steps);
    else if (sortBy === "outcome") rows = [...rows].sort((a, b) => a.outcome.localeCompare(b.outcome));
    return rows;
  }, [baseline.id, filter, sortBy]);

  return (
    <section className="bl-section" data-screen-label="02 Trace list">
      <div className="bl-section-head">
        <div>
          <span className="eyebrow">runs · {baseline.traces.length}</span>
          <h2>{baseline.traces.length} runs of this task</h2>
        </div>
        <div className="row" style={{gap: 8}}>
          <div className="seg">
            {["all", "ok", "warn", "bad", "novel"].map(f => (
              <button key={f} className={"seg-btn" + (filter === f ? " active" : "")} onClick={() => setFilter(f)}>
                {f}
              </button>
            ))}
          </div>
          <select className="select" value={sortBy} onChange={e => setSortBy(e.target.value)}>
            <option value="default">sort: run order</option>
            <option value="steps">sort: step count</option>
            <option value="outcome">sort: outcome</option>
          </select>
        </div>
      </div>

      <div className="tl-table">
        <div className="tl-row tl-head mono">
          <span className="tl-cell tl-cell-id">trace</span>
          <span className="tl-cell tl-cell-out">outcome</span>
          <span className="tl-cell tl-cell-steps">steps</span>
          <span className="tl-cell tl-cell-dur">duration</span>
          <span className="tl-cell tl-cell-strat">strategy</span>
          <span className="tl-cell tl-cell-decision">key decision</span>
          <span className="tl-cell tl-cell-arrow"></span>
        </div>

        {filtered.map(t => (
          <a key={t.id} href="#" className="tl-row" onClick={e => e.preventDefault()}>
            <span className="tl-cell tl-cell-id">
              <span className="tl-trace-name">{t.name}</span>
              <span className="mono dim" style={{fontSize: 11}}>{t.id}</span>
            </span>
            <span className="tl-cell tl-cell-out">
              <OutcomeBadge kind={t.outcome === "ok" ? "ok" : t.outcome === "bad" ? "bad" : t.outcome === "warn" ? "warn" : "novel"} label={t.outcomeLabel}/>
            </span>
            <span className="tl-cell tl-cell-steps mono">{t.steps}</span>
            <span className="tl-cell tl-cell-dur mono">{t.duration}</span>
            <span className="tl-cell tl-cell-strat mono">{t.strategy}</span>
            <span className="tl-cell tl-cell-decision">{t.keyDecision}</span>
            <span className="tl-cell tl-cell-arrow"><Icon.ChevronRight/></span>
          </a>
        ))}

        {filtered.length === 0 && (
          <div className="tl-empty mono">No runs match this filter.</div>
        )}
      </div>
    </section>
  );
};

// Path graph section with legend
const GraphSection = ({ baseline }) => (
  <section className="bl-section" data-screen-label="03 Path graph">
    <div className="bl-section-head">
      <div>
        <span className="eyebrow">path graph · all runs overlaid</span>
        <h2>How the runs flowed</h2>
      </div>
      <div className="row" style={{gap: 6, flexWrap: "wrap", justifyContent: "flex-end"}}>
        <button className="btn ghost">
          <Icon.Filter/> Filter by trace
        </button>
        <button className="btn ghost">
          <Icon.Diff/> Overlay
        </button>
      </div>
    </div>

    <div className="pg-container">
      <PathGraph baselineId={baseline.id}/>
    </div>

    <div className="pg-legend">
      <div className="pg-legend-rules mono">
        <span><span className="pg-rule-thick"/> edge thickness = run count</span>
        <span><span className="pg-rule-size"/> node size = call frequency</span>
        <span><span className="pg-rule-color"/> color = outcome cluster</span>
      </div>
      <div className="pg-legend-clusters">
        {baseline.legend.map((l, i) => (
          <div key={i} className="pg-cluster">
            <span className="pg-cluster-dot" style={{background: l.color}}/>
            <span className="mono" style={{fontSize: 12}}>{l.label}</span>
            <span className="mono dim" style={{fontSize: 11}}>×{l.count}</span>
          </div>
        ))}
      </div>
    </div>
  </section>
);

// Money feature buttons
const MoneyRow = ({ onCF, onEdit, onSim }) => (
  <section className="bl-section" data-screen-label="04 Money features">
    <div className="bl-section-head">
      <div>
        <span className="eyebrow">analysis</span>
        <h2>Now ask "what if?"</h2>
      </div>
    </div>

    <div className="money-grid">
      <button className="money-card" onClick={onCF}>
        <div className="money-card-icon"><Icon.Branch/></div>
        <div className="money-card-body">
          <div className="money-card-title">Run counterfactual</div>
          <div className="money-card-sub">Pick a trace, pick a step, force a different decision and see what happens.</div>
        </div>
        <div className="money-card-cta">
          <span className="mono dim">⌘1</span>
          <Icon.ArrowRight/>
        </div>
      </button>

      <button className="money-card" onClick={onEdit}>
        <div className="money-card-icon"><Icon.Edit/></div>
        <div className="money-card-body">
          <div className="money-card-title">Edit the prompt</div>
          <div className="money-card-sub">Rewrite the prompt and re-run 5 traces to see how the path distribution shifts.</div>
        </div>
        <div className="money-card-cta">
          <span className="mono dim">⌘2</span>
          <Icon.ArrowRight/>
        </div>
      </button>

      <button className="money-card" onClick={onSim}>
        <div className="money-card-icon"><Icon.Search/></div>
        <div className="money-card-body">
          <div className="money-card-title">Find similar traces</div>
          <div className="money-card-sub">Pull other baselines in your workspace with similar shape, failure mode, or toolchain.</div>
        </div>
        <div className="money-card-cta">
          <span className="mono dim">⌘3</span>
          <Icon.ArrowRight/>
        </div>
      </button>
    </div>
  </section>
);

// MAIN
const Baseline = () => {
  const params = new URLSearchParams(location.search);
  const id = params.get("id") || "rate-limit";
  const baseline = BASELINES[id] || BASELINES["rate-limit"];

  const [filter, setFilter] = useStateB("all");
  const [sortBy, setSortBy] = useStateB("default");
  const [cfOpen, setCfOpen] = useStateB(false);
  const [editOpen, setEditOpen] = useStateB(false);
  const [simOpen, setSimOpen] = useStateB(false);

  // ⌘ shortcuts
  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "1") { e.preventDefault(); setCfOpen(true); }
      if ((e.metaKey || e.ctrlKey) && e.key === "2") { e.preventDefault(); setEditOpen(true); }
      if ((e.metaKey || e.ctrlKey) && e.key === "3") { e.preventDefault(); setSimOpen(true); }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <>
      <Nav active="home"/>

      <main>
        <BaselineHeader baseline={baseline}/>
        <div className="container">
          <TraceList baseline={baseline} filter={filter} setFilter={setFilter} sortBy={sortBy} setSortBy={setSortBy}/>
          <GraphSection baseline={baseline}/>
          <MoneyRow onCF={() => setCfOpen(true)} onEdit={() => setEditOpen(true)} onSim={() => setSimOpen(true)}/>

          <div className="bl-other">
            <span className="eyebrow">other baselines</span>
            <div className="row" style={{gap: 8, flexWrap: "wrap", marginTop: 12}}>
              {Object.values(BASELINES).filter(b => b.id !== baseline.id).map(b => (
                <a key={b.id} href={"baseline.html?id=" + b.id} className="bl-other-chip">
                  <OutcomeBadge kind={b.outcomeKind} label={b.outcomeLabel}/>
                  <span>{b.task}</span>
                  <Icon.ArrowRight/>
                </a>
              ))}
            </div>
          </div>
        </div>
      </main>

      <Footer/>

      <CounterfactualModal open={cfOpen} onClose={() => setCfOpen(false)} baseline={baseline}/>
      <EditPromptModal open={editOpen} onClose={() => setEditOpen(false)} baseline={baseline}/>
      <SimilarModal open={simOpen} onClose={() => setSimOpen(false)} baseline={baseline}/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<Baseline/>);
