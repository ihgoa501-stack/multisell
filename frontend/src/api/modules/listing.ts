import http from '@/api/http'

// ── Listing Task ──────────────────────────────────────────────────────

export interface ListingTaskDecisionResult {
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
  platform_fee_source: string
}

export interface ListingTaskCreateFromDecisionItem {
  item_key?: string | null
  sku_id: number
  platform_id: number
  decision_result: ListingTaskDecisionResult
}

export interface ListingTaskCreateResult {
  id: number
  product_id: number
  platform_id: number
  status: string
  missing_requirements: string[]
  source_item_key?: string | null
}

export interface ListingTaskCreateFromDecisionResponse {
  created_count: number
  reused_count: number
  skipped_count: number
  tasks: ListingTaskCreateResult[]
  skipped: { item_key?: string; reason: string }[]
}

export interface ListingTask {
  id: number
  product_id: number
  product_name: string
  platform_id: number
  platform_name: string
  sku_id?: number | null
  product_listing_id?: number | null
  source_type: string
  source_item_key?: string | null
  status: string
  missing_requirements: string[]
  target_sale_price?: number | null
  target_profit_margin?: number | null
  destination_country?: string | null
  last_error?: string | null
  created_by?: string | null
  created_at?: string | null
  updated_at?: string | null
}

export function createListingTasksFromDecisions(items: ListingTaskCreateFromDecisionItem[]) {
  return http.post('/listing-tasks/from-decisions', { items })
}

export function getListingTasks(params?: { status?: string; platform_id?: number }) {
  return http.get('/listing-tasks', { params })
}

export function recheckListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/recheck`)
}

export function cancelListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/cancel`)
}

export function publishListingTask(taskId: number) {
  return http.post(`/listing-tasks/${taskId}/publish`)
}
