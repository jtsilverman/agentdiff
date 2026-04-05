# AgentDiff v3

**pytest for AI agents.** New adapters for OpenAI Agents SDK, LangChain, and custom formats. DBSCAN clustering to identify behavioral strategies in baselines.

## Problem

AgentDiff supports only Claude and OpenAI conversation logs, leaving users of the OpenAI Agents SDK, LangChain, and custom frameworks unable to snapshot their agent traces. Additionally, baselines accumulate runs with no way to identify distinct behavioral strategies (e.g., an agent that sometimes searches then summarizes vs. one that directly answers), making regression signals noisy when the agent legitimately takes different paths.

## Solution

Add three new adapters following the existing registry pattern, plus a DBSCAN clustering package that groups baseline snapshots by tool-call sequence similarity.

**Architecture decisions:**
- **Adapter registry pattern (existing):** Each adapter is a single file implementing `Parse(input []byte) ([]Step, map[string]string, error)`. Registration in `adapter.go` init(). Detection in `detect.go`. Proven pattern, no changes needed.
- **Generic adapter uses config, not code:** Users define field mappings in `.agentdiff.yaml` with dot-notation paths for nested JSON. Selected explicitly via `--adapter generic` (not auto-detected, since any JSONL could match).
- **DBSCAN over k-means:** Agent strategies are non-spherical clusters with unknown count. DBSCAN finds arbitrary-shaped clusters and identifies noise (outlier traces). No need to pre-specify k.
- **Levenshtein reuse:** Export the existing `levenshtein()` and `extractToolNames()` from `internal/diff/tools.go` for the cluster package. Zero new dependencies.
- **Pure Go, zero new deps:** DBSCAN is ~80 lines. Epsilon auto-selection is ~50 lines. Not worth an external library.
- **Scale note:** DBSCAN distance matrix is O(n^2). Expected baseline size: 5-100 snapshots. For baselines >200 snapshots, recommend setting `--epsilon` manually to avoid slow auto-selection.

## Scope

**Building:**
- OpenAI Agents SDK adapter (hierarchical span JSON with nested `children`)
- LangChain callback adapter (JSONL with `on_tool_start`/`on_tool_end` events)
- Generic JSONL adapter (configurable field mapping via `.agentdiff.yaml`)
- Auto-detection updates for Agents SDK and LangChain formats
- `internal/cluster/` package: DBSCAN algorithm, snapshot clustering, epsilon auto-selection
- `agentdiff cluster <baseline>` subcommand with terminal and JSON output
- `agentdiff cluster compare <baseline> <snapshot>` to match a snapshot against known strategies
- Config extensions for `adapter.generic` and `cluster` sections
- Testdata fixtures for all three new formats
- Integration tests for record + cluster round-trips

**Not building:**
- `agentdiff run` (execution orchestration)
- VCR-style tool response mocking (v4)
- OTel/OpenTelemetry ingestion (unstable spec)
- Web UI / dashboard
- GitHub Check Runs API annotations
- Vercel AI SDK adapter (existing OpenAI adapter handles it)

**Ship target:** GitHub (jtsilverman/agentdiff), tag v0.3.0

## Stack

Go 1.24, Cobra, gopkg.in/yaml.v3, stdlib only. Same as v2. Zero new external dependencies.

## Architecture

### File Structure (new/modified only)

```
agentdiff/
  internal/
    adapter/
      adapter.go              # MODIFY: register agents_sdk + langchain
      detect.go               # MODIFY: add detection for agents_sdk + langchain
      detect_test.go          # MODIFY: add detection tests
      agents_sdk.go           # NEW: OpenAI Agents SDK span parser
      agents_sdk_test.go      # NEW
      langchain.go            # NEW: LangChain callback JSONL parser
      langchain_test.go       # NEW
      generic.go              # NEW: configurable JSONL parser
      generic_test.go         # NEW
    diff/
      tools.go                # MODIFY: export Levenshtein + ExtractToolNames
    cluster/                  # NEW package
      dbscan.go               # DBSCAN algorithm
      dbscan_test.go
      cluster.go              # High-level: snapshots -> StrategyReport
      cluster_test.go
      epsilon.go              # Auto epsilon via k-distance elbow
      epsilon_test.go
    config/
      config.go               # MODIFY: add GenericAdapterConfig + ClusterConfig
      config_test.go          # MODIFY: add config merge tests
  cmd/
    cluster.go                # NEW: agentdiff cluster subcommand
    cluster_test.go           # NEW
    record.go                 # MODIFY: update adapterSourceName
    integration_test.go       # MODIFY: add v3 integration tests
  testdata/
    agents_sdk_trace.json     # NEW
    langchain_callbacks.jsonl # NEW
    generic_trace.jsonl       # NEW
```

### Data Models

```go
// internal/cluster/dbscan.go
type Cluster struct {
    ID       int   // cluster index, -1 for noise
    Members  []int // indices into input slice
    Exemplar int   // most central member (min avg distance to others)
}

type DBSCANResult struct {
    Clusters []Cluster
    Noise    []int   // indices of noise points
    Epsilon  float64 // epsilon used (may be auto-selected)
    MinPts   int
}

// internal/cluster/cluster.go
type StrategyReport struct {
    BaselineName  string     `json:"baseline_name"`
    SnapshotCount int        `json:"snapshot_count"`
    Strategies    []Strategy `json:"strategies"`
    Noise         []string   `json:"noise"`
    Epsilon       float64    `json:"epsilon"`
}

type Strategy struct {
    ID       int      `json:"id"`
    Count    int      `json:"count"`
    Exemplar string   `json:"exemplar"`
    ToolSeq  []string `json:"tool_sequence"`
    Members  []string `json:"members"`
}

type MatchResult struct {
    Matched             bool   `json:"matched"`
    StrategyID          int    `json:"strategy_id"`
    Exemplar            string `json:"exemplar"`
    Distance            int    `json:"distance"`
    MaxIntraClusterDist int    `json:"max_intra_cluster_dist"`
}

// internal/config/config.go additions
type GenericAdapterConfig struct {
    RoleField       string            `yaml:"role_field"`
    RoleMap         map[string]string `yaml:"role_map"`
    ToolNameField   string            `yaml:"tool_name_field"`
    ToolArgsField   string            `yaml:"tool_args_field"`
    ToolOutputField string            `yaml:"tool_output_field"`
    ContentField    string            `yaml:"content_field"`
}

type ClusterConfig struct {
    Epsilon   float64 `yaml:"epsilon"`
    MinPoints int     `yaml:"min_points"`
}
```

### Config Extension (.agentdiff.yaml)

```yaml
adapter:
  generic:
    role_field: "type"
    role_map:
      "llm_call": "assistant"
      "tool_invoke": "tool_call"
      "tool_output": "tool_result"
    tool_name_field: "function.name"
    tool_args_field: "function.arguments"
    tool_output_field: "result"
    content_field: "text"

cluster:
  epsilon: 0       # 0 = auto-select via elbow method
  min_points: 2
```

### CLI Contracts

```
# New adapter names for --adapter flag
agentdiff record <trace-file> --adapter agents_sdk
agentdiff record <trace-file> --adapter langchain
agentdiff record <trace-file> --adapter generic   # requires adapter.generic config
agentdiff record <trace-file>                      # auto-detect (agents_sdk, langchain, claude, openai)

# New cluster subcommand
agentdiff cluster <baseline-name> [--json] [--epsilon N] [--min-points N]
  # Output: N strategies found, exemplar per strategy, noise traces

agentdiff cluster compare <baseline-name> <snapshot> [--json]
  # Output: matched/unmatched, closest strategy, distance
```

## Tasks

### Task 1: Export Levenshtein and ExtractToolNames

**Files:** `internal/diff/tools.go`, `internal/diff/tools_test.go`

**Do:**
- Rename `levenshtein` to `Levenshtein` (exported). Rename `extractToolNames` to `ExtractToolNames` (exported).
- Update all callers within the `diff` package (`tools.go` and `tools_test.go`) to use the new names. There are 2 call sites for `levenshtein` (none currently, it was kept for direct use but `Align` replaced it in v2 -- verify) and multiple for `extractToolNames`.
- Add direct unit tests for the exported functions:
  - `TestLevenshteinDirect`: empty vs non-empty (returns len), identical (returns 0), single substitution (returns 1), completely different (returns max len).
  - `TestExtractToolNamesDirect`: empty steps (nil), steps with no tool calls (nil), mixed steps with some tool calls (returns only tool call names in order).
- Run full test suite to verify no regressions from the rename.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/diff/ -run "TestLevenshtein|TestExtractToolNames" -v && go test ./... -count=1`

**Dependencies:** none

### Task 2: OpenAI Agents SDK Adapter

**Files:** `internal/adapter/agents_sdk.go` (new), `internal/adapter/agents_sdk_test.go` (new), `testdata/agents_sdk_trace.json` (new)

**Do:**
- Create `AgentsSdkAdapter` struct implementing `Adapter` interface.
- Parse JSON input with structure: `{"trace_id": "...", "spans": [...]}`. Each span has `type` (string: "agent", "function", "generation"), `name` (string), `span_data` (object), and optional `children` (array of spans).
- Recursively walk the span tree depth-first. For each span:
  - `type: "function"`: emit a `tool_call` step with `Name = span.name`, `Args` from `span_data.input` (if it's a map, use directly; if string, wrap as `{"input": value}`). Then emit a `tool_result` step with `Name = span.name`, `Output` from `span_data.output` (stringify if not already string).
  - `type: "agent"`: extract `span_data.model` into metadata map. Recurse into `children`.
  - `type: "generation"`: emit an `assistant` step with content from `span_data.output` if present, otherwise skip.
  - Other types: skip silently.
- Handle malformed spans defensively: skip nil entries in `children` arrays, skip spans where `span_data` is missing or not an object, cap recursion depth at 100 levels (return error if exceeded).
- Return metadata map with `model` if found, `trace_id` always.
- Create testdata fixture `agents_sdk_trace.json` with nested agent > 2 function spans + 1 generation span.
- Tests: empty trace (no spans), single function span, nested agent with multiple children, metadata extraction (model + trace_id), unknown span types ignored.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -run TestAgentsSdk -v`

**Dependencies:** none

### Task 3: LangChain Callback Adapter

**Files:** `internal/adapter/langchain.go` (new), `internal/adapter/langchain_test.go` (new), `testdata/langchain_callbacks.jsonl` (new)

**Do:**
- Create `LangChainAdapter` struct implementing `Adapter` interface.
- Parse JSONL input. Each line is a JSON object with `type` (string), `run_id` (string), optional `parent_run_id` (string), `name` (string), and type-specific fields.
- Process events in line order (chronological):
  - `on_tool_start`: emit `tool_call` step. `Name` from `name` field. `Args` from `inputs` field (map). If `inputs` is not a map, wrap as `{"input": value}`.
  - `on_tool_end`: emit `tool_result` step. `Name` from `name` field. `Output` from `outputs.output` if present, otherwise JSON-stringify entire `outputs` map.
  - `on_llm_end`: emit `assistant` step. `Content` from `outputs.generations[0][0].text` if present, otherwise from `outputs.output` if present, otherwise skip.
  - `on_chain_start`: extract `name` into metadata as `agent_name` (only for the first chain_start). Skip as step.
  - `on_chain_end`, `on_llm_start`, and unknown types: skip.
- Return metadata with `agent_name` if found.
- Create testdata fixture with: chain_start, tool_start, tool_end, tool_start, tool_end, llm_end, chain_end.
- Tests: empty input, single tool round-trip (start + end), multiple tools, llm_end with generation text, unknown events ignored, missing optional fields handled gracefully.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -run TestLangChain -v`

**Dependencies:** none

### Task 4: Generic JSONL Adapter and Config Extension

**Files:** `internal/adapter/generic.go` (new), `internal/adapter/generic_test.go` (new), `internal/config/config.go` (modify), `internal/config/config_test.go` (modify), `testdata/generic_trace.jsonl` (new)

**Do:**
- Extend `Config` struct in `config.go`:
  - Add `Adapter AdapterConfig` field with yaml tag `adapter`.
  - `AdapterConfig` has `Generic GenericAdapterConfig` field.
  - Add `Cluster ClusterConfig` field with yaml tag `cluster`.
  - Update `DefaultConfig()` with zero-value defaults: empty strings/maps for generic adapter, `ClusterConfig{Epsilon: 0, MinPoints: 2}`.
  - Update `Load()` to merge `adapter` and `cluster` sections. Use the same partial-parse pattern: add `Adapter` and `Cluster` fields to `fileConfig` as `map[string]interface{}`. For adapter.generic, use `yaml.Marshal` then `yaml.Unmarshal` to convert `map[string]interface{}` to `GenericAdapterConfig` struct (avoids manual type-assertion loops for nested role_map). For cluster, merge epsilon and min_points.
- Create `GenericAdapter` struct with a `GenericAdapterConfig` field. Constructor: `NewGenericAdapter(cfg GenericAdapterConfig) *GenericAdapter`.
- Implement `Parse()`:
  - Split input on newlines, parse each as JSON object.
  - For each line: resolve `role_field` to get raw role string, map through `role_map` (if no mapping exists, use raw value -- must be one of user/assistant/tool_call/tool_result, skip otherwise).
  - For `tool_call` role: resolve `tool_name_field` and `tool_args_field` via dot-notation paths.
  - For `tool_result` role: resolve `tool_name_field` and `tool_output_field`.
  - For `user`/`assistant`: resolve `content_field`.
- Implement `resolveFieldPath(obj map[string]interface{}, path string) (interface{}, bool)`: split path on `.`, walk nested maps, return value and ok. If any intermediate level is not a map or key is missing, return nil, false.
- Create testdata fixture with 4 lines: user message, tool_call, tool_result, assistant reply using custom field names.
- Tests: `TestResolveFieldPath` (flat field, nested field, missing field, non-map intermediate), `TestGenericParse` (full round-trip with role mapping), `TestGenericMissingFields` (graceful skip on missing fields), config load with adapter.generic section.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -run TestGeneric -v && go test ./internal/config/ -v`

**Dependencies:** none

### Task 5: Detection and Registry Updates

**Files:** `internal/adapter/adapter.go` (modify), `internal/adapter/detect.go` (modify), `internal/adapter/detect_test.go` (modify), `cmd/record.go` (modify)

**Do:**
- Register new adapters in `adapter.go` init():
  ```go
  registry["agents_sdk"] = &AgentsSdkAdapter{}
  registry["langchain"] = &LangChainAdapter{}
  ```
  Do NOT register GenericAdapter (it requires config). Add `GetGeneric(cfg config.GenericAdapterConfig) Adapter` function that returns `NewGenericAdapter(cfg)`.
- Update `detect.go` detection logic:
  - In `detectJSONObject()`: BEFORE the existing `choices`/`messages` checks, check for `trace_id` AND `spans` keys. If both present, return `&AgentsSdkAdapter{}`.
  - In the JSONL fallback paths (both in `case '{'` and the default case): before returning `&ClaudeAdapter{}`, peek at the first JSONL line. Parse it as JSON, check for both `run_id` key AND `type` key where the type value matches one of: `on_tool_start`, `on_tool_end`, `on_llm_start`, `on_llm_end`, `on_chain_start`, `on_chain_end`, `on_retriever_start`, `on_retriever_end`. If both present, return `&LangChainAdapter{}`. Otherwise fall through to Claude.
  - Generic adapter is NEVER auto-detected.
- Update `cmd/record.go`:
  - In `adapterSourceName()`, add type switch cases: `*AgentsSdkAdapter` -> "agents_sdk", `*LangChainAdapter` -> "langchain", `*GenericAdapter` -> "generic".
  - In `runRecord()`: when `recordAdapterName == "generic"`, load config from cwd via `config.Load()`. Before constructing the adapter, validate that `cfg.Adapter.Generic.RoleField` is non-empty. If empty, return error: `generic adapter requires adapter.generic.role_field in .agentdiff.yaml`. Construct adapter via `adapter.GetGeneric(cfg.Adapter.Generic)`, set `sourceName = "generic"`. Skip Detect path.
- Add detection tests in `detect_test.go`:
  - Agents SDK JSON detected correctly (input with trace_id + spans).
  - LangChain JSONL detected correctly (lines with run_id + on_tool_start).
  - Claude JSONL still detected correctly (not misrouted to LangChain).
  - OpenAI array/object formats still detected correctly.
  - JSONL without run_id/on_ prefix falls through to Claude.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -run TestDetect -v && go build ./...`

**Dependencies:** Task 2, Task 3, Task 4

### Task 6: DBSCAN Algorithm

**Files:** `internal/cluster/dbscan.go` (new), `internal/cluster/dbscan_test.go` (new)

**Do:**
- Create `internal/cluster/` package.
- Implement `DistanceMatrix(sequences [][]string, distFn func(a, b []string) int) [][]int`. Build symmetric N x N matrix. `distMatrix[i][j] = distFn(sequences[i], sequences[j])`. Diagonal is 0.
- Implement `DBSCAN(distMatrix [][]int, epsilon float64, minPts int) DBSCANResult`. Standard DBSCAN algorithm:
  1. Initialize all points as unvisited.
  2. For each unvisited point p: mark as visited. Find neighbors (all points q where `distMatrix[p][q] <= epsilon`).
  3. If len(neighbors) >= minPts: create new cluster, expand cluster (add neighbors, recursively check their neighborhoods).
  4. Otherwise: mark as noise (may later be added to a cluster as a border point).
  5. After clustering, for each cluster compute exemplar: the member with minimum average distance to other members in the same cluster.
- Return `DBSCANResult` with clusters (each has ID starting at 0, Members as sorted indices, Exemplar index), Noise (sorted indices), Epsilon, MinPts.
- Tests:
  - Two clear clusters with gap: sequences `[["a","b"], ["a","b","c"], ["x","y","z"], ["x","y"]]` with epsilon=2, should produce 2 clusters.
  - All same: all identical sequences, epsilon=0, single cluster.
  - All noise: epsilon=0, all different sequences, minPts=2, all noise.
  - Empty input: returns empty result.
  - Single point: with minPts=1, single cluster; with minPts=2, noise.
  - Exemplar correctness: verify exemplar is the most central member.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/cluster/ -run TestDBSCAN -v`

**Dependencies:** none

### Task 7: Epsilon Auto-Selection

**Files:** `internal/cluster/epsilon.go` (new), `internal/cluster/epsilon_test.go` (new)

**Do:**
- Implement `AutoEpsilon(distMatrix [][]int, minPts int) (float64, error)`.
  1. For each point i, collect distances to all other points, sort ascending, take the k-th value (k = minPts). This is the k-distance for point i.
  2. Sort all k-distances ascending.
  3. Compute discrete second derivative: `d2[i] = kd[i+1] - 2*kd[i] + kd[i-1]` for `i in [1, len-2]`.
  4. Find index of maximum d2 value. The k-distance at that index is epsilon.
- Edge cases:
  - Fewer than 3 points: return error `"need at least 3 points for auto epsilon"`.
  - All k-distances equal: return that distance (valid, every point is equidistant).
  - Second derivative all zeros (linear): return median k-distance.
  - k-distance at elbow is 0: return 1.0 as minimum epsilon (avoid degenerate 0-epsilon).
- Tests:
  - Clear elbow: two tight clusters far apart. Epsilon should be between intra-cluster and inter-cluster distance.
  - Degenerate all-same: all distances equal, returns that distance.
  - Linear gradient: returns median.
  - Too few points: returns error.
  - Verify auto-epsilon produces reasonable clustering when fed back to DBSCAN.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/cluster/ -run TestAutoEpsilon -v`

**Dependencies:** Task 6

### Task 8: Snapshot Clustering

**Files:** `internal/cluster/cluster.go` (new), `internal/cluster/cluster_test.go` (new)

**Do:**
- Implement `ClusterBaseline(baseline snapshot.Baseline, epsilon float64, minPts int) (StrategyReport, error)`.
  0. If baseline has fewer than max(3, minPts+1) snapshots, return error: `baseline needs at least N snapshots for clustering (has M)`.
  1. Extract tool name sequences from each snapshot: `seqs[i] = diff.ExtractToolNames(baseline.Snapshots[i].Steps)`.
  2. Build distance matrix: `DistanceMatrix(seqs, diff.Levenshtein)`.
  3. If epsilon == 0, call `AutoEpsilon(distMatrix, minPts)`. If auto-epsilon fails, return error.
  4. Run `DBSCAN(distMatrix, epsilon, minPts)`.
  5. Build `StrategyReport`: map cluster members to snapshot names, get exemplar snapshot's tool sequence, collect noise snapshot names.
- Implement `CompareToCluster(baseline snapshot.Baseline, snap snapshot.Snapshot, epsilon float64, minPts int) (MatchResult, error)`.
  1. Call `ClusterBaseline()` to get strategies.
  2. Extract new snapshot's tool sequence.
  3. For each strategy, compute Levenshtein distance from new snapshot to the strategy's exemplar sequence.
  4. For each strategy, compute max intra-cluster distance (max Levenshtein distance from any member to the exemplar). Find minimum distance from new snapshot to any exemplar. If min distance to closest exemplar <= that strategy's max intra-cluster distance, return `MatchResult{Matched: true, StrategyID: id, Exemplar: name, Distance: dist, MaxIntraClusterDist: maxDist}`. If a strategy has only one member (no intra-cluster distances), fall back to epsilon as the match threshold for that strategy.
  5. If no match, return `MatchResult{Matched: false, Distance: minDist}` with the closest strategy info.
- Tests:
  - Baseline with 2 clear strategies (3 search-summarize runs, 3 direct-answer runs): verify 2 strategies found.
  - Single-strategy baseline: verify 1 strategy, no noise.
  - Snapshot matching a strategy: verify Matched=true, correct StrategyID.
  - Snapshot not matching any strategy: verify Matched=false.
  - Empty baseline: return error.
  - Baseline with 1 snapshot: return error (need at least minPts).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/cluster/ -run "TestClusterBaseline|TestCompareToCluster" -v`

**Dependencies:** Task 1, Task 6, Task 7

### Task 9: Cluster Subcommand and Integration Tests

**Files:** `cmd/cluster.go` (new), `cmd/cluster_test.go` (new), `cmd/integration_test.go` (modify)

**Do:**
- Create `agentdiff cluster` parent command.
- `agentdiff cluster <baseline-name>` subcommand (positional arg):
  1. Load config from cwd.
  2. Load baseline from `BaselineStore`.
  3. Read epsilon from `--epsilon` flag (default 0 = auto, overrides config). Read min-points from `--min-points` flag (default 0 = use config, which defaults to 2).
  4. Call `cluster.ClusterBaseline()`.
  5. Terminal output: header line with baseline name and snapshot count. Table with columns: Strategy, Count, Exemplar, Tool Sequence. Noise section listing unclassified snapshots. Footer with epsilon used.
  6. JSON output (--json flag): marshal `StrategyReport` directly.
- `agentdiff cluster compare <baseline-name> <snapshot>` subcommand:
  1. Load baseline and snapshot (snapshot by name/ID from store).
  2. Call `cluster.CompareToCluster()`.
  3. Terminal output: "Matched strategy N (exemplar: name, distance: D)" or "No matching strategy (closest: name, distance: D)".
  4. JSON output: marshal `MatchResult`.
  5. Exit codes: `cluster compare` exits 0 if matched, exits 1 if no matching strategy found. This enables CI usage: `agentdiff cluster compare main-baseline current-snap || echo 'New strategy detected'`.
- Wire both into `rootCmd` via `init()`.
- Flags: `--epsilon` (float64), `--min-points` (int) on the parent cluster command (inherited by subcommands).
- Add integration tests in `cmd/integration_test.go`:
  - `TestIntegrationV3Cluster`: programmatically create 6 snapshots (3 with tool sequence [search, summarize], 3 with [lookup, answer]). Add all to a baseline. Run cluster, verify 2 strategies found.
  - `TestIntegrationV3ClusterCompare`: create a new snapshot with [search, summarize, cite], run cluster compare, verify it matches the search-summarize strategy.
  - `TestIntegrationV3Record`: record from each new testdata fixture (agents_sdk, langchain, generic), verify snapshots saved with correct source names.
- Run full test suite, confirm all tests pass.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./cmd/ -run "TestCluster|TestIntegrationV3" -v && go build ./...`

**Dependencies:** Task 5, Task 8

## The One Hard Thing

**Automatic epsilon selection for DBSCAN via k-distance elbow detection.**

Agent tool sequences have variable-length edit distances with no guaranteed cluster structure. The elbow method works by: (1) computing each point's distance to its k-th nearest neighbor (k = minPts), (2) sorting these distances ascending, (3) finding the point of maximum curvature (largest discrete second derivative) in the sorted curve.

**Approach:** Discrete second derivative on sorted k-distances. `d2[i] = kd[i+1] - 2*kd[i] + kd[i-1]`. Index of max `d2` gives the elbow. Use `kd[elbow]` as epsilon.

**Fallback:** Three degenerate cases handled explicitly:
1. All k-distances equal: use that distance as epsilon (every point is equidistant).
2. Linear gradient (no bend): use median k-distance.
3. Fewer than 3 points: return error, require manual `--epsilon`.

This is the novel technical contribution. Nobody else clusters agent tool-call sequences with auto-tuned DBSCAN. Jake should be able to explain the k-distance graph and why second-derivative detects the natural cluster boundary.

## Risks

1. **LangChain format fragility (medium).** No standard callback export format. Different callback handlers produce different JSONL schemas. Mitigation: support the most common pattern (`on_tool_start`/`on_tool_end` with `run_id` + `name` + `inputs`/`outputs`), document expected format in README, point users to the generic adapter as escape hatch.

2. **Epsilon auto-selection quality (medium).** Elbow method fails on degenerate distributions (uniform distances, no natural clusters). Mitigation: explicit fallback to median with warning. Manual `--epsilon` flag always available. Document when auto-selection works well (5+ baseline runs, 2+ distinct strategies) vs. when to set manually.

3. **Generic adapter field path parsing (low).** Dot-notation paths like `function.arguments` require recursive JSON map traversal. Nested arrays not supported (only nested objects). Mitigation: simple `strings.Split(path, ".")` + type-assert each level. Clear error on missing or wrong-type fields.

4. **OpenAI Agents SDK format stability (low).** SDK is relatively new, trace format may evolve. Mitigation: adapter is ~100 lines, isolated in one file, easy to update.

5. **Levenshtein export rename (low).** Renaming `levenshtein` to `Levenshtein` changes internal API. All callers are in the same package. Mitigation: find-and-replace, verify with `go build ./...` and full test suite.
