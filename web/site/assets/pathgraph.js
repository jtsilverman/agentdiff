/* ============================================================================
   agentdiff — PathGraph (vanilla SVG)
   Renders a merged trie (AD.buildTrie) as a left-to-right network of tool
   calls. Nodes = tool calls, edge thickness = run count, color = branch kind
   (main / alt-variance / regression / additive). Tidy-tree layout is exact
   because a trie is a tree. Draw-in is staggered by depth, with divergent
   (regression) branches saved for last so the reveal reads as "baseline
   established, then a run diverged."
       AD.renderPathGraph(container, trie, opts)
   opts: { animate=true, loop=false, showCounts=true, onNode, compact }
   ========================================================================= */
(function () {
  'use strict';
  const SVGNS = 'http://www.w3.org/2000/svg';

  const KIND_STROKE = {
    main:  'var(--pg-main, rgba(244,241,236,0.45))',
    alt:   'var(--warn)',
    bad:   'var(--bad)',
    novel: 'var(--novel)',
  };
  const KIND_NODE = {
    main:  { border: 'var(--line-2)', glow: 'transparent' },
    alt:   { border: 'color-mix(in oklab, var(--warn) 55%, transparent)',  glow: 'var(--warn)' },
    bad:   { border: 'color-mix(in oklab, var(--bad) 65%, transparent)',   glow: 'var(--bad)' },
    novel: { border: 'color-mix(in oklab, var(--novel) 60%, transparent)', glow: 'var(--novel)' },
  };
  const KIND_PRIORITY = { main: 0, alt: 220, novel: 360, bad: 520 };

  function el(tag, attrs, parent) {
    const e = document.createElementNS(SVGNS, tag);
    if (attrs) for (const k in attrs) e.setAttribute(k, attrs[k]);
    if (parent) parent.appendChild(e);
    return e;
  }

  // ── tidy tree layout (LR): x by depth, y by leaf order ────────────────
  function layout(trie, opts) {
    const colGap = opts.compact ? 168 : 210;
    const rowGap = opts.compact ? 64 : 78;
    const nodeH = opts.compact ? 38 : 44;
    const padX = 24, padY = 36;

    let leafSlot = 0;
    (function assign(n) {
      if (n.children.length === 0) { n._y = leafSlot++; return; }
      n.children.forEach(assign);
      n._y = (n.children[0]._y + n.children[n.children.length - 1]._y) / 2;
    })(trie.root);

    const charW = opts.compact ? 7.4 : 8.0;
    trie.nodes.forEach(function (n) {
      const labelW = n.label.length * charW;
      n._w = Math.max(opts.compact ? 96 : 112, labelW + 52);
      n._h = nodeH;
      n._cx = padX + n.depth * colGap;          // left edge x
      n._cy = padY + n._y * rowGap;             // top edge y
    });

    let maxX = 0, maxY = 0;
    trie.nodes.forEach(function (n) {
      maxX = Math.max(maxX, n._cx + n._w);
      maxY = Math.max(maxY, n._cy + n._h);
    });
    return { w: maxX + padX, h: maxY + padY, nodeH: nodeH };
  }

  function nodeById(trie, id) { return trie.nodes.find(function (n) { return n.id === id; }); }

  AD.renderPathGraph = function (container, trie, opts) {
    opts = opts || {};
    const animate = opts.animate !== false;
    const showCounts = opts.showCounts !== false;

    const dims = layout(trie, opts);
    container.innerHTML = '';
    container.classList.add('pg-root');

    const svg = el('svg', {
      viewBox: '0 0 ' + dims.w + ' ' + dims.h,
      preserveAspectRatio: 'xMidYMid meet',
      class: 'pg-svg',
    }, container);
    svg.style.width = '100%';
    svg.style.height = '100%';
    svg.style.display = 'block';

    const gEdges = el('g', { class: 'pg-edges' }, svg);
    const gNodes = el('g', { class: 'pg-nodes' }, svg);

    const maxWeight = Math.max.apply(null, trie.edges.map(function (e) { return e.weight; }));

    // ── edges ──
    const edgeEls = trie.edges.map(function (e) {
      const a = nodeById(trie, e.from);
      const b = nodeById(trie, e.to);
      const x1 = a._cx + a._w, y1 = a._cy + a._h / 2;
      const x2 = b._cx, y2 = b._cy + b._h / 2;
      const mx = (x1 + x2) / 2;
      const d = 'M ' + x1 + ' ' + y1 + ' C ' + mx + ' ' + y1 + ', ' + mx + ' ' + y2 + ', ' + x2 + ' ' + y2;
      const sw = 1.4 + (e.weight / maxWeight) * (opts.compact ? 5 : 7);
      const path = el('path', {
        d: d, fill: 'none',
        stroke: KIND_STROKE[e.kind] || KIND_STROKE.main,
        'stroke-width': sw.toFixed(2),
        'stroke-linecap': 'round',
        pathLength: 1,
        class: 'pg-edge pg-edge-' + e.kind,
      }, gEdges);
      path.style.opacity = e.kind === 'main' ? 0.85 : 1;
      return { e: e, path: path, order: b.depth };
    });

    // ── nodes ──
    const nodeEls = trie.nodes.map(function (n) {
      const g = el('g', { class: 'pg-node pg-node-' + n.kind, 'data-tool': n.tool }, gNodes);
      g.style.transformOrigin = (n._cx + n._w / 2) + 'px ' + (n._cy + n._h / 2) + 'px';
      const palette = KIND_NODE[n.kind] || KIND_NODE.main;

      const isTask = n.tool === 'task';
      const rect = el('rect', {
        x: n._cx, y: n._cy, width: n._w, height: n._h, rx: 10,
        fill: isTask ? 'var(--fg)' : 'var(--bg-raise)',
        stroke: isTask ? 'var(--fg)' : palette.border,
        'stroke-width': 1.4,
        class: 'pg-rect',
      }, g);
      if (palette.glow !== 'transparent' && n.kind !== 'main') {
        rect.style.filter = 'drop-shadow(0 0 9px color-mix(in oklab, ' + palette.glow + ' 40%, transparent))';
      }

      const label = el('text', {
        x: n._cx + 14, y: n._cy + n._h / 2 + 0.5,
        'dominant-baseline': 'middle',
        class: 'pg-label',
        fill: isTask ? 'var(--bg)' : 'var(--fg)',
      }, g);
      label.textContent = isTask ? 'task' : n.label;
      label.style.fontFamily = 'var(--mono)';
      label.style.fontSize = (opts.compact ? 12 : 12.5) + 'px';
      label.style.fontWeight = isTask ? '600' : '500';

      // run-count badge
      if (showCounts && !isTask) {
        const bx = n._cx + n._w - 12;
        const cnt = el('text', {
          x: bx, y: n._cy + n._h / 2 + 0.5,
          'text-anchor': 'end',
          'dominant-baseline': 'middle',
          class: 'pg-count',
          fill: n.kind === 'main' ? 'var(--muted)' : (KIND_STROKE[n.kind]),
        }, g);
        cnt.textContent = '×' + n.count;
        cnt.style.fontFamily = 'var(--mono)';
        cnt.style.fontSize = '11px';
      }

      if (opts.onNode) {
        g.style.cursor = 'pointer';
        g.addEventListener('click', function () { opts.onNode(n); });
      }
      return { n: n, g: g, order: n.depth };
    });

    // ── animation ──
    function resetAnim() {
      edgeEls.forEach(function (o) {
        o.path.style.transition = 'none';
        o.path.style.strokeDasharray = '1';
        o.path.style.strokeDashoffset = '1';
      });
      nodeEls.forEach(function (o) {
        o.g.style.transition = 'none';
        o.g.style.opacity = '0';
        o.g.style.transform = 'scale(0.82)';
      });
    }
    function play() {
      const step = 230;
      // force reflow so transitions re-arm
      void svg.getBoundingClientRect();
      nodeEls.forEach(function (o) {
        const delay = o.n.depth * step + (KIND_PRIORITY[o.n.kind] || 0);
        o.g.style.transition = 'opacity 0.5s var(--ease), transform 0.55s var(--ease)';
        o.g.style.transitionDelay = delay + 'ms';
        o.g.style.opacity = '1';
        o.g.style.transform = 'scale(1)';
        if (o.n.kind === 'bad' || o.n.kind === 'novel') {
          o.g.classList.add('pg-pulse');
          o.g.style.animationDelay = (delay + 360) + 'ms';
        }
      });
      edgeEls.forEach(function (o) {
        const toNode = nodeById(trie, o.e.to);
        const delay = (toNode.depth - 0.5) * step + (KIND_PRIORITY[o.e.kind] || 0);
        o.path.style.transition = 'stroke-dashoffset 0.6s var(--ease)';
        o.path.style.transitionDelay = Math.max(0, delay) + 'ms';
        o.path.style.strokeDashoffset = '0';
      });
    }

    if (animate) {
      resetAnim();
      // expose a play hook + optional intersection trigger
      container._pgPlay = play;
      if (opts.playWhenVisible !== false && 'IntersectionObserver' in window) {
        let played = false;
        const io = new IntersectionObserver(function (entries) {
          entries.forEach(function (en) {
            if (en.isIntersecting && !played) { played = true; play(); if (!opts.loop) io.disconnect(); }
          });
        }, { threshold: 0.25 });
        io.observe(container);
      } else {
        requestAnimationFrame(play);
      }
      if (opts.loop) {
        const total = 4200;
        setInterval(function () { resetAnim(); requestAnimationFrame(function () { setTimeout(play, 80); }); }, total + 2600);
      }
    }

    return { svg: svg, play: play, reset: resetAnim, dims: dims };
  };
})();
