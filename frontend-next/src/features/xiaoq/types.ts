export type XiaoQTruthStatus =
  | 'actual'
  | 'quoted'
  | 'estimated'
  | 'inferred'
  | 'unknown'
  | 'mock';

export type XiaoQMode = 'read_only' | 'read_only_v1' | 'suggestion';

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

export type XiaoQMessageRequest =
  | XiaoQDemandCaseMessageRequest
  | XiaoQExperimentMessageRequest
  | XiaoQSourcing1688MessageRequest;

export interface XiaoQMessageResponse {
  trace_id: string;
  agent_id: string;
  answer: string;
  truth_status: XiaoQTruthStatus;
  mode: XiaoQMode;
  target_type?: 'demand_case' | 'experiment' | 'sourcing_1688';
  demand_case_id?: number;
  experiment_id?: string;
  source_id?: number;
  trusted?: boolean;
  case_summary?: string;
  evidence: XiaoQEvidence[];
  unknowns: string[];
  links: XiaoQLink[];
  provenance?: XiaoQProvenance | XiaoQProvenance[];
}
