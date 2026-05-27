/* traces-data.jsx — shared corpus + helpers
   Used by /traces (list browser) and /diff (compare view).
   Exposes globals on `window` so the host page can pick them up regardless of
   <script> load order (every consumer guards with `window.CORPUS_TRACES`).
*/

// Synthetic clock (so "Xm ago" is stable across reloads in this demo build)
const CORPUS_NOW = Date.parse("2026-05-27T16:00:00Z");
const _ago = (mins) => new Date(CORPUS_NOW - mins * 60_000).toISOString();

// ─── API outcome → existing OutcomeBadge kind ───
const CORPUS_OUTCOME_KIND = {
  succeeded: "ok",
  variance:  "warn",
  regressed: "bad",
  additive:  "novel",
};
const CORPUS_OUTCOME_LABEL = {
  succeeded: "succeeded",
  variance:  "variance",
  regressed: "regressed",
  additive:  "additive",
};

// ─── Corpus — mirrors GET /api/traces shape ───
// {id, name, adapter, step_count, metadata: {task, outcome, key_decision?}, created_at}
// + the additive baseline_name field (designed for, will ship backend-side).
const CORPUS_TRACES = [
  // rate-limit baseline
  { id: "t-7a4f", name: "Run 1",        baseline_name: "rate-limit",       baseline_id: "rate-limit",       adapter: "claude-code", step_count: 9,  metadata: { task: "Add rate limiting to a Flask endpoint",     outcome: "succeeded", key_decision: "Chose flask-limiter — used decorator pattern, no Redis" }, created_at: _ago(35) },
  { id: "t-2c81", name: "Run 2",        baseline_name: "rate-limit",       baseline_id: "rate-limit",       adapter: "claude-code", step_count: 13, metadata: { task: "Add rate limiting to a Flask endpoint",     outcome: "succeeded", key_decision: "Built before_request middleware backed by Redis INCR" }, created_at: _ago(40) },
  { id: "t-9b30", name: "Run 3",        baseline_name: "rate-limit",       baseline_id: "rate-limit",       adapter: "claude-code", step_count: 10, metadata: { task: "Add rate limiting to a Flask endpoint",     outcome: "succeeded", key_decision: "flask-limiter again, in-memory backend" }, created_at: _ago(44) },
  { id: "t-d57e", name: "Run 4",        baseline_name: "rate-limit",       baseline_id: "rate-limit",       adapter: "claude-code", step_count: 14, metadata: { task: "Add rate limiting to a Flask endpoint",     outcome: "additive",  key_decision: "Wrote a Lua atomic counter, skipped Python middleware" }, created_at: _ago(48) },
  { id: "t-411a", name: "Run 5",        baseline_name: "rate-limit",       baseline_id: "rate-limit",       adapter: "claude-code", step_count: 10, metadata: { task: "Add rate limiting to a Flask endpoint",     outcome: "succeeded", key_decision: "flask-limiter, simpler config than Run 1" }, created_at: _ago(53) },

  // auth-migration baseline
  { id: "t-c1b2", name: "v1.2 · Run 1", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 16, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "succeeded", key_decision: "express-session + connect-redis (baseline pattern)" }, created_at: _ago(160) },
  { id: "t-44d8", name: "v1.2 · Run 2", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 17, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "succeeded", key_decision: "Same pattern, explicit session.touch() in middleware" }, created_at: _ago(168) },
  { id: "t-9aef", name: "v1.2 · Run 3", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 18, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "succeeded", key_decision: "Same pattern, refactored middleware order" }, created_at: _ago(174) },
  { id: "t-0f30", name: "v1.2 · Run 4", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 16, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "succeeded", key_decision: "Same pattern, clean" }, created_at: _ago(186) },
  { id: "t-bc91", name: "v1.3 · Run 1", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 19, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "variance",  key_decision: "Added a refresh loop — slower but passes" }, created_at: _ago(80) },
  { id: "t-bc92", name: "v1.3 · Run 2", baseline_name: "auth-migration",   baseline_id: "auth-migration",   adapter: "cursor",      step_count: 22, metadata: { task: "Migrate auth from JWT to session cookies",  outcome: "regressed", key_decision: "Refresh loop deadlocked when session was read mid-rotate" }, created_at: _ago(72) },

  // flaky-test baseline
  { id: "t-aa11", name: "Run 1",        baseline_name: "flaky-test",       baseline_id: "flaky-test",       adapter: "claude-code", step_count: 24, metadata: { task: "Debug intermittent test failure in CI",     outcome: "variance",  key_decision: "Added mutex around user_count() — still flakes" }, created_at: _ago(720) },
  { id: "t-aa12", name: "Run 2",        baseline_name: "flaky-test",       baseline_id: "flaky-test",       adapter: "claude-code", step_count: 22, metadata: { task: "Debug intermittent test failure in CI",     outcome: "variance",  key_decision: "Added retry-on-failure (masks the bug)" }, created_at: _ago(740) },
  { id: "t-aa13", name: "Run 3",        baseline_name: "flaky-test",       baseline_id: "flaky-test",       adapter: "claude-code", step_count: 25, metadata: { task: "Debug intermittent test failure in CI",     outcome: "variance",  key_decision: "Added DB transaction wrapping — wrong root cause" }, created_at: _ago(750) },
  { id: "t-aa14", name: "Run 4",        baseline_name: "flaky-test",       baseline_id: "flaky-test",       adapter: "claude-code", step_count: 21, metadata: { task: "Debug intermittent test failure in CI",     outcome: "additive",  key_decision: "Noticed 2y CI clock skew, fixed timestamp parser" }, created_at: _ago(760) },
  { id: "t-aa15", name: "Run 5",        baseline_name: "flaky-test",       baseline_id: "flaky-test",       adapter: "claude-code", step_count: 25, metadata: { task: "Debug intermittent test failure in CI",     outcome: "variance",  key_decision: "Promise.race await — still flakes" }, created_at: _ago(770) },

  { id: "t-key1",  name: "Run 1",       baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code", step_count: 12, metadata: { task: "Rotate API keys without downtime",          outcome: "succeeded", key_decision: "Dual-write window, retire old key after 24h" }, created_at: _ago(2880) },
  { id: "t-key2",  name: "Run 2",       baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code", step_count: 15, metadata: { task: "Rotate API keys without downtime",          outcome: "succeeded", key_decision: "Same dual-write, added healthcheck on new key" }, created_at: _ago(2900) },
  { id: "t-key3",  name: "Run 3",       baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code", step_count: 11, metadata: { task: "Rotate API keys without downtime",          outcome: "succeeded", key_decision: "Skipped healthcheck, monitored 5xx rate instead" }, created_at: _ago(2940) },
  { id: "t-key4",  name: "Run 4",       baseline_name: "api-key-rotation", baseline_id: "api-key-rotation", adapter: "claude-code", step_count: 19, metadata: { task: "Rotate API keys without downtime",          outcome: "additive",  key_decision: "Wrote a key-rotation runbook, then automated it" }, created_at: _ago(3000) },

  { id: "t-cors1", name: "Run 1",       baseline_name: "cors-debug",       baseline_id: "cors-debug",       adapter: "cursor",      step_count: 8,  metadata: { task: "Debug CORS preflight on /api/upload",        outcome: "succeeded", key_decision: "Added OPTIONS handler with explicit Access-Control-Allow-Headers" }, created_at: _ago(8640) },
  { id: "t-cors2", name: "Run 2",       baseline_name: "cors-debug",       baseline_id: "cors-debug",       adapter: "cursor",      step_count: 11, metadata: { task: "Debug CORS preflight on /api/upload",        outcome: "succeeded", key_decision: "Same as Run 1, also fixed Access-Control-Max-Age" }, created_at: _ago(8700) },
  { id: "t-cors3", name: "Run 3",       baseline_name: "cors-debug",       baseline_id: "cors-debug",       adapter: "cursor",      step_count: 14, metadata: { task: "Debug CORS preflight on /api/upload",        outcome: "regressed", key_decision: "Wildcard origin caused credentials check to fail" }, created_at: _ago(8760) },
  { id: "t-cors4", name: "Run 4",       baseline_name: "cors-debug",       baseline_id: "cors-debug",       adapter: "cursor",      step_count: 9,  metadata: { task: "Debug CORS preflight on /api/upload",        outcome: "succeeded", key_decision: "Reverted wildcard, allow-list of trusted origins" }, created_at: _ago(8820) },
  { id: "t-cors5", name: "Run 5",       baseline_name: "cors-debug",       baseline_id: "cors-debug",       adapter: "cursor",      step_count: 10, metadata: { task: "Debug CORS preflight on /api/upload",        outcome: "succeeded", key_decision: "Same, with origin extracted to env var" }, created_at: _ago(8880) },

  { id: "t-cel1",  name: "Run 1",       baseline_name: "celery-retry",     baseline_id: "celery-retry",     adapter: "claude-code", step_count: 17, metadata: { task: "Make a Celery task idempotent",              outcome: "succeeded", key_decision: "Idempotency key in Redis, 24h TTL" }, created_at: _ago(14400) },
  { id: "t-cel2",  name: "Run 2",       baseline_name: "celery-retry",     baseline_id: "celery-retry",     adapter: "claude-code", step_count: 21, metadata: { task: "Make a Celery task idempotent",              outcome: "additive",  key_decision: "Idempotency via DB unique constraint, no Redis" }, created_at: _ago(14500) },
  { id: "t-cel3",  name: "Run 3",       baseline_name: "celery-retry",     baseline_id: "celery-retry",     adapter: "claude-code", step_count: 18, metadata: { task: "Make a Celery task idempotent",              outcome: "succeeded", key_decision: "Redis lock + DB unique constraint (belt + suspenders)" }, created_at: _ago(14600) },

  { id: "t-pg1",   name: "Run 1",       baseline_name: "redis-cache",      baseline_id: "redis-cache",      adapter: "claude-code", step_count: 13, metadata: { task: "Add Redis-backed caching to /products",      outcome: "succeeded", key_decision: "Cache-aside pattern, 5min TTL" }, created_at: _ago(20000) },
  { id: "t-pg2",   name: "Run 2",       baseline_name: "redis-cache",      baseline_id: "redis-cache",      adapter: "claude-code", step_count: 16, metadata: { task: "Add Redis-backed caching to /products",      outcome: "regressed", key_decision: "Cache invalidation missed on PUT — stale reads for 5min" }, created_at: _ago(20100) },
];

// ─── Helpers ───
function corpusRelativeTime(iso) {
  const t = new Date(iso).getTime();
  const diff = (CORPUS_NOW - t) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 86400 * 7) return `${Math.floor(diff / 86400)}d ago`;
  if (diff < 86400 * 30) return `${Math.floor(diff / (86400 * 7))}w ago`;
  return `${Math.floor(diff / (86400 * 30))}mo ago`;
}

function corpusFindTrace(id) {
  return CORPUS_TRACES.find(t => t.id === id) || null;
}

function corpusBaselines() {
  const counts = {};
  CORPUS_TRACES.forEach(t => { counts[t.baseline_id] = (counts[t.baseline_id] || 0) + 1; });
  const order = [];
  CORPUS_TRACES.forEach(t => {
    if (!order.find(b => b.id === t.baseline_id)) {
      order.push({ id: t.baseline_id, name: t.baseline_name, count: counts[t.baseline_id] });
    }
  });
  return order;
}

Object.assign(window, {
  CORPUS_NOW,
  CORPUS_TRACES,
  CORPUS_OUTCOME_KIND,
  CORPUS_OUTCOME_LABEL,
  corpusRelativeTime,
  corpusFindTrace,
  corpusBaselines,
});
