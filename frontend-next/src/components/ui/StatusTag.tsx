import { Tag } from 'antd';

/** Preset color maps for common status values across the app. */
export const STATUS_COLOR_MAP: Record<string, string> = {
  // Entity statuses
  active: 'green',
  inactive: 'default',
  draft: 'default',
  archived: 'red',
  // Agent / action statuses
  pending: 'orange',
  suggested: 'blue',
  approved: 'green',
  rejected: 'red',
  executing: 'cyan',
  executed: 'green',
  failed: 'red',
  reviewed: 'default',
  // Risk levels
  critical: 'red',
  high: 'red',
  medium: 'orange',
  low: 'green',
  // Health
  ok: 'green',
  warn: 'orange',
  // Autonomy
  advisory: 'green',
  guided: 'blue',
  autonomous: 'cyan',
  supervised: 'orange',
  // Settlement
  reconciled: 'green',
  unreconciled: 'orange',
  // Generic
  enabled: 'green',
  disabled: 'default',
  yes: 'green',
  no: 'default',
  on: 'green',
  off: 'default',
  success: 'green',
  error: 'red',
  warning: 'orange',
  info: 'blue',
};

export interface StatusTagProps {
  status: string;
  /** Optional override color (Ant Design preset or hex). */
  color?: string;
  /** Custom label. Defaults to status value. */
  label?: string;
  /** Size variant. */
  size?: 'small' | 'default';
}

export default function StatusTag({ status, color, label, size = 'default' }: StatusTagProps) {
  const resolvedColor = color ?? STATUS_COLOR_MAP[status.toLowerCase()] ?? 'default';

  return (
    <Tag
      color={resolvedColor}
      style={size === 'small' ? { fontSize: 11, lineHeight: '18px', padding: '0 4px' } : undefined}
    >
      {label ?? status}
    </Tag>
  );
}
