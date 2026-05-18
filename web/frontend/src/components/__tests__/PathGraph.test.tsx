import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { mockPathGraph, mockEmptyPathGraph, mockOverlay } from '@/test/mocks/fixtures';
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
});
