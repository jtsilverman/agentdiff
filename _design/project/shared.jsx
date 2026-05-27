/* shared.jsx — Nav, Footer, Icons, Badges */

const { useState, useEffect, useRef, useMemo } = React;

// ─── Icons (1px-stroke line icons, never AI-slop SVG) ───
const Icon = {
  ArrowRight: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M3 7h8M7.5 3.5L11 7l-3.5 3.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  ChevronRight: (p) => (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" {...p}>
      <path d="M4.5 3l3 3-3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  ChevronDown: (p) => (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" {...p}>
      <path d="M3 4.5l3 3 3-3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  Branch: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <circle cx="3.5" cy="3" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <circle cx="3.5" cy="11" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <circle cx="10.5" cy="7" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <path d="M3.5 4.5v5M3.5 9.5C3.5 8 6 8.5 6 7c0-1.5 2.5-1 3-1" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
    </svg>
  ),
  Play: (p) => (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" {...p}>
      <path d="M3.5 2.5v7L9 6l-5.5-3.5z" fill="currentColor"/>
    </svg>
  ),
  Search: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <circle cx="6" cy="6" r="3.5" stroke="currentColor" strokeWidth="1.3"/>
      <path d="M9 9l2.5 2.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  Edit: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M2 12h3l6.5-6.5-3-3L2 9v3z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round"/>
      <path d="M8 4l3 3" stroke="currentColor" strokeWidth="1.2"/>
    </svg>
  ),
  Fork: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <circle cx="3.5" cy="3" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <circle cx="10.5" cy="3" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <circle cx="7" cy="11" r="1.4" stroke="currentColor" strokeWidth="1.2"/>
      <path d="M3.5 4.5C3.5 7 7 8 7 9.5M10.5 4.5c0 2.5-3.5 3.5-3.5 5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
    </svg>
  ),
  Diff: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M4 2v10M10 2v10" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
      <path d="M4 5h2M4 9h2M10 5h-2M10 9h-2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
    </svg>
  ),
  Close: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  Filter: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M2 3h10M3.5 7h7M5 11h4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/>
    </svg>
  ),
  Check: (p) => (
    <svg width="12" height="12" viewBox="0 0 12 12" fill="none" {...p}>
      <path d="M2.5 6l2.5 2.5L9.5 3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  Sparkles: (p) => (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...p}>
      <path d="M7 1.5v3M7 9.5v3M1.5 7h3M9.5 7h3M3 3l1.5 1.5M9.5 9.5L11 11M11 3L9.5 4.5M4.5 9.5L3 11" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
    </svg>
  ),
};

// ─── Brand Mark ───
const BrandMark = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 22 22" fill="none">
    {/* abstract path-graph: two nodes converging into a third */}
    <circle cx="4" cy="6" r="2.2" stroke="#22d3ee" strokeWidth="1.4"/>
    <circle cx="4" cy="16" r="2.2" stroke="#5b6068" strokeWidth="1.4"/>
    <circle cx="17" cy="11" r="2.2" fill="#22d3ee"/>
    <path d="M6 6.6 L15 10.4" stroke="#22d3ee" strokeWidth="1.2"/>
    <path d="M6 15.4 L15 11.6" stroke="#5b6068" strokeWidth="1.2"/>
  </svg>
);

// ─── Nav ───
const Nav = ({ active }) => (
  <nav className="nav">
    <div className="nav-inner">
      <a href="index.html" className="brand">
        <BrandMark/>
        <span className="brand-name">agentdiff <span className="tag">v0.4.1</span></span>
      </a>
      <div className="nav-links">
        <a href="index.html" className={"nav-link" + (active === "home" ? " active" : "")}>Home</a>
        <a href="about.html" className={"nav-link" + (active === "about" ? " active" : "")}>About</a>
        <a href="#" className="nav-link">Docs</a>
        <a href="#" className="nav-link">Changelog</a>
      </div>
      <div className="row">
        <a href="#" className="nav-link" style={{fontSize:13}}>Sign in</a>
        <a href="#" className="nav-cta">Start recording <Icon.ArrowRight/></a>
      </div>
    </div>
  </nav>
);

// ─── Footer ───
const Footer = () => (
  <footer className="site">
    <div className="container">
      <div className="row-12">
        <BrandMark size={18}/>
        <span className="mono" style={{fontSize:12, color:"var(--dim)"}}>agentdiff © 2026 — built for engineers who don't trust black boxes</span>
      </div>
      <div className="footer-links mono" style={{fontSize:12}}>
        <a href="#">github</a>
        <a href="#">docs</a>
        <a href="#">changelog</a>
        <a href="#">status</a>
      </div>
    </div>
  </footer>
);

// ─── Outcome Badge helper ───
const OutcomeBadge = ({ kind, label }) => {
  const labels = {
    ok: label || "converged",
    bad: label || "regressed",
    warn: label || "variance",
    novel: label || "novel strategy",
    neutral: label || "neutral",
    accent: label || "accent",
  };
  return (
    <span className={"badge " + kind}>
      <span className="dot"/>
      {labels[kind]}
    </span>
  );
};

Object.assign(window, { Icon, BrandMark, Nav, Footer, OutcomeBadge });
