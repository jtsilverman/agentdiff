'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import SiteFooter from '@/components/SiteFooter';
import { ArrowRight } from '@/components/Icons';
import {
  ConceptIcon,
  TourMini,
  type ConceptIconKey,
  type TourMiniKey,
} from '@/components/about/AboutIcons';

const CONCEPTS: { id: string; title: string; icon: ConceptIconKey; desc: string }[] = [
  { id: 'trace', title: 'Trace', icon: 'trace', desc: 'A recording of a single agent run on a task — every tool call, every chain-of-thought step, every file the agent touched, with timestamps.' },
  { id: 'baseline', title: 'Baseline', icon: 'baseline', desc: "A group of traces all running the same task. The unit of comparison. You don't compare two single runs — you compare distributions." },
  { id: 'strategy', title: 'Strategy', icon: 'strategy', desc: 'A cluster of traces that took the same shape — same tools, same sequence. agentdiff auto-clusters runs into strategies and labels each one.' },
  { id: 'drift', title: 'Drift', icon: 'drift', desc: 'When new runs take a different path than the baseline. Useful for detecting regressions after a prompt or model change. Drift is signed: better, worse, or sideways.' },
  { id: 'pathGraph', title: 'Path graph', icon: 'pathGraph', desc: 'A network visualization of every run, overlaid. Nodes are tool calls; edges are transitions. Edge thickness = how many runs took that transition.' },
  { id: 'overlay', title: 'Overlay', icon: 'overlay', desc: 'A one-trace-vs-baseline comparison. Layer a single run on top of the baseline graph and see exactly where it agreed and where it diverged.' },
  { id: 'counterfactual', title: 'Counterfactual replay', icon: 'counterfactual', desc: '"What if the agent had gone left at step 5?" Pick a trace, pick a divergence step, force a different decision, and watch the rest play out.' },
  { id: 'editPrompt', title: 'Edit-prompt rewrite', icon: 'editPrompt', desc: '"What if the prompt had said X instead of Y?" Rewrite the prompt; agentdiff re-runs the agent 5× and shows you how the distribution shifted.' },
];

const TOUR: { id: TourMiniKey; page: string; title: string; what: string; look: string }[] = [
  { id: 'home', page: '/', title: 'Home page', what: 'The shop window. One sentence, three example baselines, link to this tour.', look: "Each example card uses a different visual — branching tree, overlaid timelines, fork. That's intentional. Cards previewing different things should LOOK different." },
  { id: 'baseline', page: '/baselines/:id', title: 'Baseline detail', what: "Where you'll spend 90% of your time. Task description, trace list, path graph, three action buttons.", look: 'The path graph is overlay-first — every run shown at once with edge thickness encoding run count. Click any trace row to descend into a single-trace view.' },
  { id: 'trace', page: '/traces/:id', title: 'Trace detail', what: 'One run, top to bottom: context, reasoning, every tool call, the output.', look: 'The sidebar is a step scrubber. Click a step on the left, the right pane shows what the agent was thinking and what it did at that moment.' },
  { id: 'counterfactual', page: 'modal on baseline', title: 'Counterfactual replay', what: 'Pick a trace + a step where the agent decided something. agentdiff replays from there with a forced alternative.', look: "This is the 'what if' tool. The result shows up inline — same modal, no navigation away." },
  { id: 'editPrompt', page: 'modal on baseline', title: 'Edit-prompt rewrite', what: 'Rewrite the prompt; agentdiff re-runs the agent 5×. See how the path distribution shifted.', look: "The distribution bars at the bottom are the whole point. You're not looking at a new output — you're looking at how the population of behaviour moved." },
];

const MONEY: { title: string; scenario: string; how: string; tag: string }[] = [
  {
    title: 'Counterfactual replay',
    scenario: 'Your agent shipped a bug-fix that worked. You want to know if it got lucky — or if it would have worked from any reasonable starting state.',
    how: 'Pick the run that worked. At step 3, the agent decided to read the test file before the source. Force the other order. Replay.',
    tag: 'what-if',
  },
  {
    title: 'Edit-prompt rewrite',
    scenario: 'You added one sentence to your system prompt. The next day, one in five runs is regressing on a task that used to be solid.',
    how: 'Open the baseline. Click Edit prompt. Restore the missing sentence. Rerun. The distribution snaps back. You just bisected your prompt.',
    tag: 'regression',
  },
  {
    title: 'Similarity search',
    scenario: 'Your agent is hung on a new task. You want to know if it has seen this shape of problem before — and what worked.',
    how: 'From the baseline page, click Find similar. agentdiff pulls other baselines with the same toolchain, failure mode, or step distribution.',
    tag: 'recall',
  },
  {
    title: 'AI triage',
    scenario: "You have 200 baselines and you don't know which ones are actually drifting. You don't want to look at all of them.",
    how: 'agentdiff ranks baselines by drift magnitude and surfaces only the ones whose distribution moved this week. One screen, ranked.',
    tag: 'ranking',
  },
];

export default function AboutPage() {
  const [activeTour, setActiveTour] = useState(0);

  useEffect(() => {
    const obs = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            const idx = Number((e.target as HTMLElement).getAttribute('data-tour-idx'));
            setActiveTour(idx);
          }
        });
      },
      { rootMargin: '-40% 0px -50% 0px' },
    );
    document.querySelectorAll('.tour-card').forEach((el) => obs.observe(el));
    return () => obs.disconnect();
  }, []);

  return (
    <>
      <section className="ab-hero">
        <div className="ad-container">
          <span className="ad-eyebrow">the full tour</span>
          <h1 className="ab-hero-title">
            agentdiff in{' '}
            <span style={{ color: 'var(--accent)', fontStyle: 'italic', fontWeight: 300 }}>
              ~6 minutes
            </span>
            .
          </h1>
          <p className="ab-hero-sub">
            agentdiff records every step your coding agent takes — the prompt it received, the
            chain of thought it produced, the tools it called, the output it returned. Then it
            lets you compare runs of the same task across versions of your prompt, your model,
            and your codebase. This page walks through every concept, every screen, and the
            four features people care most about.
          </p>
          <div className="ab-hero-toc">
            <a href="#concepts">01 · concepts</a>
            <span className="ad-dim">·</span>
            <a href="#tour">02 · page tour</a>
            <span className="ad-dim">·</span>
            <a href="#money">03 · money features</a>
            <span className="ad-dim">·</span>
            <a href="#start">04 · get started</a>
            <span className="ad-dim">·</span>
            <a href="#about-jake">05 · about jake</a>
          </div>
        </div>
      </section>

      <section className="ab-section" id="concepts">
        <div className="ad-container">
          <div className="ab-section-head">
            <div>
              <span className="ad-eyebrow">01 · concepts</span>
              <h2>Eight ideas. Memorize these and the rest is obvious.</h2>
            </div>
          </div>
          <div className="concepts-grid">
            {CONCEPTS.map((c) => {
              const Icon = ConceptIcon[c.icon];
              return (
                <div key={c.id} className="concept-card">
                  <div className="concept-icon">
                    <Icon />
                  </div>
                  <h4 className="concept-title">{c.title}</h4>
                  <p className="concept-desc">{c.desc}</p>
                </div>
              );
            })}
          </div>
        </div>
      </section>

      <section className="ab-section" id="tour">
        <div className="ad-container">
          <div className="ab-section-head">
            <div>
              <span className="ad-eyebrow">02 · page tour</span>
              <h2>What every screen looks like, and why.</h2>
            </div>
          </div>

          <div className="tour-layout">
            <aside className="tour-rail">
              <div className="tour-rail-inner">
                <span className="ad-eyebrow">pages</span>
                <ol className="tour-list">
                  {TOUR.map((t, i) => (
                    <li key={t.id} className={i === activeTour ? 'active' : ''}>
                      <a href={`#tour-${t.id}`}>
                        <span className="tour-idx">{String(i + 1).padStart(2, '0')}</span>
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
                  <article
                    key={t.id}
                    id={`tour-${t.id}`}
                    className="tour-card"
                    data-tour-idx={i}
                  >
                    <div className="tour-card-frame">
                      <div className="tour-card-chrome">
                        <span className="dot" />
                        <span className="dot" />
                        <span className="dot" />
                        <span className="tour-card-url">agentdiff.dev{t.page}</span>
                      </div>
                      <div className="tour-card-mini">
                        <Mini />
                      </div>
                    </div>
                    <div className="tour-card-body">
                      <span className="ad-dim" style={{ fontSize: 11, fontFamily: 'var(--font-mono)' }}>
                        {String(i + 1).padStart(2, '0')} / {TOUR.length}
                      </span>
                      <h3 className="tour-card-title">{t.title}</h3>
                      <div className="tour-card-pin">
                        <span className="ad-dim" style={{ fontFamily: 'var(--font-mono)' }}>what it does</span>
                        <p>{t.what}</p>
                      </div>
                      <div className="tour-card-pin">
                        <span className="ad-dim" style={{ fontFamily: 'var(--font-mono)' }}>what to look at</span>
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

      <section className="ab-section ab-money" id="money">
        <div className="ad-container">
          <div className="ab-section-head">
            <div>
              <span className="ad-eyebrow">03 · money features</span>
              <h2>Four things people pay for.</h2>
            </div>
            <p className="ab-section-sub">
              Each scenario is a real story we've heard from teams shipping agentic systems.
              The "how" is one click in agentdiff.
            </p>
          </div>

          <div className="money-deep">
            {MONEY.map((m, i) => (
              <article key={m.title} className="money-deep-card">
                <div className="money-deep-num">{String(i + 1).padStart(2, '0')}</div>
                <div className="money-deep-body">
                  <div className="money-deep-headrow">
                    <h3 className="money-deep-title">{m.title}</h3>
                    <span className="money-deep-badge">{m.tag}</span>
                  </div>
                  <div className="money-deep-row">
                    <span className="ad-eyebrow">scenario</span>
                    <p>{m.scenario}</p>
                  </div>
                  <div className="money-deep-row">
                    <span className="ad-eyebrow">how agentdiff handles it</span>
                    <p>{m.how}</p>
                  </div>
                </div>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="ab-cta" id="start">
        <div className="ad-container">
          <div className="ab-cta-inner">
            <div>
              <span className="ad-eyebrow">04 · get started</span>
              <h2 className="ab-cta-title">Now go try it.</h2>
              <p className="ab-cta-sub">
                Three baselines are loaded on the home page. Click into one, fan out the path
                graph, run a counterfactual. The whole loop takes about a minute.
              </p>
            </div>
            <div className="ab-cta-actions">
              <Link href="/" className="ad-btn primary">
                Back to home <ArrowRight />
              </Link>
              <Link href="/" className="ad-btn">
                Skip to the regression example
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className="ab-section ab-about-jake" id="about-jake">
        <div className="ad-container">
          <div className="ab-section-head">
            <div>
              <span className="ad-eyebrow">05 · about jake</span>
              <h2>
                I built this. I&apos;m{' '}
                <span style={{ color: 'var(--accent)', fontStyle: 'italic', fontWeight: 300 }}>
                  available
                </span>{' '}
                to build for your team.
              </h2>
            </div>
            <p className="ab-section-sub">
              agentdiff is a worked example of what I ship when given a focused
              AI-engineering problem. The same approach drops into your codebase.
            </p>
          </div>

          <div className="aj-grid">
            <div className="aj-col">
              <span className="ad-eyebrow">what I do</span>
              <ul className="aj-list">
                <li>
                  Drop into AI engineering teams as a forward-deployed engineer —
                  same git branch, same PR queue, same standup as your engineers.
                  Not a contractor on the side.
                </li>
                <li>
                  Ship the AI feature your team scoped three months ago and never
                  built. Coding-agent eval pipelines, retrieval rewrites, internal
                  debug tools, the unfun glue that ships product.
                </li>
                <li>
                  Move in days and weeks, not quarters. agentdiff itself is one
                  worked example — a debug tool for AI agents, built end to end
                  and shipped to a hosted demo, source open.
                </li>
              </ul>
            </div>

            <div className="aj-col">
              <span className="ad-eyebrow">credentials</span>
              <ul className="aj-list aj-creds">
                <li>
                  Enterprise data infrastructure for <strong>$100M+</strong> duty
                  drawback programs on Azure Databricks and Microsoft Fabric.
                </li>
                <li>
                  Production AI agents — including an Executive Assistant
                  deployed under her own identity inside a private equity office.
                </li>
                <li>
                  <strong>North America winner</strong>, PwC Transfer Pricing AI
                  competition.
                </li>
              </ul>
            </div>
          </div>

          <div className="aj-cta">
            <span className="ad-eyebrow">get in touch</span>
            <div className="aj-cta-row">
              <a
                href="mailto:jakesilverman.pro@gmail.com"
                className="ad-btn primary"
              >
                Email Jake <ArrowRight />
              </a>
              <a
                href="https://www.linkedin.com/in/jacob-silverman1/"
                target="_blank"
                rel="noreferrer"
                className="ad-btn"
              >
                LinkedIn
              </a>
              <a
                href="https://github.com/jtsilverman"
                target="_blank"
                rel="noreferrer"
                className="ad-btn"
              >
                GitHub
              </a>
              <span className="aj-cta-hint ad-mono ad-dim">
                fastest reply: email
              </span>
            </div>
          </div>
        </div>
      </section>

      <SiteFooter />
    </>
  );
}
