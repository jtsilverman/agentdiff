import { vi, describe, it, expect, beforeEach } from 'vitest';
import '@/test/mocks/next-navigation';
import { mockUsePathname } from '@/test/mocks/next-navigation';
import { render, screen } from '@testing-library/react';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

import Nav from '../Nav';

beforeEach(() => {
  mockUsePathname.mockReturnValue('/');
});

describe('Nav', () => {
  it('renders Baselines and Traces links', () => {
    render(<Nav />);
    expect(screen.getByText('Baselines')).toBeInTheDocument();
    expect(screen.getByText('Traces')).toBeInTheDocument();
  });

  it('Baselines link has active styles when pathname is /', () => {
    mockUsePathname.mockReturnValue('/');
    render(<Nav />);
    const baselines = screen.getByText('Baselines');
    expect(baselines).toHaveClass('bg-gray-800');
    const traces = screen.getByText('Traces');
    expect(traces).toHaveClass('text-gray-400');
  });

  it('Traces link has active styles when pathname is /traces', () => {
    mockUsePathname.mockReturnValue('/traces');
    render(<Nav />);
    const traces = screen.getByText('Traces');
    expect(traces).toHaveClass('bg-gray-800');
    const baselines = screen.getByText('Baselines');
    expect(baselines).toHaveClass('text-gray-400');
  });

  it('Traces link has active styles when pathname is /traces/some-id (startsWith match)', () => {
    mockUsePathname.mockReturnValue('/traces/some-id');
    render(<Nav />);
    const traces = screen.getByText('Traces');
    expect(traces).toHaveClass('bg-gray-800');
  });
});
