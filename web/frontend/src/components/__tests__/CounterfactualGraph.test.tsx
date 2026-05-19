import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CounterfactualGraph from '../CounterfactualGraph';
import type { CounterfactualComparison } from '@/lib/types';

// Per the react-flow-text-in-custom-node lesson (chunk 11): assertable text must
// live inside the custom node template, not on edge labels — jsdom never paints
// the EdgeLabelRenderer portal. CounterfactualGraph renders both branches via
// custom nodes with data-overlay-state="shared|original|counterfactual|divergence".
// Fixture is chosen so each branch has at least one tool name unique to it,
// per the rtl-assertions-gate-on-sut-unique-text lesson.

const comparison: CounterfactualComparison = {
  original_path: ['start', 'read_file', 'write_file'],
  new_path: ['start', 'edit_file', 'bash'],
  divergence_step: 1,
};

function nodeFor(label: string): Element | null {
  return screen.getByText(label).closest('[data-overlay-state]');
}

describe('CounterfactualGraph', () => {
  it('renders the shared prefix node with shared state', () => {
    render(<CounterfactualGraph comparison={comparison} />);
    expect(nodeFor('start')).toHaveAttribute('data-overlay-state', 'shared');
  });

  it('marks the first node after fork on the original branch as divergence', () => {
    render(<CounterfactualGraph comparison={comparison} />);
    expect(nodeFor('read_file')).toHaveAttribute('data-overlay-state', 'divergence');
  });

  it('marks the first node after fork on the counterfactual branch as divergence', () => {
    render(<CounterfactualGraph comparison={comparison} />);
    expect(nodeFor('edit_file')).toHaveAttribute('data-overlay-state', 'divergence');
  });

  it('renders nodes further along the original branch with original state', () => {
    render(<CounterfactualGraph comparison={comparison} />);
    expect(nodeFor('write_file')).toHaveAttribute('data-overlay-state', 'original');
  });

  it('renders nodes further along the counterfactual branch with counterfactual state', () => {
    render(<CounterfactualGraph comparison={comparison} />);
    expect(nodeFor('bash')).toHaveAttribute('data-overlay-state', 'counterfactual');
  });

  it('handles divergence_step=0 (no shared prefix, immediate fork)', () => {
    const noShared: CounterfactualComparison = {
      original_path: ['read_file', 'write_file'],
      new_path: ['bash', 'edit_file'],
      divergence_step: 0,
    };
    render(<CounterfactualGraph comparison={noShared} />);
    expect(nodeFor('read_file')).toHaveAttribute('data-overlay-state', 'divergence');
    expect(nodeFor('bash')).toHaveAttribute('data-overlay-state', 'divergence');
    expect(nodeFor('write_file')).toHaveAttribute('data-overlay-state', 'original');
    expect(nodeFor('edit_file')).toHaveAttribute('data-overlay-state', 'counterfactual');
  });
});
