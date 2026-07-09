'use client';

import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';

type RiskLevel = 'low' | 'medium' | 'high';
type EnvironmentMode = 'dry_run' | 'sandbox' | 'production';
export type ActionRiskConfirmMode = 'approve' | 'reject' | 'execute';

export interface RiskConfirmAction {
  id: number | string;
  title: string;
  action_type?: string;
  agent_id?: string;
  risk_level?: string;
  requires_approval?: boolean;
  execution_mode?: string;
  trace_id?: string;
  description?: string;
  before_snapshot?: Record<string, unknown> | null;
  after_snapshot?: Record<string, unknown> | null;
  payload?: Record<string, unknown>;
}

interface Props {
  action: RiskConfirmAction | null;
  mode: ActionRiskConfirmMode | null;
  open: boolean;
  loading?: boolean;
  onCancel: () => void;
  onConfirm: (action: RiskConfirmAction, reason?: string) => void;
}

const actionNames: Record<ActionRiskConfirmMode, string> = {
  approve: '批准 AI 建议',
  reject: '拒绝 AI 建议',
  execute: '执行 AI 动作',
};

const confirmTexts: Record<ActionRiskConfirmMode, string> = {
  approve: '确认批准',
  reject: '确认拒绝',
  execute: '确认执行',
};

function normalizeRisk(level?: string): RiskLevel {
  if (level === 'low' || level === 'medium' || level === 'high') return level;
  if (level === 'critical') return 'high';
  return 'medium';
}

function normalizeMode(mode?: string): EnvironmentMode {
  if (mode === 'dry_run' || mode === 'sandbox' || mode === 'production') return mode;
  return 'production';
}

function compactJson(value?: Record<string, unknown> | null): string | undefined {
  if (!value || Object.keys(value).length === 0) return undefined;
  return JSON.stringify(value);
}

function expectedConsequence(mode: ActionRiskConfirmMode, action: RiskConfirmAction): string {
  if (mode === 'reject') return '该建议会被标记为已拒绝，不会继续执行。';
  if (mode === 'approve') {
    return action.requires_approval === false
      ? '该建议会被记录为已批准；后续执行仍由执行入口触发。'
      : '该建议会绑定当前登录用户审批身份，进入可执行状态。';
  }
  return normalizeMode(action.execution_mode) === 'production'
    ? '后端会按统一执行门禁执行动作；高风险生产动作必须已审批。'
    : '动作会在模拟或沙箱模式下执行，不应产生真实外部业务影响。';
}

export default function ActionRiskConfirmDialog({
  action,
  mode,
  open,
  loading,
  onCancel,
  onConfirm,
}: Props) {
  if (!action || !mode) return null;

  return (
    <HighRiskConfirmDialog
      open={open}
      actionName={actionNames[mode]}
      riskLevel={normalizeRisk(action.risk_level)}
      environmentMode={normalizeMode(action.execution_mode)}
      requiresApproval={action.requires_approval ?? normalizeRisk(action.risk_level) === 'high'}
      detail={{
        targetLabel: `${action.title} #${action.id}${action.action_type ? ` / ${action.action_type}` : ''}${action.agent_id ? ` / ${action.agent_id}` : ''}`,
        beforeValue: compactJson(action.before_snapshot),
        afterValue: compactJson(action.after_snapshot),
      }}
      expectedConsequence={expectedConsequence(mode, action)}
      auditDestination={
        action.trace_id
          ? `审批和执行结果会写入 Action 审计，Trace: ${action.trace_id}`
          : '审批和执行结果会写入 Action 审计。'
      }
      rollbackNote={
        mode === 'execute'
          ? '生产执行后的业务回滚需按对应领域流程处理；本确认不会绕过后端门禁。'
          : '批准或拒绝只改变建议状态，不直接修改价格、库存、订单或外部平台。'
      }
      confirmLoading={loading}
      confirmText={confirmTexts[mode]}
      showReason
      reasonPlaceholder={mode === 'reject' ? '拒绝原因（建议填写）' : '补充说明（选填）'}
      onConfirm={(reason) => onConfirm(action, reason)}
      onCancel={onCancel}
    />
  );
}
