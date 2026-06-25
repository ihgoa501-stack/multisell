// Product analysis API types and client
// API base: Next.js rewrites /api/v1/* -> Go backend /api/v1/*

export interface AnalyzeInput {
  sourcing_product_id: number
  target_sale_price: number
}

export interface ProductAnalysis {
  id: number
  sourcing_product_id: number
  target_sale_price: number
  estimated_cost: number
  estimated_profit_margin: number | null
  demand_score: number | null
  demand_score_status: string  // 'available' | 'no_data'
  competition_index: number | null
  competition_status: string   // 'available' | 'no_data'
  analysis_status: string      // 'completed' | 'pending' | 'error'
  error_message: string | null
  analyzed_by: string
  analyzed_at: string | null
  created_at: string
  updated_at: string
}

export interface AnalysisResult {
  analysis: ProductAnalysis
  profit_score: number | null
  demand_score: number | null
  demand_score_status: string
  competition_score: number | null
  competition_status: string
  warning: string
}

export interface FeedbackInput {
  decision: 'imported' | 'abandoned'
  actual_profit_margin?: number
  notes?: string
}

const BASE = '/api/v1/product-analysis'

/** Analyze a sourced 1688 product */
export async function analyzeProduct(input: AnalyzeInput): Promise<AnalysisResult> {
  const res = await fetch(`${BASE}/analyze`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
  const body = await res.json()
  return body.data as AnalysisResult
}

/** List previous analyses */
export async function listAnalyses(params?: { status?: string; page?: number; size?: number }) {
  const qs = new URLSearchParams()
  if (params?.status) qs.set('status', params.status)
  if (params?.page) qs.set('page', String(params.page))
  if (params?.size) qs.set('size', String(params.size))
  const q = qs.toString()
  const res = await fetch(`${BASE}/analyses${q ? '?' + q : ''}`)
  if (!res.ok) throw new Error(await res.text())
  const body = await res.json()
  return { items: body.data as ProductAnalysis[], total: body.total as number }
}

/** Get analysis detail */
export async function getAnalysis(id: number): Promise<ProductAnalysis> {
  const res = await fetch(`${BASE}/analyses/${id}`)
  if (!res.ok) throw new Error(await res.text())
  const body = await res.json()
  return body.data as ProductAnalysis
}

/** Submit feedback for an analysis */
export async function recordFeedback(id: number, input: FeedbackInput): Promise<void> {
  const res = await fetch(`${BASE}/analyses/${id}/feedback`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(await res.text())
}
