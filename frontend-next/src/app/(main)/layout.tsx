'use client';

import AppSidebar from '@/components/layout/AppSidebar';
import AppHeader from '@/components/layout/AppHeader';
import ToolPanel from '@/components/layout/ToolPanel';
import CopilotPanel from '@/components/copilot/CopilotPanel';
import CommandPalette from '@/components/layout/CommandPalette';
import AuthGuard from '@/components/auth/AuthGuard';
import { useAppStore } from '@/stores/app-store';

export default function MainLayout({ children }: { children: React.ReactNode }) {
  const { copilotOpen, toolPanelOpen } = useAppStore();

  return (
    <AuthGuard>
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          background: 'var(--bg)',
          overflow: 'hidden',
        }}
      >
        {/* Compact title bar */}
        <header
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '4px 14px',
            borderBottom: '1px solid var(--bd)',
            flexShrink: 0,
            background: 'var(--s1)',
            zIndex: 2,
            height: 36,
          }}
        >
          <AppHeader />
        </header>

        {/* Three-panel layout */}
        <div
          style={{
            display: 'flex',
            flex: '1 1 0',
            overflow: 'hidden',
            minHeight: 0,
          }}
        >
          <AppSidebar />
          <main
            style={{
              flex: toolPanelOpen ? '1.3' : 1,
              overflow: 'auto',
              background: 'var(--bg)',
              display: 'flex',
              flexDirection: 'column',
              transition: 'flex 0.4s cubic-bezier(0.22, 1, 0.36, 1)',
            }}
          >
            {children}
          </main>
          <ToolPanel />
        </div>

        <CopilotPanel open={copilotOpen} />
        <CommandPalette />
      </div>
    </AuthGuard>
  );
}
