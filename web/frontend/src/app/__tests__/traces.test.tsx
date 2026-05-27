import { vi, describe, it, expect, beforeEach } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import type { TraceSummary } from '@/lib/types';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { listTraces } from '@/lib/api';
import TracesPage from '../traces/page';

const mockedListTraces = vi.mocked(listTraces);

const corpus: TraceSummary[] = [
  {
    id: 't-rate-1',
    name: 'Run 1',
    adapter: 'claude-code',
    step_count: 9,
    baseline_id: 'b-rate',
    baseline_name: 'rate-limit',
    metadata: {
      task: 'Add rate limiting to a Flask endpoint',
      outcome: 'succeeded',
      key_decision: 'Chose flask-limiter',
    },
    created_at: '2026-05-27T15:30:00Z',
  },
  {
    id: 't-auth-1',
    name: 'v1.3 · Run 2',
    adapter: 'cursor',
    step_count: 22,
    baseline_id: 'b-auth',
    baseline_name: 'auth-migration',
    metadata: {
      task: 'Migrate auth from JWT to session cookies',
      outcome: 'regressed',
      key_decision: 'Refresh loop deadlocked when session was read mid-rotate',
    },
    created_at: '2026-05-27T13:00:00Z',
  },
  {
    id: 't-auth-2',
    name: 'v1.3 · Run 1',
    adapter: 'cursor',
    step_count: 19,
    baseline_id: 'b-auth',
    baseline_name: 'auth-migration',
    metadata: {
      task: 'Migrate auth from JWT to session cookies',
      outcome: 'variance',
      key_decision: 'Added a refresh loop — slower but passes',
    },
    created_at: '2026-05-27T12:00:00Z',
  },
];

describe('TracesPage (redesigned corpus browser)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the hero with title "Traces" and total trace count', async () => {
    mockedListTraces.mockResolvedValue(corpus);
    render(<TracesPage />);

    expect(
      await screen.findByRole('heading', { name: /^traces$/i, level: 1 }),
    ).toBeInTheDocument();
    // Total appears in the hero stats grid as a .bl-stat-val sibling
    // of the "traces" label
    await waitFor(() => {
      const tracesLabel = screen.getAllByText('traces').find((el) =>
        el.classList.contains('ad-dim'),
      );
      expect(tracesLabel).toBeDefined();
      const statVal = tracesLabel!.parentElement!.querySelector('.bl-stat-val');
      expect(statVal?.textContent).toBe(String(corpus.length));
    });
  });

  it('renders one row per trace with name, baseline pill, outcome badge, and step count', async () => {
    mockedListTraces.mockResolvedValue(corpus);
    render(<TracesPage />);

    await waitFor(() => {
      expect(screen.getByText('Run 1')).toBeInTheDocument();
    });
    // Both auth runs render
    expect(screen.getByText('v1.3 · Run 1')).toBeInTheDocument();
    expect(screen.getByText('v1.3 · Run 2')).toBeInTheDocument();
    // Baseline pills render (auth-migration appears for two traces +
    // once as a <select> option; rate-limit for one trace + option)
    expect(screen.getAllByText('auth-migration').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('rate-limit').length).toBeGreaterThanOrEqual(1);
    // Outcome badge labels appear (also as filter chip labels, hence ≥ 2)
    expect(screen.getAllByText('regressed').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('variance').length).toBeGreaterThanOrEqual(2);
    // Step counts appear inside the table cells (9, 19, 22)
    const table = document.querySelector('.tr-table')!;
    expect(within(table as HTMLElement).getByText('9')).toBeInTheDocument();
    expect(within(table as HTMLElement).getByText('19')).toBeInTheDocument();
    expect(within(table as HTMLElement).getByText('22')).toBeInTheDocument();
  });

  it('clicking the "regressed" outcome chip filters rows to regressed only', async () => {
    mockedListTraces.mockResolvedValue(corpus);
    render(<TracesPage />);

    await waitFor(() => {
      expect(screen.getByText('Run 1')).toBeInTheDocument();
    });

    // The filter chips are <button> with class seg-btn; outcome badges
    // are <span>. getByRole('button', name: /regressed/) resolves to
    // the chip uniquely.
    const chip = screen.getByRole('button', { name: /^regressed$/i });
    fireEvent.click(chip);

    await waitFor(() => {
      // Run 1 (succeeded) hidden
      expect(screen.queryByText('Run 1')).not.toBeInTheDocument();
      // v1.3 · Run 1 (variance) hidden
      expect(screen.queryByText('v1.3 · Run 1')).not.toBeInTheDocument();
      // v1.3 · Run 2 (regressed) still visible
      expect(screen.getByText('v1.3 · Run 2')).toBeInTheDocument();
    });
  });

  it('shows the empty state when filters match no rows', async () => {
    mockedListTraces.mockResolvedValue(corpus);
    render(<TracesPage />);

    await waitFor(() => {
      expect(screen.getByText('Run 1')).toBeInTheDocument();
    });

    // Type a query that matches nothing
    const search = screen.getByPlaceholderText(/filter by task/i);
    fireEvent.change(search, { target: { value: 'no-such-task-zzz' } });

    await waitFor(() => {
      expect(screen.getByText(/no traces match/i)).toBeInTheDocument();
    });
  });
});
