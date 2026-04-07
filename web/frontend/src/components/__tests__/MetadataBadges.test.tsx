import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MetadataBadges from '../MetadataBadges';

describe('MetadataBadges', () => {
  it('renders nothing when metadata is undefined', () => {
    const { container } = render(<MetadataBadges />);
    expect(container.firstChild).toBeNull();
  });

  it('renders nothing when metadata is empty object', () => {
    const { container } = render(<MetadataBadges metadata={{}} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders a badge for each key-value pair', () => {
    render(<MetadataBadges metadata={{ model: 'gpt-4', env: 'prod' }} />);
    expect(screen.getByText('model: gpt-4')).toBeInTheDocument();
    expect(screen.getByText('env: prod')).toBeInTheDocument();
  });

  it('uses "key: value" format in badges', () => {
    render(<MetadataBadges metadata={{ version: 'v2' }} />);
    expect(screen.getByText('version: v2')).toBeInTheDocument();
  });
});
