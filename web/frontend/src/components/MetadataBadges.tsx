'use client';

import { Badge } from '@tremor/react';

interface MetadataBadgesProps {
  metadata?: Record<string, string>;
}

export default function MetadataBadges({ metadata }: MetadataBadgesProps) {
  if (!metadata || Object.keys(metadata).length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1">
      {Object.entries(metadata).map(([key, value]) => (
        <Badge key={key} color="gray" size="sm">
          {key}: {value}
        </Badge>
      ))}
    </div>
  );
}
