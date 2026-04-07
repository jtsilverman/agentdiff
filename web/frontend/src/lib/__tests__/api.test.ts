import { vi, describe, it, expect, beforeEach } from 'vitest';
import {
  listTraces,
  getTrace,
  uploadTrace,
  createBaseline,
  listBaselines,
  getCluster,
  getDiff,
  compareTrace,
} from '../api';

function mockFetchResponse(data: unknown, ok = true, status = 200, statusText = 'OK') {
  return vi.fn().mockResolvedValue({
    ok,
    status,
    statusText,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(typeof data === 'string' ? data : JSON.stringify(data)),
  } as unknown as Response);
}

beforeEach(() => {
  globalThis.fetch = mockFetchResponse([]);
});

describe('listTraces', () => {
  it('calls GET /api/traces and returns parsed JSON array', async () => {
    const fixture = [{ id: 't1', name: 'trace1', adapter: 'raw', step_count: 3, created_at: '2026-01-01' }];
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await listTraces();

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/traces', undefined);
    expect(result).toEqual(fixture);
  });
});

describe('getTrace', () => {
  it('calls GET /api/traces/{id} and returns parsed JSON', async () => {
    const fixture = { id: 't1', name: 'trace1', adapter: 'raw', source: '', metadata: {}, steps: [], created_at: '2026-01-01' };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await getTrace('t1');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/traces/t1', undefined);
    expect(result).toEqual(fixture);
  });
});

describe('uploadTrace', () => {
  it('calls POST /api/traces?name=X with octet-stream content type', async () => {
    const fixture = { id: 't2', name: 'my-trace', adapter: 'raw', step_count: 5, created_at: '2026-01-01' };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await uploadTrace('raw body content', 'my-trace');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/traces?name=my-trace', {
      method: 'POST',
      headers: { 'Content-Type': 'application/octet-stream' },
      body: 'raw body content',
    });
    expect(result).toEqual(fixture);
  });

  it('includes adapter query param when provided', async () => {
    const fixture = { id: 't3', name: 'my-trace', adapter: 'claude', step_count: 2, created_at: '2026-01-01' };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await uploadTrace('body', 'my-trace', 'claude');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/traces?name=my-trace&adapter=claude', {
      method: 'POST',
      headers: { 'Content-Type': 'application/octet-stream' },
      body: 'body',
    });
    expect(result).toEqual(fixture);
  });
});

describe('createBaseline', () => {
  it('calls POST /api/baselines with JSON body containing name and trace_ids', async () => {
    const fixture = { id: 'b1', name: 'baseline1', trace_count: 2, created_at: '2026-01-01' };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await createBaseline('baseline1', ['t1', 't2']);

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/baselines', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'baseline1', trace_ids: ['t1', 't2'] }),
    });
    expect(result).toEqual(fixture);
  });
});

describe('listBaselines', () => {
  it('calls GET /api/baselines', async () => {
    const fixture = [{ id: 'b1', name: 'baseline1', trace_count: 2, created_at: '2026-01-01' }];
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await listBaselines();

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/baselines', undefined);
    expect(result).toEqual(fixture);
  });
});

describe('getCluster', () => {
  it('calls GET /api/baselines/{id}/cluster with no query params', async () => {
    const fixture = { baseline_name: 'b1', snapshot_count: 3, strategies: [], noise: [], epsilon: 0.3 };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await getCluster('b1');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/baselines/b1/cluster', undefined);
    expect(result).toEqual(fixture);
  });

  it('includes epsilon and min_points query params when provided', async () => {
    const fixture = { baseline_name: 'b1', snapshot_count: 3, strategies: [], noise: [], epsilon: 0.5 };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await getCluster('b1', 0.5, 3);

    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/baselines/b1/cluster?epsilon=0.5&min_points=3',
      undefined,
    );
    expect(result).toEqual(fixture);
  });
});

describe('getDiff', () => {
  it('calls GET /api/diff/{idA}/{idB}', async () => {
    const fixture = {
      trace_a: { id: 'a1', name: 'traceA' },
      trace_b: { id: 'b1', name: 'traceB' },
      alignment: [],
      distance: 2,
      summary: { matches: 3, insertions: 1, deletions: 0, substitutions: 1 },
    };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await getDiff('a1', 'b1');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/diff/a1/b1', undefined);
    expect(result).toEqual(fixture);
  });
});

describe('compareTrace', () => {
  it('calls POST /api/baselines/{id}/compare with JSON body containing trace_id', async () => {
    const fixture = { matched: true, strategy_id: 1, exemplar: 'e1', distance: 0.2, max_intra_cluster_dist: 0.5 };
    globalThis.fetch = mockFetchResponse(fixture);

    const result = await compareTrace('b1', 't1');

    expect(globalThis.fetch).toHaveBeenCalledWith('/api/baselines/b1/compare', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ trace_id: 't1' }),
    });
    expect(result).toEqual(fixture);
  });
});

describe('error handling', () => {
  it('throws Error with status code and response text when res.ok is false', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
      text: () => Promise.resolve('resource not found'),
    } as unknown as Response);

    await expect(listTraces()).rejects.toThrow('API error 404: resource not found');
  });

  it('falls back to statusText when res.text() rejects', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
      text: () => Promise.reject(new Error('stream failed')),
    } as unknown as Response);

    await expect(listTraces()).rejects.toThrow('API error 500: Internal Server Error');
  });
});
