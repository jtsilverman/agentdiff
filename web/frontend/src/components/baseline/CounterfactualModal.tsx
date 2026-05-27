'use client';

import { useState } from 'react';
import Modal from './Modal';
import OutcomeBadge, { type OutcomeKind } from '../OutcomeBadge';
import CounterfactualGraph from '../CounterfactualGraph';
import { runCounterfactual } from '@/lib/api';
import type { CounterfactualResponse, TraceRef } from '@/lib/types';

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  if (outcome === 'regressed') return 'bad';
  if (outcome === 'additive') return 'novel';
  if (outcome === 'variance') return 'warn';
  if (outcome === 'succeeded') return 'ok';
  return 'neutral';
}

export default function CounterfactualModal({
  open,
  onClose,
  traces,
}: {
  open: boolean;
  onClose: () => void;
  traces: TraceRef[];
}) {
  const [traceIdx, setTraceIdx] = useState(0);
  const [stepIndex, setStepIndex] = useState(0);
  const [modifiedInput, setModifiedInput] = useState('');
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<CounterfactualResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const trace = traces[traceIdx];
  const stepCount = trace?.step_count ?? 0;

  const run = async () => {
    if (!trace) return;
    setRunning(true);
    setResult(null);
    setError(null);
    try {
      const res = await runCounterfactual(trace.id, stepIndex, modifiedInput.trim());
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'counterfactual failed');
    } finally {
      setRunning(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      eyebrow="money feature · 01"
      title="Run counterfactual"
      width={780}
      footer={
        <div className="modal-foot-row">
          <span className="ad-mono ad-dim" style={{ fontSize: 12 }}>esc to close</span>
          <div className="modal-foot-actions">
            <button type="button" className="ad-btn" onClick={onClose}>Cancel</button>
            <button
              type="button"
              className="ad-btn primary"
              onClick={run}
              disabled={running || !modifiedInput.trim()}
            >
              {running ? 'Replaying…' : 'Run counterfactual'}
            </button>
          </div>
        </div>
      }
    >
      <p style={{ color: 'var(--muted)', marginBottom: 18 }}>
        Pick a trace and a step. agentdiff will replay the agent from that point with
        the same context but force a different first decision — then show you what
        would have happened.
      </p>

      <div className="cf-grid">
        <div className="cf-col">
          <label className="cf-label">Source trace</label>
          <div className="cf-list">
            {traces.map((t, i) => (
              <button
                key={t.id}
                type="button"
                className={'cf-row' + (i === traceIdx ? ' active' : '')}
                onClick={() => {
                  setTraceIdx(i);
                  setStepIndex(0);
                }}
              >
                <span className="ad-mono">{t.name}</span>
                <OutcomeBadge
                  kind={outcomeKindFor(t.metadata?.outcome)}
                  label={t.metadata?.outcome ?? 'unknown'}
                />
              </button>
            ))}
          </div>
        </div>

        <div className="cf-col">
          <label className="cf-label" htmlFor="cf-step">Divergence step</label>
          <select
            id="cf-step"
            className="select"
            value={stepIndex}
            onChange={(e) => setStepIndex(Number(e.target.value))}
            disabled={stepCount === 0}
          >
            {Array.from({ length: Math.max(stepCount, 1) }, (_, i) => (
              <option key={i} value={i}>
                step {String(i + 1).padStart(2, '0')} of {stepCount}
              </option>
            ))}
          </select>
          <label className="cf-label" style={{ marginTop: 14 }} htmlFor="cf-input">
            Modified input
          </label>
          <textarea
            id="cf-input"
            className="ep-textarea"
            placeholder="Modified input for this step"
            value={modifiedInput}
            onChange={(e) => setModifiedInput(e.target.value)}
            rows={3}
          />
        </div>
      </div>

      {(running || result || error) && (
        <div className="cf-result">
          <div className="cf-result-head">
            <span className="ad-eyebrow">counterfactual result</span>
          </div>
          {running && (
            <div className="cf-running ad-mono">
              <span className="pg-spinner" />
              replaying from step {stepIndex + 1} · simulating {trace?.name}…
            </div>
          )}
          {error && (
            <p className="ad-mono" style={{ color: 'var(--bad)' }}>error: {error}</p>
          )}
          {result && (
            <>
              <p className="cf-result-summary ad-mono ad-dim">
                new trace · {result.new_trace_id}
              </p>
              <CounterfactualGraph comparison={result.comparison} />
            </>
          )}
        </div>
      )}
    </Modal>
  );
}
