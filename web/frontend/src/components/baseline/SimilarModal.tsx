'use client';

import { useState } from 'react';
import Modal from './Modal';
import OutcomeBadge, { type OutcomeKind } from '../OutcomeBadge';
import { getSimilar } from '@/lib/api';
import type { SimilarTracesResponse, TraceRef } from '@/lib/types';

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  if (outcome === 'regressed') return 'bad';
  if (outcome === 'additive') return 'novel';
  if (outcome === 'variance') return 'warn';
  if (outcome === 'succeeded') return 'ok';
  return 'neutral';
}

export default function SimilarModal({
  open,
  onClose,
  traces,
}: {
  open: boolean;
  onClose: () => void;
  traces: TraceRef[];
}) {
  const [traceIdx, setTraceIdx] = useState(0);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<SimilarTracesResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const trace = traces[traceIdx];

  const run = async () => {
    if (!trace) return;
    setRunning(true);
    setResult(null);
    setError(null);
    try {
      const res = await getSimilar(trace.id);
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'similar lookup failed');
    } finally {
      setRunning(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      eyebrow="money feature · 03"
      title="Find similar traces"
      width={680}
      footer={
        <div className="modal-foot-row">
          <span className="ad-mono ad-dim" style={{ fontSize: 12 }}>esc to close</span>
          <div className="modal-foot-actions">
            <button type="button" className="ad-btn" onClick={onClose}>Close</button>
            <button
              type="button"
              className="ad-btn primary"
              onClick={run}
              disabled={running || !trace}
            >
              {running ? 'Searching…' : 'Find similar'}
            </button>
          </div>
        </div>
      }
    >
      <p style={{ color: 'var(--muted)', marginBottom: 14 }}>
        Pick a trace from this baseline. agentdiff scores other traces in your
        workspace against it by shape, tool usage, and outcome.
      </p>

      <label className="cf-label">Source trace</label>
      <div className="cf-list">
        {traces.map((t, i) => (
          <button
            key={t.id}
            type="button"
            className={'cf-row' + (i === traceIdx ? ' active' : '')}
            onClick={() => setTraceIdx(i)}
          >
            <span className="ad-mono">{t.name}</span>
            <OutcomeBadge
              kind={outcomeKindFor(t.metadata?.outcome)}
              label={t.metadata?.outcome ?? 'unknown'}
            />
          </button>
        ))}
      </div>

      {(running || result || error) && (
        <div className="cf-result" style={{ marginTop: 18 }}>
          <div className="cf-result-head">
            <span className="ad-eyebrow">similar traces</span>
          </div>
          {running && (
            <div className="cf-running ad-mono">
              <span className="pg-spinner" /> scoring against workspace…
            </div>
          )}
          {error && (
            <p className="ad-mono" style={{ color: 'var(--bad)' }}>error: {error}</p>
          )}
          {result && result.matches.length === 0 && (
            <p className="ad-mono ad-dim">no similar traces found.</p>
          )}
          {result && result.matches.length > 0 && (
            <div className="sim-list">
              {result.matches.map((m) => (
                <div key={m.trace_id} className="sim-row">
                  <div style={{ flex: 1 }}>
                    <div className="sim-task">{m.name}</div>
                    <div className="sim-reason ad-mono">{m.trace_id}</div>
                  </div>
                  <div className="sim-meta">
                    <div className="sim-score ad-mono">
                      {(m.similarity_score * 100).toFixed(0)}%
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </Modal>
  );
}
