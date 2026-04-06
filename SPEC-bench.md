# SPEC: AgentDiff Empirical Validation Suite

## Problem

AgentDiff has 219 correctness tests but zero evidence its algorithms actually work on realistic data. The thresholds (tool 0.3, text 0.5, step delta 5) are arbitrary guesses. There is no measured precision/recall for regression detection, no ROC curves for threshold calibration, and no clustering quality metrics. Without empirical validation, every threshold recommendation is a liability.

## Solution

A synthetic benchmark suite using mutation testing methodology. Generate baseline traces, apply controlled mutations (known regressions), and measure whether AgentDiff's algorithms correctly distinguish regressions from normal variance.

Three evaluation dimensions:
1. **Regression detection** -- precision/recall on labeled pairs (mutation = regression, normal variance = not regression)
2. **Threshold calibration** -- sweep thresholds across a range, produce ROC data, identify optimal operating points via F1 maximization
3. **Clustering quality** -- Adjusted Rand Index (ARI) comparing DBSCAN output to known ground-truth strategy labels

Synthetic traces over real traces because: (a) ground truth labels are free and exact, (b) reproducible via deterministic seed, (c) can control mutation severity independently. Cross-validation (5-fold) prevents threshold overfitting.

All code lives in `internal/bench/` with a CLI command at `cmd/bench.go`. Output is terminal table + JSON file for CI consumption.

## Scope

**Building:**
- Deterministic synthetic trace generator with configurable seed
- 6 mutation operators: tool removal, insertion, reordering, substitution, text drift, combined
- 3 normal-variance generators: arg jitter, text rephrase, step count +/-1-2
- Regression detection evaluator (precision, recall, F1, accuracy)
- Threshold sweep with ROC curve data (AUC computation)
- Clustering quality evaluator (ARI metric)
- 5-fold cross-validation for threshold calibration
- `agentdiff bench` CLI command with `--seed`, `--json`, `--verbose` flags
- JSON results output

**Not building:**
- Visual ROC curve rendering (JSON data only, plot externally)
- Real-world trace collection or evaluation
- Automated threshold update (report only, human decides)
- Benchmark against other tools
- Performance/speed benchmarks

**Ship target:** Merged to main, `agentdiff bench` runnable, results in README.

## Stack

- **Go 1.24** -- matches existing project
- **math/rand (v2)** -- deterministic seeded generation
- **encoding/json** -- results output
- **fmt/tabwriter** -- terminal table output
- **No new dependencies** -- pure stdlib for bench package

## Architecture

### File Structure

```
internal/bench/
  generate.go      -- trace generator + mutation operators
  generate_test.go -- unit tests for generators
  evaluate.go      -- precision/recall/F1/AUC/ARI computation
  evaluate_test.go -- unit tests for metrics
  bench.go         -- orchestrator: runs all evaluations, collects results
  bench_test.go    -- integration test for full bench run
cmd/bench.go       -- cobra command wiring
```

### Data Models

```go
// internal/bench/generate.go

// TraceConfig controls synthetic trace generation.
type TraceConfig struct {
    Seed         int64
    NumTools     int    // tools per trace (default 8)
    NumTraces    int    // baseline traces to generate (default 50)
    NumStrategies int   // distinct behavioral strategies (default 4)
    ToolVocab    []string // pool of tool names
    TextVocab    []string // pool of text phrases
}

// MutationType identifies what kind of regression was injected.
type MutationType string

const (
    MutRemoval      MutationType = "removal"
    MutInsertion    MutationType = "insertion"
    MutReorder      MutationType = "reorder"
    MutSubstitution MutationType = "substitution"
    MutTextDrift    MutationType = "text_drift"
    MutCombined     MutationType = "combined"
)

// LabeledPair is a trace pair with ground truth.
type LabeledPair struct {
    Baseline  snapshot.Snapshot
    Candidate snapshot.Snapshot
    IsRegression bool
    MutationType MutationType // empty if not a regression
}

// LabeledTrace is a trace with a ground-truth strategy label.
type LabeledTrace struct {
    Trace      snapshot.Snapshot
    StrategyID int
}
```

```go
// internal/bench/evaluate.go

// DetectionResult holds regression detection metrics.
type DetectionResult struct {
    Precision float64 `json:"precision"`
    Recall    float64 `json:"recall"`
    F1        float64 `json:"f1"`
    Accuracy  float64 `json:"accuracy"`
    TP        int     `json:"true_positives"`
    FP        int     `json:"false_positives"`
    FN        int     `json:"false_negatives"`
    TN        int     `json:"true_negatives"`
}

// ROCPoint is one point on the ROC curve.
type ROCPoint struct {
    Threshold float64 `json:"threshold"`
    TPR       float64 `json:"tpr"` // recall
    FPR       float64 `json:"fpr"` // false positive rate
    F1        float64 `json:"f1"`
}

// ThresholdResult holds calibration results for one dimension.
type ThresholdResult struct {
    Dimension    string     `json:"dimension"` // "tool", "text", "step"
    ROC          []ROCPoint `json:"roc"`
    AUC          float64    `json:"auc"`
    OptimalPoint ROCPoint   `json:"optimal"` // max F1
}

// ClusterResult holds clustering quality metrics.
type ClusterResult struct {
    ARI            float64 `json:"ari"`
    NumClusters    int     `json:"num_clusters"`
    NumStrategies  int     `json:"num_strategies"` // ground truth
    NoiseCount     int     `json:"noise_count"`
}

// BenchResult is the complete benchmark output.
type BenchResult struct {
    Seed          int64              `json:"seed"`
    Detection     DetectionResult    `json:"detection"`
    ToolThreshold ThresholdResult    `json:"tool_threshold"`
    TextThreshold ThresholdResult    `json:"text_threshold"`
    StepThreshold ThresholdResult    `json:"step_threshold"`
    Clustering    ClusterResult      `json:"clustering"`
    CrossVal      CrossValResult     `json:"cross_validation"`
}

// CrossValResult holds 5-fold cross-validation summary.
type CrossValResult struct {
    Folds       int       `json:"folds"`
    MeanF1      float64   `json:"mean_f1"`
    StdF1       float64   `json:"std_f1"`
    OptimalTool float64   `json:"optimal_tool_threshold"`
    OptimalText float64   `json:"optimal_text_threshold"`
    OptimalStep int       `json:"optimal_step_threshold"`
}
```

### CLI Interface

```
agentdiff bench [flags]

Flags:
  --seed int       Random seed for reproducibility (default 42)
  --json           Output results as JSON
  --verbose        Show per-mutation-type breakdowns
  --output string  Write JSON results to file
```

Exit codes: 0 always (bench is informational, not pass/fail).

## Tasks

### Task 1: Synthetic Trace Generator

**Files:** `internal/bench/generate.go`, `internal/bench/generate_test.go`

**Do:**
- Define `TraceConfig`, `MutationType`, `LabeledPair`, `LabeledTrace` types
- Implement `DefaultConfig()` returning config with seed=42, 8 tools/trace, 50 traces, 4 strategies, 20-item tool vocabulary, 30-phrase text vocabulary
- Implement `GenerateBaseline(cfg TraceConfig, rng *rand.Rand) snapshot.Snapshot` -- creates one trace with N tool call steps interleaved with assistant text steps. Tool names drawn from vocabulary, args are simple key-value maps
- Implement `GenerateStrategyTraces(cfg TraceConfig) []LabeledTrace` -- generates `cfg.NumTraces` traces across `cfg.NumStrategies` strategies. Each strategy has a distinct tool sequence template. Traces within a strategy share the template with minor arg/text variance
- Implement 6 mutation functions, each taking a snapshot and rng, returning a mutated copy:
  - `MutateRemoval(snap, rng)` -- drop 1-3 tool call steps
  - `MutateInsertion(snap, rng, vocab)` -- insert 1-3 new tool call steps at random positions
  - `MutateReorder(snap, rng)` -- swap 2-4 adjacent tool call steps
  - `MutateSubstitution(snap, rng, vocab)` -- replace 1-3 tool names with different ones
  - `MutateTextDrift(snap, rng)` -- shuffle/replace 30-60% of words in assistant text
  - `MutateCombined(snap, rng, vocab)` -- apply 2-3 of the above
- Implement 3 normal-variance functions:
  - `VarianceArgs(snap, rng)` -- change 1-2 arg values, keep tool names identical
  - `VarianceText(snap, rng)` -- rephrase (swap synonymous words or reorder clauses) in <15% of text
  - `VarianceSteps(snap, rng)` -- add or remove 1-2 non-tool steps (assistant text only)
- Implement `GenerateLabeledPairs(cfg TraceConfig) []LabeledPair` -- generates 60 regression pairs (10 per mutation type) + 30 normal-variance pairs (10 per variance type) = 90 total pairs. Each pair's Baseline snapshot gets a unique ID (e.g. `fmt.Sprintf("base-%d", i)`) so cross-validation can group pairs sharing a baseline
- All functions use `*rand.Rand` from seed for determinism. Deep-copy snapshots before mutating.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/bench/ -run TestGenerate -count=1 -v`

**Dependencies:** None

### Task 2: Evaluation Metrics

**Files:** `internal/bench/evaluate.go`, `internal/bench/evaluate_test.go`

**Do:**
- Implement `EvaluateDetection(pairs []LabeledPair, cfg config.Config) DetectionResult` -- runs `diff.Compare` (at `internal/diff/text.go:66`) on each pair, classifies verdict against ground truth. Classification rule: `regression` verdict = positive (predicted regression), `pass` or `changed` = negative (predicted normal). Computes confusion matrix against `LabeledPair.IsRegression` labels
- Implement `SweepThreshold(pairs []LabeledPair, dimension string, min, max, step float64) ThresholdResult` -- for "tool" and "text" dimensions, sweep threshold from min to max, at each point run `diff.CompareToolsWithDiagnostics`/`diff.CompareText` to get raw scores, then classify using `score > threshold` as the regression predicate. For "step" dimension, cast threshold to int and use `delta > threshold`. At each point compute TPR/FPR/F1. Compute AUC via trapezoidal rule. Find optimal point (max F1). Note: sweep uses raw per-dimension scores, NOT the combined verdict, so each dimension is evaluated independently
- Implement `ComputeAUC(points []ROCPoint) float64` -- trapezoidal AUC from sorted ROC points
- Implement `EvaluateClustering(traces []LabeledTrace, epsilon float64, minPts int) ClusterResult` -- build a `snapshot.Baseline{Name: "bench", Snapshots: traces}` from traces, run `cluster.ClusterBaseline(baseline, epsilon, minPts)`, compute ARI against ground truth `StrategyID` labels. Default in orchestrator: `epsilon=0` (auto), `minPts=2`. DBSCAN noise points (cluster -1) are included in ARI as their own predicted label (not dropped), so noise penalizes the score honestly
- Implement `AdjustedRandIndex(predicted, truth []int) float64` -- standard ARI formula using contingency table (Hubert & Arabie 1985). Handle edge cases: all same cluster returns 1.0 if truth is also all same, 0.0 otherwise. When Max == Expected (degenerate, including both-all-singletons), return 1.0 if Index == Expected else 0.0
- Implement `CrossValidate(pairs []LabeledPair, folds int, rng *rand.Rand) CrossValResult` -- stratified split: group pairs by (IsRegression, MutationType), distribute each group round-robin across K folds so every fold has proportional representation. Additionally, pairs derived from the same baseline trace (same Baseline.ID) must land in the same fold to prevent leakage. For each fold: calibrate on K-1 folds (sweep thresholds, find max-F1), evaluate on held-out fold. Report mean/std F1 and averaged optimal thresholds

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/bench/ -run TestEvaluate -count=1 -v`

**Dependencies:** Task 1

### Task 3: Bench Orchestrator

**Files:** `internal/bench/bench.go`, `internal/bench/bench_test.go`

**Do:**
- Implement `Run(seed int64, verbose bool) BenchResult` -- orchestrates the full benchmark:
  1. Create `TraceConfig` with given seed
  2. Generate labeled pairs via `GenerateLabeledPairs`
  3. Run `EvaluateDetection` with default thresholds
  4. Run `SweepThreshold` for tool (0.0-1.0 step 0.05), text (0.0-1.0 step 0.05), step (0-20 step 1)
  5. Generate strategy traces via `GenerateStrategyTraces`
  6. Run `EvaluateClustering(strategyTraces, 0, 2)` -- epsilon=0 (auto), minPts=2 (matches project defaults)
  7. Run `CrossValidate(pairs, 5, rng)` with stratified folds
  8. Assemble and return `BenchResult`
- Implement `FormatTable(result BenchResult, verbose bool) string` -- renders results as a terminal table using tabwriter. Sections: Detection (P/R/F1), Thresholds (current vs optimal), Clustering (ARI), Cross-Validation (mean F1 +/- std). If verbose, add per-mutation-type detection breakdown
- Implement `FormatJSON(result BenchResult) ([]byte, error)` -- JSON marshal with indent
- Integration test: `TestRunBench` runs `Run(42, false)`, asserts no panic, asserts all metrics are in valid ranges (0-1 for rates, >= -0.5 for ARI)

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/bench/ -run TestRun -count=1 -v`

**Dependencies:** Task 1, Task 2

### Task 4: CLI Command

**Files:** `cmd/bench.go`

**Do:**
- Add `benchCmd` cobra command under root: `agentdiff bench`
- Flags: `--seed` (int64, default 42), `--verbose` (bool), `--output` (string, path to write JSON)
- `--json` already exists as persistent flag on root
- Run logic: call `bench.Run(seed, verbose)`. If `--json` or `--output`, marshal JSON. Otherwise print table via `bench.FormatTable`. If `--output` set, write JSON to file
- Register in `init()` via `rootCmd.AddCommand(benchCmd)`
- Exit code 0 always

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build -o /tmp/agentdiff . && /tmp/agentdiff bench --seed 42`

**Dependencies:** Task 3

### Task 5: Existing Test Verification

**Files:** None (read-only)

**Do:**
- Run the full existing test suite to confirm nothing is broken: `go test ./...`
- Run `agentdiff bench --seed 42` and `agentdiff bench --seed 42 --json` and confirm outputs are deterministic (run twice, diff)
- Run `agentdiff bench --seed 123` to confirm different seed produces different but valid results

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./... && /tmp/agentdiff bench --seed 42 --json > /tmp/bench1.json && /tmp/agentdiff bench --seed 42 --json > /tmp/bench2.json && diff /tmp/bench1.json /tmp/bench2.json`

**Dependencies:** Task 4

## The One Hard Thing

**Adjusted Rand Index (ARI) implementation.** ARI requires building a contingency table between predicted clusters and ground-truth labels, then computing the index from combinatorial counts. The formula involves binomial coefficients on contingency table entries, row sums, and column sums. Edge cases are tricky: all points in one cluster, every point its own cluster, predicted clusters that don't map 1:1 to ground truth.

**Approach:** Implement the standard formula from Hubert & Arabie (1985). Build the contingency table as a 2D map. Compute n_ij choose 2 for each cell, a_i choose 2 for row sums, b_j choose 2 for column sums, n choose 2 for total. ARI = (Index - Expected) / (Max - Expected). Handle the degenerate case where Max == Expected (return 0.0 or 1.0 depending on whether Index == Expected).

**Fallback:** If ARI proves unreliable on small cluster counts, fall back to Normalized Mutual Information (NMI), which is simpler to implement and more stable with few clusters. Both are standard clustering quality metrics.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Synthetic mutations too easy to detect (inflated metrics) | **High** | Include subtle mutations (single tool swap, 10% text change). Report per-mutation-type breakdown so users see where detection is weak |
| Threshold overfitting to synthetic data | **High** | 5-fold cross-validation. Report cross-val F1 alongside single-run F1. Document that optimal thresholds are synthetic-only starting points |
| DBSCAN epsilon auto-selection fails on synthetic data | **Medium** | Synthetic strategies are designed with clear separation. If ARI is low, log epsilon and cluster counts for debugging. Can also test with manually-set epsilon |
| Bench runtime too long | **Low** | 90 pairs + 50 strategy traces is small. Levenshtein on 8-tool sequences is microseconds. Full bench should run in <2 seconds. Add timing to output if it exceeds 5s |
| Non-deterministic results despite seeded rng | **Medium** | Use a single `rand.Rand` instance, never `rand.Float64()` global. Pass rng explicitly to all functions. Task 5 validates determinism via JSON diff |
