'use client';

import { useEffect, useState, type CSSProperties } from 'react';

type Density = 'compact' | 'regular' | 'spacious';
type BgTone = 'cool' | 'neutral' | 'warm';

type TweakState = {
  density: Density;
  accent: string;
  bgTone: BgTone;
};

const STORAGE_KEY = 'agentdiff:tweaks';
const DEFAULTS: TweakState = {
  density: 'regular',
  accent: '#22d3ee',
  bgTone: 'cool',
};

const DENSITY_OPTIONS: Density[] = ['compact', 'regular', 'spacious'];
const BG_TONE_OPTIONS: BgTone[] = ['cool', 'neutral', 'warm'];
const ACCENT_OPTIONS = ['#22d3ee', '#4ade80', '#a78bfa', '#fb923c'];

const BG_TONES: Record<BgTone, { bg: string; bg2: string; surface: string; border: string }> = {
  cool: { bg: '#08090b', bg2: '#0c0e12', surface: '#0f1115', border: '#1d2026' },
  neutral: { bg: '#0a0a0a', bg2: '#101010', surface: '#141414', border: '#222222' },
  warm: { bg: '#0a0907', bg2: '#100e0a', surface: '#141210', border: '#221f1a' },
};

function readStored(): TweakState {
  if (typeof window === 'undefined') return DEFAULTS;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULTS;
    const parsed = JSON.parse(raw) as Partial<TweakState>;
    return {
      density: DENSITY_OPTIONS.includes(parsed.density as Density)
        ? (parsed.density as Density)
        : DEFAULTS.density,
      accent: typeof parsed.accent === 'string' ? parsed.accent : DEFAULTS.accent,
      bgTone: BG_TONE_OPTIONS.includes(parsed.bgTone as BgTone)
        ? (parsed.bgTone as BgTone)
        : DEFAULTS.bgTone,
    };
  } catch {
    return DEFAULTS;
  }
}

function applyTweaks(t: TweakState) {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;
  root.style.setProperty('--accent', t.accent);
  root.style.setProperty('--accent-faint', `${t.accent}14`);
  root.style.setProperty('--accent-glow', `${t.accent}30`);
  const bg = BG_TONES[t.bgTone];
  if (bg) {
    root.style.setProperty('--bg', bg.bg);
    root.style.setProperty('--bg-2', bg.bg2);
    root.style.setProperty('--surface', bg.surface);
    root.style.setProperty('--border', bg.border);
  }
  document.body.setAttribute('data-density', t.density);
}

export default function Tweaks() {
  const [t, setT] = useState<TweakState>(DEFAULTS);
  const [open, setOpen] = useState(false);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    const stored = readStored();
    setT(stored);
    setHydrated(true);
  }, []);

  useEffect(() => {
    if (!hydrated) return;
    applyTweaks(t);
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(t));
    } catch {
      // localStorage may be unavailable (e.g. Safari private mode); silently fall back to in-memory state.
    }
  }, [t, hydrated]);

  const set = <K extends keyof TweakState>(key: K, value: TweakState[K]) =>
    setT((prev) => ({ ...prev, [key]: value }));

  return (
    <>
      <button
        type="button"
        aria-label={open ? 'Close tweaks' : 'Open tweaks'}
        aria-expanded={open}
        className="twk-launcher"
        onClick={() => setOpen((v) => !v)}
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="4" cy="5" r="1.4" stroke="currentColor" strokeWidth="1.2" />
          <circle cx="12" cy="11" r="1.4" stroke="currentColor" strokeWidth="1.2" />
          <path
            d="M5.6 5h7.4M3 11h7.4"
            stroke="currentColor"
            strokeWidth="1.2"
            strokeLinecap="round"
          />
        </svg>
      </button>

      {open && (
        <div className="twk-panel" role="dialog" aria-label="Tweaks">
          <div className="twk-hd">
            <b>Tweaks</b>
            <button
              type="button"
              className="twk-x"
              aria-label="Close tweaks"
              onClick={() => setOpen(false)}
            >
              ×
            </button>
          </div>
          <div className="twk-body">
            <div className="twk-sect">Layout</div>
            <TweakRadio
              label="Card density"
              value={t.density}
              options={DENSITY_OPTIONS}
              onChange={(v) => set('density', v as Density)}
            />
            <div className="twk-sect">Color</div>
            <TweakColor
              label="Accent"
              value={t.accent}
              options={ACCENT_OPTIONS}
              onChange={(v) => set('accent', v)}
            />
            <TweakRadio
              label="Background"
              value={t.bgTone}
              options={BG_TONE_OPTIONS}
              onChange={(v) => set('bgTone', v as BgTone)}
            />
          </div>
        </div>
      )}
    </>
  );
}

function TweakRadio({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: readonly string[];
  onChange: (v: string) => void;
}) {
  const idx = Math.max(0, options.indexOf(value));
  const n = options.length;
  const thumbStyle: CSSProperties = {
    left: `calc(2px + ${idx} * (100% - 4px) / ${n})`,
    width: `calc((100% - 4px) / ${n})`,
  };
  return (
    <div className="twk-row">
      <div className="twk-lbl">
        <span>{label}</span>
      </div>
      <div role="radiogroup" className="twk-seg">
        <div className="twk-seg-thumb" style={thumbStyle} />
        {options.map((o) => (
          <button
            key={o}
            type="button"
            role="radio"
            aria-checked={o === value}
            onClick={() => onChange(o)}
          >
            {o}
          </button>
        ))}
      </div>
    </div>
  );
}

function TweakColor({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: readonly string[];
  onChange: (v: string) => void;
}) {
  return (
    <div className="twk-row">
      <div className="twk-lbl">
        <span>{label}</span>
      </div>
      <div className="twk-chips" role="radiogroup">
        {options.map((o) => {
          const on = o.toLowerCase() === value.toLowerCase();
          return (
            <button
              key={o}
              type="button"
              role="radio"
              aria-checked={on}
              aria-label={o}
              className="twk-chip"
              data-on={on ? '1' : '0'}
              style={{ background: o }}
              onClick={() => onChange(o)}
            />
          );
        })}
      </div>
    </div>
  );
}
