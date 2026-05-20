import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import {
  mockStrategyReport,
  mockMatchResult,
  mockPathGraph,
  mockOverlay,
} from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import {
  getCluster,
  compareTrace,
  getGraph,
  getOverlay,
} from '@/lib/api';
import BaselineDetailPage from '../baselines/[id]/page';

const mockedGetCluster = vi.mocked(getCluster);
const mockedCompareTrace = vi.mocked(compareTrace);
const mockedGetGraph = vi.mocked(getGraph);
const mockedGetOverlay = vi.mocked(getOverlay);

describe('BaselineDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'bl-1' });
    mockedGetGraph.mockResolvedValue(mockPathGraph);
    mockedGetOverlay.mockResolvedValue(mockOverlay);
  });

  it('shows loading, then renders StrategyCluster with report data', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);
    expect(screen.getByText('Loading baseline...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });
    // StrategyCluster renders strategy details
    expect(screen.getByText('Strategy 0')).toBeInTheDocument();
  });

  it('entering trace ID and clicking Compare calls compareTrace', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockResolvedValue(mockMatchResult);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(mockedCompareTrace).toHaveBeenCalledWith('bl-1', 'trace-99');
    });
  });

  it('shows DriftBadge when compare succeeds', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockResolvedValue(mockMatchResult);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(screen.getByText('Matches Strategy 0')).toBeInTheDocument();
    });
  });

  it('shows error message when compare fails', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockRejectedValue(new Error('Compare exploded'));
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(screen.getByText('Error: Compare exploded')).toBeInTheDocument();
    });
  });

  it('renders the PathGraph hero with nodes from getGraph', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);

    // 'bash' is unique to the PathGraph fixture (not in the cluster fixture's
    // tool_sequence), so it gates on PathGraph having actually rendered.
    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });
  });

  it('renders a heatmap-mode toggle and applies heatmap buckets when Cost is selected', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });

    const costBtn = screen.getByRole('button', { name: /heatmap.*cost/i });
    fireEvent.click(costBtn);

    await waitFor(() => {
      expect(
        container.querySelector('[data-heatmap-bucket="hot"]'),
      ).not.toBeNull();
    });
  });

  it('clicking a trace from the list calls getOverlay and applies overlay state', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });

    // mockStrategyReport.strategies[0].members[0] === 'trace-1'
    const traceBtn = await screen.findByRole('button', { name: /trace-1/ });
    fireEvent.click(traceBtn);

    await waitFor(() => {
      expect(mockedGetOverlay).toHaveBeenCalledWith('bl-1', 'trace-1');
    });
    await waitFor(() => {
      expect(
        container.querySelector('[data-overlay-state="divergent"]'),
      ).not.toBeNull();
    });
  });
});
