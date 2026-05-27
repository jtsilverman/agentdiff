'use client';

import { Card, Badge, Text, Title } from '@tremor/react';
import type { StrategyReport } from '@/lib/types';

const COLORS: Array<'blue' | 'green' | 'amber' | 'purple' | 'rose' | 'cyan'> = [
  'blue',
  'green',
  'amber',
  'purple',
  'rose',
  'cyan',
];

export default function StrategyCluster({ report }: { report: StrategyReport }) {
  return (
    <div className="flex flex-col gap-6">
      <div>
        <Title>{report.baseline_name}</Title>
        <Text className="mt-1">
          {report.snapshot_count} snapshots, {report.strategies.length} strategies,{' '}
          {report.noise.length} noise traces (epsilon={report.epsilon})
        </Text>
      </div>

      {report.strategies.map((strategy) => {
        const color = COLORS[strategy.id % COLORS.length];
        return (
          <Card key={strategy.id}>
            <div className="flex items-center gap-3">
              <Badge color={color}>Strategy {strategy.id}</Badge>
              <Text>{strategy.count} members</Text>
            </div>
            <Text className="mt-2">
              Exemplar: <span className="font-mono">{strategy.exemplar}</span>
            </Text>
            <div className="mt-3 flex flex-wrap gap-1">
              {strategy.tool_sequence.map((tool, i) => (
                <Badge key={i} color={color} size="sm">
                  {tool}
                </Badge>
              ))}
            </div>
            {strategy.metadata_summary &&
              Object.keys(strategy.metadata_summary).length > 0 && (
                <div className="mt-3 space-y-1">
                  <Text className="text-xs text-gray-400">Metadata</Text>
                  {Object.entries(strategy.metadata_summary).map(([key, dist]) => {
                    const values = Object.entries(dist);
                    const isUnanimous = values.length === 1;
                    return (
                      <div key={key} className="flex items-center gap-2">
                        <Text className="text-xs text-gray-500">{key}:</Text>
                        <div className="flex flex-wrap gap-1">
                          {values.map(([val, count]) => (
                            <Badge
                              key={val}
                              color={isUnanimous ? color : 'gray'}
                              size="sm"
                            >
                              {val} ({count})
                            </Badge>
                          ))}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
          </Card>
        );
      })}

      {report.noise.length > 0 && (
        <Card className="border-gray-700 bg-gray-900">
          <Title className="text-gray-400">Noise Traces</Title>
          <div className="mt-2 flex flex-col gap-1">
            {report.noise.map((name) => (
              <Text key={name} className="font-mono text-gray-500">
                {name}
              </Text>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
