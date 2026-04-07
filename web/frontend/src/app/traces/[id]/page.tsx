'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Title, Text, Badge, Button, TextInput } from '@tremor/react';
import { getTrace } from '@/lib/api';
import type { TraceDetail } from '@/lib/types';
import StepList from '@/components/StepList';
import MetadataBadges from '@/components/MetadataBadges';

export default function TraceDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [diffId, setDiffId] = useState('');

  useEffect(() => {
    if (!params.id) return;
    getTrace(params.id)
      .then(setTrace)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [params.id]);

  const handleCompare = () => {
    const other = diffId.trim();
    if (!other || !params.id) return;
    router.push(`/diff/${params.id}/${other}`);
  };

  if (loading) return <Text>Loading trace...</Text>;
  if (error) return <Text color="red">Error: {error}</Text>;
  if (!trace) return <Text>Trace not found.</Text>;

  return (
    <div>
      <div className="mb-6">
        <Title>{trace.name}</Title>
        <div className="mt-2 flex items-center gap-3">
          <Badge>{trace.adapter}</Badge>
          <Text>{trace.steps.length} steps</Text>
          <Text className="text-sm text-gray-500">
            {new Date(trace.created_at).toLocaleString()}
          </Text>
        </div>
        {trace.metadata && Object.keys(trace.metadata).length > 0 && (
          <div className="mt-3">
            <MetadataBadges metadata={trace.metadata} />
          </div>
        )}
      </div>

      {/* Diff launcher */}
      <div className="mb-6 flex items-center gap-3 rounded border border-gray-800 bg-gray-900 p-3">
        <Text className="shrink-0 text-sm">Diff with...</Text>
        <TextInput
          placeholder="Second trace ID"
          value={diffId}
          onValueChange={setDiffId}
          className="max-w-xs"
        />
        <Button onClick={handleCompare} size="sm" disabled={!diffId.trim()}>
          Compare
        </Button>
      </div>

      <StepList steps={trace.steps} />
    </div>
  );
}
