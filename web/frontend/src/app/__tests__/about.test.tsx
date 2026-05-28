import { describe, it, expect, vi } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, within } from '@testing-library/react';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

import AboutPage from '../about/page';

const CONCEPTS = [
  'Trace',
  'Baseline',
  'Strategy',
  'Drift',
  'Path graph',
  'Overlay',
  'Counterfactual replay',
  'Edit-prompt rewrite',
];

const TOUR = [
  'Home page',
  'Baseline detail',
  'Trace detail',
  'Counterfactual replay',
  'Edit-prompt rewrite',
];

const MONEY = [
  'Counterfactual replay',
  'Edit-prompt rewrite',
  'Similarity search',
  'AI triage',
];

describe('AboutPage', () => {
  it('renders the hero with the six-minute tour title', () => {
    const { container } = render(<AboutPage />);
    const hero = container.querySelector('.ab-hero');
    expect(hero).not.toBeNull();
    expect(hero!.textContent).toMatch(/agentdiff in/i);
    expect(hero!.textContent).toMatch(/~6 minutes/i);
  });

  it('renders all 8 concept cards inside the concepts section', () => {
    const { container } = render(<AboutPage />);
    const concepts = container.querySelector('#concepts');
    expect(concepts).not.toBeNull();
    const scope = within(concepts as HTMLElement);
    CONCEPTS.forEach((title) => {
      expect(scope.getByText(title)).toBeInTheDocument();
    });
  });

  it('renders all 5 tour cards inside the tour section', () => {
    const { container } = render(<AboutPage />);
    const tourCards = container.querySelector('#tour .tour-cards');
    expect(tourCards).not.toBeNull();
    const scope = within(tourCards as HTMLElement);
    TOUR.forEach((title) => {
      expect(scope.getByText(title)).toBeInTheDocument();
    });
  });

  it('renders all 4 money-feature cards inside the money section', () => {
    const { container } = render(<AboutPage />);
    const money = container.querySelector('#money');
    expect(money).not.toBeNull();
    const scope = within(money as HTMLElement);
    MONEY.forEach((title) => {
      expect(scope.getByText(title)).toBeInTheDocument();
    });
  });

  it('renders the CTA section with "Now go try it." heading', () => {
    const { container } = render(<AboutPage />);
    const cta = container.querySelector('#start');
    expect(cta).not.toBeNull();
    const scope = within(cta as HTMLElement);
    expect(scope.getByText(/Now go try it\./i)).toBeInTheDocument();
  });

  it('renders the about-jake section with what-I-do and get-in-touch', () => {
    const { container } = render(<AboutPage />);
    const aboutJake = container.querySelector('#about-jake');
    expect(aboutJake).not.toBeNull();
    const scope = within(aboutJake as HTMLElement);
    expect(scope.getByText(/What I do/i)).toBeInTheDocument();
    expect(scope.getByText(/Get in touch/i)).toBeInTheDocument();
    const mail = aboutJake!.querySelector('a[href^="mailto:"]');
    expect(mail).not.toBeNull();
    expect(mail!.getAttribute('href')).toBe('mailto:jakesilverman.pro@gmail.com');
    const linkedin = aboutJake!.querySelector('a[href*="linkedin.com"]');
    expect(linkedin).not.toBeNull();
    const github = aboutJake!.querySelector('a[href*="github.com"]');
    expect(github).not.toBeNull();
  });
});
