import { vi, describe, it, expect, beforeEach } from 'vitest';
import { mockUseParams } from '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import {
  mockStrategyReport,
  mockMatchResult,
  mockPathGraph,
  mockOverlay,
  mockCounterfactual,
  mockSimilarTraces,
} from '@/test/mocks/fixtures';
import type { BaselineSummary } from '@/lib/types';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import {
  getCluster,
  compareTrace,
  getGraph,
  getOverlay,
  listBaselines,
  runCounterfactual,
  editPrompt,
  getSimilar,
} from '@/lib/api';
import BaselineDetailPage from '../baselines/[id]/page';

const mockedGetCluster = vi.mocked(getCluster);
const mockedCompareTrace = vi.mocked(compareTrace);
const mockedGetGraph = vi.mocked(getGraph);
const mockedGetOverlay = vi.mocked(getOverlay);
const mockedListBaselines = vi.mocked(listBaselines);
const mockedRunCounterfactual = vi.mocked(runCounterfactual);
const mockedEditPrompt = vi.mocked(editPrompt);
const mockedGetSimilar = vi.mocked(getSimilar);

const baselineWithDescription: BaselineSummary = {
  id: 'bl-1',
  name: 'seed-api-endpoint-rename',
  description: 'Rename the /users endpoint to /customers across the codebase.',
  trace_count: 5,
  created_at: '2026-01-01T00:00:00Z',
};

describe('BaselineDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseParams.mockReturnValue({ id: 'bl-1' });
    mockedGetGraph.mockResolvedValue(mockPathGraph);
    mockedGetOverlay.mockResolvedValue(mockOverlay);
    mockedListBaselines.mockResolvedValue([baselineWithDescription]);
  });

  it('shows loading, then renders StrategyCluster with report data', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);
    expect(screen.getByText('Loading baseline...')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });
    // StrategyCluster renders strategy details
    expect(screen.getByText('Strategy 0')).toBeInTheDocument();
  });

  it('entering trace ID and clicking Compare calls compareTrace', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockResolvedValue(mockMatchResult);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(mockedCompareTrace).toHaveBeenCalledWith('bl-1', 'trace-99');
    });
  });

  it('shows DriftBadge when compare succeeds', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockResolvedValue(mockMatchResult);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(screen.getByText('Matches Strategy 0')).toBeInTheDocument();
    });
  });

  it('shows error message when compare fails', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedCompareTrace.mockRejectedValue(new Error('Compare exploded'));
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const input = screen.getByPlaceholderText('Enter trace ID');
    fireEvent.change(input, { target: { value: 'trace-99' } });

    const compareBtn = screen.getByRole('button', { name: /Compare/i });
    fireEvent.click(compareBtn);

    await waitFor(() => {
      expect(screen.getByText('Error: Compare exploded')).toBeInTheDocument();
    });
  });

  it('renders the PathGraph hero with nodes from getGraph', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);

    // 'bash' is unique to the PathGraph fixture (not in the cluster fixture's
    // tool_sequence), so it gates on PathGraph having actually rendered.
    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });
  });

  it('renders a heatmap-mode toggle and applies heatmap buckets when Cost is selected', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });

    const costBtn = screen.getByRole('button', { name: /heatmap.*cost/i });
    fireEvent.click(costBtn);

    await waitFor(() => {
      expect(
        container.querySelector('[data-heatmap-bucket="hot"]'),
      ).not.toBeNull();
    });
  });

  it('renders the baseline description as a callout when listBaselines returns it', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(
        screen.getByText(/Rename the \/users endpoint to \/customers/i),
      ).toBeInTheDocument();
    });
  });

  it('renders the task statement from the first trace metadata as the h1 title', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      const heading = screen.getByRole('heading', {
        level: 1,
        name: /Rename the \/users endpoint/i,
      });
      expect(heading).toBeInTheDocument();
    });
  });

  it('renders side stats with the run count derived from cluster members + noise', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      // 2 members + 1 noise trace = 3 runs in the fixture.
      expect(container.querySelector('.bl-stat')).not.toBeNull();
    });
    const stats = container.querySelectorAll('.bl-stat-val');
    const statValues = Array.from(stats).map((el) => el.textContent);
    expect(statValues).toContain('3');
  });

  it('renders trace rows with outcome metadata badges', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(container.querySelector('.tl-table')).not.toBeNull();
    });
    const rows = container.querySelectorAll('.tl-row-button');
    expect(rows.length).toBe(3); // 2 strategy members + 1 noise trace
    // Each outcome appears in both a filter chip and a row badge.
    expect(screen.getAllByText(/^variance$/i).length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText(/^regressed$/i).length).toBeGreaterThanOrEqual(2);
  });

  it('renders the path-graph card legend with the three rule rows', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText(/edge thickness = run count/i)).toBeInTheDocument();
    });
    expect(screen.getByText(/node size = call frequency/i)).toBeInTheDocument();
    expect(screen.getByText(/color = outcome cluster/i)).toBeInTheDocument();
  });

  it('renders cluster dots derived from strategies + noise in the legend', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(container.querySelector('.pg-legend-clusters')).not.toBeNull();
    });
    const dots = container.querySelectorAll('.pg-cluster-dot');
    // 1 strategy + 1 noise bucket = 2 clusters.
    expect(dots.length).toBe(2);
  });

  it('clicking the counterfactual money card opens a modal that calls runCounterfactual', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedRunCounterfactual.mockResolvedValue(mockCounterfactual);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const cfCard = screen.getByRole('button', { name: /run counterfactual/i });
    fireEvent.click(cfCard);

    // Modal opens with a Run button.
    const runBtn = await screen.findByRole('button', { name: /^run counterfactual$/i });
    // The textarea for modified input.
    const input = screen.getByPlaceholderText(/modified input/i);
    fireEvent.change(input, { target: { value: 'try a different tool' } });
    fireEvent.click(runBtn);

    await waitFor(() => {
      // First strategy member trace id = 't1'; default step 0.
      expect(mockedRunCounterfactual).toHaveBeenCalledWith('t1', 0, 'try a different tool');
    });
  });

  it('clicking the edit-prompt money card opens a modal that calls editPrompt', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedEditPrompt.mockResolvedValue(mockCounterfactual);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const epCard = screen.getByRole('button', { name: /edit the prompt/i });
    fireEvent.click(epCard);

    const rerunBtn = await screen.findByRole('button', { name: /rerun/i });
    const textarea = screen.getByPlaceholderText(/rewrite the prompt/i);
    fireEvent.change(textarea, { target: { value: 'new prompt text' } });
    fireEvent.click(rerunBtn);

    await waitFor(() => {
      expect(mockedEditPrompt).toHaveBeenCalledWith('t1', 0, 'new prompt text');
    });
  });

  it('clicking the find-similar money card opens a modal that calls getSimilar', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    mockedGetSimilar.mockResolvedValue(mockSimilarTraces);
    render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('test-baseline')).toBeInTheDocument();
    });

    const simCard = screen.getByRole('button', { name: /find similar traces/i });
    fireEvent.click(simCard);

    const findBtn = await screen.findByRole('button', { name: /^find similar$/i });
    fireEvent.click(findBtn);

    await waitFor(() => {
      expect(mockedGetSimilar).toHaveBeenCalledWith('t1');
    });
    // Results render.
    await waitFor(() => {
      expect(screen.getByText('grep-then-cat')).toBeInTheDocument();
    });
  });

  it('clicking a trace from the list calls getOverlay and applies overlay state', async () => {
    mockedGetCluster.mockResolvedValue(mockStrategyReport);
    const { container } = render(<BaselineDetailPage />);

    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });

    // mockStrategyReport.strategies[0].members[0] === { id: 't1', name: 'trace-1' }
    // Button shows the name; click should pass the UUID to getOverlay so the
    // URL stays single-segment (trace names can contain '/').
    const traceBtn = await screen.findByRole('button', { name: /trace-1/ });
    fireEvent.click(traceBtn);

    await waitFor(() => {
      expect(mockedGetOverlay).toHaveBeenCalledWith('bl-1', 't1');
    });
    await waitFor(() => {
      expect(
        container.querySelector('[data-overlay-state="divergent"]'),
      ).not.toBeNull();
    });
  });
});
