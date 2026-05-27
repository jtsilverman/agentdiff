/* home.jsx — agentdiff landing page */

const { useState: useStateH } = React;

// ─── Hero pipeline diagram (small, sits under headline) ───
const HeroFlow = () => {
  const stages = [
    { k: "context", label: "Context", sub: "prompt + system" },
    { k: "reasoning", label: "Reasoning", sub: "chain of thought" },
    { k: "decision", label: "Decision", sub: "tool calls" },
    { k: "output", label: "Output", sub: "final answer" },
  ];
  return (
    <div className="hero-flow">
      {stages.map((s, i) => (
        <React.Fragment key={s.k}>
          <div className="hero-stage">
            <div className="hero-stage-dot"/>
            <div className="hero-stage-label mono">{s.label}</div>
            <div className="hero-stage-sub mono">{s.sub}</div>
          </div>
          {i < stages.length - 1 && (
            <svg className="hero-stage-arrow" width="36" height="14" viewBox="0 0 36 14" fill="none">
              <path d="M2 7 H30" stroke="var(--border-hi)" strokeWidth="1" strokeDasharray="2 3"/>
              <path d="M28 3.5 L33 7 L28 10.5" stroke="var(--accent)" strokeWidth="1.2" fill="none" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          )}
        </React.Fragment>
      ))}
    </div>
  );
};

// ─── CARD 1 — Variance (branching tree, 3 strategies) ───
const VarianceGraph = () => (
  <svg className="card-graph" viewBox="0 0 320 180" fill="none">
    <defs>
      <linearGradient id="vg-fade" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0" stopColor="#22d3ee" stopOpacity="0"/>
        <stop offset=".5" stopColor="#22d3ee" stopOpacity=".25"/>
        <stop offset="1" stopColor="#22d3ee" stopOpacity="0"/>
      </linearGradient>
    </defs>
    {/* root */}
    <circle cx="40" cy="90" r="6" fill="#22d3ee"/>
    <circle cx="40" cy="90" r="12" fill="none" stroke="#22d3ee" strokeOpacity=".3" strokeWidth="1"/>
    <text x="40" y="116" textAnchor="middle" fontSize="9" fill="#8a8f99" fontFamily="Geist Mono">prompt</text>

    {/* branches — strategy A (flask-limiter) — 3 runs, thick */}
    <path d="M46 88 Q 90 50 130 50 T 230 50" stroke="oklch(0.78 0.14 155)" strokeWidth="3.5" strokeOpacity=".9"/>
    <circle cx="130" cy="50" r="4" fill="oklch(0.78 0.14 155)"/>
    <circle cx="180" cy="50" r="4" fill="oklch(0.78 0.14 155)"/>
    <circle cx="230" cy="50" r="4" fill="oklch(0.78 0.14 155)"/>
    <circle cx="270" cy="50" r="7" fill="oklch(0.78 0.14 155)"/>
    <text x="295" y="53" fontSize="10" fill="oklch(0.78 0.14 155)" fontFamily="Geist Mono">×3</text>

    {/* strategy B (hand-rolled) — 2 runs, medium */}
    <path d="M46 92 Q 90 100 130 100 T 230 100" stroke="#22d3ee" strokeWidth="2.2" strokeOpacity=".85"/>
    <circle cx="130" cy="100" r="3.5" fill="#22d3ee"/>
    <circle cx="180" cy="100" r="3.5" fill="#22d3ee"/>
    <circle cx="230" cy="100" r="3.5" fill="#22d3ee"/>
    <circle cx="270" cy="100" r="6" fill="#22d3ee"/>
    <text x="295" y="103" fontSize="10" fill="#22d3ee" fontFamily="Geist Mono">×2</text>

    {/* strategy C (redis direct) — 1 run, thin, novel */}
    <path d="M46 95 Q 80 150 140 150 T 270 150" stroke="oklch(0.72 0.14 290)" strokeWidth="1.4" strokeOpacity=".85" strokeDasharray="4 3"/>
    <circle cx="140" cy="150" r="3" fill="oklch(0.72 0.14 290)"/>
    <circle cx="200" cy="150" r="3" fill="oklch(0.72 0.14 290)"/>
    <circle cx="270" cy="150" r="5" fill="oklch(0.72 0.14 290)"/>
    <text x="295" y="153" fontSize="10" fill="oklch(0.72 0.14 290)" fontFamily="Geist Mono">×1</text>
  </svg>
);

// ─── CARD 2 — Regression (two overlaid timelines) ───
const RegressionGraph = () => {
  // tool call sequence (12 steps); the regressed run forks at step 7
  const baseY = 90, hi = 30;
  const steps = 11;
  const xs = Array.from({length: steps}, (_, i) => 28 + i * 26);
  return (
    <svg className="card-graph" viewBox="0 0 320 180" fill="none">
      {/* baseline guide line */}
      <line x1="20" x2="305" y1={baseY} y2={baseY} stroke="var(--border)" strokeWidth="1" strokeDasharray="2 3"/>
      <text x="20" y={baseY - 6} fontSize="9" fill="var(--dim)" fontFamily="Geist Mono">baseline (v1.2 prompt)</text>

      {/* GOOD run (v1.2) — flat success */}
      <polyline
        points={xs.map((x, i) => `${x},${baseY - (i === 3 || i === 8 ? hi*0.3 : i === 5 ? hi*0.5 : 0)}`).join(" ")}
        stroke="oklch(0.78 0.14 155)" strokeWidth="2" fill="none" strokeLinejoin="round"
      />
      {xs.map((x, i) => (
        <circle key={"g"+i} cx={x} cy={baseY - (i === 3 || i === 8 ? hi*0.3 : i === 5 ? hi*0.5 : 0)} r="3" fill="oklch(0.78 0.14 155)"/>
      ))}

      {/* REGRESSED run (v1.3) — diverges at step 7, climbs and hits failure */}
      <polyline
        points={[
          `${xs[0]},${baseY+8}`,
          `${xs[1]},${baseY+8}`,
          `${xs[2]},${baseY+10}`,
          `${xs[3]},${baseY+12}`,
          `${xs[4]},${baseY+15}`,
          `${xs[5]},${baseY+18}`,
          `${xs[6]},${baseY+24}`,
          `${xs[7]},${baseY+40}`,
          `${xs[8]},${baseY+55}`,
          `${xs[9]},${baseY+65}`,
          `${xs[10]},${baseY+72}`,
        ].join(" ")}
        stroke="oklch(0.68 0.18 25)" strokeWidth="2" fill="none" strokeLinejoin="round"
      />
      {[
        [xs[0], baseY+8],[xs[1], baseY+8],[xs[2], baseY+10],[xs[3], baseY+12],
        [xs[4], baseY+15],[xs[5], baseY+18],[xs[6], baseY+24],[xs[7], baseY+40],
        [xs[8], baseY+55],[xs[9], baseY+65],[xs[10], baseY+72],
      ].map(([x,y], i) => (
        <circle key={"r"+i} cx={x} cy={y} r="3" fill="oklch(0.68 0.18 25)"/>
      ))}

      {/* divergence marker */}
      <line x1={xs[6]} x2={xs[6]} y1={baseY-30} y2={baseY+45} stroke="var(--accent)" strokeWidth="1" strokeDasharray="2 2" opacity=".7"/>
      <text x={xs[6]+4} y={baseY-22} fontSize="9" fill="var(--accent)" fontFamily="Geist Mono">divergence</text>

      {/* end markers */}
      <circle cx={xs[10]} cy={baseY} r="5" fill="oklch(0.78 0.14 155)"/>
      <circle cx={xs[10]} cy={baseY+72} r="5" fill="oklch(0.68 0.18 25)"/>

      {/* legend tags */}
      <text x="265" y={baseY+12} fontSize="9" fill="oklch(0.78 0.14 155)" fontFamily="Geist Mono">v1.2 ok</text>
      <text x="245" y={baseY+88} fontSize="9" fill="oklch(0.68 0.18 25)" fontFamily="Geist Mono">v1.3 hung</text>
    </svg>
  );
};

// ─── CARD 3 — Novel strategy (fork pattern) ───
const NovelGraph = () => (
  <svg className="card-graph" viewBox="0 0 320 180" fill="none">
    {/* main path - the obvious one (4 runs) */}
    <path d="M30 100 L 90 100 L 130 100 L 170 100 L 210 100 L 260 100 L 290 100"
          stroke="#22d3ee" strokeWidth="3" opacity=".9"/>
    {[90, 130, 170, 210, 260].map((x, i) => (
      <circle key={"main"+i} cx={x} cy={100} r={i === 4 ? 6 : 4} fill="#22d3ee"/>
    ))}
    <text x="30" y="118" fontSize="9" fill="var(--muted)" fontFamily="Geist Mono">obvious path · ×4</text>

    {/* the fork — branches up at step 3, discovers a parallel approach */}
    <path d="M130 99 Q 145 75 175 55 T 235 35 Q 260 30 290 50"
          stroke="oklch(0.72 0.14 290)" strokeWidth="2" fill="none" strokeDasharray="0"/>
    <circle cx="130" cy="100" r="6" fill="none" stroke="oklch(0.72 0.14 290)" strokeWidth="1.5"/>
    {[[175, 55], [235, 35]].map(([x,y], i) => (
      <circle key={"f"+i} cx={x} cy={y} r="3.5" fill="oklch(0.72 0.14 290)"/>
    ))}
    <circle cx={290} cy={50} r={6} fill="oklch(0.72 0.14 290)"/>

    {/* novel label */}
    <text x="155" y="40" fontSize="10" fill="oklch(0.72 0.14 290)" fontFamily="Geist Mono">run #4 took a different turn</text>

    {/* tool-call labels */}
    <text x="86" y="135" fontSize="8" fill="var(--dim)" fontFamily="Geist Mono">read_log</text>
    <text x="120" y="135" fontSize="8" fill="var(--dim)" fontFamily="Geist Mono">grep_tests</text>
    <text x="160" y="135" fontSize="8" fill="var(--dim)" fontFamily="Geist Mono">run_test</text>
    <text x="200" y="135" fontSize="8" fill="var(--dim)" fontFamily="Geist Mono">edit_file</text>
    <text x="245" y="135" fontSize="8" fill="var(--dim)" fontFamily="Geist Mono">commit</text>
  </svg>
);

// ─── Example card ───
const ExampleCard = ({ data }) => (
  <a className="ex-card" href={"baseline.html?id=" + data.id}>
    <div className="ex-card-header">
      <div className="row" style={{gap: 8}}>
        <span className="mono" style={{fontSize: 11, color: "var(--dim)"}}>baseline / {data.id}</span>
      </div>
      <OutcomeBadge kind={data.outcomeKind} label={data.outcomeLabel}/>
    </div>

    <div className="ex-card-graph">
      {data.id === "rate-limit" && <VarianceGraph/>}
      {data.id === "auth-migration" && <RegressionGraph/>}
      {data.id === "flaky-test" && <NovelGraph/>}
    </div>

    <div className="ex-card-body">
      <h3 className="ex-card-title">{data.task}</h3>
      <p className="ex-card-hook">{data.hook}</p>

      <div className="ex-card-meta mono">
        <span>{data.runs} runs</span>
        <span className="dim">·</span>
        <span>{data.steps} avg steps</span>
        <span className="dim">·</span>
        <span>{data.duration}</span>
      </div>
    </div>

    <div className="ex-card-foot">
      <span className="mono" style={{fontSize: 12, color: "var(--muted)"}}>Open baseline</span>
      <span className="ex-card-arrow"><Icon.ArrowRight/></span>
    </div>
  </a>
);

const EXAMPLES = [
  {
    id: "rate-limit",
    task: "Add rate limiting to a Flask endpoint",
    hook: "5 runs, 3 strategies — flask-limiter, hand-rolled middleware, and one run that went straight to Redis.",
    outcomeKind: "warn",
    outcomeLabel: "high variance",
    runs: 5, steps: "11.2", duration: "~38s/run",
  },
  {
    id: "auth-migration",
    task: "Migrate auth from JWT to session cookies",
    hook: "Same task, two prompt versions. v1.3 hung on token refresh — see where it diverged from baseline.",
    outcomeKind: "bad",
    outcomeLabel: "regression",
    runs: 6, steps: "17.5", duration: "~1m12s/run",
  },
  {
    id: "flaky-test",
    task: "Debug intermittent test failure in CI",
    hook: "Four runs went after the obvious race condition. The fifth checked the runner's clock — and was right.",
    outcomeKind: "novel",
    outcomeLabel: "novel strategy",
    runs: 5, steps: "23.4", duration: "~2m04s/run",
  },
];

// ─── Home ───
const Home = () => {
  return (
    <>
      <Nav active="home"/>

      <main>
        {/* HERO */}
        <section className="hero" data-screen-label="01 Hero">
          <div className="container">
            <div className="hero-eyebrow">
              <span className="eyebrow">trace recorder for coding agents</span>
              <span className="hero-status mono">
                <span className="hero-status-dot pulse"/>
                recording 1,284 traces · 12 baselines this week
              </span>
            </div>

            <h1 className="hero-title fade-up">
              See how your coding<br/>
              agent <span className="hero-italic">thinks.</span>
            </h1>

            <p className="hero-sub fade-up" style={{animationDelay: "60ms"}}>
              agentdiff records every step Claude Code, Cursor, and friends take —<br/>
              then lets you compare runs of the same task, side by side.
            </p>

            <HeroFlow/>

            <div className="hero-actions">
              <a href="baseline.html?id=rate-limit" className="btn primary">Open an example baseline <Icon.ArrowRight/></a>
              <a href="about.html" className="btn">Read the full tour</a>
              <span className="hero-install mono">
                <span className="dim">$</span> npm i -g agentdiff
              </span>
            </div>
          </div>
        </section>

        {/* EXAMPLES */}
        <section className="examples" data-screen-label="02 Examples">
          <div className="container">
            <div className="section-head">
              <div>
                <span className="eyebrow">try these examples</span>
                <h2 className="section-title">Three baselines. Three different stories.</h2>
              </div>
              <p className="section-sub">
                Each baseline is a real recording of the same agent task run multiple times. Click any card to dig in.
              </p>
            </div>

            <div className="ex-grid">
              {EXAMPLES.map(e => <ExampleCard key={e.id} data={e}/>)}
            </div>

            <a href="about.html" className="examples-foot">
              <span className="mono dim">→</span>
              <span>Read the full tour — how traces, baselines, drift, and counterfactuals fit together</span>
              <Icon.ArrowRight/>
            </a>
          </div>
        </section>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<Home/>);
