'use client';

import { useEffect, useMemo, useState } from 'react';
import { useParams } from 'next/navigation';
import { Card, Title, Text } from '@tremor/react';
import {
  getCluster,
  getGraph,
  getOverlay,
  compareTrace,
} from '@/lib/api';
import type {
  StrategyReport,
  MatchResult,
  PathGraph as PathGraphData,
  OverlayResult,
} from '@/lib/types';
import PathGraph, { type PathGraphMode } from '@/components/PathGraph';
import StrategyCluster from '@/components/StrategyCluster';
import DriftBadge from '@/components/DriftBadge';
import TriagePanel from '@/components/TriagePanel';

export default function BaselineDetailPage() {
  const params = useParams<{ id: string }>();
  const baselineId = params.id;

  const [report, setReport] = useState<StrategyReport | null>(null);
  const [graph, setGraph] = useState<PathGraphData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [overlay, setOverlay] = useState<OverlayResult | null>(null);
  const [overlayError, setOverlayError] = useState<string | null>(null);

  const [traceId, setTraceId] = useState('');
  const [comparing, setComparing] = useState(false);
  const [matchResult, setMatchResult] = useState<MatchResult | null>(null);
  const [compareError, setCompareError] = useState<string | null>(null);

  const [graphMode, setGraphMode] = useState<PathGraphMode>('overlay');

  useEffect(() => {
    Promise.all([
      getCluster(baselineId).then(setReport),
      getGraph(baselineId).then(setGraph),
    ])
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [baselineId]);

  const traceIds = useMemo(() => {
    if (!report) return [];
    const fromStrategies = report.strategies.flatMap((s) => s.members);
    return [...fromStrategies, ...report.noise];
  }, [report]);

  const handleTraceSelect = async (id: string) => {
    setSelectedTraceId(id);
    setOverlay(null);
    setOverlayError(null);
    try {
      const result = await getOverlay(baselineId, id);
      setOverlay(result);
    } catch (err) {
      setOverlayError(err instanceof Error ? err.message : 'Overlay failed');
    }
  };

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
    return <Text>Loading baseline...</Text>;
  }

  if (error) {
    return <Text color="red">Error: {error}</Text>;
  }

  if (!report || !graph) {
    return <Text>No baseline data found.</Text>;
  }

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <Title>Path graph</Title>
        <Text className="mt-1">
          {graph.stats.total_runs} runs, {graph.stats.branch_points} branch points.
          {selectedTraceId && ` Overlay: ${selectedTraceId}.`}
        </Text>
        <div className="mt-3 flex gap-2">
          {(
            [
              { value: 'overlay', label: 'Overlay' },
              { value: 'heatmap-cost', label: 'Heatmap: cost' },
              { value: 'heatmap-latency', label: 'Heatmap: latency' },
            ] as const
          ).map((opt) => {
            const active = graphMode === opt.value;
            return (
              <button
                key={opt.value}
                onClick={() => setGraphMode(opt.value)}
                className={`rounded px-3 py-1 text-xs transition-colors ${
                  active
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-800 text-gray-100 hover:bg-gray-700'
                }`}
              >
                {opt.label}
              </button>
            );
          })}
        </div>
        <div className="mt-4">
          <PathGraph
            graph={graph}
            overlay={graphMode === 'overlay' ? overlay ?? undefined : undefined}
            mode={graphMode}
          />
        </div>
        {overlayError && (
          <Text color="red" className="mt-2">
            Overlay error: {overlayError}
          </Text>
        )}
      </Card>

      {(() => {
        const exemplar = report.strategies[0]?.exemplar;
        if (!selectedTraceId || !exemplar || selectedTraceId === exemplar) return null;
        return <TriagePanel idA={exemplar} idB={selectedTraceId} />;
      })()}

      <Card>
        <Title>Traces in this baseline</Title>
        <Text className="mt-1">
          Click a trace to overlay its path on the graph above.
        </Text>
        {traceIds.length === 0 ? (
          <Text className="mt-3">No traces in this baseline.</Text>
        ) : (
          <div className="mt-3 flex flex-wrap gap-2">
            {traceIds.map((id) => {
              const selected = id === selectedTraceId;
              return (
                <button
                  key={id}
                  onClick={() => handleTraceSelect(id)}
                  className={`rounded px-3 py-1 text-sm transition-colors ${
                    selected
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-800 text-gray-100 hover:bg-gray-700'
                  }`}
                >
                  {id}
                </button>
              );
            })}
          </div>
        )}
      </Card>

      <Card>
        <Title>Strategies</Title>
        <Text className="mt-1">Text-based cluster view (legacy).</Text>
        <div className="mt-4">
          <StrategyCluster report={report} />
        </div>
      </Card>

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
