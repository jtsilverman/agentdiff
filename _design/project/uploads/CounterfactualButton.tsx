'use client';

import { useState } from 'react';
import { Button, Textarea, Text } from '@tremor/react';
import { runCounterfactual } from '@/lib/api';
import type { Step, CounterfactualResponse } from '@/lib/types';
import CounterfactualGraph from './CounterfactualGraph';

interface CounterfactualButtonProps {
  traceId: string;
  steps: Step[];
}

function stepLabel(step: Step, index: number): string {
  const kind = step.tool_call?.name ?? step.tool_result?.name ?? step.role;
  return `${index}: ${kind}`;
}

export default function CounterfactualButton({ traceId, steps }: CounterfactualButtonProps) {
  const [expanded, setExpanded] = useState(false);
  const [stepIndex, setStepIndex] = useState(0);
  const [modifiedInput, setModifiedInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CounterfactualResponse | null>(null);

  if (!expanded) {
    return (
      <Button onClick={() => setExpanded(true)} size="sm" variant="secondary">
        What if?
      </Button>
    );
  }

  const handleSubmit = async () => {
    const input = modifiedInput.trim();
    if (!input) return;
    setSubmitting(true);
    setError(null);
    setResult(null);
    try {
      const res = await runCounterfactual(traceId, stepIndex, input);
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed to run counterfactual');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex flex-col gap-3 rounded border border-gray-800 bg-gray-900 p-3">
      <Text className="text-sm">Replay this trace from a chosen step with modified input:</Text>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
        <label className="flex flex-col text-xs text-gray-300">
          Step
          <select
            aria-label="Step"
            value={stepIndex}
            onChange={(e) => setStepIndex(Number(e.target.value))}
            disabled={submitting}
            className="mt-1 rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-gray-100"
          >
            {steps.map((s, i) => (
              <option key={i} value={i}>
                {stepLabel(s, i)}
              </option>
            ))}
          </select>
        </label>
        <div className="flex-1">
          <Textarea
            placeholder="Modified input for this step"
            value={modifiedInput}
            onValueChange={setModifiedInput}
            disabled={submitting}
            rows={2}
          />
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button
          onClick={handleSubmit}
          size="sm"
          disabled={submitting || !modifiedInput.trim()}
        >
          {submitting ? 'Running...' : 'Run counterfactual'}
        </Button>
        <Button
          onClick={() => {
            setExpanded(false);
            setResult(null);
            setError(null);
          }}
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
      {result && (
        <div className="mt-2">
          <CounterfactualGraph comparison={result.comparison} />
        </div>
      )}
    </div>
  );
}
