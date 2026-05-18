import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';

vi.mock('@/lib/api');

import { getTranscript } from '@/lib/api';
import Transcript from '../Transcript';

const mockedGetTranscript = vi.mocked(getTranscript);

describe('Transcript', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the summary paragraph and key decision list when transcript loads', async () => {
    mockedGetTranscript.mockResolvedValue({
      summary: 'Agent listed the directory, read the largest file, and reported its line count.',
      key_decisions: ['chose ls over find', 'read README.md first', 'reported lines not bytes'],
    });

    render(<Transcript traceId="trace-1" />);

    await waitFor(() => {
      expect(
        screen.getByText(/listed the directory, read the largest file/i),
      ).toBeInTheDocument();
    });
    expect(screen.getByText(/chose ls over find/i)).toBeInTheDocument();
    expect(screen.getByText(/read README.md first/i)).toBeInTheDocument();
    expect(screen.getByText(/reported lines not bytes/i)).toBeInTheDocument();
  });

  it('shows loading state before transcript resolves', () => {
    mockedGetTranscript.mockReturnValue(new Promise(() => {}));
    render(<Transcript traceId="trace-1" />);
    expect(screen.getByText(/summariz/i)).toBeInTheDocument();
  });

  it('shows error message when transcript fails', async () => {
    mockedGetTranscript.mockRejectedValue(new Error('network down'));
    render(<Transcript traceId="trace-1" />);
    await waitFor(() => {
      expect(screen.getByText(/network down/i)).toBeInTheDocument();
    });
  });

  it('hides the key-decisions section when the list is empty (fallback case)', async () => {
    mockedGetTranscript.mockResolvedValue({
      summary: 'Transcript unavailable: anthropic 503: rate limited',
      key_decisions: [],
    });
    render(<Transcript traceId="trace-1" />);

    await waitFor(() => {
      expect(screen.getByText(/transcript unavailable/i)).toBeInTheDocument();
    });
    expect(screen.queryByText(/key decisions/i)).not.toBeInTheDocument();
  });
});
