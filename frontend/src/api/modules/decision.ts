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

export function calculatePreListingDecision(data: PreListingDecisionRequest) {
  return http.post('/decisions/prelisting', data)
}

export function comparePreListingDecision(data: CompareDecisionRequest) {
  return http.post('/decisions/prelisting/compare', data)
}
