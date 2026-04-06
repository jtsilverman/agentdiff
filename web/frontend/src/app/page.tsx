'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Title, Text, Grid } from '@tremor/react';
import { listBaselines } from '@/lib/api';
import type { BaselineSummary } from '@/lib/types';

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

  return (
    <div>
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
