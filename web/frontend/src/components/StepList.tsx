'use client';

import { useState } from 'react';
import { Badge } from '@tremor/react';
import type { Step } from '@/lib/types';

const ROLE_COLORS: Record<string, string> = {
  user: 'bg-blue-600',
  assistant: 'bg-green-600',
  tool_call: 'bg-amber-600',
  tool_result: 'bg-purple-600',
};

function Collapsible({ label, children }: { label: string; children: React.ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-1">
      <button
        onClick={() => setOpen(!open)}
        className="text-xs text-gray-400 hover:text-gray-200"
      >
        {open ? `Hide ${label}` : `Show ${label}`}
      </button>
      {open && (
        <pre className="mt-1 max-h-60 overflow-auto rounded bg-gray-900 p-2 text-xs text-gray-300">
          {children}
        </pre>
      )}
    </div>
  );
}

function StepItem({ step, index }: { step: Step; index: number }) {
  const bgClass = ROLE_COLORS[step.role] ?? 'bg-gray-600';
  const roleLabel = step.role.replace('_', ' ');

  return (
    <div className="rounded border border-gray-800 bg-gray-900/50 p-3">
      <div className="flex items-center gap-2">
        <span className="text-xs text-gray-500">#{index + 1}</span>
        <Badge className={bgClass}>{roleLabel}</Badge>
      </div>

      {step.content && (
        <p className="mt-2 whitespace-pre-wrap text-sm text-gray-300">{step.content}</p>
      )}

      {step.tool_call && (
        <div className="mt-2">
          <span className="text-xs font-medium text-amber-400">{step.tool_call.name}</span>
          <Collapsible label="args">
            {JSON.stringify(step.tool_call.args, null, 2)}
          </Collapsible>
        </div>
      )}

      {step.tool_result && (
        <div className="mt-2">
          <span className={`text-xs font-medium ${step.tool_result.is_error ? 'text-red-400' : 'text-purple-400'}`}>
            {step.tool_result.name}
          </span>
          <Collapsible label="output">
            {step.tool_result.output}
          </Collapsible>
        </div>
      )}
    </div>
  );
}

export default function StepList({ steps }: { steps: Step[] }) {
  return (
    <div className="flex flex-col gap-2">
      {steps.map((step, i) => (
        <StepItem key={i} step={step} index={i} />
      ))}
    </div>
  );
}
