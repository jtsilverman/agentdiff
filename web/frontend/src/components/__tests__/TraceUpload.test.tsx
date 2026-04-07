import { vi, describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { mockTrace } from '@/test/mocks/fixtures';

vi.mock('@/lib/api', () => ({
  uploadTrace: vi.fn(),
}));

import { uploadTrace } from '@/lib/api';
import TraceUpload from '../TraceUpload';

const mockedUploadTrace = uploadTrace as ReturnType<typeof vi.fn>;

function createDropEvent(file: File) {
  return {
    preventDefault: vi.fn(),
    dataTransfer: {
      files: [file],
      items: [{ kind: 'file', type: file.type, getAsFile: () => file }],
    },
  };
}

describe('TraceUpload', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders drop zone with "Drop JSONL file here" text and name input', () => {
    render(<TraceUpload onUploaded={vi.fn()} />);
    expect(screen.getByText('Drop JSONL file here')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('Name override (optional, defaults to filename)'),
    ).toBeInTheDocument();
  });

  it('shows "Uploading..." during upload', async () => {
    // Create a promise that never resolves so we can check the uploading state
    mockedUploadTrace.mockReturnValue(new Promise(() => {}));

    render(<TraceUpload onUploaded={vi.fn()} />);

    const dropZone = screen.getByText('Drop JSONL file here').closest('div[class*="border-dashed"]')!;
    const file = new File(['{"data":"test"}'], 'trace.jsonl', { type: 'application/json' });

    fireEvent.drop(dropZone, createDropEvent(file));

    await waitFor(() => {
      expect(screen.getByText('Uploading...')).toBeInTheDocument();
    });
  });

  it('shows success message after upload completes and calls onUploaded', async () => {
    mockedUploadTrace.mockResolvedValue(mockTrace);
    const onUploaded = vi.fn();

    render(<TraceUpload onUploaded={onUploaded} />);

    const dropZone = screen.getByText('Drop JSONL file here').closest('div[class*="border-dashed"]')!;
    const file = new File(['{"data":"test"}'], 'trace.jsonl', { type: 'application/json' });

    fireEvent.drop(dropZone, createDropEvent(file));

    await waitFor(() => {
      expect(screen.getByText('Uploaded "trace" successfully.')).toBeInTheDocument();
    });
    expect(onUploaded).toHaveBeenCalled();
  });

  it('shows error message when upload fails', async () => {
    mockedUploadTrace.mockRejectedValue(new Error('Network failure'));

    render(<TraceUpload onUploaded={vi.fn()} />);

    const dropZone = screen.getByText('Drop JSONL file here').closest('div[class*="border-dashed"]')!;
    const file = new File(['bad'], 'trace.jsonl', { type: 'application/json' });

    fireEvent.drop(dropZone, createDropEvent(file));

    await waitFor(() => {
      expect(screen.getByText('Network failure')).toBeInTheDocument();
    });
  });

  it('uses filename minus .jsonl extension as trace name when name input is empty', async () => {
    mockedUploadTrace.mockResolvedValue(mockTrace);

    render(<TraceUpload onUploaded={vi.fn()} />);

    const dropZone = screen.getByText('Drop JSONL file here').closest('div[class*="border-dashed"]')!;
    const file = new File(['{"line":1}'], 'my-trace.jsonl', { type: 'application/json' });

    fireEvent.drop(dropZone, createDropEvent(file));

    await waitFor(() => {
      expect(mockedUploadTrace).toHaveBeenCalledWith('{"line":1}', 'my-trace', undefined, undefined);
    });
  });

  it('add metadata button adds entry row with key and value inputs', () => {
    render(<TraceUpload onUploaded={vi.fn()} />);
    const addBtn = screen.getByText('Add metadata');

    fireEvent.click(addBtn);

    expect(screen.getByPlaceholderText('key')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('value')).toBeInTheDocument();
  });

  it('passes metadata entries to uploadTrace', async () => {
    mockedUploadTrace.mockResolvedValue(mockTrace);

    render(<TraceUpload onUploaded={vi.fn()} />);

    // Add a metadata entry
    fireEvent.click(screen.getByText('Add metadata'));

    const keyInput = screen.getByPlaceholderText('key');
    const valueInput = screen.getByPlaceholderText('value');

    fireEvent.change(keyInput, { target: { value: 'env' } });
    fireEvent.change(valueInput, { target: { value: 'prod' } });

    // Drop a file
    const dropZone = screen.getByText('Drop JSONL file here').closest('div[class*="border-dashed"]')!;
    const file = new File(['{"data":"test"}'], 'trace.jsonl', { type: 'application/json' });
    fireEvent.drop(dropZone, createDropEvent(file));

    await waitFor(() => {
      expect(mockedUploadTrace).toHaveBeenCalledWith(
        '{"data":"test"}',
        'trace',
        undefined,
        { env: 'prod' },
      );
    });
  });
});
