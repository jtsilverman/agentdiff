'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Title, Text, Grid } from '@tremor/react';
import { listBaselines } from '@/lib/api';
import type { BaselineSummary } from '@/lib/types';

const SEED_PREFIX = 'seed-';

// Maps the machine-readable seed baseline name to a human-readable button
// label. Names not in this map fall back to the raw baseline name with the
// prefix stripped, so adding a new scenario in seed.go doesn't break the UI.
const SEED_LABELS: Record<string, string> = {
  'seed-tool-order-stable': 'Tool-order stable',
  'seed-tool-order-variance': 'Tool-order variance',
  'seed-prompt-regression': 'Prompt-change regression',
  'seed-novel-tool-discovery': 'Novel tool discovery',
  'seed-noise-and-strategies': 'Noise and strategies',
};

function seedLabel(name: string): string {
  return SEED_LABELS[name] ?? name.replace(SEED_PREFIX, '').replace(/-/g, ' ');
}

export default function HomePage() {
  const [baselines, setBaselines] = useState<BaselineSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listBaselines()
      .then(setBaselines)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <Text>Loading baselines...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (baselines.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 py-20">
        <Title>No Baselines Yet</Title>
        <Text>Upload traces and create a baseline to get started.</Text>
      </div>
    );
  }

  const seeded = baselines.filter((b) => b.name.startsWith(SEED_PREFIX));

  return (
    <div>
      {seeded.length > 0 && (
        <div className="mb-10">
          <Title>Try these examples</Title>
          <Text className="mt-1">
            Pre-loaded scenarios that showcase agentdiff features.
          </Text>
          <div className="mt-4 flex flex-wrap gap-3">
            {seeded.map((b) => (
              <Link
                key={b.id}
                href={`/baselines/${b.id}`}
                className="rounded-md border border-tremor-brand bg-tremor-brand-faint px-4 py-2 text-sm font-medium text-tremor-brand hover:bg-tremor-brand-muted"
              >
                {seedLabel(b.name)}
              </Link>
            ))}
          </div>
        </div>
      )}

      <Title>Baselines</Title>
      <Text className="mt-1">Select a baseline to view clustered strategies.</Text>
      <Grid numItems={1} numItemsSm={2} numItemsLg={3} className="mt-6 gap-6">
        {baselines.map((b) => (
          <Link key={b.id} href={`/baselines/${b.id}`}>
            <Card className="cursor-pointer transition-colors hover:bg-gray-800">
              <Title>{b.name}</Title>
              <Text className="mt-2">{b.trace_count} traces</Text>
              <Text className="mt-1 text-sm text-tremor-brand">
                View Clusters &rarr;
              </Text>
            </Card>
          </Link>
        ))}
      </Grid>
    </div>
  );
}
