'use client';

import { Layout, Button, Space, Avatar, theme as antdTheme } from 'antd';
import { SearchOutlined, RobotOutlined, MenuFoldOutlined, MenuUnfoldOutlined, SunOutlined, MoonOutlined } from '@ant-design/icons';
import Breadcrumbs from '@/components/layout/Breadcrumbs';
import { useAppStore } from '@/stores/app-store';

const { Header } = Layout;

function toggleTheme() {
  const html = document.documentElement;
  const next = html.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
  html.setAttribute('data-theme', next);
  // Dispatch custom event for AntdProvider to react
  window.dispatchEvent(new CustomEvent('themechange', { detail: next }));
}

export default function AppHeader() {
  const { sidebarCollapsed, toggleSidebar, setCommandPaletteOpen, setCopilotOpen } = useAppStore();

  return (
    <Header
      style={{
        background: 'var(--s1)',
        padding: '0 16px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottom: '1px solid var(--bd)',
        height: 48,
        lineHeight: '48px',
      }}
    >
      <Space>
        <Button
          type="text"
          icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={toggleSidebar}
          style={{ color: 'var(--t2)' }}
        />
        <Breadcrumbs />
      </Space>
      <Space>
        <Button
          icon={<SearchOutlined />}
          onClick={() => setCommandPaletteOpen(true)}
          type="text"
          style={{ color: 'var(--t2)' }}
        >
          Cmd+K
        </Button>
        <Button
          icon={<RobotOutlined />}
          onClick={() => setCopilotOpen(true)}
          type="text"
          style={{ color: 'var(--t2)' }}
        />
        <Button
          type="text"
          icon={<SunOutlined />}
          onClick={toggleTheme}
          style={{ color: 'var(--t2)' }}
        />
        <Avatar size="small" style={{ backgroundColor: 'var(--i5)', verticalAlign: 'middle', fontFamily: 'var(--ds)' }}>
          L
        </Avatar>
      </Space>
    </Header>
  );
}
