export type DemandVerdict = 'lead' | 'evidence_missing' | 'rejected' | 'experiment_ready';
export interface DemandCase { id:number; region:string; consumer:string; need_scenario:string; sales_channel:string; target_locale:string; stop_condition:string; status:DemandVerdict; created_at:string; }
export interface DemandEvidence { id:number; dimension:string; kind:'support'|'counter'|'conflict'; truth_status:string; title:string; source_uri:string; observed_at?:string; run_id:string; snapshot_id:number; }
export interface ResearchSnapshot { id:number; run_id:string; run_type:string; source_uri:string; collected_at:string; raw_sha256:string; }
export interface DemandDetail { case:DemandCase; evidence:DemandEvidence[]; snapshots:ResearchSnapshot[]; verdict?:{status:DemandVerdict;blockers:string[];reason:string}; }
export interface OwnerDecisionCard { demand_case_id:number; verdict:DemandVerdict; hypothesis:string; proven:string; not_proven:string; strongest_counterevidence:string; next_authority_or_cost:string; stop_condition:string; }
