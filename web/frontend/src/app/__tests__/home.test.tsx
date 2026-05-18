import { vi, describe, it, expect, beforeEach } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen, waitFor } from '@testing-library/react';
import { mockBaseline } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { listBaselines } from '@/lib/api';
import HomePage from '../page';

const mockedListBaselines = vi.mocked(listBaselines);

describe('HomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows "Loading baselines..." initially', () => {
    mockedListBaselines.mockReturnValue(new Promise(() => {}));
    render(<HomePage />);
    expect(screen.getByText('Loading baselines...')).toBeInTheDocument();
  });

  it('shows error message when listBaselines rejects', async () => {
    mockedListBaselines.mockRejectedValue(new Error('Network down'));
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText('Error: Network down')).toBeInTheDocument();
    });
  });

  it('shows "No Baselines Yet" when listBaselines returns empty array', async () => {
    mockedListBaselines.mockResolvedValue([]);
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText('No Baselines Yet')).toBeInTheDocument();
    });
  });

  it('renders baseline cards with name, trace count, and link', async () => {
    mockedListBaselines.mockResolvedValue([mockBaseline]);
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });
    expect(screen.getByText('5 traces')).toBeInTheDocument();
    const link = screen.getByRole('link');
    expect(link).toHaveAttribute('href', '/baselines/bl-1');
  });

  it('renders a "Try these examples" demo row when seeded baselines are present', async () => {
    mockedListBaselines.mockResolvedValue([
      { id: 'seed-1', name: 'seed-prompt-regression', trace_count: 5, created_at: '2026-01-01T00:00:00Z' },
      { id: 'seed-2', name: 'seed-tool-order-variance', trace_count: 5, created_at: '2026-01-01T00:00:00Z' },
      mockBaseline,
    ]);
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText('Try these examples')).toBeInTheDocument();
    });
    // Per the rtl-assertions-gate-on-sut-unique-text pattern: gate on labels
    // unique to the seed buttons (the human-readable scenario names) rather
    // than on raw baseline names that also appear in the non-seed cards section.
    expect(screen.getByRole('link', { name: /Prompt-change regression/i })).toHaveAttribute(
      'href',
      '/baselines/seed-1',
    );
    expect(screen.getByRole('link', { name: /Tool-order variance/i })).toHaveAttribute(
      'href',
      '/baselines/seed-2',
    );
  });

  it('does not render the demo row when no seed-prefixed baselines exist', async () => {
    mockedListBaselines.mockResolvedValue([mockBaseline]);
    render(<HomePage />);
    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });
    expect(screen.queryByText('Try these examples')).not.toBeInTheDocument();
  });
});
