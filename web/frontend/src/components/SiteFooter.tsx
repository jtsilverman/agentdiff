import { ArrowRight, BrandMark } from './Icons';

const EMAIL = 'jakesilverman.pro@gmail.com';
const LINKEDIN_URL = 'https://www.linkedin.com/in/jacob-silverman1/';
const GITHUB_URL = 'https://github.com/jtsilverman';

const LinkedInGlyph = () => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
    <rect x="1" y="1" width="12" height="12" rx="2" stroke="currentColor" strokeWidth="1.2" />
    <rect x="3.4" y="5.5" width="1.2" height="5" fill="currentColor" />
    <circle cx="4" cy="4" r="0.8" fill="currentColor" />
    <path
      d="M6.5 10.5V5.5h1.2v.7c.3-.5.8-.85 1.5-.85 1.1 0 1.8.75 1.8 2v3.15h-1.2v-3c0-.55-.3-.95-.85-.95-.5 0-.95.3-.95.95v3H6.5z"
      fill="currentColor"
    />
  </svg>
);

const GitHubGlyph = () => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
    <path
      d="M7 1.2c-3.2 0-5.8 2.6-5.8 5.8 0 2.6 1.7 4.8 4 5.5.3.05.4-.13.4-.28v-1c-1.6.35-2-.78-2-.78-.27-.7-.66-.88-.66-.88-.54-.37.04-.36.04-.36.6.05.92.62.92.62.54.93 1.4.66 1.75.5.05-.4.21-.66.38-.81-1.3-.15-2.66-.65-2.66-2.9 0-.64.23-1.16.6-1.57-.06-.15-.26-.74.06-1.55 0 0 .5-.16 1.62.6.47-.13.97-.2 1.47-.2s1 .07 1.47.2c1.12-.76 1.62-.6 1.62-.6.32.81.12 1.4.06 1.55.38.41.6.93.6 1.57 0 2.26-1.36 2.75-2.66 2.9.22.18.4.55.4 1.1v1.65c0 .16.1.34.4.28 2.3-.77 4-2.97 4-5.55 0-3.2-2.6-5.8-5.8-5.8z"
      fill="currentColor"
    />
  </svg>
);

export default function SiteFooter() {
  return (
    <footer className="ad-footer">
      {/* Introduce-me block — present on every page that mounts SiteFooter. */}
      <section className="introduce-me" aria-label="About the author">
        <div className="introduce-me-inner">
          <div className="introduce-me-identity">
            <div className="introduce-me-monogram" aria-hidden="true">
              <span>J.S.</span>
            </div>
            <div className="introduce-me-text">
              <div className="introduce-me-name-row">
                <span className="introduce-me-name">Jake Silverman</span>
                <span className="introduce-me-role-tag ad-mono">
                  <span className="introduce-me-role-dot" aria-hidden="true" />
                  available · forward-deployed eng
                </span>
              </div>
              <p className="introduce-me-blurb">
                Data engineer architecting enterprise data infrastructure across Azure Databricks
                and Microsoft Fabric for $100M+ duty drawback programs. Independent AI engineer
                building production agents, including an Executive Assistant deployed under her
                own identity inside a private equity office. North America winner of PwC&apos;s
                Transfer Pricing AI competition.
              </p>
            </div>
          </div>

          <div className="introduce-me-ctas">
            <a href={`mailto:${EMAIL}`} className="introduce-me-primary">
              Email Jake <ArrowRight />
            </a>
            <div className="introduce-me-elsewhere">
              <span className="ad-mono ad-dim">elsewhere:</span>
              <a
                href={LINKEDIN_URL}
                target="_blank"
                rel="noreferrer"
                className="introduce-me-link"
                aria-label="LinkedIn"
              >
                <LinkedInGlyph />
                <span>LinkedIn</span>
              </a>
              <span className="ad-dim ad-mono">·</span>
              <a
                href={GITHUB_URL}
                target="_blank"
                rel="noreferrer"
                className="introduce-me-link"
                aria-label="GitHub"
              >
                <GitHubGlyph />
                <span>GitHub</span>
              </a>
            </div>
          </div>
        </div>
      </section>

      <div className="ad-footer-inner">
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <BrandMark size={18} />
          <span className="ad-mono ad-dim" style={{ fontSize: 12 }}>
            agentdiff © 2026 — built for engineers who don&apos;t trust black boxes
          </span>
        </div>
        <div className="ad-footer-links">
          <a href="/changelog">changelog</a>
          <a href="/docs">docs</a>
          <a href="/about">about</a>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer">
            github
          </a>
        </div>
      </div>
    </footer>
  );
}
