'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, TextInput, Text } from '@tremor/react';
import { promoteTrace } from '@/lib/api';

interface PromoteButtonProps {
  traceId: string;
  defaultName?: string;
}

export default function PromoteButton({ traceId, defaultName }: PromoteButtonProps) {
  const router = useRouter();
  const [expanded, setExpanded] = useState(false);
  const [name, setName] = useState(defaultName ?? `promoted-${traceId.slice(0, 8)}`);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!expanded) {
    return (
      <Button onClick={() => setExpanded(true)} size="sm" variant="secondary">
        Promote to baseline
      </Button>
    );
  }

  const handleSubmit = async () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    setSubmitting(true);
    setError(null);
    try {
      const res = await promoteTrace(traceId, trimmed);
      router.push(`/baselines/${res.baseline_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to promote');
      setSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col gap-2 rounded border border-gray-800 bg-gray-900 p-3">
      <Text className="text-sm">Promote this trace to a new baseline:</Text>
      <div className="flex items-center gap-2">
        <TextInput
          placeholder="Baseline name"
          value={name}
          onValueChange={setName}
          className="max-w-xs"
          disabled={submitting}
        />
        <Button
          onClick={handleSubmit}
          size="sm"
          disabled={submitting || !name.trim()}
        >
          {submitting ? 'Creating...' : 'Create baseline'}
        </Button>
        <Button
          onClick={() => setExpanded(false)}
          size="sm"
          variant="secondary"
          disabled={submitting}
        >
          Cancel
        </Button>
      </div>
      {error && (
        <Text color="red" className="text-sm">
          {error}
        </Text>
      )}
    </div>
  );
}
