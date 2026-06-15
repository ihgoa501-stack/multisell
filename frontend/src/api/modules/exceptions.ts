import http from '@/api/http'

export interface ExceptionItem {
  id: number
  source_module: string
  source_type?: string | null
  source_id?: number | null
  severity: string
  status: string
  title: string
  description?: string | null
  recommended_action?: string | null
  assigned_to?: string | null
  resolved_at?: string | null
  resolved_by?: string | null
  note?: string | null
  created_at?: string | null
  updated_at?: string | null
}

export function generateExceptions() {
  return http.post('/exceptions/generate')
}

export function getExceptions(params?: { source_module?: string; severity?: string; status?: string }) {
  return http.get('/exceptions', { params })
}

export function getException(id: number) {
  return http.get(`/exceptions/${id}`)
}

export function assignException(id: number, assignedTo: string) {
  return http.post(`/exceptions/${id}/assign`, { assigned_to: assignedTo })
}

export function resolveException(id: number, note?: string) {
  return http.post(`/exceptions/${id}/resolve`, { note: note || '' })
}

export function ignoreException(id: number, note?: string) {
  return http.post(`/exceptions/${id}/ignore`, { note: note || '' })
}
