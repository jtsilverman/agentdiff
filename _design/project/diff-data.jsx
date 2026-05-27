/* diff-data.jsx — trace step content + pre-computed alignments + triages
   Mirrors GET /api/diff/:a/:b and GET /api/diff/:a/:b/triage shapes.
   Falls back to a small LCS-based aligner for any pair not pre-computed.
*/

// ─── Step content per trace ───
// step ∈ { role: "user"|"assistant"|"tool_call"|"tool_result",
//          content?, tool_call?: {name, args}, tool_result?: {name, output, is_error} }

const TRACE_STEPS = {
  // ─── rate-limit / Run 1 (flask-limiter, 9 steps) ───
  "t-7a4f": [
    { role: "user", content: "Add per-IP rate limiting (100 req/min) to the POST /api/messages endpoint in our Flask app. Use whatever approach you think is cleanest. Tests should pass." },
    { role: "assistant", content: "I need to read app.py first to understand the existing endpoint." },
    { role: "tool_call", tool_call: { name: "read_file", args: { path: "app.py" } } },
    { role: "tool_result", tool_result: { name: "read_file", output: "from flask import Flask, request, jsonify\napp = Flask(__name__)\n\n@app.route('/api/messages', methods=['POST'])\ndef create_message():\n    ...", is_error: false } },
    { role: "assistant", content: "I'll use flask-limiter — clean decorator pattern, no Redis needed for this scale." },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "pip install flask-limiter" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "app.py", change: "Add Limiter(get_remote_address) and @limiter.limit('100/minute') on create_message" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "pytest -x" } } },
    { role: "assistant", content: "All 3 tests pass. Rate limit decorator is in place — 429 with Retry-After is handled automatically by flask-limiter." },
  ],

  // ─── rate-limit / Run 2 (hand-rolled Redis middleware, 13 steps) ───
  "t-2c81": [
    { role: "user", content: "Add per-IP rate limiting (100 req/min) to the POST /api/messages endpoint in our Flask app. Use whatever approach you think is cleanest. Tests should pass." },
    { role: "assistant", content: "I need to read app.py first to understand the existing endpoint." },
    { role: "tool_call", tool_call: { name: "read_file", args: { path: "app.py" } } },
    { role: "tool_result", tool_result: { name: "read_file", output: "from flask import Flask, request, jsonify\napp = Flask(__name__)\n\n@app.route('/api/messages', methods=['POST'])\ndef create_message():\n    ...", is_error: false } },
    { role: "assistant", content: "I'll build a custom before_request middleware backed by Redis INCR — gives explicit control over the counter and is portable across workers." },
    { role: "tool_call", tool_call: { name: "web_search", args: { query: "flask before_request middleware redis INCR sliding window" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "redis_client.py", change: "Create a Redis client wrapper with INCR + EXPIRE helpers" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "middleware.py", change: "before_request hook: INCR rl:{ip}, set EXPIRE 60 on first hit, return 429 if > 100" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "app.py", change: "register before_request middleware on app" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "pip install redis" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "pytest -x" } } },
    { role: "tool_result", tool_result: { name: "shell", output: "FAILED tests/test_rate_limit.py::test_429_after_100 - redis.exceptions.ConnectionError: Error 111", is_error: true } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "tests/conftest.py", change: "Patch redis client with fakeredis for the test session" } } },
  ],

  // ─── auth-migration / v1.2 Run 1 (succeeded, 14 steps) ───
  "t-c1b2": [
    { role: "user", content: "We're moving from stateless JWT to server-side sessions stored in Redis. Update the login, logout, and middleware to use session cookies. Keep the public API unchanged." },
    { role: "assistant", content: "I need to examine the existing auth code and find where jwt.verify is currently called." },
    { role: "tool_call", tool_call: { name: "read_file", args: { path: "auth/index.js" } } },
    { role: "tool_call", tool_call: { name: "grep", args: { pattern: "jwt\\.verify" } } },
    { role: "assistant", content: "I'll use express-session with connect-redis — the standard pattern, httpOnly + SameSite=Lax, 24h TTL." },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm i express-session connect-redis" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "session.js", change: "Configure express-session with RedisStore, secure cookie options" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "routes/login.js", change: "Set req.session.user on successful login, remove JWT signing" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "routes/logout.js", change: "Call req.session.destroy()" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "middleware/auth.js", change: "Replace jwt.verify with req.session.user check" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "routes/index.js", change: "Wire session middleware before routes" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm test" } } },
    { role: "tool_result", tool_result: { name: "shell", output: "12 passing (842ms)", is_error: false } },
    { role: "assistant", content: "All 12 tests pass. Sessions are httpOnly + SameSite=Lax with 24h expiry. Public API is unchanged." },
  ],

  // ─── auth-migration / v1.3 Run 2 (regressed, 22 steps) ───
  "t-bc92": [
    { role: "user", content: "We're moving from stateless JWT to server-side sessions stored in Redis. Update the login, logout, and middleware to use session cookies. Keep the public API unchanged. Also rotate the session ID on every authed request for defense in depth." },
    { role: "assistant", content: "I need to examine the existing auth code and find where jwt.verify is currently called." },
    { role: "tool_call", tool_call: { name: "read_file", args: { path: "auth/index.js" } } },
    { role: "tool_call", tool_call: { name: "grep", args: { pattern: "jwt\\.verify" } } },
    { role: "assistant", content: "I'll use express-session with connect-redis, plus a refresh middleware to rotate the session ID on every authed request as requested." },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm i express-session connect-redis" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "session.js", change: "Configure express-session with RedisStore, secure cookie options" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "routes/login.js", change: "Set req.session.user on successful login, remove JWT signing" } } },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "middleware/refresh.js", change: "On every authed request, regenerate session ID via req.session.regenerate()" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm test" } } },
    { role: "tool_result", tool_result: { name: "shell", output: "Timeout: test_concurrent_session_access exceeded 30s — process hung", is_error: true } },
    { role: "assistant", content: "Test timed out on concurrent access. Likely a race between session read and refresh rotation." },
    { role: "tool_call", tool_call: { name: "read_file", args: { path: "middleware/refresh.js" } } },
    { role: "assistant", content: "Refresh middleware acquires a write lock on the session while the route handler holds a read lock. Classic deadlock under concurrent requests." },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "middleware/refresh.js", change: "Release read lock before calling regenerate, then re-acquire" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm test" } } },
    { role: "tool_result", tool_result: { name: "shell", output: "Timeout: test_concurrent_session_access exceeded 60s — process hung", is_error: true } },
    { role: "assistant", content: "Still deadlocking. Trying to skip rotation entirely on GET requests — only rotate on state-changing verbs." },
    { role: "tool_call", tool_call: { name: "edit_file", args: { path: "middleware/refresh.js", change: "Skip regenerate when req.method === 'GET'" } } },
    { role: "tool_call", tool_call: { name: "shell", args: { cmd: "npm test" } } },
    { role: "tool_result", tool_result: { name: "shell", output: "Timeout: test_concurrent_session_access exceeded 60s — process hung", is_error: true } },
    { role: "assistant", content: "Out of budget — this is a structural deadlock between the refresh rotation and concurrent session reads. The 'rotate on every authed request' requirement conflicts with connect-redis's lock model. Recommend dropping the rotation requirement or moving to a JWT-with-rotation hybrid." },
  ],
};

// ─── Pre-computed alignments (hand-tuned for the two screenshot pairs) ───
// Mirrors API: alignment[].op ∈ "match" | "substitute" | "insert" | "delete"
// a_step / b_step are step objects (or null for insert/delete)

function _alignFromIdx(stepsA, stepsB, plan) {
  return plan.map(([a, b, op]) => ({
    a_step: a !== null ? stepsA[a] : null,
    b_step: b !== null ? stepsB[b] : null,
    a_index: a,
    b_index: b,
    op,
  }));
}

function _summary(alignment) {
  let matches = 0, substitutions = 0, insertions = 0, deletions = 0;
  alignment.forEach(a => {
    if (a.op === "match") matches++;
    else if (a.op === "substitute") substitutions++;
    else if (a.op === "insert") insertions++;
    else if (a.op === "delete") deletions++;
  });
  return { matches, substitutions, insertions, deletions };
}

// ── Pair 1 — same baseline (rate-limit Run 1 vs Run 2), subtle subs only ──
const PAIR_RL_1_2 = (() => {
  const A = TRACE_STEPS["t-7a4f"], B = TRACE_STEPS["t-2c81"];
  const plan = [
    [0, 0, "match"],
    [1, 1, "match"],
    [2, 2, "match"],
    [3, 3, "match"],
    [4, 4, "substitute"],
    [5, 5, "substitute"],
    [null, 6, "insert"],
    [null, 7, "insert"],
    [6, 8, "substitute"],
    [null, 9, "insert"],
    [7, 10, "match"],
    [null, 11, "insert"],
    [null, 12, "insert"],
    [8, null, "delete"],
  ];
  const alignment = _alignFromIdx(A, B, plan);
  const summary = _summary(alignment);
  return { alignment, summary, distance: summary.substitutions + summary.insertions + summary.deletions };
})();

// ── Pair 2 — cross-baseline (rate-limit Run 1 vs auth-migration v1.3 Run 2), heavy divergence ──
const PAIR_CROSS = (() => {
  const A = TRACE_STEPS["t-7a4f"], B = TRACE_STEPS["t-bc92"];
  const plan = [
    [0, null, "delete"],
    [null, 0, "insert"],
    [1, null, "delete"],
    [null, 1, "insert"],
    [2, 2, "substitute"],     // both read_file (different paths)
    [3, null, "delete"],
    [null, 3, "insert"],      // grep jwt.verify
    [4, 4, "substitute"],     // assistant strategy
    [5, 5, "substitute"],     // shell install (pip vs npm)
    [6, 6, "substitute"],     // edit_file (different paths)
    [null, 7, "insert"],
    [null, 8, "insert"],
    [7, 9, "substitute"],     // shell test (pytest vs npm test)
    [null, 10, "insert"],
    [null, 11, "insert"],
    [null, 12, "insert"],
    [null, 13, "insert"],
    [null, 14, "insert"],
    [null, 15, "insert"],
    [null, 16, "insert"],
    [null, 17, "insert"],
    [null, 18, "insert"],
    [null, 19, "insert"],
    [null, 20, "insert"],
    [null, 21, "insert"],
    [8, null, "delete"],
  ];
  const alignment = _alignFromIdx(A, B, plan);
  const summary = _summary(alignment);
  return { alignment, summary, distance: summary.substitutions + summary.insertions + summary.deletions };
})();

// ── Pair 3 — same baseline regression (auth-migration v1.2 succeeded vs v1.3 regressed) ──
const PAIR_AUTH_OK_HANG = (() => {
  const A = TRACE_STEPS["t-c1b2"], B = TRACE_STEPS["t-bc92"];
  const plan = [
    [0, 0, "substitute"],   // prompt — v1.3 added "rotate on every authed request"
    [1, 1, "match"],
    [2, 2, "match"],
    [3, 3, "match"],
    [4, 4, "substitute"],   // v1.2: standard pattern. v1.3: add refresh middleware
    [5, 5, "match"],
    [6, 6, "match"],
    [7, 7, "match"],
    [8, null, "delete"],    // v1.2 logout edit
    [9, null, "delete"],    // v1.2 auth middleware edit
    [10, null, "delete"],   // v1.2 routes/index wiring
    [null, 8, "insert"],    // v1.3 refresh middleware
    [11, 9, "match"],       // npm test
    [12, 10, "substitute"], // result: 12 passing vs timeout
    [13, null, "delete"],   // v1.2 final answer
    [null, 11, "insert"],   // v1.3 assistant: investigating
    [null, 12, "insert"],   // v1.3 read refresh.js
    [null, 13, "insert"],   // v1.3 deadlock theory
    [null, 14, "insert"],   // v1.3 edit
    [null, 15, "insert"],   // v1.3 npm test
    [null, 16, "insert"],   // v1.3 timeout 2
    [null, 17, "insert"],
    [null, 18, "insert"],
    [null, 19, "insert"],
    [null, 20, "insert"],
    [null, 21, "insert"],
  ];
  const alignment = _alignFromIdx(A, B, plan);
  const summary = _summary(alignment);
  return { alignment, summary, distance: summary.substitutions + summary.insertions + summary.deletions };
})();

const PRECOMPUTED_DIFFS = {
  "t-7a4f|t-2c81": PAIR_RL_1_2,
  "t-2c81|t-7a4f": PAIR_RL_1_2,  // symmetric — same result regardless of direction
  "t-7a4f|t-bc92": PAIR_CROSS,
  "t-bc92|t-7a4f": PAIR_CROSS,
  "t-c1b2|t-bc92": PAIR_AUTH_OK_HANG,
  "t-bc92|t-c1b2": PAIR_AUTH_OK_HANG,
};

const TRIAGES = {
  "t-7a4f|t-2c81": {
    summary: "Same task, different strategy. Run 1 used flask-limiter (decorator). Run 2 hand-rolled a Redis-backed middleware. Both pass tests.",
    classification: "variance",
    likely_cause: "The prompt is open about implementation, so the agent picks differently each run. Variance is expected on prompts that under-specify a library choice — not a bug.",
  },
  "t-2c81|t-7a4f": null, // populated below
  "t-7a4f|t-bc92": {
    summary: "These traces are from different baselines and aren't directly comparable. Most divergence reflects task differences, not agent behavior.",
    classification: "variance",
    likely_cause: "The traces solve different problems — rate limiting vs. session migration. For meaningful drift analysis, compare two traces from the same baseline.",
  },
  "t-bc92|t-7a4f": null,
  "t-c1b2|t-bc92": {
    summary: "v1.3 of the prompt added a 'rotate the session ID on every authed request' requirement. v1.2 succeeded with the standard pattern. v1.3 deadlocks on concurrent session reads and times out three times before giving up.",
    classification: "regression",
    likely_cause: "The new requirement conflicts with connect-redis's lock model — rotating the session ID while another request holds a read lock causes a deadlock. The prompt change is the proximate cause; suggest dropping the rotation requirement or moving to a JWT-with-rotation hybrid.",
  },
  "t-bc92|t-c1b2": null,
};
// mirror symmetric triages
TRIAGES["t-2c81|t-7a4f"] = TRIAGES["t-7a4f|t-2c81"];
TRIAGES["t-bc92|t-7a4f"] = TRIAGES["t-7a4f|t-bc92"];
TRIAGES["t-bc92|t-c1b2"] = TRIAGES["t-c1b2|t-bc92"];

// ─── Runtime LCS fallback (for any pair without pre-computed alignment) ───
// Equality predicate: same role + same tool name + content prefix matches
function _stepKey(s) {
  if (!s) return "";
  const t = s.tool_call?.name || s.tool_result?.name || "";
  const c = (s.content || s.tool_call?.args?.path || s.tool_result?.output || "").slice(0, 30);
  return `${s.role}::${t}::${c}`;
}

function _runLCS(a, b) {
  const m = a.length, n = b.length;
  const dp = Array.from({length: m + 1}, () => new Array(n + 1).fill(0));
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (_stepKey(a[i - 1]) === _stepKey(b[j - 1])) dp[i][j] = dp[i - 1][j - 1] + 1;
      else dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
    }
  }
  const out = [];
  let i = m, j = n;
  while (i > 0 && j > 0) {
    if (_stepKey(a[i - 1]) === _stepKey(b[j - 1])) {
      out.unshift({ a: i - 1, b: j - 1, op: "match" });
      i--; j--;
    } else if (dp[i - 1][j] >= dp[i][j - 1]) {
      out.unshift({ a: i - 1, b: null, op: "delete" });
      i--;
    } else {
      out.unshift({ a: null, b: j - 1, op: "insert" });
      j--;
    }
  }
  while (i > 0) { i--; out.unshift({ a: i, b: null, op: "delete" }); }
  while (j > 0) { j--; out.unshift({ a: null, b: j, op: "insert" }); }
  // Merge adjacent (delete, insert) pairs of same role into substitute
  const merged = [];
  for (let k = 0; k < out.length; k++) {
    const cur = out[k], nxt = out[k + 1];
    if (cur && nxt &&
        cur.op === "delete" && nxt.op === "insert" &&
        a[cur.a] && b[nxt.b] &&
        a[cur.a].role === b[nxt.b].role) {
      merged.push({ a: cur.a, b: nxt.b, op: "substitute" });
      k++;
    } else {
      merged.push(cur);
    }
  }
  return merged;
}

function computeDiff(aId, bId) {
  if (aId === bId) {
    return { alignment: [], summary: { matches: 0, substitutions: 0, insertions: 0, deletions: 0 }, distance: 0 };
  }
  const key = `${aId}|${bId}`;
  if (PRECOMPUTED_DIFFS[key]) return PRECOMPUTED_DIFFS[key];

  const A = TRACE_STEPS[aId];
  const B = TRACE_STEPS[bId];
  if (!A || !B) return null;

  const plan = _runLCS(A, B).map(p => [p.a, p.b, p.op]);
  const alignment = _alignFromIdx(A, B, plan);
  const summary = _summary(alignment);
  return { alignment, summary, distance: summary.substitutions + summary.insertions + summary.deletions };
}

function getTriage(aId, bId) {
  if (aId === bId) return null;
  return TRIAGES[`${aId}|${bId}`] || {
    summary: "No AI triage available for this pair yet. The diff is shown below.",
    classification: "variance",
    likely_cause: "We haven't run triage on this trace pair. Run analysis to populate.",
  };
}

Object.assign(window, { TRACE_STEPS, computeDiff, getTriage });
