'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { Title, Text } from '@tremor/react';
import { getDiff } from '@/lib/api';
import type { DiffResponse } from '@/lib/types';
import DiffView from '@/components/DiffView';

export default function DiffPage() {
  const params = useParams<{ idA: string; idB: string }>();
  const [diff, setDiff] = useState<DiffResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!params.idA || !params.idB) return;
    getDiff(params.idA, params.idB)
      .then(setDiff)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [params.idA, params.idB]);

  if (loading) return <Text>Loading diff...</Text>;
  if (error) return <Text color="red">Error: {error}</Text>;
  if (!diff) return <Text>No diff data.</Text>;

  return (
    <div>
      <div className="mb-6">
        <Title>
          Diff: {diff.trace_a.name} vs {diff.trace_b.name}
        </Title>
      </div>
      <DiffView diff={diff} />
    </div>
  );
}
