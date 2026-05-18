'use client';

import { useEffect, useState } from 'react';
import { Card, Title, Text } from '@tremor/react';
import { getTranscript } from '@/lib/api';
import type { TranscriptResponse } from '@/lib/types';

interface TranscriptProps {
  traceId: string;
}

export default function Transcript({ traceId }: TranscriptProps) {
  const [transcript, setTranscript] = useState<TranscriptResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    setTranscript(null);
    getTranscript(traceId)
      .then(setTranscript)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [traceId]);

  return (
    <Card>
      <Title>Transcript</Title>
      {loading && <Text className="mt-2">Summarizing trace...</Text>}
      {error && (
        <Text color="red" className="mt-2">
          {error}
        </Text>
      )}
      {transcript && (
        <div className="mt-3 flex flex-col gap-3">
          <Text className="text-base">{transcript.summary}</Text>
          {transcript.key_decisions.length > 0 && (
            <div>
              <Text className="text-sm font-medium text-gray-300">Key decisions</Text>
              <ul className="mt-1 list-disc pl-5 text-sm text-gray-400">
                {transcript.key_decisions.map((d, i) => (
                  <li key={i}>{d}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
