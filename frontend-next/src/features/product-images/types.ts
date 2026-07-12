export type ImageProcessorCode = 'deterministic' | 'photoroom' | 'adobe' | 'openai';

export interface ProductImageAsset {
  id: number;
  owner_id: number;
  blob_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  sha256: string;
  truth: string;
  created_at?: string;
}

export interface ImageProcessorCapability {
  code: ImageProcessorCode;
  name: string;
  configured: boolean;
  operations: string[];
  reason?: string;
}

export interface ProductImageOutput {
  id: string;
  asset_id: string;
  url?: string;
  sha256?: string;
  width?: number;
  height?: number;
}

export type ProductImageJobStatus =
  | 'created'
  | 'pending'
  | 'queued'
  | 'running'
  | 'completed'
  | 'succeeded'
  | 'failed'
  | 'reconcile_required'
  | 'QUEUED'
  | 'RUNNING'
  | 'READY'
  | 'FAILED';

export interface ProductImageJob {
  id: number;
  owner_id: number;
  asset_id: number;
  image_service_job_id?: string;
  idempotency_key: string;
  manifest_hash: string;
  operation: string;
  processor?: string;
  version?: number;
  width: number;
  height: number;
  format: string;
  status: ProductImageJobStatus;
  output_blob_id?: string;
  output_url?: string;
  outputs?: ProductImageOutput[];
  error_code?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateProductImageJobInput {
  asset_id: number;
  idempotency_key: string;
  operation: 'DETERMINISTIC_RESIZE';
  width: number;
  height: number;
  format: 'png' | 'jpeg';
}

export type ProductImageRole = 'main' | 'gallery' | 'detail' | 'size' | 'packaging' | 'ad_cover';

export interface ProductImageSetItemInput { task_id: number; role: ProductImageRole; ordinal: number; }
export interface ProductImageSetItem { id: number; role: ProductImageRole; ordinal: number; locale: string; channel: string; asset_sha256: string; }
export interface ProductImageSet {
  id: number; listing_id: number; channel: string; locale: string; version: number;
  status: 'draft' | 'frozen'; manifest_sha256?: string; items: ProductImageSetItem[];
}
export interface CreateProductImageSetInput { listing_id: number; channel: string; locale: string; items: ProductImageSetItemInput[]; }

export type ImageGateStatus = 'passed' | 'blocked' | 'unknown';

export interface ProductImageRightsGrant {
  id: number; asset_sha256: string; purpose: string; jurisdiction: string; channel: string;
  provider: string; region: string; grantor: string; owner_verified: boolean; version: number;
  revoked_at?: string;
}

export interface CreateRightsGrantInput {
  asset_sha256: string; can_copy: boolean; can_modify: boolean; can_third_party_ai: boolean;
  can_cross_border: boolean; can_commercial_publish: boolean; can_platform_sublicense: boolean;
  trademark_cleared: boolean; likeness_cleared: boolean; purpose: string; jurisdiction: string;
  channel: string; provider: string; region: string; grantor: string; rights_chain: string;
  evidence_sha256: string; owner_verified: boolean; valid_from: string; idempotency_key: string;
  expected_version: 1;
}

export interface FiveAxisReviewInput {
  asset_sha256: string; purpose: string; channel: string; product_authenticity: ImageGateStatus;
  rights: ImageGateStatus; channel_rules: ImageGateStatus; claims_scene: ImageGateStatus;
  technical_visual: ImageGateStatus; evidence_sha256: string; evidence_truth: 'quoted' | 'inferred' | 'unknown';
  notes?: string; idempotency_key: string; expected_version: number;
}

export interface ProductImageReview extends FiveAxisReviewInput { id: number; task_id: number; verified_at?: string; }

export interface CreateCostEntryInput {
  kind: 'estimated' | 'actual'; category: string; provider: string; amount: string; currency: string;
  exchange_rate: string; exchange_rate_source: string; observed_at: string;
  billing_status: 'estimated' | 'pending' | 'invoiced' | 'paid' | 'reconciled' | 'unknown';
  evidence_sha256?: string; idempotency_key: string; expected_version: number;
}

export interface ProductImageCostEntry extends CreateCostEntryInput { id: number; task_id: number; }
