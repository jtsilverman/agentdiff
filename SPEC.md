# AgentDiff

**pytest for AI agents.** Snapshot agent behavior, diff across changes, catch silent regressions.

## Problem

AI agents are going to production but there is no standard way to test whether a prompt, model, or config change caused a regression. Outputs are non-deterministic, so traditional assertion-based testing fails. Developers merge agent changes blind, hoping nothing broke.

## Solution

A CLI tool that records agent execution traces (tool calls, outputs, reasoning) as structured snapshots, then diffs snapshots across runs to surface behavioral regressions. Works like pytest snapshot testing: `agentdiff record` captures a baseline, `agentdiff diff` compares two snapshots, `agentdiff report` summarizes changes in CI-friendly output.

**Architecture decisions:**
- **Trace-first, not output-first.** Comparing final text output is noisy (non-deterministic phrasing). Comparing the structural trace (which tools called, in what order, with what args) is stable and meaningful. Text output gets fuzzy-matched as a secondary signal.
- **Adapter pattern for agent formats.** Claude Code, OpenAI, LangChain all emit different trace formats. Adapters normalize to a common schema. Ship with Claude Code + OpenAI adapters, extensible for others.
- **Local-first, file-based.** Snapshots are JSON files in `.agentdiff/` (like `.pytest_cache`). No server, no account, no cloud. Git-friendly.
- **Fuzzy structural diff.** Tool call sequences compared with edit distance. Text compared with Jaccard similarity on token n-grams. Thresholds configurable.

## Scope

**Building:**
- `agentdiff record` -- ingest a trace file (or stdin), normalize via adapter, save snapshot
- `agentdiff diff <a> <b>` -- structural + fuzzy comparison of two snapshots
- `agentdiff report` -- human-readable and JSON summary of diff results
- `agentdiff list` -- list recorded snapshots
- Claude Code adapter (JSONL conversation format)
- OpenAI adapter (conversation log format: messages array, not raw API response)
- Configurable diff thresholds via `.agentdiff.yaml`
- Exit code 1 on regression (CI integration)

**Not building:**
- Web UI or dashboard (Phase 2)
- GitHub Actions marketplace action (Phase 2)
- Semantic embedding similarity (requires API calls, adds cost/complexity)
- Live agent execution/recording (user provides trace files)
- `agentdiff accept` command for updating golden snapshots (Phase 2)
- Team collaboration features
- Cost tracking

**Ship target:** GitHub (jtsilverman/agentdiff), Homebrew tap, `go install`

## Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.24 | Jake has Go experience (council, skillscore, probe). No Go CLI tools in this space. Fast, single binary. |
| CLI framework | cobra | Same as council. Proven, familiar. |
| Config | gopkg.in/yaml.v3 | Same as council. `.agentdiff.yaml` for thresholds. |
| Test | stdlib testing | No external test deps needed. |
| Output | stdlib text/tabwriter + encoding/json | Terminal tables + JSON mode. |

Zero external dependencies beyond cobra and yaml. No LLM API calls at runtime.

## Architecture

### Directory Structure

```
agentdiff/
  cmd/
    root.go          -- cobra root, global flags
    record.go        -- agentdiff record
    diff.go          -- agentdiff diff
    report.go        -- agentdiff report (wraps diff with formatting)
    list.go          -- agentdiff list
  internal/
    adapter/
      adapter.go     -- Adapter interface + registry
      claude.go      -- Claude Code JSONL adapter
      claude_test.go
      openai.go      -- OpenAI chat completions adapter
      openai_test.go
      detect.go      -- auto-detect format from input
      detect_test.go
    snapshot/
      snapshot.go    -- Snapshot data model + read/write
      snapshot_test.go
      store.go       -- .agentdiff/ directory management
      store_test.go
    diff/
      diff.go        -- core diff engine
      diff_test.go
      tools.go       -- tool call sequence comparison (edit distance)
      tools_test.go
      text.go        -- fuzzy text comparison (Jaccard n-gram)
      text_test.go
    config/
      config.go      -- .agentdiff.yaml loading + defaults
      config_test.go
    report/
      terminal.go    -- human-readable output
      json.go        -- JSON output
      report_test.go
  main.go            -- func main() { cmd.Execute() }
  go.mod
  go.sum
  .agentdiff.yaml    -- example config
  testdata/
    claude_trace.jsonl
    openai_trace.json
```

### Data Models

**Snapshot** (normalized trace, stored as JSON in `.agentdiff/snapshots/`):
```go
type Snapshot struct {
    ID        string            `json:"id"`        // hash of content
    Name      string            `json:"name"`      // user-provided label
    Source    string            `json:"source"`     // "claude-code", "openai"
    Timestamp time.Time         `json:"timestamp"`
    Metadata  map[string]string `json:"metadata"`  // model, config hash, etc.
    Steps     []Step            `json:"steps"`
}

type Step struct {
    Role       string          `json:"role"`        // "user", "assistant", "tool_call", "tool_result"
    Content    string          `json:"content"`     // text content (may be empty for tool calls)
    ToolCall   *ToolCall       `json:"tool_call,omitempty"`
    ToolResult *ToolResult     `json:"tool_result,omitempty"`
}

type ToolCall struct {
    Name string                `json:"name"`
    Args map[string]any        `json:"args"`
}

type ToolResult struct {
    Name    string             `json:"name"`
    Output  string             `json:"output"`
    IsError bool               `json:"is_error"`
}
```

**DiffResult** (output of comparison):
```go
type DiffResult struct {
    Snapshot1   string          `json:"snapshot_1"`
    Snapshot2   string          `json:"snapshot_2"`
    Overall     Verdict         `json:"overall"`       // "pass", "regression", "changed"
    ToolDiff    ToolDiffResult  `json:"tool_diff"`
    TextDiff    TextDiffResult  `json:"text_diff"`
    StepsDiff   StepsDiffResult `json:"steps_diff"`
}

type Verdict string
const (
    VerdictPass       Verdict = "pass"
    VerdictChanged    Verdict = "changed"
    VerdictRegression Verdict = "regression"
)

type ToolDiffResult struct {
    Added     []string  `json:"added"`
    Removed   []string  `json:"removed"`
    Reordered bool      `json:"reordered"`
    EditDist  int       `json:"edit_distance"`
    Score     float64   `json:"score"`    // 0.0 = identical, 1.0 = completely different
}

type TextDiffResult struct {
    Similarity float64  `json:"similarity"` // 0.0 = unrelated, 1.0 = identical
    Score      float64  `json:"score"`      // inverted: 0.0 = identical, 1.0 = completely different
}

type StepsDiffResult struct {
    CountA int `json:"count_a"`
    CountB int `json:"count_b"`
    Delta  int `json:"delta"`
}
```

**Config** (`.agentdiff.yaml`):
```yaml
thresholds:
  tool_score: 0.3      # tool diff score above this = regression
  text_score: 0.5      # text diff score above this = regression
  step_delta: 5        # step count change above this = regression
```

### CLI Contract

```
agentdiff record [--name <label>] [--adapter claude|openai|auto] <trace-file>
  # Reads trace, normalizes via adapter, saves snapshot to .agentdiff/snapshots/
  # Prints: "Recorded snapshot: <name> (<id>)"
  # Exit 0

agentdiff diff <snapshot-a> <snapshot-b> [--json] [--threshold-tool 0.3] [--threshold-text 0.5]
  # Compares two snapshots by name or ID
  # Prints: diff summary (tool changes, text similarity, step delta, verdict)
  # Exit 0 if pass/changed, Exit 1 if regression

agentdiff list [--json]
  # Lists all snapshots in .agentdiff/snapshots/
  # Exit 0

agentdiff report <snapshot-a> <snapshot-b> [--json]
  # Same as diff but with detailed output: per-step breakdown, arg-level changes, full text excerpts
  # Exit 0 if pass, Exit 1 if regression
```

## Tasks

### Task 1: Project scaffold + config

**Files:** `main.go`, `cmd/root.go`, `internal/config/config.go`, `internal/config/config_test.go`, `go.mod`, `.agentdiff.yaml`

**Do:** Initialize Go module `github.com/jtsilverman/agentdiff`. Create `main.go` calling `cmd.Execute()`. Create cobra root command with global flags (`--json`, `--verbose`). Implement config loading: look for `.agentdiff.yaml` in cwd, parse thresholds with defaults (tool_score: 0.3, text_score: 0.5, step_delta: 5). Write tests for config loading with missing file (defaults), partial file (merge), and full file.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/config/ -v`

**Dependencies:** none

### Task 2: Snapshot data model + store

**Files:** `internal/snapshot/snapshot.go`, `internal/snapshot/snapshot_test.go`, `internal/snapshot/store.go`, `internal/snapshot/store_test.go`

**Do:** Implement the Snapshot, Step, ToolCall, ToolResult structs as specified in Architecture. Implement Store: `Save(snapshot) error`, `Load(nameOrID) (Snapshot, error)`, `List() ([]Snapshot, error)`, `Dir() string`. Store saves snapshots as `<name>.json` in `.agentdiff/snapshots/`. ID is SHA256 of the JSON-serialized Steps array (first 12 hex chars). Save overwrites existing snapshots with the same name by default (last write wins). Tests: round-trip save/load, list multiple, load by name, load by ID prefix, missing snapshot error, overwrite-on-duplicate.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/snapshot/ -v`

**Dependencies:** none

### Task 3: Claude Code adapter

**Files:** `internal/adapter/adapter.go`, `internal/adapter/claude.go`, `internal/adapter/claude_test.go`, `testdata/claude_trace.jsonl`

**Do:** Define the Adapter interface: `Parse(input []byte) ([]snapshot.Step, map[string]string, error)`. The metadata map captures model name, any config info found in the trace. Implement ClaudeAdapter that parses Claude Code JSONL format. Each line is a JSON object with `type` field. Map: `type: "human"` or `type: "user"` -> role user, `type: "assistant"` with `content[].type: "tool_use"` -> role tool_call, `type: "assistant"` with `content[].type: "text"` -> role assistant, tool results -> role tool_result. Skip unknown line types gracefully (log warning, don't error). This handles metadata lines, session markers, etc. Create a realistic testdata file with 5-8 steps including tool calls (Bash, Read, Write). Test: parse testdata, verify step count, verify tool call extraction, verify text content extraction.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -v`

**Dependencies:** Task 2 (imports snapshot types)

### Task 4: OpenAI adapter + auto-detect

**Files:** `internal/adapter/openai.go`, `internal/adapter/openai_test.go`, `internal/adapter/detect.go`, `internal/adapter/detect_test.go`, `testdata/openai_trace.json`

**Do:** Implement OpenAIAdapter that parses OpenAI conversation logs (messages array with role/content/tool_calls/tool_call_id fields). Note: this is the request-side messages array, not the raw API response (which wraps in `choices[].message`). Also support unwrapping raw responses if `choices` key is detected. Create testdata with function calling example (5-8 messages). Implement `Detect(input []byte) (Adapter, error)` that auto-detects format: if input starts with `[` or has `"messages"` key -> try OpenAI; if input has newline-separated JSON objects -> try Claude. Return error if neither works. Test: both adapters parse correctly, detect correctly routes, detect fails gracefully on garbage input.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/adapter/ -v`

**Dependencies:** Task 3 (adapter interface defined there)

### Task 5: Diff engine -- tool comparison

**Files:** `internal/diff/diff.go`, `internal/diff/tools.go`, `internal/diff/tools_test.go`

**Do:** Implement DiffResult, ToolDiffResult, TextDiffResult, StepsDiffResult, Verdict types as specified. Implement `CompareTools(a, b []snapshot.Step) ToolDiffResult`. Two-layer comparison: (1) Sequence diff — extract tool call names as ordered sequences, compute Levenshtein edit distance, detect added/removed/reordered tools. (2) Argument diff — for tool calls with matching names at aligned positions, compute Jaccard similarity on the JSON-serialized arg key-value pairs. Final tool score = weighted average (0.6 * sequence score + 0.4 * arg score). For traces with >1000 tool calls, truncate to last 1000 to keep Levenshtein O(n*m) tractable. Tests: identical sequences (score 0), completely different (score 1), reordered (flag set), same tools with different args (score > 0), added/removed detection, empty sequences.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/diff/ -v`

**Dependencies:** Task 2 (snapshot types)

### Task 6: Diff engine -- text comparison + overall verdict

**Files:** `internal/diff/text.go`, `internal/diff/text_test.go`, `internal/diff/diff_test.go`

**Do:** Implement `CompareText(a, b []snapshot.Step) TextDiffResult`. Extract all assistant-role text content, concatenate per snapshot. Tokenize by whitespace, compute bigram sets, calculate Jaccard similarity (intersection/union of bigram sets). Score = 1 - similarity. Implement `CompareSteps(a, b []snapshot.Step) StepsDiffResult` (just counts and delta). Implement `Compare(a, b snapshot.Snapshot, cfg config.Config) DiffResult` that runs all three comparisons and computes overall verdict: regression if any score exceeds its threshold, changed if any score > 0 but none exceed threshold, pass if all scores are 0. Tests: identical text (similarity 1.0), completely different text, partial overlap, verdict logic with various threshold combinations.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/diff/ -v`

**Dependencies:** Task 5 (diff types defined there), Task 1 (config types)

### Task 7: Report formatters

**Files:** `internal/report/terminal.go`, `internal/report/json.go`, `internal/report/report_test.go`

**Do:** Implement `Terminal(result diff.DiffResult, w io.Writer) error` that prints a human-readable diff report. Format: header with snapshot names, tool changes section (added/removed/reordered with color: green for added, red for removed), text similarity percentage, step count delta, overall verdict in bold (PASS green, CHANGED yellow, REGRESSION red). Use ANSI escape codes for color (with a `--no-color` check via env var `NO_COLOR`). Implement `JSON(result diff.DiffResult, w io.Writer) error` that marshals to indented JSON. Tests: verify terminal output contains expected sections, verify JSON output round-trips through DiffResult unmarshal.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go test ./internal/report/ -v`

**Dependencies:** Task 5 (diff types)

### Task 8: CLI commands -- record + list

**Files:** `cmd/record.go`, `cmd/list.go`

**Do:** Implement `agentdiff record` command. Accepts positional arg (trace file path) or reads stdin if `-` or no arg. Flags: `--name` (default: filename without extension + timestamp), `--adapter` (claude, openai, auto; default auto). Flow: read input -> detect/select adapter -> parse -> create Snapshot -> store.Save. Print confirmation line. Implement `agentdiff list` command. Reads store, prints table (name, id, source, timestamp, step count). Respects `--json` global flag. Test by building binary and running against testdata files.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build -o agentdiff . && ./agentdiff record --name test-claude testdata/claude_trace.jsonl && ./agentdiff list | grep test-claude && rm -rf .agentdiff`

**Dependencies:** Task 4 (adapters complete), Task 2 (store)

### Task 9: CLI commands -- diff + report

**Files:** `cmd/diff.go`, `cmd/report.go`

**Do:** Implement `agentdiff diff` command. Takes two positional args (snapshot names or ID prefixes). Flags: `--threshold-tool`, `--threshold-text`, `--threshold-steps` (override config). Flow: load both snapshots from store -> run diff.Compare -> print terminal report (or JSON if `--json`). Exit 1 if verdict is regression. Implement `agentdiff report` as an alias for diff with verbose output (same logic, just always verbose). Test by recording two snapshots from testdata, running diff, checking exit code.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build -o agentdiff . && ./agentdiff record --name snap-a testdata/claude_trace.jsonl && ./agentdiff record --name snap-b testdata/claude_trace.jsonl && ./agentdiff diff snap-a snap-b && rm -rf .agentdiff`

**Dependencies:** Task 8 (record works), Task 6 (diff engine complete), Task 7 (report formatters)

### Task 10: Integration tests + README

**Files:** `cmd/integration_test.go`, `README.md`

**Do:** Write integration tests that exercise the full CLI flow: record Claude trace, record OpenAI trace, list both, diff identical snapshots (expect pass, exit 0), diff Claude vs OpenAI (expect changed/regression, exit 1). Use `os/exec` to run the built binary. Write README with: problem statement (1 para), quick start (install + 3 commands), supported formats, config reference, CI usage example (GitHub Actions snippet showing `agentdiff diff` with exit code check), "What I Learned" section placeholder. Include a `go install` command.

**Validate:** `cd /Users/rock/Rock/projects/agentdiff && go build -o agentdiff . && go test ./cmd/ -v -run Integration`

**Dependencies:** Task 9 (all commands work)

## The One Hard Thing

**Meaningful comparison of non-deterministic agent outputs.**

Agents don't produce identical outputs between runs. The same prompt might produce different phrasing, different tool call argument formatting, or tools called in a slightly different order. A naive byte-level diff would flag everything as a regression.

**Approach:** Split comparison into structural (tool calls) and textual (output content) channels with independent scoring. Tool call comparison uses Levenshtein edit distance on the ordered sequence of tool names, which catches meaningful changes (different tools, different order) while ignoring irrelevant variation (arg formatting). Text comparison uses Jaccard similarity on bigram token sets, which is robust to rephrasing while catching topical drift. Configurable thresholds let users tune sensitivity.

**Fallback:** If bigram Jaccard proves too noisy in practice, fall back to unigram Jaccard (simpler, less sensitive to word order). The adapter pattern means we can also add an optional `--semantic` flag later that calls an embedding API for true semantic similarity, without changing the core architecture.

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Claude Code trace format is undocumented and may change | High | Adapter pattern isolates format parsing. Pin to known format, add version detection. Ship with real testdata so format changes break tests early. |
| Bigram Jaccard may be too coarse for meaningful text comparison | Medium | Thresholds are configurable. Text diff is a secondary signal (tool-call structural diff is primary). Can upgrade to trigrams, TF-IDF weighting, or optional `--semantic` embedding flag without API changes. Default text threshold is lenient (0.5) to reduce false positives from rephrasing. |
| Users may not have trace files readily available | Medium | Document how to capture traces from Claude Code (`--output-format jsonl`) and OpenAI (log the API response). Phase 2 could add a record-wrapper that runs an agent and captures the trace. |
| Scope creep into observability/monitoring territory | Low | Spec explicitly defers dashboards, cost tracking, live recording. CLI-only, file-based, offline. |
| Go CLI in a Python-dominated AI tooling ecosystem | Low | Go ships as a single binary with no runtime deps. This is a feature for CI/CD. Python users can still use it without managing a venv. |
