'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { getTrace, listTraces } from '@/lib/api';
import type { Step, TraceDetail, TraceSummary } from '@/lib/types';
import OutcomeBadge, { type OutcomeKind } from '@/components/OutcomeBadge';
import SiteFooter from '@/components/SiteFooter';
import { ArrowRight, Branch, DiffIcon } from '@/components/Icons';
import PromoteButton from '@/components/PromoteButton';
import CounterfactualButton from '@/components/CounterfactualButton';
import EditPromptButton from '@/components/EditPromptButton';
import Scrubber from '@/components/Scrubber';
import SimilarTraces from '@/components/SimilarTraces';
import Transcript from '@/components/Transcript';

const OUTCOME_KIND: Record<string, OutcomeKind> = {
  succeeded: 'ok',
  variance: 'warn',
  regressed: 'bad',
  additive: 'novel',
};
const OUTCOME_LABEL: Record<string, string> = {
  succeeded: 'succeeded',
  variance: 'variance',
  regressed: 'regressed',
  additive: 'additive',
};

const ROLE_LABEL: Record<string, string> = {
  user: 'user',
  assistant: 'assistant',
  tool_call: 'tool call',
  tool_result: 'tool result',
};

function outcomeKindFor(o: string | undefined): OutcomeKind {
  return (o && OUTCOME_KIND[o]) || 'neutral';
}
function outcomeLabelFor(o: string | undefined): string {
  return (o && OUTCOME_LABEL[o]) || o || 'unknown';
}

function relativeTime(iso: string, now: number): string {
  const t = new Date(iso).getTime();
  const d = (now - t) / 1000;
  if (d < 60) return `${Math.floor(d)}s ago`;
  if (d < 3600) return `${Math.floor(d / 60)}m ago`;
  if (d < 86400) return `${Math.floor(d / 3600)}h ago`;
  if (d < 86400 * 7) return `${Math.floor(d / 86400)}d ago`;
  if (d < 86400 * 30) return `${Math.floor(d / (86400 * 7))}w ago`;
  return `${Math.floor(d / (86400 * 30))}mo ago`;
}

function stepSummary(step: Step): string {
  if (step.tool_call) {
    const args = step.tool_call.args ?? {};
    const hint =
      (args.path as string | undefined) ??
      (args.cmd as string | undefined) ??
      (args.query as string | undefined) ??
      (args.pattern as string | undefined) ??
      '';
    return `${step.tool_call.name}(${hint})`;
  }
  if (step.tool_result) return `${step.tool_result.name} →`;
  if (step.content)
    return step.content.length > 60
      ? step.content.slice(0, 60) + '…'
      : step.content;
  return '(empty)';
}

function autoPickOther(
  currentId: string,
  currentBaselineId: string | undefined,
  corpus: TraceSummary[],
): string | null {
  if (corpus.length < 2) return null;
  const others = corpus.filter((t) => t.id !== currentId);
  if (others.length === 0) return null;
  const sorted = [...others].sort(
    (x, y) =>
      new Date(y.created_at).getTime() - new Date(x.created_at).getTime(),
  );
  const crossBaseline = sorted.find(
    (t) => t.baseline_id && t.baseline_id !== currentBaselineId,
  );
  return (crossBaseline ?? sorted[0]).id;
}

export default function TraceDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [corpus, setCorpus] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeStep, setActiveStep] = useState(0);
  const railRef = useRef<HTMLDivElement>(null);
  const now = useMemo(() => Date.now(), []);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    setError(null);
    getTrace(id)
      .then(setTrace)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
    listTraces()
      .then(setCorpus)
      .catch(() => setCorpus([]));
  }, [id]);

  const summary = corpus.find((c) => c.id === id);
  const baselineId = summary?.baseline_id;
  const baselineName = summary?.baseline_name;
  const stepCount = trace?.steps.length ?? 0;

  // Active step tracking on scroll.
  useEffect(() => {
    if (!trace || trace.steps.length === 0) return;
    let lastIdx = -1;
    const computeActive = () => {
      const stepEls = document.querySelectorAll<HTMLElement>('.td-step');
      const threshold = 140;
      let activeIdx = 0;
      stepEls.forEach((el) => {
        const r = el.getBoundingClientRect();
        if (r.top - threshold <= 0) {
          activeIdx = Number(el.dataset.stepIdx ?? '0');
        }
      });
      if (activeIdx !== lastIdx) {
        lastIdx = activeIdx;
        setActiveStep(activeIdx);
      }
    };
    const onScroll = () => computeActive();
    window.addEventListener('scroll', onScroll, { passive: true });
    computeActive();
    return () => window.removeEventListener('scroll', onScroll);
  }, [trace]);

  // Scroll the rail to keep the active item visible.
  useEffect(() => {
    if (!railRef.current) return;
    const active = railRef.current.querySelector<HTMLElement>(
      '.td-rail-step.active',
    );
    if (!active) return;
    const rTop = railRef.current.scrollTop;
    const rH = railRef.current.clientHeight;
    const aTop = active.offsetTop;
    const aH = active.clientHeight;
    if (aTop < rTop + 20 || aTop + aH > rTop + rH - 20) {
      railRef.current.scrollTop = aTop - rH / 2 + aH / 2;
    }
  }, [activeStep]);

  const jumpToStep = (idx: number) => {
    const target = document.getElementById('step-' + idx);
    if (!target) return;
    const top = target.getBoundingClientRect().top + window.scrollY - 80;
    window.scrollTo({ top, behavior: 'smooth' });
  };

  const handleCompare = () => {
    if (!trace) return;
    const otherId = autoPickOther(trace.id, baselineId, corpus);
    if (otherId) router.push(`/diff/${trace.id}/${otherId}?auto=1`);
    else router.push('/diff');
  };

  if (loading) {
    return (
      <main>
        <div className="container">
          <p className="ad-muted mono" style={{ padding: '40px 0' }}>
            Loading trace...
          </p>
        </div>
      </main>
    );
  }
  if (error) {
    return (
      <main>
        <div className="container">
          <p className="ad-badge bad mono" style={{ padding: '40px 0' }}>
            Error: {error}
          </p>
        </div>
      </main>
    );
  }
  if (!trace) {
    return (
      <>
        <main>
          <div className="container" style={{ padding: '80px 0' }}>
            <div className="tr-empty">
              <h4>Trace not found</h4>
              <p>
                No trace with id <span className="mono">{id}</span>.
              </p>
              <Link href="/traces" className="ad-btn">
                Back to traces
              </Link>
            </div>
          </div>
        </main>
        <SiteFooter />
      </>
    );
  }

  const steps = trace.steps;
  const toolCalls = steps.filter((s) => s.tool_call);
  const errorCount = steps.filter((s) => s.tool_result?.is_error).length;
  const assistantTurns = steps.filter((s) => s.role === 'assistant').length;
  const outcome = trace.metadata?.outcome;
  const task = trace.metadata?.task ?? trace.name;
  const keyDecision = trace.metadata?.key_decision;

  return (
    <>
      <main>
        <header className="tr-hero td-hero">
          <div className="container">
            <div className="bl-breadcrumb mono">
              <Link href="/" className="bl-crumb">
                home
              </Link>
              <span className="dim">/</span>
              <Link href="/traces" className="bl-crumb">
                traces
              </Link>
              <span className="dim">/</span>
              <span className="bl-crumb-current">{trace.id}</span>
            </div>

            <div className="td-hero-grid">
              <div className="td-hero-main">
                <div
                  className="row"
                  style={{ gap: 8, marginBottom: 16, flexWrap: 'wrap' }}
                >
                  <span className="td-hero-name mono">{trace.name}</span>
                  <span className="mono dim" style={{ fontSize: 13 }}>
                    · {trace.id}
                  </span>
                </div>
                <h1 className="td-hero-title">{task}</h1>
                {keyDecision && (
                  <div className="td-keydecision">
                    <span className="eyebrow">key decision</span>
                    <p>{keyDecision}</p>
                  </div>
                )}
              </div>

              <aside className="td-hero-actions">
                <button
                  className="ad-btn"
                  onClick={handleCompare}
                  disabled={corpus.length < 2}
                >
                  <DiffIcon /> Compare with another trace
                </button>
                {baselineId && (
                  <Link
                    href={`/baselines/${baselineId}`}
                    className="ad-btn ghost"
                  >
                    <Branch /> View baseline
                  </Link>
                )}
              </aside>
            </div>

            <div className="td-meta-grid">
              <div className="td-meta">
                <span className="mono dim">outcome</span>
                <div className="td-meta-val">
                  <OutcomeBadge
                    kind={outcomeKindFor(outcome)}
                    label={outcomeLabelFor(outcome)}
                  />
                </div>
              </div>
              <div className="td-meta">
                <span className="mono dim">baseline</span>
                {baselineId && baselineName ? (
                  <Link
                    href={`/baselines/${baselineId}`}
                    className="td-meta-val mono td-meta-link"
                  >
                    {baselineName}
                  </Link>
                ) : (
                  <span className="td-meta-val mono dim">(none)</span>
                )}
              </div>
              <div className="td-meta">
                <span className="mono dim">adapter</span>
                <span className="td-meta-val mono">{trace.adapter}</span>
              </div>
              <div className="td-meta">
                <span className="mono dim">steps</span>
                <span className="td-meta-val mono">{stepCount}</span>
              </div>
              <div className="td-meta">
                <span className="mono dim">tool calls</span>
                <span className="td-meta-val mono">{toolCalls.length}</span>
              </div>
              <div className="td-meta">
                <span className="mono dim">assistant turns</span>
                <span className="td-meta-val mono">{assistantTurns}</span>
              </div>
              <div className="td-meta">
                <span className="mono dim">errors</span>
                <span
                  className="td-meta-val mono"
                  style={{
                    color:
                      errorCount > 0 ? 'var(--bad)' : 'var(--fg)',
                  }}
                >
                  {errorCount}
                </span>
              </div>
              <div className="td-meta">
                <span className="mono dim">recorded</span>
                <span className="td-meta-val mono">
                  {relativeTime(trace.created_at, now)}
                </span>
              </div>
            </div>
          </div>
        </header>

        <div className="container td-body">
          <aside className="td-rail">
            <div className="td-rail-inner" ref={railRef}>
              <div className="td-rail-head">
                <span className="eyebrow">step list</span>
                <span
                  className="mono dim"
                  style={{ fontSize: 11 }}
                >
                  {steps.length}
                </span>
              </div>
              <div className="td-rail-steps">
                {steps.map((step, idx) => (
                  <button
                    key={idx}
                    type="button"
                    className={
                      'td-rail-step role-' +
                      step.role +
                      (idx === activeStep ? ' active' : '')
                    }
                    onClick={() => jumpToStep(idx)}
                  >
                    <span className="td-rail-step-idx mono">
                      {String(idx + 1).padStart(2, '0')}
                    </span>
                    <span
                      className={'td-rail-step-dot role-' + step.role}
                    />
                    <span className="td-rail-step-summary mono">
                      {stepSummary(step)}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          </aside>

          <section className="td-transcript">
            {toolCalls.length > 0 && (
              <div className="td-toolseq">
                <span className="eyebrow">
                  tool sequence · {toolCalls.length} calls
                </span>
                <div className="td-toolseq-chain">
                  {toolCalls.map((s, i) => (
                    <span key={i} style={{ display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                      <span className="td-toolseq-chip mono">
                        {s.tool_call!.name}
                      </span>
                      {i < toolCalls.length - 1 && (
                        <span className="td-toolseq-arrow mono">→</span>
                      )}
                    </span>
                  ))}
                </div>
              </div>
            )}

            <div className="td-transcript-head">
              <span className="eyebrow">
                transcript · {steps.length} steps
              </span>
              <span className="mono dim" style={{ fontSize: 11 }}>
                step {String(activeStep + 1).padStart(2, '0')} in view
              </span>
            </div>

            <div className="td-steps">
              {steps.map((step, idx) => {
                const tc = step.tool_call;
                const tr = step.tool_result;
                return (
                  <article
                    key={idx}
                    id={'step-' + idx}
                    className={'td-step role-' + step.role}
                    data-step-idx={idx}
                  >
                    <div className="td-step-gutter">
                      <span className="td-step-num mono">
                        {String(idx + 1).padStart(2, '0')}
                      </span>
                      <div
                        className={'td-step-rail-mark role-' + step.role}
                      />
                    </div>
                    <div className="td-step-body">
                      <header className="td-step-head">
                        <span
                          className={'td-role-tag mono role-' + step.role}
                        >
                          {ROLE_LABEL[step.role] ?? step.role}
                        </span>
                        {tc && (
                          <span className="td-toolname mono">{tc.name}()</span>
                        )}
                        {tr && (
                          <>
                            <span className="td-toolname mono">
                              {tr.name}() →
                            </span>
                            <span
                              className={
                                'ad-badge mono ' + (tr.is_error ? 'bad' : 'ok')
                              }
                              style={{ padding: '1px 6px', fontSize: 10 }}
                            >
                              {tr.is_error ? 'error' : 'ok'}
                            </span>
                          </>
                        )}
                        <a
                          href={'#step-' + idx}
                          className="td-step-anchor mono"
                          title="anchor"
                        >
                          #step-{idx + 1}
                        </a>
                      </header>

                      {step.content && (
                        <p className="td-step-content">{step.content}</p>
                      )}

                      {tc && tc.args && Object.keys(tc.args).length > 0 && (
                        <div className="td-args">
                          <div className="td-args-head mono">arguments</div>
                          {Object.entries(tc.args).map(([k, v]) => (
                            <div key={k} className="td-arg">
                              <span className="mono dim">{k}</span>
                              <span className="mono td-arg-val">
                                {typeof v === 'string'
                                  ? v
                                  : JSON.stringify(v)}
                              </span>
                            </div>
                          ))}
                        </div>
                      )}

                      {tr && tr.output && (
                        <div
                          className={
                            'td-output ' + (tr.is_error ? 'error' : '')
                          }
                        >
                          <div className="td-output-head mono">
                            <span>output</span>
                            {tr.is_error && (
                              <span style={{ color: 'var(--bad)' }}>
                                · error
                              </span>
                            )}
                          </div>
                          <pre className="mono">{tr.output}</pre>
                        </div>
                      )}
                    </div>
                  </article>
                );
              })}
            </div>

            <div className="td-actions-section">
              <span className="eyebrow">actions</span>
              <Transcript traceId={trace.id} />
              <PromoteButton
                traceId={trace.id}
                defaultName={`promoted-${trace.name}`}
              />
              <Scrubber steps={trace.steps} />
              <CounterfactualButton traceId={trace.id} steps={trace.steps} />
              <EditPromptButton traceId={trace.id} steps={trace.steps} />
              <SimilarTraces traceId={trace.id} />
            </div>

            <div className="td-foot">
              <Link href="/traces" className="ad-btn ghost">
                <ArrowRight style={{ transform: 'rotate(180deg)' }} /> Back to
                traces
              </Link>
              <span className="mono dim" style={{ fontSize: 11 }}>
                api · GET /api/traces/{trace.id}
              </span>
            </div>
          </section>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
