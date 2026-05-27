import Link from 'next/link';
import { ArrowRight } from '../Icons';
import { NovelGraph, RegressionGraph, VarianceGraph } from './Graphs';
import type { BaselineSummary } from '@/lib/types';

type OutcomeKind = 'warn' | 'bad' | 'novel';

interface Example {
  slug: string;
  task: string;
  hook: string;
  outcomeKind: OutcomeKind;
  outcomeLabel: string;
  runs: number;
  steps: string;
  duration: string;
  Graph: () => JSX.Element;
  /**
   * Substring matched against actual baseline names from the API so the card
   * deep-links to a real baseline. Falls back to whichever baseline shares
   * its index when no match is found.
   */
  matchHint: string;
}

const EXAMPLES: Example[] = [
  {
    slug: 'variance',
    task: 'Add a "Buy Now" button to the pricing page',
    hook: 'Five attempts. Three used the design system’s button, two built one from scratch — and their checkout flows don’t behave the same way.',
    outcomeKind: 'warn',
    outcomeLabel: 'wide variance',
    runs: 5,
    steps: '11.2',
    duration: '~38s/run',
    Graph: VarianceGraph,
    matchHint: 'variance',
  },
  {
    slug: 'regression',
    task: 'Diagnose yesterday’s site outage',
    hook: 'Same task, different prompt versions. The newer prompt skipped the deploy log — and missed where things actually broke.',
    outcomeKind: 'bad',
    outcomeLabel: 'regression',
    runs: 6,
    steps: '17.5',
    duration: '~1m12s/run',
    Graph: RegressionGraph,
    matchHint: 'regression',
  },
  {
    slug: 'novel',
    task: 'Build a weekly report on customer signups',
    hook: 'Four runs queried the database directly. The fifth found an existing dashboard already tracking it — and saved hours of work.',
    outcomeKind: 'novel',
    outcomeLabel: 'novel strategy',
    runs: 5,
    steps: '23.4',
    duration: '~2m04s/run',
    Graph: NovelGraph,
    matchHint: 'novel',
  },
];

function resolveHref(example: Example, index: number, baselines: BaselineSummary[]): string {
  const matched = baselines.find((b) =>
    b.name.toLowerCase().includes(example.matchHint),
  );
  const fallback = baselines[index];
  const target = matched ?? fallback;
  return target ? `/baselines/${target.id}` : '#';
}

function OutcomeBadge({ kind, label }: { kind: OutcomeKind; label: string }) {
  return (
    <span className={`ad-badge ${kind}`}>
      <span className="dot" />
      {label}
    </span>
  );
}

function ExampleCard({
  example,
  href,
}: {
  example: Example;
  href: string;
}) {
  const { Graph } = example;
  return (
    <Link href={href} className="ad-ex-card">
      <div className="ad-ex-card-header">
        <span className="ad-mono" style={{ fontSize: 11, color: 'var(--dim)' }}>
          baseline / {example.slug}
        </span>
        <OutcomeBadge kind={example.outcomeKind} label={example.outcomeLabel} />
      </div>

      <div className="ad-ex-card-graph">
        <Graph />
      </div>

      <div className="ad-ex-card-body">
        <h3 className="ad-ex-card-title">{example.task}</h3>
        <p className="ad-ex-card-hook">{example.hook}</p>
        <div className="ad-ex-card-meta">
          <span>{example.runs} runs</span>
          <span className="ad-dim">·</span>
          <span>{example.steps} avg steps</span>
          <span className="ad-dim">·</span>
          <span>{example.duration}</span>
        </div>
      </div>

      <div className="ad-ex-card-foot">
        <span>Open baseline</span>
        <span className="ad-ex-card-arrow">
          <ArrowRight />
        </span>
      </div>
    </Link>
  );
}

export default function Examples({ baselines }: { baselines: BaselineSummary[] }) {
  return (
    <section className="ad-examples">
      <div className="ad-container">
        <div className="ad-section-head">
          <div>
            <span className="ad-eyebrow">try these examples</span>
            <h2 className="ad-h2 ad-section-title">
              Three baselines. Three different stories.
            </h2>
          </div>
          <p className="ad-section-sub">
            Each baseline is a real recording of the same agent task run multiple
            times. Click any card to dig in.
          </p>
        </div>

        <div className="ad-ex-grid">
          {EXAMPLES.map((example, i) => (
            <ExampleCard
              key={example.slug}
              example={example}
              href={resolveHref(example, i, baselines)}
            />
          ))}
        </div>

        <Link href="/about" className="ad-examples-foot">
          <span className="ad-arrow-glyph">→</span>
          <span>
            Read the full tour — how traces, baselines, drift, and counterfactuals
            fit together
          </span>
          <ArrowRight />
        </Link>
      </div>
    </section>
  );
}

export { EXAMPLES, resolveHref };
