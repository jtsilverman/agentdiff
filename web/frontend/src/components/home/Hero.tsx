import Link from 'next/link';
import { Fragment } from 'react';
import { ArrowRight } from '../Icons';

const STAGES = [
  { k: 'context', label: 'Context', sub: 'prompt + system' },
  { k: 'reasoning', label: 'Reasoning', sub: 'chain of thought' },
  { k: 'decision', label: 'Decision', sub: 'tool calls' },
  { k: 'output', label: 'Output', sub: 'final answer' },
];

const HeroFlow = () => (
  <div className="ad-hero-flow">
    {STAGES.map((s, i) => (
      <Fragment key={s.k}>
        <div className="ad-hero-stage">
          <div className="ad-hero-stage-dot" />
          <div className="ad-hero-stage-label">{s.label}</div>
          <div className="ad-hero-stage-sub">{s.sub}</div>
        </div>
        {i < STAGES.length - 1 && (
          <svg width="36" height="14" viewBox="0 0 36 14" fill="none" style={{ flexShrink: 0, opacity: 0.7 }}>
            <path d="M2 7 H30" stroke="var(--border-hi)" strokeWidth="1" strokeDasharray="2 3" />
            <path
              d="M28 3.5 L33 7 L28 10.5"
              stroke="var(--accent)"
              strokeWidth="1.2"
              fill="none"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </Fragment>
    ))}
  </div>
);

export default function Hero({ firstBaselineHref }: { firstBaselineHref: string }) {
  return (
    <section className="ad-hero">
      <div className="ad-container">
        <div className="ad-hero-eyebrow">
          <span className="ad-eyebrow">trace recorder for AI agents</span>
          <span className="ad-hero-status">
            <span className="ad-hero-status-dot ad-pulse" />
            recording 1,284 traces · 12 baselines this week
          </span>
        </div>

        <h1 className="ad-h1 ad-hero-title ad-fade-up">
          See how your AI
          <br />
          <span className="ad-italic">thinks.</span>
        </h1>

        <p className="ad-hero-sub ad-fade-up" style={{ animationDelay: '60ms' }}>
          agentdiff records every move your AI agent makes — what it saw, what it
          thought, what it decided, what it did. Compare runs of the same task to
          spot where strategies diverged or things broke.
        </p>

        <HeroFlow />

        <div className="ad-hero-actions">
          <Link href={firstBaselineHref} className="ad-btn primary">
            See an example <ArrowRight />
          </Link>
          <Link href="/about" className="ad-btn">
            Read the full tour
          </Link>
          <span className="ad-hero-install">
            <span className="ad-dim">$</span> npm i -g agentdiff
          </span>
        </div>

        <div className="ad-hero-fde ad-fade-up" style={{ animationDelay: '120ms' }}>
          <span className="ad-hero-fde-dot ad-pulse" aria-hidden="true" />
          <span className="ad-hero-fde-text">
            Built by{' '}
            <strong>Jake Silverman</strong> — forward-deployed engineer, available
            to drop into your codebase and ship the AI feature your team has been
            meaning to.{' '}
            <Link href="/about#about-jake" className="ad-hero-fde-link">
              What I do <ArrowRight />
            </Link>
          </span>
        </div>
      </div>
    </section>
  );
}
