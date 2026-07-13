export interface MenuItem {
  key: string;
  label: string;
  icon?: string;
  /** Optional RBAC permission code required to see this menu item. */
  permission?: string;
  /** Shows a small status badge next to the label. */
  status?: 'mock' | 'sandbox' | 'shell';
}

export interface MenuGroup {
  label: string;
  items: MenuItem[];
}

export const statusLabels: Record<string, string> = {
  mock: 'Mock',
  sandbox: '沙箱',
  shell: '壳',
};


export const menuGroups: MenuGroup[] = [
  {
    label: 'Owner 总控台',
    items: [
      { key: '/xiaoq', icon: 'RobotOutlined', label: '小Q' },
      { key: '/platform-truth', icon: 'AuditOutlined', label: '平台真相' },
      { key: '/experiments', icon: 'ExperimentOutlined', label: '事实核验案卷' },
      { key: '/demand-cases', icon: 'SearchOutlined', label: '候选市场' },
      { key: '/product-opportunities', icon: 'BulbOutlined', label: '商品机会' },
      { key: '/candidates', label: '候选商品', status: 'mock' },
      { key: '/approval', label: '审批管理', status: 'mock' },
      { key: '/sandbox-listing', label: '沙箱上架', status: 'sandbox' },
      { key: '/profit', icon: 'DollarOutlined', label: '利润真相' },
      { key: '/support', icon: 'MessageOutlined', label: '统一客服' },
      { key: '/dashboard', label: '每日简报' },
    ],
  },
  {
    label: '商品经营',
    items: [
      { key: '/product-hub', icon: 'DatabaseOutlined', label: '产品档案' },
      { key: '/product-images', label: '商品图片工作室' },
      { key: '/products', icon: 'ShoppingOutlined', label: '商品', permission: 'product.read' },
      { key: '/categories', label: '类目' },
      { key: '/brands', label: '品牌' },
      { key: '/sku', label: 'SKU' },
      { key: '/inventory', label: '库存' },
      { key: '/suppliers', label: '供应商' },
      { key: '/competitors', label: '竞品监控' },
      { key: '/listings', label: '刊登管理' },
      { key: '/listing-tasks', label: '刊登任务' },
      { key: '/platform-integrations', label: '平台集成', status: 'sandbox' },
    ],
  },
  {
    label: '订单与履约',
    items: [
      { key: '/orders', icon: 'FileTextOutlined', label: '订单', permission: 'order.read' },
      { key: '/shipping', label: '物流' },
      { key: '/fulfillment', icon: 'ControlOutlined', label: '履约中枢', status: 'sandbox' },
      { key: '/supplychain', icon: 'ApartmentOutlined', label: '供应链追踪' },
      { key: '/aftersales', label: '售后' },
      { key: '/aftersales/disputes', label: '争议管理' },
      { key: '/platform-fees', label: '平台费用' },
      { key: '/purchase', icon: 'ShoppingCartOutlined', label: '采购订单' },
      { key: '/purchase/suggestions', label: '采购建议' },
      { key: '/sourcing1688', label: '1688采购' },
    ],
  },
  {
    label: '审计与异常',
    items: [
      { key: '/exceptions', label: '异常监控' },
      { key: '/operation-logs', label: '操作日志', permission: 'audit.read' },
    ],
  },
  {
    label: '系统设置',
    items: [
      { key: '/settings', icon: 'SettingOutlined', label: '系统设置', permission: 'settings.read' },
      { key: '/settings/rbac', label: '权限管理', permission: 'rbac.manage' },
      { key: '/settings/policy', label: '审批策略' },
      { key: '/design-system', label: '设计系统', status: 'shell' },
    ],
  },
  {
    label: '工作流',
    items: [
      { key: '/workflows', icon: 'ApartmentOutlined', label: '工作流管理' },
      { key: '/workflow/defs', icon: 'ApartmentOutlined', label: '工作流定义' },
      { key: '/workflow/runs', label: '运行记录' },
      { key: '/workflow/monitor', icon: 'DashboardOutlined', label: '监控面板' },
    ],
  },
];
