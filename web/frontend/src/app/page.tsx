'use client';

import { useEffect, useState } from 'react';
import Hero from '@/components/home/Hero';
import Examples, { EXAMPLES, resolveHref } from '@/components/home/Examples';
import SiteFooter from '@/components/SiteFooter';
import { listBaselines } from '@/lib/api';
import type { BaselineSummary } from '@/lib/types';

export default function HomePage() {
  const [baselines, setBaselines] = useState<BaselineSummary[]>([]);

  useEffect(() => {
    listBaselines()
      .then(setBaselines)
      .catch(() => setBaselines([]));
  }, []);

  const firstBaselineHref = resolveHref(EXAMPLES[0], 0, baselines);

  return (
    <>
      <Hero firstBaselineHref={firstBaselineHref} />
      <Examples baselines={baselines} />
      <SiteFooter />
    </>
  );
}
