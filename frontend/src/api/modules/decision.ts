import http from '@/api/http'

export interface PreListingDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_id?: number
  platform_fee_pct: number
  payment_fee_pct: number
  other_fee: number
  minimum_margin_pct: number
  cargo_type: string
}

export interface PreListingDecisionResponse {
  sku_id: number
  destination_country: string
  target_sale_price: number
  product_cost: number
  shipping_fee: number
  platform_fee: number
  payment_fee: number
  other_fee: number
  profit_amount: number
  profit_margin: number
  recommendation: string
  blocking_reasons: string[]
  warnings: string[]
}

export interface CompareDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_ids: number[]
  payment_fee_pct: number
  other_fee: number
  minimum_margin_pct: number
  cargo_type: string
}

export interface CompareItem {
  platform_id: number
  platform_name: string
  product_cost: number
  shipping_fee: number
  platform_fee: number
  payment_fee: number
  total_cost: number
  profit_amount: number
  profit_margin: number
  recommendation: string
  blocking_reasons: string[]
  warnings: string[]
}

export interface CompareDecisionResponse {
  sku_id: number
  destination_country: string
  target_sale_price: number
  results: CompareItem[]
}

export interface PreListingDecisionBatchItem extends PreListingDecisionRequest {
  item_key?: string | null
}

export interface PreListingDecisionBatchRequest {
  items: PreListingDecisionBatchItem[]
}

export interface PreListingDecisionBatchItemResult {
  index: number
  item_key?: string | null
  sku_id?: number | null
  status: 'success' | 'error'
  result?: PreListingDecisionResponse | null
  error_message?: string | null
}

export interface PreListingDecisionBatchSummary {
  total_items: number
  success_count: number
  error_count: number
  approve_count: number
  reject_count: number
  needs_data_count: number
  average_profit_margin: number
}

export interface PreListingDecisionBatchResponse {
  summary: PreListingDecisionBatchSummary
  items: PreListingDecisionBatchItemResult[]
}

export interface PreListingDecisionExcelPreviewRow {
  row_number: number
  item?: PreListingDecisionBatchItem | null
  errors: string[]
}

export interface PreListingDecisionExcelPreviewResponse {
  total_rows: number
  valid_rows: number
  error_rows: number
  items: PreListingDecisionExcelPreviewRow[]
}

export function calculatePreListingDecision(data: PreListingDecisionRequest) {
  return http.post('/decisions/prelisting', data)
}

export function comparePreListingDecision(data: CompareDecisionRequest) {
  return http.post('/decisions/prelisting/compare', data)
}

export function calculateBatchPreListingDecision(data: PreListingDecisionBatchRequest) {
  return http.post('/decisions/prelisting/batch', data)
}

export async function downloadBatchPreListingDecisionTemplate() {
  const resp = await http.get('/decisions/prelisting/batch/template', { responseType: 'blob' })
  return resp.data
}

export function previewBatchPreListingDecisionExcel(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post('/decisions/prelisting/batch/preview', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export async function exportBatchPreListingDecisionResults(data: PreListingDecisionBatchResponse) {
  const resp = await http.post('/decisions/prelisting/batch/export', data, { responseType: 'blob' })
  return resp.data
}
