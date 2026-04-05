# AgentDiff v2

**pytest for AI agents.** Snapshot behavior, diff across changes, catch regressions -- now with edit-distance alignment, CI integration, and statistical baselines.

## Problem

AgentDiff v1 compares agent traces positionally (step i to step i), so a single tool insertion shifts all downstream comparisons and produces noisy false regressions. There is no CI integration for automated regression detection in pull requests, and single-run comparisons are inherently noisy for non-deterministic agents. V2 solves all three: accurate step alignment, GitHub Action CI, and multi-run statistical baselines.

## Solution

Three additive features layered onto the existing v1 codebase (3.5K lines, 57 tests, Go 1.24, cobra CLI):

1. **Diagnostics (edit-distance alignment).** Replace positional alignment with Levenshtein-based alignment (edit distance with substitution) in CompareTools and TerminalVerbose. New `Align` function backtraces the full DP matrix to produce step-pair mappings with match/substitute/insert/delete operations. Retry-aware heuristic collapses consecutive same-tool-name-AND-same-args calls before alignment. Adds `Diagnostics` field to `DiffResult` (additive -- no breaking JSON change).

2. **GitHub Action CI.** New `agentdiff ci` subcommand reads a CI config section from `.agentdiff.yaml`, runs diffs against committed baselines, outputs GitHub-flavored markdown. Separate exit codes for functional regressions vs stylistic drift. `action.yml` wraps the binary for the Actions marketplace.

3. **Multi-Run Statistical Baselines.** New `agentdiff baseline` subcommand family (record/compare/list). Bootstrap confidence intervals in pure Go. Component-aware weighting based on coefficient of variation.

**Key architectural decisions:**
- Zero new external deps. Stats are pure Go (no gonum). Action uses pre-built binary.
- `DiffResult` only gains fields, never loses them. Existing JSON consumers unaffected.
- Existing `levenshtein` stays as-is. New `Align` function uses full DP matrix (not 2-row) to enable backtrace. Algorithm is Levenshtein edit distance (match/substitute/insert/delete), NOT LCS. This produces a single deterministic optimal alignment with consistent substitution handling.
- `CollapseRetries` returns `remapA []int, remapB []int` arrays mapping collapsed indices back to original step indices, so terminal reports and arg comparison can reference the original snapshot steps.
- Baselines are gzip-compressed JSON in `.agentdiff/baselines/`.
- Nobody else does step-level behavioral alignment for agent testing. This is the differentiator.

## Scope

**Building:**
- Levenshtein edit-distance alignment with backtrace (step-pair mapping, insertions, deletions, substitutions)
- Argument-aware retry collapse heuristic (consecutive same-tool AND same-args calls)
- Divergence detection (traces sharing <20% alignment marked "diverged")
- Diagnostics in DiffResult JSON output and terminal reports
- `agentdiff ci` subcommand with markdown output and exit code semantics
- `action.yml` for GitHub Actions marketplace
- `agentdiff baseline record/compare/list` subcommands
- Bootstrap CI computation (percentile method, B=10,000)
- Component-aware weighting (CV-based threshold adjustment)

**Not building:**
- Live agent execution from CI (users record traces in their own harness)
- GitHub Check Runs API annotations (defer to v3, adds GitHub API dependency)
- Bimodal strategy detection / clustering (defer, needs real usage data)
- Schema evolution detection (defer)
- Web UI or dashboard
- `agentdiff accept` for updating golden snapshots

**Ship target:** GitHub (jtsilverman/agentdiff), tag v0.2.0

## Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.24 | Existing codebase. Single binary, fast, CI-friendly. |
| CLI | cobra v1.10 | Existing. Proven subcommand pattern. |
| Config | gopkg.in/yaml.v3 | Existing. Extends `.agentdiff.yaml`. |
| Compression | compress/gzip (stdlib) | Baseline storage. No new deps. |
| Stats | math/rand + sort (stdlib) | Bootstrap resampling. No gonum needed. |
| Test | stdlib testing | Existing. No external test deps. |

Zero external dependencies beyond cobra + yaml (unchanged from v1).

## Architecture

### New/Modified Files

```
agentdiff/
  cmd/
    root.go              -- EXISTING (unchanged)
    record.go            -- EXISTING (unchanged)
    diff.go              -- EXISTING (unchanged)
    report.go            -- MODIFY: pass diagnostics to TerminalVerbose
    list.go              -- EXISTING (unchanged)
    ci.go                -- NEW: agentdiff ci subcommand
    ci_test.go           -- NEW
    baseline.go          -- NEW: agentdiff baseline record/compare/list
    baseline_test.go     -- NEW
    integration_test.go  -- MODIFY: add v2 integration tests
  internal/
    adapter/             -- EXISTING (unchanged)
    diff/
      diff.go            -- MODIFY: add Diagnostics field to DiffResult
      align.go           -- NEW: Levenshtein alignment, backtrace, retry collapse
      align_test.go      -- NEW
      tools.go           -- MODIFY: use alignment in CompareTools
      tools_test.go      -- MODIFY: update assertions for alignment-based scoring
      text.go            -- MODIFY: Compare() populates diagnostics
      text_test.go       -- EXISTING (may need assertion updates)
      diff_test.go       -- MODIFY: add diagnostics assertions
    config/
      config.go          -- MODIFY: add CI and Baseline config sections
      config_test.go     -- MODIFY: test new config sections
    report/
      terminal.go        -- MODIFY: alignment-aware TerminalVerbose
      json.go            -- EXISTING (unchanged, DiffResult change flows through)
      markdown.go        -- NEW: GitHub-flavored markdown formatter
      markdown_test.go   -- NEW
      report_test.go     -- MODIFY: update for new TerminalVerbose format
    snapshot/
      snapshot.go        -- EXISTING (unchanged)
      store.go           -- EXISTING (unchanged)
      baseline.go        -- NEW: baseline collection storage
      baseline_test.go   -- NEW
    stats/               -- NEW package
      bootstrap.go       -- Bootstrap CI computation
      bootstrap_test.go
      weights.go         -- Component-aware CV weighting
      weights_test.go
  action.yml             -- NEW: GitHub Action definition
  main.go                -- EXISTING (unchanged)
  go.mod                 -- EXISTING (unchanged)
```

### New Data Types

```go
// internal/diff/align.go

type AlignOp int
const (
    AlignMatch  AlignOp = iota // same tool at both positions
    AlignSubst                  // different tool
    AlignInsert                 // tool in B only (not in A)
    AlignDelete                 // tool in A only (not in B)
)

type AlignedPair struct {
    IndexA int     `json:"index_a"` // -1 if insert
    IndexB int     `json:"index_b"` // -1 if delete
    Op     AlignOp `json:"op"`
    ToolA  string  `json:"tool_a,omitempty"`
    ToolB  string  `json:"tool_b,omitempty"`
}

type AlignResult struct {
    Pairs           []AlignedPair
    FirstDivergence int          // index into Pairs, -1 if identical
    Diverged        bool         // true if <20% match ratio
    RetryGroups     []RetryGroup
    RemapA          []int        // remapA[collapsedIdx] = original step index
    RemapB          []int        // remapB[collapsedIdx] = original step index
}

type RetryGroup struct {
    ToolName string `json:"tool_name"`
    CountA   int    `json:"count_a"`
    CountB   int    `json:"count_b"`
    StartA   int    `json:"start_a"` // original index before collapse
    StartB   int    `json:"start_b"`
}
```

```go
// internal/diff/diff.go (new field on DiffResult)

type Diagnostics struct {
    Alignment       []AlignedPair `json:"alignment"`
    FirstDivergence int           `json:"first_divergence"`
    Diverged        bool          `json:"diverged"`
    RetryGroups     []RetryGroup  `json:"retry_groups,omitempty"`
    RemapA          []int         `json:"remap_a"` // collapsed index -> original step index (A)
    RemapB          []int         `json:"remap_b"` // collapsed index -> original step index (B)
}

// DiffResult gains one field:
//   Diagnostics *Diagnostics `json:"diagnostics,omitempty"`
```

```go
// internal/snapshot/baseline.go

type Baseline struct {
    Name      string     `json:"name"`
    Snapshots []Snapshot `json:"snapshots"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

```go
// internal/stats/bootstrap.go

type BootstrapResult struct {
    Mean       float64 `json:"mean"`
    Lower      float64 `json:"lower"`      // 2.5th percentile
    Upper      float64 `json:"upper"`      // 97.5th percentile
    SampleSize int     `json:"sample_size"`
}

type BaselineStats struct {
    ToolScore BootstrapResult `json:"tool_score"`
    TextScore BootstrapResult `json:"text_score"`
    StepDelta BootstrapResult `json:"step_delta"`
}
```

```go
// internal/stats/weights.go

type ComponentWeight struct {
    Name      string  `json:"name"`
    CV        float64 `json:"cv"`
    Weight    float64 `json:"weight"`
    Threshold float64 `json:"threshold"`
}
```

### Config Extension (.agentdiff.yaml)

```yaml
# Existing (unchanged)
thresholds:
  tool_score: 0.3
  text_score: 0.5
  step_delta: 5

# New sections
ci:
  baseline_path: .agentdiff/baselines/main.json.gz
  fail_on_style_drift: false

baseline:
  runs: 5
  confidence: 0.95
```

```go
// config.go additions
type CIConfig struct {
    BaselinePath     string `yaml:"baseline_path"`
    FailOnStyleDrift bool   `yaml:"fail_on_style_drift"`
}

type BaselineConfig struct {
    Runs       int     `yaml:"runs"`
    Confidence float64 `yaml:"confidence"`
}
```

### Exit Code Semantics (CI)

| Code | Meaning |
|------|---------|
| 0 | Pass (all metrics within thresholds) |
| 1 | Functional regression (tool score or step delta exceeds threshold) |
| 2 | Stylistic drift only (text score exceeds threshold, tools OK) |

### CLI Contract (new commands)

```
agentdiff ci [--baseline PATH] [--output FILE]
  # Compare current snapshots against committed baseline
  # Output markdown report (stdout or file)
  # Exit 0/1/2 per exit code semantics

agentdiff baseline record <name> <snapshot>
  # Add snapshot to named baseline (creates if new)
  # Exit 0

agentdiff baseline compare <name> <snapshot> [--json]
  # Compare snapshot against baseline using bootstrap CI
  # Exit 0 if within bounds, Exit 1 if regression

agentdiff baseline list [--json]
  # List baselines with snapshot count and date range
  # Exit 0
```

## Tasks

### Task 1: Levenshtein Alignment Core

**Files:** `internal/diff/align.go` (new), `internal/diff/align_test.go` (new)

**Do:**
- Implement `Align(seqA, seqB []string) AlignResult`. This is **Levenshtein edit-distance alignment** (not LCS). Build full `(len(seqA)+1) x (len(seqB)+1)` DP matrix (not 2-row -- need backtrace). Cost: match=0, substitute=1, insert=1, delete=1. Backtrace from `[la][lb]` to `[0][0]`, producing `[]AlignedPair` in forward order. On ties during backtrace, prefer: match > substitute > delete > insert (deterministic tie-breaking ensures consistent output for the same input).
- Implement `CollapseRetries(steps []snapshot.Step) (collapsed []snapshot.Step, remap []int, groups []RetryGroup)`. A retry is defined as 2+ consecutive steps where: (a) both have non-nil ToolCall, (b) same tool name, AND (c) same arguments (JSON-canonicalized comparison: marshal Args to sorted-key JSON, compare strings). This avoids reflect.DeepEqual pitfalls where encoding/json decodes numbers as float64 but other paths might use int. Calls with different arguments (e.g., `Read("a.go")` then `Read("b.go")`) are NOT retries and must NOT be collapsed. Collapse to single representative (keep first call). Return `remap []int` where `remap[collapsedIdx] = originalStepIdx`, enabling callers to map alignment indices back to original snapshot positions. Return groups with original start indices and counts.
- Implement divergence detection: if `matchCount / max(len(seqA), len(seqB)) < 0.2`, set `Diverged = true`.
- Compute `FirstDivergence`: index of first `AlignedPair` where `Op != AlignMatch`. Set to -1 if all match.
- Respect `maxToolCalls = 1000` cap (overridable via `--max-steps` flag on diff/report/ci commands). For sequences longer than the cap, truncate to the last N before alignment. **Remap indices remain absolute** (reference the original full snapshot, not the truncated suffix). **Emit a warning to stderr** when truncation occurs: "Warning: truncated N steps to 1000 for alignment. Use --max-steps to increase."
- Tests: identical sequences (all AlignMatch, FirstDivergence=-1), single insertion in middle, single deletion, substitution, complete divergence (Diverged=true), retry collapse (3 consecutive Read calls with SAME args -> 1, group records count=3), retry non-collapse (3 consecutive Read calls with DIFFERENT args -> 3, no group), remap array correctness (collapsed index maps to correct original index), empty sequences (both empty, one empty), mixed operations producing correct backtrace order, tie-breaking determinism (["A","B"] vs ["B","A"] produces consistent output).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/diff/ -run TestAlign -v`

**Dependencies:** None

### Task 2: Integrate Alignment into CompareTools and DiffResult

**Files:** `internal/diff/tools.go` (modify), `internal/diff/diff.go` (modify), `internal/diff/text.go` (modify), `internal/diff/tools_test.go` (modify), `internal/diff/diff_test.go` (modify)

**Do:**
- Add `Diagnostics` struct to `diff.go`. Add `Diagnostics *Diagnostics` field to `DiffResult` with tag `json:"diagnostics,omitempty"`. This is additive -- existing JSON output is unchanged when Diagnostics is nil (omitempty).
- Create `CompareToolsWithDiagnostics(a, b []snapshot.Step) (ToolDiffResult, Diagnostics)` in `tools.go`. This function: (1) calls `CollapseRetries` on both step sequences, getting collapsed steps + remap arrays, (2) extracts tool names from collapsed steps, (3) calls `Align` on collapsed tool name sequences, (4) derives `editDist` from alignment (count of non-AlignMatch ops), (5) computes `argScore` using alignment pairs — for AlignMatch pairs, use `remapA[pair.IndexA]` and `remapB[pair.IndexB]` to index into the ORIGINAL step slices for arg comparison via `jaccardArgs`, (6) computes added/removed/reordered from alignment, (7) populates `Diagnostics` with alignment, remap arrays, retry groups, first divergence, and diverged flag, (8) returns both ToolDiffResult and Diagnostics.
- Update `CompareTools` to call `CompareToolsWithDiagnostics` and discard the diagnostics (keeps existing API unchanged).
- Update `Compare()` in `text.go` to call `CompareToolsWithDiagnostics` instead of `CompareTools`, and populate `result.Diagnostics`.
- Existing `levenshtein` function stays untouched.
- Update test assertions: alignment-based arg scoring may shift some `ToolDiffResult.Score` values slightly. These shifts are expected and more accurate. Verify all existing tests pass (update assertions where values shift). The total test count should not decrease.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./... -count=1`

**Dependencies:** Task 1

### Task 3: Alignment-Aware Terminal Reports

**Files:** `internal/report/terminal.go` (modify), `cmd/report.go` (modify), `internal/report/report_test.go` (modify)

**Do:**
- Change `TerminalVerbose` signature to: `TerminalVerbose(result diff.DiffResult, snapA, snapB snapshot.Snapshot, w io.Writer) error`. It already receives the DiffResult which now contains Diagnostics.
- When `result.Diagnostics != nil`, replace positional iteration with alignment-aware iteration. Use `Diagnostics.RemapA` and `RemapB` to map alignment indices back to original step indices in `snapA.Steps` and `snapB.Steps`. For each `AlignedPair`:
  - `AlignMatch`: use `remapA[pair.IndexA]` to get original step from snapA, `remapB[pair.IndexB]` for snapB. Show step comparison with tool name, compare args with `printArgDiff`. Display original step numbers (not collapsed indices) for user clarity.
  - `AlignSubst`: show tool name change in red/green: `- [A step N] toolA` / `+ [B step M] toolB` using remapped original indices.
  - `AlignInsert`: show `+ [B only] step M: toolName` in green (M = remapB[pair.IndexB]).
  - `AlignDelete`: show `- [A only] step N: toolName` in red (N = remapA[pair.IndexA]).
- When `result.Diagnostics == nil`, fall back to existing positional logic (backward compat).
- Show retry groups: "Retries: Read x3 (A) vs Read x2 (B)" for each group.
- If `Diagnostics.Diverged`, print warning: "WARNING: Traces diverged (different strategies). Alignment unreliable."
- Print "First divergence at aligned step N" when `FirstDivergence >= 0`.
- Update `cmd/report.go`: no signature change needed since DiffResult already flows through.
- Update `report_test.go` to test alignment-aware output: test with diagnostics present (verify insert/delete markers), test with diagnostics nil (verify fallback to positional).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/report/ -v && go test ./cmd/ -v`

**Dependencies:** Task 2

### Task 4: CI Config and Markdown Reporter

**Files:** `internal/config/config.go` (modify), `internal/config/config_test.go` (modify), `internal/report/markdown.go` (new), `internal/report/markdown_test.go` (new)

**Do:**
- Add `CIConfig` and `BaselineConfig` structs to `config.go`. Extend `Config` with `CI CIConfig` and `Baseline BaselineConfig` fields (yaml tags: `ci` and `baseline`). Update `DefaultConfig()`: `CI.BaselinePath = ""`, `CI.FailOnStyleDrift = false`, `Baseline.Runs = 5`, `Baseline.Confidence = 0.95`. Update `Load()` to parse new sections using the same partial-parse approach (fileConfig map). **Validate `Baseline.Confidence` is in (0.0, 1.0) — return error from `Load()` if not.**
- Create `markdown.go` with two functions:
  - `Markdown(result diff.DiffResult, cfg config.Config, w io.Writer) error`: single diff as markdown. Format: `## AgentDiff Report` header, summary table (`| Metric | Score | Threshold | Status |` with checkmark/X/warning indicators), collapsible `<details>` section with per-step alignment breakdown (if diagnostics present), diagnostics summary (first divergence, retry groups), verdict line.
  - `CIMarkdown(results []diff.DiffResult, cfg config.Config, w io.Writer) error`: multiple diffs, iterates and renders each with a `### <snapshot-name>` sub-header.
- Tests: verify markdown contains expected table headers, `<details>` tags, correct status indicators for pass/regression/drift, handles empty diagnostics gracefully.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/report/ -run TestMarkdown -v && go test ./internal/config/ -v`

**Dependencies:** Task 2

### Task 5: CI Subcommand and GitHub Action

**Files:** `cmd/ci.go` (new), `cmd/ci_test.go` (new), `action.yml` (new)

**Do:**
- Implement `agentdiff ci` cobra subcommand. Behavior:
  1. Load config from `.agentdiff.yaml` in cwd
  2. Load baseline from `ci.baseline_path` (error with clear message if not configured or file missing)
  3. Load current snapshots from `.agentdiff/snapshots/` via store.List()
  4. For each current snapshot, find the baseline snapshot with the same `Name` field. A baseline file contains multiple snapshots; match by name, use the **latest** (most recent Timestamp) if multiple share a name. Skip current snapshots with no baseline match (print warning).
  5. Run `diff.Compare` for each matched pair, collect results
  6. **Always** render markdown via `report.CIMarkdown` to stdout or `--output` file — this must happen BEFORE any non-zero exit, so the report is always available.
  7. Exit code logic: if any result has tool regression or step regression -> exit 1. If any result has text-only regression and `fail_on_style_drift: false` -> exit 2. Otherwise exit 0.
- Flags: `--output FILE` (write to file instead of stdout), `--baseline PATH` (override `ci.baseline_path`).
- Define `ErrStyleDrift` error type for exit code 2. Update `cmd/root.go` Execute() to handle exit code 2: `if err == ErrStyleDrift { os.Exit(2) }` (checked before the generic error handler).
- Create `action.yml` at repo root:
  - name: "AgentDiff CI"
  - inputs: `baseline_path`, `fail_on_style_drift`, `threshold_tool`, `threshold_text`, `threshold_steps`
  - runs using composite steps: (1) `actions/setup-go@v5`, (2) `go install github.com/jtsilverman/agentdiff@latest`, (3) `echo "$GOPATH/bin" >> $GITHUB_PATH` to ensure binary is on PATH, (4) write inputs to `.agentdiff.yaml` if not present, (5) `agentdiff ci --output report.md || echo "exit_code=$?" >> $GITHUB_ENV` (capture exit code, don't fail yet), (6) post `report.md` as PR comment via `actions/github-script`, (7) final step: `if [ "$exit_code" = "1" ]; then exit 1; fi` — exit 1 only for functional regressions. Exit 2 (style drift) does NOT fail the action when `fail_on_style_drift: false`.
- Tests: create temp dirs with mock baseline files and snapshots. Test pass scenario (exit 0), functional regression (exit 1), stylistic drift only (exit 2), missing baseline path (error).

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./cmd/ -run TestCI -v`

**Dependencies:** Task 4

### Task 6: Baseline Storage

**Files:** `internal/snapshot/baseline.go` (new), `internal/snapshot/baseline_test.go` (new)

**Do:**
- Define `Baseline` struct: `Name string`, `Snapshots []Snapshot`, `CreatedAt time.Time`, `UpdatedAt time.Time`.
- Implement `BaselineStore` struct with `dir` field (`.agentdiff/baselines/`).
- `NewBaselineStore(baseDir string) *BaselineStore` -- analogous to `NewStore`.
- `Save(b Baseline) error` -- JSON marshal, gzip compress via `compress/gzip`, write to `<dir>/<name>.json.gz`. Create dir if needed.
- `Load(name string) (Baseline, error)` -- open file, gzip decompress, JSON unmarshal. Clear error if file not found.
- `List() ([]Baseline, error)` -- scan dir for `*.json.gz`, load each, return sorted by UpdatedAt.
- `AddSnapshot(name string, snap Snapshot) error` -- load existing baseline (or create new if not found), append snapshot, update `UpdatedAt`, save. Set `CreatedAt` only on first create.
- Tests: round-trip save/load verifies data integrity, compression produces smaller output than raw JSON, AddSnapshot creates new baseline then appends, List returns multiple baselines sorted, Load returns clear error for missing name.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/snapshot/ -run TestBaseline -v`

**Dependencies:** None

### Task 7: Bootstrap Statistics

**Files:** `internal/stats/bootstrap.go` (new), `internal/stats/bootstrap_test.go` (new), `internal/stats/weights.go` (new), `internal/stats/weights_test.go` (new)

**Do:**
- Create `internal/stats/` package.
- Implement `Bootstrap(samples []float64, confidence float64, B int, seed int64) (BootstrapResult, error)`. **Validate confidence is in (0.0, 1.0) — return error if not.** Use `math/rand.NewSource(seed)` for reproducibility. Resample with replacement B times, compute mean of each resample, sort ascending, extract percentile bounds. For 95% CI with B=10,000: lower = sorted[250], upper = sorted[9750]. Return mean of original samples as `Mean`.
- Handle edge cases: empty samples returns zero result (no error), single sample returns that value for mean/lower/upper (no error).
- Implement `ComputeBaselineStats(diffs []diff.DiffResult, confidence float64) BaselineStats`. Collect tool scores, text scores, step deltas from the diff results. Run `Bootstrap` on each with B=10,000 and seed derived from len(diffs).
- Implement `ComputeWeights(stats BaselineStats, thresholds config.Thresholds) []ComponentWeight`. For each component: CV = std/mean. Low CV (< 0.1) -> tighten threshold by 20% (multiply by 0.8). High CV (> 0.5) -> relax threshold by 30% (multiply by 1.3). Normal -> keep default. Normalize weights so they sum to 1.0.
- Implement `IsRegression(current diff.DiffResult, stats BaselineStats, weights []ComponentWeight) (bool, string)`. Check each component: if current score > component's adjusted threshold AND current score > upper CI bound, it's a regression. Return true + human-readable reason. **The reason string must include both the configured threshold and the effective (CV-adjusted) threshold** so users understand why a score below their configured threshold can still trigger a regression (e.g., "tool_score 0.18 > effective threshold 0.16 (configured 0.20, tightened due to low variance CV=0.05)").
- Tests: all-same samples produce CI width near 0 and low CV, high-variance samples produce wide CI, bootstrap CI contains sample mean (statistical test: run 100 times, verify containment > 90%), weight adjustment for low/high CV, IsRegression with known inputs, **confidence validation (1.5 returns error, -0.3 returns error, 0.95 succeeds)**.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/stats/ -v`

**Dependencies:** None

### Task 8: Baseline Subcommands

**Files:** `cmd/baseline.go` (new), `cmd/baseline_test.go` (new)

**Do:**
- Implement `agentdiff baseline` parent command with three subcommands:
- `agentdiff baseline record <name> <snapshot>`:
  - Load snapshot from store by name/ID
  - Call `baselineStore.AddSnapshot(name, snapshot)`
  - Print confirmation: "Added <snapshot-name> to baseline <name> (N snapshots total)"
- `agentdiff baseline compare <name> <snapshot>`:
  - Load baseline and current snapshot
  - For each baseline snapshot, run `diff.Compare` against current snapshot
  - Compute `stats.ComputeBaselineStats` and `stats.ComputeWeights`
  - Run `stats.IsRegression`
  - Print terminal report: per-component CI bounds, current score, pass/fail
  - Respect `--json` global flag
  - Exit 1 if regression
- `agentdiff baseline list`:
  - List all baselines: name, snapshot count, date range (created -> updated)
  - Respect `--json` global flag
- Tests: record creates new baseline (verify file exists), record appends (verify count), compare detects regression (mock high-scoring diff), compare passes (mock low-scoring diff), list shows correct info. Use temp dirs.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./cmd/ -run TestBaseline -v`

**Dependencies:** Task 6, Task 7

### Task 9: Integration Tests and Full Suite Validation

**Files:** `cmd/integration_test.go` (modify)

**Do:**
- Add integration tests exercising v2 features end-to-end:
  - **Alignment test:** Record two snapshots from testdata where B has one extra tool call inserted in the middle. Run `agentdiff diff --json`, parse JSON output, verify `diagnostics.alignment` contains an AlignInsert entry and `diagnostics.first_divergence >= 0`.
  - **Retry collapse test:** Record two snapshots where both have retry sequences (3 consecutive Read calls). Verify `diagnostics.retry_groups` is populated.
  - **CI test:** Set up `.agentdiff.yaml` with `ci.baseline_path`, create a baseline file and current snapshots. Run `agentdiff ci --output report.md`. Verify report.md contains markdown table. Verify exit code 0 for passing case.
  - **Baseline round-trip test:** Run `agentdiff baseline record test-bl snap-a`, then `agentdiff baseline record test-bl snap-b`. Run `agentdiff baseline compare test-bl snap-a --json`. Verify JSON output contains bootstrap CI fields.
- Create any needed testdata fixtures inline in tests (use `store.Save` to create snapshots programmatically).
- Run full suite, confirm all tests pass and total count exceeds 57.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./... -count=1 -v 2>&1 | tail -5`

**Dependencies:** Task 3, Task 5, Task 8

## The One Hard Thing

**Levenshtein edit-distance alignment with argument-aware retry heuristics** (Task 1).

The challenge: standard edit distance operates on flat string sequences, but agent steps are complex structs with arguments. The alignment must:
1. Build and backtrace a full DP matrix (O(n*m) space vs the existing 2-row optimization) with deterministic tie-breaking (match > substitute > delete > insert)
2. Preprocess retry sequences with argument-aware matching (same name AND same args = retry, same name with different args = distinct operation) and output remap arrays so post-alignment indices map back to original step positions
3. Detect when traces are so divergent that forcing an alignment is misleading

**Approach:** Align on tool call names only (strings), reducing to classic Levenshtein on `[]string`. Retry collapse is a preprocessing step that outputs collapsed sequences, remap arrays (`[]int`), and retry groups. After alignment on collapsed sequences, use remap arrays to index into original step slices for arg comparison and terminal reporting. Divergence threshold (20% match ratio) prevents forcing meaningless alignments on completely different strategies.

**Fallback:** If the full DP matrix is too memory-heavy for 1000x1000 sequences (~8MB, should be fine), implement banded alignment: only compute within k diagonals of the main diagonal. This caps memory at O(n*k) where k = expected max edit distance.

## Risks

**Alignment changes existing test assertions (medium).** Switching from positional to LCS-based arg scoring will shift some `ToolDiffResult.Score` values. The shifts should be small and more accurate. Mitigation: Task 2 explicitly re-validates all 57 tests and updates assertions where needed.

**Bootstrap reproducibility (medium).** Random resampling produces different results across runs without fixed seed. Mitigation: seed derived from input data length. Tests use explicit seed. Document that results are deterministic for same input.

**Memory for large traces (low).** 1000x1000 DP matrix = ~8MB. The existing `maxToolCalls = 1000` cap already enforces this bound. No mitigation needed.

**CI exit code semantics in GitHub Actions (medium).** Non-zero exit codes fail composite action steps. Mitigation: action.yml captures the exit code before it fails the step, posts the markdown report unconditionally, then only fails the job for exit code 1 (functional regression). Exit code 2 (style drift) is surfaced in the PR comment but does not fail the action when `fail_on_style_drift: false`.

**Truncation discards early divergence (low).** The maxToolCalls=1000 cap truncates to the last 1000 steps, which could hide root causes occurring early in long traces. Mitigation: emit a stderr warning when truncation occurs, including the number of steps dropped.

**Scope creep into GitHub API (low).** Check Runs API annotations are explicitly deferred. The `action.yml` uses `actions/github-script` for PR comments -- no Go GitHub client needed.
