import apiClient from '@/lib/api-client';
import type {
  XiaoQCapability,
  XiaoQIdentity,
  XiaoQMessageRequest,
  XiaoQMessageResponse,
} from './types';

export async function getXiaoQIdentity(): Promise<XiaoQIdentity> {
  const result = await apiClient.get<XiaoQIdentity>('/v1/xiao-q/identity');
  if (!result.data) throw new Error('小Q身份信息为空');
  return result.data;
}

export async function getXiaoQCapabilities(): Promise<XiaoQCapability[]> {
  const result = await apiClient.get<Array<XiaoQCapability | {
    id: string; description: string; risk: string; side_effect: boolean;
    required_permission?: string; status?: string; approval_required?: boolean; approval?: string;
  }>>('/v1/xiao-q/capabilities');
  return (result.data ?? []).map((item) => {
    if ('code' in item) return item;
    return {
      code: item.id,
      name: item.id,
      description: item.description,
      mode: item.side_effect ? 'suggestion' : 'read_only',
      available: item.status !== 'unavailable' && item.status !== 'disabled',
      required_permission: item.required_permission,
      status: item.status,
      approval_required: item.approval_required,
      approval: item.approval,
    };
  });
}

type BackendLinks = Partial<Record<'demand_case' | 'decision_card' | 'trace', string>>;

function normalizeLinks(
  links: XiaoQMessageResponse['links'] | BackendLinks | undefined,
  demandCaseID?: number,
): XiaoQMessageResponse['links'] {
  if (Array.isArray(links)) return links;
  const normalized: XiaoQMessageResponse['links'] = [];
  if (links?.demand_case) normalized.push({ label: '查看候选市场案件', href: links.demand_case });
  if (links?.decision_card) normalized.push({ label: '查看决策卡', href: links.decision_card });
  if (links?.trace) normalized.push({ label: '查看运行追踪', href: links.trace });
  if (normalized.length === 0 && demandCaseID) {
    normalized.push({ label: '查看候选市场案件', href: `/demand-cases/${demandCaseID}` });
  }
  return normalized;
}

export async function sendXiaoQMessage(
  request: XiaoQMessageRequest,
): Promise<XiaoQMessageResponse> {
  const result = await apiClient.post<XiaoQMessageResponse>('/v1/xiao-q/messages', request);
  if (!result.data) throw new Error('小Q没有返回回答');
  const data = result.data as Omit<XiaoQMessageResponse, 'links'> & {
    links?: XiaoQMessageResponse['links'] | BackendLinks;
    provider?: string; model?: string; demand_case_id?: number;
  };
  return {
    ...data,
    mode: data.mode ?? 'read_only',
    evidence: data.evidence ?? [],
    unknowns: data.unknowns ?? [],
    links: normalizeLinks(data.links, data.demand_case_id),
    provenance: data.provenance ?? (data.provider ? { source: data.provider, model: data.model } : undefined),
  };
}
