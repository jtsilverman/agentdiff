import type {
  TraceSummary,
  TraceDetail,
  BaselineSummary,
  StrategyReport,
  DiffResponse,
  MatchResult,
  PathGraph,
  OverlayResult,
  TranscriptResponse,
  CounterfactualResponse,
  SimilarTracesResponse,
} from '@/lib/types';

export const mockTrace: TraceSummary = {
  id: 'trace-1',
  name: 'test-trace',
  adapter: 'claudecode',
  step_count: 3,
  created_at: '2026-01-01T00:00:00Z',
};

export const mockTraceDetail: TraceDetail = {
  ...mockTrace,
  source: 'stream-json',
  metadata: { model: 'claude-4' },
  steps: [
    { role: 'user', content: 'Hello' },
    {
      role: 'assistant',
      content: 'Hi',
      tool_call: { name: 'read_file', args: { path: '/tmp/x' } },
    },
    {
      role: 'tool_result',
      content: '',
      tool_result: {
        name: 'read_file',
        output: 'file contents',
        is_error: false,
      },
    },
  ],
};

export const mockBaseline: BaselineSummary = {
  id: 'bl-1',
  name: 'test-baseline',
  trace_count: 5,
  created_at: '2026-01-01T00:00:00Z',
};

export const mockStrategyReport: StrategyReport = {
  baseline_name: 'test-baseline',
  snapshot_count: 10,
  strategies: [
    {
      id: 0,
      count: 7,
      exemplar: { id: 't1', name: 'trace-1' },
      tool_sequence: ['read_file', 'write_file'],
      members: [
        { id: 't1', name: 'trace-1' },
        { id: 't2', name: 'trace-2' },
      ],
    },
  ],
  noise: [{ id: 'to1', name: 'trace-outlier' }],
  epsilon: 0.3,
};

export const mockDiff: DiffResponse = {
  trace_a: { id: 'a1', name: 'trace-a' },
  trace_b: { id: 'b1', name: 'trace-b' },
  alignment: [
    {
      a_step: { role: 'user', content: 'Hello' },
      b_step: { role: 'user', content: 'Hello' },
      op: 'match',
    },
    {
      a_step: { role: 'assistant', content: 'A reply' },
      b_step: null,
      op: 'delete',
    },
    {
      a_step: null,
      b_step: { role: 'assistant', content: 'Different' },
      op: 'insert',
    },
  ],
  distance: 2,
  summary: { matches: 1, insertions: 1, deletions: 1, substitutions: 0 },
};

export const mockMatchResult: MatchResult = {
  matched: true,
  strategy_id: 0,
  exemplar: 'trace-1',
  distance: 2,
  max_intra_cluster_dist: 5,
};

export const mockDriftResult: MatchResult = {
  matched: false,
  strategy_id: -1,
  exemplar: '',
  distance: 12.345,
  max_intra_cluster_dist: 5,
};

export const mockPathGraph: PathGraph = {
  nodes: [
    { id: 'read_file', tool_name: 'read_file', count: 5, cost_tokens: 2500, latency_ms: 5000 },
    { id: 'write_file', tool_name: 'write_file', count: 4, cost_tokens: 800, latency_ms: 1200 },
    { id: 'bash', tool_name: 'bash', count: 1, cost_tokens: 9000, latency_ms: 11000 },
  ],
  edges: [
    { from: 'read_file', to: 'write_file', count: 4, weight: 0.8 },
    { from: 'read_file', to: 'bash', count: 1, weight: 0.2 },
  ],
  stats: { total_runs: 5, branch_points: 1 },
};

export const mockEmptyPathGraph: PathGraph = {
  nodes: [],
  edges: [],
  stats: { total_runs: 0, branch_points: 0 },
};

export const mockPathGraphNoCost: PathGraph = {
  nodes: [
    { id: 'legacy_tool', tool_name: 'legacy_tool', count: 3 },
    { id: 'another', tool_name: 'another', count: 2 },
  ],
  edges: [{ from: 'legacy_tool', to: 'another', count: 2, weight: 1.0 }],
  stats: { total_runs: 3, branch_points: 0 },
};

export const mockTranscript: TranscriptResponse = {
  summary: 'Agent read the file, applied the rename, and confirmed the change in one turn.',
  key_decisions: [
    'used read_file before write_file',
    'kept the original whitespace',
    'reported the renamed path back to the user',
  ],
};

export const mockOverlay: OverlayResult = {
  matched_node_ids: ['read_file', 'write_file'],
  matched_edge_ids: ['read_file->write_file'],
  divergence_points: [
    { node_id: 'read_file', baseline_pct: 0.2, this_trace_chose: 'bash' },
  ],
};

export const mockCounterfactual: CounterfactualResponse = {
  new_trace_id: 'cf-trace-1',
  comparison: {
    original_path: ['start', 'read_file', 'write_file'],
    new_path: ['start', 'edit_file', 'bash'],
    divergence_step: 1,
  },
};

export const mockSimilarTraces: SimilarTracesResponse = {
  matches: [
    { trace_id: 'sim-1', name: 'grep-then-cat', similarity_score: 0.92 },
    { trace_id: 'sim-2', name: 'grep-then-head', similarity_score: 0.84 },
    { trace_id: 'sim-3', name: 'find-then-cat', similarity_score: 0.71 },
    { trace_id: 'sim-4', name: 'ls-then-cat', similarity_score: 0.58 },
    { trace_id: 'sim-5', name: 'echo-then-write', similarity_score: 0.33 },
  ],
};

export const mockEmptySimilarTraces: SimilarTracesResponse = {
  matches: [],
};
