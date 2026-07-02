import type { ReactNode } from 'react';

export interface WorkItemDetail {
  id: number;
  title: string;
  agent_id: string;
  squad_id: string;
  risk_level: string;
  status: string;
  confidence: number | null;
  proposed_at: string;
  decision_point: string;
  reason: string;
  input_summary: string;
  output_summary: string;
  entity_type: string;
  entity_id: number | null;
  entity_status: string;
  approval: {
    id: number;
    status: string;
    risk_level: string;
  } | null;
  trace_id: string | null;
  upstream_items: Array<{ id: number; type: string; title: string; status: string }>;
  downstream_items: Array<{ id: number; type: string; title: string; status: string }>;
  audit_logs: Array<{ id: number; action: string; content: string; operator: string; created_at: string }>;
}
