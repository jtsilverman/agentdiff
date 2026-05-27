/* about.jsx — /about long-scroll tour */

const { useState: useStateA, useEffect: useEffectA, useMemo: useMemoA } = React;

// ─── Concept icons (tiny line diagrams) ───
const ConceptIcon = {
  trace: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="3" fill="#22d3ee"/>
      <circle cx="22" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3"/>
      <circle cx="36" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3"/>
      <circle cx="50" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3"/>
      <circle cx="58" cy="20" r="3" fill="#22d3ee"/>
      <path d="M11 20h8M25 20h8M39 20h8M53 20h2" stroke="#22d3ee" strokeWidth="1.3" strokeDasharray="0"/>
    </svg>
  ),
  baseline: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee"/>
      <path d="M10 20 Q 32 8 56 8" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <path d="M10 20 Q 32 14 56 14" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <path d="M10 20 Q 32 20 56 20" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <path d="M10 20 Q 32 26 56 26" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <path d="M10 20 Q 32 32 56 32" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <circle cx="56" cy="8" r="2" fill="oklch(0.78 0.14 155)"/>
      <circle cx="56" cy="14" r="2" fill="oklch(0.78 0.14 155)"/>
      <circle cx="56" cy="20" r="2" fill="oklch(0.78 0.14 155)"/>
      <circle cx="56" cy="26" r="2" fill="oklch(0.78 0.14 155)"/>
      <circle cx="56" cy="32" r="2" fill="oklch(0.78 0.14 155)"/>
    </svg>
  ),
  strategy: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee"/>
      <path d="M10 20 L 26 12 L 42 12 L 56 12" stroke="oklch(0.78 0.14 155)" strokeWidth="2.5"/>
      <path d="M10 20 L 26 20 L 42 20 L 56 20" stroke="#22d3ee" strokeWidth="1.5"/>
      <path d="M10 20 L 26 28 L 42 28 L 56 28" stroke="oklch(0.72 0.14 290)" strokeWidth="1"/>
      <circle cx="56" cy="12" r="2.5" fill="oklch(0.78 0.14 155)"/>
      <circle cx="56" cy="20" r="2" fill="#22d3ee"/>
      <circle cx="56" cy="28" r="1.5" fill="oklch(0.72 0.14 290)"/>
    </svg>
  ),
  drift: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <line x1="6" x2="58" y1="20" y2="20" stroke="var(--border-hi)" strokeWidth="1" strokeDasharray="2 2"/>
      <path d="M6 20 L 22 20 L 32 16 L 42 22 L 58 18" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none"/>
      <path d="M6 20 L 22 20 L 32 26 L 42 34 L 58 34" stroke="oklch(0.68 0.18 25)" strokeWidth="1.5" fill="none"/>
      <circle cx="32" cy="20" r="3" fill="none" stroke="#22d3ee" strokeWidth="1.3"/>
    </svg>
  ),
  pathGraph: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee"/>
      <circle cx="24" cy="12" r="2" fill="#22d3ee"/>
      <circle cx="24" cy="28" r="2" fill="#22d3ee"/>
      <circle cx="42" cy="12" r="2.5" fill="#22d3ee"/>
      <circle cx="42" cy="28" r="2" fill="#22d3ee"/>
      <circle cx="58" cy="20" r="2.5" fill="#22d3ee"/>
      <path d="M10 19 L 22 13M10 21 L 22 27M26 12 L 40 12M26 28 L 40 28M44 13 L 56 19M44 27 L 56 21" stroke="#22d3ee" strokeWidth="1"/>
    </svg>
  ),
  overlay: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <path d="M6 28 Q 16 16 32 20 T 58 16" stroke="oklch(0.78 0.14 155)" strokeWidth="2" fill="none"/>
      <path d="M6 28 Q 16 22 32 24 T 58 28" stroke="#22d3ee" strokeWidth="2" fill="none" opacity=".8"/>
      <circle cx="32" cy="22" r="6" fill="none" stroke="#22d3ee" strokeWidth="1" strokeDasharray="2 2"/>
    </svg>
  ),
  counterfactual: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <path d="M6 20 L 24 20" stroke="#22d3ee" strokeWidth="1.8"/>
      <circle cx="24" cy="20" r="3.5" fill="none" stroke="#22d3ee" strokeWidth="1.5"/>
      <path d="M27 18 L 58 10" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" strokeDasharray="3 2"/>
      <path d="M27 22 L 58 30" stroke="oklch(0.72 0.14 290)" strokeWidth="1.5" strokeDasharray="3 2"/>
      <circle cx="58" cy="10" r="2" fill="oklch(0.78 0.14 155)"/>
      <circle cx="58" cy="30" r="2" fill="oklch(0.72 0.14 290)"/>
    </svg>
  ),
  editPrompt: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <rect x="6" y="10" width="20" height="20" rx="2" fill="none" stroke="var(--border-hi)" strokeWidth="1"/>
      <path d="M9 16h14M9 20h14M9 24h10" stroke="var(--border-hi)" strokeWidth="1"/>
      <path d="M30 20 L 38 20" stroke="#22d3ee" strokeWidth="1.5"/>
      <path d="M35 17 L 38 20 L 35 23" stroke="#22d3ee" strokeWidth="1.3" fill="none"/>
      <rect x="42" y="10" width="20" height="20" rx="2" fill="var(--accent-faint)" stroke="#22d3ee" strokeWidth="1"/>
      <path d="M45 16h14M45 20h14M45 24h10" stroke="#22d3ee" strokeWidth="1"/>
    </svg>
  ),
};

const CONCEPTS = [
  { id: "trace", title: "Trace", icon: "trace", desc: "A recording of a single agent run on a task — every tool call, every chain-of-thought step, every file the agent touched, with timestamps." },
  { id: "baseline", title: "Baseline", icon: "baseline", desc: "A group of traces all running the same task. The unit of comparison. You don't compare two single runs — you compare distributions." },
  { id: "strategy", title: "Strategy", icon: "strategy", desc: "A cluster of traces that took the same shape — same tools, same sequence. agentdiff auto-clusters runs into strategies and labels each one." },
  { id: "drift", title: "Drift", icon: "drift", desc: "When new runs take a different path than the baseline. Useful for detecting regressions after a prompt or model change. Drift is signed: better, worse, or sideways." },
  { id: "pathGraph", title: "Path graph", icon: "pathGraph", desc: "A network visualization of every run, overlaid. Nodes are tool calls; edges are transitions. Edge thickness = how many runs took that transition." },
  { id: "overlay", title: "Overlay", icon: "overlay", desc: "A one-trace-vs-baseline comparison. Layer a single run on top of the baseline graph and see exactly where it agreed and where it diverged." },
  { id: "counterfactual", title: "Counterfactual replay", icon: "counterfactual", desc: "\"What if the agent had gone left at step 5?\" Pick a trace, pick a divergence step, force a different decision, and watch the rest play out." },
  { id: "editPrompt", title: "Edit-prompt rewrite", icon: "editPrompt", desc: "\"What if the prompt had said X instead of Y?\" Rewrite the prompt; agentdiff re-runs the agent 5× and shows you how the distribution shifted." },
];

// ─── Tour mini-page diagrams ───
// We use stylized wireframe-y diagrams w/ callout pins. Lower fidelity, higher legibility.

const TourMini = {
  home: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6"/>
      {/* nav */}
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1"/>
      <circle cx="14" cy="11" r="3" fill="#22d3ee"/>
      <rect x="22" y="8" width="38" height="6" rx="1.5" fill="var(--muted)"/>
      {/* hero text */}
      <rect x="20" y="46" width="240" height="14" rx="2" fill="var(--fg-2)"/>
      <rect x="20" y="64" width="180" height="14" rx="2" fill="#22d3ee"/>
      <rect x="20" y="90" width="220" height="5" rx="1" fill="var(--border-hi)"/>
      <rect x="20" y="100" width="200" height="5" rx="1" fill="var(--border-hi)"/>
      {/* 3 cards */}
      <g>
        <rect x="20" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)"/>
        <rect x="130" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)"/>
        <rect x="240" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)"/>
        <circle cx="32" cy="146" r="2" fill="oklch(0.74 0.16 60)"/>
        <circle cx="142" cy="146" r="2" fill="oklch(0.68 0.18 25)"/>
        <circle cx="252" cy="146" r="2" fill="oklch(0.72 0.14 290)"/>
        <path d="M30 175 Q 50 160 75 160 T 105 160" stroke="oklch(0.78 0.14 155)" strokeWidth="1"/>
        <path d="M140 165 L 165 165 L 175 175 L 200 175 L 215 165" stroke="oklch(0.68 0.18 25)" strokeWidth="1" fill="none"/>
        <path d="M250 175 L 280 175 L 305 165 L 325 175" stroke="oklch(0.72 0.14 290)" strokeWidth="1" fill="none"/>
      </g>
    </svg>
  ),

  baseline: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6"/>
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1"/>
      {/* breadcrumb */}
      <rect x="14" y="32" width="60" height="4" rx="1" fill="var(--dim)"/>
      {/* title */}
      <rect x="14" y="46" width="180" height="10" rx="2" fill="var(--fg-2)"/>
      {/* callout */}
      <rect x="14" y="68" width="220" height="32" rx="3" fill="var(--surface)" stroke="var(--border)"/>
      <rect x="22" y="76" width="200" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="22" y="84" width="180" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="22" y="92" width="140" height="3" rx="1" fill="var(--border-hi)"/>
      {/* stats */}
      <rect x="245" y="46" width="100" height="54" rx="3" fill="var(--surface)" stroke="var(--border)"/>
      <line x1="295" y1="46" x2="295" y2="100" stroke="var(--border)" strokeWidth="1"/>
      <line x1="245" y1="73" x2="345" y2="73" stroke="var(--border)" strokeWidth="1"/>
      {/* trace list */}
      <rect x="14" y="115" width="220" height="95" rx="3" fill="var(--surface)" stroke="var(--border)"/>
      {[0,1,2,3].map(i => (
        <React.Fragment key={i}>
          <rect x="22" y={125 + i*20} width="40" height="4" rx="1" fill="var(--fg-2)"/>
          <circle cx={75} cy={127 + i*20} r="2.5" fill={["oklch(0.78 0.14 155)","#22d3ee","oklch(0.78 0.14 155)","oklch(0.68 0.18 25)"][i]}/>
          <rect x="90" y={125 + i*20} width="135" height="4" rx="1" fill="var(--border-hi)"/>
        </React.Fragment>
      ))}
      {/* path graph thumb */}
      <rect x="245" y="115" width="100" height="95" rx="3" fill="var(--bg)" stroke="var(--border)"/>
      <circle cx="255" cy="160" r="3" fill="#22d3ee"/>
      <circle cx="278" cy="145" r="2" fill="#22d3ee"/>
      <circle cx="278" cy="175" r="2" fill="#22d3ee"/>
      <circle cx="310" cy="145" r="2.5" fill="oklch(0.78 0.14 155)"/>
      <circle cx="310" cy="175" r="2" fill="oklch(0.72 0.14 290)"/>
      <circle cx="335" cy="160" r="3" fill="#22d3ee"/>
      <path d="M258 159 L 276 146 M 258 161 L 276 174 M 280 145 L 308 145 M 280 175 L 308 175 M 312 146 L 333 159 M 312 174 L 333 161" stroke="#22d3ee" strokeWidth="0.8"/>
    </svg>
  ),

  trace: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6"/>
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1"/>
      {/* sidebar */}
      <rect x="0" y="22" width="100" height="198" fill="var(--bg-2)"/>
      <line x1="100" x2="100" y1="22" y2="220" stroke="var(--border)" strokeWidth="1"/>
      {[0,1,2,3,4,5,6,7,8].map(i => (
        <React.Fragment key={i}>
          <circle cx="14" cy={40 + i*18} r="2" fill={i === 4 ? "#22d3ee" : "var(--muted)"}/>
          <rect x="22" y={38 + i*18} width={60 - i*3} height="3" rx="1" fill={i === 4 ? "var(--fg)" : "var(--border-hi)"}/>
        </React.Fragment>
      ))}
      {/* main pane */}
      <rect x="110" y="36" width="60" height="6" rx="1.5" fill="var(--fg-2)"/>
      <rect x="110" y="50" width="150" height="3" rx="1" fill="var(--border-hi)"/>
      {/* reasoning block */}
      <rect x="110" y="68" width="240" height="60" rx="3" fill="var(--surface)" stroke="var(--border)"/>
      <rect x="118" y="76" width="40" height="4" rx="1" fill="#22d3ee"/>
      <rect x="118" y="88" width="220" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="118" y="96" width="200" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="118" y="104" width="180" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="118" y="112" width="160" height="3" rx="1" fill="var(--border-hi)"/>
      {/* tool call */}
      <rect x="110" y="138" width="240" height="36" rx="3" fill="var(--bg-2)" stroke="#22d3ee" strokeWidth="0.8"/>
      <rect x="118" y="146" width="50" height="4" rx="1" fill="#22d3ee"/>
      <rect x="118" y="158" width="220" height="3" rx="1" fill="var(--border-hi)"/>
      {/* output */}
      <rect x="110" y="184" width="240" height="28" rx="3" fill="var(--surface)" stroke="var(--border)"/>
      <rect x="118" y="192" width="40" height="4" rx="1" fill="oklch(0.78 0.14 155)"/>
      <rect x="118" y="202" width="200" height="3" rx="1" fill="var(--border-hi)"/>
    </svg>
  ),

  counterfactual: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6"/>
      <rect x="20" y="20" width="320" height="180" rx="6" fill="var(--surface)" stroke="var(--border-hi)"/>
      {/* head */}
      <line x1="20" x2="340" y1="50" y2="50" stroke="var(--border)" strokeWidth="1"/>
      <rect x="32" y="32" width="100" height="6" rx="1.5" fill="var(--fg)"/>
      <circle cx="328" cy="36" r="6" fill="none" stroke="var(--muted)" strokeWidth="1"/>
      {/* split */}
      <rect x="32" y="62" width="140" height="80" rx="3" fill="var(--bg-2)" stroke="var(--border)"/>
      <rect x="40" y="70" width="80" height="3" rx="1" fill="var(--dim)"/>
      {[0,1,2,3].map(i => (
        <rect key={i} x="40" y={82 + i*14} width="124" height="9" rx="2" fill={i === 1 ? "var(--accent-faint)" : "var(--surface)"} stroke={i === 1 ? "#22d3ee" : "var(--border)"} strokeWidth="0.6"/>
      ))}
      <rect x="188" y="62" width="140" height="80" rx="3" fill="var(--bg-2)" stroke="var(--border)"/>
      <rect x="196" y="70" width="90" height="3" rx="1" fill="var(--dim)"/>
      {[0,1,2,3].map(i => (
        <rect key={i} x="196" y={82 + i*14} width="124" height="9" rx="2" fill={i === 2 ? "var(--accent-faint)" : "var(--surface)"} stroke={i === 2 ? "#22d3ee" : "var(--border)"} strokeWidth="0.6"/>
      ))}
      {/* result */}
      <rect x="32" y="152" width="296" height="38" rx="3" fill="var(--bg)" stroke="var(--border-hi)"/>
      <rect x="40" y="160" width="80" height="4" rx="1" fill="oklch(0.72 0.14 290)"/>
      <rect x="40" y="172" width="280" height="3" rx="1" fill="var(--border-hi)"/>
      <rect x="40" y="180" width="220" height="3" rx="1" fill="var(--border-hi)"/>
    </svg>
  ),

  editPrompt: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6"/>
      <rect x="20" y="20" width="320" height="180" rx="6" fill="var(--surface)" stroke="var(--border-hi)"/>
      <line x1="20" x2="340" y1="50" y2="50" stroke="var(--border)" strokeWidth="1"/>
      <rect x="32" y="32" width="100" height="6" rx="1.5" fill="var(--fg)"/>
      {/* textarea */}
      <rect x="32" y="62" width="296" height="60" rx="3" fill="var(--bg)" stroke="var(--border)"/>
      <rect x="40" y="72" width="270" height="3" rx="1" fill="var(--fg-2)"/>
      <rect x="40" y="80" width="280" height="3" rx="1" fill="var(--fg-2)"/>
      <rect x="40" y="88" width="200" height="3" rx="1" fill="#22d3ee"/>
      <rect x="40" y="96" width="240" height="3" rx="1" fill="var(--fg-2)"/>
      <rect x="40" y="104" width="180" height="3" rx="1" fill="var(--fg-2)"/>
      {/* bars */}
      <rect x="32" y="132" width="296" height="60" rx="3" fill="var(--bg-2)" stroke="var(--border)"/>
      {[
        ["oklch(0.78 0.14 155)", 200],
        ["#22d3ee", 110],
        ["oklch(0.72 0.14 290)", 60],
      ].map(([c, w], i) => (
        <React.Fragment key={i}>
          <rect x="40" y={144 + i*14} width="80" height="3" rx="1" fill={c}/>
          <rect x="130" y={143 + i*14} width="186" height="6" rx="2" fill="var(--surface)"/>
          <rect x="130" y={143 + i*14} width={w*0.95} height="6" rx="2" fill={c}/>
        </React.Fragment>
      ))}
    </svg>
  ),
};

const TOUR = [
  { id: "home", page: "/", title: "Home page", what: "The shop window. One sentence, three example baselines, link to this tour.", look: "Each example card uses a different visual — branching tree, overlaid timelines, fork. That's intentional. Cards previewing different things should LOOK different." },
  { id: "baseline", page: "/baselines/:id", title: "Baseline detail", what: "Where you'll spend 90% of your time. Task description, trace list, path graph, three action buttons.", look: "The path graph is overlay-first — every run shown at once with edge thickness encoding run count. Click any trace row to descend into a single-trace view." },
  { id: "trace", page: "/traces/:id", title: "Trace detail", what: "One run, top to bottom: context, reasoning, every tool call, the output.", look: "The sidebar is a step scrubber. Click a step on the left, the right pane shows what the agent was thinking and what it did at that moment." },
  { id: "counterfactual", page: "modal on baseline", title: "Counterfactual replay", what: "Pick a trace + a step where the agent decided something. agentdiff replays from there with a forced alternative.", look: "This is the 'what if' tool. The result shows up inline — same modal, no navigation away." },
  { id: "editPrompt", page: "modal on baseline", title: "Edit-prompt rewrite", what: "Rewrite the prompt; agentdiff re-runs the agent 5×. See how the path distribution shifted.", look: "The distribution bars at the bottom are the whole point. You're not looking at a new output — you're looking at how the population of behaviour moved." },
];

const MONEY = [
  {
    title: "Counterfactual replay",
    scenario: "Your agent shipped a bug-fix that worked. You want to know if it got lucky — or if it would have worked from any reasonable starting state.",
    how: "Pick the run that worked. At step 3, the agent decided to read the test file before the source. Force the other order. Replay.",
    tag: "what-if",
  },
  {
    title: "Edit-prompt rewrite",
    scenario: "You added one sentence to your system prompt. The next day, one in five runs is regressing on a task that used to be solid.",
    how: "Open the baseline. Click Edit prompt. Restore the missing sentence. Rerun. The distribution snaps back. You just bisected your prompt.",
    tag: "regression",
  },
  {
    title: "Similarity search",
    scenario: "Your agent is hung on a new task. You want to know if it has seen this shape of problem before — and what worked.",
    how: "From the baseline page, click Find similar. agentdiff pulls other baselines with the same toolchain, failure mode, or step distribution.",
    tag: "recall",
  },
  {
    title: "AI triage",
    scenario: "You have 200 baselines and you don't know which ones are actually drifting. You don't want to look at all of them.",
    how: "agentdiff ranks baselines by drift magnitude and surfaces only the ones whose distribution moved this week. One screen, ranked.",
    tag: "ranking",
  },
];

// MAIN
const About = () => {
  // Track which tour card is in view (for sticky index highlight)
  const [activeTour, setActiveTour] = useStateA(0);

  useEffectA(() => {
    const obs = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            const idx = Number(e.target.getAttribute("data-tour-idx"));
            setActiveTour(idx);
          }
        });
      },
      { rootMargin: "-40% 0px -50% 0px" }
    );
    document.querySelectorAll(".tour-card").forEach(el => obs.observe(el));
    return () => obs.disconnect();
  }, []);

  return (
    <>
      <Nav active="about"/>

      <main>
        {/* HERO */}
        <section className="ab-hero" data-screen-label="01 About hero">
          <div className="container">
            <span className="eyebrow">the full tour</span>
            <h1 className="ab-hero-title">
              agentdiff in <span style={{color: "var(--accent)", fontStyle: "italic", fontWeight: 300}}>~6 minutes</span>.
            </h1>
            <p className="ab-hero-sub">
              agentdiff records every step your coding agent takes — the prompt it received, the chain of thought it produced, the tools it called, the output it returned. Then it lets you compare runs of the same task across versions of your prompt, your model, and your codebase. This page walks through every concept, every screen, and the four features people care most about.
            </p>
            <div className="ab-hero-toc mono">
              <a href="#concepts">01 · concepts</a>
              <span className="dim">·</span>
              <a href="#tour">02 · page tour</a>
              <span className="dim">·</span>
              <a href="#money">03 · money features</a>
              <span className="dim">·</span>
              <a href="#start">04 · get started</a>
            </div>
          </div>
        </section>

        {/* CONCEPTS */}
        <section className="ab-section" id="concepts" data-screen-label="02 Concepts">
          <div className="container">
            <div className="ab-section-head">
              <div>
                <span className="eyebrow">01 · concepts</span>
                <h2>Eight ideas. Memorize these and the rest is obvious.</h2>
              </div>
            </div>
            <div className="concepts-grid">
              {CONCEPTS.map(c => {
                const Icon = ConceptIcon[c.icon];
                return (
                  <div key={c.id} className="concept-card">
                    <div className="concept-icon"><Icon/></div>
                    <h4 className="concept-title">{c.title}</h4>
                    <p className="concept-desc">{c.desc}</p>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        {/* TOUR */}
        <section className="ab-section" id="tour" data-screen-label="03 Tour">
          <div className="container">
            <div className="ab-section-head">
              <div>
                <span className="eyebrow">02 · page tour</span>
                <h2>What every screen looks like, and why.</h2>
              </div>
            </div>

            <div className="tour-layout">
              <aside className="tour-rail">
                <div className="tour-rail-inner">
                  <span className="eyebrow">pages</span>
                  <ol className="tour-list mono">
                    {TOUR.map((t, i) => (
                      <li key={t.id} className={i === activeTour ? "active" : ""}>
                        <a href={"#tour-" + t.id}>
                          <span className="tour-idx">{String(i+1).padStart(2,"0")}</span>
                          <span>{t.title}</span>
                        </a>
                      </li>
                    ))}
                  </ol>
                </div>
              </aside>

              <div className="tour-cards">
                {TOUR.map((t, i) => {
                  const Mini = TourMini[t.id];
                  return (
                    <article key={t.id} id={"tour-" + t.id} className="tour-card" data-tour-idx={i}>
                      <div className="tour-card-frame">
                        <div className="tour-card-chrome mono">
                          <span className="dot"/><span className="dot"/><span className="dot"/>
                          <span className="tour-card-url">agentdiff.dev{t.page}</span>
                        </div>
                        <div className="tour-card-mini"><Mini/></div>
                      </div>
                      <div className="tour-card-body">
                        <span className="mono dim" style={{fontSize: 11}}>{String(i+1).padStart(2,"0")} / {TOUR.length}</span>
                        <h3 className="tour-card-title">{t.title}</h3>
                        <div className="tour-card-pin">
                          <span className="mono dim">what it does</span>
                          <p>{t.what}</p>
                        </div>
                        <div className="tour-card-pin">
                          <span className="mono dim">what to look at</span>
                          <p>{t.look}</p>
                        </div>
                      </div>
                    </article>
                  );
                })}
              </div>
            </div>
          </div>
        </section>

        {/* MONEY FEATURES */}
        <section className="ab-section ab-money" id="money" data-screen-label="04 Money features">
          <div className="container">
            <div className="ab-section-head">
              <div>
                <span className="eyebrow">03 · money features</span>
                <h2>Four things people pay for.</h2>
              </div>
              <p className="section-sub" style={{maxWidth: 380}}>
                Each scenario is a real story we've heard from teams shipping agentic systems. The "how" is one click in agentdiff.
              </p>
            </div>

            <div className="money-deep">
              {MONEY.map((m, i) => (
                <article key={m.title} className="money-deep-card">
                  <div className="money-deep-num mono">{String(i+1).padStart(2,"0")}</div>
                  <div className="money-deep-body">
                    <div className="row" style={{gap: 8, marginBottom: 10}}>
                      <h3 className="money-deep-title">{m.title}</h3>
                      <span className="badge accent">{m.tag}</span>
                    </div>
                    <div className="money-deep-row">
                      <span className="eyebrow">scenario</span>
                      <p>{m.scenario}</p>
                    </div>
                    <div className="money-deep-row">
                      <span className="eyebrow">how agentdiff handles it</span>
                      <p>{m.how}</p>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="ab-cta" id="start">
          <div className="container">
            <div className="ab-cta-inner">
              <div>
                <span className="eyebrow">04 · get started</span>
                <h2 className="ab-cta-title">Now go try it.</h2>
                <p className="ab-cta-sub">Three baselines are loaded on the home page. Click into one, fan out the path graph, run a counterfactual. The whole loop takes about a minute.</p>
              </div>
              <div className="row" style={{gap: 10}}>
                <a href="index.html" className="btn primary">Back to home <Icon.ArrowRight/></a>
                <a href="baseline.html?id=auth-migration" className="btn">Skip to the regression example</a>
              </div>
            </div>
          </div>
        </section>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<About/>);
