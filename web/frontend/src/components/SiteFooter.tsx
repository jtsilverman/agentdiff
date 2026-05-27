import { BrandMark } from './Icons';

export default function SiteFooter() {
  return (
    <footer className="ad-footer">
      <div className="ad-footer-inner">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <BrandMark size={18} />
          <span className="ad-mono ad-dim" style={{ fontSize: 12 }}>
            agentdiff © 2026 — built for engineers who don&apos;t trust black boxes
          </span>
        </div>
        <div className="ad-footer-links">
          <a href="https://github.com" target="_blank" rel="noreferrer">github</a>
          <a href="/about">docs</a>
          <a href="/about">changelog</a>
        </div>
      </div>
    </footer>
  );
}
