'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import Link from 'next/link';
import { listTraces } from '@/lib/api';
import type { TraceSummary } from '@/lib/types';
import OutcomeBadge, { type OutcomeKind } from '@/components/OutcomeBadge';
import SiteFooter from '@/components/SiteFooter';
import { ArrowRight, ChevronRight, Close, Search } from '@/components/Icons';

const OUTCOMES = ['succeeded', 'variance', 'regressed', 'additive'] as const;
type Outcome = (typeof OUTCOMES)[number];

const OUTCOME_KIND: Record<string, OutcomeKind> = {
  succeeded: 'ok',
  variance: 'warn',
  regressed: 'bad',
  additive: 'novel',
};

type SortKey =
  | 'recent'
  | 'oldest'
  | 'steps_desc'
  | 'steps_asc'
  | 'outcome'
  | 'baseline';

const SORTS: { id: SortKey; label: string }[] = [
  { id: 'recent', label: 'newest first' },
  { id: 'oldest', label: 'oldest first' },
  { id: 'steps_desc', label: 'most steps' },
  { id: 'steps_asc', label: 'fewest steps' },
  { id: 'outcome', label: 'outcome' },
  { id: 'baseline', label: 'baseline name' },
];

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

function highlight(text: string, query: string): ReactNode {
  if (!query.trim()) return text;
  const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const re = new RegExp(`(${escaped})`, 'ig');
  const parts = text.split(re);
  return parts.map((p, i) =>
    re.test(p) ? <mark key={i}>{p}</mark> : <span key={i}>{p}</span>,
  );
}

function outcomeKindFor(outcome: string | undefined): OutcomeKind {
  return (outcome && OUTCOME_KIND[outcome]) || 'neutral';
}

export default function TracesPage() {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [baseline, setBaseline] = useState<string>('all');
  const [outcomes, setOutcomes] = useState<Set<Outcome>>(new Set());
  const [query, setQuery] = useState('');
  const [sort, setSort] = useState<SortKey>('recent');

  useEffect(() => {
    setLoading(true);
    listTraces()
      .then(setTraces)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  // now is captured once per render; close-enough for "X ago" labels
  const now = useMemo(() => Date.now(), [traces]);

  const baselines = useMemo(() => {
    const counts: Record<string, { id: string; name: string; count: number }> = {};
    traces.forEach((t) => {
      const id = t.baseline_id;
      const name = t.baseline_name;
      if (!id || !name) return;
      if (!counts[id]) counts[id] = { id, name, count: 0 };
      counts[id].count++;
    });
    return Object.values(counts);
  }, [traces]);

  const outcomeCounts = useMemo(() => {
    const c: Record<Outcome, number> = {
      succeeded: 0,
      variance: 0,
      regressed: 0,
      additive: 0,
    };
    traces.forEach((t) => {
      const o = t.metadata?.outcome as Outcome | undefined;
      if (o && o in c) c[o]++;
    });
    return c;
  }, [traces]);

  const adapterCounts = useMemo(() => {
    const c: Record<string, number> = {};
    traces.forEach((t) => {
      c[t.adapter] = (c[t.adapter] || 0) + 1;
    });
    return c;
  }, [traces]);

  const filtered = useMemo(() => {
    let rows = traces;
    if (baseline !== 'all') {
      rows = rows.filter((t) => t.baseline_id === baseline);
    }
    if (outcomes.size > 0) {
      rows = rows.filter((t) => {
        const o = t.metadata?.outcome as Outcome | undefined;
        return o ? outcomes.has(o) : false;
      });
    }
    if (query.trim()) {
      const q = query.toLowerCase();
      rows = rows.filter((t) => {
        const task = t.metadata?.task?.toLowerCase() || '';
        const decision = t.metadata?.key_decision?.toLowerCase() || '';
        const name = t.name.toLowerCase();
        const blName = t.baseline_name?.toLowerCase() || '';
        return (
          task.includes(q) ||
          decision.includes(q) ||
          name.includes(q) ||
          blName.includes(q)
        );
      });
    }

    rows = [...rows];
    if (sort === 'recent') {
      rows.sort(
        (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
      );
    } else if (sort === 'oldest') {
      rows.sort(
        (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
      );
    } else if (sort === 'steps_desc') {
      rows.sort((a, b) => b.step_count - a.step_count);
    } else if (sort === 'steps_asc') {
      rows.sort((a, b) => a.step_count - b.step_count);
    } else if (sort === 'outcome') {
      rows.sort((a, b) =>
        (a.metadata?.outcome || '').localeCompare(b.metadata?.outcome || ''),
      );
    } else if (sort === 'baseline') {
      rows.sort((a, b) =>
        (a.baseline_name || '').localeCompare(b.baseline_name || ''),
      );
    }
    return rows;
  }, [traces, baseline, outcomes, query, sort]);

  const toggleOutcome = (o: Outcome) => {
    setOutcomes((prev) => {
      const next = new Set(prev);
      if (next.has(o)) next.delete(o);
      else next.add(o);
      return next;
    });
  };
  const clearFilters = () => {
    setBaseline('all');
    setOutcomes(new Set());
    setQuery('');
  };
  const clearOutcomes = () => setOutcomes(new Set());

  const isFiltering =
    baseline !== 'all' || outcomes.size > 0 || query.trim() !== '';

  return (
    <>
      <header className="tr-hero">
        <div className="container">
          <div className="tr-hero-grid">
            <div>
              <div className="bl-breadcrumb ad-mono">
                <Link href="/" className="bl-crumb">
                  home
                </Link>
                <span className="ad-dim">/</span>
                <span className="bl-crumb-current">traces</span>
              </div>
              <h1 className="tr-hero-title">Traces</h1>
              <p className="tr-hero-sub">
                Every recording across every baseline, in one place. Filter,
                sort, click in.
              </p>
            </div>

            <aside className="tr-hero-stats">
              <div className="bl-stat">
                <span className="ad-mono ad-dim">traces</span>
                <span className="bl-stat-val">{traces.length}</span>
              </div>
              <div className="bl-stat">
                <span className="ad-mono ad-dim">baselines</span>
                <span className="bl-stat-val">{baselines.length}</span>
              </div>
              <div className="bl-stat">
                <span className="ad-mono ad-dim">regressed</span>
                <span
                  className="bl-stat-val"
                  style={{
                    color:
                      outcomeCounts.regressed > 0
                        ? 'var(--bad)'
                        : 'var(--fg)',
                  }}
                >
                  {outcomeCounts.regressed}
                </span>
              </div>
              <div className="bl-stat">
                <span className="ad-mono ad-dim">additive</span>
                <span
                  className="bl-stat-val"
                  style={{
                    color:
                      outcomeCounts.additive > 0
                        ? 'var(--novel)'
                        : 'var(--fg)',
                  }}
                >
                  {outcomeCounts.additive}
                </span>
              </div>
            </aside>
          </div>

          {Object.keys(adapterCounts).length > 0 && (
            <div className="tr-hero-adapters">
              <span className="ad-eyebrow">recorded via</span>
              {Object.entries(adapterCounts).map(([a, n]) => (
                <span key={a} className="ad-badge neutral ad-mono">
                  {a} · {n}
                </span>
              ))}
            </div>
          )}
        </div>
      </header>

      <div className="container">
        {error && (
          <div className="tr-empty" role="alert">
            <h4>Couldn&apos;t load traces.</h4>
            <p>{error}</p>
          </div>
        )}

        {!error && (
          <div className="tr-filters">
            <div className="tr-filter-row">
              <div className="tr-filter-group">
                <label className="tr-filter-label ad-mono">baseline</label>
                <select
                  className="select"
                  value={baseline}
                  onChange={(e) => setBaseline(e.target.value)}
                >
                  <option value="all">
                    all baselines · {baselines.length}
                  </option>
                  {baselines.map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.name} · {b.count}
                    </option>
                  ))}
                </select>
              </div>

              <div className="tr-filter-group">
                <label className="tr-filter-label ad-mono">outcome</label>
                <div className="seg">
                  <button
                    type="button"
                    className={'seg-btn' + (outcomes.size === 0 ? ' active' : '')}
                    onClick={clearOutcomes}
                  >
                    all
                  </button>
                  {OUTCOMES.map((o) => (
                    <button
                      type="button"
                      key={o}
                      className={'seg-btn' + (outcomes.has(o) ? ' active' : '')}
                      onClick={() => toggleOutcome(o)}
                    >
                      {o}
                    </button>
                  ))}
                </div>
              </div>

              <div className="tr-filter-group tr-filter-search">
                <label className="tr-filter-label ad-mono">task</label>
                <div className="tr-search">
                  <Search />
                  <input
                    type="text"
                    className="tr-search-input"
                    placeholder="filter by task name or key decision…"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                  {query && (
                    <button
                      type="button"
                      className="tr-search-clear"
                      onClick={() => setQuery('')}
                      aria-label="Clear search"
                    >
                      <Close />
                    </button>
                  )}
                </div>
              </div>
            </div>

            <div className="tr-filter-summary">
              <span className="ad-mono ad-dim">
                {filtered.length === traces.length ? (
                  <>
                    showing{' '}
                    <span style={{ color: 'var(--fg)' }}>{traces.length}</span>{' '}
                    traces
                  </>
                ) : (
                  <>
                    showing{' '}
                    <span style={{ color: 'var(--fg)' }}>{filtered.length}</span>{' '}
                    of {traces.length} traces
                  </>
                )}
              </span>
              {isFiltering && (
                <button
                  type="button"
                  className="ad-btn ghost"
                  onClick={clearFilters}
                  style={{ padding: '4px 10px', fontSize: 12 }}
                >
                  <Close /> clear filters
                </button>
              )}
            </div>
          </div>
        )}

        {!error && (
          <div className="tr-list-head">
            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <span className="ad-eyebrow">trace list</span>
              <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>
                {filtered.length} rows
              </span>
            </div>
            <select
              className="select"
              value={sort}
              onChange={(e) => setSort(e.target.value as SortKey)}
              aria-label="Sort traces"
            >
              {SORTS.map((s) => (
                <option key={s.id} value={s.id}>
                  sort: {s.label}
                </option>
              ))}
            </select>
          </div>
        )}

        {!error && loading && (
          <div className="tr-empty">
            <p className="ad-mono ad-dim">Loading traces…</p>
          </div>
        )}

        {!error && !loading && filtered.length === 0 && (
          <div className="tr-empty">
            <div className="tr-empty-icon">
              <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
                <circle
                  cx="18"
                  cy="18"
                  r="11"
                  stroke="var(--border-hi)"
                  strokeWidth="1.4"
                  strokeDasharray="3 3"
                />
                <path
                  d="M26 26 L 34 34"
                  stroke="var(--border-hi)"
                  strokeWidth="1.4"
                  strokeLinecap="round"
                />
                <circle cx="18" cy="18" r="3" fill="var(--accent)" opacity="0.4" />
              </svg>
            </div>
            <h4>No traces match these filters.</h4>
            <p>
              Try widening the baseline, clearing the outcome chips, or
              shortening your search.
            </p>
            <button type="button" className="ad-btn" onClick={clearFilters}>
              Clear all filters
            </button>
          </div>
        )}

        {!error && !loading && filtered.length > 0 && (
          <div className="tl-table tr-table">
            <div className="tl-row tl-head tr-head ad-mono">
              <span className="tl-cell tr-cell-id">trace</span>
              <span className="tl-cell tr-cell-baseline">baseline</span>
              <span className="tl-cell tr-cell-out">outcome</span>
              <span className="tl-cell tr-cell-steps">steps</span>
              <span className="tl-cell tr-cell-task">task · key decision</span>
              <span className="tl-cell tr-cell-adapter">adapter</span>
              <span className="tl-cell tr-cell-time">recorded</span>
              <span className="tl-cell tl-cell-arrow"></span>
            </div>

            {filtered.map((t) => {
              const outcome = t.metadata?.outcome;
              return (
                <Link
                  key={t.id}
                  href={`/traces/${t.id}`}
                  className="tl-row tr-row"
                >
                  <span className="tl-cell tr-cell-id">
                    <span className="tl-trace-name">{t.name}</span>
                    <span
                      className="ad-mono ad-dim"
                      style={{ fontSize: 11 }}
                    >
                      {t.id}
                    </span>
                  </span>

                  <span className="tl-cell tr-cell-baseline">
                    {t.baseline_name && t.baseline_id ? (
                      <Link
                        href={`/baselines/${t.baseline_id}`}
                        className="tr-baseline-link ad-mono"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {t.baseline_name}
                      </Link>
                    ) : (
                      <span className="ad-mono ad-dim">—</span>
                    )}
                  </span>

                  <span className="tl-cell tr-cell-out">
                    <OutcomeBadge
                      kind={outcomeKindFor(outcome)}
                      label={outcome ?? 'unknown'}
                    />
                  </span>

                  <span className="tl-cell tr-cell-steps ad-mono">
                    {t.step_count}
                  </span>

                  <span className="tl-cell tr-cell-task">
                    <span className="tr-task-text">
                      {highlight(t.metadata?.task ?? '—', query)}
                    </span>
                    {t.metadata?.key_decision && (
                      <span className="tr-decision-text ad-mono">
                        {highlight(t.metadata.key_decision, query)}
                      </span>
                    )}
                  </span>

                  <span className="tl-cell tr-cell-adapter">
                    <span className="tr-adapter ad-mono">{t.adapter}</span>
                  </span>

                  <span
                    className="tl-cell tr-cell-time ad-mono"
                    title={t.created_at}
                  >
                    {relativeTime(t.created_at, now)}
                  </span>

                  <span className="tl-cell tl-cell-arrow">
                    <ChevronRight />
                  </span>
                </Link>
              );
            })}
          </div>
        )}

        <div className="tr-foot">
          <span className="ad-mono ad-dim" style={{ fontSize: 11 }}>
            api · GET /api/traces · {traces.length} records
          </span>
          <Link href="/" className="ad-btn ghost">
            <ArrowRight style={{ transform: 'rotate(180deg)' }} /> Back to home
          </Link>
        </div>
      </div>

      <SiteFooter />
    </>
  );
}
