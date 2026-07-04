import apiClient from "@/lib/api-client";
import type { Result, PageResult } from "@/types/api";

export interface UnifiedAction {
  id: number;
  trace_id: string;
  agent_id: string;
  squad_id: string;
  user_id: number | null;
  action_type: string;
  title: string;
  description: string;
  risk_level: string;
  requires_approval: boolean;
  status: string;
  confidence: number | null;
  proposed_by: string;
  approved_by: string | null;
  rejected_by: string | null;
  executed_by: string | null;
  rejection_reason: string | null;
  payload: Record<string, unknown>;
  before_snapshot: Record<string, unknown> | null;
  after_snapshot: Record<string, unknown> | null;
  proposed_at: string;
  approved_at: string | null;
  rejected_at: string | null;
  executed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ActionListParams {
  page?: number;
  size?: number;
  status?: string;
  risk_level?: string;
  agent_id?: string;
  squad_id?: string;
  search?: string;
}

export interface PolicyRule {
  id: number;
  name: string;
  description: string;
  risk_level: string;
  action_type: string;
  squad_id: string;
  agent_id: string;
  business_object_type: string;
  max_amount: number | null;
  max_quantity: number | null;
  min_confidence: number | null;
  auto_approve: boolean;
  outcome: string;
  priority: number;
  enabled: boolean;
}

export async function fetchActions(params: ActionListParams = {}): Promise<PageResult<UnifiedAction>> {
  const query: Record<string, string> = {};
  if (params.page) query["page"] = String(params.page);
  if (params.size) query["size"] = String(params.size);
  if (params.status) query["status"] = params.status;
  if (params.risk_level) query["risk_level"] = params.risk_level;
  if (params.agent_id) query["agent_id"] = params.agent_id;
  if (params.search) query["search"] = params.search;
  return apiClient.getPage<UnifiedAction>("/v1/ai/actions", query);
}

export async function fetchAction(id: number): Promise<Result<UnifiedAction>> {
  return apiClient.get<UnifiedAction>("/v1/ai/actions/" + id);
}

export async function approveAction(id: number, reason: string): Promise<Result<UnifiedAction>> {
  return apiClient.post<UnifiedAction>("/v1/ai/actions/" + id + "/approve", { reason });
}

export async function rejectAction(id: number, reason: string): Promise<Result<UnifiedAction>> {
  return apiClient.post<UnifiedAction>("/v1/ai/actions/" + id + "/reject", { reason });
}

export async function executeAction(id: number): Promise<Result<UnifiedAction>> {
  return apiClient.post<UnifiedAction>("/v1/ai/actions/" + id + "/execute", {});
}

export async function fetchPolicyRules(): Promise<Result<PolicyRule[]>> {
  return apiClient.get<PolicyRule[]>("/v1/policy/rules");
}
