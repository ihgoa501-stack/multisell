import http from '@/api/http'

export interface AgentAction {
  id: number
  source_module?: string | null
  source_type?: string | null
  source_id?: number | null
  exception_id?: number | null
  action_type: string
  title: string
  description?: string | null
  proposed_payload?: Record<string, any> | null
  before_snapshot?: Record<string, any> | null
  after_snapshot?: Record<string, any> | null
  status: string
  proposed_by?: string | null
  approved_by?: string | null
  approved_at?: string | null
  rejected_by?: string | null
  rejected_at?: string | null
  rejection_reason?: string | null
  executed_by?: string | null
  executed_at?: string | null
  created_at?: string | null
}

export function createAgentAction(data: {
  source_module?: string
  source_type?: string
  source_id?: number
  exception_id?: number
  action_type: string
  title: string
  description?: string
  proposed_payload?: Record<string, any>
  before_snapshot?: Record<string, any>
}) {
  return http.post('/agent-actions', data)
}

export function getAgentActions(params?: { exception_id?: number; status?: string }) {
  return http.get('/agent-actions', { params })
}

export function getAgentAction(id: number) {
  return http.get(`/agent-actions/${id}`)
}

export function approveAgentAction(id: number) {
  return http.post(`/agent-actions/${id}/approve`)
}

export function rejectAgentAction(id: number, rejection_reason?: string) {
  return http.post(`/agent-actions/${id}/reject`, { rejection_reason })
}

export function markExecutedAgentAction(id: number, after_snapshot?: Record<string, any>) {
  return http.post(`/agent-actions/${id}/mark-executed`, { after_snapshot: after_snapshot || {} })
}
