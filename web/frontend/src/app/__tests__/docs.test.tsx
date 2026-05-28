import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import '@/test/mocks/next-navigation';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

import DocsPage from '../docs/page';

describe('DocsPage', () => {
  it('renders the hero with H1 "Reference" and the version/openapi meta strip', () => {
    render(<DocsPage />);
    expect(
      screen.getByRole('heading', { level: 1, name: /Reference/i }),
    ).toBeInTheDocument();
    expect(screen.getByText('v0.4.1')).toBeInTheDocument();
    expect(screen.getByText('/api/openapi.json')).toBeInTheDocument();
  });

  it('renders all five top-level section headings', () => {
    render(<DocsPage />);
    expect(
      screen.getByRole('heading', { level: 2, name: /^Concepts$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 2, name: /^API reference$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 2, name: /^Data flow$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 2, name: /^Embed cache$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { level: 2, name: /^Interpretation$/i }),
    ).toBeInTheDocument();
  });

  it('renders a TOC entry per section with the section number', () => {
    render(<DocsPage />);
    const tocList = document.querySelector('ol.dx-toc') as HTMLElement;
    expect(tocList).not.toBeNull();
    const tocScope = within(tocList);
    ['01', '02', '03', '04', '05'].forEach((n) =>
      expect(tocScope.getByText(n)).toBeInTheDocument(),
    );
    expect(tocScope.getByText('Concepts')).toBeInTheDocument();
    expect(tocScope.getByText('API reference')).toBeInTheDocument();
    expect(tocScope.getByText('Data flow')).toBeInTheDocument();
    expect(tocScope.getByText('Embed cache')).toBeInTheDocument();
    expect(tocScope.getByText('Interpretation')).toBeInTheDocument();
  });

  it('renders each concept card with an anchored id', () => {
    render(<DocsPage />);
    for (const id of [
      'trace',
      'baseline',
      'strategy',
      'drift',
      'pathgraph',
      'overlay',
      'counterfactual',
      'edit-prompt',
      'similarity',
    ]) {
      expect(document.getElementById(id)).not.toBeNull();
    }
  });

  it('renders every endpoint card with its method pill', () => {
    render(<DocsPage />);
    expect(document.getElementById('list-baselines')).not.toBeNull();
    expect(document.getElementById('post-counterfactual')).not.toBeNull();
    expect(document.getElementById('get-similar')).not.toBeNull();
    expect(screen.getAllByText('GET').length).toBeGreaterThanOrEqual(5);
    expect(screen.getAllByText('POST').length).toBeGreaterThanOrEqual(2);
  });

  it('renders the embed-cache Go snippet with //go:embed and the GetEmbedding symbol', () => {
    render(<DocsPage />);
    const pre = Array.from(
      document.querySelectorAll<HTMLPreElement>('pre'),
    ).find(
      (p) =>
        p.textContent?.includes('//go:embed') &&
        p.textContent?.includes('GetEmbedding'),
    );
    expect(pre).toBeDefined();
    expect(pre?.textContent).toContain('seedCacheBytes');
  });
});
