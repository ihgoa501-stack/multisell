'use client';

import { Layout } from 'antd';
import AppSidebar from '@/components/layout/AppSidebar';
import AppHeader from '@/components/layout/AppHeader';
import CopilotPanel from '@/components/copilot/CopilotPanel';
import CommandPalette from '@/components/layout/CommandPalette';
import AuthGuard from '@/components/auth/AuthGuard';
import { useAppStore } from '@/stores/app-store';

const { Content } = Layout;

export default function MainLayout({ children }: { children: React.ReactNode }) {
  const { copilotOpen } = useAppStore();

  return (
    <AuthGuard>
      {/* ponytail: inline styles on Layout — Ant Design requires it */}
      <Layout style={{
        minHeight: '100vh',
        background: 'var(--bg)',
      }}>
        <AppSidebar />
        <Layout style={{ background: 'var(--s1)' }}>
          <AppHeader />
          <Content
            style={{
              margin: 0,
              padding: 0,
              background: 'var(--bg)',
              minHeight: 280,
            }}
          >
            {children}
          </Content>
        </Layout>
        <CopilotPanel open={copilotOpen} />
        <CommandPalette />
      </Layout>
    </AuthGuard>
  );
}
