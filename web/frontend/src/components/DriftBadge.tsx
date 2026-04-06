'use client';

import { Badge } from '@tremor/react';
import type { MatchResult } from '@/lib/types';

export default function DriftBadge({ result }: { result: MatchResult }) {
  return (
    <div className="flex items-center gap-3">
      {result.matched ? (
        <Badge color="green">Matches Strategy {result.strategy_id}</Badge>
      ) : (
        <Badge color="red">New Strategy Detected</Badge>
      )}
      <span className="text-sm text-gray-400">
        distance: {result.distance.toFixed(3)}
      </span>
    </div>
  );
}
