import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseRouter } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

vi.mock('@/lib/api');

import { promoteTrace } from '@/lib/api';
import PromoteButton from '../PromoteButton';

const mockedPromoteTrace = vi.mocked(promoteTrace);

describe('PromoteButton', () => {
  const routerMock = {
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseRouter.mockReturnValue(routerMock);
  });

  it('renders the collapsed button by default', () => {
    render(<PromoteButton traceId="trace-abc" />);
    expect(
      screen.getByRole('button', { name: /Promote to baseline/i }),
    ).toBeInTheDocument();
    // No input shown until expanded.
    expect(screen.queryByPlaceholderText('Baseline name')).not.toBeInTheDocument();
  });

  it('expands to show a name input and create button on click', () => {
    render(<PromoteButton traceId="trace-abc12345" />);
    fireEvent.click(screen.getByRole('button', { name: /Promote to baseline/i }));

    const input = screen.getByPlaceholderText('Baseline name') as HTMLInputElement;
    expect(input).toBeInTheDocument();
    // Default name uses trace id prefix.
    expect(input.value).toBe('promoted-trace-ab');
    expect(screen.getByRole('button', { name: /Create baseline/i })).toBeInTheDocument();
  });

  it('uses defaultName prop when provided', () => {
    render(<PromoteButton traceId="trace-abc" defaultName="promoted-my-trace" />);
    fireEvent.click(screen.getByRole('button', { name: /Promote to baseline/i }));
    const input = screen.getByPlaceholderText('Baseline name') as HTMLInputElement;
    expect(input.value).toBe('promoted-my-trace');
  });

  it('calls promoteTrace and pushes to the new baseline on success', async () => {
    mockedPromoteTrace.mockResolvedValue({
      baseline_id: 'baseline-xyz',
      baseline_name: 'my-promoted',
    });

    render(<PromoteButton traceId="trace-abc" defaultName="my-promoted" />);
    fireEvent.click(screen.getByRole('button', { name: /Promote to baseline/i }));
    fireEvent.click(screen.getByRole('button', { name: /Create baseline/i }));

    await waitFor(() => {
      expect(mockedPromoteTrace).toHaveBeenCalledWith('trace-abc', 'my-promoted');
    });
    await waitFor(() => {
      expect(routerMock.push).toHaveBeenCalledWith('/baselines/baseline-xyz');
    });
  });

  it('shows an error and stays expanded when promoteTrace rejects', async () => {
    mockedPromoteTrace.mockRejectedValue(new Error('baseline name already exists'));

    render(<PromoteButton traceId="trace-abc" defaultName="dup" />);
    fireEvent.click(screen.getByRole('button', { name: /Promote to baseline/i }));
    fireEvent.click(screen.getByRole('button', { name: /Create baseline/i }));

    await waitFor(() => {
      expect(screen.getByText(/baseline name already exists/i)).toBeInTheDocument();
    });
    expect(routerMock.push).not.toHaveBeenCalled();
    // Input still visible so the user can fix and retry.
    expect(screen.getByPlaceholderText('Baseline name')).toBeInTheDocument();
  });

  it('Cancel collapses back to the initial button', () => {
    render(<PromoteButton traceId="trace-abc" />);
    fireEvent.click(screen.getByRole('button', { name: /Promote to baseline/i }));
    fireEvent.click(screen.getByRole('button', { name: /Cancel/i }));

    expect(screen.queryByPlaceholderText('Baseline name')).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Promote to baseline/i }),
    ).toBeInTheDocument();
  });
});
