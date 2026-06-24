'use client';

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
  MessageOutlined,
} from '@ant-design/icons';
import { usePathname, useRouter } from 'next/navigation';
import { useAppStore } from '@/stores/app-store';

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
  MessageOutlined: <MessageOutlined />,
};

interface MenuItemConfig {
  key: string;
  label: string;
  icon?: string;
}

interface MenuGroupConfig {
  label: string;
  items: MenuItemConfig[];
}

const menuGroups: MenuGroupConfig[] = [
  {
    label: '总览',
    items: [
      { key: '/dashboard', icon: 'DashboardOutlined', label: 'Dashboard' },
      { key: '/ai', icon: 'RobotOutlined', label: 'AI 指挥中心' },
    ],
  },
  {
    label: '商品管理',
    items: [
      { key: '/products', icon: 'ShoppingOutlined', label: '商品' },
      { key: '/categories', label: '类目' },
      { key: '/brands', label: '品牌' },
      { key: '/sku', label: 'SKU' },
      { key: '/inventory', label: '库存' },
      { key: '/suppliers', label: '供应商' },
    ],
  },
  {
    label: '销售管理',
    items: [
      { key: '/platforms', icon: 'ShopOutlined', label: '平台' },
      { key: '/platform-integrations', label: '平台集成' },
      { key: '/listings', label: '刊登' },
      { key: '/listing-tasks', label: '刊登任务' },
    ],
  },
  {
    label: '订单物流',
    items: [
      { key: '/orders', icon: 'FileTextOutlined', label: '订单' },
      { key: '/order-import', label: '订单导入' },
      { key: '/shipping', label: '物流' },
      { key: '/platform-fees', label: '平台费用' },
    ],
  },
  {
    label: '财务',
    items: [
      { key: '/finance', icon: 'DollarOutlined', label: '财务总览' },
      { key: '/settlement', label: '结算' },
      { key: '/decision', label: '决策' },
      { key: '/allocation', label: '分配' },
      { key: '/allocation/cost', label: '成本分摊' },
    ],
  },
  {
    label: 'AgentOS',
    items: [
      { key: '/agentos', icon: 'ThunderboltOutlined', label: '控制台' },
      { key: '/agents', label: 'Agent 列表' },
      { key: '/agents/actions', label: 'Action 中心' },
      { key: '/agents/evolution', label: '进化' },
      { key: '/agents/entropy', label: '熵监控' },
      { key: '/agentos/work-items', label: '工作队列' },
    ],
  },
  {
    label: '运营',
    items: [
      { key: '/exceptions', icon: 'WarningOutlined', label: '异常' },
      { key: '/notifications', label: '通知' },
      { key: '/image-gen', label: '图片生成' },
      { key: '/import-batches', label: '批量导入' },
      { key: '/operation-logs', label: '操作日志' },
      { key: '/search', label: '搜索' },
      { key: '/reports', label: '报表' },
      { key: '/aftersales', label: '售后' },
      { key: '/sourcing1688', label: '1688采购' },
    ],
  },
  {
    label: '设置',
    items: [
      { key: '/settings', icon: 'SettingOutlined', label: '系统设置' },
      { key: '/settings/llm', label: 'LLM 配置' },
      { key: '/settings/rbac', label: '权限管理' },
    ],
  },
];

function buildMenuItems() {
  return menuGroups.map((group) => ({
    type: 'group' as const,
    label: group.label,
    children: group.items.map((item) => ({
      key: item.key,
      icon: item.icon ? iconMap[item.icon] : undefined,
      label: item.label,
    })),
  }));
}

export default function AppSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { sidebarCollapsed, setSidebarCollapsed } = useAppStore();

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
