import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  mockPathGraph,
  mockEmptyPathGraph,
  mockOverlay,
  mockPathGraphNoCost,
} from '@/test/mocks/fixtures';
import PathGraph from '../PathGraph';

describe('PathGraph', () => {
  it('renders one node per aggregate-graph node, labeled by tool name', () => {
    render(<PathGraph graph={mockPathGraph} />);
    expect(screen.getByText('read_file')).toBeInTheDocument();
    expect(screen.getByText('write_file')).toBeInTheDocument();
    expect(screen.getByText('bash')).toBeInTheDocument();
  });

  it('renders an empty state when the aggregate graph has no nodes', () => {
    render(<PathGraph graph={mockEmptyPathGraph} />);
    expect(screen.getByText(/no graph data/i)).toBeInTheDocument();
  });

  it('shows the branch-confidence percentage for each outgoing edge at a branching node', () => {
    render(<PathGraph graph={mockPathGraph} />);
    expect(screen.getByText(/80%/)).toBeInTheDocument();
    expect(screen.getByText(/20%/)).toBeInTheDocument();
  });

  it('marks overlay-matched nodes with data-overlay-state="matched"', () => {
    const { container } = render(
      <PathGraph graph={mockPathGraph} overlay={mockOverlay} />,
    );
    const matched = container.querySelectorAll('[data-overlay-state="matched"]');
    expect(matched.length).toBeGreaterThan(0);
  });

  it('marks the divergence-point source node with data-overlay-state="divergent"', () => {
    const { container } = render(
      <PathGraph graph={mockPathGraph} overlay={mockOverlay} />,
    );
    const divergent = container.querySelector('[data-overlay-state="divergent"]');
    expect(divergent).not.toBeNull();
  });

  it('renders heatmap buckets on each node when mode="heatmap-cost"', () => {
    const { container } = render(
      <PathGraph graph={mockPathGraph} mode="heatmap-cost" />,
    );
    // bash has highest cost (9000), should bucket as "hot".
    const bashNode = container.querySelector(
      '[data-node-id="bash"][data-heatmap-bucket="hot"]',
    );
    expect(bashNode).not.toBeNull();
    // write_file has lowest cost (800), should bucket as "cold".
    const writeNode = container.querySelector(
      '[data-node-id="write_file"][data-heatmap-bucket="cold"]',
    );
    expect(writeNode).not.toBeNull();
  });

  it('renders heatmap buckets on each node when mode="heatmap-latency"', () => {
    const { container } = render(
      <PathGraph graph={mockPathGraph} mode="heatmap-latency" />,
    );
    // bash has highest latency (11000) = hot.
    const bashNode = container.querySelector(
      '[data-node-id="bash"][data-heatmap-bucket="hot"]',
    );
    expect(bashNode).not.toBeNull();
  });

  it('shows the actual cost number on each node in heatmap-cost mode', () => {
    // Per the acceptance criterion "hover shows actual cost in tokens" —
    // jsdom can't simulate hover, so cost text renders inline in heatmap mode.
    // Values are formatted via toLocaleString (e.g. "9,000 tokens"); regex
    // tolerates the thousands separator.
    render(<PathGraph graph={mockPathGraph} mode="heatmap-cost" />);
    expect(screen.getByText(/9,?000\s*tokens/i)).toBeInTheDocument();
    expect(screen.getByText(/2,?500\s*tokens/i)).toBeInTheDocument();
    expect(screen.getByText(/800\s*tokens/i)).toBeInTheDocument();
  });

  it('falls back to data-heatmap-bucket="none" when a node has no cost data', () => {
    const { container } = render(
      <PathGraph graph={mockPathGraphNoCost} mode="heatmap-cost" />,
    );
    const noneNodes = container.querySelectorAll('[data-heatmap-bucket="none"]');
    expect(noneNodes.length).toBe(2);
  });

  it('does not emit data-heatmap-bucket in default overlay mode', () => {
    const { container } = render(<PathGraph graph={mockPathGraph} />);
    const bucketed = container.querySelector('[data-heatmap-bucket]');
    expect(bucketed).toBeNull();
  });
});
