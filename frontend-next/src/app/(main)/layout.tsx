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
  const { sidebarCollapsed, copilotOpen } = useAppStore();

  return (
    <AuthGuard>
      <Layout style={{ minHeight: '100vh' }}>
        <AppSidebar />
        <Layout>
          <AppHeader />
          <Content
            style={{
              margin: 0,
              padding: 0,
              background: '#f5f5f5',
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
