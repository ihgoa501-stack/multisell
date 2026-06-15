import http from '@/api/http'

export interface SettlementBatch {
  id: number
  platform_name?: string | null
  filename: string
  row_count: number
  matched_count: number
  unmatched_count: number
  import_status: string
  status: string
  created_by?: string | null
  created_at?: string | null
}

export interface SettlementItem {
  id: number
  batch_id: number
  row_number: number
  platform?: string | null
  store_name?: string | null
  platform_order_no?: string | null
  order_no?: string | null
  transaction_type: string
  currency: string
  amount: number
  settled_at?: string | null
  description?: string | null
  match_status: string
  matched_order_id?: number | null
  cost_layer: string
  created_at?: string | null
}

export function importSettlement(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post('/settlements/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function getSettlementBatches(params?: { status?: string }) {
  return http.get('/settlements', { params })
}

export function getSettlementBatch(batchId: number) {
  return http.get(`/settlements/${batchId}`)
}

export function getSettlementItems(batchId: number, params?: { match_status?: string }) {
  return http.get(`/settlements/${batchId}/items`, { params })
}

export function getUnmatchedSettlements() {
  return http.get('/settlements/unmatched')
}
