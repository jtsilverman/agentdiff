import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import type { Step } from '@/lib/types';

vi.mock('@/lib/api');

import { editPrompt } from '@/lib/api';
import EditPromptButton from '../EditPromptButton';

const mockedEditPrompt = vi.mocked(editPrompt);

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

describe('EditPromptButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders collapsed Edit prompt button by default', () => {
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    expect(screen.getByRole('button', { name: /Edit prompt/i })).toBeInTheDocument();
    expect(screen.queryByPlaceholderText(/rewritten prompt/i)).not.toBeInTheDocument();
  });

  it('expands to show step picker + rewritten prompt field + Re-run button', () => {
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /Edit prompt/i }));
    expect(screen.getByLabelText(/prompt-edit step/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/rewritten prompt/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Re-run from here/i })).toBeInTheDocument();
  });

  it('submitting calls editPrompt with the picked step + modified prompt', async () => {
    mockedEditPrompt.mockResolvedValue({
      new_trace_id: 'ep-1',
      comparison: { original_path: ['a'], new_path: ['b'], divergence_step: 0 },
    });
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /Edit prompt/i }));
    fireEvent.change(screen.getByLabelText(/prompt-edit step/i), { target: { value: '1' } });
    fireEvent.change(screen.getByPlaceholderText(/rewritten prompt/i), {
      target: { value: 'search the src folder instead' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Re-run from here/i }));
    await waitFor(() => {
      expect(mockedEditPrompt).toHaveBeenCalledWith('trace-abc', 1, 'search the src folder instead');
    });
  });

  it('renders the CounterfactualGraph inline on success', async () => {
    mockedEditPrompt.mockResolvedValue({
      new_trace_id: 'ep-1',
      comparison: {
        original_path: ['start', 'read_file'],
        new_path: ['start', 'grep'],
        divergence_step: 1,
      },
    });
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /Edit prompt/i }));
    fireEvent.change(screen.getByPlaceholderText(/rewritten prompt/i), {
      target: { value: 'x' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Re-run from here/i }));
    // 'grep' is unique to the prompt-edit branch — SUT-unique text gate per
    // rtl-assertions-gate-on-sut-unique-text. read_file appears in both paths
    // here, so use grep to prove the new branch rendered.
    await waitFor(() => {
      expect(screen.getByText('grep')).toBeInTheDocument();
    });
    expect(screen.getByText('read_file')).toBeInTheDocument();
  });

  it('shows error and stays expanded when editPrompt rejects', async () => {
    mockedEditPrompt.mockRejectedValue(new Error('LLM call failed'));
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /Edit prompt/i }));
    fireEvent.change(screen.getByPlaceholderText(/rewritten prompt/i), {
      target: { value: 'x' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Re-run from here/i }));
    await waitFor(() => {
      expect(screen.getByText(/LLM call failed/i)).toBeInTheDocument();
    });
    expect(screen.getByPlaceholderText(/rewritten prompt/i)).toBeInTheDocument();
  });

  it('Re-run is disabled while rewritten prompt is empty', () => {
    render(<EditPromptButton traceId="trace-abc" steps={steps} />);
    fireEvent.click(screen.getByRole('button', { name: /Edit prompt/i }));
    expect(screen.getByRole('button', { name: /Re-run from here/i })).toBeDisabled();
  });
});
