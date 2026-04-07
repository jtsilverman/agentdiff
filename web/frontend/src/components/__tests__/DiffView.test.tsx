import { vi, describe, it, expect } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen } from '@testing-library/react';
import { mockDiff } from '@/test/mocks/fixtures';
import type { DiffResponse } from '@/lib/types';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

import DiffView from '../DiffView';

describe('DiffView', () => {
  it('renders summary badges with correct counts', () => {
    render(<DiffView diff={mockDiff} />);
    expect(screen.getByText('1 matches')).toBeInTheDocument();
    expect(screen.getByText('0 substitutions')).toBeInTheDocument();
    expect(screen.getByText('1 deletions')).toBeInTheDocument();
    expect(screen.getByText('1 insertions')).toBeInTheDocument();
    expect(screen.getByText('distance: 2')).toBeInTheDocument();
  });

  it('renders trace names as column headers', () => {
    render(<DiffView diff={mockDiff} />);
    expect(screen.getByText('trace-a')).toBeInTheDocument();
    expect(screen.getByText('trace-b')).toBeInTheDocument();
  });

  it('renders match rows with both steps visible', () => {
    render(<DiffView diff={mockDiff} />);
    // The match row has 'Hello' in both a_step and b_step
    const hellos = screen.getAllByText('Hello');
    expect(hellos.length).toBe(2);
  });

  it('renders delete rows with a_step present and b_step null showing "-" placeholder', () => {
    render(<DiffView diff={mockDiff} />);
    expect(screen.getByText('A reply')).toBeInTheDocument();
    // Null b_step renders "-" (there are 2 total: delete row b_step + insert row a_step)
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it('renders insert rows with a_step null and b_step present', () => {
    render(<DiffView diff={mockDiff} />);
    expect(screen.getByText('Different')).toBeInTheDocument();
    // There should be two "-" placeholders (delete row b_step + insert row a_step)
    const dashes = screen.getAllByText('-');
    expect(dashes.length).toBe(2);
  });

  it('StepCell shows tool_call name with () suffix', () => {
    const diff: DiffResponse = {
      ...mockDiff,
      alignment: [
        {
          a_step: {
            role: 'assistant',
            content: '',
            tool_call: { name: 'read_file', args: { path: '/tmp' } },
          },
          b_step: null,
          op: 'delete',
        },
      ],
    };
    render(<DiffView diff={diff} />);
    expect(screen.getByText('read_file()')).toBeInTheDocument();
  });

  it('StepCell shows tool_result name and truncated output', () => {
    const diff: DiffResponse = {
      ...mockDiff,
      alignment: [
        {
          a_step: {
            role: 'tool_result',
            content: '',
            tool_result: { name: 'read_file', output: 'file contents here', is_error: false },
          },
          b_step: null,
          op: 'delete',
        },
      ],
    };
    render(<DiffView diff={diff} />);
    expect(screen.getByText(/read_file: file contents here/)).toBeInTheDocument();
  });
});
