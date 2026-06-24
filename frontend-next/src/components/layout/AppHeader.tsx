'use client';

import { Layout, Button, Space, Avatar } from 'antd';
import { SearchOutlined, RobotOutlined, MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import Breadcrumbs from '@/components/layout/Breadcrumbs';
import { useAppStore } from '@/stores/app-store';

const { Header } = Layout;

export default function AppHeader() {
  const { sidebarCollapsed, toggleSidebar, setCommandPaletteOpen, setCopilotOpen } = useAppStore();

  return (
    <Header
      style={{
        background: '#fff',
        padding: '0 24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        borderBottom: '1px solid #f0f0f0',
        height: 56,
      }}
    >
      <Space>
        <Button
          type="text"
          icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={toggleSidebar}
        />
        <Breadcrumbs />
      </Space>
      <Space>
        <Button
          icon={<SearchOutlined />}
          onClick={() => setCommandPaletteOpen(true)}
          type="text"
        >
          Cmd+K
        </Button>
        <Button
          icon={<RobotOutlined />}
          onClick={() => setCopilotOpen(true)}
          type="text"
        />
        <Avatar size="small" style={{ backgroundColor: '#1677ff' }}>
          U
        </Avatar>
      </Space>
    </Header>
  );
}
