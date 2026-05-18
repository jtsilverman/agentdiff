'use client';

import { useEffect, useState } from 'react';
import { Card, Title, Text, Badge } from '@tremor/react';
import { getTriage } from '@/lib/api';
import type { TriageResponse, TriageClassification } from '@/lib/types';

interface TriagePanelProps {
  idA: string;
  idB: string;
}

const classificationColor: Record<TriageClassification, 'red' | 'amber' | 'emerald'> = {
  regression: 'red',
  variance: 'amber',
  additive: 'emerald',
};

export default function TriagePanel({ idA, idB }: TriagePanelProps) {
  const [triage, setTriage] = useState<TriageResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    setTriage(null);
    getTriage(idA, idB)
      .then(setTriage)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [idA, idB]);

  return (
    <Card>
      <Title>AI triage</Title>
      {loading && <Text className="mt-2">Analyzing traces...</Text>}
      {error && (
        <Text color="red" className="mt-2">
          {error}
        </Text>
      )}
      {triage && (
        <div className="mt-3 flex flex-col gap-3">
          <div>
            <Badge color={classificationColor[triage.classification]} size="sm">
              {triage.classification}
            </Badge>
          </div>
          <Text className="text-base">{triage.summary}</Text>
          <Text className="text-sm text-gray-400">
            Likely cause: {triage.likely_cause}
          </Text>
        </div>
      )}
    </Card>
  );
}
