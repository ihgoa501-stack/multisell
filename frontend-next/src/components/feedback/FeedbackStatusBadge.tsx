'use client';

import { Tag } from 'antd';

const statusConfig: Record<string, { color: string; label: string }> = {
  pending:       { color: 'default',    label: '待审核' },
  under_review:  { color: 'processing', label: '审核中' },
  accepted:      { color: 'success',    label: '已采纳' },
  rejected:      { color: 'error',      label: '已拒绝' },
  planned:       { color: 'blue',       label: '已规划' },
  in_progress:   { color: 'orange',     label: '开发中' },
  shipped:       { color: 'purple',     label: '已上线' },
  declined:      { color: 'default',    label: '已关闭' },
};

const typeConfig: Record<string, { color: string; label: string }> = {
  bug:         { color: 'red',     label: 'Bug' },
  feature:     { color: 'blue',    label: '功能需求' },
  improvement: { color: 'green',   label: '改进建议' },
  question:    { color: 'gold',    label: '问题咨询' },
  other:       { color: 'default', label: '其他' },
};

const severityConfig: Record<string, { color: string; label: string }> = {
  critical: { color: 'red',     label: '严重' },
  major:    { color: 'orange',  label: '重要' },
  minor:    { color: 'blue',    label: '一般' },
  trivial:  { color: 'default', label: '轻微' },
};

export function StatusBadge({ status }: { status: string }) {
  const cfg = statusConfig[status] || { color: 'default', label: status };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export function TypeBadge({ type }: { type: string }) {
  const cfg = typeConfig[type] || { color: 'default', label: type };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export function SeverityBadge({ severity }: { severity: string }) {
  if (!severity) return null;
  const cfg = severityConfig[severity] || { color: 'default', label: severity };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export const feedbackStatusList = Object.entries(statusConfig).map(([k, v]) => ({ value: k, label: v.label }));
export const feedbackTypeList = Object.entries(typeConfig).map(([k, v]) => ({ value: k, label: v.label }));
export const feedbackSeverityList = Object.entries(severityConfig).map(([k, v]) => ({ value: k, label: v.label }));
