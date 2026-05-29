/* agentdiff — abstract monochrome feature-card glyphs.
   Simple primitives only (dots, lines, bars) that echo each feature's shape. */
(function () {
  'use strict';
  const S = 'http://www.w3.org/2000/svg';
  const stroke = 'rgba(244,241,236,0.4)';
  const faint = 'rgba(244,241,236,0.16)';

  function frame() {
    const svg = document.createElementNS(S, 'svg');
    svg.setAttribute('viewBox', '0 0 240 92');
    svg.setAttribute('preserveAspectRatio', 'xMidYMid meet');
    svg.style.width = '100%'; svg.style.height = '100%';
    return svg;
  }
  function add(svg, tag, attrs) {
    const e = document.createElementNS(S, tag);
    for (const k in attrs) e.setAttribute(k, attrs[k]);
    svg.appendChild(e); return e;
  }

  const builders = {
    // path graph: a node fanning into three branches
    path: function (svg) {
      const nodes = [[34,46],[120,24],[120,46],[120,68],[206,46]];
      [[0,1],[0,2],[0,3],[2,4]].forEach(function (p, i) {
        add(svg, 'line', { x1: nodes[p[0]][0], y1: nodes[p[0]][1], x2: nodes[p[1]][0], y2: nodes[p[1]][1], stroke: i === 3 ? stroke : faint, 'stroke-width': i === 3 ? 2.4 : 1.4, 'stroke-linecap': 'round' });
      });
      nodes.forEach(function (n, i) {
        add(svg, 'circle', { cx: n[0], cy: n[1], r: i === 0 ? 5.5 : 4, fill: i === 0 ? stroke : 'var(--bg)', stroke: stroke, 'stroke-width': 1.4 });
      });
    },
    // diff: two columns of aligned bars, one row offset
    diff: function (svg) {
      const rows = [16, 35, 54, 73];
      rows.forEach(function (y, i) {
        const sub = i === 2;
        add(svg, 'rect', { x: 26, y: y, width: 78, height: 12, rx: 3, fill: 'none', stroke: sub ? stroke : faint, 'stroke-width': 1.4 });
        add(svg, 'rect', { x: 136, y: y, width: 78, height: 12, rx: 3, fill: sub ? 'rgba(244,241,236,0.06)' : 'none', stroke: sub ? stroke : faint, 'stroke-width': 1.4 });
      });
      add(svg, 'line', { x1: 120, y1: 12, x2: 120, y2: 80, stroke: faint, 'stroke-width': 1, 'stroke-dasharray': '3 4' });
    },
    // scrubber: a track with a handle and step ticks
    scrub: function (svg) {
      add(svg, 'line', { x1: 28, y1: 46, x2: 212, y2: 46, stroke: faint, 'stroke-width': 2, 'stroke-linecap': 'round' });
      [28, 74, 120, 166, 212].forEach(function (x, i) {
        add(svg, 'circle', { cx: x, cy: 46, r: i === 2 ? 7 : 3, fill: i === 2 ? stroke : 'var(--bg)', stroke: stroke, 'stroke-width': 1.4 });
      });
      [20, 36, 52].forEach(function (y, i) {
        add(svg, 'rect', { x: 150, y: y, width: 52 - i * 12, height: 6, rx: 3, fill: faint });
      });
    },
  };

  document.querySelectorAll('.glyph[data-glyph]').forEach(function (host) {
    const kind = host.getAttribute('data-glyph');
    if (!builders[kind]) return;
    const svg = frame();
    builders[kind](svg);
    host.appendChild(svg);
  });
})();
