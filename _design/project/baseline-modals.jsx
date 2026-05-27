/* baseline-modals.jsx — Counterfactual, Edit prompt, Find similar */

const { useState: useStateM, useEffect: useEffectM } = React;

const Modal = ({ open, onClose, title, eyebrow, width = 720, children, footer }) => {
  useEffectM(() => {
    const onKey = (e) => { if (e.key === "Escape" && open) onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);
  if (!open) return null;
  return (
    <div className="modal-scrim" onClick={onClose}>
      <div className="modal" style={{maxWidth: width}} onClick={e => e.stopPropagation()}>
        <header className="modal-head">
          <div>
            <span className="eyebrow">{eyebrow}</span>
            <h3 style={{marginTop: 6}}>{title}</h3>
          </div>
          <button className="btn ghost" onClick={onClose}><Icon.Close/></button>
        </header>
        <div className="modal-body">{children}</div>
        {footer && <footer className="modal-foot">{footer}</footer>}
      </div>
    </div>
  );
};

// ─── Counterfactual replay ───
const CounterfactualModal = ({ open, onClose, baseline }) => {
  const [traceIdx, setTraceIdx] = useStateM(0);
  const [divergeStep, setDivergeStep] = useStateM(3);
  const [running, setRunning] = useStateM(false);
  const [result, setResult] = useStateM(null);

  const trace = baseline.traces[traceIdx];
  const steps = ["read_file:app.py", "search:flask rate limit", "shell:pip install flask-limiter", "edit:app.py @limiter", "shell:pytest", "done"];

  const run = () => {
    setRunning(true);
    setResult(null);
    setTimeout(() => {
      setRunning(false);
      setResult({
        outcome: divergeStep <= 2 ? "novel" : "ok",
        steps: 9 + Math.floor(Math.random() * 4),
        delta: divergeStep <= 2 ? "Found a faster path: skipped library install and wrote a 12-line middleware" : "Reached same outcome via slight reordering",
        path: ["read_file:app.py", divergeStep === 3 ? "edit:middleware.py" : "search:flask rate limit", divergeStep === 3 ? "edit:redis_client.py" : "shell:pip install flask-limiter", "edit:app.py", "shell:pytest", "done"],
      });
    }, 1400);
  };

  return (
    <Modal open={open} onClose={onClose} eyebrow="money feature · 01" title="Run counterfactual" width={780}
      footer={
        <div className="row" style={{gap: 10, width: "100%", justifyContent: "space-between"}}>
          <span className="mono dim" style={{fontSize: 12}}>↵ to run · esc to close</span>
          <div className="row" style={{gap: 8}}>
            <button className="btn" onClick={onClose}>Cancel</button>
            <button className="btn primary" onClick={run} disabled={running}>
              {running ? "Replaying…" : <>Run counterfactual <Icon.Play/></>}
            </button>
          </div>
        </div>
      }>
      <p style={{color: "var(--muted)", marginBottom: 18}}>
        Pick a trace and a step. agentdiff will replay the agent from that point with the same context but force a different first decision — then show you what would have happened.
      </p>

      <div className="cf-grid">
        <div className="cf-col">
          <label className="cf-label">Source trace</label>
          <div className="cf-list">
            {baseline.traces.map((t, i) => (
              <button key={t.id} className={"cf-row" + (i === traceIdx ? " active" : "")} onClick={() => setTraceIdx(i)}>
                <span className="mono">{t.name}</span>
                <OutcomeBadge kind={t.outcome === "ok" ? "ok" : t.outcome === "bad" ? "bad" : t.outcome === "warn" ? "warn" : "novel"} label={t.outcomeLabel}/>
              </button>
            ))}
          </div>
        </div>

        <div className="cf-col">
          <label className="cf-label">Divergence step</label>
          <div className="cf-steps">
            {steps.map((s, i) => (
              <button key={i} className={"cf-step" + (i === divergeStep ? " active" : "") + (i < divergeStep ? " before" : "")} onClick={() => setDivergeStep(i)}>
                <span className="cf-step-idx mono">{String(i+1).padStart(2,"0")}</span>
                <span className="cf-step-label mono">{s}</span>
                {i === divergeStep && <span className="cf-step-tag mono">↳ fork</span>}
              </button>
            ))}
          </div>
        </div>
      </div>

      {(running || result) && (
        <div className="cf-result">
          <div className="cf-result-head">
            <span className="eyebrow">counterfactual result</span>
            {result && <OutcomeBadge kind={result.outcome} label={result.outcome === "novel" ? "novel · faster" : "converged"}/>}
          </div>
          {running ? (
            <div className="cf-running mono">
              <span className="pg-spinner"/>
              replaying from step {divergeStep + 1} · simulating {trace.name}…
            </div>
          ) : result && (
            <>
              <p className="cf-result-summary">{result.delta}</p>
              <div className="cf-result-path mono">
                {result.path.map((s, i) => (
                  <React.Fragment key={i}>
                    <span className={"cf-result-step" + (i === divergeStep ? " fork" : "")}>{s}</span>
                    {i < result.path.length - 1 && <span className="dim">→</span>}
                  </React.Fragment>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </Modal>
  );
};

// ─── Edit prompt ───
const EditPromptModal = ({ open, onClose, baseline }) => {
  const [prompt, setPrompt] = useStateM(baseline.prompt);
  const [running, setRunning] = useStateM(false);
  const [result, setResult] = useStateM(null);

  const reset = () => setPrompt(baseline.prompt);
  const run = () => {
    setRunning(true);
    setResult(null);
    setTimeout(() => {
      setRunning(false);
      setResult({
        outcomes: [
          { strategy: "flask-limiter", count: 1, kind: "ok" },
          { strategy: "redis lua", count: 2, kind: "novel" },
          { strategy: "hand-rolled", count: 2, kind: "ok" },
        ],
        delta: "+1.4 avg steps · variance ↑ · 2/5 runs chose Redis (vs 1/5 before)",
      });
    }, 1600);
  };

  return (
    <Modal open={open} onClose={onClose} eyebrow="money feature · 02" title="Edit the prompt" width={840}
      footer={
        <div className="row" style={{gap: 10, width: "100%", justifyContent: "space-between"}}>
          <button className="btn ghost" onClick={reset}>Reset to original</button>
          <div className="row" style={{gap: 8}}>
            <button className="btn" onClick={onClose}>Cancel</button>
            <button className="btn primary" onClick={run} disabled={running || prompt === baseline.prompt}>
              {running ? "Re-running…" : <>Rerun 5 traces <Icon.Play/></>}
            </button>
          </div>
        </div>
      }>
      <p style={{color: "var(--muted)", marginBottom: 14}}>
        Rewrite the prompt. agentdiff will re-run the agent 5 times with the new prompt and show you how the path distribution shifted.
      </p>

      <div className="ep-diff-head">
        <span className="eyebrow">prompt · editable</span>
        <span className="mono dim" style={{fontSize: 11}}>{prompt.length} chars · {prompt.split(/\s+/).length} words</span>
      </div>
      <textarea className="ep-textarea mono" value={prompt} onChange={e => setPrompt(e.target.value)} rows={6}/>

      {prompt !== baseline.prompt && (
        <div className="ep-hint mono">
          <Icon.Edit/> prompt changed — click "Rerun" to see how the agent behaves with this version
        </div>
      )}

      {(running || result) && (
        <div className="cf-result">
          <div className="cf-result-head">
            <span className="eyebrow">rewrite result · 5 new runs</span>
          </div>
          {running ? (
            <div className="cf-running mono">
              <span className="pg-spinner"/>
              re-running with new prompt · 5 traces in flight…
            </div>
          ) : result && (
            <>
              <p className="cf-result-summary">{result.delta}</p>
              <div className="ep-bars">
                {result.outcomes.map(o => (
                  <div key={o.strategy} className="ep-bar">
                    <div className="ep-bar-head">
                      <OutcomeBadge kind={o.kind} label={o.strategy}/>
                      <span className="mono dim">{o.count}/5</span>
                    </div>
                    <div className="ep-bar-track">
                      <div className={"ep-bar-fill " + o.kind} style={{width: (o.count / 5 * 100) + "%"}}/>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </Modal>
  );
};

// ─── Find similar ───
const SimilarModal = ({ open, onClose, baseline }) => {
  const items = [
    { id: "bl-api-key-rotation", task: "Rotate API keys without downtime", reason: "Same shape: small Flask service, choose-a-library decision", score: 0.94, runs: 4 },
    { id: "bl-cors-debug", task: "Debug CORS preflight on /api/upload", reason: "Both have a clear failure mode + 3 plausible fixes", score: 0.88, runs: 5 },
    { id: "bl-redis-cache", task: "Add Redis-backed caching to /products", reason: "Library-vs-hand-roll branching pattern", score: 0.81, runs: 6 },
    { id: "bl-celery-retry", task: "Make a Celery task idempotent", reason: "Same model, same toolchain, similar step count distribution", score: 0.77, runs: 5 },
  ];

  return (
    <Modal open={open} onClose={onClose} eyebrow="money feature · 03" title="Find similar traces" width={680}
      footer={
        <div className="row" style={{gap: 10, width: "100%", justifyContent: "flex-end"}}>
          <button className="btn" onClick={onClose}>Close</button>
        </div>
      }>
      <p style={{color: "var(--muted)", marginBottom: 14}}>
        Other baselines in your workspace with similar agent behaviour — same shape, same failure mode, or same toolchain. Click to compare.
      </p>

      <div className="sim-list">
        {items.map(it => (
          <a key={it.id} className="sim-row" href="#">
            <div style={{flex: 1}}>
              <div className="sim-task">{it.task}</div>
              <div className="sim-reason mono">{it.reason}</div>
            </div>
            <div className="sim-meta">
              <div className="sim-score mono">{(it.score * 100).toFixed(0)}%</div>
              <div className="sim-runs mono">{it.runs} runs</div>
            </div>
            <Icon.ChevronRight/>
          </a>
        ))}
      </div>
    </Modal>
  );
};

Object.assign(window, { CounterfactualModal, EditPromptModal, SimilarModal, Modal });
