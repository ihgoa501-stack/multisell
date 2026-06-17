import http from '@/api/http'

export const agentApi = {
  list() {
    return http.get('/agents')
  },
  get(agentId: string) {
    return http.get(`/agents/${agentId}`)
  },
  decide(agentId: string, data: { decision_point: string; context: any; dry_run?: boolean }) {
    return http.post(`/agents/${agentId}/decide`, data)
  },
  getDashboard() {
    return http.get('/agents/dashboard')
  },
  listActions(params?: any) {
    return http.get('/agents/actions', { params })
  },
  executeAction(actionId: number) {
    return http.post(`/agents/actions/${actionId}/execute`)
  },
  rejectAction(actionId: number) {
    return http.post(`/agents/actions/${actionId}/reject`)
  },
  getDecisions(params?: any) {
    return http.get('/agents/decisions', { params })
  },
  submitFeedback(decisionId: number, data: { user_action: string; user_overrides?: any; user_feedback?: string }) {
    return http.post(`/agents/decisions/${decisionId}/feedback`, data)
  },
  listRules(params?: any) {
    return http.get('/agents/rules', { params })
  },
  createRule(data: any) {
    return http.post('/agents/rules', data)
  },
  updateRule(ruleId: number, data: any) {
    return http.put(`/agents/rules/${ruleId}`, data)
  },
  deleteRule(ruleId: number) {
    return http.delete(`/agents/rules/${ruleId}`)
  },
  getProfile() {
    return http.get('/agents/profile')
  },
  updateProfile(data: any) {
    return http.put('/agents/profile', data)
  },
  listEpisodes(params?: any) {
    return http.get('/agents/episodes', { params })
  },
}

export const entropyApi = {
  getDashboard() {
    return http.get('/entropy/dashboard')
  },
  runDefenses() {
    return http.post('/entropy/defend')
  },
  getHealthScores() {
    return http.get('/entropy/health')
  },
  getSpc() {
    return http.get('/entropy/spc')
  },
  getChanges(params?: any) {
    return http.get('/entropy/changes', { params })
  },
}
