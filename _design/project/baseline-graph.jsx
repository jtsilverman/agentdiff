/* baseline-graph.jsx — interactive path graph */

const { useState: useStateG, useMemo: useMemoG } = React;

const EDGE_COLOR = {
  ok: "oklch(0.78 0.14 155)",
  accent: "#22d3ee",
  novel: "oklch(0.72 0.14 290)",
  bad: "oklch(0.68 0.18 25)",
  default: "#4b5560",
};

const NODE_FILL = {
  start: "#22d3ee",
  end: "#22d3ee",
  fail: "oklch(0.68 0.18 25)",
  warn: "oklch(0.74 0.16 60)",
  novelEnd: "oklch(0.72 0.14 290)",
};

// Build smooth bezier between two points
function curvePath(x1, y1, x2, y2) {
  const dx = (x2 - x1) * 0.5;
  return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

const PathGraph = ({ baselineId, highlightTrace = null, onNodeClick }) => {
  const graph = PATH_GRAPHS[baselineId];
  const [hoverNode, setHoverNode] = useStateG(null);
  const [hoverEdge, setHoverEdge] = useStateG(null);

  if (!graph) return null;

  const nodeById = useMemoG(() => Object.fromEntries(graph.nodes.map(n => [n.id, n])), [baselineId]);

  // VIEWBOX: tight to content
  const xs = graph.nodes.map(n => n.x);
  const ys = graph.nodes.map(n => n.y);
  const pad = 70;
  const minX = Math.min(...xs) - pad;
  const maxX = Math.max(...xs) + pad + 40; // room for labels
  const minY = Math.min(...ys) - pad;
  const maxY = Math.max(...ys) + pad;

  // edge stroke width by run count
  const maxCount = Math.max(...graph.edges.map(e => e.count));

  return (
    <div className="path-graph">
      <svg viewBox={`${minX} ${minY} ${maxX - minX} ${maxY - minY}`} className="path-graph-svg">
        <defs>
          {/* Soft glow for hovered edge */}
          <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="2" result="blur"/>
            <feMerge>
              <feMergeNode in="blur"/>
              <feMergeNode in="SourceGraphic"/>
            </feMerge>
          </filter>
        </defs>

        {/* faint grid */}
        <pattern id="pg-grid" width="40" height="40" patternUnits="userSpaceOnUse">
          <path d="M40 0 L0 0 0 40" fill="none" stroke="rgba(255,255,255,0.025)" strokeWidth="1"/>
        </pattern>
        <rect x={minX} y={minY} width={maxX - minX} height={maxY - minY} fill="url(#pg-grid)"/>

        {/* edges */}
        {graph.edges.map((e, i) => {
          const a = nodeById[e.from], b = nodeById[e.to];
          if (!a || !b) return null;
          const stroke = EDGE_COLOR[e.color || "default"];
          const width = 1.2 + (e.count / maxCount) * 5.5;
          const isHover = hoverEdge === i;
          return (
            <g key={"e"+i}>
              <path
                d={curvePath(a.x + 18, a.y, b.x - 18, b.y)}
                stroke={stroke}
                strokeWidth={width}
                fill="none"
                opacity={isHover ? 1 : 0.85}
                filter={isHover ? "url(#glow)" : ""}
                onMouseEnter={() => setHoverEdge(i)}
                onMouseLeave={() => setHoverEdge(null)}
                style={{cursor: "pointer", transition: "opacity .15s"}}
              />
              {/* run count label */}
              <text
                x={(a.x + b.x)/2}
                y={(a.y + b.y)/2 - 7}
                fontSize="9"
                fill={stroke}
                fontFamily="Geist Mono"
                textAnchor="middle"
                opacity={isHover ? 1 : (e.count > 1 ? 0.85 : 0)}
                style={{pointerEvents: "none"}}
              >
                ×{e.count}
              </text>
            </g>
          );
        })}

        {/* nodes */}
        {graph.nodes.map(n => {
          const r = 12 + Math.min(n.freq, 6) * 1.6;
          const fill = NODE_FILL[n.kind] || "#0f1115";
          const stroke = NODE_FILL[n.kind] || "#22d3ee";
          const isHover = hoverNode === n.id;
          return (
            <g
              key={n.id}
              onMouseEnter={() => setHoverNode(n.id)}
              onMouseLeave={() => setHoverNode(null)}
              onClick={() => onNodeClick && onNodeClick(n)}
              style={{cursor: "pointer"}}
            >
              {/* outer halo on hover */}
              {isHover && (
                <circle cx={n.x} cy={n.y} r={r + 8} fill={stroke} opacity=".15"/>
              )}
              <circle
                cx={n.x}
                cy={n.y}
                r={r}
                fill={fill}
                stroke={stroke}
                strokeWidth="1.5"
                opacity={n.kind ? 1 : 1}
              />
              {/* freq pip */}
              <text
                x={n.x}
                y={n.y + 3.5}
                fontSize="10"
                fontFamily="Geist Mono"
                fill={n.kind ? "#051518" : "var(--fg)"}
                textAnchor="middle"
                style={{pointerEvents: "none"}}
              >
                {n.freq}
              </text>
              {/* label below */}
              <text
                x={n.x}
                y={n.y + r + 13}
                fontSize="10"
                fontFamily="Geist Mono"
                fill={isHover ? "var(--fg)" : "var(--muted)"}
                textAnchor="middle"
                style={{pointerEvents: "none"}}
              >
                {n.label}
              </text>
            </g>
          );
        })}
      </svg>

      {/* Floating info chip */}
      {hoverNode && nodeById[hoverNode] && (
        <div className="pg-info">
          <span className="mono dim">node</span>
          <span className="mono">{nodeById[hoverNode].label}</span>
          <span className="mono dim">·</span>
          <span className="mono">{nodeById[hoverNode].freq} runs passed through</span>
        </div>
      )}
    </div>
  );
};

Object.assign(window, { PathGraph });
