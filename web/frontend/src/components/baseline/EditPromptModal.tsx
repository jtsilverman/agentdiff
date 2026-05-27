'use client';

import { useState } from 'react';
import Modal from './Modal';
import CounterfactualGraph from '../CounterfactualGraph';
import OutcomeBadge, { type OutcomeKind } from '../OutcomeBadge';
import { editPrompt } from '@/lib/api';
import type { EditPromptResponse, TraceRef } from '@/lib/types';

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  if (outcome === 'regressed') return 'bad';
  if (outcome === 'additive') return 'novel';
  if (outcome === 'variance') return 'warn';
  if (outcome === 'succeeded') return 'ok';
  return 'neutral';
}

export default function EditPromptModal({
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
  const [prompt, setPrompt] = useState('');
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<EditPromptResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const trace = traces[traceIdx];
  const stepCount = trace?.step_count ?? 0;

  const run = async () => {
    if (!trace) return;
    setRunning(true);
    setResult(null);
    setError(null);
    try {
      const res = await editPrompt(trace.id, stepIndex, prompt.trim());
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'edit-prompt failed');
    } finally {
      setRunning(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      eyebrow="money feature · 02"
      title="Edit the prompt"
      width={840}
      footer={
        <div className="modal-foot-row">
          <button
            type="button"
            className="ad-btn"
            onClick={() => setPrompt('')}
          >
            Clear
          </button>
          <div className="modal-foot-actions">
            <button type="button" className="ad-btn" onClick={onClose}>Cancel</button>
            <button
              type="button"
              className="ad-btn primary"
              onClick={run}
              disabled={running || !prompt.trim()}
            >
              {running ? 'Re-running…' : 'Rerun trace'}
            </button>
          </div>
        </div>
      }
    >
      <p style={{ color: 'var(--muted)', marginBottom: 14 }}>
        Rewrite the system or step prompt for the chosen trace. agentdiff replays
        the agent with the new prompt; the resulting path is overlaid against the
        original so you can see how the behaviour shifted.
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
          <label className="cf-label" htmlFor="ep-step">Step to rewrite</label>
          <select
            id="ep-step"
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
        </div>
      </div>

      <div className="ep-diff-head">
        <span className="ad-eyebrow">new prompt</span>
        <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>
          {prompt.length} chars
        </span>
      </div>
      <textarea
        className="ep-textarea"
        placeholder="Rewrite the prompt for this step"
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        rows={6}
      />

      {(running || result || error) && (
        <div className="cf-result">
          <div className="cf-result-head">
            <span className="ad-eyebrow">rewrite result</span>
          </div>
          {running && (
            <div className="cf-running ad-mono">
              <span className="pg-spinner" />
              re-running with new prompt…
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
