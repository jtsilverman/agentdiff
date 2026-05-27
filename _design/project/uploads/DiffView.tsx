'use client';

import { Badge } from '@tremor/react';
import type { DiffResponse, Step } from '@/lib/types';

const OP_BG: Record<string, { left: string; right: string }> = {
  match: { left: 'bg-green-950/40', right: 'bg-green-950/40' },
  substitute: { left: 'bg-yellow-950/40', right: 'bg-yellow-950/40' },
  delete: { left: 'bg-red-950/40', right: '' },
  insert: { left: '', right: 'bg-red-950/40' },
};

function StepCell({ step }: { step: Step | null }) {
  if (!step) {
    return <div className="p-3 text-gray-600 text-sm italic">-</div>;
  }

  return (
    <div className="p-3">
      <span className="text-xs font-medium text-gray-400">{step.role}</span>
      {step.content && (
        <p className="mt-1 text-sm text-gray-300 line-clamp-4">{step.content}</p>
      )}
      {step.tool_call && (
        <p className="mt-1 text-xs text-amber-400">{step.tool_call.name}()</p>
      )}
      {step.tool_result && (
        <p className="mt-1 text-xs text-purple-400">{step.tool_result.name}: {step.tool_result.output.slice(0, 120)}</p>
      )}
    </div>
  );
}

export default function DiffView({ diff }: { diff: DiffResponse }) {
  const { summary, alignment } = diff;

  return (
    <div>
      {/* Summary bar */}
      <div className="mb-4 flex flex-wrap gap-2">
        <Badge color="green">{summary.matches} matches</Badge>
        <Badge color="yellow">{summary.substitutions} substitutions</Badge>
        <Badge color="red">{summary.deletions} deletions</Badge>
        <Badge color="blue">{summary.insertions} insertions</Badge>
        <Badge color="gray">distance: {diff.distance}</Badge>
      </div>

      {/* Two-column diff */}
      <div className="grid grid-cols-2 gap-px rounded border border-gray-800 bg-gray-800">
        {/* Header */}
        <div className="bg-gray-900 p-2 text-sm font-medium text-gray-300">
          {diff.trace_a.name}
        </div>
        <div className="bg-gray-900 p-2 text-sm font-medium text-gray-300">
          {diff.trace_b.name}
        </div>

        {/* Rows */}
        {alignment.map((pair, i) => {
          const bg = OP_BG[pair.op] ?? { left: '', right: '' };
          return (
            <div key={i} className="contents">
              <div className={`${bg.left} border-t border-gray-800`}>
                <StepCell step={pair.a_step} />
              </div>
              <div className={`${bg.right} border-t border-gray-800`}>
                <StepCell step={pair.b_step} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
