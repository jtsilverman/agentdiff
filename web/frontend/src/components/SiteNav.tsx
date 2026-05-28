'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { ArrowRight, BrandMark } from './Icons';

const LINKS = [
  { href: '/', label: 'Home', match: (p: string) => p === '/' },
  { href: '/about', label: 'About', match: (p: string) => p.startsWith('/about') },
  { href: '/traces', label: 'Traces', match: (p: string) => p.startsWith('/traces') },
  { href: '/diff', label: 'Diff', match: (p: string) => p.startsWith('/diff') },
  { href: '/docs', label: 'Docs', match: (p: string) => p.startsWith('/docs') },
];

export default function SiteNav() {
  const pathname = usePathname();

  return (
    <nav className="ad-nav">
      <div className="ad-nav-inner">
        <Link href="/" className="ad-brand">
          <BrandMark />
          <span>
            agentdiff <span className="tag">v0.4.1</span>
          </span>
        </Link>

        <div className="ad-nav-links">
          {LINKS.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className={`ad-nav-link${l.match(pathname) ? ' active' : ''}`}
            >
              {l.label}
            </Link>
          ))}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <Link href="/" className="ad-nav-cta">
            Try an example <ArrowRight />
          </Link>
        </div>
      </div>
    </nav>
  );
}
