import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import '@/test/mocks/next-navigation';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

import ChangelogPage from '../changelog/page';

describe('ChangelogPage', () => {
  it('renders the hero with H1 "Changelog" and the four hero stat cells', () => {
    render(<ChangelogPage />);
    expect(
      screen.getByRole('heading', { level: 1, name: /^Changelog$/i }),
    ).toBeInTheDocument();
    const hero = document.querySelector(
      'header.tr-hero.cl-hero',
    ) as HTMLElement;
    expect(hero).not.toBeNull();
    const scope = within(hero);
    expect(scope.getByText('total')).toBeInTheDocument();
    expect(scope.getByText('features')).toBeInTheDocument();
    expect(scope.getByText('infra')).toBeInTheDocument();
    expect(scope.getByText('docs')).toBeInTheDocument();
  });

  it('renders entries grouped under month headings (May 2026 + April 2026)', () => {
    render(<ChangelogPage />);
    expect(
      screen.getByRole('heading', { level: 2, name: /May 2026/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 2, name: /April 2026/i }),
    ).toBeInTheDocument();
  });

  it('renders representative entry titles from the git log', () => {
    render(<ChangelogPage />);
    expect(screen.getByText('Site redesign')).toBeInTheDocument();
    expect(screen.getByText('Counterfactual replay')).toBeInTheDocument();
    expect(screen.getByText('Strategy clustering')).toBeInTheDocument();
  });

  it('filters entries by tag when a seg button is clicked', () => {
    render(<ChangelogPage />);
    expect(screen.getByText('Developer reference')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /^infra/i }));
    expect(screen.queryByText('Developer reference')).not.toBeInTheDocument();
    expect(screen.getByText('Hosted demo live')).toBeInTheDocument();
  });

  it('shows an empty state when no entries match the selected tag', () => {
    render(<ChangelogPage />);
    fireEvent.click(screen.getByRole('button', { name: /^fix/i }));
    expect(
      screen.getByText(/No entries with this tag/i),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /Show all/i }));
    expect(screen.getByText('Site redesign')).toBeInTheDocument();
  });
});
