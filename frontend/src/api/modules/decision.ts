import http from '@/api/http'

export interface PreListingDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_id?: number | null
  category_id?: number | null
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
  fixed_fee: number
  advertising_fee: number
  other_fee: number
  profit_amount: number
  profit_margin: number
  recommendation: string
  blocking_reasons: string[]
  warnings: string[]
  applied_platform_fee_rule_id?: number | null
  platform_fee_source: string
  platform_fee_rule_summary?: string | null
}

export function calculatePreListingDecision(data: PreListingDecisionRequest) {
  return http.post('/decisions/prelisting', data)
}

// --- 批量 ---

export interface PreListingDecisionBatchItem {
  item_key?: string | null
  sku_id: number
  destination_country: string
  target_sale_price: number
  platform_id?: number | null
  category_id?: number | null
  platform_fee_pct: number
  payment_fee_pct: number
  other_fee: number
  minimum_margin_pct: number
  cargo_type: string
}

export interface PreListingDecisionBatchItemResult {
  index: number
  item_key?: string | null
  sku_id?: number | null
  status: string
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

export function calculateBatchPreListingDecision(data: { items: PreListingDecisionBatchItem[] }) {
  return http.post('/decisions/prelisting/batch', data)
}

// --- Excel ---

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

export function downloadBatchPreListingDecisionTemplate() {
  return http.get('/decisions/prelisting/batch/template', { responseType: 'blob' })
}

export function previewBatchPreListingDecisionExcel(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return http.post('/decisions/prelisting/batch/preview', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

export function exportBatchPreListingDecisionResults(data: PreListingDecisionBatchResponse) {
  return http.post('/decisions/prelisting/batch/export', data, { responseType: 'blob' })
}
