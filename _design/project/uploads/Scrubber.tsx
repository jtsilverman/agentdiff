'use client';

import { useState } from 'react';
import { Card, Title, Text, Badge } from '@tremor/react';
import type { Step } from '@/lib/types';

interface ScrubberProps {
  steps: Step[];
}

export default function Scrubber({ steps }: ScrubberProps) {
  const [index, setIndex] = useState(0);

  if (steps.length === 0) {
    return (
      <Card>
        <Title>Replay</Title>
        <Text className="mt-2 text-gray-500">No steps to replay.</Text>
      </Card>
    );
  }

  const max = steps.length - 1;
  const clamped = Math.min(Math.max(index, 0), max);
  const step = steps[clamped];

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = Number(e.target.value);
    if (Number.isNaN(raw)) return;
    setIndex(Math.min(Math.max(raw, 0), max));
  };

  return (
    <Card>
      <div className="flex items-center justify-between">
        <Title>Replay</Title>
        <Text className="text-sm text-gray-400">
          Step {clamped + 1} of {steps.length}
        </Text>
      </div>

      <div className="mt-3">
        <input
          type="range"
          aria-label="step scrubber"
          min={0}
          max={max}
          value={clamped}
          onChange={handleChange}
          className="w-full accent-purple-500"
        />
      </div>

      <div className="mt-4 flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <Badge>{step.role}</Badge>
        </div>

        {step.content && (
          <div>
            <Text className="text-sm font-medium text-gray-300">Context</Text>
            <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-gray-950 p-3 text-xs text-gray-200">
              {step.content}
            </pre>
          </div>
        )}

        {step.tool_call && (
          <div>
            <Text className="text-sm font-medium text-gray-300">Tool call</Text>
            <div className="mt-1 text-sm text-purple-400">{step.tool_call.name}</div>
            <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-gray-950 p-3 text-xs text-gray-200">
              {JSON.stringify(step.tool_call.args, null, 2)}
            </pre>
          </div>
        )}

        {step.tool_result && (
          <div>
            <Text className="text-sm font-medium text-gray-300">Tool output</Text>
            <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-gray-950 p-3 text-xs text-gray-200">
              {step.tool_result.output}
            </pre>
          </div>
        )}
      </div>
    </Card>
  );
}
