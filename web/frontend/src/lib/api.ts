import type {
  TraceSummary,
  TraceDetail,
  BaselineSummary,
  StrategyReport,
  DiffResponse,
  MatchResult,
} from './types';

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? '';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, init);
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

export function uploadTrace(
  body: string,
  name: string,
  adapter?: string,
): Promise<TraceSummary> {
  const params = new URLSearchParams({ name });
  if (adapter) params.set('adapter', adapter);
  return request<TraceSummary>(`/api/traces?${params.toString()}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  });
}

export function listTraces(): Promise<TraceSummary[]> {
  return request<TraceSummary[]>('/api/traces');
}

export function getTrace(id: string): Promise<TraceDetail> {
  return request<TraceDetail>(`/api/traces/${id}`);
}

export function createBaseline(
  name: string,
  traceIds: string[],
): Promise<BaselineSummary> {
  return request<BaselineSummary>('/api/baselines', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, trace_ids: traceIds }),
  });
}

export function listBaselines(): Promise<BaselineSummary[]> {
  return request<BaselineSummary[]>('/api/baselines');
}

export function getCluster(
  baselineId: string,
  epsilon?: number,
  minPoints?: number,
): Promise<StrategyReport> {
  const params = new URLSearchParams();
  if (epsilon !== undefined) params.set('epsilon', String(epsilon));
  if (minPoints !== undefined) params.set('min_points', String(minPoints));
  const qs = params.toString();
  return request<StrategyReport>(
    `/api/baselines/${baselineId}/cluster${qs ? `?${qs}` : ''}`,
  );
}

export function getDiff(idA: string, idB: string): Promise<DiffResponse> {
  return request<DiffResponse>(`/api/diff/${idA}/${idB}`);
}

export function compareTrace(
  baselineId: string,
  traceId: string,
): Promise<MatchResult> {
  return request<MatchResult>(`/api/baselines/${baselineId}/compare`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ trace_id: traceId }),
  });
}
