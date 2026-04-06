# AgentDiff

**pytest for AI agents.** Snapshot agent behavior, diff across changes, catch silent regressions.

## Problem

AI agents ship to production but there's no standard way to test whether a prompt, model, or config change caused a regression. Outputs are non-deterministic, so traditional assertion-based testing fails.

## Quick Start

```bash
go install github.com/jtsilverman/agentdiff@latest

# Record a baseline
agentdiff record --name baseline trace.jsonl

# Make changes, record again
agentdiff record --name after-change trace.jsonl

# Diff
agentdiff diff baseline after-change
```

## Supported Formats

- **Claude Code** -- JSONL conversation traces and stream-json format
- **OpenAI** -- Chat completions messages array (direct or API response wrapper)
- Auto-detection (default)

## How It Works

AgentDiff compares agent traces on two dimensions:

1. **Structural (tool calls)** -- Levenshtein edit distance on the ordered sequence of tool names. Catches: different tools used, different order, different arguments.
2. **Textual (output content)** -- Jaccard similarity on bigram token sets. Robust to rephrasing, catches topical drift.

Configurable thresholds determine when a change is a regression vs. expected variation.

## Configuration

Create `.agentdiff.yaml`:
```yaml
thresholds:
  tool_score: 0.3    # tool diff above this = regression
  text_score: 0.5    # text diff above this = regression
  step_delta: 5      # step count change above this = regression
```

## CI Usage

```yaml
- name: Check for agent regressions
  run: |
    agentdiff record --name baseline golden/trace.jsonl
    agentdiff record --name current current/trace.jsonl
    agentdiff diff baseline current  # exits 1 on regression
```

## Bench Suite

Empirical validation of AgentDiff's regression detection using synthetic traces and mutation testing.

```bash
# Run bench with table output
agentdiff bench

# JSON output for CI/plotting
agentdiff bench --json

# Custom seed, save to file
agentdiff bench --seed 123 --output results.json
```

Evaluates four dimensions:
- **Regression detection** -- precision, recall, F1 on 90 labeled trace pairs (60 mutated + 30 natural variance)
- **Threshold calibration** -- ROC curves + AUC across tool, text, and step dimensions; identifies optimal operating points
- **Clustering quality** -- Adjusted Rand Index on DBSCAN clustering of strategy-labeled traces
- **Cross-validation** -- 5-fold stratified validation with mean/std F1 and averaged optimal thresholds

Deterministic (same seed = same output), runs in <2 seconds.

## What I Learned

- Levenshtein on token sequences (not characters) gives stable structural comparison for non-deterministic agent traces
- Bigram Jaccard is surprisingly robust for "same topic, different words" detection
- The adapter pattern cleanly isolates format changes -- adding a new agent framework is ~100 lines
