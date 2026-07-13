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
  availability?: 'available' | 'unavailable';
  paid?: boolean;
  reconcile_safe?: boolean;
  safety_level?: string;
  provider_environment?: string;
  region?: string;
  watermarked?: boolean;
  non_publishable?: boolean;
  quota_available?: boolean;
  quota_remaining?: number;
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
  | 'FAILED'
  | 'RECONCILE_REQUIRED';

export interface ProductImageJob {
  id: number;
  owner_id: number;
  asset_id: number;
  sku_id?: number;
  recipe_key?: string;
  recipe_version?: number;
  recipe_manifest?: ProductImageRecipe;
  recipe_hash?: string;
  parent_task_id?: number;
  candidate_round?: number;
  image_service_job_id?: string;
  idempotency_key: string;
  manifest_hash: string;
  operation: string;
  processor?: string;
  purpose?: string;
  channel?: string;
  region?: string;
  provider_environment?: string;
  max_cost?: string;
  currency?: string;
  sandbox?: boolean;
  watermarked?: boolean;
  non_publishable?: boolean;
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

export type ProductImageOperation =
  | 'DETERMINISTIC_RESIZE'
  | 'PHOTOROOM_REMOVE_BACKGROUND_SANDBOX'
  | 'PHOTOROOM_WHITE_BACKGROUND_SANDBOX'
  | 'PHOTOROOM_AI_SHADOW_SANDBOX'
  | 'OPENAI_IMAGE_EDIT';

export interface CreateProductImageJobInput {
  asset_id: number;
  sku_id: number;
  recipe_key: string;
  recipe_version: number;
  recipe: ProductImageRecipe;
  parent_task_id?: number;
  candidate_round: number;
  idempotency_key: string;
  operation: ProductImageOperation;
  processor: 'deterministic' | 'photoroom' | 'openai';
  purpose: string;
  channel: string;
  region: 'local' | 'us';
  width: number;
  height: number;
  format: 'png' | 'jpeg';
  max_cost?: string;
  currency?: 'USD';
}

export interface ProductImageRecipe {
  reference_asset_ids?: number[];
  mask_asset_id?: number;
  scene_structure: string;
  prompt?: string;
  negative_prompt?: string;
  model: string;
  model_version: string;
  parameters: Record<string, unknown>;
  must_not_change: string[];
}

export type CandidateOutcome = 'selected' | 'rejected' | 'rework_requested';
export type CandidateReasonCode = 'product_structure' | 'color' | 'text_logo' | 'quantity_accessories' | 'scene' | 'visual_quality' | 'other';
export interface CandidateFeedbackInput {
  outcome: CandidateOutcome; reason_codes?: CandidateReasonCode[]; error_regions?: Array<Record<string, unknown>>;
  rework_instruction?: string; review_seconds: number; notes?: string; asset_sha256: string;
  idempotency_key: string; expected_version: number;
}
export interface CandidateFeedback extends CandidateFeedbackInput { id: number; task_id: number; verified_at?: string; }
export interface RecipeSummary {
  recipe_key: string; sku_id: number; purpose: string; channel: string; latest_recipe_version: number;
  candidates: number; selected: number; rejected: number; rework_requested: number; acceptance_rate: number;
  review_seconds: number; production_seconds: number; rework_rounds: number; actual_cost: string; currency?: string;
}

export interface SKUOption { id: number; code: string; spec_desc: string; product_id: number; }

export type ProductImageRole = 'main' | 'gallery' | 'detail' | 'size' | 'packaging' | 'ad_cover';

export interface ProductImageSetItemInput { task_id: number; role: ProductImageRole; ordinal: number; }
export interface ProductImageSetItem { id: number; role: ProductImageRole; ordinal: number; locale: string; channel: string; asset_sha256: string; }
export interface ProductImageSet {
  id: number; listing_id: number; channel: string; locale: string; version: number;
  status: 'draft' | 'frozen'; manifest_sha256?: string; items: ProductImageSetItem[];
}
export interface CreateProductImageSetInput { listing_id: number; channel: string; locale: string; items: ProductImageSetItemInput[]; }

export interface ImageRuleSnapshot {
  id: number; owner_id: number; channel: string; site: string; locale: string; category_id: number;
  version: number; rules: Record<string, unknown>; rules_sha256: string; effective_at: string;
  expires_at?: string; idempotency_key: string; created_at?: string;
}
export interface CreateImageRuleSnapshotInput {
  channel: string; site: string; locale: string; category_id: number; rules: Record<string, unknown>;
  effective_at: string; expires_at?: string; idempotency_key: string;
}
export interface ImageSetDecision {
  id: number; owner_id: number; image_set_id: number; image_set_version: number;
  set_manifest_sha256: string; decision: 'approved' | 'rejected'; reason: string; decided_at: string;
}
export interface DecideImageSetInput {
  decision: 'approved' | 'rejected'; reason: string; expected_version: number; idempotency_key: string;
}
export interface ImageReleaseAttestationItem {
  id: number; ordinal: number; role: ProductImageRole; task_id: number; blob_id: string;
  sha256: string; mime: string; width: number; height: number;
}
export interface ImageReleaseAttestation {
  id: number; owner_id: number; listing_id: number; product_id: number; platform_id: number;
  platform_account_id: number; channel: string; site: string; locale: string; category_id: number;
  listing_snapshot_sha256: string; image_set_id: number; image_set_version: number;
  set_manifest_sha256: string; rule_snapshot_id: number; rule_snapshot_sha256: string;
  set_decision_id: number; status: 'issued' | 'consumed' | 'revoked'; issued_at: string;
  expires_at: string; consumed_at?: string; items: ImageReleaseAttestationItem[];
}
export interface IssueImageReleaseAttestationInput {
  image_set_id: number; rule_snapshot_id: number; platform_account_id: number; site: string;
  ttl_seconds: number; idempotency_key: string;
}

export type ImageGateStatus = 'passed' | 'blocked' | 'unknown';

export interface ProductImageRightsGrant {
  id: number; asset_id?: number; asset_sha256: string; purpose: string; jurisdiction: string; channel: string;
  provider: string; region: string; grantor: string; owner_verified: boolean; version: number;
  revoked_at?: string;
}

export interface CreateRightsGrantInput {
  asset_id?: number; asset_sha256: string; can_copy: boolean; can_modify: boolean; can_third_party_ai: boolean;
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

export interface ImageBudgetPolicy {
  id: number; owner_id: number; currency: string; period_start: string; period_end: string;
  total_amount: string; idempotency_key: string; created_at?: string;
}

export type ImageBudgetReservationState = 'reserved' | 'claimed' | 'spent' | 'released' | 'no_charge';
export interface ImageBudgetReservation {
  id: number; owner_id: number; policy_id: number; approval_id: number; task_id: number;
  task_version: number; manifest_hash: string; provider: string; currency: string;
  reserved_amount: string; state: ImageBudgetReservationState; claimed_at?: string;
  released_at?: string; release_reason?: string; created_at?: string; updated_at?: string;
}

export interface ImageBudgetCharge {
  id: number; owner_id: number; reservation_id: number; amount: string; delta_amount: string;
  currency: string; kind: string; over_budget: boolean; evidence_sha256: string;
  observed_at: string; idempotency_key: string; created_at?: string;
}

export interface CreateImageBudgetPolicyInput {
  currency: 'USD' | 'EUR' | 'CNY' | 'GBP' | 'JPY'; period_start: string; period_end: string;
  total_amount: string; idempotency_key: string;
}

export interface ReconcileImageBudgetChargeInput {
  amount: string; currency: string; evidence_sha256: string; observed_at: string; idempotency_key: string;
  resolution?: 'charged_no_output';
}

export interface ReconcileImageBudgetNoChargeInput {
  evidence_sha256: string; observed_at: string; reason: string; idempotency_key: string;
}

export type ManualImportKind = 'manual_import' | 'channel_native_import';
export interface CreateManualImportInput {
  file: File; parent_asset_id: number; parent_asset_sha256: string; import_kind: ManualImportKind;
  tool: string; operation: string; fee_amount: string; fee_currency: 'USD' | 'EUR' | 'CNY' | 'GBP' | 'JPY';
  model: string; model_version: string; original_channel?: string; channel_restriction: string;
  source_observed_at: string; idempotency_key: string;
}
export interface ProductImageManualImport {
  id: number; asset_id: number; asset_sha256: string; parent_asset_id: number; parent_asset_sha256: string;
  import_kind: ManualImportKind; tool: string; operation: string; fee_amount: string; fee_currency: string;
  model: string; model_version: string; original_channel?: string; channel_restriction: string;
  source_observed_at: string; truth: 'unknown'; idempotency_key: string; created_at?: string;
}
