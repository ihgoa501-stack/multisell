import type { EvidenceTruth, ExperimentStage, GateResult } from '@/types/experiment';

export const stageLabels: Record<ExperimentStage, string> = {
  opportunity: '机会与需求', product: '商品定义', supply: '供应与合规', channel: '渠道准备',
  order: '成交与支付', fulfillment: '履约交付', aftersales: '售后责任', profit: '最终利润',
  cash: '现金回收', decision: '经营决策',
};

export const truthMeta: Record<EvidenceTruth, { label: string; color: string; trustedForHighRisk: boolean }> = {
  actual: { label: '真实发生', color: 'green', trustedForHighRisk: true },
  quoted: { label: '有效报价', color: 'cyan', trustedForHighRisk: true },
  estimated: { label: '估算', color: 'gold', trustedForHighRisk: true },
  unknown: { label: '未知', color: 'default', trustedForHighRisk: false },
  mock: { label: '模拟', color: 'magenta', trustedForHighRisk: false },
  inferred: { label: 'AI 推断', color: 'purple', trustedForHighRisk: false },
};

export const gateMeta: Record<GateResult, { label: string; color: string }> = {
  pass: { label: '通过', color: 'green' }, conditional: { label: '附条件通过', color: 'gold' },
  return: { label: '退回补证', color: 'orange' }, reject: { label: '淘汰', color: 'red' },
  expired: { label: '证据过期', color: 'default' },
};

export const stageGateCodes: Record<ExperimentStage, string> = {
  opportunity: 'demand_evidence', product: 'spec_ready', supply: 'supply_ready',
  channel: 'channel_ready', order: 'paid_order', fulfillment: 'delivered',
  aftersales: 'aftersales_closed', profit: 'profit_final', cash: 'cash_recovered',
  decision: 'final_decision',
};

export function nextExperimentStage(stage: ExperimentStage): ExperimentStage | null {
  const stages = Object.keys(stageLabels) as ExperimentStage[];
  return stages[stages.indexOf(stage) + 1] ?? null;
}

export function formatBlocker(blocker: string): string {
  if (blocker === 'required_gates_incomplete') return '必要经营闸门尚未完成';
  const [gate, result] = blocker.split(':');
  return `${gate}${result ? ` · ${gateMeta[result as GateResult]?.label ?? result}` : ''}`;
}
