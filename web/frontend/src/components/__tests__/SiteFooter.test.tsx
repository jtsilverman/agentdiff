import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import SiteFooter from '../SiteFooter';

describe('SiteFooter introduce-me surface', () => {
  it('renders Jake’s identity, role pill, and CV blurb', () => {
    render(<SiteFooter />);
    expect(screen.getByText('Jake Silverman')).toBeInTheDocument();
    expect(screen.getByText(/forward-deployed eng/i)).toBeInTheDocument();
    expect(screen.getByText(/Azure Databricks and Microsoft Fabric/i)).toBeInTheDocument();
    expect(screen.getByText(/PwC.* Transfer Pricing AI competition/i)).toBeInTheDocument();
  });

  it('renders working contact CTAs (mailto, LinkedIn, GitHub)', () => {
    render(<SiteFooter />);
    const email = screen.getByRole('link', { name: /email jake/i });
    expect(email).toHaveAttribute('href', 'mailto:jakesilverman.pro@gmail.com');

    const linkedin = screen.getByRole('link', { name: 'LinkedIn' });
    expect(linkedin).toHaveAttribute('href', 'https://www.linkedin.com/in/jacob-silverman1/');
    expect(linkedin).toHaveAttribute('target', '_blank');

    const github = screen.getByRole('link', { name: 'GitHub' });
    expect(github).toHaveAttribute('href', 'https://github.com/jtsilverman');
    expect(github).toHaveAttribute('target', '_blank');
  });
});
