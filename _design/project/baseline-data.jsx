/* baseline-data.jsx — baselines + traces + path graphs */

const BASELINES = {
  "rate-limit": {
    id: "rate-limit",
    task: "Add rate limiting to a Flask endpoint",
    prompt: "Add per-IP rate limiting (100 requests/minute) to the POST /api/messages endpoint in our Flask app. Use whatever approach you think is cleanest. Tests should pass.",
    description: [
      "The agent receives a small Flask repo with one endpoint and a test suite.",
      "Success means the endpoint enforces 100 req/min per IP, returns 429 with a Retry-After header on excess, and the existing tests still pass.",
      "The agent is free to pick a library, hand-roll middleware, or use Redis directly. We ran the same prompt five times.",
    ],
    model: "claude-sonnet-4.5",
    captured: "2026-05-24 14:22 UTC",
    durationAvg: "38s",
    stepsAvg: 11.2,
    outcomeKind: "warn",
    outcomeLabel: "high variance",
    traces: [
      { id: "t-7a4f", name: "Run 1", outcome: "ok", outcomeLabel: "ok", steps: 9, duration: "31s", strategy: "flask-limiter", keyDecision: "Chose flask-limiter — used decorator pattern, no Redis", cluster: "A" },
      { id: "t-2c81", name: "Run 2", outcome: "ok", outcomeLabel: "ok", steps: 13, duration: "45s", strategy: "hand-rolled middleware", keyDecision: "Built before_request middleware backed by Redis INCR", cluster: "B" },
      { id: "t-9b30", name: "Run 3", outcome: "ok", outcomeLabel: "ok", steps: 10, duration: "34s", strategy: "flask-limiter", keyDecision: "flask-limiter again, but with in-memory backend (no Redis)", cluster: "A" },
      { id: "t-d57e", name: "Run 4", outcome: "novel", outcomeLabel: "novel", steps: 14, duration: "52s", strategy: "redis lua script", keyDecision: "Wrote a Lua atomic counter, skipped Python middleware entirely", cluster: "C" },
      { id: "t-411a", name: "Run 5", outcome: "ok", outcomeLabel: "ok", steps: 10, duration: "32s", strategy: "flask-limiter", keyDecision: "flask-limiter, simpler config than Run 1", cluster: "A" },
    ],
    legend: [
      { label: "Strategy A · flask-limiter", color: "oklch(0.78 0.14 155)", count: 3 },
      { label: "Strategy B · hand-rolled middleware", color: "#22d3ee", count: 1 },
      { label: "Strategy C · redis lua (novel)", color: "oklch(0.72 0.14 290)", count: 1 },
    ],
  },

  "auth-migration": {
    id: "auth-migration",
    task: "Migrate auth from JWT to session cookies",
    prompt: "We're moving from stateless JWT to server-side sessions stored in Redis. Update the login, logout, and middleware to use session cookies. Keep the public API unchanged.",
    description: [
      "The agent is given an Express + Redis monorepo with existing JWT middleware and a routes file.",
      "Success means cookies are httpOnly + SameSite=Lax, sessions expire after 24h, and the existing integration tests still pass without modification.",
      "We ran the v1.2 prompt (4 runs) and v1.3 prompt (2 runs) — v1.3 added a single sentence about token refresh, and one run regressed.",
    ],
    model: "claude-sonnet-4.5",
    captured: "2026-05-26 09:08 UTC",
    durationAvg: "1m12s",
    stepsAvg: 17.5,
    outcomeKind: "bad",
    outcomeLabel: "regression",
    traces: [
      { id: "t-c1b2", name: "v1.2 · Run 1", outcome: "ok", outcomeLabel: "baseline", steps: 16, duration: "1m04s", strategy: "express-session + connect-redis", keyDecision: "Standard pattern — express-session w/ connect-redis store", cluster: "baseline" },
      { id: "t-44d8", name: "v1.2 · Run 2", outcome: "ok", outcomeLabel: "baseline", steps: 17, duration: "1m09s", strategy: "express-session + connect-redis", keyDecision: "Same pattern, added explicit session.touch() in middleware", cluster: "baseline" },
      { id: "t-9aef", name: "v1.2 · Run 3", outcome: "ok", outcomeLabel: "baseline", steps: 18, duration: "1m11s", strategy: "express-session + connect-redis", keyDecision: "Same pattern, refactored middleware order", cluster: "baseline" },
      { id: "t-0f30", name: "v1.2 · Run 4", outcome: "ok", outcomeLabel: "baseline", steps: 16, duration: "1m05s", strategy: "express-session + connect-redis", keyDecision: "Same pattern, clean", cluster: "baseline" },
      { id: "t-bc91", name: "v1.3 · Run 1", outcome: "ok", outcomeLabel: "new ok", steps: 19, duration: "1m18s", strategy: "express-session + manual refresh", keyDecision: "Added a refresh loop — slower but passes", cluster: "v13" },
      { id: "t-bc92", name: "v1.3 · Run 2", outcome: "bad", outcomeLabel: "hung", steps: 22, duration: "timeout", strategy: "express-session + manual refresh", keyDecision: "Refresh loop deadlocked when session was being read mid-rotate", cluster: "regressed" },
    ],
    legend: [
      { label: "v1.2 baseline · ok", color: "oklch(0.78 0.14 155)", count: 4 },
      { label: "v1.3 · ok but slower", color: "#22d3ee", count: 1 },
      { label: "v1.3 · regressed (hung)", color: "oklch(0.68 0.18 25)", count: 1 },
    ],
  },

  "flaky-test": {
    id: "flaky-test",
    task: "Debug intermittent test failure in CI",
    prompt: "test_user_signup is failing about 1 in 20 times on CI, never locally. Find the cause and fix it. The CI logs are in /var/log/ci. The test code is in tests/integration/.",
    description: [
      "The agent gets read access to CI logs, the failing test file, and the codebase.",
      "Success means a fix that eliminates the flake — verified by 50 consecutive CI runs.",
      "We ran the prompt 5 times. Four runs chased a presumed race condition. The fifth noticed the test parses timestamps with a 2-digit year — and CI's runner had a clock skew. It was right.",
    ],
    model: "claude-sonnet-4.5",
    captured: "2026-05-25 17:51 UTC",
    durationAvg: "2m04s",
    stepsAvg: 23.4,
    outcomeKind: "novel",
    outcomeLabel: "novel strategy",
    traces: [
      { id: "t-aa11", name: "Run 1", outcome: "warn", outcomeLabel: "wrong fix", steps: 24, duration: "2m12s", strategy: "patch race condition", keyDecision: "Added mutex around user_count() — passed locally, still flakes on CI", cluster: "race" },
      { id: "t-aa12", name: "Run 2", outcome: "warn", outcomeLabel: "wrong fix", steps: 22, duration: "1m58s", strategy: "patch race condition", keyDecision: "Added retry-on-failure to test (masks the bug)", cluster: "race" },
      { id: "t-aa13", name: "Run 3", outcome: "warn", outcomeLabel: "wrong fix", steps: 25, duration: "2m21s", strategy: "patch race condition", keyDecision: "Added DB transaction wrapping — wrong root cause", cluster: "race" },
      { id: "t-aa14", name: "Run 4", outcome: "novel", outcomeLabel: "fix", steps: 21, duration: "1m44s", strategy: "fix clock skew handling", keyDecision: "Parsed CI runner clock from logs, noticed 2y skew, fixed timestamp parser", cluster: "novel" },
      { id: "t-aa15", name: "Run 5", outcome: "warn", outcomeLabel: "wrong fix", steps: 25, duration: "2m04s", strategy: "patch race condition", keyDecision: "Added await on a Promise.race — passed locally, still flakes", cluster: "race" },
    ],
    legend: [
      { label: "Obvious path · race condition", color: "#22d3ee", count: 4 },
      { label: "Novel · fixed real bug", color: "oklch(0.72 0.14 290)", count: 1 },
    ],
  },
};

// ─── Path graph node layouts per baseline ───
// Each node: id, label, x, y, freq (size scale)
// Each edge: from, to, runCount, color

const PATH_GRAPHS = {
  "rate-limit": {
    nodes: [
      { id: "start", label: "prompt", x: 60,  y: 200, freq: 5, kind: "start" },
      { id: "read", label: "read_file:app.py", x: 180, y: 200, freq: 5 },
      { id: "search", label: "search:flask rate limit", x: 320, y: 140, freq: 4 },
      { id: "search2", label: "search:redis rate limit", x: 320, y: 280, freq: 1 },
      // Strategy A — flask-limiter
      { id: "instA", label: "shell:pip install flask-limiter", x: 480, y: 80, freq: 3, cluster: "A" },
      { id: "writeA", label: "edit:app.py @limiter", x: 640, y: 80, freq: 3, cluster: "A" },
      // Strategy B — hand-rolled middleware
      { id: "writeB", label: "edit:middleware.py", x: 480, y: 200, freq: 1, cluster: "B" },
      { id: "redisB", label: "edit:redis_client.py", x: 640, y: 200, freq: 1, cluster: "B" },
      // Strategy C — redis lua novel
      { id: "writeC", label: "write:rate_limit.lua", x: 480, y: 320, freq: 1, cluster: "C" },
      { id: "wireC", label: "edit:app.py (eval lua)", x: 640, y: 320, freq: 1, cluster: "C" },
      // Tests
      { id: "test", label: "shell:pytest", x: 800, y: 200, freq: 5 },
      { id: "done", label: "done", x: 920, y: 200, freq: 5, kind: "end" },
    ],
    edges: [
      { from: "start", to: "read", count: 5 },
      { from: "read", to: "search", count: 4 },
      { from: "read", to: "search2", count: 1, color: "novel" },
      { from: "search", to: "instA", count: 3, color: "ok" },
      { from: "search", to: "writeB", count: 1, color: "accent" },
      { from: "search2", to: "writeC", count: 1, color: "novel" },
      { from: "instA", to: "writeA", count: 3, color: "ok" },
      { from: "writeA", to: "test", count: 3, color: "ok" },
      { from: "writeB", to: "redisB", count: 1, color: "accent" },
      { from: "redisB", to: "test", count: 1, color: "accent" },
      { from: "writeC", to: "wireC", count: 1, color: "novel" },
      { from: "wireC", to: "test", count: 1, color: "novel" },
      { from: "test", to: "done", count: 5 },
    ],
  },

  "auth-migration": {
    nodes: [
      { id: "start", label: "prompt", x: 60,  y: 200, freq: 6, kind: "start" },
      { id: "read", label: "read:auth/index.js", x: 180, y: 200, freq: 6 },
      { id: "grep", label: "grep:jwt.verify", x: 320, y: 200, freq: 6 },
      { id: "instSess", label: "shell:npm i express-session connect-redis", x: 470, y: 200, freq: 6 },
      { id: "writeSess", label: "edit:session.js", x: 620, y: 200, freq: 6 },
      // Baseline path
      { id: "editLogin", label: "edit:routes/login.js", x: 760, y: 140, freq: 5, cluster: "baseline" },
      { id: "testBase", label: "shell:npm test", x: 900, y: 140, freq: 5, cluster: "baseline" },
      // Regressed path — extra refresh middleware
      { id: "refresh", label: "edit:middleware/refresh.js", x: 760, y: 260, freq: 2, cluster: "v13" },
      { id: "testR", label: "shell:npm test", x: 900, y: 260, freq: 1, cluster: "v13" },
      { id: "hung", label: "TIMEOUT", x: 900, y: 320, freq: 1, kind: "fail", cluster: "regressed" },
      { id: "done", label: "done", x: 1020, y: 200, freq: 5, kind: "end" },
    ],
    edges: [
      { from: "start", to: "read", count: 6 },
      { from: "read", to: "grep", count: 6 },
      { from: "grep", to: "instSess", count: 6 },
      { from: "instSess", to: "writeSess", count: 6 },
      { from: "writeSess", to: "editLogin", count: 5, color: "ok" },
      { from: "editLogin", to: "testBase", count: 5, color: "ok" },
      { from: "testBase", to: "done", count: 5, color: "ok" },
      { from: "writeSess", to: "refresh", count: 2, color: "accent" },
      { from: "refresh", to: "testR", count: 1, color: "accent" },
      { from: "testR", to: "done", count: 1, color: "accent" },
      { from: "refresh", to: "hung", count: 1, color: "bad" },
    ],
  },

  "flaky-test": {
    nodes: [
      { id: "start", label: "prompt", x: 60,  y: 200, freq: 5, kind: "start" },
      { id: "readT", label: "read:tests/integration/signup_test.js", x: 220, y: 200, freq: 5 },
      { id: "grep", label: "grep:flake|race", x: 380, y: 200, freq: 5 },
      // Obvious path
      { id: "readCode", label: "read:src/user.js", x: 520, y: 140, freq: 4, cluster: "race" },
      { id: "patch", label: "edit:add mutex/await/retry", x: 700, y: 140, freq: 4, cluster: "race" },
      { id: "localTest", label: "shell:npm test (local pass)", x: 860, y: 140, freq: 4, cluster: "race" },
      { id: "submit", label: "submit (still flaky)", x: 1000, y: 140, freq: 4, kind: "warn", cluster: "race" },
      // Novel path
      { id: "readLogs", label: "read:/var/log/ci/runner-*.log", x: 520, y: 280, freq: 1, cluster: "novel" },
      { id: "noticeSkew", label: "noticed: clock skew 2y", x: 700, y: 280, freq: 1, cluster: "novel" },
      { id: "editParser", label: "edit:lib/parse_ts.js", x: 860, y: 280, freq: 1, cluster: "novel" },
      { id: "fix", label: "fix verified", x: 1000, y: 280, freq: 1, kind: "novelEnd", cluster: "novel" },
    ],
    edges: [
      { from: "start", to: "readT", count: 5 },
      { from: "readT", to: "grep", count: 5 },
      { from: "grep", to: "readCode", count: 4, color: "accent" },
      { from: "grep", to: "readLogs", count: 1, color: "novel" },
      { from: "readCode", to: "patch", count: 4, color: "accent" },
      { from: "patch", to: "localTest", count: 4, color: "accent" },
      { from: "localTest", to: "submit", count: 4, color: "accent" },
      { from: "readLogs", to: "noticeSkew", count: 1, color: "novel" },
      { from: "noticeSkew", to: "editParser", count: 1, color: "novel" },
      { from: "editParser", to: "fix", count: 1, color: "novel" },
    ],
  },
};

Object.assign(window, { BASELINES, PATH_GRAPHS });
