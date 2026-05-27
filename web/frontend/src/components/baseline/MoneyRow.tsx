'use client';

interface MoneyRowProps {
  onCounterfactual: () => void;
  onEditPrompt: () => void;
  onSimilar: () => void;
}

export default function MoneyRow({
  onCounterfactual,
  onEditPrompt,
  onSimilar,
}: MoneyRowProps) {
  return (
    <section className="bl-section">
      <div className="bl-section-head">
        <div>
          <span className="ad-eyebrow">analysis</span>
          <h2>Now ask "what if?"</h2>
        </div>
      </div>

      <div className="money-grid">
        <button type="button" className="money-card" onClick={onCounterfactual}>
          <div className="money-card-icon" aria-hidden>↳</div>
          <div className="money-card-body">
            <div className="money-card-title">Run counterfactual</div>
            <div className="money-card-sub">
              Pick a trace, pick a step, force a different decision and see what
              happens.
            </div>
          </div>
          <div className="money-card-cta">
            <span className="ad-mono ad-dim">⌘1</span>
            <span aria-hidden>→</span>
          </div>
        </button>

        <button type="button" className="money-card" onClick={onEditPrompt}>
          <div className="money-card-icon" aria-hidden>✎</div>
          <div className="money-card-body">
            <div className="money-card-title">Edit the prompt</div>
            <div className="money-card-sub">
              Rewrite the prompt for a step; agentdiff re-runs the agent and shows
              how the path shifted.
            </div>
          </div>
          <div className="money-card-cta">
            <span className="ad-mono ad-dim">⌘2</span>
            <span aria-hidden>→</span>
          </div>
        </button>

        <button type="button" className="money-card" onClick={onSimilar}>
          <div className="money-card-icon" aria-hidden>⌕</div>
          <div className="money-card-body">
            <div className="money-card-title">Find similar traces</div>
            <div className="money-card-sub">
              Pull other traces in your workspace with similar shape, failure mode,
              or toolchain.
            </div>
          </div>
          <div className="money-card-cta">
            <span className="ad-mono ad-dim">⌘3</span>
            <span aria-hidden>→</span>
          </div>
        </button>
      </div>
    </section>
  );
}
