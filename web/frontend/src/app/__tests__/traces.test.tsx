import { vi, describe, it, expect, beforeEach } from 'vitest';
import '@/test/mocks/next-navigation';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { mockTrace } from '@/test/mocks/fixtures';

vi.mock('next/link', () => ({
  default: ({ children, href, ...props }: any) => (
    <a href={href} {...props}>{children}</a>
  ),
}));

vi.mock('@/lib/api');

import { listTraces, createBaseline } from '@/lib/api';
import TracesPage from '../traces/page';

const mockedListTraces = vi.mocked(listTraces);
const mockedCreateBaseline = vi.mocked(createBaseline);

const mockTrace2 = { ...mockTrace, id: 'trace-2', name: 'second-trace' };

describe('TracesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows "Loading traces..." initially', () => {
    mockedListTraces.mockReturnValue(new Promise(() => {}));
    render(<TracesPage />);
    expect(screen.getByText('Loading traces...')).toBeInTheDocument();
  });

  it('shows trace table with name, adapter, steps, date columns after load', async () => {
    mockedListTraces.mockResolvedValue([mockTrace]);
    render(<TracesPage />);
    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Adapter')).toBeInTheDocument();
    expect(screen.getByText('Steps')).toBeInTheDocument();
    expect(screen.getByText('Date')).toBeInTheDocument();
    expect(screen.getByText('claudecode')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('shows empty state "No traces yet" when no traces', async () => {
    mockedListTraces.mockResolvedValue([]);
    render(<TracesPage />);
    await waitFor(() => {
      expect(screen.getByText(/No traces yet/)).toBeInTheDocument();
    });
  });

  it('selecting traces shows baseline creation form; clicking Create Baseline calls createBaseline', async () => {
    mockedListTraces.mockResolvedValue([mockTrace, mockTrace2]);
    mockedCreateBaseline.mockResolvedValue({ id: 'bl-new', name: 'my-bl', trace_count: 1, created_at: '' });
    render(<TracesPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    // Select first trace checkbox (index 0 is select-all, index 1 is first trace)
    const checkboxes = screen.getAllByRole('checkbox');
    fireEvent.click(checkboxes[1]);

    // Baseline creation form should appear
    await waitFor(() => {
      expect(screen.getByPlaceholderText('Baseline name')).toBeInTheDocument();
    });

    // Type baseline name into the Tremor TextInput's underlying input
    const nameInput = screen.getByPlaceholderText('Baseline name');
    fireEvent.change(nameInput, { target: { value: 'my-bl' } });

    // Click create button
    const createBtn = screen.getByRole('button', { name: /Create Baseline/ });
    fireEvent.click(createBtn);

    await waitFor(() => {
      expect(mockedCreateBaseline).toHaveBeenCalledWith('my-bl', ['trace-1']);
    });
  });

  it('"Select all" checkbox toggles all trace checkboxes', async () => {
    mockedListTraces.mockResolvedValue([mockTrace, mockTrace2]);
    render(<TracesPage />);

    await waitFor(() => {
      expect(screen.getByText('test-trace')).toBeInTheDocument();
    });

    const checkboxes = screen.getAllByRole('checkbox');
    // First checkbox is select-all
    const selectAll = checkboxes[0];

    // Click select all
    fireEvent.click(selectAll);

    // All checkboxes should be checked
    await waitFor(() => {
      const allBoxes = screen.getAllByRole('checkbox');
      allBoxes.forEach((box) => {
        expect(box).toBeChecked();
      });
    });

    // Click select all again to deselect
    fireEvent.click(selectAll);

    await waitFor(() => {
      const allBoxes = screen.getAllByRole('checkbox');
      allBoxes.forEach((box) => {
        expect(box).not.toBeChecked();
      });
    });
  });
});
