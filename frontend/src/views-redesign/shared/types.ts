/**
 * 凌镜 LingMirror — 重设计共享类型定义
 * 覆盖：全局布局 / Dashboard / AgentOS / 商品管理 四个 Flow
 */

// ═══════════════════════════════════════
// 全局类型
// ═══════════════════════════════════════

export interface User {
  id: string
  username: string
  display_name: string
  role: 'admin' | 'manager' | 'member'
  avatar_url?: string
  permissions: string[]
}

// ═══════════════════════════════════════
// 导航 & 布局
// ═══════════════════════════════════════

export interface MenuItem {
  key: string
  label: string
  icon: string
  path: string
  badge?: number
  children?: MenuItem[]
}

export interface MenuGroup {
  key: string
  label: string
  items: MenuItem[]
}

export interface Notification {
  id: string
  title: string
  description: string
  severity: 'info' | 'warning' | 'error' | 'critical'
  alert_type: string
  is_read: boolean
  created_at: string
  link_url?: string
  source_type: 'agent' | 'system' | 'exception' | 'listing'
}

export interface CommandPaletteItem {
  id: string
  label: string
  description?: string
  icon: string
  category: 'page' | 'product' | 'agent' | 'command'
  action: string
  shortcut?: string
}

// ═══════════════════════════════════════
// Agent & AgentOS
// ═══════════════════════════════════════

export type AgentStatus = 'online' | 'idle' | 'offline' | 'thinking'

export type AutonomyLevel = 'observation' | 'suggestion' | 'semi_autonomous' | 'full_autonomous'

export type EvolutionStage = 'OBSERVATION' | 'SUGGESTION' | 'SEMI_AUTONOMOUS' | 'FULL_AUTONOMOUS'

export interface Agent {
  id: string
  name: string
  code: string
  description: string
  status: AgentStatus
  autonomy_level: AutonomyLevel
  evolution_stage: EvolutionStage
  trust_score: number
  health_score: number
  squad_id?: string
  squad_name?: string
  avatar_color: string
  total_actions: number
  pending_approvals: number
  success_rate: number
  last_active_at?: string
}

export interface AgentSquad {
  id: string
  name: string
  description: string
  agents: Agent[]
  health_score: number
  active_work_items: number
  pending_approvals: number
}

export type WorkItemStatus = 'pending' | 'approved' | 'rejected' | 'executed' | 'failed'
export type RiskLevel = 'critical' | 'high' | 'medium' | 'low'

export interface WorkItem {
  id: string
  title: string
  description: string
  status: WorkItemStatus
  risk_level: RiskLevel
  agent_id: string
  agent_name: string
  squad_id: string
  squad_name: string
  requires_approval: boolean
  created_at: string
  updated_at: string
  action_url?: string
  proposed_action: string
  expected_impact: string
  confidence_score: number
}

export interface AgentAction {
  id: string
  agent_id: string
  agent_name: string
  action_type: string
  description: string
  status: 'proposed' | 'approved' | 'executed' | 'rejected'
  created_at: string
  confidence: number
  impact: string
}

// ═══════════════════════════════════════
// Dashboard
// ═══════════════════════════════════════

export interface DashboardKPI {
  key: string
  label: string
  value: number
  unit: string
  trend: number // 正数=上升, 负数=下降, 0=持平
  trend_label: string
  color: string
}

export interface OrderTrend {
  date: string
  orders: number
  revenue: number
}

export interface AgentDecisionSummary {
  total_decisions_today: number
  pending_approvals: number
  auto_executed: number
  avg_trust_score: number
  top_agent: string
}

// ═══════════════════════════════════════
// 商品管理
// ═══════════════════════════════════════

export type ProductStatus = 'draft' | 'active' | 'inactive' | 'archived'

export interface Product {
  id: string
  name: string
  name_en?: string
  category_name: string
  brand_name?: string
  status: ProductStatus
  sku_count: number
  on_shelf_count: number
  total_sales: number
  total_revenue: number
  created_at: string
  updated_at: string
  cover_image?: string
  platforms: string[]
}

export interface ProductCategory {
  id: string
  name: string
  parent_id?: string
  children?: ProductCategory[]
}

export interface BatchAction {
  key: string
  label: string
  icon: string
  danger?: boolean
}
