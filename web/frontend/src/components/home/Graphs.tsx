export const VarianceGraph = () => (
  <svg viewBox="0 0 320 180" fill="none">
    <circle cx="40" cy="90" r="6" fill="#22d3ee" />
    <circle cx="40" cy="90" r="12" fill="none" stroke="#22d3ee" strokeOpacity=".3" strokeWidth="1" />
    <text x="40" y="116" textAnchor="middle" fontSize="9" fill="#8a8f99" fontFamily="var(--font-mono)">
      prompt
    </text>

    {/* Strategy A — 3 runs, thick */}
    <path
      d="M46 88 Q 90 50 130 50 T 230 50"
      stroke="oklch(0.78 0.14 155)"
      strokeWidth="3.5"
      strokeOpacity=".9"
    />
    <circle cx="130" cy="50" r="4" fill="oklch(0.78 0.14 155)" />
    <circle cx="180" cy="50" r="4" fill="oklch(0.78 0.14 155)" />
    <circle cx="230" cy="50" r="4" fill="oklch(0.78 0.14 155)" />
    <circle cx="270" cy="50" r="7" fill="oklch(0.78 0.14 155)" />
    <text x="295" y="53" fontSize="10" fill="oklch(0.78 0.14 155)" fontFamily="var(--font-mono)">
      ×3
    </text>

    {/* Strategy B — 2 runs, medium */}
    <path
      d="M46 92 Q 90 100 130 100 T 230 100"
      stroke="#22d3ee"
      strokeWidth="2.2"
      strokeOpacity=".85"
    />
    <circle cx="130" cy="100" r="3.5" fill="#22d3ee" />
    <circle cx="180" cy="100" r="3.5" fill="#22d3ee" />
    <circle cx="230" cy="100" r="3.5" fill="#22d3ee" />
    <circle cx="270" cy="100" r="6" fill="#22d3ee" />
    <text x="295" y="103" fontSize="10" fill="#22d3ee" fontFamily="var(--font-mono)">
      ×2
    </text>

    {/* Strategy C — 1 run, thin, novel */}
    <path
      d="M46 95 Q 80 150 140 150 T 270 150"
      stroke="oklch(0.72 0.14 290)"
      strokeWidth="1.4"
      strokeOpacity=".85"
      strokeDasharray="4 3"
    />
    <circle cx="140" cy="150" r="3" fill="oklch(0.72 0.14 290)" />
    <circle cx="200" cy="150" r="3" fill="oklch(0.72 0.14 290)" />
    <circle cx="270" cy="150" r="5" fill="oklch(0.72 0.14 290)" />
    <text x="295" y="153" fontSize="10" fill="oklch(0.72 0.14 290)" fontFamily="var(--font-mono)">
      ×1
    </text>
  </svg>
);

export const RegressionGraph = () => {
  const baseY = 90;
  const hi = 30;
  const steps = 11;
  const xs = Array.from({ length: steps }, (_, i) => 28 + i * 26);
  const goodOffset = (i: number) =>
    i === 3 || i === 8 ? hi * 0.3 : i === 5 ? hi * 0.5 : 0;
  const bad = [8, 8, 10, 12, 15, 18, 24, 40, 55, 65, 72];

  return (
    <svg viewBox="0 0 320 180" fill="none">
      <line
        x1="20"
        x2="305"
        y1={baseY}
        y2={baseY}
        stroke="var(--border)"
        strokeWidth="1"
        strokeDasharray="2 3"
      />
      <text x="20" y={baseY - 6} fontSize="9" fill="var(--dim)" fontFamily="var(--font-mono)">
        baseline (v1.2 prompt)
      </text>

      <polyline
        points={xs.map((x, i) => `${x},${baseY - goodOffset(i)}`).join(' ')}
        stroke="oklch(0.78 0.14 155)"
        strokeWidth="2"
        fill="none"
        strokeLinejoin="round"
      />
      {xs.map((x, i) => (
        <circle key={`g${i}`} cx={x} cy={baseY - goodOffset(i)} r="3" fill="oklch(0.78 0.14 155)" />
      ))}

      <polyline
        points={xs.map((x, i) => `${x},${baseY + bad[i]}`).join(' ')}
        stroke="oklch(0.68 0.18 25)"
        strokeWidth="2"
        fill="none"
        strokeLinejoin="round"
      />
      {xs.map((x, i) => (
        <circle key={`r${i}`} cx={x} cy={baseY + bad[i]} r="3" fill="oklch(0.68 0.18 25)" />
      ))}

      <line
        x1={xs[6]}
        x2={xs[6]}
        y1={baseY - 30}
        y2={baseY + 45}
        stroke="var(--accent)"
        strokeWidth="1"
        strokeDasharray="2 2"
        opacity=".7"
      />
      <text x={xs[6] + 4} y={baseY - 22} fontSize="9" fill="var(--accent)" fontFamily="var(--font-mono)">
        divergence
      </text>

      <circle cx={xs[10]} cy={baseY} r="5" fill="oklch(0.78 0.14 155)" />
      <circle cx={xs[10]} cy={baseY + 72} r="5" fill="oklch(0.68 0.18 25)" />

      <text x="265" y={baseY + 12} fontSize="9" fill="oklch(0.78 0.14 155)" fontFamily="var(--font-mono)">
        v1.2 ok
      </text>
      <text x="245" y={baseY + 88} fontSize="9" fill="oklch(0.68 0.18 25)" fontFamily="var(--font-mono)">
        v1.3 hung
      </text>
    </svg>
  );
};

export const NovelGraph = () => (
  <svg viewBox="0 0 320 180" fill="none">
    <path
      d="M30 100 L 90 100 L 130 100 L 170 100 L 210 100 L 260 100 L 290 100"
      stroke="#22d3ee"
      strokeWidth="3"
      opacity=".9"
    />
    {[90, 130, 170, 210, 260].map((x, i) => (
      <circle key={`main${i}`} cx={x} cy={100} r={i === 4 ? 6 : 4} fill="#22d3ee" />
    ))}
    <text x="30" y="118" fontSize="9" fill="var(--muted)" fontFamily="var(--font-mono)">
      obvious path · ×4
    </text>

    <path
      d="M130 99 Q 145 75 175 55 T 235 35 Q 260 30 290 50"
      stroke="oklch(0.72 0.14 290)"
      strokeWidth="2"
      fill="none"
    />
    <circle cx="130" cy="100" r="6" fill="none" stroke="oklch(0.72 0.14 290)" strokeWidth="1.5" />
    {[
      [175, 55],
      [235, 35],
    ].map(([x, y], i) => (
      <circle key={`f${i}`} cx={x} cy={y} r="3.5" fill="oklch(0.72 0.14 290)" />
    ))}
    <circle cx={290} cy={50} r={6} fill="oklch(0.72 0.14 290)" />

    <text x="155" y="40" fontSize="10" fill="oklch(0.72 0.14 290)" fontFamily="var(--font-mono)">
      run #5 took a different turn
    </text>

    <text x="86" y="135" fontSize="8" fill="var(--dim)" fontFamily="var(--font-mono)">
      query_db
    </text>
    <text x="120" y="135" fontSize="8" fill="var(--dim)" fontFamily="var(--font-mono)">
      join_tables
    </text>
    <text x="160" y="135" fontSize="8" fill="var(--dim)" fontFamily="var(--font-mono)">
      summarize
    </text>
    <text x="205" y="135" fontSize="8" fill="var(--dim)" fontFamily="var(--font-mono)">
      format
    </text>
    <text x="245" y="135" fontSize="8" fill="var(--dim)" fontFamily="var(--font-mono)">
      send
    </text>
  </svg>
);
