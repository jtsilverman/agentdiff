'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import {
  Table,
  TableHead,
  TableHeaderCell,
  TableBody,
  TableRow,
  TableCell,
  Badge,
  Button,
  TextInput,
  Title,
  Text,
} from '@tremor/react';
import { listTraces, createBaseline } from '@/lib/api';
import type { TraceSummary } from '@/lib/types';
import TraceUpload from '@/components/TraceUpload';
import MetadataBadges from '@/components/MetadataBadges';

export default function TracesPage() {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [baselineName, setBaselineName] = useState('');
  const [creating, setCreating] = useState(false);

  const refresh = useCallback(() => {
    setLoading(true);
    listTraces()
      .then(setTraces)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === traces.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(traces.map((t) => t.id)));
    }
  };

  const handleCreateBaseline = async () => {
    const name = baselineName.trim();
    if (!name || selected.size === 0) return;
    setCreating(true);
    try {
      await createBaseline(name, Array.from(selected));
      setSelected(new Set());
      setBaselineName('');
      refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create baseline';
      setError(message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div>
      <Title>Traces</Title>
      <Text className="mt-1 mb-4">Upload and manage agent traces.</Text>

      <TraceUpload onUploaded={refresh} />

      {loading && <Text>Loading traces...</Text>}
      {error && <Text color="red">Error: {error}</Text>}

      {!loading && traces.length === 0 && (
        <Text className="py-8 text-center text-gray-500">No traces yet. Upload a JSONL file to get started.</Text>
      )}

      {!loading && traces.length > 0 && (
        <>
          {selected.size > 0 && (
            <div className="mb-4 flex items-center gap-3">
              <TextInput
                placeholder="Baseline name"
                value={baselineName}
                onValueChange={setBaselineName}
                className="max-w-xs"
              />
              <Button
                onClick={handleCreateBaseline}
                disabled={creating || !baselineName.trim()}
                size="sm"
              >
                {creating ? 'Creating...' : `Create Baseline (${selected.size} traces)`}
              </Button>
            </div>
          )}

          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>
                  <input
                    type="checkbox"
                    checked={selected.size === traces.length}
                    onChange={toggleAll}
                    className="rounded"
                  />
                </TableHeaderCell>
                <TableHeaderCell>Name</TableHeaderCell>
                <TableHeaderCell>Adapter</TableHeaderCell>
                <TableHeaderCell>Metadata</TableHeaderCell>
                <TableHeaderCell>Steps</TableHeaderCell>
                <TableHeaderCell>Date</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {traces.map((trace) => (
                <TableRow key={trace.id}>
                  <TableCell>
                    <input
                      type="checkbox"
                      checked={selected.has(trace.id)}
                      onChange={() => toggleSelect(trace.id)}
                      className="rounded"
                    />
                  </TableCell>
                  <TableCell>
                    <Link
                      href={`/traces/${trace.id}`}
                      className="text-tremor-brand hover:underline"
                    >
                      {trace.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge>{trace.adapter}</Badge>
                  </TableCell>
                  <TableCell>
                    <MetadataBadges metadata={trace.metadata} />
                  </TableCell>
                  <TableCell>{trace.step_count}</TableCell>
                  <TableCell className="text-sm text-gray-400">
                    {new Date(trace.created_at).toLocaleDateString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </>
      )}
    </div>
  );
}
