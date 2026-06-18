import http from '@/api/http'

export interface AgentOSBusinessObject {
  type?: string | null
  id?: string | null
  label?: string | null
}

export interface AgentOSWorkItem {
  id: string
  source_type: string
  source_id: string
  source_module?: string | null
  business_object: AgentOSBusinessObject
  squad: string
  agent_id?: string | null
  title: string
  summary?: string | null
  recommendation?: string | null
  risk_level: string
  approval_required: boolean
  status: string
  action_type?: string | null
  context: Record<string, any>
  audit_link?: string | null
  created_at?: string | null
}

export interface AgentOSSquad {
  id: string
  name: string
  description: string
  agents: string[]
  decision_count_7d: number
  pending_approvals: number
  risk_count: number
  adoption_rate: number
  autonomy_level: string
}

export interface AgentOSTemplate {
  id: string
  title: string
  squad: string
  description: string
  mode: string
  route: string
  phase: string
}

export interface AgentOSSummary {
  sales_today: number
  profit_today: number
  inventory_risks: number
  pending_approvals: number
  active_work_items: number
  agent_automation_rate: number
}

export interface AgentOSControlCenter {
  summary: AgentOSSummary
  work_items: AgentOSWorkItem[]
  squads: AgentOSSquad[]
  templates: AgentOSTemplate[]
}

export const agentosApi = {
  getControlCenter() {
    return http.get('/agentos/control-center')
  },
  getWorkItems(params?: {
    source_type?: string
    squad?: string
    status?: string
    page?: number
    page_size?: number
  }) {
    return http.get('/agentos/work-items', { params })
  },
  getSquads() {
    return http.get('/agentos/squads')
  },
  getTemplates() {
    return http.get('/agentos/templates')
  },
}
