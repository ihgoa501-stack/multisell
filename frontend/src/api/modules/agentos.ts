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
  source_type?: string
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

// ── Phase 3: Operation Log ───────────────────────────────

export interface AgentOSOperationLog {
  id: number
  user_id: number
  item_id: string
  action: string
  source_type: string | null
  previous_status: string | null
  new_status: string | null
  comment: string | null
  created_at: string | null
}

export interface AutonomyCandidate {
  agent_id: string
  agent_name: string
  squad_id: string
  squad_name: string
  current_level: string
  suggested: boolean
  direction: string | null
  target_level: string | null
  confidence: number
  reason: string
}

export function getAgentOSOperations(params?: {
  item_id?: string
  action?: string
  source_type?: string
  limit?: number
  offset?: number
}) {
  return http.get('/agentos/operations', { params })
}

export function getAgentOSUpgradeCandidates() {
  return http.get('/agentos/agents/upgrade-candidates')
}

export function upgradeAgentLevel(agentId: string, targetLevel: string) {
  return http.post(`/agentos/agents/${agentId}/upgrade`, null, {
    params: { target_level: targetLevel },
  })
}

export function downgradeAgentLevel(agentId: string, targetLevel: string) {
  return http.post(`/agentos/agents/${agentId}/downgrade`, null, {
    params: { target_level: targetLevel },
  })
}

// ── Phase 4 Finale: Agent Detail ─────────────────────────

export interface AgentDetailResponse {
  agent: AgentOSAgent
  squad_name: string
  current_work_items: AgentOSWorkItem[]
  recent_operations: AgentOSOperationLog[]
  decision_count_7d: number
  adoption_rate_7d: number
}

export function getAgentOSAgentDetail(agentId: string) {
  return http.get(`/agentos/agents/${agentId}/detail`)
}

// ── Phase 5: Action Proposal API ─────────────────────────────

export interface ActionProposalCreatePayload {
  source_type: string
  source_id?: string | null
  agent_id?: string | null
  squad_id?: string | null
  action_type: string
  business_object_type?: string | null
  business_object_id?: string | null
  title: string
  description?: string | null
  proposed_payload?: Record<string, any>
  before_snapshot?: Record<string, any> | null
  risk_level: RiskLevel
  requires_approval: boolean
  confidence?: number | null
}

export interface ActionApprovalPayload {
  comment?: string | null
}

export interface ActionExecutionPayload {
  executor?: string | null
}

export interface ActionReviewPayload {
  outcome: 'positive' | 'neutral' | 'negative'
  business_metric?: string | null
  metric_delta?: number | null
  notes?: string | null
}

export function createActionProposal(payload: ActionProposalCreatePayload) {
  return http.post('/agentos/action-proposals', payload)
}

export function approveActionProposal(proposalId: number, payload: ActionApprovalPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/approve`, payload)
}

export function rejectActionProposal(proposalId: number, payload: ActionApprovalPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/reject`, payload)
}

export function executeActionProposal(proposalId: number, payload: ActionExecutionPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/execute`, payload)
}

export function reviewActionProposal(proposalId: number, payload: ActionReviewPayload) {
  return http.post(`/agentos/action-proposals/${proposalId}/review`, payload)
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
  getAgentOSOperations,
  getAgentOSUpgradeCandidates,
  upgradeAgentLevel,
  downgradeAgentLevel,
  getAgentOSAgentDetail,
  createActionProposal,
  approveActionProposal,
  rejectActionProposal,
  executeActionProposal,
  reviewActionProposal,
}
