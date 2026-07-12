export type DemandVerdict = 'lead' | 'evidence_missing' | 'rejected' | 'experiment_ready';
export interface DemandCase { id:number; region:string; consumer:string; need_scenario:string; sales_channel:string; target_locale:string; stop_condition:string; status:DemandVerdict; created_at:string; }
export interface DemandEvidence { id:number; dimension:string; kind:'support'|'counter'|'conflict'; truth_status:string; title:string; source_uri:string; observed_at?:string; run_id:string; snapshot_id:number; }
export interface ResearchSnapshot { id:number; run_id:string; run_type:string; source_uri:string; collected_at:string; raw_sha256:string; }
export interface DemandDetail { case:DemandCase; evidence:DemandEvidence[]; snapshots:ResearchSnapshot[]; verdict?:{status:DemandVerdict;blockers:string[];reason:string}; }
export interface OwnerDecisionCard { demand_case_id:number; verdict:DemandVerdict; hypothesis:string; proven:string; not_proven:string; strongest_counterevidence:string; next_authority_or_cost:string; stop_condition:string; }
export type MarketDecisionValue = 'selected'|'rejected'|'paused'|'request_more_evidence';
export interface MarketOwnerDecision { id:number; demand_case_id:number; owner_id:number; verdict_id:number; decision:MarketDecisionValue; reason:string; evidence_hash:string; created_at:string; }
export type ProductOpportunityStatus = 'draft'|'evidence_missing'|'ready_for_owner'|'approved'|'rejected'|'paused';
export interface ProductOpportunity { id:number; owner_id:number; demand_case_id:number; market_decision_id:number; title:string; consumer_problem:string; product_thesis:string; target_channel:string; value_hypothesis:string; price_hypothesis:string; source_uri:string; truth_status:'quoted'|'estimated'; strongest_counterevidence:string; unknowns:string[]; stop_condition:string; status:ProductOpportunityStatus; version:number; content_hash:string; created_at:string; updated_at:string; }
