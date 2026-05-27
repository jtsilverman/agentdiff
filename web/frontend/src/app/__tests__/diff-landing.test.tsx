import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseRouter } from '@/test/mocks/next-navigation';
import { render, screen, waitFor } from '@testing-library/react';
import type { TraceSummary } from '@/lib/types';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

vi.mock('@/lib/api');

import { listTraces } from '@/lib/api';
import DiffLandingPage from '../diff/page';

const mockedListTraces = vi.mocked(listTraces);

function makeTrace(
  id: string,
  baseline_id: string,
  created_at: string,
): TraceSummary {
  return {
    id,
    name: `name-${id}`,
    adapter: 'test',
    step_count: 1,
    created_at,
    baseline_id,
    baseline_name: `baseline-${baseline_id}`,
  };
}

describe('DiffLandingPage', () => {
  const replaceSpy = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRouter.mockReturnValue({
      push: vi.fn(),
      replace: replaceSpy,
      back: vi.fn(),
      prefetch: vi.fn(),
    });
  });

  it('auto-picks two most-recent cross-baseline traces and redirects with ?auto=1', async () => {
    mockedListTraces.mockResolvedValue([
      makeTrace('newest', 'b1', '2026-05-27T12:00:00Z'),
      makeTrace('mid', 'b2', '2026-05-26T12:00:00Z'),
      makeTrace('old', 'b1', '2026-05-25T12:00:00Z'),
    ]);
    render(<DiffLandingPage />);
    await waitFor(() => {
      expect(replaceSpy).toHaveBeenCalledWith('/diff/newest/mid?auto=1');
    });
  });

  it('falls back to second-most-recent when no cross-baseline trace exists', async () => {
    mockedListTraces.mockResolvedValue([
      makeTrace('a', 'b1', '2026-05-27T12:00:00Z'),
      makeTrace('b', 'b1', '2026-05-26T12:00:00Z'),
    ]);
    render(<DiffLandingPage />);
    await waitFor(() => {
      expect(replaceSpy).toHaveBeenCalledWith('/diff/a/b?auto=1');
    });
  });

  it('renders an empty state when fewer than two traces exist', async () => {
    mockedListTraces.mockResolvedValue([
      makeTrace('only', 'b1', '2026-05-27T12:00:00Z'),
    ]);
    render(<DiffLandingPage />);
    await waitFor(() => {
      expect(
        screen.getByText(/Need at least two traces to diff\./i),
      ).toBeInTheDocument();
    });
    expect(replaceSpy).not.toHaveBeenCalled();
  });
});
