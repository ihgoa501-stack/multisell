export interface ProblemCase {id:number;region:string;observable_population:string;problem_scenario:string;current_workaround:string;responsibility:string;product_solvability:string;harm_risk:string;next_minimum_evidence:string;status:'lead'|'evidence_missing'|'survives_falsification'|'rejected'}
export interface ProblemEvidence{id:number;kind:'support'|'counter';title:string;source_uri:string;observed_at:string;collector:string;raw_sha256:string;raw_payload:string;trusted_run:boolean}
export interface ProblemDetail{case:ProblemCase;evidence:ProblemEvidence[]}
export interface ReviewedProblemBatchOutcome{batch_key:string;problems:ProblemCase[];status_counts:Record<string,number>;paid_demand_status:'unknown';selected_items:number;selected_channels:number}
