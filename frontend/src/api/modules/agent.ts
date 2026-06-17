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
  getDecisions(params?: any) {
    return http.get('/agents/decisions', { params })
  },
  submitFeedback(decisionId: number, params: { user_action: string; user_feedback?: string }) {
    return http.post(`/agents/decisions/${decisionId}/feedback`, null, { params })
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
    return http.get('/agents/entropy/dashboard')
  },
  runDefenses() {
    return http.post('/agents/entropy/defend')
  },
  getHealthScores() {
    return http.get('/agents/entropy/health')
  },
  getSpc() {
    return http.get('/agents/entropy/spc')
  },
  getChanges(params?: any) {
    return http.get('/agents/entropy/changes', { params })
  },
}
