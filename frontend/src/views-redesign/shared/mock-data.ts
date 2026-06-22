/**
 * 凌镜 LingMirror — 重设计共享 Mock 数据
 * 为所有重设计 Flow 提供可交互的模拟数据
 */
import type {
  User,
  MenuGroup,
  Notification,
  CommandPaletteItem,
  Agent,
  AgentSquad,
  WorkItem,
  AgentAction,
  DashboardKPI,
  OrderTrend,
  AgentDecisionSummary,
  Product,
  BatchAction,
} from './types'

// ═══════════════════════════════════════
// 用户
// ═══════════════════════════════════════

export const mockUser: User = {
  id: 'u-001',
  username: 'admin',
  display_name: '凌管理员',
  role: 'admin',
  permissions: ['*'],
}

// ═══════════════════════════════════════
// 导航分组
// ═══════════════════════════════════════

export const mockMenuGroups: MenuGroup[] = [
  {
    key: 'command',
    label: '指挥中心',
    items: [
      { key: 'dashboard', label: '运营驾驶舱', icon: 'DashboardOutlined', path: '/redesign/dashboard' },
      { key: 'agentos', label: 'AgentOS 总控', icon: 'RobotOutlined', path: '/redesign/agentos', badge: 3 },
    ],
  },
  {
    key: 'product',
    label: '商品中心',
    items: [
      { key: 'products', label: '商品管理', icon: 'ShoppingOutlined', path: '/redesign/products' },
      { key: 'categories', label: '分类管理', icon: 'AppstoreOutlined', path: '/redesign/categories' },
      { key: 'inventory', label: '库存管理', icon: 'DatabaseOutlined', path: '/redesign/products/u-001/inventory' },
      { key: 'brands', label: '品牌管理', icon: 'TagOutlined', path: '/redesign/brands' },
    ],
  },
  {
    key: 'supply',
    label: '供应链',
    items: [
      { key: 'suppliers', label: '供应商', icon: 'TeamOutlined', path: '/redesign/suppliers' },
      { key: 'sourcing', label: '1688 选品', icon: 'CloudDownloadOutlined', path: '/redesign/1688-sourcing' },
      { key: 'shipping', label: '物流管理', icon: 'SendOutlined', path: '/redesign/shipping/manage' },
    ],
  },
  {
    key: 'trade',
    label: '交易平台',
    items: [
      { key: 'platforms', label: '平台管理', icon: 'GlobalOutlined', path: '/redesign/platforms' },
      { key: 'listings', label: '上架管理', icon: 'UploadOutlined', path: '/redesign/listings' },
      { key: 'orders', label: '订单管理', icon: 'ShoppingCartOutlined', path: '/redesign/orders' },
      { key: 'settlements', label: '结算中心', icon: 'AccountBookOutlined', path: '/redesign/settlements' },
      { key: 'finance', label: '财务管理', icon: 'MoneyCollectOutlined', path: '/redesign/finance' },
    ],
  },
  {
    key: 'system',
    label: '系统',
    items: [
      { key: 'notifications', label: '通知中心', icon: 'BellOutlined', path: '/redesign/notifications', badge: 5 },
      { key: 'users', label: '用户管理', icon: 'UserOutlined', path: '/redesign/users' },
      { key: 'agents', label: 'Agent 管理', icon: 'ExperimentOutlined', path: '/redesign/agents' },
      { key: 'reports', label: '数据报表', icon: 'BarChartOutlined', path: '/redesign/reports' },
    ],
  },
]

// ═══════════════════════════════════════
// 通知
// ═══════════════════════════════════════

export const mockNotifications: Notification[] = [
  {
    id: 'n-001',
    title: 'Agent A1 建议调价：SKU-2847 利润率为负',
    description: '检测到 SKU-2847（蓝牙音箱-黑色）当前售价低于成本线，建议提价 15%',
    severity: 'warning',
    alert_type: 'agent_suggestion',
    is_read: false,
    created_at: '2026-06-22T10:30:00',
    link_url: '/agentos',
    source_type: 'agent',
  },
  {
    id: 'n-002',
    title: 'Ozon 上架任务完成：3 个商品已发布',
    description: '批量上架任务成功完成，3 个商品已在 Ozon 平台上架',
    severity: 'info',
    alert_type: 'listing_complete',
    is_read: false,
    created_at: '2026-06-22T09:15:00',
    link_url: '/listings',
    source_type: 'listing',
  },
  {
    id: 'n-003',
    title: '库存预警：5 个 SKU 库存低于安全线',
    description: '以下 SKU 库存不足，建议尽快补货',
    severity: 'error',
    alert_type: 'inventory_alert',
    is_read: false,
    created_at: '2026-06-22T08:00:00',
    link_url: '/products/u-001/inventory',
    source_type: 'system',
  },
  {
    id: 'n-004',
    title: 'Agent A5 分析报告：本周退货率上升 12%',
    description: '分析师 A5 发现本周退货率异常上升，主要集中于电子类目',
    severity: 'warning',
    alert_type: 'agent_analysis',
    is_read: true,
    created_at: '2026-06-21T18:30:00',
    source_type: 'agent',
  },
  {
    id: 'n-005',
    title: 'Shopee 平台 API 连接异常',
    description: 'Shopee API 返回 503 错误，已自动重试，当前恢复正常',
    severity: 'info',
    alert_type: 'platform_error',
    is_read: true,
    created_at: '2026-06-21T14:20:00',
    source_type: 'exception',
  },
]

// ═══════════════════════════════════════
// Command Palette
// ═══════════════════════════════════════

export const mockCommandItems: CommandPaletteItem[] = [
  { id: 'cmd-1', label: '运营驾驶舱', description: '查看全局数据概览', icon: 'DashboardOutlined', category: 'page', action: '/redesign/dashboard' },
  { id: 'cmd-2', label: 'AgentOS 总控', description: 'Agent 团队管理与审批', icon: 'RobotOutlined', category: 'page', action: '/redesign/agentos' },
  { id: 'cmd-3', label: '商品管理', description: '查看和管理所有商品', icon: 'ShoppingOutlined', category: 'page', action: '/redesign/products' },
  { id: 'cmd-4', label: '新增商品', description: '创建新的商品', icon: 'PlusCircleOutlined', category: 'command', action: '/redesign/products/create', shortcut: '⌘N' },
  { id: 'cmd-5', label: '订单管理', description: '查看和处理订单', icon: 'ShoppingCartOutlined', category: 'page', action: '/orders' },
  { id: 'cmd-6', label: '上架管理', description: '多平台上架任务', icon: 'UploadOutlined', category: 'page', action: '/listings' },
  { id: 'cmd-7', label: 'Agent A1 — 定价优化师', description: '信任分 92 · 在线', icon: 'ThunderboltOutlined', category: 'agent', action: '/agents/a1' },
  { id: 'cmd-8', label: 'Agent A5 — 市场分析师', description: '信任分 88 · 在线', icon: 'LineChartOutlined', category: 'agent', action: '/agents/a5' },
  { id: 'cmd-9', label: '导入订单', description: '从 CSV/Excel 导入', icon: 'FileAddOutlined', category: 'command', action: '/order-import' },
  { id: 'cmd-10', label: '切换暗色模式', description: '切换亮色/暗色主题', icon: 'MoonOutlined', category: 'command', action: 'toggle-theme', shortcut: '⌘D' },
]

// ═══════════════════════════════════════
// Agent
// ═══════════════════════════════════════

export const mockAgents: Agent[] = [
  {
    id: 'a1', name: '定价优化师', code: 'A1', description: '分析市场数据，优化商品定价策略',
    status: 'online', autonomy_level: 'semi_autonomous', evolution_stage: 'SEMI_AUTONOMOUS',
    trust_score: 92, health_score: 95, squad_id: 'sq-1', squad_name: '营收小队',
    avatar_color: '#2962FF', total_actions: 347, pending_approvals: 2, success_rate: 94.5,
    last_active_at: '2026-06-22T10:25:00',
  },
  {
    id: 'a2', name: '上架专员', code: 'A2', description: '自动生成多平台 Listing 并发布',
    status: 'online', autonomy_level: 'semi_autonomous', evolution_stage: 'SEMI_AUTONOMOUS',
    trust_score: 88, health_score: 91, squad_id: 'sq-1', squad_name: '营收小队',
    avatar_color: '#7C3AED', total_actions: 512, pending_approvals: 1, success_rate: 97.2,
    last_active_at: '2026-06-22T10:20:00',
  },
  {
    id: 'a3', name: '库存管家', code: 'A3', description: '监控库存水平，自动触发补货建议',
    status: 'idle', autonomy_level: 'suggestion', evolution_stage: 'SUGGESTION',
    trust_score: 85, health_score: 88, squad_id: 'sq-2', squad_name: '供应链小队',
    avatar_color: '#059669', total_actions: 189, pending_approvals: 0, success_rate: 91.0,
    last_active_at: '2026-06-22T09:45:00',
  },
  {
    id: 'a4', name: '客服助手', code: 'A4', description: '智能处理客户消息和售后问题',
    status: 'online', autonomy_level: 'observation', evolution_stage: 'OBSERVATION',
    trust_score: 72, health_score: 80, squad_id: 'sq-1', squad_name: '营收小队',
    avatar_color: '#D97706', total_actions: 78, pending_approvals: 0, success_rate: 85.3,
    last_active_at: '2026-06-22T10:15:00',
  },
  {
    id: 'a5', name: '市场分析师', code: 'A5', description: '市场趋势分析、竞品监控、选品推荐',
    status: 'online', autonomy_level: 'semi_autonomous', evolution_stage: 'SEMI_AUTONOMOUS',
    trust_score: 88, health_score: 92, squad_id: 'sq-2', squad_name: '供应链小队',
    avatar_color: '#3B82F6', total_actions: 234, pending_approvals: 0, success_rate: 89.7,
    last_active_at: '2026-06-22T10:10:00',
  },
  {
    id: 'a6', name: '供应链分析师', code: 'A6', description: '供应商评估、物流优化、成本分析',
    status: 'thinking', autonomy_level: 'suggestion', evolution_stage: 'SUGGESTION',
    trust_score: 81, health_score: 85, squad_id: 'sq-2', squad_name: '供应链小队',
    avatar_color: '#EC4899', total_actions: 156, pending_approvals: 0, success_rate: 87.8,
    last_active_at: '2026-06-22T10:28:00',
  },
  {
    id: 'a7', name: '财务分析师', code: 'A7', description: '利润核算、成本归因、财务报表分析',
    status: 'offline', autonomy_level: 'observation', evolution_stage: 'OBSERVATION',
    trust_score: 76, health_score: 70, squad_id: 'sq-3', squad_name: '风控小队',
    avatar_color: '#14B8A6', total_actions: 67, pending_approvals: 0, success_rate: 93.1,
    last_active_at: '2026-06-21T22:00:00',
  },
  {
    id: 'g1', name: '合规总督', code: 'G1', description: '确保所有操作符合平台规则和公司政策',
    status: 'online', autonomy_level: 'full_autonomous', evolution_stage: 'FULL_AUTONOMOUS',
    trust_score: 96, health_score: 98, squad_id: 'sq-3', squad_name: '风控小队',
    avatar_color: '#DC2626', total_actions: 891, pending_approvals: 0, success_rate: 99.2,
    last_active_at: '2026-06-22T10:30:00',
  },
  {
    id: 'g2', name: '效率总督', code: 'G2', description: '监控系统效率，优化资源分配',
    status: 'online', autonomy_level: 'semi_autonomous', evolution_stage: 'SEMI_AUTONOMOUS',
    trust_score: 90, health_score: 93, squad_id: 'sq-3', squad_name: '风控小队',
    avatar_color: '#F59E0B', total_actions: 423, pending_approvals: 0, success_rate: 95.0,
    last_active_at: '2026-06-22T10:22:00',
  },
  {
    id: 'g3', name: '战略总督', code: 'G3', description: '全局战略规划和执行，跨团队协调',
    status: 'idle', autonomy_level: 'semi_autonomous', evolution_stage: 'SEMI_AUTONOMOUS',
    trust_score: 94, health_score: 96, squad_id: 'sq-3', squad_name: '风控小队',
    avatar_color: '#8B5CF6', total_actions: 312, pending_approvals: 0, success_rate: 96.4,
    last_active_at: '2026-06-22T09:30:00',
  },
]

// ═══════════════════════════════════════
// Agent 团队
// ═══════════════════════════════════════

export const mockSquads: AgentSquad[] = [
  {
    id: 'sq-1', name: '营收小队', description: '负责商品上架、定价和客服',
    agents: mockAgents.filter(a => a.squad_id === 'sq-1'),
    health_score: 89, active_work_items: 12, pending_approvals: 3,
  },
  {
    id: 'sq-2', name: '供应链小队', description: '负责库存、采购和物流优化',
    agents: mockAgents.filter(a => a.squad_id === 'sq-2'),
    health_score: 85, active_work_items: 8, pending_approvals: 0,
  },
  {
    id: 'sq-3', name: '风控小队', description: '负责合规、效率和战略规划',
    agents: mockAgents.filter(a => a.squad_id === 'sq-3'),
    health_score: 95, active_work_items: 5, pending_approvals: 0,
  },
]

// ═══════════════════════════════════════
// WorkItems
// ═══════════════════════════════════════

export const mockWorkItems: WorkItem[] = [
  {
    id: 'wi-001', title: '建议调价：蓝牙音箱-黑色 SKU-2847',
    description: '当前售价 ¥89.00，成本 ¥76.50，利润率仅 14%。建议提价至 ¥103.00（利润率 25.7%）',
    status: 'pending', risk_level: 'high',
    agent_id: 'a1', agent_name: '定价优化师', squad_id: 'sq-1', squad_name: '营收小队',
    requires_approval: true, created_at: '2026-06-22T10:00:00', updated_at: '2026-06-22T10:00:00',
    proposed_action: '将 SKU-2847 售价从 ¥89.00 调整至 ¥103.00',
    expected_impact: '预计月增利润 ¥420', confidence_score: 87,
  },
  {
    id: 'wi-002', title: '自动上架：3 个商品发布到 Ozon',
    description: '根据选品策略，将以下 3 个商品自动发布到 Ozon 平台',
    status: 'pending', risk_level: 'medium',
    agent_id: 'a2', agent_name: '上架专员', squad_id: 'sq-1', squad_name: '营收小队',
    requires_approval: true, created_at: '2026-06-22T09:30:00', updated_at: '2026-06-22T09:30:00',
    proposed_action: '发布商品 P-101, P-102, P-103 至 Ozon',
    expected_impact: '预计月增销售额 ¥15,000', confidence_score: 92,
  },
  {
    id: 'wi-003', title: '补货建议：USB-C 数据线 库存不足',
    description: '当前库存 23 件，日均销售 8 件，预计 3 天内断货。建议补货 200 件',
    status: 'pending', risk_level: 'critical',
    agent_id: 'a3', agent_name: '库存管家', squad_id: 'sq-2', squad_name: '供应链小队',
    requires_approval: true, created_at: '2026-06-22T08:00:00', updated_at: '2026-06-22T08:00:00',
    proposed_action: '向供应商 S-003 下单 USB-C 数据线 200 件',
    expected_impact: '避免断货损失约 ¥3,600/天', confidence_score: 95,
  },
  {
    id: 'wi-004', title: '市场分析：电子类目退货率异常',
    description: '本周电子类目退货率上升至 8.2%（上周 5.1%），主要集中在充电器品类',
    status: 'approved', risk_level: 'medium',
    agent_id: 'a5', agent_name: '市场分析师', squad_id: 'sq-2', squad_name: '供应链小队',
    requires_approval: false, created_at: '2026-06-21T18:00:00', updated_at: '2026-06-22T09:00:00',
    proposed_action: '生成退货原因分析报告并通知品控团队',
    expected_impact: '及时发现问题商品，减少退货损失', confidence_score: 89,
  },
  {
    id: 'wi-005', title: '合规检查：Ozon 平台政策更新',
    description: 'Ozon 更新了商品描述规范，检测到 5 个 Listing 不符合新要求',
    status: 'executed', risk_level: 'low',
    agent_id: 'g1', agent_name: '合规总督', squad_id: 'sq-3', squad_name: '风控小队',
    requires_approval: false, created_at: '2026-06-21T14:00:00', updated_at: '2026-06-22T08:30:00',
    proposed_action: '自动修正 5 个 Listing 的商品描述格式',
    expected_impact: '避免被平台下架的风险', confidence_score: 99,
  },
]

// ═══════════════════════════════════════
// Agent Actions（时间线用）
// ═══════════════════════════════════════

export const mockAgentActions: AgentAction[] = [
  { id: 'act-1', agent_id: 'a1', agent_name: '定价优化师', action_type: 'price_adjust', description: '建议 SKU-2847 提价 15%', status: 'proposed', created_at: '2026-06-22T10:00:00', confidence: 87, impact: '月增利润 ¥420' },
  { id: 'act-2', agent_id: 'g1', agent_name: '合规总督', action_type: 'compliance_check', description: '自动修正 5 个 Ozon Listing', status: 'executed', created_at: '2026-06-22T08:30:00', confidence: 99, impact: '避免下架风险' },
  { id: 'act-3', agent_id: 'a5', agent_name: '市场分析师', action_type: 'analysis', description: '生成电子类目退货分析报告', status: 'approved', created_at: '2026-06-21T18:00:00', confidence: 89, impact: '发现质量问题' },
  { id: 'act-4', agent_id: 'a2', agent_name: '上架专员', action_type: 'listing_publish', description: '发布 3 个商品到 Ozon', status: 'proposed', created_at: '2026-06-22T09:30:00', confidence: 92, impact: '月增销售 ¥15K' },
  { id: 'act-5', agent_id: 'a3', agent_name: '库存管家', action_type: 'reorder', description: '补货 USB-C 数据线 200 件', status: 'proposed', created_at: '2026-06-22T08:00:00', confidence: 95, impact: '避免断货 ¥3.6K/天' },
  { id: 'act-6', agent_id: 'g2', agent_name: '效率总督', action_type: 'optimize', description: '重新分配 Agent 任务队列', status: 'executed', created_at: '2026-06-22T07:00:00', confidence: 96, impact: '处理效率提升 12%' },
]

// ═══════════════════════════════════════
// Dashboard KPI
// ═══════════════════════════════════════

export const mockDashboardKPIs: DashboardKPI[] = [
  { key: 'revenue', label: '总收入', value: 1285640.50, unit: '元', trend: 12.3, trend_label: '较上月', color: '#2962FF' },
  { key: 'orders', label: '订单总数', value: 3847, unit: '单', trend: 8.7, trend_label: '较上月', color: '#059669' },
  { key: 'profit_margin', label: '利润率', value: 23.5, unit: '%', trend: -2.1, trend_label: '较上月', color: '#7C3AED' },
  { key: 'inventory_health', label: '库存健康', value: 91.2, unit: '%', trend: 3.4, trend_label: '较上月', color: '#D97706' },
]

export const mockOrderTrend: OrderTrend[] = [
  { date: '2026-05-24', orders: 42, revenue: 12600 },
  { date: '2026-05-25', orders: 38, revenue: 11400 },
  { date: '2026-05-26', orders: 55, revenue: 16500 },
  { date: '2026-05-27', orders: 47, revenue: 14100 },
  { date: '2026-05-28', orders: 61, revenue: 18300 },
  { date: '2026-05-29', orders: 53, revenue: 15900 },
  { date: '2026-05-30', orders: 44, revenue: 13200 },
  { date: '2026-05-31', orders: 39, revenue: 11700 },
  { date: '2026-06-01', orders: 67, revenue: 20100 },
  { date: '2026-06-02', orders: 58, revenue: 17400 },
  { date: '2026-06-03', orders: 72, revenue: 21600 },
  { date: '2026-06-04', orders: 63, revenue: 18900 },
  { date: '2026-06-05', orders: 49, revenue: 14700 },
  { date: '2026-06-06', orders: 55, revenue: 16500 },
  { date: '2026-06-07', orders: 41, revenue: 12300 },
  { date: '2026-06-08', orders: 78, revenue: 23400 },
  { date: '2026-06-09', orders: 69, revenue: 20700 },
  { date: '2026-06-10', orders: 82, revenue: 24600 },
  { date: '2026-06-11', orders: 71, revenue: 21300 },
  { date: '2026-06-12', orders: 59, revenue: 17700 },
  { date: '2026-06-13', orders: 45, revenue: 13500 },
  { date: '2026-06-14', orders: 37, revenue: 11100 },
  { date: '2026-06-15', orders: 85, revenue: 25500 },
  { date: '2026-06-16', orders: 73, revenue: 21900 },
  { date: '2026-06-17', orders: 91, revenue: 27300 },
  { date: '2026-06-18', orders: 66, revenue: 19800 },
  { date: '2026-06-19', orders: 78, revenue: 23400 },
  { date: '2026-06-20', orders: 84, revenue: 25200 },
  { date: '2026-06-21', orders: 62, revenue: 18600 },
  { date: '2026-06-22', orders: 70, revenue: 21000 },
]

export const mockAgentDecisionSummary: AgentDecisionSummary = {
  total_decisions_today: 18,
  pending_approvals: 3,
  auto_executed: 12,
  avg_trust_score: 86.5,
  top_agent: '定价优化师',
}

// ═══════════════════════════════════════
// 商品
// ═══════════════════════════════════════

export const mockProducts: Product[] = [
  {
    id: 'p-001', name: '蓝牙音箱 Pro Max', name_en: 'Bluetooth Speaker Pro Max',
    category_name: '电子产品', brand_name: 'SoundMax', status: 'active',
    sku_count: 4, on_shelf_count: 3, total_sales: 1247, total_revenue: 186530,
    created_at: '2026-01-15T10:00:00', updated_at: '2026-06-20T14:30:00',
    platforms: ['ozon', 'shopee'],
  },
  {
    id: 'p-002', name: 'USB-C 快充数据线 1.5m', name_en: 'USB-C Fast Charge Cable 1.5m',
    category_name: '配件', brand_name: 'ChargeTech', status: 'active',
    sku_count: 3, on_shelf_count: 3, total_sales: 3892, total_revenue: 77840,
    created_at: '2026-02-20T08:00:00', updated_at: '2026-06-21T09:15:00',
    platforms: ['ozon', 'shopee', 'wildberries'],
  },
  {
    id: 'p-003', name: '无线充电板 15W', name_en: 'Wireless Charger Pad 15W',
    category_name: '电子产品', brand_name: 'ChargeTech', status: 'active',
    sku_count: 2, on_shelf_count: 2, total_sales: 856, total_revenue: 59920,
    created_at: '2026-03-10T12:00:00', updated_at: '2026-06-19T16:45:00',
    platforms: ['ozon'],
  },
  {
    id: 'p-004', name: '硅胶手机壳 iPhone 15 Pro', name_en: 'Silicone Phone Case iPhone 15 Pro',
    category_name: '配件', brand_name: 'CasePro', status: 'draft',
    sku_count: 6, on_shelf_count: 0, total_sales: 0, total_revenue: 0,
    created_at: '2026-06-18T14:00:00', updated_at: '2026-06-22T10:00:00',
    platforms: [],
  },
  {
    id: 'p-005', name: 'LED 台灯 护眼款', name_en: 'LED Desk Lamp Eye-Care',
    category_name: '家居', brand_name: 'LightWell', status: 'active',
    sku_count: 3, on_shelf_count: 2, total_sales: 432, total_revenue: 64800,
    created_at: '2026-04-05T09:00:00', updated_at: '2026-06-17T11:30:00',
    platforms: ['ozon', 'shopee'],
  },
  {
    id: 'p-006', name: '运动蓝牙耳机 TWS', name_en: 'Sports TWS Earbuds',
    category_name: '电子产品', brand_name: 'SoundMax', status: 'inactive',
    sku_count: 2, on_shelf_count: 0, total_sales: 2103, total_revenue: 252360,
    created_at: '2025-11-20T10:00:00', updated_at: '2026-05-10T08:00:00',
    platforms: ['shopee'],
  },
  {
    id: 'p-007', name: '不锈钢保温杯 500ml', name_en: 'Stainless Steel Thermos 500ml',
    category_name: '家居', brand_name: 'ThermoLife', status: 'active',
    sku_count: 4, on_shelf_count: 4, total_sales: 1876, total_revenue: 150080,
    created_at: '2026-01-28T11:00:00', updated_at: '2026-06-22T07:20:00',
    platforms: ['ozon', 'wildberries', 'shopee'],
  },
  {
    id: 'p-008', name: '机械键盘 87键 青轴', name_en: 'Mechanical Keyboard 87-Key Blue Switch',
    category_name: '电子产品', brand_name: 'KeyMaster', status: 'active',
    sku_count: 3, on_shelf_count: 3, total_sales: 567, total_revenue: 170100,
    created_at: '2026-03-22T15:00:00', updated_at: '2026-06-21T13:00:00',
    platforms: ['ozon'],
  },
]

export const mockBatchActions: BatchAction[] = [
  { key: 'publish', label: '批量上架', icon: 'UploadOutlined' },
  { key: 'unpublish', label: '批量下架', icon: 'CloudDownloadOutlined' },
  { key: 'price', label: '批量调价', icon: 'DollarOutlined' },
  { key: 'category', label: '修改分类', icon: 'AppstoreOutlined' },
  { key: 'export', label: '导出', icon: 'DownloadOutlined' },
  { key: 'archive', label: '归档', icon: 'InboxOutlined', danger: true },
]
