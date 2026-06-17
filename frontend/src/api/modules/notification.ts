/**
 * 通知与预警 API 模块
 */
import http from '@/api/http'

export const notificationApi = {
  list(params?: { unread_only?: boolean; alert_type?: string; page?: number; page_size?: number }) {
    return http.get('/notifications', { params })
  },
  getUnreadCount() {
    return http.get('/notifications/unread-count')
  },
  markRead(id: number) {
    return http.put(`/notifications/${id}/read`)
  },
  markAllRead() {
    return http.put('/notifications/read-all')
  },
  delete(id: number) {
    return http.delete(`/notifications/${id}`)
  },
  checkAlerts() {
    return http.post('/notifications/check')
  },
  // 预警规则
  listRules() {
    return http.get('/alert-rules')
  },
  updateRule(id: number, data: any) {
    return http.put(`/alert-rules/${id}`, data)
  },
  initializeRules() {
    return http.post('/alert-rules/initialize')
  },
}
