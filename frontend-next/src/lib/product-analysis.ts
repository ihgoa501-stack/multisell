import apiClient from "@/lib/api-client";
import type { PageResult, Result } from "@/types/api";

export interface AnalyzeInput {
  sourcing_product_id: number;
  target_sale_price: number;
}

export interface ProductAnalysis {
  id: number;
  sourcing_product_id: number;
  target_sale_price: number;
  estimated_cost: number;
  estimated_profit_margin: number | null;
  demand_score: number | null;
  demand_score_status: string; // 'available' | 'no_data'
  competition_index: number | null;
  competition_status: string; // 'available' | 'no_data'
  analysis_status: string; // 'completed' | 'pending' | 'error'
  error_message: string | null;
  analyzed_by: string;
  analyzed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface AnalysisResult {
  analysis: ProductAnalysis;
  profit_score: number | null;
  demand_score: number | null;
  demand_score_status: string;
  competition_score: number | null;
  competition_status: string;
  warning: string;
}

export interface FeedbackInput {
  decision: "imported" | "abandoned";
  actual_profit_margin?: number;
  notes?: string;
}

const BASE = "/v1/product-analysis";

/** Analyze a sourced 1688 product */
export async function analyzeProduct(
  input: AnalyzeInput,
): Promise<Result<AnalysisResult>> {
  return apiClient.post<AnalysisResult>(`${BASE}/analyze`, input);
}

/** List previous analyses */
export async function listAnalyses(params?: {
  status?: string;
  page?: number;
  size?: number;
}): Promise<PageResult<ProductAnalysis>> {
  const query: Record<string, string> = {};
  if (params?.status) query.status = params.status;
  if (params?.page) query.page = String(params.page);
  if (params?.size) query.size = String(params.size);
  return apiClient.getPage<ProductAnalysis>(`${BASE}/analyses`, query);
}

/** Get analysis detail */
export async function getAnalysis(
  id: number,
): Promise<Result<ProductAnalysis>> {
  return apiClient.get<ProductAnalysis>(`${BASE}/analyses/${id}`);
}

/** Submit feedback for an analysis */
export async function recordFeedback(
  id: number,
  input: FeedbackInput,
): Promise<Result<{ message: string }>> {
  return apiClient.post<{ message: string }>(
    `${BASE}/analyses/${id}/feedback`,
    input,
  );
}
