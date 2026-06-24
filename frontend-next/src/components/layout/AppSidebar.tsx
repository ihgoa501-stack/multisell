'use client';

import { useEffect } from 'react';
import { Layout, Menu } from 'antd';
import {
  DashboardOutlined,
  RobotOutlined,
  ShoppingOutlined,
  ShopOutlined,
  FileTextOutlined,
  DollarOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { usePathname, useRouter } from 'next/navigation';
import { useAppStore } from '@/stores/app-store';
import { usePermissionStore } from '@/stores/permission-store';
import { menuGroups, type MenuItem } from '@/config/menu';

const { Sider } = Layout;

const iconMap: Record<string, React.ReactNode> = {
  DashboardOutlined: <DashboardOutlined />,
  RobotOutlined: <RobotOutlined />,
  ShoppingOutlined: <ShoppingOutlined />,
  ShopOutlined: <ShopOutlined />,
  FileTextOutlined: <FileTextOutlined />,
  DollarOutlined: <DollarOutlined />,
  ThunderboltOutlined: <ThunderboltOutlined />,
  WarningOutlined: <WarningOutlined />,
  SettingOutlined: <SettingOutlined />,
};

export default function AppSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { sidebarCollapsed, setSidebarCollapsed } = useAppStore();
  const { permissions, fetched, fetchPermissions, hasPermission } = usePermissionStore();

  // Fetch permissions on mount — this runs once because the store prevents
  // redundant fetches via the `fetched` flag.
  useEffect(() => {
    fetchPermissions();
  }, [fetchPermissions]);

  function isItemVisible(item: MenuItem): boolean {
    if (!item.permission) return true;
    return hasPermission(item.permission);
  }

  function buildMenuItems() {
    return menuGroups
      .map((group) => {
        const visibleItems = group.items.filter(isItemVisible);
        // Hide the group label if all items are filtered out
        if (visibleItems.length === 0) return null;
        return {
          type: 'group' as const,
          label: group.label,
          children: visibleItems.map((item) => ({
            key: item.key,
            icon: item.icon ? iconMap[item.icon] : undefined,
            label: item.label,
          })),
        };
      })
      .filter(Boolean);
  }

  return (
    <Sider
      collapsible
      collapsed={sidebarCollapsed}
      onCollapse={setSidebarCollapsed}
      theme="light"
      width={240}
      style={{
        overflow: 'auto',
        height: '100vh',
        position: 'sticky',
        top: 0,
        left: 0,
      }}
    >
      <div
        style={{
          height: 48,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          borderBottom: '1px solid #f0f0f0',
          fontWeight: 700,
          fontSize: sidebarCollapsed ? 14 : 18,
          color: '#1677ff',
        }}
      >
        {sidebarCollapsed ? 'LM' : 'LingMirror'}
      </div>
      <Menu
        mode="inline"
        selectedKeys={[pathname]}
        items={buildMenuItems()}
        onClick={({ key }) => router.push(key)}
        style={{ borderRight: 0 }}
      />
    </Sider>
  );
}
