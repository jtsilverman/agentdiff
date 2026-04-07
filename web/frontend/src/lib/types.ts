export interface TraceSummary {
  id: string;
  name: string;
  adapter: string;
  step_count: number;
  metadata?: Record<string, string>;
  created_at: string;
}

export interface TraceDetail {
  id: string;
  name: string;
  adapter: string;
  source: string;
  metadata: Record<string, string>;
  steps: Step[];
  created_at: string;
}

export interface Step {
  role: string;
  content: string;
  tool_call?: ToolCall;
  tool_result?: ToolResult;
}

export interface ToolCall {
  name: string;
  args: Record<string, unknown>;
}

export interface ToolResult {
  name: string;
  output: string;
  is_error: boolean;
}

export interface BaselineSummary {
  id: string;
  name: string;
  trace_count: number;
  created_at: string;
}

export interface StrategyReport {
  baseline_name: string;
  snapshot_count: number;
  strategies: Strategy[];
  noise: string[];
  epsilon: number;
}

export interface Strategy {
  id: number;
  count: number;
  exemplar: string;
  tool_sequence: string[];
  members: string[];
  metadata_summary?: Record<string, Record<string, number>>;
}

export interface DiffTraceRef {
  id: string;
  name: string;
}

export interface DiffResponse {
  trace_a: DiffTraceRef;
  trace_b: DiffTraceRef;
  alignment: AlignedPair[];
  distance: number;
  summary: DiffSummary;
}

export interface AlignedPair {
  a_step: Step | null;
  b_step: Step | null;
  op: string;
}

export interface DiffSummary {
  matches: number;
  insertions: number;
  deletions: number;
  substitutions: number;
}

export interface MatchResult {
  matched: boolean;
  strategy_id: number;
  exemplar: string;
  distance: number;
  max_intra_cluster_dist: number;
}
