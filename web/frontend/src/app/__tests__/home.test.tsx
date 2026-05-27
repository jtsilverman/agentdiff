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

  it('renders the hero title and pipeline stages', async () => {
    mockedListBaselines.mockResolvedValue([]);
    render(<HomePage />);
    expect(screen.getByText(/See how your AI/i)).toBeInTheDocument();
    expect(screen.getByText(/Context/)).toBeInTheDocument();
    expect(screen.getByText(/Reasoning/)).toBeInTheDocument();
    expect(screen.getByText(/Decision/)).toBeInTheDocument();
    expect(screen.getByText(/Output/)).toBeInTheDocument();
  });

  it('renders the three example cards with their task copy', () => {
    mockedListBaselines.mockResolvedValue([]);
    render(<HomePage />);
    expect(
      screen.getByText(/Add a "Buy Now" button to the pricing page/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Diagnose yesterday.s site outage/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Build a weekly report on customer signups/i),
    ).toBeInTheDocument();
  });

  it('links cards to matching baselines by name hint', async () => {
    mockedListBaselines.mockResolvedValue([
      { id: 'seed-variance', name: 'seed-tool-order-variance', trace_count: 5, created_at: '2026-01-01T00:00:00Z' },
      { id: 'seed-regression', name: 'seed-prompt-regression', trace_count: 5, created_at: '2026-01-01T00:00:00Z' },
      { id: 'seed-novel', name: 'seed-novel-tool-discovery', trace_count: 5, created_at: '2026-01-01T00:00:00Z' },
    ]);
    render(<HomePage />);
    await waitFor(() => {
      const varianceCard = screen.getByText(/Add a "Buy Now" button/i).closest('a');
      expect(varianceCard).toHaveAttribute('href', '/baselines/seed-variance');
    });
    const regressionCard = screen.getByText(/Diagnose yesterday.s site outage/i).closest('a');
    expect(regressionCard).toHaveAttribute('href', '/baselines/seed-regression');
    const novelCard = screen.getByText(/Build a weekly report/i).closest('a');
    expect(novelCard).toHaveAttribute('href', '/baselines/seed-novel');
  });

  it('falls back to positional baselines when name hints do not match', async () => {
    mockedListBaselines.mockResolvedValue([
      { id: 'a', name: 'alpha', trace_count: 1, created_at: '2026-01-01T00:00:00Z' },
      { id: 'b', name: 'beta', trace_count: 2, created_at: '2026-01-01T00:00:00Z' },
      { id: 'c', name: 'gamma', trace_count: 3, created_at: '2026-01-01T00:00:00Z' },
      mockBaseline,
    ]);
    render(<HomePage />);
    await waitFor(() => {
      const card = screen.getByText(/Add a "Buy Now" button/i).closest('a');
      expect(card).toHaveAttribute('href', '/baselines/a');
    });
  });
});
