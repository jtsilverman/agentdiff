'use client';

import PathGraph, { type PathGraphMode } from '../PathGraph';
import type {
  PathGraph as PathGraphData,
  OverlayResult,
  StrategyReport,
} from '@/lib/types';
import { type OutcomeKind } from '../OutcomeBadge';

interface ClusterEntry {
  label: string;
  count: number;
  cssVar: string;
}

const OUTCOME_TO_VAR: Record<OutcomeKind, string> = {
  ok: 'var(--ok)',
  warn: 'var(--warn)',
  bad: 'var(--bad)',
  novel: 'var(--novel)',
  accent: 'var(--accent)',
  neutral: 'var(--muted)',
};

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  if (outcome === 'regressed') return 'bad';
  if (outcome === 'additive') return 'novel';
  if (outcome === 'variance') return 'warn';
  if (outcome === 'succeeded') return 'ok';
  return 'neutral';
}

function dominantOutcomeKind(members: { metadata?: Record<string, string> }[]): OutcomeKind {
  const counts: Partial<Record<OutcomeKind, number>> = {};
  for (const m of members) {
    const k = outcomeKindFor(m.metadata?.outcome);
    counts[k] = (counts[k] ?? 0) + 1;
  }
  let best: OutcomeKind = 'neutral';
  let bestCount = -1;
  for (const [k, c] of Object.entries(counts) as [OutcomeKind, number][]) {
    if (c > bestCount) {
      best = k;
      bestCount = c;
    }
  }
  return best;
}

function deriveClusters(report: StrategyReport): ClusterEntry[] {
  const out: ClusterEntry[] = report.strategies.map((s, i) => {
    const kind = dominantOutcomeKind(s.members);
    const exemplarName = s.exemplar?.name ?? `strategy ${i}`;
    return {
      label: exemplarName,
      count: s.members.length,
      cssVar: OUTCOME_TO_VAR[kind],
    };
  });
  if (report.noise.length > 0) {
    out.push({
      label: 'noise / outliers',
      count: report.noise.length,
      cssVar: OUTCOME_TO_VAR.neutral,
    });
  }
  return out;
}

const MODES: { value: PathGraphMode; label: string }[] = [
  { value: 'overlay', label: 'overlay' },
  { value: 'heatmap-cost', label: 'heatmap: cost' },
  { value: 'heatmap-latency', label: 'heatmap: latency' },
];

export default function BaselinePathGraphCard({
  graph,
  overlay,
  overlayError,
  mode,
  onModeChange,
  selectedTraceId,
  report,
}: {
  graph: PathGraphData;
  overlay: OverlayResult | null;
  overlayError: string | null;
  mode: PathGraphMode;
  onModeChange: (m: PathGraphMode) => void;
  selectedTraceId: string | null;
  report: StrategyReport;
}) {
  const clusters = deriveClusters(report);
  return (
    <section className="bl-section">
      <div className="bl-section-head">
        <div>
          <span className="ad-eyebrow">path graph · all runs overlaid</span>
          <h2>How the runs flowed</h2>
        </div>
        <div className="bl-controls">
          <div className="seg">
            {MODES.map((m) => (
              <button
                key={m.value}
                className={'seg-btn' + (mode === m.value ? ' active' : '')}
                onClick={() => onModeChange(m.value)}
              >
                {m.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="pg-container">
        <PathGraph
          graph={graph}
          overlay={mode === 'overlay' ? overlay ?? undefined : undefined}
          mode={mode}
        />
      </div>

      <div className="pg-legend">
        <div className="pg-legend-rules ad-mono">
          <span><span className="pg-rule-thick" /> edge thickness = run count</span>
          <span><span className="pg-rule-size" /> node size = call frequency</span>
          <span><span className="pg-rule-color" /> color = outcome cluster</span>
        </div>
        <div className="pg-legend-clusters">
          {clusters.map((c, i) => (
            <div key={i} className="pg-cluster">
              <span className="pg-cluster-dot" style={{ background: c.cssVar }} />
              <span className="ad-mono" style={{ fontSize: 12 }}>{c.label}</span>
              <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>×{c.count}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bl-stats-row">
        <span className="ad-mono ad-dim" style={{ fontSize: 12 }}>
          {graph.stats.total_runs} runs · {graph.stats.branch_points} branch points
          {selectedTraceId && ` · overlay: ${selectedTraceId}`}
        </span>
        {overlayError && (
          <span className="ad-mono" style={{ fontSize: 12, color: 'var(--bad)' }}>
            overlay error: {overlayError}
          </span>
        )}
      </div>
    </section>
  );
}
