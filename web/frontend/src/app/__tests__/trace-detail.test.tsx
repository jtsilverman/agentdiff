import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams, mockUseRouter } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { mockTraceDetail } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { getTrace } from '@/lib/api';
import TraceDetailPage from '../traces/[id]/page';

const mockedGetTrace = vi.mocked(getTrace);

describe('TraceDetailPage', () => {
  const routerMock = { push: vi.fn(), replace: vi.fn(), back: vi.fn(), prefetch: vi.fn() };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'trace-1' });
    mockUseRouter.mockReturnValue(routerMock);
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
    // StepList renders steps with role labels and content
    expect(screen.getByText('Hello')).toBeInTheDocument();
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
});
