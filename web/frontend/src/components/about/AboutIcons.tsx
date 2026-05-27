import { Fragment } from 'react';

export type ConceptIconKey =
  | 'trace'
  | 'baseline'
  | 'strategy'
  | 'drift'
  | 'pathGraph'
  | 'overlay'
  | 'counterfactual'
  | 'editPrompt';

export const ConceptIcon: Record<ConceptIconKey, () => JSX.Element> = {
  trace: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="3" fill="#22d3ee" />
      <circle cx="22" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3" />
      <circle cx="36" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3" />
      <circle cx="50" cy="20" r="2.5" fill="none" stroke="#22d3ee" strokeWidth="1.3" />
      <circle cx="58" cy="20" r="3" fill="#22d3ee" />
      <path d="M11 20h8M25 20h8M39 20h8M53 20h2" stroke="#22d3ee" strokeWidth="1.3" />
    </svg>
  ),
  baseline: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee" />
      <path d="M10 20 Q 32 8 56 8" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <path d="M10 20 Q 32 14 56 14" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <path d="M10 20 Q 32 20 56 20" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <path d="M10 20 Q 32 26 56 26" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <path d="M10 20 Q 32 32 56 32" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <circle cx="56" cy="8" r="2" fill="oklch(0.78 0.14 155)" />
      <circle cx="56" cy="14" r="2" fill="oklch(0.78 0.14 155)" />
      <circle cx="56" cy="20" r="2" fill="oklch(0.78 0.14 155)" />
      <circle cx="56" cy="26" r="2" fill="oklch(0.78 0.14 155)" />
      <circle cx="56" cy="32" r="2" fill="oklch(0.78 0.14 155)" />
    </svg>
  ),
  strategy: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee" />
      <path d="M10 20 L 26 12 L 42 12 L 56 12" stroke="oklch(0.78 0.14 155)" strokeWidth="2.5" />
      <path d="M10 20 L 26 20 L 42 20 L 56 20" stroke="#22d3ee" strokeWidth="1.5" />
      <path d="M10 20 L 26 28 L 42 28 L 56 28" stroke="oklch(0.72 0.14 290)" strokeWidth="1" />
      <circle cx="56" cy="12" r="2.5" fill="oklch(0.78 0.14 155)" />
      <circle cx="56" cy="20" r="2" fill="#22d3ee" />
      <circle cx="56" cy="28" r="1.5" fill="oklch(0.72 0.14 290)" />
    </svg>
  ),
  drift: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <line x1="6" x2="58" y1="20" y2="20" stroke="var(--border-hi)" strokeWidth="1" strokeDasharray="2 2" />
      <path d="M6 20 L 22 20 L 32 16 L 42 22 L 58 18" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" fill="none" />
      <path d="M6 20 L 22 20 L 32 26 L 42 34 L 58 34" stroke="oklch(0.68 0.18 25)" strokeWidth="1.5" fill="none" />
      <circle cx="32" cy="20" r="3" fill="none" stroke="#22d3ee" strokeWidth="1.3" />
    </svg>
  ),
  pathGraph: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <circle cx="8" cy="20" r="2.5" fill="#22d3ee" />
      <circle cx="24" cy="12" r="2" fill="#22d3ee" />
      <circle cx="24" cy="28" r="2" fill="#22d3ee" />
      <circle cx="42" cy="12" r="2.5" fill="#22d3ee" />
      <circle cx="42" cy="28" r="2" fill="#22d3ee" />
      <circle cx="58" cy="20" r="2.5" fill="#22d3ee" />
      <path d="M10 19 L 22 13M10 21 L 22 27M26 12 L 40 12M26 28 L 40 28M44 13 L 56 19M44 27 L 56 21" stroke="#22d3ee" strokeWidth="1" />
    </svg>
  ),
  overlay: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <path d="M6 28 Q 16 16 32 20 T 58 16" stroke="oklch(0.78 0.14 155)" strokeWidth="2" fill="none" />
      <path d="M6 28 Q 16 22 32 24 T 58 28" stroke="#22d3ee" strokeWidth="2" fill="none" opacity=".8" />
      <circle cx="32" cy="22" r="6" fill="none" stroke="#22d3ee" strokeWidth="1" strokeDasharray="2 2" />
    </svg>
  ),
  counterfactual: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <path d="M6 20 L 24 20" stroke="#22d3ee" strokeWidth="1.8" />
      <circle cx="24" cy="20" r="3.5" fill="none" stroke="#22d3ee" strokeWidth="1.5" />
      <path d="M27 18 L 58 10" stroke="oklch(0.78 0.14 155)" strokeWidth="1.5" strokeDasharray="3 2" />
      <path d="M27 22 L 58 30" stroke="oklch(0.72 0.14 290)" strokeWidth="1.5" strokeDasharray="3 2" />
      <circle cx="58" cy="10" r="2" fill="oklch(0.78 0.14 155)" />
      <circle cx="58" cy="30" r="2" fill="oklch(0.72 0.14 290)" />
    </svg>
  ),
  editPrompt: () => (
    <svg viewBox="0 0 64 40" fill="none">
      <rect x="6" y="10" width="20" height="20" rx="2" fill="none" stroke="var(--border-hi)" strokeWidth="1" />
      <path d="M9 16h14M9 20h14M9 24h10" stroke="var(--border-hi)" strokeWidth="1" />
      <path d="M30 20 L 38 20" stroke="#22d3ee" strokeWidth="1.5" />
      <path d="M35 17 L 38 20 L 35 23" stroke="#22d3ee" strokeWidth="1.3" fill="none" />
      <rect x="42" y="10" width="20" height="20" rx="2" fill="var(--accent-faint)" stroke="#22d3ee" strokeWidth="1" />
      <path d="M45 16h14M45 20h14M45 24h10" stroke="#22d3ee" strokeWidth="1" />
    </svg>
  ),
};

export type TourMiniKey = 'home' | 'baseline' | 'trace' | 'counterfactual' | 'editPrompt';

export const TourMini: Record<TourMiniKey, () => JSX.Element> = {
  home: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6" />
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1" />
      <circle cx="14" cy="11" r="3" fill="#22d3ee" />
      <rect x="22" y="8" width="38" height="6" rx="1.5" fill="var(--muted)" />
      <rect x="20" y="46" width="240" height="14" rx="2" fill="var(--fg-2)" />
      <rect x="20" y="64" width="180" height="14" rx="2" fill="#22d3ee" />
      <rect x="20" y="90" width="220" height="5" rx="1" fill="var(--border-hi)" />
      <rect x="20" y="100" width="200" height="5" rx="1" fill="var(--border-hi)" />
      <g>
        <rect x="20" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)" />
        <rect x="130" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)" />
        <rect x="240" y="135" width="100" height="65" rx="4" fill="var(--surface)" stroke="var(--border)" />
        <circle cx="32" cy="146" r="2" fill="oklch(0.74 0.16 60)" />
        <circle cx="142" cy="146" r="2" fill="oklch(0.68 0.18 25)" />
        <circle cx="252" cy="146" r="2" fill="oklch(0.72 0.14 290)" />
        <path d="M30 175 Q 50 160 75 160 T 105 160" stroke="oklch(0.78 0.14 155)" strokeWidth="1" />
        <path d="M140 165 L 165 165 L 175 175 L 200 175 L 215 165" stroke="oklch(0.68 0.18 25)" strokeWidth="1" fill="none" />
        <path d="M250 175 L 280 175 L 305 165 L 325 175" stroke="oklch(0.72 0.14 290)" strokeWidth="1" fill="none" />
      </g>
    </svg>
  ),
  baseline: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6" />
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1" />
      <rect x="14" y="32" width="60" height="4" rx="1" fill="var(--dim)" />
      <rect x="14" y="46" width="180" height="10" rx="2" fill="var(--fg-2)" />
      <rect x="14" y="68" width="220" height="32" rx="3" fill="var(--surface)" stroke="var(--border)" />
      <rect x="22" y="76" width="200" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="22" y="84" width="180" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="22" y="92" width="140" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="245" y="46" width="100" height="54" rx="3" fill="var(--surface)" stroke="var(--border)" />
      <line x1="295" y1="46" x2="295" y2="100" stroke="var(--border)" strokeWidth="1" />
      <line x1="245" y1="73" x2="345" y2="73" stroke="var(--border)" strokeWidth="1" />
      <rect x="14" y="115" width="220" height="95" rx="3" fill="var(--surface)" stroke="var(--border)" />
      {[0, 1, 2, 3].map((i) => (
        <Fragment key={i}>
          <rect x="22" y={125 + i * 20} width="40" height="4" rx="1" fill="var(--fg-2)" />
          <circle
            cx={75}
            cy={127 + i * 20}
            r="2.5"
            fill={['oklch(0.78 0.14 155)', '#22d3ee', 'oklch(0.78 0.14 155)', 'oklch(0.68 0.18 25)'][i]}
          />
          <rect x="90" y={125 + i * 20} width="135" height="4" rx="1" fill="var(--border-hi)" />
        </Fragment>
      ))}
      <rect x="245" y="115" width="100" height="95" rx="3" fill="var(--bg)" stroke="var(--border)" />
      <circle cx="255" cy="160" r="3" fill="#22d3ee" />
      <circle cx="278" cy="145" r="2" fill="#22d3ee" />
      <circle cx="278" cy="175" r="2" fill="#22d3ee" />
      <circle cx="310" cy="145" r="2.5" fill="oklch(0.78 0.14 155)" />
      <circle cx="310" cy="175" r="2" fill="oklch(0.72 0.14 290)" />
      <circle cx="335" cy="160" r="3" fill="#22d3ee" />
      <path
        d="M258 159 L 276 146 M 258 161 L 276 174 M 280 145 L 308 145 M 280 175 L 308 175 M 312 146 L 333 159 M 312 174 L 333 161"
        stroke="#22d3ee"
        strokeWidth="0.8"
      />
    </svg>
  ),
  trace: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6" />
      <line x1="0" x2="360" y1="22" y2="22" stroke="var(--border)" strokeWidth="1" />
      <rect x="0" y="22" width="100" height="198" fill="var(--bg-2)" />
      <line x1="100" x2="100" y1="22" y2="220" stroke="var(--border)" strokeWidth="1" />
      {[0, 1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
        <Fragment key={i}>
          <circle cx="14" cy={40 + i * 18} r="2" fill={i === 4 ? '#22d3ee' : 'var(--muted)'} />
          <rect
            x="22"
            y={38 + i * 18}
            width={60 - i * 3}
            height="3"
            rx="1"
            fill={i === 4 ? 'var(--fg)' : 'var(--border-hi)'}
          />
        </Fragment>
      ))}
      <rect x="110" y="36" width="60" height="6" rx="1.5" fill="var(--fg-2)" />
      <rect x="110" y="50" width="150" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="110" y="68" width="240" height="60" rx="3" fill="var(--surface)" stroke="var(--border)" />
      <rect x="118" y="76" width="40" height="4" rx="1" fill="#22d3ee" />
      <rect x="118" y="88" width="220" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="118" y="96" width="200" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="118" y="104" width="180" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="118" y="112" width="160" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="110" y="138" width="240" height="36" rx="3" fill="var(--bg-2)" stroke="#22d3ee" strokeWidth="0.8" />
      <rect x="118" y="146" width="50" height="4" rx="1" fill="#22d3ee" />
      <rect x="118" y="158" width="220" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="110" y="184" width="240" height="28" rx="3" fill="var(--surface)" stroke="var(--border)" />
      <rect x="118" y="192" width="40" height="4" rx="1" fill="oklch(0.78 0.14 155)" />
      <rect x="118" y="202" width="200" height="3" rx="1" fill="var(--border-hi)" />
    </svg>
  ),
  counterfactual: () => (
    <svg viewBox="0 0 360 220" fill="none">
      <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6" />
      <rect x="20" y="20" width="320" height="180" rx="6" fill="var(--surface)" stroke="var(--border-hi)" />
      <line x1="20" x2="340" y1="50" y2="50" stroke="var(--border)" strokeWidth="1" />
      <rect x="32" y="32" width="100" height="6" rx="1.5" fill="var(--fg)" />
      <circle cx="328" cy="36" r="6" fill="none" stroke="var(--muted)" strokeWidth="1" />
      <rect x="32" y="62" width="140" height="80" rx="3" fill="var(--bg-2)" stroke="var(--border)" />
      <rect x="40" y="70" width="80" height="3" rx="1" fill="var(--dim)" />
      {[0, 1, 2, 3].map((i) => (
        <rect
          key={i}
          x="40"
          y={82 + i * 14}
          width="124"
          height="9"
          rx="2"
          fill={i === 1 ? 'var(--accent-faint)' : 'var(--surface)'}
          stroke={i === 1 ? '#22d3ee' : 'var(--border)'}
          strokeWidth="0.6"
        />
      ))}
      <rect x="188" y="62" width="140" height="80" rx="3" fill="var(--bg-2)" stroke="var(--border)" />
      <rect x="196" y="70" width="90" height="3" rx="1" fill="var(--dim)" />
      {[0, 1, 2, 3].map((i) => (
        <rect
          key={i}
          x="196"
          y={82 + i * 14}
          width="124"
          height="9"
          rx="2"
          fill={i === 2 ? 'var(--accent-faint)' : 'var(--surface)'}
          stroke={i === 2 ? '#22d3ee' : 'var(--border)'}
          strokeWidth="0.6"
        />
      ))}
      <rect x="32" y="152" width="296" height="38" rx="3" fill="var(--bg)" stroke="var(--border-hi)" />
      <rect x="40" y="160" width="80" height="4" rx="1" fill="oklch(0.72 0.14 290)" />
      <rect x="40" y="172" width="280" height="3" rx="1" fill="var(--border-hi)" />
      <rect x="40" y="180" width="220" height="3" rx="1" fill="var(--border-hi)" />
    </svg>
  ),
  editPrompt: () => {
    const bars: [string, number][] = [
      ['oklch(0.78 0.14 155)', 200],
      ['#22d3ee', 110],
      ['oklch(0.72 0.14 290)', 60],
    ];
    return (
      <svg viewBox="0 0 360 220" fill="none">
        <rect x="0" y="0" width="360" height="220" fill="#0a0c10" rx="6" />
        <rect x="20" y="20" width="320" height="180" rx="6" fill="var(--surface)" stroke="var(--border-hi)" />
        <line x1="20" x2="340" y1="50" y2="50" stroke="var(--border)" strokeWidth="1" />
        <rect x="32" y="32" width="100" height="6" rx="1.5" fill="var(--fg)" />
        <rect x="32" y="62" width="296" height="60" rx="3" fill="var(--bg)" stroke="var(--border)" />
        <rect x="40" y="72" width="270" height="3" rx="1" fill="var(--fg-2)" />
        <rect x="40" y="80" width="280" height="3" rx="1" fill="var(--fg-2)" />
        <rect x="40" y="88" width="200" height="3" rx="1" fill="#22d3ee" />
        <rect x="40" y="96" width="240" height="3" rx="1" fill="var(--fg-2)" />
        <rect x="40" y="104" width="180" height="3" rx="1" fill="var(--fg-2)" />
        <rect x="32" y="132" width="296" height="60" rx="3" fill="var(--bg-2)" stroke="var(--border)" />
        {bars.map(([c, w], i) => (
          <Fragment key={i}>
            <rect x="40" y={144 + i * 14} width="80" height="3" rx="1" fill={c} />
            <rect x="130" y={143 + i * 14} width="186" height="6" rx="2" fill="var(--surface)" />
            <rect x="130" y={143 + i * 14} width={w * 0.95} height="6" rx="2" fill={c} />
          </Fragment>
        ))}
      </svg>
    );
  },
};
