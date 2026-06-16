import http from '@/api/http'

export interface AllocationBatch {
  id: number
  allocation_type: string
  allocation_method: string
  total_amount: number
  currency: string
  source_filename?: string | null
  row_count: number
  status: string
  posted_count: number
  created_by?: string | null
  created_at?: string | null
}

export interface AllocationItem {
  id: number
  batch_id: number
  row_number: number
  sku_id?: number | null
  sku_code?: string | null
  order_id?: number | null
  quantity: number
  weight_kg?: number | null
  volume_m3?: number | null
  item_value?: number | null
  allocation_factor?: number | null
  allocated_amount: number
  cost_layer: string
  posted_to_ledger: boolean
  created_at?: string | null
}

export function importAllocation(file: File, params: {
  allocation_type: string
  allocation_method: string
  total_amount: number
  currency?: string
}) {
  const formData = new FormData()
  formData.append('file', file)
  const query = new URLSearchParams({
    allocation_type: params.allocation_type,
    allocation_method: params.allocation_method,
    total_amount: String(params.total_amount),
    currency: params.currency || 'CNY',
  }).toString()
  return http.post(`/allocations/import?${query}`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function getAllocationBatches(params?: { status?: string }) {
  return http.get('/allocations', { params })
}

export function getAllocationBatch(batchId: number) {
  return http.get(`/allocations/${batchId}`)
}

export function getAllocationItems(batchId: number) {
  return http.get(`/allocations/${batchId}/items`)
}

export function calculateAllocation(batchId: number) {
  return http.post(`/allocations/${batchId}/calculate`)
}

export function postAllocationToLedger(batchId: number) {
  return http.post(`/allocations/${batchId}/post-to-ledger`)
}
