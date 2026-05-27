'use client';

import { useMemo, useState } from 'react';
import MetadataBadges from '../MetadataBadges';
import OutcomeBadge, { type OutcomeKind } from '../OutcomeBadge';
import type { TraceRef } from '@/lib/types';

type FilterKind = 'all' | 'succeeded' | 'variance' | 'regressed' | 'additive';
type SortKind = 'default' | 'steps' | 'outcome';

const FILTERS: FilterKind[] = ['all', 'succeeded', 'variance', 'regressed', 'additive'];

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  if (outcome === 'regressed') return 'bad';
  if (outcome === 'additive') return 'novel';
  if (outcome === 'variance') return 'warn';
  if (outcome === 'succeeded') return 'ok';
  return 'neutral';
}

function rowMetadata(trace: TraceRef): Record<string, string> {
  const meta: Record<string, string> = {};
  if (typeof trace.step_count === 'number') {
    meta.steps = String(trace.step_count);
  }
  const outcome = trace.metadata?.outcome;
  if (outcome) meta.outcome = outcome;
  return meta;
}

export default function TraceList({
  traces,
  selectedTraceId,
  onSelect,
}: {
  traces: TraceRef[];
  selectedTraceId: string | null;
  onSelect: (id: string) => void;
}) {
  const [filter, setFilter] = useState<FilterKind>('all');
  const [sortBy, setSortBy] = useState<SortKind>('default');

  const filtered = useMemo(() => {
    let rows = traces;
    if (filter !== 'all') {
      rows = rows.filter((t) => t.metadata?.outcome === filter);
    }
    if (sortBy === 'steps') {
      rows = [...rows].sort((a, b) => (a.step_count ?? 0) - (b.step_count ?? 0));
    } else if (sortBy === 'outcome') {
      rows = [...rows].sort((a, b) =>
        (a.metadata?.outcome ?? '').localeCompare(b.metadata?.outcome ?? ''),
      );
    }
    return rows;
  }, [traces, filter, sortBy]);

  return (
    <section className="bl-section">
      <div className="bl-section-head">
        <div>
          <span className="ad-eyebrow">runs · {traces.length}</span>
          <h2>{traces.length} runs of this task</h2>
        </div>
        <div className="bl-controls">
          <div className="seg">
            {FILTERS.map((f) => (
              <button
                key={f}
                className={'seg-btn' + (filter === f ? ' active' : '')}
                onClick={() => setFilter(f)}
              >
                {f}
              </button>
            ))}
          </div>
          <select
            className="select"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value as SortKind)}
          >
            <option value="default">sort: run order</option>
            <option value="steps">sort: step count</option>
            <option value="outcome">sort: outcome</option>
          </select>
        </div>
      </div>

      <div className="tl-table">
        <div className="tl-row tl-head ad-mono">
          <span className="tl-cell tl-cell-id">trace</span>
          <span className="tl-cell tl-cell-out">outcome</span>
          <span className="tl-cell tl-cell-steps">steps</span>
          <span className="tl-cell tl-cell-meta">metadata</span>
        </div>

        {filtered.map((t) => {
          const selected = t.id === selectedTraceId;
          const outcome = t.metadata?.outcome;
          return (
            <button
              type="button"
              key={t.id}
              onClick={() => onSelect(t.id)}
              className={'tl-row tl-row-button' + (selected ? ' tl-row-selected' : '')}
            >
              <span className="tl-cell tl-cell-id">
                <span className="tl-trace-name">{t.name}</span>
                <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>
                  {t.id}
                </span>
              </span>
              <span className="tl-cell tl-cell-out">
                <OutcomeBadge
                  kind={outcomeKindFor(outcome)}
                  label={outcome ?? 'unknown'}
                />
              </span>
              <span className="tl-cell tl-cell-steps ad-mono">
                {t.step_count ?? '—'}
              </span>
              <span className="tl-cell tl-cell-meta">
                <MetadataBadges metadata={rowMetadata(t)} />
              </span>
            </button>
          );
        })}

        {filtered.length === 0 && (
          <div className="tl-empty ad-mono">No runs match this filter.</div>
        )}
      </div>
    </section>
  );
}
