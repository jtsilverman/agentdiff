import { vi, describe, it, expect, beforeEach } from 'vitest';
import {
  mockUseParams,
  mockUseSearchParams,
} from '@/test/mocks/next-navigation';
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react';
import { mockDiff, mockTraceDetail } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getDiff, getTrace, getTriage, listTraces } from '@/lib/api';
import DiffPage from '../diff/[idA]/[idB]/page';

const mockedGetDiff = vi.mocked(getDiff);
const mockedGetTrace = vi.mocked(getTrace);
const mockedGetTriage = vi.mocked(getTriage);
const mockedListTraces = vi.mocked(listTraces);

describe('DiffPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ idA: 'a1', idB: 'b1' });
    mockedGetTrace.mockImplementation((id: string) =>
      Promise.resolve({
        ...mockTraceDetail,
        id,
        name: id === 'a1' ? 'trace-a' : 'trace-b',
        metadata: { task: 'task for ' + id, outcome: 'variance' },
      }),
    );
    mockedGetTriage.mockReturnValue(new Promise(() => {}));
    mockedListTraces.mockResolvedValue([]);
    mockedGetDiff.mockResolvedValue(mockDiff);
  });

  it('renders the new hero with H1 "Diff" and summary stat values', async () => {
    render(<DiffPage />);
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: /^Diff$/i })).toBeInTheDocument();
    });
    // Hero stat row shows match/sub/insert/delete counts from mockDiff.summary.
    const hero = screen.getByRole('banner');
    expect(within(hero).getByText('matches')).toBeInTheDocument();
    expect(within(hero).getByText('subs')).toBeInTheDocument();
    expect(within(hero).getByText('insertions')).toBeInTheDocument();
    expect(within(hero).getByText('deletions')).toBeInTheDocument();
  });

  it('renders switcher strip with both trace names and a vs divider', async () => {
    render(<DiffPage />);
    await waitFor(() => {
      expect(screen.getByText('trace-a')).toBeInTheDocument();
    });
    expect(screen.getByText('trace-b')).toBeInTheDocument();
    expect(screen.getByText('vs')).toBeInTheDocument();
  });

  it('renders 3 alignment rows with op-marker labels (match / delete / insert)', async () => {
    render(<DiffPage />);
    await waitFor(() => {
      expect(screen.getByText('match')).toBeInTheDocument();
    });
    expect(screen.getByText('delete')).toBeInTheDocument();
    expect(screen.getByText('insert')).toBeInTheDocument();
  });

  it('op-filter "insertions" hides match + delete rows, leaves only insert visible', async () => {
    render(<DiffPage />);
    await waitFor(() => {
      expect(screen.getByText('match')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'insertions' }));
    await waitFor(() => {
      expect(screen.queryByText('match')).not.toBeInTheDocument();
    });
    expect(screen.queryByText('delete')).not.toBeInTheDocument();
    expect(screen.getByText('insert')).toBeInTheDocument();
  });

  it('shows the auto-selected info banner when ?auto=1 is present', async () => {
    mockUseSearchParams.mockReturnValue(
      new URLSearchParams('auto=1') as unknown as ReturnType<typeof vi.fn>,
    );
    render(<DiffPage />);
    await waitFor(() => {
      expect(
        screen.getByText(/Auto-selected · most recent two traces/i),
      ).toBeInTheDocument();
    });
  });

  it('does not show the auto-selected banner without ?auto=1', async () => {
    mockUseSearchParams.mockReturnValue(
      new URLSearchParams() as unknown as ReturnType<typeof vi.fn>,
    );
    render(<DiffPage />);
    await waitFor(() => {
      expect(screen.getByText('trace-a')).toBeInTheDocument();
    });
    expect(
      screen.queryByText(/Auto-selected · most recent two traces/i),
    ).not.toBeInTheDocument();
  });
});
