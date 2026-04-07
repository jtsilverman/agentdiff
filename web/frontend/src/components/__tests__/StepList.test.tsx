import { vi, describe, it, expect } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { mockTraceDetail } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

import StepList from '../StepList';

const steps = mockTraceDetail.steps;

describe('StepList', () => {
  it('renders correct number of step cards', () => {
    render(<StepList steps={steps} />);
    // 3 steps => 3 index labels
    expect(screen.getByText('#1')).toBeInTheDocument();
    expect(screen.getByText('#2')).toBeInTheDocument();
    expect(screen.getByText('#3')).toBeInTheDocument();
  });

  it('shows step index numbers (#1, #2, #3)', () => {
    render(<StepList steps={steps} />);
    expect(screen.getByText('#1')).toBeInTheDocument();
    expect(screen.getByText('#2')).toBeInTheDocument();
    expect(screen.getByText('#3')).toBeInTheDocument();
  });

  it('shows role badges', () => {
    render(<StepList steps={steps} />);
    expect(screen.getByText('user')).toBeInTheDocument();
    expect(screen.getByText('assistant')).toBeInTheDocument();
    expect(screen.getByText('tool result')).toBeInTheDocument();
  });

  it('shows step content text', () => {
    render(<StepList steps={steps} />);
    expect(screen.getByText('Hello')).toBeInTheDocument();
    expect(screen.getByText('Hi')).toBeInTheDocument();
  });

  it('Collapsible: "Show args" button reveals tool_call args JSON; "Hide args" hides it', async () => {
    const user = userEvent.setup();
    render(<StepList steps={steps} />);

    // Initially, args JSON is not visible
    expect(screen.queryByText(/\/tmp\/x/)).not.toBeInTheDocument();

    // Click "Show args" to expand
    const showBtn = screen.getByText('Show args');
    await user.click(showBtn);

    // Args JSON should now be visible
    expect(screen.getByText(/\/tmp\/x/)).toBeInTheDocument();

    // Button text changes to "Hide args"
    const hideBtn = screen.getByText('Hide args');
    await user.click(hideBtn);

    // Args JSON should be hidden again
    expect(screen.queryByText(/\/tmp\/x/)).not.toBeInTheDocument();
  });
});
