import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams } from '@/test/mocks/next-navigation';
import { render, screen, waitFor } from '@testing-library/react';
import { mockDiff } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getDiff } from '@/lib/api';
import DiffPage from '../diff/[idA]/[idB]/page';

const mockedGetDiff = vi.mocked(getDiff);

describe('DiffPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ idA: 'a1', idB: 'b1' });
  });

  it('shows loading, then renders diff title with both trace names', async () => {
    mockedGetDiff.mockResolvedValue(mockDiff);
    render(<DiffPage />);
    expect(screen.getByText('Loading diff...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Diff: trace-a vs trace-b')).toBeInTheDocument();
    });
  });

  it('renders DiffView component with diff data', async () => {
    mockedGetDiff.mockResolvedValue(mockDiff);
    render(<DiffPage />);

    await waitFor(() => {
      expect(screen.getByText('Diff: trace-a vs trace-b')).toBeInTheDocument();
    });
    // DiffView renders alignment rows - "Hello" appears on both sides of the match
    expect(screen.getAllByText('Hello')).toHaveLength(2);
  });

  it('shows error when getDiff rejects', async () => {
    mockedGetDiff.mockRejectedValue(new Error('Diff failed'));
    render(<DiffPage />);

    await waitFor(() => {
      expect(screen.getByText('Error: Diff failed')).toBeInTheDocument();
    });
  });
});
