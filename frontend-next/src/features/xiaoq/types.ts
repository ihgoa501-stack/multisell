export type XiaoQTruthStatus =
  | 'actual'
  | 'quoted'
  | 'estimated'
  | 'inferred'
  | 'unknown'
  | 'mock'
  | 'external_observed'
  | 'reconciled';

export type XiaoQMode = 'read_only' | 'read_only_v1' | 'read_only_v2' | 'agent_runtime_v1' | 'suggestion' | 'decision_support_v1';

export interface XiaoQIdentity {
  agent_id: string;
  name: string;
  description?: string;
  mode: XiaoQMode;
  truth_status?: XiaoQTruthStatus;
}

export interface XiaoQCapability {
  code: string;
  name: string;
  description?: string;
  mode: XiaoQMode;
  available: boolean;
  unavailable_reason?: string;
  truth_status?: XiaoQTruthStatus;
  required_permission?: string;
  status?: string;
  approval_required?: boolean;
  approval?: string;
}

export interface XiaoQEvidence {
  id?: string | number;
  title: string;
  summary?: string;
  truth_status: XiaoQTruthStatus;
  source_url?: string;
  observed_at?: string;
  run_id?: string | number;
  snapshot_id?: string | number;
  snapshot_sha256?: string;
}

export interface XiaoQLink {
  label: string;
  href: string;
}

export interface XiaoQProvenance {
  source?: string;
  source_type?: string;
  observed_at?: string;
  generated_at?: string;
  [key: string]: unknown;
}

export interface XiaoQDemandCaseMessageRequest {
  message: string;
  demand_case_id: number;
  target_type?: 'demand_case';
}

export interface XiaoQExperimentMessageRequest {
  message: string;
  target_type: 'experiment';
  experiment_id: string;
}

export interface XiaoQSourcing1688MessageRequest {
  message: string;
  target_type: 'sourcing_1688';
  source_id: number;
}

export interface XiaoQOperatingFactsMessageRequest {
  message: string;
  target_type: 'operating_facts';
  order_id: number;
}

export interface XiaoQBusinessDecisionMessageRequest {
  message: string;
  target_type: 'business_decision';
  decision_case_id: number;
  create_recommendation?: boolean;
  idempotency_key?: string;
}

export type XiaoQMessageRequest =
  | XiaoQDemandCaseMessageRequest
  | XiaoQExperimentMessageRequest
  | XiaoQSourcing1688MessageRequest
  | XiaoQOperatingFactsMessageRequest
  | XiaoQBusinessDecisionMessageRequest;

export interface XiaoQMessageResponse {
  trace_id: string;
  agent_id: string;
  answer: string;
  truth_status: XiaoQTruthStatus;
  mode: XiaoQMode;
  target_type?: 'demand_case' | 'experiment' | 'sourcing_1688' | 'business_closure' | 'operating_facts' | 'business_decision';
  demand_case_id?: number;
  experiment_id?: string;
  source_id?: number;
  order_id?: number;
  decision_case_id?: number;
  recommendation_id?: number;
  trusted?: boolean;
  case_summary?: string;
  evidence: XiaoQEvidence[];
  unknowns: string[];
  blockers?: string[];
  links: XiaoQLink[];
  provenance?: XiaoQProvenance | XiaoQProvenance[];
}
