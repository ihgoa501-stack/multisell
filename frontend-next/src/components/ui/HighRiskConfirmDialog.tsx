'use client';

import { useState } from 'react';
import { Modal, Descriptions, Tag, Typography, Input, Divider, Alert } from 'antd';
import {
  ExclamationCircleOutlined,
  AuditOutlined,
  RollbackOutlined,
} from '@ant-design/icons';

const { Text } = Typography;
const { TextArea } = Input;

export interface HighRiskActionDetail {
  /** What the operation targets */
  targetLabel: string;
  /** The before and after values, if applicable */
  beforeValue?: string;
  afterValue?: string;
}

export interface HighRiskConfirmDialogProps {
  open: boolean;
  /** Action being performed (execute, approve, publish, etc.) */
  actionName: string;
  /** Risk level. @default 'medium' */
  riskLevel?: 'low' | 'medium' | 'high';
  /** Operation detail — target, before/after */
  detail?: HighRiskActionDetail;
  /** Environment mode. @default 'production' */
  environmentMode?: 'dry_run' | 'sandbox' | 'production';
  /** Whether this action requires explicit approval. @default true */
  requiresApproval?: boolean;
  /** Expected consequence description */
  expectedConsequence?: string;
  /** Audit destination note */
  auditDestination?: string;
  /** Rollback / recovery note */
  rollbackNote?: string;
  /** Confirm loading state */
  confirmLoading?: boolean;
  /** Confirm button label. @default '确认执行' */
  confirmText?: string;
  /** Called on confirm */
  onConfirm: (reason?: string) => void;
  /** Called on cancel */
  onCancel: () => void;
  /** Whether to show a reason field. @default false */
  showReason?: boolean;
  /** Reason placeholder text */
  reasonPlaceholder?: string;
}

const riskColors: Record<string, string> = {
  low: '#52c41a',
  medium: '#fa8c16',
  high: '#f5222d',
};

const riskLabels: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
};

const modeColors: Record<string, string> = {
  dry_run: '#722ed1',
  sandbox: '#1890ff',
  production: '#f5222d',
};

const modeLabels: Record<string, string> = {
  dry_run: 'Dry-Run（模拟）',
  sandbox: 'Sandbox（沙箱）',
  production: 'Production（生产）',
};

export default function HighRiskConfirmDialog({
  open,
  actionName,
  riskLevel = 'medium',
  detail,
  environmentMode = 'production',
  requiresApproval = true,
  expectedConsequence,
  auditDestination,
  rollbackNote,
  confirmLoading,
  confirmText = '确认执行',
  onConfirm,
  onCancel,
  showReason,
  reasonPlaceholder = '补充说明（选填）',
}: HighRiskConfirmDialogProps) {
  const [reason, setReason] = useState('');

  const handleConfirm = () => {
    onConfirm(showReason ? reason : undefined);
    if (showReason) setReason('');
  };

  const handleCancel = () => {
    setReason('');
    onCancel();
  };

  return (
    <Modal
      open={open}
      title={
        <span>
          <ExclamationCircleOutlined
            style={{ color: riskColors[riskLevel], marginRight: 8 }}
          />
          高风险操作确认
        </span>
      }
      okText={confirmText}
      cancelText="取消"
      okButtonProps={{ danger: true, type: 'primary' }}
      confirmLoading={confirmLoading}
      onOk={handleConfirm}
      onCancel={handleCancel}
      destroyOnClose
      width={520}
    >
      <Alert
        message={
          <span>
            即将执行 <Text strong>{actionName}</Text>
            {riskLevel !== 'low' && (
              <Tag color={riskColors[riskLevel]} style={{ marginLeft: 8 }}>
                {riskLabels[riskLevel]}
              </Tag>
            )}
          </span>
        }
        type={riskLevel === 'high' ? 'error' : riskLevel === 'medium' ? 'warning' : 'info'}
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Descriptions column={1} size="small" bordered>
        {detail?.targetLabel && (
          <Descriptions.Item label="目标对象">
            <Text code>{detail.targetLabel}</Text>
          </Descriptions.Item>
        )}
        {detail?.beforeValue && (
          <Descriptions.Item label="当前值">{detail.beforeValue}</Descriptions.Item>
        )}
        {detail?.afterValue && (
          <Descriptions.Item label="变更后">{detail.afterValue}</Descriptions.Item>
        )}

        <Descriptions.Item label="环境模式">
          <Tag color={modeColors[environmentMode]}>{modeLabels[environmentMode]}</Tag>
        </Descriptions.Item>

        <Descriptions.Item label="审批要求">
          <Tag color={requiresApproval ? 'orange' : 'green'}>
            {requiresApproval ? '需要审批' : '不需要审批'}
          </Tag>
        </Descriptions.Item>

        {expectedConsequence && (
          <Descriptions.Item label="预期后果">{expectedConsequence}</Descriptions.Item>
        )}
      </Descriptions>

      {showReason && (
        <>
          <Divider style={{ margin: '12px 0' }} />
          <TextArea
            placeholder={reasonPlaceholder}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={2}
          />
        </>
      )}

      {(auditDestination || rollbackNote) && (
        <div style={{ marginTop: 12, fontSize: 13, color: '#888' }}>
          {auditDestination && (
            <div style={{ marginBottom: 4 }}>
              <AuditOutlined style={{ marginRight: 6 }} />
              <Text type="secondary">{auditDestination}</Text>
            </div>
          )}
          {rollbackNote && (
            <div>
              <RollbackOutlined style={{ marginRight: 6 }} />
              <Text type="secondary">{rollbackNote}</Text>
            </div>
          )}
        </div>
      )}
    </Modal>
  );
}
