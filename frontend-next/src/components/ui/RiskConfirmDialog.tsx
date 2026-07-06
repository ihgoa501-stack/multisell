import { Modal, Descriptions, Tag, Typography } from 'antd';
import {
  ExclamationCircleOutlined,
  WarningOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';

const { Text } = Typography;

export type RiskLevel = 'low' | 'medium' | 'high';

export interface ChangeField {
  /** Human-readable field name */
  label: string;
  /** Value before the change */
  before: string;
  /** Value after the change */
  after: string;
}

export interface RiskConfirmDialogProps {
  /** Whether the dialog is visible */
  open: boolean;
  /** The target object being acted upon (e.g. product name, order SN) */
  target: string;
  /** Risk level of the operation */
  risk: RiskLevel;
  /** Fields that will change, shown as before/after pairs */
  changes: ChangeField[];
  /** Where the audit trail is recorded (e.g. "操作日志", "系统审计") */
  auditDestination?: string;
  /** Label for the confirm button */
  okText?: string;
  /** Label for the cancel button */
  cancelText?: string;
  /** Confirm button loading state */
  confirmLoading?: boolean;
  /** Callback on confirm */
  onOk: () => void;
  /** Callback on cancel */
  onCancel: () => void;
}

const riskMeta: Record<RiskLevel, { color: string; icon: React.ReactNode; label: string }> = {
  low: { color: '#52c41a', icon: <InfoCircleOutlined />, label: '低风险' },
  medium: { color: '#fa8c16', icon: <WarningOutlined />, label: '中风险' },
  high: { color: '#f5222d', icon: <ExclamationCircleOutlined />, label: '高风险' },
};

const riskBg: Record<RiskLevel, string> = {
  low: '#f6ffed',
  medium: '#fff7e6',
  high: '#fff1f0',
};

export default function RiskConfirmDialog({
  open,
  target,
  risk,
  changes,
  auditDestination = '操作日志',
  okText = '确认执行',
  cancelText = '取消',
  confirmLoading,
  onOk,
  onCancel,
}: RiskConfirmDialogProps) {
  const meta = riskMeta[risk];

  return (
    <Modal
      open={open}
      title={
        <span>
          <span style={{ color: meta.color, marginRight: 8 }}>{meta.icon}</span>
          确认操作
        </span>
      }
      okText={okText}
      cancelText={cancelText}
      okButtonProps={{ danger: risk === 'high' }}
      confirmLoading={confirmLoading}
      onOk={onOk}
      onCancel={onCancel}
      destroyOnHidden
      width={560}
    >
      <div style={{ padding: '8px 0' }}>
        {/* Target + Risk */}
        <div
          style={{
            background: riskBg[risk],
            borderRadius: 6,
            padding: '12px 16px',
            marginBottom: 16,
          }}
        >
          <div style={{ marginBottom: 4 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              目标对象
            </Text>
            <div style={{ fontWeight: 600, fontSize: 15, marginTop: 2 }}>{target}</div>
          </div>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              风险等级
            </Text>
            <div style={{ marginTop: 2 }}>
              <Tag color={meta.color} icon={meta.icon}>
                {meta.label}
              </Tag>
            </div>
          </div>
        </div>

        {/* Changes */}
        <Descriptions
          title="变更内容"
          column={1}
          size="small"
          bordered
          styles={{ label: { fontWeight: 500, width: 140 } }}
        >
          {changes.map((c, i) => (
            <Descriptions.Item key={i} label={c.label}>
              <Text type="secondary" delete style={{ marginRight: 12 }}>
                {c.before}
              </Text>
              <Text strong style={{ color: '#1890ff' }}>
                {c.after}
              </Text>
            </Descriptions.Item>
          ))}
        </Descriptions>

        {/* Audit */}
        <div style={{ marginTop: 12, fontSize: 12, color: '#999' }}>
          <InfoCircleOutlined style={{ marginRight: 4 }} />
          此操作将被记录到 <Text code>{auditDestination}</Text>
        </div>
      </div>
    </Modal>
  );
}
