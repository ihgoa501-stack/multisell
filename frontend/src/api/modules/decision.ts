import http from '@/api/http'

export interface PreListingDecisionRequest {
  sku_id: number
  destination_country: string
  target_sale_price: number
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

export function calculatePreListingDecision(data: PreListingDecisionRequest) {
  return http.post('/decisions/prelisting', data)
}
