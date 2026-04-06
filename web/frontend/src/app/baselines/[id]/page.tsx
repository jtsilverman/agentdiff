'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { Card, Title, Text } from '@tremor/react';
import { getCluster, compareTrace } from '@/lib/api';
import type { StrategyReport, MatchResult } from '@/lib/types';
import StrategyCluster from '@/components/StrategyCluster';
import DriftBadge from '@/components/DriftBadge';

export default function BaselineDetailPage() {
  const params = useParams<{ id: string }>();
  const baselineId = params.id;

  const [report, setReport] = useState<StrategyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [traceId, setTraceId] = useState('');
  const [comparing, setComparing] = useState(false);
  const [matchResult, setMatchResult] = useState<MatchResult | null>(null);
  const [compareError, setCompareError] = useState<string | null>(null);

  useEffect(() => {
    getCluster(baselineId)
      .then(setReport)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [baselineId]);

  const handleCompare = async () => {
    if (!traceId.trim()) return;
    setComparing(true);
    setMatchResult(null);
    setCompareError(null);
    try {
      const result = await compareTrace(baselineId, traceId.trim());
      setMatchResult(result);
    } catch (err) {
      setCompareError(err instanceof Error ? err.message : 'Compare failed');
    } finally {
      setComparing(false);
    }
  };

  if (loading) {
    return <Text>Loading cluster data...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (!report) {
    return <Text>No cluster data found.</Text>;
  }

  return (
    <div className="flex flex-col gap-8">
      <StrategyCluster report={report} />

      <Card>
        <Title>Compare Trace</Title>
        <Text className="mt-1">
          Check if a trace matches an existing strategy or represents drift.
        </Text>
        <div className="mt-4 flex gap-3">
          <input
            type="text"
            value={traceId}
            onChange={(e) => setTraceId(e.target.value)}
            placeholder="Enter trace ID"
            className="flex-1 rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-500 focus:border-blue-500 focus:outline-none"
          />
          <button
            onClick={handleCompare}
            disabled={comparing || !traceId.trim()}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-500 disabled:opacity-50"
          >
            {comparing ? 'Comparing...' : 'Compare'}
          </button>
        </div>
        {matchResult && (
          <div className="mt-4">
            <DriftBadge result={matchResult} />
          </div>
        )}
        {compareError && (
          <Text color="red" className="mt-4">
            Error: {compareError}
          </Text>
        )}
      </Card>
    </div>
  );
}
