import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { mockStrategyReport, mockMatchResult } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getCluster, compareTrace } from '@/lib/api';
import BaselineDetailPage from '../baselines/[id]/page';

const mockedGetCluster = vi.mocked(getCluster);
const mockedCompareTrace = vi.mocked(compareTrace);

describe('BaselineDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'bl-1' });
  });

  it('shows loading, then renders StrategyCluster with report data', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);
    expect(screen.getByText('Loading cluster data...')).toBeInTheDocument();

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
});
