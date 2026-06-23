import http from '@/api/http'

export function fetchReturns(params?: { status?: string; order_id?: number }) {
  return http.get('/aftersales', { params })
}

export function fetchReturnDetail(id: number) {
  return http.get(`/aftersales/${id}`)
}

export function createReturn(data: {
  order_id: number
  sku_id: number
  return_quantity: number
  reason: string
  item_id?: number
}) {
  return http.post('/aftersales', data)
}

export function approveReturn(id: number, refund_amount: number) {
  return http.post(`/aftersales/${id}/approve`, { refund_amount })
}

export function rejectReturn(id: number, rejection_reason: string) {
  return http.post(`/aftersales/${id}/reject`, { rejection_reason })
}

export function receiveReturn(id: number, inspection_result?: string) {
  return http.post(`/aftersales/${id}/receive`, { inspection_result })
}

export function refundReturn(id: number) {
  return http.post(`/aftersales/${id}/refund`)
}
