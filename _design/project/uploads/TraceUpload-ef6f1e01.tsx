'use client';

import { useState, useCallback } from 'react';
import { TextInput, Button, Text } from '@tremor/react';
import { uploadTrace } from '@/lib/api';

interface TraceUploadProps {
  onUploaded: () => void;
}

export default function TraceUpload({ onUploaded }: TraceUploadProps) {
  const [dragging, setDragging] = useState(false);
  const [nameOverride, setNameOverride] = useState('');
  const [metadataEntries, setMetadataEntries] = useState<Array<{ key: string; value: string }>>([]);
  const [status, setStatus] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleFile = useCallback(
    async (file: File) => {
      setUploading(true);
      setStatus(null);
      try {
        const content = await file.text();
        const name = nameOverride.trim() || file.name.replace(/\.jsonl?$/, '');
        const metadata = metadataEntries
          .filter((e) => e.key.trim() !== '')
          .reduce<Record<string, string>>((acc, e) => {
            acc[e.key.trim()] = e.value;
            return acc;
          }, {});
        await uploadTrace(content, name, undefined, Object.keys(metadata).length > 0 ? metadata : undefined);
        setStatus({ type: 'success', message: `Uploaded "${name}" successfully.` });
        setNameOverride('');
        setMetadataEntries([]);
        onUploaded();
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Upload failed';
        setStatus({ type: 'error', message });
      } finally {
        setUploading(false);
      }
    },
    [nameOverride, metadataEntries, onUploaded],
  );

  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(true);
  }, []);

  const onDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
  }, []);

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragging(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile],
  );

  return (
    <div className="mb-6">
      <div className="mb-3">
        <TextInput
          placeholder="Name override (optional, defaults to filename)"
          value={nameOverride}
          onValueChange={setNameOverride}
        />
      </div>
      <div className="mb-3 space-y-2">
        <Button
          variant="secondary"
          size="xs"
          onClick={() => setMetadataEntries([...metadataEntries, { key: '', value: '' }])}
        >
          Add metadata
        </Button>
        {metadataEntries.map((entry, i) => (
          <div key={i} className="flex items-center gap-2">
            <TextInput
              placeholder="key"
              value={entry.key}
              onValueChange={(v) => {
                const updated = [...metadataEntries];
                updated[i] = { ...updated[i], key: v };
                setMetadataEntries(updated);
              }}
            />
            <TextInput
              placeholder="value"
              value={entry.value}
              onValueChange={(v) => {
                const updated = [...metadataEntries];
                updated[i] = { ...updated[i], value: v };
                setMetadataEntries(updated);
              }}
            />
            <Button
              variant="secondary"
              size="xs"
              onClick={() => setMetadataEntries(metadataEntries.filter((_, j) => j !== i))}
            >
              x
            </Button>
          </div>
        ))}
      </div>
      <div
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
        className={`flex cursor-pointer items-center justify-center rounded-lg border-2 border-dashed p-8 text-center transition-colors ${
          dragging
            ? 'border-tremor-brand bg-tremor-brand/10 text-white'
            : 'border-gray-700 text-gray-400 hover:border-gray-500'
        }`}
      >
        {uploading ? (
          <Text>Uploading...</Text>
        ) : (
          <Text>Drop JSONL file here</Text>
        )}
      </div>
      {status && (
        <Text className={`mt-2 text-sm ${status.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
          {status.message}
        </Text>
      )}
    </div>
  );
}
