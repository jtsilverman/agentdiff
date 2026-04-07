import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockStrategyReport } from '@/test/mocks/fixtures';
import type { StrategyReport } from '@/lib/types';
import StrategyCluster from '../StrategyCluster';

describe('StrategyCluster', () => {
  it('renders baseline name as title', () => {
    render(<StrategyCluster report={mockStrategyReport} />);
    expect(screen.getByText('test-baseline')).toBeInTheDocument();
  });

  it('shows snapshot count, strategy count, noise count, epsilon in summary text', () => {
    render(<StrategyCluster report={mockStrategyReport} />);
    expect(
      screen.getByText('10 snapshots, 1 strategies, 1 noise traces (epsilon=0.3)'),
    ).toBeInTheDocument();
  });

  it('renders strategy cards with badges, member counts, exemplar, and tool sequence', () => {
    render(<StrategyCluster report={mockStrategyReport} />);
    expect(screen.getByText('Strategy 0')).toBeInTheDocument();
    expect(screen.getByText('7 members')).toBeInTheDocument();
    expect(screen.getByText('trace-1')).toBeInTheDocument();
    expect(screen.getByText('read_file')).toBeInTheDocument();
    expect(screen.getByText('write_file')).toBeInTheDocument();
  });

  it('renders noise traces section when noise.length > 0; hides when empty', () => {
    // With noise
    const { unmount } = render(<StrategyCluster report={mockStrategyReport} />);
    expect(screen.getByText('Noise Traces')).toBeInTheDocument();
    expect(screen.getByText('trace-outlier')).toBeInTheDocument();
    unmount();

    // Without noise
    const noNoiseReport: StrategyReport = {
      ...mockStrategyReport,
      noise: [],
    };
    render(<StrategyCluster report={noNoiseReport} />);
    expect(screen.queryByText('Noise Traces')).not.toBeInTheDocument();
  });
});
