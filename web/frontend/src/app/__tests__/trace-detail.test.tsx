import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams, mockUseRouter } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { mockTraceDetail, mockTranscript } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getTrace, getTranscript } from '@/lib/api';
import TraceDetailPage from '../traces/[id]/page';

const mockedGetTrace = vi.mocked(getTrace);
const mockedGetTranscript = vi.mocked(getTranscript);

describe('TraceDetailPage', () => {
  const routerMock = { push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'trace-1' });
    mockUseRouter.mockReturnValue(routerMock);
    // Default: transcript hangs so it stays in "Summarizing" state and doesn't
    // overwhelm the existing assertions. Individual tests override as needed.
    mockedGetTranscript.mockReturnValue(new Promise(() => {}));
  });

  it('shows loading state, then trace name, adapter badge, step count', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);
    expect(screen.getByText('Loading trace...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });
    expect(screen.getByText('claudecode')).toBeInTheDocument();
    expect(screen.getByText('3 steps')).toBeInTheDocument();
  });

  it('shows StepList with trace steps', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });
    // StepList and Scrubber both render step content; assert at least one match exists.
    expect(screen.getAllByText('Hello').length).toBeGreaterThanOrEqual(1);
  });

  it('"Compare" button is disabled when diff input is empty', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    const compareBtn = screen.getByRole('button', { name: /Compare/ });
    expect(compareBtn).toBeDisabled();
  });

  it('entering a trace ID and clicking Compare calls router.push', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Second trace ID');
    fireEvent.change(input, { target: { value: 'other-trace' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/ });
    fireEvent.click(compareBtn);

    expect(routerMock.push).toHaveBeenCalledWith('/diff/trace-1/other-trace');
  });

  it('renders the Promote-to-baseline button', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    expect(
      screen.getByRole('button', { name: /Promote to baseline/i }),
    ).toBeInTheDocument();
  });

  it('renders the What if? counterfactual button', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    expect(
      screen.getByRole('button', { name: /What if\?/i }),
    ).toBeInTheDocument();
  });

  it('renders the Edit prompt button (inline prompt editor)', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    // EditPromptButton's collapsed default. "Edit prompt" is SUT-unique on this
    // page; no other sibling component renders it.
    expect(
      screen.getByRole('button', { name: /Edit prompt/i }),
    ).toBeInTheDocument();
  });

  it('mounts the replay Scrubber with a step scrubber slider and the "Step 1 of N" label', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    // Scrubber-unique gates: aria-label="step scrubber" and "Step N of M" text
    // are not emitted by StepList, Transcript, PromoteButton, or CounterfactualButton.
    expect(screen.getByRole('slider', { name: /step scrubber/i })).toBeInTheDocument();
    expect(screen.getByText(/Step 1 of 3/i)).toBeInTheDocument();
  });

  it('mounts Transcript above the step list with the loaded summary', async () => {
    mockedGetTrace.mockResolvedValue(mockTraceDetail);
    mockedGetTranscript.mockResolvedValue(mockTranscript);
    const { container } = render(<TraceDetailPage />);

    await waitFor(() => {
      expect(screen.getByText(mockTranscript.summary)).toBeInTheDocument();
    });

    const summaryEl = screen.getByText(mockTranscript.summary);
    // Both Scrubber and StepList render "Hello"; the LAST occurrence is StepList's
    // (mounted after Scrubber in JSX). Assert Transcript is above the step list.
    const helloEls = screen.getAllByText('Hello');
    const lastStepEl = helloEls[helloEls.length - 1];
    expect(summaryEl.compareDocumentPosition(lastStepEl)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    );
    expect(mockedGetTranscript).toHaveBeenCalledWith('trace-1');
    // The container is used implicitly via screen queries; suppress unused warning.
    void container;
  });
});
