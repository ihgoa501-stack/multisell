import http from '@/api/http'

// ─── 枚举 ──────────────────────────────────────────────────

export type AutonomyLevel = 'OBSERVATION' | 'SUGGESTION' | 'SEMI_AUTONOMOUS' | 'FULL_AUTONOMOUS'
export type WorkItemPriority = 'low' | 'medium' | 'high' | 'critical'
export type WorkItemStatus = 'pending' | 'in_progress' | 'completed' | 'failed' | 'blocked' | 'cancelled'
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

// ─── 核心数据模型 ──────────────────────────────────────────

export interface AgentOSAgent {
  id: string
  name: string
  role: string
  squad_id: string
  status: string
  autonomy_level: AutonomyLevel
  current_workload: number
  success_rate: number
  last_activity_at: string | null
  risk_level: RiskLevel
}

export interface AgentOSSquad {
  id: string
  name: string
  description: string
  domain: string
  status: string
  autonomy_level: AutonomyLevel
  agents: AgentOSAgent[]
  active_work_items: number
  pending_approvals: number
  risk_level: RiskLevel
  health_score: number
}

export interface AgentOSWorkItem {
  id: string
  source_type: string
  source_id: string
  title: string
  description: string | null
  priority: WorkItemPriority
  status: WorkItemStatus
  risk_level: RiskLevel
  agent_id: string | null
  agent_name: string | null
  squad_id: string | null
  squad_name: string | null
  autonomy_level: AutonomyLevel
  requires_approval: boolean
  created_at: string | null
  updated_at: string | null
  due_at: string | null
  action_url: string | null
  metadata: Record<string, any>
}

export interface AgentOSOverview {
  health_score: number
  active_agents: number
  pending_approvals: number
  critical_items: number
}

export interface AgentOSMetric {
  key: string
  label: string
  value: number
  trend: string | null
  unit: string
}

export interface AgentOSTemplate {
  id: string
  title: string
  description: string
  squad: string
  mode: string
  route: string
  phase: string
}

// ─── 响应模型 ──────────────────────────────────────────────

export interface ControlCenterResponse {
  overview: AgentOSOverview
  squads: AgentOSSquad[]
  priority_work_items: AgentOSWorkItem[]
  metrics: AgentOSMetric[]
  recent_activity: AgentOSWorkItem[]
}

export interface WorkItemsResponse {
  items: AgentOSWorkItem[]
  total: number
  limit: number
  offset: number
}

export interface SquadsResponse {
  squads: AgentOSSquad[]
  summary: AgentOSOverview | null
}

export interface TemplatesResponse {
  templates: AgentOSTemplate[]
}

// ─── 查询参数 ──────────────────────────────────────────────

export interface WorkItemQuery {
  status?: string
  priority?: string
  squad?: string
  agent_id?: string
  requires_approval?: boolean
  limit?: number
  offset?: number
}

// ─── API 方法 ──────────────────────────────────────────────

export function getAgentOSControlCenter() {
  return http.get('/agentos/control-center')
}

export function getAgentOSWorkItems(query?: WorkItemQuery) {
  return http.get('/agentos/work-items', { params: query })
}

export function getAgentOSSquads() {
  return http.get('/agentos/squads')
}

export function getAgentOSTemplates() {
  return http.get('/agentos/templates')
}

// ── Phase 2: Mutation API ────────────────────────────────────

export interface WorkItemStatusPayload {
  status: WorkItemStatus
  comment?: string
}

export interface WorkItemApprovalPayload {
  action: 'approve' | 'reject'
  comment?: string
}

export function updateWorkItemStatus(itemId: string, payload: WorkItemStatusPayload) {
  return http.patch(`/agentos/work-items/${itemId}/status`, payload)
}

export function approveWorkItem(itemId: string, payload: WorkItemApprovalPayload) {
  return http.post(`/agentos/work-items/${itemId}/approve`, payload)
}

export function rejectWorkItem(itemId: string, payload: WorkItemApprovalPayload) {
  return http.post(`/agentos/work-items/${itemId}/reject`, payload)
}

// ─── 兼容对象式导出（用于 apiModules 合并） ───────────────

export const agentosApi = {
  getControlCenter: getAgentOSControlCenter,
  getWorkItems: getAgentOSWorkItems,
  getSquads: getAgentOSSquads,
  getTemplates: getAgentOSTemplates,
  updateWorkItemStatus,
  approveWorkItem,
  rejectWorkItem,
}
