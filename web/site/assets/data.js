/* ============================================================================
   agentdiff — demo data + pure derivations
   Source of truth: seed-scenarios.md (the three demo stories). Each story is
   one realistic task run multiple times. From the raw tool sequences we derive:
     • replayable step lists (user prompt + tool_call / tool_result chain)
     • a merged path-graph trie (nodes = tool calls, edge weight = run count)
     • step-aligned diffs (match / substitute / insert / delete)
   Plain data + pure functions. No framework. Exposed on window.AD.
   ========================================================================= */
(function () {
  'use strict';

  /* ── The three stories ────────────────────────────────────────────────
     Each trace is a sequence of tool calls. `outcome` is the internal badge
     (succeeded / variance / regressed / additive) — per the brief it attaches
     AFTER the story lands, never as the lede. */

  const SCENARIOS = [
    /* STORY 1 — variance · code-review agent on a PR ------------------- */
    {
      id: 'pr-security-review',
      kind: 'variance',
      hero: 'Different paths, same outcome.',
      question: 'Different paths, same outcome — should I be worried?',
      setting:
        'A FinTech runs a code-review agent on every pull request. The team notices it behaves differently across runs on the same PR and wants to know: regression, or normal variation?',
      task: 'Review the pull request that changes the user authentication flow. Flag any security issues.',
      plain:
        'Five runs all caught the same critical bug — a missing CSRF check on the new login form — but reached it three different ways. The path graph shows three strategies fanning out from one task and all converging on success. The variation lives inside known-good strategies. False alarm avoided.',
      readsAs: 'Three strategies, one task, all green. Healthy variance — not a regression.',
      diffPair: [0, 2],
      triage: {
        classification: 'variance',
        summary:
          'Both runs flag the identical issue — the missing CSRF token check on /login. One greps the codebase for comparable auth patterns first; the other runs the auth test suite first. Same finding, different route to it.',
        cause:
          'Non-deterministic tool selection on an under-specified exploration step. The outcome never changed, so this is healthy variance — not drift.',
      },
      traces: [
        { name: 'run-1', outcome: 'variance',
          keyDecision: 'Read the diff first, then grep\u2019d the codebase for similar auth patterns before flagging.',
          calls: [
            { name: 'read_diff', args: { pr_number: 4231 }, output: '+ <form method="POST" action="/login"> \u2026 no hidden _csrf input added' },
            { name: 'grep', args: { pattern: 'csrf', path: 'src/' }, output: 'csrf.go:14 CSRFToken() \u00b7 contact.html + signup.html include _csrf \u2014 login.html does not' },
            { name: 'comment_on_line', args: { file: 'src/templates/forms/login.html', line: 12, comment: 'Missing CSRF token \u2014 every other form binds a hidden _csrf input. Add it here.' }, output: 'Comment posted to PR #4231.' },
          ] },
        { name: 'run-2', outcome: 'variance',
          keyDecision: 'Repeated the grep-first comparison — within-strategy reproducibility check.',
          calls: [
            { name: 'read_diff', args: { pr_number: 4231 }, output: '+ <form method="POST" action="/login"> \u2026 no hidden _csrf input added' },
            { name: 'grep', args: { pattern: 'csrf', path: 'src/' }, output: 'csrf.go:14 CSRFToken() \u00b7 contact.html + signup.html include _csrf \u2014 login.html does not' },
            { name: 'comment_on_line', args: { file: 'src/templates/forms/login.html', line: 12, comment: 'Missing CSRF token \u2014 every other form binds a hidden _csrf input. Add it here.' }, output: 'Comment posted to PR #4231.' },
          ] },
        { name: 'run-3', outcome: 'variance',
          keyDecision: 'Read the diff, then ran the auth test suite to check coverage of the change before flagging.',
          calls: [
            { name: 'read_diff', args: { pr_number: 4231 }, output: '+ <form method="POST" action="/login"> \u2026 no hidden _csrf input added' },
            { name: 'run_tests', args: { suite: 'auth' }, output: 'pass \u00b7 0 of 41 tests exercise the /login CSRF path' },
            { name: 'comment_on_line', args: { file: 'src/templates/forms/login.html', line: 12, comment: 'Missing CSRF token \u2014 every other form binds a hidden _csrf input. Add it here.' }, output: 'Comment posted to PR #4231.' },
          ] },
        { name: 'run-4', outcome: 'variance',
          keyDecision: 'Repeated the test-suite-first strategy — same exploration depth as run-3.',
          calls: [
            { name: 'read_diff', args: { pr_number: 4231 }, output: '+ <form method="POST" action="/login"> \u2026 no hidden _csrf input added' },
            { name: 'run_tests', args: { suite: 'auth' }, output: 'pass \u00b7 0 of 41 tests exercise the /login CSRF path' },
            { name: 'comment_on_line', args: { file: 'src/templates/forms/login.html', line: 12, comment: 'Missing CSRF token \u2014 every other form binds a hidden _csrf input. Add it here.' }, output: 'Comment posted to PR #4231.' },
          ] },
        { name: 'run-5', outcome: 'variance',
          keyDecision: 'Flagged the missing CSRF check straight from the diff — no exploration step at all.',
          calls: [
            { name: 'read_diff', args: { pr_number: 4231 }, output: '+ <form method="POST" action="/login"> \u2026 no hidden _csrf input added' },
            { name: 'comment_on_line', args: { file: 'src/templates/forms/login.html', line: 12, comment: 'Missing CSRF token \u2014 every other form binds a hidden _csrf input. Add it here.' }, output: 'Comment posted to PR #4231.' },
          ] },
      ],
    },

    /* STORY 2 — regression · support-triage agent --------------------- */
    {
      id: 'support-triage-double-charge',
      kind: 'regression',
      hero: 'A simplified prompt silently broke it.',
      question: 'Did the new prompt silently break the agent?',
      setting:
        'A SaaS team runs a support-triage agent that classifies tickets and routes them. Someone simplified its prompt to make it faster. Did edge-case handling survive?',
      task: 'Triage an incoming ticket: a customer says they were charged twice for the same product on the same day.',
      plain:
        'Four runs before the prompt change pulled the customer\u2019s order history, saw the two charges came from different IPs and shipping addresses, and routed to fraud-risk. The run after the change skipped the database lookup, judged from the ticket text alone, and routed to billing. The fraud case sat unnoticed for two days. The diff puts the runs side by side — the missing lookup step is the change that broke the outcome.',
      readsAs: 'One step disappeared. The outcome changed. That is a regression — no assertion test required.',
      diffPair: [0, 4],
      triage: {
        classification: 'regression',
        summary:
          'The "after" run dropped the order-history lookup and classified from the ticket text alone, routing a fraud case to the billing queue. The "before" run pulled the history, saw mismatched IP and shipping, and routed to fraud-risk.',
        cause:
          'The simplified prompt no longer instructs the agent to gather account context before classifying. A load-bearing tool call disappeared and the routing outcome changed with it — this is a regression, not stylistic variation.',
      },
      traces: [
        { name: 'before-1', outcome: 'succeeded',
          keyDecision: 'Pulled 24h order history and spotted mismatched IP + shipping before routing to fraud-risk.',
          calls: [
            { name: 'read_ticket', args: { id: 'TKT-4471' }, output: '"I was charged twice for the same product today."' },
            { name: 'fetch_orders', args: { customer: 'c_9920', window: '24h' }, output: '2 charges · different IP (mobile + desktop) · different shipping addr' },
            { name: 'route_ticket', args: { queue: 'fraud-risk', reason: 'two charges, mismatched IP + shipping' }, output: 'routed \u2192 fraud-risk' },
          ] },
        { name: 'before-2', outcome: 'succeeded',
          keyDecision: 'Reproduced the lookup-then-route flow — same shape as before-1.',
          calls: [
            { name: 'read_ticket', args: { id: 'TKT-4471' }, output: '"I was charged twice for the same product today."' },
            { name: 'fetch_orders', args: { customer: 'c_9920', window: '24h' }, output: '2 charges · different IP (mobile + desktop) · different shipping addr' },
            { name: 'route_ticket', args: { queue: 'fraud-risk', reason: 'two charges, mismatched IP + shipping' }, output: 'routed \u2192 fraud-risk' },
          ] },
        { name: 'before-3', outcome: 'succeeded',
          keyDecision: 'Reproduced the lookup-then-route flow a third time — the pre-change prompt elicits it reliably.',
          calls: [
            { name: 'read_ticket', args: { id: 'TKT-4471' }, output: '"I was charged twice for the same product today."' },
            { name: 'fetch_orders', args: { customer: 'c_9920', window: '24h' }, output: '2 charges · different IP (mobile + desktop) · different shipping addr' },
            { name: 'route_ticket', args: { queue: 'fraud-risk', reason: 'two charges, mismatched IP + shipping' }, output: 'routed \u2192 fraud-risk' },
          ] },
        { name: 'before-4', outcome: 'succeeded',
          keyDecision: 'Fourth pre-change run — establishes the baseline the after-prompt run is measured against.',
          calls: [
            { name: 'read_ticket', args: { id: 'TKT-4471' }, output: '"I was charged twice for the same product today."' },
            { name: 'fetch_orders', args: { customer: 'c_9920', window: '24h' }, output: '2 charges · different IP (mobile + desktop) · different shipping addr' },
            { name: 'route_ticket', args: { queue: 'fraud-risk', reason: 'two charges, mismatched IP + shipping' }, output: 'routed \u2192 fraud-risk' },
          ] },
        { name: 'after-prompt-change', outcome: 'regressed',
          keyDecision: 'Classified from the ticket text alone — skipped the order-history lookup that surfaces the fraud signal.',
          calls: [
            { name: 'read_ticket', args: { id: 'TKT-4471' }, output: '"I was charged twice for the same product today."' },
            { name: 'route_ticket', args: { queue: 'billing', reason: 'duplicate charge complaint' }, output: 'routed \u2192 billing' },
          ] },
      ],
    },

    /* STORY 3 — additive · docs-generation agent ---------------------- */
    {
      id: 'api-docs-generation',
      kind: 'additive',
      hero: 'The agent started doing more, not worse.',
      question: 'The agent started doing something new — is that a problem?',
      setting:
        'A B2B platform runs a docs-generation agent. After a model upgrade the new docs look different. New capability, or new bug?',
      task: 'Generate API documentation for the newly-merged user-preferences service.',
      plain:
        'The first three runs read the service source and wrote a Markdown file with endpoints, schemas, and an overview. The two runs on the upgraded model did all of that and then queried the internal wiki to cross-link related concept pages. Nothing the old runs did stopped working — the new runs just do more. The path graph shows the extra steps as a branch that extends the known-good path rather than breaking it.',
      readsAs: 'The path grew, it didn\u2019t break. New capability, not a regression.',
      diffPair: [0, 3],
      triage: {
        classification: 'additive',
        summary:
          'The upgraded-model run reproduces the original read-source \u2192 write-docs path exactly, then appends two new steps: a wiki query and a cross-linking pass. The original behavior is intact; the new run is a strict superset.',
        cause:
          'The newer model has a stronger documentation habit and reaches for the wiki tool unprompted. Because nothing regressed and the additions are coherent, this is additive behavior worth promoting — not drift to suppress.',
      },
      traces: [
        { name: 'run-1', outcome: 'succeeded',
          keyDecision: 'Read the service source and wrote endpoints, schemas, and an overview to Markdown.',
          calls: [
            { name: 'read_source', args: { path: 'src/services/preferences' }, output: '4 endpoints · GET/PUT prefs · schema PreferenceSet' },
            { name: 'write_docs', args: { path: 'docs/api/preferences.md' }, output: 'wrote 142 lines · endpoints, schemas, overview' },
          ] },
        { name: 'run-2', outcome: 'succeeded',
          keyDecision: 'Reproduced the read-source then write-docs flow — old-model baseline.',
          calls: [
            { name: 'read_source', args: { path: 'src/services/preferences' }, output: '4 endpoints · GET/PUT prefs · schema PreferenceSet' },
            { name: 'write_docs', args: { path: 'docs/api/preferences.md' }, output: 'wrote 142 lines · endpoints, schemas, overview' },
          ] },
        { name: 'run-3', outcome: 'succeeded',
          keyDecision: 'Third old-model run — establishes the pre-upgrade baseline.',
          calls: [
            { name: 'read_source', args: { path: 'src/services/preferences' }, output: '4 endpoints · GET/PUT prefs · schema PreferenceSet' },
            { name: 'write_docs', args: { path: 'docs/api/preferences.md' }, output: 'wrote 142 lines · endpoints, schemas, overview' },
          ] },
        { name: 'run-4', outcome: 'additive',
          keyDecision: 'After writing the docs, queried the internal wiki and cross-linked related concept pages.',
          calls: [
            { name: 'read_source', args: { path: 'src/services/preferences' }, output: '4 endpoints · GET/PUT prefs · schema PreferenceSet' },
            { name: 'write_docs', args: { path: 'docs/api/preferences.md' }, output: 'wrote 142 lines · endpoints, schemas, overview' },
            { name: 'query_wiki', args: { topics: 'auth, user model, api style guide' }, output: '3 related wiki pages found' },
            { name: 'link_pages', args: { targets: 'auth, user-model, style-guide' }, output: 'added 3 cross-links to preferences.md' },
          ] },
        { name: 'run-5', outcome: 'additive',
          keyDecision: 'Repeated the docs-plus-wiki-links pattern — the new shape is stable once the model adopts it.',
          calls: [
            { name: 'read_source', args: { path: 'src/services/preferences' }, output: '4 endpoints · GET/PUT prefs · schema PreferenceSet' },
            { name: 'write_docs', args: { path: 'docs/api/preferences.md' }, output: 'wrote 142 lines · endpoints, schemas, overview' },
            { name: 'query_wiki', args: { topics: 'auth, user model, api style guide' }, output: '3 related wiki pages found' },
            { name: 'link_pages', args: { targets: 'auth, user-model, style-guide' }, output: 'added 3 cross-links to preferences.md' },
          ] },
      ],
    },
  ];

  /* ── canned extras for views 04 / 05 (attached by scenario id) ────────
     counterfactual: a single alternate continuation; the fork point follows
     whichever step the user picks. promptRewrite: before/after behavior
     populations (3 representative paths each) showing distribution shift. */
  const EXTRAS = {
    'pr-security-review': {
      counterfactual: {
        primary: 0,
        defaultStep: 1, // grep
        question: 'What if the comparison turned up nothing?',
        altTail: [
          { name: 'escalate_review', args: { to: 'security-team', reason: 'no comparable CSRF pattern found' }, output: 'escalated PR #4231 to security-team' },
        ],
        note: 'Change what the exploration step finds and the terminal action changes with it — an inline flag becomes an escalation. The shared prefix is identical; only the tail forks.',
      },
      promptRewrite: {
        suggested: 'Review the pull request for security issues. Before flagging, always grep the codebase for comparable patterns to confirm the finding.',
        beforeLabel: 'current prompt',
        afterLabel: 'rewritten prompt',
        before: [['read_diff', 'grep', 'comment_on_line'], ['read_diff', 'run_tests', 'comment_on_line'], ['read_diff', 'comment_on_line']],
        after: [['read_diff', 'grep', 'comment_on_line'], ['read_diff', 'grep', 'comment_on_line'], ['read_diff', 'grep', 'comment_on_line']],
        afterKind: 'succeeded',
        note: 'The rewrite pins the exploration step. Variance collapses — every run greps before flagging instead of three different strategies.',
      },
    },
    'support-triage-double-charge': {
      counterfactual: {
        primary: 0,
        defaultStep: 1, // fetch_orders
        question: 'What if the order history looked legitimate?',
        altTail: [
          { name: 'route_ticket', args: { queue: 'billing', reason: 'two charges, matching IP + shipping — legitimate duplicate' }, output: 'routed \u2192 billing' },
        ],
        note: 'Same tool, different evidence. If the lookup had shown matching IP and shipping, the agent routes the very same ticket to billing instead of fraud-risk — the fork is in the args, not the tool.',
      },
      promptRewrite: {
        suggested: 'Triage the incoming ticket. Always pull the customer\u2019s recent order history before classifying, and weigh IP and shipping mismatches.',
        beforeLabel: 'simplified prompt',
        afterLabel: 'rewritten prompt',
        before: [['read_ticket', 'route_ticket'], ['read_ticket', 'route_ticket'], ['read_ticket', 'fetch_orders', 'route_ticket']],
        after: [['read_ticket', 'fetch_orders', 'route_ticket'], ['read_ticket', 'fetch_orders', 'route_ticket'], ['read_ticket', 'fetch_orders', 'route_ticket']],
        afterKind: 'succeeded',
        note: 'Re-adding the lookup instruction restores the order-history step across the whole population — the regression is designed out, not just patched on one run.',
      },
    },
    'api-docs-generation': {
      counterfactual: {
        primary: 0,
        defaultStep: 1, // write_docs
        question: 'What if we asked it to enrich the docs?',
        altTail: [
          { name: 'query_wiki', args: { topics: 'auth, user model, api style guide' }, output: '3 related wiki pages found' },
          { name: 'link_pages', args: { targets: 'auth, user-model, style-guide' }, output: 'added 3 cross-links to preferences.md' },
        ],
        note: 'The counterfactual extends the known-good path rather than replacing it — the agent writes the docs, then queries the wiki and cross-links. Additive behavior, summoned on demand.',
      },
      promptRewrite: {
        suggested: 'Generate API documentation for the service. Also query the internal wiki and link related concept pages from the docs.',
        beforeLabel: 'old-model prompt',
        afterLabel: 'rewritten prompt',
        before: [['read_source', 'write_docs'], ['read_source', 'write_docs'], ['read_source', 'write_docs']],
        after: [['read_source', 'write_docs', 'query_wiki', 'link_pages'], ['read_source', 'write_docs', 'query_wiki', 'link_pages'], ['read_source', 'write_docs', 'query_wiki', 'link_pages']],
        afterKind: 'additive',
        note: 'The rewrite makes wiki cross-linking the default. The additive behavior, once an outlier on two runs, becomes the population norm.',
      },
    },
  };
  SCENARIOS.forEach(function (s) {
    const ex = EXTRAS[s.id];
    if (ex) { s.counterfactual = ex.counterfactual; s.promptRewrite = ex.promptRewrite; }
  });

  /* ── derivations ───────────────────────────────────────────────────── */

  // buildSteps: task prompt + alternating tool_call / tool_result chain.
  function buildSteps(task, calls) {
    const steps = [{ role: 'user', content: task }];
    calls.forEach(function (c) {
      steps.push({ role: 'assistant', toolCall: { name: c.name, args: c.args } });
      steps.push({ role: 'tool_result', toolResult: { name: c.name, output: c.output } });
    });
    return steps;
  }

  // outcomeMeta: badge label + semantic kind for a per-trace outcome.
  const OUTCOME = {
    succeeded: { label: 'succeeded', kind: 'ok' },
    variance:  { label: 'variance',  kind: 'warn' },
    regressed: { label: 'regressed', kind: 'bad' },
    additive:  { label: 'additive',  kind: 'novel' },
  };
  function outcomeMeta(o) { return OUTCOME[o] || { label: o || 'unknown', kind: 'neutral' }; }

  // trieFromRuns: merge any list of runs (each {calls:[{name,...}], outcome})
  // into a prefix-trie.
  //   node.count  = runs passing through  (drives node size + edge weight)
  //   node.kind   = 'main' | 'alt' | 'bad' | 'novel'  (drives color)
  // opts.key(call) decides what merges: default is tool NAME (so strategies
  // cluster, view 01). Counterfactual/rewrite views key by name+args so a
  // same-tool / different-args choice forks instead of silently merging.
  // The trie is a tree, so tidy-tree layout is exact.
  function trieFromRuns(runs, opts) {
    opts = opts || {};
    const keyOf = opts.key || function (c) { return c.name; };
    const labelOf = opts.label || function (c) { return c.name; };
    let uid = 0;
    const root = { id: 'n' + uid++, label: opts.rootLabel || 'task', tool: 'task', _key: '__root', count: 0, children: [], traces: [], depth: 0 };
    runs.forEach(function (trace, ti) {
      let cur = root;
      cur.count++;
      cur.traces.push(ti);
      trace.calls.forEach(function (call) {
        const k = keyOf(call);
        let child = cur.children.find(function (c) { return c._key === k; });
        if (!child) {
          child = { id: 'n' + uid++, label: labelOf(call), tool: call.name, _key: k, count: 0, children: [], traces: [], depth: cur.depth + 1, parent: cur };
          cur.children.push(child);
        }
        child.count++;
        child.traces.push(ti);
        cur = child;
      });
    });

    // flatten + classify
    const nodes = [];
    const edges = [];
    (function walk(n) {
      nodes.push(n);
      n.children.forEach(function (c) {
        edges.push({ id: n.id + '>' + c.id, from: n.id, to: c.id, weight: c.count, traces: c.traces });
        walk(c);
      });
    })(root);

    function kindFor(traceIdxs, parentBranching) {
      const outs = new Set(traceIdxs.map(function (i) { return runs[i].outcome; }));
      if (outs.size === 1 && (outs.has('regressed') || outs.has('bad'))) return 'bad';
      if (outs.size === 1 && (outs.has('additive') || outs.has('counterfactual'))) return 'novel';
      if (parentBranching) return 'alt';
      return 'main';
    }
    nodes.forEach(function (n) {
      if (!n.parent) { n.kind = 'main'; return; }
      const parentBranches = n.parent.children.length > 1;
      n.kind = kindFor(n.traces, parentBranches && n.count < n.parent.count);
    });
    edges.forEach(function (e) {
      const to = nodes.find(function (n) { return n.id === e.to; });
      e.kind = to.kind;
    });

    return { root: root, nodes: nodes, edges: edges };
  }

  function buildTrie(scenario) { return trieFromRuns(scenario.traces); }

  // ── view 04: counterfactual overlay ──────────────────────────────────
  // Build a 2-run trie: the original primary run + a synthetic branch that
  // shares the prefix up to the chosen step then forks into the canned tail.
  // Keyed by name+args so a same-tool/different-args choice is a visible fork.
  function cfKey(c) { return c.name + '|' + JSON.stringify(c.args || {}); }
  function buildCounterfactual(scenario, stepIndex, editedArgs) {
    const cf = scenario.counterfactual;
    const primary = scenario.traces[cf.primary];
    const pick = stepIndex == null ? cf.defaultStep : stepIndex;
    const original = { calls: primary.calls.slice(), outcome: 'succeeded' };
    // prefix [0..pick] with edited args applied to the picked call, then altTail
    const prefix = primary.calls.slice(0, pick + 1).map(function (c, i) {
      if (i === pick && editedArgs) return { name: c.name, args: editedArgs, output: c.output };
      return c;
    });
    const branch = { calls: prefix.concat(cf.altTail), outcome: 'counterfactual' };
    return trieFromRuns([original, branch], { key: cfKey });
  }

  // ── view 05: prompt-rewrite distribution ─────────────────────────────
  // each cluster entry is an array of tool-name strings → a 1-run trie.
  function miniTrie(names, outcome) {
    return trieFromRuns([{ calls: names.map(function (n) { return { name: n }; }), outcome: outcome || 'succeeded' }]);
  }

  function similarRuns(scenario, idx) {
    const scores = [0.94, 0.88, 0.71];
    const others = scenario.traces.map(function (_, i) { return i; }).filter(function (i) { return i !== idx; });
    return others.slice(0, 3).map(function (i, k) {
      return { idx: i, name: scenario.traces[i].name, outcome: scenario.traces[i].outcome, score: scores[k] };
    });
  }

  // alignSteps: Needleman-Wunsch on tool-call signatures → match / substitute /
  // insert / delete rows, exactly the op vocabulary the diff page uses.
  function stepSig(s) {
    if (s.role === 'user') return 'user';
    if (s.toolCall) return 'call:' + s.toolCall.name + ':' + JSON.stringify(s.toolCall.args);
    if (s.toolResult) return 'res:' + s.toolResult.name + ':' + s.toolResult.output;
    return s.role;
  }
  function stepToolName(s) {
    if (s.toolCall) return s.toolCall.name;
    if (s.toolResult) return s.toolResult.name;
    return null;
  }
  function alignSteps(A, B) {
    const n = A.length, m = B.length;
    const GAP = -1, MATCH = 2, SUB = -1;
    const dp = [];
    for (let i = 0; i <= n; i++) { dp.push(new Array(m + 1).fill(0)); dp[i][0] = i * GAP; }
    for (let j = 0; j <= m; j++) dp[0][j] = j * GAP;
    for (let i = 1; i <= n; i++) {
      for (let j = 1; j <= m; j++) {
        const same = stepSig(A[i - 1]) === stepSig(B[j - 1]);
        const diag = dp[i - 1][j - 1] + (same ? MATCH : SUB);
        dp[i][j] = Math.max(diag, dp[i - 1][j] + GAP, dp[i][j - 1] + GAP);
      }
    }
    const rows = [];
    let i = n, j = m;
    while (i > 0 || j > 0) {
      const same = i > 0 && j > 0 && stepSig(A[i - 1]) === stepSig(B[j - 1]);
      const diag = i > 0 && j > 0 ? dp[i - 1][j - 1] + (same ? MATCH : SUB) : -Infinity;
      const up = i > 0 ? dp[i - 1][j] + GAP : -Infinity;
      if (i > 0 && j > 0 && dp[i][j] === diag) {
        rows.push({
          op: same ? 'match' : (stepToolName(A[i - 1]) === stepToolName(B[j - 1]) || !same ? 'substitute' : 'substitute'),
          aStep: A[i - 1], bStep: B[j - 1], aIndex: i - 1, bIndex: j - 1,
        });
        i--; j--;
      } else if (i > 0 && dp[i][j] === up) {
        rows.push({ op: 'delete', aStep: A[i - 1], bStep: null, aIndex: i - 1, bIndex: null });
        i--;
      } else {
        rows.push({ op: 'insert', aStep: null, bStep: B[j - 1], aIndex: null, bIndex: j - 1 });
        j--;
      }
    }
    rows.reverse();
    // collapse match-on-substitute noise: keep substitute only when sigs differ
    rows.forEach(function (r) { if (r.op === 'substitute' && stepSig(r.aStep) === stepSig(r.bStep)) r.op = 'match'; });
    return rows;
  }

  function diffSummary(rows) {
    const s = { matches: 0, substitutions: 0, insertions: 0, deletions: 0 };
    rows.forEach(function (r) {
      if (r.op === 'match') s.matches++;
      else if (r.op === 'substitute') s.substitutions++;
      else if (r.op === 'insert') s.insertions++;
      else if (r.op === 'delete') s.deletions++;
    });
    s.distance = s.substitutions + s.insertions + s.deletions;
    return s;
  }

  function scenarioById(id) { return SCENARIOS.find(function (s) { return s.id === id; }); }

  window.AD = {
    scenarios: SCENARIOS,
    scenarioById: scenarioById,
    buildSteps: buildSteps,
    buildTrie: buildTrie,
    trieFromRuns: trieFromRuns,
    buildCounterfactual: buildCounterfactual,
    miniTrie: miniTrie,
    similarRuns: similarRuns,
    alignSteps: alignSteps,
    diffSummary: diffSummary,
    outcomeMeta: outcomeMeta,
  };
})();
