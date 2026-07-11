'use client';

import { useState } from 'react';
import { MenuOutlined } from '@ant-design/icons';
import AppSidebar from '@/components/layout/AppSidebar';
import AppHeader from '@/components/layout/AppHeader';
import ToolPanel from '@/components/layout/ToolPanel';
import CopilotPanel from '@/components/copilot/CopilotPanel';
import CommandPalette from '@/components/layout/CommandPalette';
import AuthGuard from '@/components/auth/AuthGuard';
import { useAppStore, type PanelMode } from '@/stores/app-store';

const panelFlexMap: Record<PanelMode, { main: number; tool: number; copilot: number }> = {
  ai:       { main: 0, tool: 0, copilot: 7 },
  balanced: { main: 5, tool: 5, copilot: 5 },
  tool:     { main: 3, tool: 7, copilot: 0 },
};

const modeLabels: Record<PanelMode, string> = {
  ai: 'AI',
  balanced: '50/50',
  tool: 'Tools',
};

export default function MainLayout({ children }: { children: React.ReactNode }) {
  const { toolPanelOpen, panelMode, setPanelMode, toggleToolPanel } = useAppStore();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const flex = panelFlexMap[panelMode];

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
          <button
            type="button"
            className="app-shell-mobile-menu"
            aria-label="打开导航"
            aria-expanded={mobileNavOpen}
            onClick={() => setMobileNavOpen(true)}
          >
            <MenuOutlined />
          </button>
          <AppHeader />

          {/* Mode indicator */}
          <div
            className="app-shell-mode-switcher"
            style={{
              display: 'flex',
              alignItems: 'center',
              marginLeft: 12,
              background: 'var(--s2)',
              borderRadius: 6,
              padding: '2px',
              gap: 1,
              flexShrink: 0,
            }}
          >
            {(Object.keys(panelFlexMap) as PanelMode[]).map((mode) => (
              <button
                key={mode}
                onClick={() => {
                  setPanelMode(mode);
                  if (mode === 'tool' && !toolPanelOpen) toggleToolPanel();
                }}
                style={{
                  padding: '2px 10px',
                  border: 'none',
                  borderRadius: 4,
                  cursor: 'pointer',
                  fontFamily: 'var(--body)',
                  fontSize: '0.65rem',
                  fontWeight: panelMode === mode ? 600 : 400,
                  lineHeight: '1.5',
                  letterSpacing: '0.02em',
                  background: panelMode === mode ? 'var(--s3)' : 'transparent',
                  color: panelMode === mode ? 'var(--t1)' : 'var(--t3)',
                  transition: 'all var(--dur-short)',
                  whiteSpace: 'nowrap',
                }}
              >
                {modeLabels[mode]}
              </button>
            ))}
          </div>
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
          {mobileNavOpen && (
            <button
              type="button"
              className="app-shell-mobile-backdrop"
              aria-label="关闭导航"
              onClick={() => setMobileNavOpen(false)}
            />
          )}
          <div
            className={`app-shell-sidebar${mobileNavOpen ? ' is-open' : ''}`}
            onClick={() => setMobileNavOpen(false)}
          >
            <AppSidebar />
          </div>
          <main
            className="app-shell-main"
            style={{
              flex: flex.main || 1,
              overflow: 'auto',
              background: 'var(--bg)',
              display: 'flex',
              flexDirection: 'column',
              transition: 'flex 0.4s var(--cubic-panel)',
              minWidth: flex.main === 0 ? 0 : 280,
            }}
          >
            {flex.main === 0 ? null : children}
          </main>

          {/* Copilot panel (inline, not Drawer) */}
          <div
            className="app-shell-copilot"
            style={{
              flex: flex.copilot,
              overflow: 'hidden',
              borderLeft: flex.copilot > 0 ? '1px solid var(--bd)' : 'none',
              background: 'var(--s1)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: flex.copilot === 0 ? 0 : 320,
              maxWidth: flex.copilot === 0 ? 0 : 600,
              transition: 'flex 0.4s var(--cubic-panel), min-width 0.4s var(--cubic-panel), max-width 0.4s var(--cubic-panel)',
            }}
          >
            {flex.copilot > 0 && <CopilotPanel />}
          </div>

          <ToolPanel />
        </div>

        <CommandPalette />
      </div>
    </AuthGuard>
  );
}
