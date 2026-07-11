export const experimentStages = [
  'opportunity', 'product', 'supply', 'channel', 'order',
  'fulfillment', 'aftersales', 'profit', 'cash', 'decision',
] as const;

export type ExperimentStage = (typeof experimentStages)[number];
export type EvidenceTruth = 'actual' | 'quoted' | 'estimated' | 'unknown' | 'mock' | 'inferred';
export type GateResult = 'pass' | 'conditional' | 'return' | 'reject' | 'expired';
export type EvidenceKind = 'support' | 'counter' | 'conflict';

export interface ExperimentCase {
  id: number;
  experiment_id: string;
  name: string;
  stage: ExperimentStage;
  evidence_kind: EvidenceKind;
  status: string;
  final_profit_status: string;
  final_revenue: number;
  final_total_cost: number;
  final_profit_amount: number;
  profit_currency: string;
  cash_recovery_status: string;
  cash_recovered_amount: number;
  cash_currency: string;
  cash_recovered_at?: string;
  final_decision: string;
  owner_id: number;
  created_at: string;
  updated_at: string;
}

export interface ExperimentEvidence {
  id: number;
  experiment_id: string;
  stage: ExperimentStage;
  truth_status: EvidenceTruth;
  title: string;
  source_uri?: string;
  observed_at?: string;
  expires_at?: string;
  created_at: string;
}

export interface ExperimentGate {
  id: number;
  experiment_id: string;
  stage: ExperimentStage;
  gate_code: string;
  result: GateResult;
  reason: string;
  decided_by: number;
  created_at: string;
}

export interface ExperimentObjectLink {
  id: number;
  experiment_id: string;
  object_type: string;
  object_id: string;
  created_at: string;
}

export interface ExperimentDetail {
  case: ExperimentCase;
  gates: ExperimentGate[];
  evidence: ExperimentEvidence[];
  object_links: ExperimentObjectLink[];
}

export interface ExperimentOwnerSummary {
  experiment_id: string;
  stage: ExperimentStage;
  passed_gates: number;
  blockers: string[];
  final_profit_status: string;
  final_revenue: number;
  final_total_cost: number;
  final_profit_amount: number;
  profit_currency: string;
  cash_recovery_status: string;
  cash_recovered_amount: number;
  cash_currency: string;
  cash_recovered_at?: string;
  final_decision: string;
}
