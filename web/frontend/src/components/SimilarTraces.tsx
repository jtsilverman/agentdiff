'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Title, Text } from '@tremor/react';
import { getSimilar } from '@/lib/api';
import type { SimilarTracesResponse } from '@/lib/types';

interface SimilarTracesProps {
  traceId: string;
}

export default function SimilarTraces({ traceId }: SimilarTracesProps) {
  const [data, setData] = useState<SimilarTracesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    setData(null);
    getSimilar(traceId)
      .then(setData)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [traceId]);

  return (
    <Card>
      <Title>Similar traces</Title>
      {loading && <Text className="mt-2">Finding similar traces...</Text>}
      {error && (
        <Text color="red" className="mt-2">
          {error}
        </Text>
      )}
      {data && data.matches.length === 0 && (
        <Text className="mt-2 text-sm text-gray-400">
          No matches yet. Upload more traces (with a Voyage API key configured) to populate this panel.
        </Text>
      )}
      {data && data.matches.length > 0 && (
        <ul className="mt-3 flex flex-col gap-2">
          {data.matches.map((m) => (
            <li
              key={m.trace_id}
              className="flex items-center justify-between rounded border border-gray-800 bg-gray-900/50 p-2 hover:bg-gray-900"
            >
              <Link
                href={`/traces/${m.trace_id}`}
                className="flex-1 text-sm text-blue-400 hover:underline"
              >
                {m.name}
              </Link>
              <Text className="text-sm tabular-nums text-gray-400">
                {m.similarity_score.toFixed(2)}
              </Text>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
