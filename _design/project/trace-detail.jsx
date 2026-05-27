/* trace-detail.jsx — /traces/[id] full trace view
   "Stack trace viewer for AI agents." Header w/ metadata, sticky step list
   on the left, full transcript on the right with anchored sections.
*/

const { useState: useStateTD, useEffect: useEffectTD, useRef: useRefTD, useMemo: useMemoTD } = React;

const TRACES        = window.CORPUS_TRACES;
const TRACE_STEPS   = window.TRACE_STEPS;
const OUTCOME_KIND  = window.CORPUS_OUTCOME_KIND;
const OUTCOME_LABEL = window.CORPUS_OUTCOME_LABEL;

// ─── Step content helpers ───
const ROLE_LABEL = { user: "user", assistant: "assistant", tool_call: "tool call", tool_result: "tool result" };

function stepSummary(step) {
  if (!step) return "";
  if (step.tool_call) return `${step.tool_call.name}(${step.tool_call.args?.path || step.tool_call.args?.cmd || step.tool_call.args?.query || step.tool_call.args?.pattern || ""})`;
  if (step.tool_result) return `${step.tool_result.name} →`;
  if (step.content) return step.content.length > 60 ? step.content.slice(0, 60) + "…" : step.content;
  return "(empty)";
}

// Fallback: synthesize placeholder steps for traces without content
function getSteps(traceId, trace) {
  if (TRACE_STEPS && TRACE_STEPS[traceId]) return TRACE_STEPS[traceId];
  if (!trace) return [];
  const count = trace.step_count || 8;
  const steps = [{ role: "user", content: trace.metadata.task }];
  for (let i = 1; i < count - 1; i++) {
    const isCall = i % 2 === 1;
    steps.push(isCall
      ? { role: "tool_call", tool_call: { name: ["read_file","edit_file","shell","grep"][i % 4], args: { path: "file" + i } } }
      : { role: "tool_result", tool_result: { name: "read_file", output: "(placeholder — step content not recorded)", is_error: false } }
    );
  }
  if (trace.metadata.key_decision) {
    steps.push({ role: "assistant", content: trace.metadata.key_decision });
  }
  return steps;
}

// ─── Compressed step in the left rail ───
const StepRailItem = ({ step, idx, active, onClick }) => {
  const role = step.role;
  return (
    <a
      href={"#step-" + idx}
      className={"td-rail-step role-" + role + (active ? " active" : "")}
      onClick={(e) => { e.preventDefault(); onClick(idx); }}
    >
      <span className="td-rail-step-idx mono">{String(idx + 1).padStart(2, "0")}</span>
      <span className={"td-rail-step-dot role-" + role}/>
      <span className="td-rail-step-summary mono">{stepSummary(step)}</span>
    </a>
  );
};

// ─── Full step in the transcript ───
const TranscriptStep = ({ step, idx, anchorId }) => {
  const role = step.role;
  const tc = step.tool_call;
  const tr = step.tool_result;

  return (
    <article id={anchorId} className={"td-step role-" + role} data-step-idx={idx}>
      <div className="td-step-gutter">
        <span className="td-step-num mono">{String(idx + 1).padStart(2, "0")}</span>
        <div className={"td-step-rail-mark role-" + role}/>
      </div>

      <div className="td-step-body">
        <header className="td-step-head">
          <span className={"td-role-tag mono role-" + role}>{ROLE_LABEL[role]}</span>
          {tc && <span className="td-toolname mono">{tc.name}()</span>}
          {tr && (
            <>
              <span className="td-toolname mono">{tr.name}() →</span>
              {tr.is_error
                ? <span className="badge bad mono" style={{padding: "1px 6px", fontSize: 10}}>error</span>
                : <span className="badge ok mono" style={{padding: "1px 6px", fontSize: 10}}>ok</span>}
            </>
          )}
          <span className="td-step-anchor mono" title="anchor">#step-{idx + 1}</span>
        </header>

        {step.content && <p className="td-step-content">{step.content}</p>}

        {tc && tc.args && (
          <div className="td-args">
            <div className="td-args-head mono">arguments</div>
            {Object.entries(tc.args).map(([k, v]) => (
              <div key={k} className="td-arg">
                <span className="mono dim">{k}</span>
                <span className="mono td-arg-val">{String(v)}</span>
              </div>
            ))}
          </div>
        )}

        {tr && tr.output && (
          <div className={"td-output " + (tr.is_error ? "error" : "")}>
            <div className="td-output-head mono">
              <span>output</span>
              {tr.is_error && <span style={{color: "var(--bad)"}}>· error</span>}
            </div>
            <pre className="mono">{tr.output}</pre>
          </div>
        )}
      </div>
    </article>
  );
};

// ─── Tool sequence summary chip row (above transcript) ───
const ToolSequence = ({ steps }) => {
  const tools = steps
    .filter(s => s.tool_call)
    .map(s => s.tool_call.name);
  if (tools.length === 0) return null;
  return (
    <div className="td-toolseq">
      <span className="eyebrow">tool sequence · {tools.length} calls</span>
      <div className="td-toolseq-chain">
        {tools.map((t, i) => (
          <React.Fragment key={i}>
            <span className="td-toolseq-chip mono">{t}</span>
            {i < tools.length - 1 && <span className="td-toolseq-arrow mono">→</span>}
          </React.Fragment>
        ))}
      </div>
    </div>
  );
};

// ─── Metadata badges row ───
const MetadataBadges = ({ trace, steps }) => {
  const toolCount = steps.filter(s => s.tool_call).length;
  const errorCount = steps.filter(s => s.tool_result?.is_error).length;
  const turns = steps.filter(s => s.role === "assistant").length;
  return (
    <div className="td-meta-grid">
      <div className="td-meta">
        <span className="mono dim">outcome</span>
        <div className="td-meta-val">
          <OutcomeBadge kind={OUTCOME_KIND[trace.metadata.outcome]} label={OUTCOME_LABEL[trace.metadata.outcome]}/>
        </div>
      </div>
      <div className="td-meta">
        <span className="mono dim">baseline</span>
        <a href={"baseline.html?id=" + trace.baseline_id} className="td-meta-val mono td-meta-link">{trace.baseline_name}</a>
      </div>
      <div className="td-meta">
        <span className="mono dim">adapter</span>
        <span className="td-meta-val mono">{trace.adapter}</span>
      </div>
      <div className="td-meta">
        <span className="mono dim">steps</span>
        <span className="td-meta-val mono">{steps.length}</span>
      </div>
      <div className="td-meta">
        <span className="mono dim">tool calls</span>
        <span className="td-meta-val mono">{toolCount}</span>
      </div>
      <div className="td-meta">
        <span className="mono dim">assistant turns</span>
        <span className="td-meta-val mono">{turns}</span>
      </div>
      <div className="td-meta">
        <span className="mono dim">errors</span>
        <span className="td-meta-val mono" style={{color: errorCount > 0 ? "var(--bad)" : "var(--fg)"}}>{errorCount}</span>
      </div>
      <div className="td-meta">
        <span className="mono dim">recorded</span>
        <span className="td-meta-val mono">{window.corpusRelativeTime(trace.created_at)}</span>
      </div>
    </div>
  );
};

// ─── Main ───
const TraceDetail = () => {
  const params = new URLSearchParams(location.search);
  const id = params.get("id") || "t-bc92"; // default to a long trace so sticky rail is meaningful

  const trace = useMemoTD(() => window.corpusFindTrace(id), [id]);
  const steps = useMemoTD(() => getSteps(id, trace), [id, trace]);

  const [activeStep, setActiveStep] = useStateTD(0);
  const railRef = useRefTD(null);

  // Active step tracking. Defensive: real browsers fire scroll/IO; some embed
  // contexts pause those APIs, so we also fall back to setInterval polling.
  useEffectTD(() => {
    if (steps.length === 0) return;

    let lastScrollY = -1;
    let lastIdx = -1;

    const computeActive = () => {
      const sy = window.scrollY;
      if (sy === lastScrollY) return;
      lastScrollY = sy;
      const stepEls = document.querySelectorAll(".td-step");
      const threshold = 140;
      let activeIdx = 0;
      stepEls.forEach((el) => {
        const r = el.getBoundingClientRect();
        if (r.top - threshold <= 0) {
          activeIdx = Number(el.getAttribute("data-step-idx"));
        }
      });
      if (activeIdx !== lastIdx) {
        lastIdx = activeIdx;
        setActiveStep(activeIdx);
      }
    };

    // Primary: scroll listener (works in normal browsers)
    const onScroll = () => computeActive();
    window.addEventListener("scroll", onScroll, { passive: true });

    // Secondary: IntersectionObserver (also fires on layout shifts)
    const obs = new IntersectionObserver(() => computeActive(), {
      rootMargin: "0px 0px -50% 0px",
      threshold: [0, 0.1, 0.5, 1],
    });
    document.querySelectorAll(".td-step").forEach(el => obs.observe(el));

    // Fallback: setInterval polling for contexts that pause scroll/rAF events
    // (e.g. some sandboxed iframes). 60ms is light and feels live.
    const poll = setInterval(computeActive, 60);

    // Initial compute
    computeActive();

    return () => {
      window.removeEventListener("scroll", onScroll);
      obs.disconnect();
      clearInterval(poll);
    };
  }, [id, steps.length]);

  // Auto-scroll the active rail item into view when activeStep changes
  useEffectTD(() => {
    if (!railRef.current) return;
    const active = railRef.current.querySelector(".td-rail-step.active");
    if (active) {
      const rTop = railRef.current.scrollTop;
      const rH = railRef.current.clientHeight;
      const aTop = active.offsetTop;
      const aH = active.clientHeight;
      if (aTop < rTop + 20 || aTop + aH > rTop + rH - 20) {
        railRef.current.scrollTop = aTop - rH / 2 + aH / 2;
      }
    }
  }, [activeStep]);

  const jumpToStep = (idx) => {
    const target = document.getElementById("step-" + idx);
    if (target) {
      const top = target.getBoundingClientRect().top + window.scrollY - 80;
      window.scrollTo({ top, behavior: "smooth" });
    }
  };

  if (!trace) {
    return (
      <>
        <Nav active="traces"/>
        <main>
          <div className="container" style={{padding: "80px 0"}}>
            <div className="tr-empty">
              <h4>Trace not found</h4>
              <p>No trace with id <span className="mono">{id}</span> in the corpus.</p>
              <a href="traces.html" className="btn">Back to traces</a>
            </div>
          </div>
        </main>
        <Footer/>
      </>
    );
  }

  return (
    <>
      <Nav active="traces"/>

      <main>
        {/* ─── HERO ─── */}
        <header className="tr-hero td-hero" data-screen-label="01 Trace header">
          <div className="container">
            <div className="bl-breadcrumb mono">
              <a href="index.html" className="bl-crumb">home</a>
              <span className="dim">/</span>
              <a href="traces.html" className="bl-crumb">traces</a>
              <span className="dim">/</span>
              <span className="bl-crumb-current">{id}</span>
            </div>

            <div className="td-hero-grid">
              <div className="td-hero-main">
                <div className="row" style={{gap: 8, marginBottom: 16, flexWrap: "wrap"}}>
                  <span className="td-hero-name mono">{trace.name}</span>
                  <span className="mono dim" style={{fontSize: 13}}>· {trace.id}</span>
                </div>
                <h1 className="td-hero-title">{trace.metadata.task}</h1>
                {trace.metadata.key_decision && (
                  <div className="td-keydecision">
                    <span className="eyebrow">key decision</span>
                    <p>{trace.metadata.key_decision}</p>
                  </div>
                )}
              </div>

              <aside className="td-hero-actions">
                <a href={"diff.html?a=" + trace.id} className="btn">
                  <Icon.Diff/> Compare with another trace
                </a>
                <a href={"baseline.html?id=" + trace.baseline_id} className="btn ghost">
                  <Icon.Branch/> View baseline
                </a>
              </aside>
            </div>

            <MetadataBadges trace={trace} steps={steps}/>
          </div>
        </header>

        {/* ─── BODY ─── */}
        <div className="container td-body">
          {/* Sticky step list */}
          <aside className="td-rail">
            <div className="td-rail-inner" ref={railRef}>
              <div className="td-rail-head">
                <span className="eyebrow">step list</span>
                <span className="mono dim" style={{fontSize: 11}}>{steps.length}</span>
              </div>
              <div className="td-rail-steps">
                {steps.map((step, idx) => (
                  <StepRailItem
                    key={idx}
                    step={step}
                    idx={idx}
                    active={idx === activeStep}
                    onClick={jumpToStep}
                  />
                ))}
              </div>
            </div>
          </aside>

          {/* Transcript */}
          <section className="td-transcript">
            <ToolSequence steps={steps}/>

            <div className="td-transcript-head">
              <span className="eyebrow">transcript · {steps.length} steps</span>
              <span className="mono dim" style={{fontSize: 11}}>step {String(activeStep + 1).padStart(2, "0")} in view</span>
            </div>

            <div className="td-steps">
              {steps.map((step, idx) => (
                <TranscriptStep
                  key={idx}
                  step={step}
                  idx={idx}
                  anchorId={"step-" + idx}
                />
              ))}
            </div>

            <div className="td-foot">
              <a href="traces.html" className="btn ghost">
                <Icon.ArrowRight style={{transform: "rotate(180deg)"}}/> Back to traces
              </a>
              <span className="mono dim" style={{fontSize: 11}}>
                api · GET /api/traces/{trace.id}
              </span>
            </div>
          </section>
        </div>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<TraceDetail/>);
