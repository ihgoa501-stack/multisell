/** Candidate product types — mirrors backend domain/candidate/model.go */

export interface CandidateProduct {
  id: number;
  title: string;
  description: string;
  main_image: string;
  purchase_price: number;
  purchase_currency: string;
  package_weight_kg: number;
  target_sale_price: number;
  target_currency: string;
  target_platform_id: number | null;
  destination_country: string;
  hs_code: string;
  origin_country: string;
  status: string;
  completeness_status: string;
  is_seed_data: boolean;
  created_by: string;
  created_at: string;
  [key: string]: unknown; // extendable fields
}

export interface CandidateDetail extends CandidateProduct {
  missing_fields: string[];
}

export interface CollectLead {
  id: number;
  title: string;
  price_range: string;
  detail_url: string;
  image_url: string;
  source_page_url: string;
  status: string;
  created_at: string;
}

export interface CompletenessDimension {
  dimension: string;
  label: string;
  score: number;
  weight: number;
  complete: boolean;
  reason: string;
}

export interface CompletenessCheckResult {
  product_id: number;
  score: number;
  status: string;
  dimensions: CompletenessDimension[];
  missing_items: string[];
}

export interface EvaluateResult {
  product_id: number;
  title: string;
  completeness_score: number;
  completeness_status: string;
  missing_items: string[];
  profit_margin: number;
  estimated_profit: number;
  profit_status: string;
}
