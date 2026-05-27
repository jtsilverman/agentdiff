/* tweaks-app.jsx — Tweaks panel for agentdiff (works on every page) */

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "density": "regular",
  "accent": "#22d3ee",
  "bgTone": "cool",
  "showStatusDots": true
}/*EDITMODE-END*/;

function applyTweaks(t) {
  const root = document.documentElement;

  // Accent
  root.style.setProperty("--accent", t.accent);
  // Derive accent-2 (lighter) and accent-faint from chosen color
  root.style.setProperty("--accent-faint", t.accent + "14"); // rough alpha
  root.style.setProperty("--accent-glow", t.accent + "30");

  // Background tone
  const bg = {
    cool:    { bg: "#08090b", bg2: "#0c0e12", surface: "#0f1115", border: "#1d2026" },
    neutral: { bg: "#0a0a0a", bg2: "#101010", surface: "#141414", border: "#222222" },
    warm:    { bg: "#0a0907", bg2: "#100e0a", surface: "#141210", border: "#221f1a" },
  }[t.bgTone] || {};
  if (bg.bg) {
    root.style.setProperty("--bg", bg.bg);
    root.style.setProperty("--bg-2", bg.bg2);
    root.style.setProperty("--surface", bg.surface);
    root.style.setProperty("--border", bg.border);
  }

  // Density
  document.body.setAttribute("data-density", t.density);
}

function TweaksApp() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);

  React.useEffect(() => { applyTweaks(t); }, [t.accent, t.bgTone, t.density]);

  return (
    <TweaksPanel>
      <TweakSection label="Layout"/>
      <TweakRadio
        label="Card density"
        value={t.density}
        options={["compact", "regular", "spacious"]}
        onChange={(v) => setTweak("density", v)}
      />
      <TweakSection label="Color"/>
      <TweakColor
        label="Accent"
        value={t.accent}
        options={["#22d3ee", "#4ade80", "#a78bfa", "#fb923c"]}
        onChange={(v) => setTweak("accent", v)}
      />
      <TweakRadio
        label="Background"
        value={t.bgTone}
        options={["cool", "neutral", "warm"]}
        onChange={(v) => setTweak("bgTone", v)}
      />
    </TweaksPanel>
  );
}

// Mount when document is ready
(function mount() {
  let root = document.getElementById("tweaks-root");
  if (!root) {
    root = document.createElement("div");
    root.id = "tweaks-root";
    document.body.appendChild(root);
  }
  ReactDOM.createRoot(root).render(<TweaksApp/>);
})();
