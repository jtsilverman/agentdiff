import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import type { Step } from '@/lib/types';

vi.mock('@/lib/api');

import { runCounterfactual } from '@/lib/api';
import CounterfactualButton from '../CounterfactualButton';

const mockedRunCounterfactual = vi.mocked(runCounterfactual);

const steps: Step[] = [
  { role: 'user', content: 'list the files' },
  {
    role: 'assistant',
    content: 'reading',
    tool_call: { name: 'read_file', args: { path: '/tmp/x' } },
  },
  {
    role: 'assistant',
    content: 'writing',
    tool_call: { name: 'write_file', args: { path: '/tmp/x' } },
  },
];

describe('CounterfactualButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders collapsed What if? button by default', () => {
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    expect(screen.getByRole('button', { name: /What if\?/i })).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/modified input/i)).not.toBeInTheDocument();
  });

  it('expands to show step picker + modified input field + Run', () => {
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /What if\?/i }));
    expect(screen.getByLabelText(/step/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/modified input/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Run counterfactual/i })).toBeInTheDocument();
  });

  it('submitting calls runCounterfactual with the picked step + modified input', async () => {
    mockedRunCounterfactual.mockResolvedValue({
      new_trace_id: 'cf-1',
      comparison: { original_path: ['a'], new_path: ['b'], divergence_step: 0 },
    });
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /What if\?/i }));
    fireEvent.change(screen.getByLabelText(/step/i), { target: { value: '1' } });
    fireEvent.change(screen.getByPlaceholderText(/modified input/i), {
      target: { value: 'use bash instead' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Run counterfactual/i }));
    await waitFor(() => {
      expect(mockedRunCounterfactual).toHaveBeenCalledWith('trace-abc', 1, 'use bash instead');
    });
  });

  it('renders the CounterfactualGraph inline on success', async () => {
    mockedRunCounterfactual.mockResolvedValue({
      new_trace_id: 'cf-1',
      comparison: {
        original_path: ['start', 'read_file'],
        new_path: ['start', 'bash'],
        divergence_step: 1,
      },
    });
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /What if\?/i }));
    fireEvent.change(screen.getByPlaceholderText(/modified input/i), {
      target: { value: 'x' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Run counterfactual/i }));
    // 'bash' is unique to the counterfactual branch — SUT-unique text gate.
    await waitFor(() => {
      expect(screen.getByText('bash')).toBeInTheDocument();
    });
    expect(screen.getByText('read_file')).toBeInTheDocument();
  });

  it('shows error and stays expanded when runCounterfactual rejects', async () => {
    mockedRunCounterfactual.mockRejectedValue(new Error('LLM call failed'));
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /What if\?/i }));
    fireEvent.change(screen.getByPlaceholderText(/modified input/i), {
      target: { value: 'x' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Run counterfactual/i }));
    await waitFor(() => {
      expect(screen.getByText(/LLM call failed/i)).toBeInTheDocument();
    });
    // Input still visible so the user can fix and retry.
    expect(screen.getByPlaceholderText(/modified input/i)).toBeInTheDocument();
  });

  it('Run is disabled while modified input is empty', () => {
    render(<CounterfactualButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /What if\?/i }));
    expect(screen.getByRole('button', { name: /Run counterfactual/i })).toBeDisabled();
  });
});
