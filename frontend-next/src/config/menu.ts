export interface MenuItem {
  key: string;
  label: string;
  icon?: string;
  /** Optional RBAC permission code required to see this menu item. */
  permission?: string;
}

export interface MenuGroup {
  label: string;
  items: MenuItem[];
}

export const menuGroups: MenuGroup[] = [
  {
    label: '总览',
    items: [
      { key: '/dashboard', icon: 'DashboardOutlined', label: 'Dashboard' },
      { key: '/ai', icon: 'RobotOutlined', label: 'AI 指挥中心', permission: 'ai:view' },
    ],
  },
  {
    label: '商品管理',
    items: [
      { key: '/products', icon: 'ShoppingOutlined', label: '商品', permission: 'products:read' },
      { key: '/categories', label: '类目', permission: 'products:read' },
      { key: '/brands', label: '品牌', permission: 'products:read' },
      { key: '/sku', label: 'SKU', permission: 'products:read' },
      { key: '/inventory', label: '库存', permission: 'inventory:read' },
      { key: '/suppliers', label: '供应商', permission: 'products:read' },
    ],
  },
  {
    label: '销售管理',
    items: [
      { key: '/platforms', icon: 'ShopOutlined', label: '平台', permission: 'platforms:read' },
      { key: '/platform-integrations', label: '平台集成', permission: 'platforms:read' },
      { key: '/listings', label: '刊登', permission: 'listings:read' },
      { key: '/listing-tasks', label: '刊登任务', permission: 'listings:read' },
    ],
  },
  {
    label: '订单物流',
    items: [
      { key: '/orders', icon: 'FileTextOutlined', label: '订单', permission: 'orders:read' },
      { key: '/order-import', label: '订单导入', permission: 'orders:read' },
      { key: '/shipping', label: '物流', permission: 'shipping:read' },
      { key: '/platform-fees', label: '平台费用', permission: 'finance:read' },
    ],
  },
  {
    label: '财务',
    items: [
      { key: '/finance', icon: 'DollarOutlined', label: '财务总览', permission: 'finance:read' },
      { key: '/settlement', label: '结算', permission: 'finance:read' },
      { key: '/decision', label: '决策', permission: 'decision:read' },
      { key: '/allocation', label: '分配', permission: 'allocation:read' },
      { key: '/allocation/cost', label: '成本分摊', permission: 'allocation:read' },
    ],
  },
  {
    label: 'AgentOS',
    items: [
      { key: '/agentos', icon: 'ThunderboltOutlined', label: '控制台', permission: 'agentos:view' },
      { key: '/agents', label: 'Agent 列表', permission: 'agentos:view' },
      { key: '/agents/actions', label: 'Action 中心', permission: 'agentos:view' },
      { key: '/agents/evolution', label: '进化', permission: 'agentos:view' },
      { key: '/agents/entropy', label: '熵监控', permission: 'agentos:view' },
      { key: '/agentos/work-items', label: '工作队列', permission: 'agentos:view' },
    ],
  },
  {
    label: '运营',
    items: [
      { key: '/exceptions', icon: 'WarningOutlined', label: '异常', permission: 'exceptions:read' },
      { key: '/notifications', label: '通知', permission: 'notifications:read' },
      { key: '/image-gen', label: '图片生成', permission: 'imagegen:view' },
      { key: '/import-batches', label: '批量导入', permission: 'importbatches:read' },
      { key: '/operation-logs', label: '操作日志', permission: 'operationlogs:read' },
      { key: '/search', label: '搜索' },
      { key: '/reports', label: '报表', permission: 'reports:read' },
      { key: '/aftersales', label: '售后', permission: 'aftersales:read' },
      { key: '/sourcing1688', label: '1688采购', permission: 'sourcing:read' },
    ],
  },
  {
    label: '设置',
    items: [
      { key: '/settings', icon: 'SettingOutlined', label: '系统设置', permission: 'settings:view' },
      { key: '/settings/llm', label: 'LLM 配置', permission: 'settings:view' },
      { key: '/settings/rbac', label: '权限管理', permission: 'rbac:view' },
    ],
  },
];
