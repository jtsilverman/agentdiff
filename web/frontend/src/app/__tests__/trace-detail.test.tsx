import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams, mockUseRouter } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import {
  mockTraceDetail,
  mockTranscript,
  mockEmptySimilarTraces,
} from '@/test/mocks/fixtures';
import type { TraceSummary } from '@/lib/types';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import {
  getTrace,
  getTranscript,
  getSimilar,
  listTraces,
} from '@/lib/api';
import TraceDetailPage from '../traces/[id]/page';

const mockedGetTrace = vi.mocked(getTrace);
const mockedGetTranscript = vi.mocked(getTranscript);
const mockedGetSimilar = vi.mocked(getSimilar);
const mockedListTraces = vi.mocked(listTraces);

function makeTrace(
  id: string,
  baseline_id: string,
  created_at: string,
): TraceSummary {
  return {
    id,
    name: `name-${id}`,
    adapter: 'claudecode',
    step_count: 3,
    created_at,
    baseline_id,
    baseline_name: `baseline-${baseline_id}`,
  };
}

describe('TraceDetailPage', () => {
  const routerMock = {
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'trace-1' });
    mockUseRouter.mockReturnValue(routerMock);
    mockedGetTranscript.mockReturnValue(new Promise(() => {}));
    mockedGetSimilar.mockResolvedValue(mockEmptySimilarTraces);
    mockedListTraces.mockResolvedValue([]);
  });

  it('shows loading state, then the trace name and adapter in the hero', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);
    expect(screen.getByText('Loading trace...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    // Adapter is in the metadata grid as a mono value, not a Tremor badge.
    expect(screen.getByText('claudecode')).toBeInTheDocument();
  });

  it('renders step content from the trace in the transcript', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(screen.getAllByText('Hello').length).toBeGreaterThanOrEqual(1);
  });

  it('shows a "Compare with another trace" action that is disabled when the corpus has fewer than 2 traces', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    mockedListTraces.mockResolvedValue([]);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });

    const compareBtn = screen.getByRole('button', {
      name: /Compare with another trace/i,
    });
    expect(compareBtn).toBeDisabled();
  });

  it('clicking Compare routes to /diff with an auto-picked second trace and ?auto=1', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    mockedListTraces.mockResolvedValue([
      makeTrace('trace-1', 'b1', '2026-05-27T12:00:00Z'),
      makeTrace('other-trace', 'b2', '2026-05-26T12:00:00Z'),
    ]);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', {
          name: /Compare with another trace/i,
        }),
      ).not.toBeDisabled();
    });

    fireEvent.click(
      screen.getByRole('button', { name: /Compare with another trace/i }),
    );
    expect(routerMock.push).toHaveBeenCalledWith(
      '/diff/trace-1/other-trace?auto=1',
    );
  });

  it('renders a "View baseline" link in the hero when the trace has a baseline', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    mockedListTraces.mockResolvedValue([
      makeTrace('trace-1', 'b1', '2026-05-27T12:00:00Z'),
    ]);
    render(<TraceDetailPage />);

    await waitFor(() => {
      const link = screen.getByRole('link', { name: /View baseline/i });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute('href', '/baselines/b1');
    });
  });

  it('renders the Promote-to-baseline button in the actions section', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(
      screen.getByRole('button', { name: /Promote to baseline/i }),
    ).toBeInTheDocument();
  });

  it('renders the What if? counterfactual button', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(
      screen.getByRole('button', { name: /What if\?/i }),
    ).toBeInTheDocument();
  });

  it('renders the Edit prompt button', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(
      screen.getByRole('button', { name: /Edit prompt/i }),
    ).toBeInTheDocument();
  });

  it('mounts the replay Scrubber with a step scrubber slider and the "Step 1 of N" label', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(
      screen.getByRole('slider', { name: /step scrubber/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Step 1 of 3/i)).toBeInTheDocument();
  });

  it('mounts the SimilarTraces panel with the "Similar traces" title', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Compare with another trace/i })).toBeInTheDocument();
    });
    expect(await screen.findByText(/Similar traces/i)).toBeInTheDocument();
    expect(mockedGetSimilar).toHaveBeenCalledWith('trace-1');
  });

  it('mounts the Transcript summary in the actions section', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    mockedGetTranscript.mockResolvedValue(mockTranscript);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText(mockTranscript.summary)).toBeInTheDocument();
    });
    expect(mockedGetTranscript).toHaveBeenCalledWith('trace-1');
  });
});
