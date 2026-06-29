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
      { key: '/ai', icon: 'RobotOutlined', label: 'AI 指挥中心' },
    ],
  },
  {
    label: '经营闭环',
    items: [
      { key: '/owner', icon: 'DashboardOutlined', label: '经营总控台' },
    ],
  },
  {
    label: '采购管理',
    items: [
      { key: '/purchase', icon: 'ShoppingCartOutlined', label: '采购订单' },
      { key: '/purchase/suggestions', label: '采购建议' },
      { key: '/sourcing', icon: 'SearchOutlined', label: 'AI 选品' },
      { key: '/candidates', label: '候选商品' },
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
      { key: '/supplychain', icon: 'ApartmentOutlined', label: '供应链追踪' },
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
      { key: '/agents/trust', label: '信任与自主度' },
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
      { key: '/support', label: '客服中心', icon: 'CustomerServiceOutlined' },
      { key: '/support/templates', label: '回复模板' },
    ],
  },
  {
    label: '设置',
    items: [
      { key: '/settings', icon: 'SettingOutlined', label: '系统设置' },
      { key: '/settings/llm', label: 'LLM 配置' },
      { key: '/settings/rbac', label: '权限管理' },
      { key: '/settings/policy', label: '审批策略' },
    ],
  },
];
