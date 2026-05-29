/* ============================================================================
   agentdiff — demo controller. Switches between the three stories and drives
   three live views: path graph, single-run transcript+scrubber, and a
   step-aligned diff with AI triage. Pure DOM, reads window.AD.
   ========================================================================= */
(function () {
  'use strict';
  const $ = function (s, r) { return (r || document).querySelector(s); };
  function esc(s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  }

  const KIND_WORD = { variance: 'variance', regression: 'regression', additive: 'additive' };

  const state = {
    scn: null,
    run: 0,        // index into scenario.traces for the inspector
    step: 0,       // active step in inspector
    diffA: 0,
    diffB: 1,
    opFilter: 'all',
    graph: null,
  };

  /* ── scenario tabs ─────────────────────────────────────────────────── */
  function renderTabs() {
    const host = $('#scnTabs');
    host.innerHTML = '';
    AD.scenarios.forEach(function (s) {
      const b = document.createElement('button');
      b.className = 'scn-tab' + (s === state.scn ? ' active' : '');
      b.setAttribute('data-kind', s.kind);
      b.innerHTML = '<span class="scn-tab-k">story · ' + KIND_WORD[s.kind] + '</span>' +
        '<span class="scn-tab-t">' + esc(s.question) + '</span>';
      b.addEventListener('click', function () { selectScenario(s); });
      host.appendChild(b);
    });
  }

  /* ── story framing ─────────────────────────────────────────────────── */
  function renderStoryFrame() {
    const s = state.scn;
    const host = $('#storyFrame');
    const meta = AD.outcomeMeta(s.kind === 'regression' ? 'regressed' : s.kind === 'additive' ? 'additive' : 'variance');
    host.innerHTML =
      '<div class="story-frame">' +
        '<div>' +
          '<span class="eyebrow">the question</span>' +
          '<h2>' + esc(s.question) + '</h2>' +
          '<p class="setting">' + esc(s.setting) + '</p>' +
        '</div>' +
        '<div class="story-task">' +
          '<span class="lbl">the task · run ' + s.traces.length + '\u00d7</span>' +
          '<p class="task">' + esc(s.task) + '</p>' +
          '<div class="runs">' + s.traces.map(function (t) {
            const m = AD.outcomeMeta(t.outcome);
            return '<span class="badge ' + m.kind + '"><span class="dot"></span>' + esc(m.label) + '</span>';
          }).join('') + '</div>' +
        '</div>' +
      '</div>';
  }

  /* ── path graph ────────────────────────────────────────────────────── */
  function renderGraph() {
    const s = state.scn;
    const trie = AD.buildTrie(s);
    $('#graphBarTitle').innerHTML = 'agentdiff &nbsp;/&nbsp; baseline <b>' + esc(s.id) + '</b> &nbsp;·&nbsp; ' + s.traces.length + ' runs';
    $('#graphNote').textContent = graphNoteFor(s.kind);
    state.graph = AD.renderPathGraph($('#demoGraph'), trie, { compact: false, onNode: null, playWhenVisible: true });
    // legend by kind
    const L = $('#graphLegend');
    const items = [['shared path', 'var(--pg-main,rgba(244,241,236,0.5))']];
    if (s.kind === 'variance') items.push(['strategy branch', 'var(--warn)']);
    if (s.kind === 'regression') items.push(['regressed run', 'var(--bad)']);
    if (s.kind === 'additive') items.push(['added steps', 'var(--novel)']);
    L.innerHTML = items.map(function (i) {
      return '<span><i style="background:' + i[1] + '"></i> ' + i[0] + '</span>';
    }).join('') + '<span class="dim">thickness = runs through the step</span>';
  }
  function graphNoteFor(kind) {
    if (kind === 'variance') return 'Three strategies fan out from one task and all converge on success — variation inside known-good behavior.';
    if (kind === 'regression') return 'Four runs share one path. One run skips a step and the outcome changes — that divergence is the regression.';
    return 'The new runs reproduce the known-good path, then extend it with extra steps. The shape grew; nothing broke.';
  }

  /* ── inspector (transcript + scrubber) ─────────────────────────────── */
  function renderRunPicker() {
    const host = $('#runPicker');
    host.innerHTML = '';
    state.scn.traces.forEach(function (t, i) {
      const m = AD.outcomeMeta(t.outcome);
      const b = document.createElement('button');
      b.className = 'run-chip' + (i === state.run ? ' active' : '');
      b.setAttribute('data-kind', m.kind);
      b.innerHTML = '<span class="dot"></span>' + esc(t.name);
      b.addEventListener('click', function () { state.run = i; state.step = 0; renderRunPicker(); renderInspector(); });
      host.appendChild(b);
    });
  }

  function stepLabel(step) {
    if (step.role === 'user') return { role: 'user', label: 'task prompt', mono: false };
    if (step.toolCall) return { role: 'assistant', label: step.toolCall.name + '()', mono: true };
    if (step.toolResult) return { role: 'tool_result', label: step.toolResult.name + ' \u2192', mono: true };
    return { role: step.role, label: step.role, mono: false };
  }

  function renderInspector() {
    const s = state.scn;
    const trace = s.traces[state.run];
    const steps = AD.buildSteps(s.task, trace.calls);
    state.step = Math.min(state.step, steps.length - 1);

    $('#railMeta').textContent = trace.name + ' · ' + steps.length + ' steps';
    $('#railRunName').textContent = trace.name;

    // key decision
    $('#keyDecision').innerHTML = '<span class="lbl">key decision</span><p>' + esc(trace.keyDecision) + '</p>';

    // rail
    const rail = $('#railList');
    rail.innerHTML = '';
    steps.forEach(function (step, i) {
      const info = stepLabel(step);
      const b = document.createElement('button');
      b.className = 'rail-step' + (i === state.step ? ' active' : '');
      b.innerHTML = '<span class="rail-step-n">' + String(i + 1).padStart(2, '0') + '</span>' +
        '<span class="rail-step-role ' + info.role + '">' + info.role.replace('_', ' ') + '</span>' +
        '<span class="rail-step-label' + (info.mono ? ' mono' : '') + '">' + esc(info.label) + '</span>';
      b.addEventListener('click', function () { setStep(i); });
      rail.appendChild(b);
    });

    // scrubber
    const range = $('#scrubRange');
    range.max = steps.length - 1;
    range.value = state.step;

    // transcript
    const tx = $('#txList');
    tx.innerHTML = '';
    steps.forEach(function (step, i) {
      tx.appendChild(renderTxStep(step, i));
    });
    syncActive();
  }

  function renderTxStep(step, i) {
    const div = document.createElement('div');
    div.className = 'tx-step' + (i === state.step ? ' active' : '');
    div.setAttribute('data-i', i);
    let head = '';
    if (step.role === 'user') {
      head = '<span class="tx-role user">user</span><span class="dstep-idx">step ' + String(i + 1).padStart(2, '0') + '</span>';
    } else if (step.toolCall) {
      head = '<span class="tx-role assistant">assistant</span><span class="tx-toolname">' + esc(step.toolCall.name) + '()</span><span class="dstep-idx">step ' + String(i + 1).padStart(2, '0') + '</span>';
    } else if (step.toolResult) {
      head = '<span class="tx-role">tool result</span><span class="tx-toolname">' + esc(step.toolResult.name) + ' \u2192</span><span class="badge ok" style="padding:2px 8px;font-size:10px;"><span class="dot"></span>ok</span>';
    }
    let body = '';
    if (step.content) body = '<p class="tx-content">' + esc(step.content) + '</p>';
    if (step.toolCall && step.toolCall.args) {
      body = '<div class="tx-args">' + Object.keys(step.toolCall.args).map(function (k) {
        return '<div class="tx-arg"><span class="k">' + esc(k) + '</span><span class="v">' + esc(String(step.toolCall.args[k])) + '</span></div>';
      }).join('') + '</div>';
    }
    if (step.toolResult && step.toolResult.output) {
      body = '<pre class="tx-output">' + esc(step.toolResult.output) + '</pre>';
    }
    div.innerHTML = '<div class="tx-step-head">' + head + '</div>' + body;
    div.addEventListener('click', function () { setStep(i); });
    return div;
  }

  function setStep(i) {
    state.step = i;
    $('#scrubRange').value = i;
    syncActive();
  }
  function syncActive() {
    document.querySelectorAll('#railList .rail-step').forEach(function (el, i) {
      el.classList.toggle('active', i === state.step);
    });
    document.querySelectorAll('#txList .tx-step').forEach(function (el) {
      const i = +el.getAttribute('data-i');
      el.classList.toggle('active', i === state.step);
    });
    const active = document.querySelector('#railList .rail-step.active');
    if (active) {
      const rail = $('#railList');
      const top = active.offsetTop - rail.clientHeight / 2 + active.clientHeight / 2;
      rail.scrollTo({ top: top, behavior: 'smooth' });
    }
    const txa = document.querySelector('#txList .tx-step.active');
    if (txa) {
      const tx = $('.transcript');
      const top = txa.offsetTop - 90;
      tx.scrollTo({ top: top, behavior: 'smooth' });
    }
  }

  function wireScrubber() {
    $('#scrubRange').addEventListener('input', function (e) { setStep(+e.target.value); });
    $('#scrubPrev').addEventListener('click', function () { if (state.step > 0) setStep(state.step - 1); });
    $('#scrubNext').addEventListener('click', function () {
      const max = +$('#scrubRange').max;
      if (state.step < max) setStep(state.step + 1);
    });
  }

  /* ── view 02 affordances: find similar + promote ───────────────────── */
  function wireSimilar() {
    const link = $('#findSimilar');
    const pop = $('#similarPop');
    link.addEventListener('click', function (e) {
      e.stopPropagation();
      const open = pop.classList.contains('open');
      if (open) { closeSimilar(); return; }
      renderSimilar();
      pop.classList.add('open');
      link.setAttribute('aria-expanded', 'true');
    });
    pop.addEventListener('click', function (e) { e.stopPropagation(); });
  }
  function closeSimilar() {
    $('#similarPop').classList.remove('open');
    $('#findSimilar').setAttribute('aria-expanded', 'false');
  }
  function renderSimilar() {
    const sims = AD.similarRuns(state.scn, state.run);
    const pop = $('#similarPop');
    pop.innerHTML = '<div class="similar-pop-head">most similar runs · cosine on tool-path</div>' +
      sims.map(function (s) {
        const m = AD.outcomeMeta(s.outcome);
        return '<button class="similar-item" data-i="' + s.idx + '">' +
          '<span class="dot" style="background:var(--' + m.kind + ')"></span>' +
          '<span class="sname">' + esc(s.name) + '</span>' +
          '<span class="sscore">' + s.score.toFixed(2) + '</span></button>';
      }).join('');
    pop.querySelectorAll('.similar-item').forEach(function (b) {
      b.addEventListener('click', function () {
        state.run = +b.getAttribute('data-i');
        state.step = 0;
        closeSimilar();
        renderRunPicker();
        renderInspector();
      });
    });
  }
  function wirePromote() {
    const btn = $('#promoteBtn');
    const conf = $('#promoteConfirm');
    btn.addEventListener('click', function () {
      btn.classList.add('done');
      btn.textContent = '✓ promoted';
      conf.textContent = 'This run is now a baseline · the next upload gets diffed against it.';
      conf.classList.add('show');
      clearTimeout(btn._t);
      btn._t = setTimeout(function () {
        btn.classList.remove('done');
        btn.textContent = 'Promote to baseline';
        conf.classList.remove('show');
      }, 3200);
    });
  }

  /* ── view 04: counterfactual replay ────────────────────────────────── */
  function renderCounterfactual() {
    const s = state.scn;
    const cf = s.counterfactual;
    state.cfStep = cf.defaultStep;
    $('#cfNote').textContent = 'Pick a step from the primary run, change its inputs, and replay. The branch shares everything up to that step, then forks on the change — the interaction shape, not a live re-run.';
    renderCfSteps();
    renderCfEdit();
    renderCfGraph(false);
    $('#cfCaption').innerHTML = '<b>' + esc(cf.question) + '</b> Change the inputs above and hit “What if?” to fork a counterfactual branch.';
  }
  function renderCfSteps() {
    const s = state.scn;
    const primary = s.traces[s.counterfactual.primary];
    const host = $('#cfSteps');
    host.innerHTML = '';
    primary.calls.forEach(function (c, i) {
      const b = document.createElement('button');
      b.className = 'cf-step' + (i === state.cfStep ? ' active' : '');
      b.innerHTML = '<span class="cf-step-n">' + String(i + 1).padStart(2, '0') + '</span><span class="cf-step-tool">' + esc(c.name) + '()</span>';
      b.addEventListener('click', function () {
        state.cfStep = i;
        renderCfSteps();
        renderCfEdit();
        renderCfGraph(false);
        $('#cfCaption').innerHTML = '<b>' + esc(s.counterfactual.question) + '</b> Edit the inputs and hit “What if?”.';
      });
      host.appendChild(b);
    });
  }
  function renderCfEdit() {
    const s = state.scn;
    const primary = s.traces[s.counterfactual.primary];
    const call = primary.calls[state.cfStep];
    const host = $('#cfEdit');
    host.innerHTML = Object.keys(call.args || {}).map(function (k) {
      return '<div class="cf-field"><label>' + esc(k) + '</label>' +
        '<input class="cf-input" data-k="' + esc(k) + '" value="' + esc(String(call.args[k])) + '" /></div>';
    }).join('') || '<p class="cf-caption" style="margin:0;">This step takes no editable inputs — replay forks on the step itself.</p>';
  }
  function readCfArgs() {
    const args = {};
    $('#cfEdit').querySelectorAll('.cf-input').forEach(function (inp) {
      args[inp.getAttribute('data-k')] = inp.value;
    });
    return args;
  }
  function renderCfGraph(withBranch) {
    const s = state.scn;
    let trie;
    if (withBranch) {
      trie = AD.buildCounterfactual(s, state.cfStep, readCfArgs());
    } else {
      const primary = s.traces[s.counterfactual.primary];
      trie = AD.trieFromRuns([{ calls: primary.calls, outcome: 'succeeded' }], { key: function (c) { return c.name + '|' + JSON.stringify(c.args || {}); } });
    }
    AD.renderPathGraph($('#cfGraph'), trie, { compact: true, animate: withBranch, playWhenVisible: true });
  }
  function wireCounterfactual() {
    $('#cfRun').addEventListener('click', function () {
      const s = state.scn;
      const cf = s.counterfactual;
      const edited = readCfArgs();
      renderCfGraph(true);
      const call = s.traces[cf.primary].calls[state.cfStep];
      const changed = Object.keys(edited).filter(function (k) { return String(edited[k]) !== String(call.args[k]); });
      const changeStr = changed.length
        ? 'Changed <b>' + esc(call.name) + '</b> · ' + changed.map(function (k) { return esc(k) + ' → “' + esc(edited[k]) + '”'; }).join(', ') + '. '
        : 'Replayed from <b>' + esc(call.name) + '()</b>. ';
      $('#cfCaption').innerHTML = changeStr + esc(cf.note);
    });
  }

  /* ── view 05: edit-prompt rewrite ──────────────────────────────────── */
  function renderPromptRewrite() {
    const s = state.scn;
    const pr = s.promptRewrite;
    $('#prText').value = s.task;
    $('#prBeforeLabel').textContent = pr.beforeLabel;
    $('#prAfterLabel').textContent = pr.afterLabel;
    $('#prBeforeCount').textContent = s.traces.length + ' runs';
    $('#prAfterCount').textContent = s.traces.length + ' runs';
    $('#prNote').innerHTML = '<b>Rewrite the prompt, then re-run.</b> The right-hand cluster is dimmed until you do — it shows where the population lands after the change.';
    renderThumbs('#prBefore', pr.before, 'succeeded');
    renderThumbs('#prAfter', pr.after, pr.afterKind);
    $('#prAfter').classList.add('masked');
    $('#prShift').classList.remove('live');
  }
  function renderThumbs(sel, runs, outcome) {
    const host = $(sel);
    host.innerHTML = '';
    runs.forEach(function (names) {
      const d = document.createElement('div');
      d.className = 'pr-thumb';
      host.appendChild(d);
      AD.renderPathGraph(d, AD.miniTrie(names, outcome), { compact: true, animate: false, showCounts: false });
    });
  }
  function wirePromptRewrite() {
    $('#prSuggest').addEventListener('click', function () {
      $('#prText').value = state.scn.promptRewrite.suggested;
      $('#prText').focus();
    });
    $('#prRun').addEventListener('click', function () {
      $('#prAfter').classList.remove('masked');
      $('#prShift').classList.add('live');
      $('#prNote').innerHTML = esc(state.scn.promptRewrite.note);
    });
  }

  /* ── diff + triage ─────────────────────────────────────────────────── */
  function renderDiffSides() {
    renderDiffSide('A', state.diffA, state.diffB, '#diffSideA', function (i) { state.diffA = i; renderDiff(); });
    renderDiffSide('B', state.diffB, state.diffA, '#diffSideB', function (i) { state.diffB = i; renderDiff(); });
  }
  function renderDiffSide(side, idx, otherIdx, sel, onPick) {
    const s = state.scn;
    const t = s.traces[idx];
    const m = AD.outcomeMeta(t.outcome);
    const host = $(sel);
    const steps = AD.buildSteps(s.task, t.calls);
    host.innerHTML =
      '<div class="run-menu">' +
        '<div class="side-lbl">side ' + side + ' <span class="switch-link">switch</span></div>' +
        '<div class="side-name">' + esc(t.name) + '</div>' +
        '<div class="side-meta">' +
          '<span class="badge ' + m.kind + '"><span class="dot"></span>' + esc(m.label) + '</span>' +
          '<span class="badge neutral">' + steps.length + ' steps</span>' +
        '</div>' +
        '<div class="run-menu-pop">' + s.traces.map(function (tt, i) {
          const mm = AD.outcomeMeta(tt.outcome);
          const dis = i === otherIdx;
          return '<button class="run-menu-item" data-i="' + i + '"' + (dis ? ' disabled' : '') + '>' +
            '<span class="dot" style="background:var(--' + mm.kind + ')"></span>' + esc(tt.name) +
            (i === idx ? ' <span class="mono dim" style="margin-left:auto;font-size:11px;">current</span>' : '') +
            (dis ? ' <span class="mono dim" style="margin-left:auto;font-size:11px;">other side</span>' : '') +
            '</button>';
        }).join('') + '</div>' +
      '</div>';
    const menu = host.querySelector('.run-menu');
    host.querySelector('.switch-link').addEventListener('click', function (e) {
      e.stopPropagation();
      document.querySelectorAll('.run-menu.open').forEach(function (m2) { if (m2 !== menu) m2.classList.remove('open'); });
      menu.classList.toggle('open');
    });
    host.querySelectorAll('.run-menu-item').forEach(function (b) {
      b.addEventListener('click', function () {
        if (b.disabled) return;
        menu.classList.remove('open');
        onPick(+b.getAttribute('data-i'));
      });
    });
  }

  function renderDiff() {
    const s = state.scn;
    if (state.diffA === state.diffB) state.diffB = (state.diffA + 1) % s.traces.length;
    renderDiffSides();

    const A = AD.buildSteps(s.task, s.traces[state.diffA].calls);
    const B = AD.buildSteps(s.task, s.traces[state.diffB].calls);
    const rows = AD.alignSteps(A, B);
    const sum = AD.diffSummary(rows);

    // summary pills
    $('#diffSummary').innerHTML =
      pill('ok', sum.matches, 'matches') +
      pill('warn', sum.substitutions, 'substitutions') +
      pill('novel', sum.insertions, 'insertions · in B') +
      pill('bad', sum.deletions, 'deletions · in A') +
      pill('dist', sum.distance, 'edit distance');

    // triage
    renderTriage();

    // align count
    const visible = rows.filter(function (r) { return state.opFilter === 'all' || r.op === state.opFilter; });
    $('#alignCount').textContent = 'alignment · ' + rows.length + ' rows' +
      (visible.length !== rows.length ? '  ·  ' + visible.length + ' shown' : '');

    // rows
    const host = $('#diffRows');
    if (visible.length === 0) {
      host.innerHTML = '<div style="padding:40px;text-align:center;color:var(--muted);font-family:var(--mono);font-size:13px;">no rows match this filter.</div>';
      return;
    }
    host.innerHTML = visible.map(renderDiffRow).join('');
  }

  function pill(kind, num, label) {
    return '<div class="pill ' + kind + '"><div class="num">' + num + '</div><span class="plbl">' + label + '</span></div>';
  }

  const OP_SYM = { match: '=', substitute: '~', insert: '+', delete: '\u2212' };
  function renderDiffRow(r) {
    return '<div class="diff-row op-' + r.op + '">' +
      '<div class="diff-row-side a">' + stepCell(r.aStep, r.aIndex) + '</div>' +
      '<div class="diff-marker"><div class="op-badge">' + OP_SYM[r.op] + '</div><span class="op-name">' + r.op + '</span></div>' +
      '<div class="diff-row-side b">' + stepCell(r.bStep, r.bIndex) + '</div>' +
      '</div>';
  }
  function stepCell(step, idx) {
    if (!step) return '<span class="dstep-ghost">\u2014 not in this run \u2014</span>';
    const n = idx === null || idx === undefined ? '··' : String(idx + 1).padStart(2, '0');
    let head = '<span class="dstep-idx">' + n + '</span>';
    let body = '';
    if (step.role === 'user') {
      head += '<span class="dstep-role">user</span>';
      body = '<p class="dstep-content">' + esc(step.content) + '</p>';
    } else if (step.toolCall) {
      head += '<span class="dstep-role">assistant</span><span class="dstep-tool">' + esc(step.toolCall.name) + '()</span>';
      body = '<div class="dstep-args">' + Object.keys(step.toolCall.args).map(function (k) {
        return '<div class="dstep-arg"><span class="k">' + esc(k) + '</span><span class="v">' + esc(String(step.toolCall.args[k])) + '</span></div>';
      }).join('') + '</div>';
    } else if (step.toolResult) {
      head += '<span class="dstep-role">tool \u2192</span><span class="dstep-tool">' + esc(step.toolResult.name) + '</span>';
      body = '<pre class="dstep-out">' + esc(step.toolResult.output) + '</pre>';
    }
    return '<div class="dstep-head">' + head + '</div>' + body;
  }

  function renderTriage() {
    const tri = state.scn.triage;
    const kind = tri.classification === 'regression' ? 'bad' : tri.classification === 'additive' ? 'novel' : 'warn';
    $('#triageBlock').className = 'triage ' + kind;
    $('#triageBlock').innerHTML =
      '<div class="triage-head">' +
        '<span class="eyebrow">ai triage · GET /api/diff/.../triage</span>' +
        '<span class="badge ' + kind + '"><span class="dot"></span>' + esc(tri.classification) + '</span>' +
      '</div>' +
      '<p class="triage-summary">' + esc(tri.summary) + '</p>' +
      '<div class="triage-cause"><span class="lbl">likely cause</span><p>' + esc(tri.cause) + '</p></div>';
  }

  function wireOpFilter() {
    $('#opFilter').querySelectorAll('.seg-btn').forEach(function (b) {
      b.addEventListener('click', function () {
        state.opFilter = b.getAttribute('data-op');
        $('#opFilter').querySelectorAll('.seg-btn').forEach(function (x) { x.classList.toggle('active', x === b); });
        renderDiff();
      });
    });
  }

  /* ── scenario select ───────────────────────────────────────────────── */
  function selectScenario(s) {
    state.scn = s;
    state.run = 0;
    state.step = 0;
    state.diffA = s.diffPair[0];
    state.diffB = s.diffPair[1];
    state.opFilter = 'all';
    $('#opFilter').querySelectorAll('.seg-btn').forEach(function (x) { x.classList.toggle('active', x.getAttribute('data-op') === 'all'); });
    renderTabs();
    renderStoryFrame();
    renderGraph();
    renderRunPicker();
    renderInspector();
    renderDiff();
    renderCounterfactual();
    renderPromptRewrite();
    closeSimilar();
    if (location.hash.slice(1) !== s.id) history.replaceState(null, '', '#' + s.id);
  }

  /* ── init ──────────────────────────────────────────────────────────── */
  document.addEventListener('click', function () {
    document.querySelectorAll('.run-menu.open').forEach(function (m) { m.classList.remove('open'); });
    closeSimilar();
  });
  wireScrubber();
  wireOpFilter();
  wireSimilar();
  wirePromote();
  wireCounterfactual();
  wirePromptRewrite();
  const initial = AD.scenarioById(location.hash.slice(1)) || AD.scenarios[1]; // default: regression story
  selectScenario(initial);
})();
