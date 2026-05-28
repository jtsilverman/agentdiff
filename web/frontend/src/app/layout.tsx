import type { Metadata } from 'next';
import { GeistSans } from 'geist/font/sans';
import { GeistMono } from 'geist/font/mono';
import './globals.css';
import SiteNav from '@/components/SiteNav';
import Tweaks from '@/components/Tweaks';

export const metadata: Metadata = {
  title: 'agentdiff — see how your AI thinks',
  description:
    'agentdiff records every move your AI agent makes — context, reasoning, decisions, output. Compare runs of the same task side by side.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body>
        <SiteNav />
        <main>{children}</main>
        <Tweaks />
      </body>
    </html>
  );
}
