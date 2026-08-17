# Every LLM call sits behind an interface with a fallback

**Picked:** each LLM-backed endpoint declares a one-method interface in `web/api/handlers/`. The
handler constructor takes it. `web/api/main.go` is the only place that binds a real implementation.

`Triager`, `Summarizer`, `Counterfactualer`, `EditPrompter`, `Embedder`. The production side lives in
a sibling `triage_anthropic.go`, `transcripts_anthropic.go`, `counterfactual_anthropic.go`,
`edit_prompt_anthropic.go`, and `embedder_voyage.go`.

**Rejected:** calling an HTTP client from inside a handler.

**Reason:** two failures at once. A test would need a network and a key, and a stranger cloning the
repo would hit a dead dashboard. The interface fixes the first, the deterministic fallback fixes the
second.

**What this constrains:**

- A missing key never breaks a request. `main.go` logs a warning and starts. The affected endpoints
  return a deterministic fallback rather than a 500.
- `Embedder` carries nil-receiver semantics. A nil `Embedder` means embedding is off:
  `POST /api/traces` still succeeds, no `trace_embeddings` row is written, and `/similar` returns an
  empty match list for that trace. `main.go` assigns through a typed local because Go's
  typed-nil-interface behavior would otherwise defeat the `embedder != nil` guard in `PostTrace`.
  Read the comment there before touching that block.
- Every result caches on a hash of the canonical step content, so a repeat ask is free. The tables
  are `triage_cache`, `transcripts`, `counterfactual_runs`, `prompt_edit_runs`, and
  `trace_embeddings`.
- The three seeded baselines ship with their LLM responses pre-cached in
  `web/api/seed/seed-cache.json`. That is why the demo reads identically with and without a key.
- The two endpoints that generate a fresh trace, counterfactual and edit-prompt, are the only
  rate-limited routes: 5 per IP per day. `DEMO_KILL_SWITCH=true` short-circuits both to a fixed
  payload with no LLM call at all.
- The browser makes zero LLM calls. Every model request originates in the Go process.

**What would reopen it:** nothing. The interface is what makes the handler tests runnable offline.
