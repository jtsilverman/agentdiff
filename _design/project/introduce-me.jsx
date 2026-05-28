/* introduce-me.jsx — showcase page documenting the persistent "introduce-me" footer surface.
   The actual surface is rendered on every page by shared.jsx's Footer.
   This page exists to call attention to the placement choice and rationale.
*/

const { useState: useStateIM } = React;

const PLACEMENTS = [
  {
    id: "footer",
    label: "Footer block",
    chosen: true,
    pros: [
      "Always visible at the bottom of every page",
      "Discovered AFTER the product, not before — \"there's a real person behind this\" moment lands at the right time",
      "Same dark-tool aesthetic as the rest of the site, treats credentials with the same hi-fi care",
      "Doesn't compete for attention with the actual product content",
    ],
    cons: [
      "Below the fold by default — visitor has to scroll to reach it",
    ],
  },
  {
    id: "nav-chip",
    label: "Nav-resident chip",
    chosen: false,
    pros: [
      "Above the fold on first paint",
    ],
    cons: [
      "Crowds the nav, which already carries six product links",
      "Reads as marketing-y on a developer tool",
      "Forces every page to lead with \"hire me\" before the visitor knows what \"this\" is",
    ],
  },
  {
    id: "floating-badge",
    label: "Floating bottom-corner badge",
    chosen: false,
    pros: [
      "Persistently visible without needing to scroll",
    ],
    cons: [
      "Reads as a chat-widget or intercom popup — wrong aesthetic for a developer tool",
      "Brief asks for not-a-popup, and a hover-to-expand floating element is effectively a popup",
      "Steals attention from path graphs, diffs, and other dense visuals where every pixel matters",
    ],
  },
];

const PAGES = [
  { id: "home", label: "Home", href: "index.html" },
  { id: "baseline", label: "Baseline detail", href: "baseline.html?id=auth-migration" },
  { id: "about", label: "About", href: "about.html" },
  { id: "docs", label: "Docs", href: "docs.html" },
  { id: "diff", label: "Diff", href: "diff.html?a=t-7a4f&b=t-2c81" },
  { id: "trace", label: "Trace detail", href: "trace-detail.html?id=t-7a4f" },
  { id: "traces", label: "Traces", href: "traces.html" },
  { id: "changelog", label: "Changelog", href: "changelog.html" },
];

const IntroduceMe = () => {
  return (
    <>
      <Nav active="about"/>

      <main>
        {/* Hero */}
        <header className="tr-hero im-hero" data-screen-label="01 Introduce-me hero">
          <div className="container">
            <div className="bl-breadcrumb mono">
              <a href="index.html" className="bl-crumb">home</a>
              <span className="dim">/</span>
              <span className="bl-crumb-current">introduce-me</span>
            </div>
            <span className="eyebrow">design note · placement decision</span>
            <h1 className="tr-hero-title" style={{marginTop: 16}}>The introduce-me surface lives in the footer.</h1>
            <p className="tr-hero-sub">
              This whole site is Jake Silverman's FDE portfolio. The brief asked for a "hire me" surface present enough to register without competing with page content. Three placements were considered. The footer was chosen.
            </p>
          </div>
        </header>

        <div className="container">
          {/* The justification */}
          <section className="im-section" data-screen-label="02 Justification">
            <div className="im-justify">
              <span className="eyebrow">2-sentence justification</span>
              <p className="im-justify-text">
                The footer is always visible at the end of every page but never competes with content above it — visitors discover Jake after they've read what he built, not before, which is exactly the "noticed there's a real person behind it" moment the brief describes.
              </p>
              <p className="im-justify-text">
                Nav-resident chips and floating badges both interrupt the existing dark-tool aesthetic; a richer footer treats the credentials with the same hi-fi care as the rest of the product.
              </p>
            </div>
          </section>

          {/* Placement compare */}
          <section className="im-section" data-screen-label="03 Placement options">
            <header className="im-section-head">
              <span className="eyebrow">placements considered</span>
              <h2 className="im-section-title">Three options. One chosen.</h2>
            </header>

            <div className="im-placements">
              {PLACEMENTS.map(p => (
                <article key={p.id} className={"im-placement" + (p.chosen ? " chosen" : "")}>
                  <header className="im-placement-head">
                    <h3>{p.label}</h3>
                    {p.chosen
                      ? <span className="badge accent mono">chosen</span>
                      : <span className="badge neutral mono">rejected</span>
                    }
                  </header>
                  <div className="im-placement-cols">
                    <div>
                      <span className="eyebrow im-placement-eyebrow good">+ pros</span>
                      <ul className="im-placement-list">
                        {p.pros.map((x, i) => <li key={i}>{x}</li>)}
                      </ul>
                    </div>
                    <div>
                      <span className="eyebrow im-placement-eyebrow bad">− cons</span>
                      <ul className="im-placement-list">
                        {p.cons.map((x, i) => <li key={i}>{x}</li>)}
                      </ul>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </section>

          {/* See it on every page */}
          <section className="im-section" data-screen-label="04 Live on every page">
            <header className="im-section-head">
              <span className="eyebrow">live on every page</span>
              <h2 className="im-section-title">Scroll to the bottom of any page.</h2>
              <p className="im-section-sub">
                The surface ships in the shared Footer component, so it appears identically on every page in the product. Pick one:
              </p>
            </header>

            <div className="im-page-grid">
              {PAGES.map(pg => (
                <a key={pg.id} href={pg.href} className="im-page-card">
                  <span className="mono dim" style={{fontSize: 11}}>{pg.href}</span>
                  <span className="im-page-label">{pg.label}</span>
                  <span className="im-page-arrow"><Icon.ArrowRight/></span>
                </a>
              ))}
            </div>
          </section>

          {/* Anatomy callout */}
          <section className="im-section" data-screen-label="05 Anatomy">
            <header className="im-section-head">
              <span className="eyebrow">anatomy</span>
              <h2 className="im-section-title">What's in the surface.</h2>
            </header>

            <div className="im-anatomy">
              <ol className="im-anatomy-list">
                <li>
                  <span className="im-anatomy-num mono">01</span>
                  <div>
                    <h4>Monogram</h4>
                    <p>J.S. in the brand cyan. No stock photo, no AI portrait — just two initials. The brief was explicit.</p>
                  </div>
                </li>
                <li>
                  <span className="im-anatomy-num mono">02</span>
                  <div>
                    <h4>Name + status pill</h4>
                    <p><span className="mono">Jake Silverman</span> next to a pulsing-dot pill that reads <span className="mono">available · forward-deployed eng</span>. Signals "real human, currently shipping."</p>
                  </div>
                </li>
                <li>
                  <span className="im-anatomy-num mono">03</span>
                  <div>
                    <h4>Full CV blurb</h4>
                    <p>Verbatim from the brief. Three load-bearing credentials: $100M+ enterprise data programs, production AI agents, PwC competition win.</p>
                  </div>
                </li>
                <li>
                  <span className="im-anatomy-num mono">04</span>
                  <div>
                    <h4>Email Jake — primary CTA</h4>
                    <p><span className="mono">mailto:</span> link styled as the brightest button on the page. Path-of-least-resistance for serious inquiries.</p>
                  </div>
                </li>
                <li>
                  <span className="im-anatomy-num mono">05</span>
                  <div>
                    <h4>LinkedIn + GitHub</h4>
                    <p>Inline as "elsewhere: LinkedIn · GitHub" with small icon glyphs. Renders to non-technical readers as "here's where to vet credentials" and to engineers as "here's where to see code."</p>
                  </div>
                </li>
              </ol>
            </div>
          </section>

          <footer className="dx-foot">
            <span className="mono dim">scroll down to see the footer surface in context ↓</span>
            <a href="index.html" className="btn">
              <Icon.ArrowRight style={{transform: "rotate(180deg)"}}/> Back to home
            </a>
          </footer>
        </div>
      </main>

      <Footer/>
    </>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<IntroduceMe/>);
