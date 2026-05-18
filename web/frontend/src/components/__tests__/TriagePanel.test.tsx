import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';

vi.mock('@/lib/api');

import { getTriage } from '@/lib/api';
import TriagePanel from '../TriagePanel';

const mockedGetTriage = vi.mocked(getTriage);

describe('TriagePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders summary, classification, and likely_cause when triage loads', async () => {
    mockedGetTriage.mockResolvedValue({
      summary: 'trace B added a verification step',
      classification: 'additive',
      likely_cause: 'agent learned to verify before responding',
    });

    render(<TriagePanel idA="a1" idB="b1" />);

    await waitFor(() => {
      expect(screen.getByText('trace B added a verification step')).toBeInTheDocument();
    });
    expect(screen.getByText(/additive/i)).toBeInTheDocument();
    expect(screen.getByText(/agent learned to verify before responding/i)).toBeInTheDocument();
  });

  it('shows loading state before triage resolves', () => {
    mockedGetTriage.mockReturnValue(new Promise(() => {}));
    render(<TriagePanel idA="a1" idB="b1" />);
    expect(screen.getByText(/analyzing/i)).toBeInTheDocument();
  });

  it('shows error message when triage fails', async () => {
    mockedGetTriage.mockRejectedValue(new Error('network down'));
    render(<TriagePanel idA="a1" idB="b1" />);
    await waitFor(() => {
      expect(screen.getByText(/network down/i)).toBeInTheDocument();
    });
  });
});
