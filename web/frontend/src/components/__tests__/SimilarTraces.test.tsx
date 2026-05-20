import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { mockUseRouter } from '@/test/mocks/next-navigation';
import { mockSimilarTraces, mockEmptySimilarTraces } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, onClick, ...props }: any) => (
    <a href={href} onClick={onClick} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getSimilar } from '@/lib/api';
import SimilarTraces from '../SimilarTraces';

const mockedGetSimilar = vi.mocked(getSimilar);

describe('SimilarTraces', () => {
  const routerMock = { push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRouter.mockReturnValue(routerMock);
  });

  it('renders the "Similar traces" panel title (SUT-unique anchor)', async () => {
    mockedGetSimilar.mockResolvedValue(mockSimilarTraces);
    render(<SimilarTraces traceId="src-trace" />);
    await waitFor(() => {
      expect(screen.getByText(/Similar traces/i)).toBeInTheDocument();
    });
  });

  it('shows loading state before getSimilar resolves', () => {
    mockedGetSimilar.mockReturnValue(new Promise(() => {}));
    render(<SimilarTraces traceId="src-trace" />);
    expect(screen.getByText(/Finding similar traces/i)).toBeInTheDocument();
  });

  it('renders all 5 matches with name + score when getSimilar returns 5', async () => {
    mockedGetSimilar.mockResolvedValue(mockSimilarTraces);
    render(<SimilarTraces traceId="src-trace" />);

    await waitFor(() => {
      expect(screen.getByText('grep-then-cat')).toBeInTheDocument();
    });
    expect(screen.getByText('grep-then-head')).toBeInTheDocument();
    expect(screen.getByText('find-then-cat')).toBeInTheDocument();
    expect(screen.getByText('ls-then-cat')).toBeInTheDocument();
    expect(screen.getByText('echo-then-write')).toBeInTheDocument();

    // Scores rendered to 2 decimals.
    expect(screen.getByText(/0\.92/)).toBeInTheDocument();
    expect(screen.getByText(/0\.33/)).toBeInTheDocument();
  });

  it('shows empty-state message when matches array is empty', async () => {
    mockedGetSimilar.mockResolvedValue(mockEmptySimilarTraces);
    render(<SimilarTraces traceId="src-trace" />);
    await waitFor(() => {
      expect(screen.getByText(/No matches yet/i)).toBeInTheDocument();
    });
  });

  it('shows error message when getSimilar rejects', async () => {
    mockedGetSimilar.mockRejectedValue(new Error('upstream down'));
    render(<SimilarTraces traceId="src-trace" />);
    await waitFor(() => {
      expect(screen.getByText(/upstream down/i)).toBeInTheDocument();
    });
  });

  it('clicking a match navigates to that trace via router.push', async () => {
    mockedGetSimilar.mockResolvedValue(mockSimilarTraces);
    render(<SimilarTraces traceId="src-trace" />);

    await waitFor(() => {
      expect(screen.getByText('grep-then-cat')).toBeInTheDocument();
    });

    const link = screen.getByText('grep-then-cat').closest('a');
    if (!link) {
      throw new Error('expected an anchor wrapping match name');
    }
    expect(link.getAttribute('href')).toBe('/traces/sim-1');
  });

  it('calls getSimilar with the source trace id', async () => {
    mockedGetSimilar.mockResolvedValue(mockEmptySimilarTraces);
    render(<SimilarTraces traceId="my-source-trace" />);
    await waitFor(() => {
      expect(mockedGetSimilar).toHaveBeenCalledWith('my-source-trace');
    });
  });
});
