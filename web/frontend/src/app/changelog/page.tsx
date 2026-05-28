'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import SiteFooter from '@/components/SiteFooter';
import { ArrowRight } from '@/components/Icons';

type Tag = 'feature' | 'fix' | 'docs' | 'infra';
type Entry = { date: string; tag: Tag; title: string; body: string };
type TagFilter = 'all' | Tag;

const ENTRIES: Entry[] = [
  {
    date: '2026-05-27',
    tag: 'feature',
    title: 'Site redesign',
    body: 'Full visual rebuild on Geist + dark theme. New /about, /docs, /traces, /diff pages. Live tweaks panel for trying out color, density, and font variations in-product.',
  },
  {
    date: '2026-05-27',
    tag: 'feature',
    title: 'Realistic seed scenarios',
    body: 'Replaced 5 abstract baselines with 3 real-engineering-task scenarios — endpoint rename, password-hash migration, new endpoint with tests — each carrying a task description and per-run outcome labels.',
  },
  {
    date: '2026-05-27',
    tag: 'feature',
    title: 'Trace detail rebuild',
    body: 'New deep-dive view per trace. Sticky step list on the left, role-tagged transcript on the right, tool-sequence chip row at the top, full metadata grid in the header.',
  },
  {
    date: '2026-05-27',
    tag: 'feature',
    title: 'Diff page rebuild',
    body: 'Two traces side-by-side with step-by-step comparison, an AI-written explanation of why they diverged, and a switchable picker on either side.',
  },
  {
    date: '2026-05-27',
    tag: 'docs',
    title: 'Developer reference',
    body: 'Added /docs — concepts, API contract, data flow, embed cache mechanism, and a viz interpretation guide all in one scannable page.',
  },
  {
    date: '2026-05-26',
    tag: 'infra',
    title: 'Per-IP rate cap',
    body: "Daily cap on counterfactual and edit-prompt requests so a single visitor can't burn the demo's LLM budget.",
  },
  {
    date: '2026-05-26',
    tag: 'infra',
    title: 'Hosted demo live',
    body: 'Go API deployed on Fly.io, Next.js frontend proxied through Vercel.',
  },
  {
    date: '2026-05-26',
    tag: 'infra',
    title: 'Pre-computed embeddings ship in the binary',
    body: 'Voyage embeddings for every seeded baseline baked into the Go binary at build time — zero cold-start cost on first visit.',
  },
  {
    date: '2026-05-20',
    tag: 'feature',
    title: 'Similar-traces panel',
    body: '"Find traces like this one" on the trace detail page, backed by Voyage embeddings and cosine similarity.',
  },
  {
    date: '2026-05-19',
    tag: 'feature',
    title: 'GitHub Action with inline screenshots',
    body: 'CI step posts the trace-comparison PNG directly into PR comments via an orphan-branch image host — no third-party uploads needed.',
  },
  {
    date: '2026-05-19',
    tag: 'feature',
    title: 'Inline prompt editor',
    body: 'Rewrite the prompt at any step on the trace detail page and re-run from that point. Useful for asking "would it have worked if I\'d said this instead?"',
  },
  {
    date: '2026-05-19',
    tag: 'feature',
    title: 'Replay scrubber',
    body: 'Drag through any trace step by step to see what the model saw at each turn — context window, intermediate reasoning, tool result.',
  },
  {
    date: '2026-05-19',
    tag: 'feature',
    title: 'Cost/latency heatmap on the path graph',
    body: 'Each tool-call node shaded by token cost and wall time so expensive or slow steps stand out at a glance.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'Counterfactual replay',
    body: 'Pick any step in a trace, force a different first decision, and replay from there. Shows how the rest of the trace would have unfolded.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'Audit-to-test',
    body: 'Turn any one-off trace into a permanent baseline with one click — promote-to-test workflow for the runs you want to keep regressing against.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'AI-written transcript summaries',
    body: 'Per-trace summary generated and cached by Claude, surfaced on the trace detail page so you can read the gist before opening the full transcript.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'First-boot demo gallery',
    body: 'Out-of-the-box scenarios load on first run so visitors see real comparisons without any setup.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'AI-written explanations for divergent runs',
    body: 'When a trace diverges from its baseline, Claude writes the one-paragraph explanation of why — surfaced on the diff page.',
  },
  {
    date: '2026-05-18',
    tag: 'feature',
    title: 'Path-graph view',
    body: 'Network visualization showing every trace in a baseline overlaid. Tool calls as nodes, transitions as edges, edge thickness for run count.',
  },
  {
    date: '2026-05-17',
    tag: 'feature',
    title: 'Path-graph backend',
    body: 'Aggregation and overlay endpoints that power the network visualization.',
  },
  {
    date: '2026-04-07',
    tag: 'feature',
    title: 'Metadata tagging on traces',
    body: 'Attach arbitrary key/value labels to any trace for grouping and filtering.',
  },
  {
    date: '2026-04-06',
    tag: 'infra',
    title: 'Frontend test suite',
    body: '90+ unit and component tests across pages, components, and the API client. Vitest + React Testing Library.',
  },
  {
    date: '2026-04-06',
    tag: 'feature',
    title: 'agentdiff bench CLI',
    body: 'Run a synthetic-trace benchmark suite from the command line with table or JSON output.',
  },
  {
    date: '2026-04-06',
    tag: 'feature',
    title: 'Evaluation metrics for benchmarks',
    body: 'ARI, F1, recall and precision computed against synthetic ground truth — so cluster-quality regressions are visible per-commit.',
  },
  {
    date: '2026-04-05',
    tag: 'feature',
    title: 'Web comparison dashboard',
    body: 'Initial Next.js frontend. Lists baselines, diffs traces, shows clustering and drift across runs.',
  },
  {
    date: '2026-04-05',
    tag: 'feature',
    title: 'Claude Code adapter',
    body: "Parses Claude Code's stream-json output into the agentdiff trace format.",
  },
  {
    date: '2026-04-05',
    tag: 'feature',
    title: 'Strategy clustering',
    body: 'DBSCAN-based clustering of trace tool-sequences into named strategies per baseline.',
  },
];

const TAG_KIND: Record<Tag, string> = {
  feature: 'accent',
  fix: 'warn',
  docs: 'novel',
  infra: 'neutral',
};

const TAGS: TagFilter[] = ['all', 'feature', 'fix', 'docs', 'infra'];

function formatDay(iso: string): string {
  const d = new Date(iso + 'T12:00:00Z');
  return String(d.getUTCDate()).padStart(2, '0');
}
function monthGroup(iso: string): string {
  const d = new Date(iso + 'T12:00:00Z');
  return d.toLocaleString('en-US', {
    month: 'long',
    year: 'numeric',
    timeZone: 'UTC',
  });
}

function EntryRow({ e }: { e: Entry }) {
  return (
    <article className={'cl-entry tag-' + e.tag} data-tag={e.tag}>
      <div className="cl-entry-rail">
        <div className="cl-entry-day mono">{formatDay(e.date)}</div>
        <div className={'cl-entry-dot tag-' + e.tag} />
      </div>
      <div className="cl-entry-body">
        <header className="cl-entry-head">
          <h3 className="cl-entry-title">{e.title}</h3>
          <span
            className={'ad-badge ' + TAG_KIND[e.tag] + ' mono cl-entry-tag'}
          >
            {e.tag}
          </span>
        </header>
        <p className="cl-entry-text">{e.body}</p>
        <div className="cl-entry-meta mono">
          <span>{e.date}</span>
        </div>
      </div>
    </article>
  );
}

export default function ChangelogPage() {
  const [tagFilter, setTagFilter] = useState<TagFilter>('all');

  const counts = useMemo(() => {
    const c: Record<TagFilter, number> = {
      all: ENTRIES.length,
      feature: 0,
      fix: 0,
      docs: 0,
      infra: 0,
    };
    ENTRIES.forEach((e) => {
      c[e.tag] = (c[e.tag] ?? 0) + 1;
    });
    return c;
  }, []);

  const groups = useMemo(() => {
    const filtered =
      tagFilter === 'all'
        ? ENTRIES
        : ENTRIES.filter((e) => e.tag === tagFilter);
    const map = new Map<string, Entry[]>();
    filtered.forEach((e) => {
      const k = monthGroup(e.date);
      if (!map.has(k)) map.set(k, []);
      map.get(k)!.push(e);
    });
    return Array.from(map.entries());
  }, [tagFilter]);

  return (
    <>
      <main>
        <header className="tr-hero cl-hero">
          <div className="container">
            <div className="bl-breadcrumb mono">
              <Link href="/" className="bl-crumb">
                home
              </Link>
              <span className="dim">/</span>
              <span className="bl-crumb-current">changelog</span>
            </div>

            <div className="cl-hero-grid">
              <div>
                <h1 className="tr-hero-title">Changelog</h1>
                <p className="tr-hero-sub">
                  Everything that&apos;s shipped, newest first. Each entry is
                  one or two plain-English sentences about what changed for
                  you.
                </p>
              </div>

              <aside className="cl-hero-stats tr-hero-stats">
                <div className="bl-stat">
                  <span className="mono dim">total</span>
                  <span className="bl-stat-val">{ENTRIES.length}</span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">features</span>
                  <span
                    className="bl-stat-val"
                    style={{ color: 'var(--accent)' }}
                  >
                    {counts.feature}
                  </span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">infra</span>
                  <span className="bl-stat-val">{counts.infra}</span>
                </div>
                <div className="bl-stat">
                  <span className="mono dim">docs</span>
                  <span
                    className="bl-stat-val"
                    style={{ color: 'var(--novel)' }}
                  >
                    {counts.docs}
                  </span>
                </div>
              </aside>
            </div>
          </div>
        </header>

        <div className="container">
          <div className="cl-controls">
            <span className="eyebrow">filter by tag</span>
            <div className="seg">
              {TAGS.map((t) => (
                <button
                  key={t}
                  className={'seg-btn' + (tagFilter === t ? ' active' : '')}
                  onClick={() => setTagFilter(t)}
                >
                  {t}
                  <span className="cl-seg-count mono">{counts[t]}</span>
                </button>
              ))}
            </div>
          </div>

          <div className="cl-timeline">
            {groups.length === 0 && (
              <div className="tr-empty">
                <h4>No entries with this tag.</h4>
                <button
                  className="ad-btn"
                  onClick={() => setTagFilter('all')}
                >
                  Show all
                </button>
              </div>
            )}
            {groups.map(([month, entries]) => (
              <section key={month} className="cl-month">
                <header className="cl-month-head">
                  <h2 className="cl-month-name">{month}</h2>
                  <span className="mono dim cl-month-count">
                    {entries.length}{' '}
                    {entries.length === 1 ? 'entry' : 'entries'}
                  </span>
                </header>
                <div className="cl-entries">
                  {entries.map((e, i) => (
                    <EntryRow key={month + i} e={e} />
                  ))}
                </div>
              </section>
            ))}
          </div>

          <footer className="dx-foot">
            <span className="mono dim">end of log · agentdiff v0.4.1</span>
            <Link href="/docs" className="ad-btn">
              <ArrowRight style={{ transform: 'rotate(180deg)' }} /> Reference
            </Link>
          </footer>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
