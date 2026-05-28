import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, cleanup, within } from '@testing-library/react';
import Tweaks from '../Tweaks';

const STORAGE_KEY = 'agentdiff:tweaks';

describe('Tweaks panel', () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.body.removeAttribute('data-density');
    document.documentElement.removeAttribute('style');
  });

  it('launcher opens the panel', () => {
    render(<Tweaks />);
    expect(screen.queryByRole('dialog', { name: /tweaks/i })).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: /open tweaks/i }));
    expect(screen.getByRole('dialog', { name: /tweaks/i })).toBeInTheDocument();
  });

  it('persists density choice to localStorage and applies data-density to body', () => {
    render(<Tweaks />);
    fireEvent.click(screen.getByRole('button', { name: /open tweaks/i }));
    fireEvent.click(screen.getByRole('radio', { name: 'compact' }));

    expect(document.body.getAttribute('data-density')).toBe('compact');
    const stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '{}');
    expect(stored.density).toBe('compact');
  });

  it('hydrates from previously stored tweaks on mount', () => {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ density: 'spacious', accent: '#fb923c', bgTone: 'warm' }),
    );
    render(<Tweaks />);
    expect(document.body.getAttribute('data-density')).toBe('spacious');
    expect(document.documentElement.style.getPropertyValue('--accent')).toBe('#fb923c');
    expect(document.documentElement.style.getPropertyValue('--bg')).toBe('#0a0907');
  });

  it('applies the chosen accent color to --accent on the document root', () => {
    render(<Tweaks />);
    fireEvent.click(screen.getByRole('button', { name: /open tweaks/i }));
    fireEvent.click(screen.getByRole('radio', { name: '#a78bfa' }));

    expect(document.documentElement.style.getPropertyValue('--accent')).toBe('#a78bfa');
    const stored = JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '{}');
    expect(stored.accent).toBe('#a78bfa');
  });

  it('closes the panel via the close button', () => {
    render(<Tweaks />);
    fireEvent.click(screen.getByRole('button', { name: /open tweaks/i }));
    const dialog = screen.getByRole('dialog', { name: /tweaks/i });
    fireEvent.click(within(dialog).getByRole('button', { name: /close tweaks/i }));
    expect(screen.queryByRole('dialog', { name: /tweaks/i })).toBeNull();
    cleanup();
  });
});
