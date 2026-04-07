import { vi, describe, it, expect } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen } from '@testing-library/react';
import { mockMatchResult, mockDriftResult } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

import DriftBadge from '../DriftBadge';

describe('DriftBadge', () => {
  it('shows "Matches Strategy X" green badge when matched is true', () => {
    render(<DriftBadge result={mockMatchResult} />);
    expect(screen.getByText('Matches Strategy 0')).toBeInTheDocument();
  });

  it('shows "New Strategy Detected" red badge when matched is false', () => {
    render(<DriftBadge result={mockDriftResult} />);
    expect(screen.getByText('New Strategy Detected')).toBeInTheDocument();
  });

  it('shows integer distance without decimal places', () => {
    render(<DriftBadge result={mockMatchResult} />);
    expect(screen.getByText('distance: 2')).toBeInTheDocument();
  });

  it('shows float distance with 3 decimal places', () => {
    render(<DriftBadge result={mockDriftResult} />);
    expect(screen.getByText('distance: 12.345')).toBeInTheDocument();
  });
});
