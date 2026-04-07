'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { Title, Text, Badge } from '@tremor/react';
import { getDiff, getTrace } from '@/lib/api';
import type { DiffResponse, TraceDetail } from '@/lib/types';
import DiffView from '@/components/DiffView';

export default function DiffPage() {
  const params = useParams<{ idA: string; idB: string }>();
  const [diff, setDiff] = useState<DiffResponse | null>(null);
  const [traceA, setTraceA] = useState<TraceDetail | null>(null);
  const [traceB, setTraceB] = useState<TraceDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!params.idA || !params.idB) return;
    Promise.all([
      getDiff(params.idA, params.idB),
      getTrace(params.idA),
      getTrace(params.idB),
    ])
      .then(([d, a, b]) => {
        setDiff(d);
        setTraceA(a);
        setTraceB(b);
      })
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
        {traceA && traceB && (() => {
          const metaA = traceA.metadata ?? {};
          const metaB = traceB.metadata ?? {};
          const allKeys = Array.from(new Set([...Object.keys(metaA), ...Object.keys(metaB)]));
          if (allKeys.length === 0) return null;
          return (
            <div className="mt-3 grid grid-cols-2 gap-4">
              <div>
                <Text className="mb-1 text-sm text-gray-400">{diff.trace_a.name}</Text>
                <div className="flex flex-wrap gap-1">
                  {allKeys.map((key) => (
                    <Badge
                      key={key}
                      color={metaA[key] !== metaB[key] ? 'amber' : 'gray'}
                      size="sm"
                    >
                      {key}: {metaA[key] ?? '—'}
                    </Badge>
                  ))}
                </div>
              </div>
              <div>
                <Text className="mb-1 text-sm text-gray-400">{diff.trace_b.name}</Text>
                <div className="flex flex-wrap gap-1">
                  {allKeys.map((key) => (
                    <Badge
                      key={key}
                      color={metaA[key] !== metaB[key] ? 'amber' : 'gray'}
                      size="sm"
                    >
                      {key}: {metaB[key] ?? '—'}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          );
        })()}
      </div>
      <DiffView diff={diff} />
    </div>
  );
}
