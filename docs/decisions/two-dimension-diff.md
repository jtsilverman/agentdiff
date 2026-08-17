# A diff is two independent scores, not an assertion

**Picked:** compare two traces on two dimensions that do not talk to each other.

1. **Structural.** Levenshtein edit distance over the ordered sequence of tool names. Catches a
   different tool, a different order, different arguments.
2. **Textual.** Jaccard similarity over bigram token sets. Survives rephrasing, catches topical drift.

Configurable thresholds in `.agentdiff.yaml` turn the two scores into a regression verdict:
`tool_score: 0.3`, `text_score: 0.5`, `step_delta: 5`.

**Rejected:** assertion-based testing, the way a normal test suite works.

**Reason:** agent output is non-deterministic. An assertion on the text fails on a rephrase that
changed nothing, and an assertion loose enough to survive a rephrase catches no real regression. The
structural shape of a run is far more stable than its wording, so the two dimensions have to be
scored separately and thresholded separately.

**What this constrains:**

- The two scores stay independent. Collapsing them into one number loses the distinction between
  "the agent took a different path" and "the agent said the same thing differently", which is the
  distinction the whole product sells. `fail_on_style_drift` exists so a consumer can fail on one and
  not the other.
- The thresholds are defaults, not truths. `SPEC-bench.md` opens by calling the original numbers
  arbitrary guesses, and `agentdiff bench` exists to measure them: ROC curves and AUC per dimension,
  precision and recall over 90 labeled pairs, 5-fold cross-validation. Change a default only with a
  bench run behind it.
- `--max-steps` defaults to 1000 and truncates to the last N tool calls before alignment. Levenshtein
  is quadratic; an unbounded trace would hang the diff.

**What would reopen it:** a third dimension that the two miss. Multi-model comparison is on the
roadmap and is a candidate.
