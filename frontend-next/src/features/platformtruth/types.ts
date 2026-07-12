export type TruthLevel = { code: string; meaning: string; can_be_direct: boolean };
export type ClaimLevel = { code: string; meaning: string };
export type SystemBoundary = { code: string; name: string; responsibility: string; must_not: string };
export type ContractRule = { code: string; rule: string };
export type DomainDisposition = {
  id: string; name: string; system: string; disposition: string; reason: string;
  evidence: string; xiao_q_support: string; owner_scope: string; risk: string;
};
export type PlatformTruth = {
  version: string; direction: string; truth_levels: TruthLevel[]; claim_levels: ClaimLevel[];
  system_boundaries: SystemBoundary[]; domain_dispositions: DomainDisposition[];
  object_identity_rules: ContractRule[]; source_rules: ContractRule[];
  boundary_rules: string[]; unknowns: string[];
};
