import Link from 'next/link';
import OutcomeBadge, { type OutcomeKind } from '../OutcomeBadge';
import type { TraceRef } from '@/lib/types';

interface BaselineHeaderProps {
  baselineId: string;
  baselineName: string;
  description: string | null;
  traces: TraceRef[];
  strategiesCount: number;
}

function deriveOutcome(traces: TraceRef[]): { kind: OutcomeKind; label: string } {
  const outcomes = traces.map((t) => t.metadata?.outcome ?? '');
  if (outcomes.includes('regressed')) return { kind: 'bad', label: 'regression' };
  if (outcomes.includes('additive')) return { kind: 'novel', label: 'novel strategy' };
  if (outcomes.includes('variance')) return { kind: 'warn', label: 'wide variance' };
  return { kind: 'ok', label: 'stable' };
}

function deriveDriftLabel(kind: OutcomeKind): string {
  if (kind === 'bad') return 'high';
  if (kind === 'novel') return 'novel';
  if (kind === 'warn') return 'med';
  return 'low';
}

function avgSteps(traces: TraceRef[]): string {
  const counts = traces
    .map((t) => t.step_count)
    .filter((n): n is number => typeof n === 'number');
  if (counts.length === 0) return '—';
  const mean = counts.reduce((a, b) => a + b, 0) / counts.length;
  return mean.toFixed(1);
}

export default function BaselineHeader({
  baselineId,
  baselineName,
  description,
  traces,
  strategiesCount,
}: BaselineHeaderProps) {
  const task = traces[0]?.metadata?.task ?? baselineName;
  const outcome = deriveOutcome(traces);
  const drift = deriveDriftLabel(outcome.kind);
  const driftVar =
    outcome.kind === 'bad'
      ? 'var(--bad)'
      : outcome.kind === 'novel'
        ? 'var(--novel)'
        : outcome.kind === 'warn'
          ? 'var(--warn)'
          : 'var(--ok)';

  return (
    <header className="bl-header">
      <div className="bl-breadcrumb ad-mono">
        <Link href="/" className="bl-crumb">home</Link>
        <span className="ad-dim">/</span>
        <span className="bl-crumb">baselines</span>
        <span className="ad-dim">/</span>
        <span className="bl-crumb-current">{baselineId}</span>
      </div>

      <div className="bl-head-grid">
        <div className="bl-head-main">
          <div className="bl-badges-row">
            <OutcomeBadge kind={outcome.kind} label={outcome.label} />
            <span className="ad-badge neutral ad-mono">{traces.length} runs</span>
            <span className="ad-badge neutral ad-mono">avg {avgSteps(traces)} steps</span>
          </div>

          <h1 className="bl-title">{task}</h1>

          {description && (
            <div className="bl-callout">
              <div className="bl-callout-head">
                <span className="ad-eyebrow">task description</span>
                <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>
                  baseline / {baselineName}
                </span>
              </div>
              <div className="bl-callout-body">
                <p>{description}</p>
              </div>
            </div>
          )}
        </div>

        <aside className="bl-head-side">
          <div className="bl-stat">
            <span className="ad-mono ad-dim">runs</span>
            <span className="bl-stat-val">{traces.length}</span>
          </div>
          <div className="bl-stat">
            <span className="ad-mono ad-dim">avg steps</span>
            <span className="bl-stat-val">{avgSteps(traces)}</span>
          </div>
          <div className="bl-stat">
            <span className="ad-mono ad-dim">strategies</span>
            <span className="bl-stat-val">{strategiesCount}</span>
          </div>
          <div className="bl-stat">
            <span className="ad-mono ad-dim">drift</span>
            <span className="bl-stat-val" style={{ color: driftVar }}>
              {drift}
            </span>
          </div>
        </aside>
      </div>
    </header>
  );
}
