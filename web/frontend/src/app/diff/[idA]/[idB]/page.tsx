'use client';

import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { getDiff, getTrace, getTriage, listTraces } from '@/lib/api';
import type {
  AlignedPair,
  DiffResponse,
  Step,
  TraceDetail,
  TraceSummary,
  TriageResponse,
} from '@/lib/types';
import OutcomeBadge, { type OutcomeKind } from '@/components/OutcomeBadge';
import SiteFooter from '@/components/SiteFooter';
import {
  ArrowRight,
  ChevronDown,
  Close,
  DiffIcon,
  Filter,
  Search,
} from '@/components/Icons';

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

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  return (outcome && OUTCOME_KIND[outcome]) || 'neutral';
}
function outcomeLabelFor(outcome: string | undefined): string {
  return (outcome && OUTCOME_LABEL[outcome]) || outcome || 'unknown';
}

function relativeTime(iso: string, now: number): string {
  const t = new Date(iso).getTime();
  const diff = (now - t) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 7) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / (86400 * 7))}w ago`;
  return `${Math.floor(diff / (86400 * 30))}mo ago`;
}

type OpFilter = 'all' | 'match' | 'substitute' | 'insert' | 'delete';

const OP_LABEL: Record<string, string> = {
  match: '=',
  substitute: '~',
  insert: '+',
  delete: '−',
};
const OP_LABEL_LONG: Record<string, string> = {
  match: 'match',
  substitute: 'substitute',
  insert: 'insert',
  delete: 'delete',
};

const ROLE_LABEL: Record<string, string> = {
  user: 'user',
  assistant: 'assistant',
  tool_call: 'tool call',
  tool_result: 'tool result',
};

function Picker({
  open,
  onClose,
  currentId,
  otherId,
  onSelect,
  corpus,
  now,
}: {
  open: boolean;
  onClose: () => void;
  currentId: string | null;
  otherId: string | null;
  onSelect: (id: string) => void;
  corpus: TraceSummary[];
  now: number;
}) {
  const [query, setQuery] = useState('');
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    const t = setTimeout(() => document.addEventListener('mousedown', onDoc), 0);
    document.addEventListener('keydown', onKey);
    return () => {
      clearTimeout(t);
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onKey);
    };
  }, [open, onClose]);

  const filtered = useMemo(() => {
    if (!query.trim()) return corpus;
    const q = query.toLowerCase();
    return corpus.filter(
      (t) =>
        t.name.toLowerCase().includes(q) ||
        t.id.toLowerCase().includes(q) ||
        (t.baseline_name ?? '').toLowerCase().includes(q) ||
        (t.metadata?.task ?? '').toLowerCase().includes(q),
    );
  }, [query, corpus]);

  const grouped = useMemo(() => {
    const groups: Record<string, { name: string; items: TraceSummary[] }> = {};
    filtered.forEach((t) => {
      const bId = t.baseline_id || '__orphan';
      const bName = t.baseline_name || '(no baseline)';
      if (!groups[bId]) groups[bId] = { name: bName, items: [] };
      groups[bId].items.push(t);
    });
    return Object.entries(groups);
  }, [filtered]);

  if (!open) return null;

  return (
    <div className="diff-picker" ref={ref}>
      <div className="diff-picker-head">
        <Search />
        <input
          autoFocus
          className="diff-picker-input"
          type="text"
          placeholder="search traces, baselines, tasks…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        {query && (
          <button
            className="tr-search-clear"
            onClick={() => setQuery('')}
            aria-label="clear search"
          >
            <Close />
          </button>
        )}
      </div>
      <div className="diff-picker-list">
        {grouped.length === 0 && (
          <div className="diff-picker-empty mono">no traces match.</div>
        )}
        {grouped.map(([bId, g]) => (
          <div key={bId} className="diff-picker-group">
            <div className="diff-picker-group-head mono">
              {g.name} <span className="dim">· {g.items.length}</span>
            </div>
            {g.items.map((t) => {
              const isSelf = t.id === otherId;
              const isCurrent = t.id === currentId;
              return (
                <button
                  key={t.id}
                  className={
                    'diff-picker-row' +
                    (isCurrent ? ' current' : '') +
                    (isSelf ? ' self' : '')
                  }
                  disabled={isSelf}
                  onClick={() => {
                    if (!isSelf) {
                      onSelect(t.id);
                      onClose();
                    }
                  }}
                >
                  <span className="diff-picker-row-name">{t.name}</span>
                  <span className="mono dim diff-picker-row-id">{t.id}</span>
                  <OutcomeBadge
                    kind={outcomeKindFor(t.metadata?.outcome)}
                    label={outcomeLabelFor(t.metadata?.outcome)}
                  />
                  <span className="mono dim diff-picker-row-time">
                    {relativeTime(t.created_at, now)}
                  </span>
                  {isCurrent && (
                    <span className="mono diff-picker-row-tag">current</span>
                  )}
                  {isSelf && (
                    <span className="mono diff-picker-row-tag dim">
                      other side
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}

function TraceCard({
  side,
  trace,
  otherId,
  onSwap,
  corpus,
  now,
}: {
  side: 'a' | 'b';
  trace: TraceDetail | null;
  otherId: string | null;
  onSwap: (id: string) => void;
  corpus: TraceSummary[];
  now: number;
}) {
  const [open, setOpen] = useState(false);

  if (!trace) {
    return (
      <div className={'diff-trace-card empty ' + side}>
        <div className="diff-trace-card-label mono">side {side}</div>
        <button
          className="ad-btn primary"
          onClick={() => setOpen(true)}
        >
          Pick a trace <ChevronDown />
        </button>
        <Picker
          open={open}
          onClose={() => setOpen(false)}
          currentId={null}
          otherId={otherId}
          onSelect={onSwap}
          corpus={corpus}
          now={now}
        />
      </div>
    );
  }

  const outcome = trace.metadata?.outcome;
  const task = trace.metadata?.task;
  // Find the corpus row to get baseline + adapter.
  const summary = corpus.find((c) => c.id === trace.id);
  const baselineId = summary?.baseline_id;
  const baselineName = summary?.baseline_name;
  const stepCount = summary?.step_count ?? trace.steps.length;

  return (
    <div className={'diff-trace-card ' + side}>
      <div className="diff-trace-card-head">
        <span className="diff-trace-card-label mono">side {side}</span>
        <button
          className="diff-trace-card-switch"
          onClick={() => setOpen(true)}
        >
          Switch <ChevronDown />
        </button>
      </div>

      <div className="diff-trace-card-body">
        <div
          className="row"
          style={{ gap: 8, marginBottom: 6, flexWrap: 'wrap' }}
        >
          <span className="diff-trace-card-name">{trace.name}</span>
          <span className="mono dim" style={{ fontSize: 11 }}>
            {trace.id}
          </span>
        </div>

        {baselineId && baselineName && (
          <Link
            href={`/baselines/${baselineId}`}
            className="diff-trace-card-baseline mono"
          >
            {baselineName}
          </Link>
        )}

        <div
          className="row"
          style={{ gap: 8, marginTop: 10, flexWrap: 'wrap' }}
        >
          <OutcomeBadge
            kind={outcomeKindFor(outcome)}
            label={outcomeLabelFor(outcome)}
          />
          <span className="ad-badge neutral mono">{stepCount} steps</span>
          <span className="ad-badge neutral mono">{trace.adapter}</span>
        </div>

        {task && <div className="diff-trace-card-task">{task}</div>}
      </div>

      <Picker
        open={open}
        onClose={() => setOpen(false)}
        currentId={trace.id}
        otherId={otherId}
        onSelect={onSwap}
        corpus={corpus}
        now={now}
      />
    </div>
  );
}

function StepBody({
  step,
  ghost,
  sideClass,
}: {
  step: Step | null;
  ghost: boolean;
  sideClass: 'a' | 'b';
}) {
  if (ghost || !step) {
    return (
      <div className={'diff-step ghost ' + sideClass}>
        <span className="diff-step-ghost mono">— not in this trace —</span>
      </div>
    );
  }
  const role = step.role;
  const tc = step.tool_call;
  const tr = step.tool_result;

  return (
    <div className={'diff-step role-' + role + ' ' + sideClass}>
      <div className="diff-step-head">
        <span className={'diff-role-tag mono role-' + role}>
          {ROLE_LABEL[role] ?? role}
        </span>
        {tc && <span className="diff-step-toolname mono">{tc.name}()</span>}
        {tr && (
          <>
            <span className="diff-step-toolname mono">{tr.name}() →</span>
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
      </div>

      {step.content && <p className="diff-step-content">{step.content}</p>}

      {tc && tc.args && (
        <div className="diff-step-args">
          {Object.entries(tc.args).map(([k, v]) => (
            <div key={k} className="diff-step-arg">
              <span className="mono dim">{k}</span>
              <span className="mono diff-step-arg-val">{String(v)}</span>
            </div>
          ))}
        </div>
      )}

      {tr && tr.output && (
        <pre
          className={
            'diff-step-output mono ' + (tr.is_error ? 'error' : '')
          }
        >
          {tr.output}
        </pre>
      )}
    </div>
  );
}

function AlignmentRow({ row }: { row: AlignedPair }) {
  const { op, a_step, b_step, a_index, b_index } = row;
  const fmt = (i: number | null) =>
    i === null || i === undefined ? '··' : String(i + 1).padStart(2, '0');
  return (
    <div className={'diff-row op-' + op}>
      <div className="diff-row-side a">
        <div className="diff-row-idx mono">{fmt(a_index)}</div>
        <StepBody step={a_step} ghost={a_step === null} sideClass="a" />
      </div>

      <div className="diff-row-marker">
        <div className={'diff-op-badge op-' + op}>
          {OP_LABEL[op] ?? '?'}
        </div>
        <span className="diff-op-name mono">{OP_LABEL_LONG[op] ?? op}</span>
      </div>

      <div className="diff-row-side b">
        <StepBody step={b_step} ghost={b_step === null} sideClass="b" />
        <div className="diff-row-idx mono">{fmt(b_index)}</div>
      </div>
    </div>
  );
}

function Banner({
  kind,
  title,
  body,
}: {
  kind: 'info' | 'warn';
  title: string;
  body: string;
}) {
  return (
    <div className={'diff-banner ' + kind}>
      <div className="diff-banner-body">
        <span className="eyebrow">{kind === 'warn' ? 'heads up' : 'info'}</span>
        <h4>{title}</h4>
        <p>{body}</p>
      </div>
    </div>
  );
}

function TriageBlock({ triage }: { triage: TriageResponse | null }) {
  if (!triage) return null;
  const kind: OutcomeKind =
    triage.classification === 'regression'
      ? 'bad'
      : triage.classification === 'additive'
        ? 'novel'
        : 'warn';
  return (
    <div className={'diff-triage ' + kind}>
      <div className="diff-triage-head">
        <span className="eyebrow">ai triage · /api/diff/.../triage</span>
        <OutcomeBadge kind={kind} label={triage.classification} />
      </div>
      <p className="diff-triage-summary">{triage.summary}</p>
      <div className="diff-triage-cause">
        <span className="mono dim">likely cause</span>
        <p>{triage.likely_cause}</p>
      </div>
    </div>
  );
}

export default function DiffPage() {
  const params = useParams<{ idA: string; idB: string }>();
  const router = useRouter();
  const idA = params.idA;
  const idB = params.idB;

  const [diff, setDiff] = useState<DiffResponse | null>(null);
  const [traceA, setTraceA] = useState<TraceDetail | null>(null);
  const [traceB, setTraceB] = useState<TraceDetail | null>(null);
  const [triage, setTriage] = useState<TriageResponse | null>(null);
  const [corpus, setCorpus] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [opFilter, setOpFilter] = useState<OpFilter>('all');
  const now = useMemo(() => Date.now(), []);

  useEffect(() => {
    if (!idA || !idB) return;
    setLoading(true);
    setError(null);
    Promise.all([getDiff(idA, idB), getTrace(idA), getTrace(idB)])
      .then(([d, a, b]) => {
        setDiff(d);
        setTraceA(a);
        setTraceB(b);
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
    getTriage(idA, idB)
      .then(setTriage)
      .catch(() => setTriage(null));
    listTraces()
      .then(setCorpus)
      .catch(() => setCorpus([]));
  }, [idA, idB]);

  const sameTrace = idA === idB;

  const summaryA = corpus.find((c) => c.id === idA);
  const summaryB = corpus.find((c) => c.id === idB);
  const baselineMismatch =
    summaryA &&
    summaryB &&
    summaryA.baseline_id &&
    summaryB.baseline_id &&
    summaryA.baseline_id !== summaryB.baseline_id;

  const visibleAlignment = useMemo(() => {
    if (!diff) return [];
    if (opFilter === 'all') return diff.alignment;
    return diff.alignment.filter((r) => r.op === opFilter);
  }, [diff, opFilter]);

  const swap = (which: 'a' | 'b', newId: string) => {
    const next = which === 'a' ? { a: newId, b: idB } : { a: idA, b: newId };
    router.push(`/diff/${next.a}/${next.b}`);
  };

  if (loading) {
    return (
      <main>
        <div className="container">
          <p className="ad-muted mono" style={{ padding: '40px 0' }}>
            Loading diff...
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

  const heroStat = (label: string, value: number, color: string) => (
    <div className="bl-stat">
      <span className="mono dim">{label}</span>
      <span
        className="bl-stat-val"
        style={{ color: `var(${color})` } as CSSProperties}
      >
        {value}
      </span>
    </div>
  );

  return (
    <>
      <main>
        <header className="tr-hero diff-hero">
          <div className="container">
            <div className="tr-hero-grid">
              <div>
                <div className="bl-breadcrumb mono">
                  <Link href="/" className="bl-crumb">
                    home
                  </Link>
                  <span className="dim">/</span>
                  <Link href="/traces" className="bl-crumb">
                    traces
                  </Link>
                  <span className="dim">/</span>
                  <span className="bl-crumb-current">
                    diff {idA && idB ? `/ ${idA} / ${idB}` : ''}
                  </span>
                </div>
                <h1 className="tr-hero-title">Diff</h1>
                <p className="tr-hero-sub">
                  Two traces, aligned step by step. See where they agreed, where
                  they substituted, and where one went somewhere the other
                  didn&apos;t.
                </p>
              </div>

              {diff && (
                <aside className="tr-hero-stats diff-hero-stats">
                  {heroStat('matches', diff.summary.matches, '--ok')}
                  {heroStat('subs', diff.summary.substitutions, '--warn')}
                  {heroStat('insertions', diff.summary.insertions, '--novel')}
                  {heroStat('deletions', diff.summary.deletions, '--bad')}
                </aside>
              )}
            </div>
          </div>
        </header>

        <div className="container">
          {baselineMismatch && !sameTrace && (
            <Banner
              kind="warn"
              title="Different baselines — drift signal will be noisy"
              body={`${summaryA?.baseline_name ?? 'side A'} vs ${summaryB?.baseline_name ?? 'side B'}. These traces solve different problems, so most divergence reflects task differences, not agent behavior. For meaningful drift, compare two traces from the same baseline.`}
            />
          )}

          {sameTrace && (
            <Banner
              kind="warn"
              title="Same trace on both sides"
              body="You picked the same trace for A and B. There's nothing to diff — switch one side to compare against a different run."
            />
          )}

          <section className="diff-switcher">
            <TraceCard
              side="a"
              trace={traceA}
              otherId={idB}
              onSwap={(id) => swap('a', id)}
              corpus={corpus}
              now={now}
            />
            <div className="diff-switcher-vs">
              <span className="mono">vs</span>
              <div className="diff-switcher-line" />
            </div>
            <TraceCard
              side="b"
              trace={traceB}
              otherId={idA}
              onSwap={(id) => swap('b', id)}
              corpus={corpus}
              now={now}
            />
          </section>

          {diff && !sameTrace && (
            <>
              <div className="diff-summary">
                <div className="diff-summary-pills">
                  <div className="diff-pill ok">
                    <span className="diff-pill-num">{diff.summary.matches}</span>
                    <span className="diff-pill-label mono">matches</span>
                  </div>
                  <div className="diff-pill warn">
                    <span className="diff-pill-num">
                      {diff.summary.substitutions}
                    </span>
                    <span className="diff-pill-label mono">substitutions</span>
                  </div>
                  <div className="diff-pill novel">
                    <span className="diff-pill-num">
                      {diff.summary.insertions}
                    </span>
                    <span className="diff-pill-label mono">
                      insertions · in B
                    </span>
                  </div>
                  <div className="diff-pill bad">
                    <span className="diff-pill-num">
                      {diff.summary.deletions}
                    </span>
                    <span className="diff-pill-label mono">deletions · in A</span>
                  </div>
                </div>
                <div className="diff-summary-dist">
                  <span className="mono dim">edit distance</span>
                  <span className="diff-summary-dist-num">{diff.distance}</span>
                </div>
              </div>
              <TriageBlock triage={triage} />
            </>
          )}

          {diff && !sameTrace && diff.alignment.length > 0 && (
            <div className="diff-controls">
              <div className="row" style={{ gap: 10 }}>
                <span className="eyebrow">
                  alignment · {diff.alignment.length} rows
                </span>
                {visibleAlignment.length !== diff.alignment.length && (
                  <span className="mono dim" style={{ fontSize: 11 }}>
                    {visibleAlignment.length} of {diff.alignment.length} shown
                  </span>
                )}
              </div>
              <div className="seg">
                {(
                  [
                    ['all', 'all'],
                    ['match', 'matches'],
                    ['substitute', 'subs'],
                    ['insert', 'insertions'],
                    ['delete', 'deletions'],
                  ] as [OpFilter, string][]
                ).map(([id, label]) => (
                  <button
                    key={id}
                    className={'seg-btn' + (opFilter === id ? ' active' : '')}
                    onClick={() => setOpFilter(id)}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>
          )}

          {diff && !sameTrace && (
            <div className="diff-rows">
              {visibleAlignment.length === 0 ? (
                <div className="tr-empty">
                  <div className="tr-empty-icon">
                    <Filter />
                  </div>
                  <h4>No rows match this op filter.</h4>
                  <p>Try widening to &quot;all&quot; or another op.</p>
                  <button className="ad-btn" onClick={() => setOpFilter('all')}>
                    Show all rows
                  </button>
                </div>
              ) : (
                visibleAlignment.map((row, i) => (
                  <AlignmentRow key={i} row={row} />
                ))
              )}
            </div>
          )}

          {sameTrace && (
            <div className="diff-rows">
              <div className="tr-empty">
                <div className="tr-empty-icon">
                  <DiffIcon />
                </div>
                <h4>Empty diff.</h4>
                <p>
                  Side A and Side B are the same trace — nothing to compare.
                </p>
              </div>
            </div>
          )}

          <div className="tr-foot">
            <span className="mono dim" style={{ fontSize: 11 }}>
              api · GET /api/diff/{idA || '·'}/{idB || '·'} ·{' '}
              {diff?.alignment.length ?? 0} alignment rows
            </span>
            <Link href="/traces" className="ad-btn ghost">
              <ArrowRight style={{ transform: 'rotate(180deg)' }} /> Back to
              traces
            </Link>
          </div>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
