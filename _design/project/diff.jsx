/* diff.jsx — /diff/[a]/[b] side-by-side trace compare
   Reads ?a= and ?b= from URL. No IDs → auto-selects most recent two
   cross-baseline traces with a banner explaining the auto-pick.
*/

const { useState: useStateD, useMemo: useMemoD, useEffect: useEffectD, useRef: useRefD } = React;

// ─── Helpers ───
const OUTCOME_KIND  = window.CORPUS_OUTCOME_KIND;
const OUTCOME_LABEL = window.CORPUS_OUTCOME_LABEL;
const TRACES        = window.CORPUS_TRACES;

function readUrlPair() {
  const p = new URLSearchParams(location.search);
  return { a: p.get("a"), b: p.get("b") };
}
function writeUrlPair(a, b) {
  const p = new URLSearchParams();
  if (a) p.set("a", a);
  if (b) p.set("b", b);
  const qs = p.toString();
  history.replaceState(null, "", "diff.html" + (qs ? "?" + qs : ""));
}

// Auto-select: most recent two traces from different baselines
function autoSelectPair() {
  const sorted = [...TRACES].sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
  const first = sorted[0];
  const second = sorted.find(t => t.baseline_id !== first.baseline_id) || sorted[1];
  return { a: first.id, b: second.id };
}

// ─── Picker dropdown ───
const Picker = ({ open, onClose, currentId, otherId, onSelect, anchor }) => {
  const [query, setQuery] = useStateD("");
  const ref = useRefD(null);

  useEffectD(() => {
    if (!open) return;
    const onDoc = (e) => {
      if (ref.current && !ref.current.contains(e.target) && !anchor?.contains(e.target)) onClose();
    };
    const onKey = (e) => { if (e.key === "Escape") onClose(); };
    setTimeout(() => document.addEventListener("mousedown", onDoc), 0);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const filtered = useMemoD(() => {
    if (!query.trim()) return TRACES;
    const q = query.toLowerCase();
    return TRACES.filter(t =>
      t.name.toLowerCase().includes(q) ||
      t.id.toLowerCase().includes(q) ||
      t.baseline_name.toLowerCase().includes(q) ||
      t.metadata.task.toLowerCase().includes(q)
    );
  }, [query]);

  // group by baseline
  const grouped = useMemoD(() => {
    const groups = {};
    filtered.forEach(t => {
      if (!groups[t.baseline_id]) groups[t.baseline_id] = { name: t.baseline_name, items: [] };
      groups[t.baseline_id].items.push(t);
    });
    return Object.entries(groups);
  }, [filtered]);

  if (!open) return null;

  return (
    <div className="diff-picker" ref={ref}>
      <div className="diff-picker-head">
        <Icon.Search/>
        <input
          autoFocus
          className="diff-picker-input"
          type="text"
          placeholder="search traces, baselines, tasks…"
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
        {query && (
          <button className="tr-search-clear" onClick={() => setQuery("")}><Icon.Close/></button>
        )}
      </div>
      <div className="diff-picker-list">
        {grouped.length === 0 && (
          <div className="diff-picker-empty mono">no traces match.</div>
        )}
        {grouped.map(([bId, g]) => (
          <div key={bId} className="diff-picker-group">
            <div className="diff-picker-group-head mono">{g.name} <span className="dim">· {g.items.length}</span></div>
            {g.items.map(t => {
              const isSelf = t.id === otherId;
              const isCurrent = t.id === currentId;
              return (
                <button
                  key={t.id}
                  className={"diff-picker-row" + (isCurrent ? " current" : "") + (isSelf ? " self" : "")}
                  disabled={isSelf}
                  onClick={() => { if (!isSelf) { onSelect(t.id); onClose(); } }}
                >
                  <span className="diff-picker-row-name">{t.name}</span>
                  <span className="mono dim diff-picker-row-id">{t.id}</span>
                  <OutcomeBadge kind={OUTCOME_KIND[t.metadata.outcome]} label={OUTCOME_LABEL[t.metadata.outcome]}/>
                  <span className="mono dim diff-picker-row-time">{window.corpusRelativeTime(t.created_at)}</span>
                  {isCurrent && <span className="mono diff-picker-row-tag">current</span>}
                  {isSelf && <span className="mono diff-picker-row-tag dim">other side</span>}
                </button>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
};

// ─── Trace card (in switcher strip) ───
const TraceCard = ({ side, trace, otherId, onSwap }) => {
  const [pickerOpen, setPickerOpen] = useStateD(false);
  const btnRef = useRefD(null);

  if (!trace) {
    return (
      <div className={"diff-trace-card empty " + side}>
        <div className="diff-trace-card-label mono">side {side}</div>
        <button className="btn primary" ref={btnRef} onClick={() => setPickerOpen(true)}>
          Pick a trace <Icon.ChevronDown/>
        </button>
        <Picker
          open={pickerOpen}
          onClose={() => setPickerOpen(false)}
          currentId={null}
          otherId={otherId}
          onSelect={onSwap}
          anchor={btnRef.current}
        />
      </div>
    );
  }

  return (
    <div className={"diff-trace-card " + side}>
      <div className="diff-trace-card-head">
        <span className="diff-trace-card-label mono">side {side}</span>
        <button
          className="diff-trace-card-switch"
          ref={btnRef}
          onClick={() => setPickerOpen(true)}
        >
          Switch <Icon.ChevronDown/>
        </button>
      </div>

      <div className="diff-trace-card-body">
        <div className="row" style={{gap: 8, marginBottom: 6, flexWrap: "wrap"}}>
          <span className="diff-trace-card-name">{trace.name}</span>
          <span className="mono dim" style={{fontSize: 11}}>{trace.id}</span>
        </div>

        <a href={"baseline.html?id=" + trace.baseline_id} className="diff-trace-card-baseline mono">
          {trace.baseline_name}
        </a>

        <div className="row" style={{gap: 8, marginTop: 10, flexWrap: "wrap"}}>
          <OutcomeBadge kind={OUTCOME_KIND[trace.metadata.outcome]} label={OUTCOME_LABEL[trace.metadata.outcome]}/>
          <span className="badge neutral mono">{trace.step_count} steps</span>
          <span className="badge neutral mono">{trace.adapter}</span>
        </div>

        <div className="diff-trace-card-task">{trace.metadata.task}</div>
      </div>

      <Picker
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        currentId={trace.id}
        otherId={otherId}
        onSelect={onSwap}
        anchor={btnRef.current}
      />
    </div>
  );
};

// ─── Summary stats pill row ───
const SummaryStats = ({ summary, distance }) => (
  <div className="diff-summary">
    <div className="diff-summary-pills">
      <div className="diff-pill ok">
        <span className="diff-pill-num">{summary.matches}</span>
        <span className="diff-pill-label mono">matches</span>
      </div>
      <div className="diff-pill warn">
        <span className="diff-pill-num">{summary.substitutions}</span>
        <span className="diff-pill-label mono">substitutions</span>
      </div>
      <div className="diff-pill novel">
        <span className="diff-pill-num">{summary.insertions}</span>
        <span className="diff-pill-label mono">insertions · in B</span>
      </div>
      <div className="diff-pill bad">
        <span className="diff-pill-num">{summary.deletions}</span>
        <span className="diff-pill-label mono">deletions · in A</span>
      </div>
    </div>
    <div className="diff-summary-dist">
      <span className="mono dim">edit distance</span>
      <span className="diff-summary-dist-num">{distance}</span>
    </div>
  </div>
);

// ─── Triage callout ───
const TriagePanel = ({ triage }) => {
  if (!triage) return null;
  const kind = triage.classification === "regression" ? "bad"
            : triage.classification === "additive"   ? "novel"
            : "warn";
  return (
    <div className={"diff-triage " + kind}>
      <div className="diff-triage-head">
        <span className="eyebrow">ai triage · /api/diff/.../triage</span>
        <OutcomeBadge kind={kind} label={triage.classification}/>
      </div>
      <p className="diff-triage-summary">{triage.summary}</p>
      <div className="diff-triage-cause">
        <span className="mono dim">likely cause</span>
        <p>{triage.likely_cause}</p>
      </div>
    </div>
  );
};

// ─── Step body (renders inside an alignment row, one side) ───
const ROLE_LABEL = {
  user: "user",
  assistant: "assistant",
  tool_call: "tool call",
  tool_result: "tool result",
};

const StepBody = ({ step, ghost, sideClass }) => {
  if (ghost || !step) {
    return (
      <div className={"diff-step ghost " + sideClass}>
        <span className="diff-step-ghost mono">— not in this trace —</span>
      </div>
    );
  }
  const role = step.role;
  const tc = step.tool_call;
  const tr = step.tool_result;

  return (
    <div className={"diff-step role-" + role + " " + sideClass}>
      <div className="diff-step-head">
        <span className={"diff-role-tag mono role-" + role}>{ROLE_LABEL[role]}</span>
        {tc && <span className="diff-step-toolname mono">{tc.name}()</span>}
        {tr && (
          <>
            <span className="diff-step-toolname mono">{tr.name}() →</span>
            {tr.is_error
              ? <span className="badge bad mono" style={{padding: "1px 6px", fontSize: 10}}>error</span>
              : <span className="badge ok mono" style={{padding: "1px 6px", fontSize: 10}}>ok</span>
            }
          </>
        )}
      </div>

      {step.content && (
        <p className="diff-step-content">{step.content}</p>
      )}

      {tc && tc.args && (
        <div className="diff-step-args">
          {Object.entries(tc.args).map(([k, v]) => (
            <div key={k} className="diff-step-arg">
              <span className="mono dim">{k}</span>
              <span className="mono diff-step-arg-val">{String(v)}</span>
            </div>
          ))}
        </div>
      )}

      {tr && tr.output && (
        <pre className={"diff-step-output mono " + (tr.is_error ? "error" : "")}>{tr.output}</pre>
      )}
    </div>
  );
};

// ─── Single alignment row ───
const OP_LABEL = {
  match: "=",
  substitute: "~",
  insert: "+",
  delete: "−",
};
const OP_LABEL_LONG = {
  match: "match",
  substitute: "substitute",
  insert: "insert",
  delete: "delete",
};

const AlignmentRow = ({ row, idx }) => {
  const { op, a_step, b_step, a_index, b_index } = row;
  return (
    <div className={"diff-row op-" + op}>
      <div className="diff-row-side a">
        <div className="diff-row-idx mono">
          {a_index !== null ? String(a_index + 1).padStart(2, "0") : "··"}
        </div>
        <StepBody step={a_step} ghost={a_step === null} sideClass="a"/>
      </div>

      <div className="diff-row-marker">
        <div className={"diff-op-badge op-" + op}>{OP_LABEL[op]}</div>
        <span className="diff-op-name mono">{OP_LABEL_LONG[op]}</span>
      </div>

      <div className="diff-row-side b">
        <StepBody step={b_step} ghost={b_step === null} sideClass="b"/>
        <div className="diff-row-idx mono">
          {b_index !== null ? String(b_index + 1).padStart(2, "0") : "··"}
        </div>
      </div>
    </div>
  );
};

// ─── Baseline mismatch / same-trace banners ───
const Banner = ({ kind, title, body, action }) => (
  <div className={"diff-banner " + kind}>
    <div className="diff-banner-body">
      <span className="eyebrow">{kind === "warn" ? "heads up" : kind === "info" ? "info" : "note"}</span>
      <h4>{title}</h4>
      <p>{body}</p>
    </div>
    {action}
  </div>
);

// ─── Main ───
const Diff = () => {
  // Read URL params; if missing, auto-select
  const [urlPair, setUrlPair] = useStateD(() => readUrlPair());
  const [autoMode, setAutoMode] = useStateD(() => {
    const { a, b } = readUrlPair();
    return !a && !b;
  });

  // Resolve trace IDs (URL takes precedence; otherwise auto-pick)
  const { aId, bId } = useMemoD(() => {
    if (urlPair.a && urlPair.b) return { aId: urlPair.a, bId: urlPair.b };
    if (autoMode) {
      const { a, b } = autoSelectPair();
      return { aId: a, bId: b };
    }
    return { aId: urlPair.a, bId: urlPair.b };
  }, [urlPair, autoMode]);

  const traceA = useMemoD(() => window.corpusFindTrace(aId), [aId]);
  const traceB = useMemoD(() => window.corpusFindTrace(bId), [bId]);

  // Op filter
  const [opFilter, setOpFilter] = useStateD("all");

  // Diff data
  const diff = useMemoD(() => {
    if (!aId || !bId) return null;
    return window.computeDiff(aId, bId);
  }, [aId, bId]);
  const triage = useMemoD(() => {
    if (!aId || !bId) return null;
    return window.getTriage(aId, bId);
  }, [aId, bId]);

  // Handlers
  const swapSide = (which, newId) => {
    setAutoMode(false);
    const next = which === "a" ? { a: newId, b: bId } : { a: aId, b: newId };
    setUrlPair(next);
    writeUrlPair(next.a, next.b);
  };

  // Visible alignment after op filter
  const visibleAlignment = useMemoD(() => {
    if (!diff) return [];
    if (opFilter === "all") return diff.alignment;
    return diff.alignment.filter(r => r.op === opFilter);
  }, [diff, opFilter]);

  const baselineMismatch = traceA && traceB && traceA.baseline_id !== traceB.baseline_id;
  const sameTrace = aId === bId;

  return (
    <>
      <Nav active="diff"/>

      <main>
        {/* ─── HERO STRIP ─── */}
        <header className="tr-hero diff-hero" data-screen-label="01 Diff hero">
          <div className="container">
            <div className="tr-hero-grid">
              <div>
                <div className="bl-breadcrumb mono">
                  <a href="index.html" className="bl-crumb">home</a>
                  <span className="dim">/</span>
                  <a href="traces.html" className="bl-crumb">traces</a>
                  <span className="dim">/</span>
                  <span className="bl-crumb-current">
                    diff{aId && bId ? ` / ${aId} / ${bId}` : ""}
                  </span>
                </div>
                <h1 className="tr-hero-title">Diff</h1>
                <p className="tr-hero-sub">
                  Two traces, aligned step by step. See where they agreed, where they substituted, and where one went somewhere the other didn't.
                </p>
              </div>

              {diff && (
                <aside className="tr-hero-stats diff-hero-stats">
                  <div className="bl-stat">
                    <span className="mono dim">matches</span>
                    <span className="bl-stat-val" style={{color: "var(--ok)"}}>{diff.summary.matches}</span>
                  </div>
                  <div className="bl-stat">
                    <span className="mono dim">subs</span>
                    <span className="bl-stat-val" style={{color: "var(--warn)"}}>{diff.summary.substitutions}</span>
                  </div>
                  <div className="bl-stat">
                    <span className="mono dim">insertions</span>
                    <span className="bl-stat-val" style={{color: "var(--novel)"}}>{diff.summary.insertions}</span>
                  </div>
                  <div className="bl-stat">
                    <span className="mono dim">deletions</span>
                    <span className="bl-stat-val" style={{color: "var(--bad)"}}>{diff.summary.deletions}</span>
                  </div>
                </aside>
              )}
            </div>
          </div>
        </header>

        <div className="container">
          {/* Auto-selected banner */}
          {autoMode && traceA && traceB && (
            <Banner
              kind="info"
              title="Auto-selected · most recent two traces from different baselines"
              body={`No IDs in the URL. Showing ${traceA.name} (${traceA.baseline_name}) vs ${traceB.name} (${traceB.baseline_name}). Click Switch on either side to compare different traces.`}
            />
          )}

          {/* Baseline mismatch warning */}
          {baselineMismatch && !sameTrace && (
            <Banner
              kind="warn"
              title="Different baselines — drift signal will be noisy"
              body={`${traceA.baseline_name} vs ${traceB.baseline_name}. These traces solve different problems, so most divergence reflects task differences, not agent behavior. For meaningful drift, compare two traces from the same baseline.`}
            />
          )}

          {/* Same trace */}
          {sameTrace && (
            <Banner
              kind="warn"
              title="Same trace on both sides"
              body="You picked the same trace for A and B. There's nothing to diff — switch one side to compare against a different run."
            />
          )}

          {/* ─── SWITCHER STRIP ─── */}
          <section className="diff-switcher" data-screen-label="02 Switcher strip">
            <TraceCard side="a" trace={traceA} otherId={bId} onSwap={(id) => swapSide("a", id)}/>
            <div className="diff-switcher-vs">
              <span className="mono">vs</span>
              <div className="diff-switcher-line"/>
            </div>
            <TraceCard side="b" trace={traceB} otherId={aId} onSwap={(id) => swapSide("b", id)}/>
          </section>

          {/* ─── SUMMARY + TRIAGE ─── */}
          {diff && !sameTrace && (
            <>
              <SummaryStats summary={diff.summary} distance={diff.distance}/>
              <TriagePanel triage={triage}/>
            </>
          )}

          {/* ─── OP FILTER ─── */}
          {diff && !sameTrace && diff.alignment.length > 0 && (
            <div className="diff-controls">
              <div className="row" style={{gap: 10}}>
                <span className="eyebrow">alignment · {diff.alignment.length} rows</span>
                <span className="mono dim" style={{fontSize: 11}}>
                  {visibleAlignment.length === diff.alignment.length ? null : `${visibleAlignment.length} of ${diff.alignment.length} shown`}
                </span>
              </div>
              <div className="seg">
                <button className={"seg-btn" + (opFilter === "all" ? " active" : "")} onClick={() => setOpFilter("all")}>all</button>
                <button className={"seg-btn" + (opFilter === "match" ? " active" : "")} onClick={() => setOpFilter("match")}>matches</button>
                <button className={"seg-btn" + (opFilter === "substitute" ? " active" : "")} onClick={() => setOpFilter("substitute")}>subs</button>
                <button className={"seg-btn" + (opFilter === "insert" ? " active" : "")} onClick={() => setOpFilter("insert")}>insertions</button>
                <button className={"seg-btn" + (opFilter === "delete" ? " active" : "")} onClick={() => setOpFilter("delete")}>deletions</button>
              </div>
            </div>
          )}

          {/* ─── ALIGNMENT LIST ─── */}
          {diff && !sameTrace && (
            <div className="diff-rows">
              {visibleAlignment.length === 0 ? (
                <div className="tr-empty">
                  <div className="tr-empty-icon"><Icon.Filter/></div>
                  <h4>No rows match this op filter.</h4>
                  <p>Try widening to "all" or another op.</p>
                  <button className="btn" onClick={() => setOpFilter("all")}>Show all rows</button>
                </div>
              ) : (
                visibleAlignment.map((row, i) => (
                  <AlignmentRow key={i} row={row} idx={i}/>
                ))
              )}
            </div>
          )}

          {sameTrace && (
            <div className="diff-rows">
              <div className="tr-empty">
                <div className="tr-empty-icon"><Icon.Diff/></div>
                <h4>Empty diff.</h4>
                <p>Side A and Side B are the same trace — nothing to compare.</p>
              </div>
            </div>
          )}

          {/* Foot — adapter / api line */}
          <div className="tr-foot">
            <span className="mono dim" style={{fontSize: 11}}>
              api · GET /api/diff/{aId || "·"}/{bId || "·"} · {diff?.alignment.length || 0} alignment rows
            </span>
            <a href="traces.html" className="btn ghost">
              <Icon.ArrowRight style={{transform: "rotate(180deg)"}}/> Back to traces
            </a>
          </div>
        </div>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<Diff/>);
