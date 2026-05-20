import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Step } from '@/lib/types';
import Scrubber from '../Scrubber';

const tenSteps: Step[] = Array.from({ length: 10 }, (_, i) => ({
  role: i % 2 === 0 ? 'user' : 'assistant',
  content: `content for step ${i}`,
}));

const mixedSteps: Step[] = [
  { role: 'user', content: 'list the files' },
  {
    role: 'assistant',
    content: 'reading the file now',
    tool_call: { name: 'read_file', args: { path: '/etc/hosts' } },
  },
  {
    role: 'tool_result',
    content: '',
    tool_result: { name: 'read_file', output: '127.0.0.1 localhost', is_error: false },
  },
];

describe('Scrubber', () => {
  it('renders a range slider with min=0 and max=steps.length-1', () => {
    render(<Scrubber steps={tenSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    expect(slider).toHaveAttribute('min', '0');
    expect(slider).toHaveAttribute('max', '9');
    expect(slider).toHaveValue('0');
  });

  it('shows step 0 in the side panel by default with role, content, and "Step 1 of 10"', () => {
    render(<Scrubber steps={tenSteps} />);
    expect(screen.getByText(/Step 1 of 10/i)).toBeInTheDocument();
    expect(screen.getByText('content for step 0')).toBeInTheDocument();
    expect(screen.getByText(/user/)).toBeInTheDocument();
  });

  it('updates the side panel when the slider moves forward', () => {
    render(<Scrubber steps={tenSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    fireEvent.change(slider, { target: { value: '4' } });
    expect(screen.getByText(/Step 5 of 10/i)).toBeInTheDocument();
    expect(screen.getByText('content for step 4')).toBeInTheDocument();
    expect(screen.queryByText('content for step 0')).not.toBeInTheDocument();
  });

  it('supports backward scrub', () => {
    render(<Scrubber steps={tenSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    fireEvent.change(slider, { target: { value: '7' } });
    expect(screen.getByText('content for step 7')).toBeInTheDocument();
    fireEvent.change(slider, { target: { value: '2' } });
    expect(screen.getByText('content for step 2')).toBeInTheDocument();
    expect(screen.queryByText('content for step 7')).not.toBeInTheDocument();
  });

  it('past-last-step is a no-op (range input clamps to max)', () => {
    render(<Scrubber steps={tenSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    // Move to last step.
    fireEvent.change(slider, { target: { value: '9' } });
    expect(screen.getByText(/Step 10 of 10/i)).toBeInTheDocument();
    // Try to drive past the max; component should clamp (no crash, no out-of-bounds index).
    fireEvent.change(slider, { target: { value: '50' } });
    expect(screen.getByText(/Step 10 of 10/i)).toBeInTheDocument();
    expect(screen.getByText('content for step 9')).toBeInTheDocument();
  });

  it('renders tool_call name + args for an assistant step with a tool call', () => {
    render(<Scrubber steps={mixedSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    fireEvent.change(slider, { target: { value: '1' } });
    expect(screen.getByText('read_file')).toBeInTheDocument();
    expect(screen.getByText(/\/etc\/hosts/)).toBeInTheDocument();
  });

  it('renders tool_result output for a tool_result step', () => {
    render(<Scrubber steps={mixedSteps} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    fireEvent.change(slider, { target: { value: '2' } });
    expect(screen.getByText('127.0.0.1 localhost')).toBeInTheDocument();
  });

  it('renders gracefully on an empty steps array', () => {
    const { container } = render(<Scrubber steps={[]} />);
    // No slider, no crash; the component should show an empty-state hint.
    expect(screen.queryByRole('slider')).not.toBeInTheDocument();
    expect(container.textContent).toMatch(/no steps/i);
  });

  it('renders a single-step trace with min=0 max=0', () => {
    const oneStep: Step[] = [{ role: 'user', content: 'only step' }];
    render(<Scrubber steps={oneStep} />);
    const slider = screen.getByRole('slider', { name: /step scrubber/i });
    expect(slider).toHaveAttribute('min', '0');
    expect(slider).toHaveAttribute('max', '0');
    expect(screen.getByText(/Step 1 of 1/i)).toBeInTheDocument();
    expect(screen.getByText('only step')).toBeInTheDocument();
  });
});
