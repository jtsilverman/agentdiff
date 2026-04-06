import type { Metadata } from 'next';
import './globals.css';
import Nav from '@/components/Nav';

export const metadata: Metadata = {
  title: 'AgentDiff',
  description: 'Git diff for agent behavior',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-950 text-gray-100">
        <div className="flex h-screen flex-col">
          {/* Header */}
          <header className="flex h-12 shrink-0 items-center border-b border-gray-800 bg-gray-900 px-4">
            <h1 className="text-lg font-bold tracking-tight">AgentDiff</h1>
          </header>

          <div className="flex flex-1 overflow-hidden">
            {/* Sidebar */}
            <aside className="w-48 shrink-0 border-r border-gray-800 bg-gray-900">
              <Nav />
            </aside>

            {/* Main content */}
            <main className="flex-1 overflow-y-auto p-6">{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
}
