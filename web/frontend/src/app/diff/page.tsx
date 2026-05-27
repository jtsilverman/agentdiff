'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { listTraces } from '@/lib/api';
import type { TraceSummary } from '@/lib/types';
import SiteFooter from '@/components/SiteFooter';
import { DiffIcon } from '@/components/Icons';

function autoSelectPair(
  corpus: TraceSummary[],
): { a: string; b: string } | null {
  if (corpus.length < 2) return null;
  const sorted = [...corpus].sort(
    (x, y) => new Date(y.created_at).getTime() - new Date(x.created_at).getTime(),
  );
  const first = sorted[0];
  const second =
    sorted.find((t) => t.baseline_id && t.baseline_id !== first.baseline_id) ??
    sorted[1];
  return { a: first.id, b: second.id };
}

export default function DiffLandingPage() {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [empty, setEmpty] = useState(false);

  useEffect(() => {
    let cancelled = false;
    listTraces()
      .then((corpus) => {
        if (cancelled) return;
        const pair = autoSelectPair(corpus);
        if (!pair) {
          setEmpty(true);
          return;
        }
        router.replace(`/diff/${pair.a}/${pair.b}?auto=1`);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message);
      });
    return () => {
      cancelled = true;
    };
  }, [router]);

  if (error) {
    return (
      <>
        <main>
          <div className="container">
            <p className="ad-badge bad mono" style={{ padding: '40px 0' }}>
              Error: {error}
            </p>
          </div>
        </main>
        <SiteFooter />
      </>
    );
  }

  if (empty) {
    return (
      <>
        <main>
          <div className="container">
            <header className="tr-hero diff-hero">
              <div className="tr-hero-grid">
                <div>
                  <h1 className="tr-hero-title">Diff</h1>
                  <p className="tr-hero-sub">
                    Two traces, aligned step by step.
                  </p>
                </div>
              </div>
            </header>
            <div className="diff-rows">
              <div className="tr-empty">
                <div className="tr-empty-icon">
                  <DiffIcon />
                </div>
                <h4>Need at least two traces to diff.</h4>
                <p>
                  Once a second trace lands, the diff page will auto-pick a pair
                  to compare.
                </p>
                <Link href="/traces" className="ad-btn">
                  Browse traces
                </Link>
              </div>
            </div>
          </div>
        </main>
        <SiteFooter />
      </>
    );
  }

  return (
    <main>
      <div className="container">
        <p className="ad-muted mono" style={{ padding: '40px 0' }}>
          Picking two traces to diff...
        </p>
      </div>
    </main>
  );
}
