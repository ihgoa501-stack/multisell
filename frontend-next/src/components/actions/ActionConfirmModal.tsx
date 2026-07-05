'use client';

import { useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Input,
  Modal,
  Space,
  Tag,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  PlayCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';

const { Text, Paragraph, Title } = Typography;

const RISK_LABELS: Record<string, { color: string; label: string; desc: string }> = {
  low: { color: 'green', label: '低风险', desc: '不会产生财务影响，系统可自动执行' },
  medium: { color: 'orange', label: '中风险', desc: '可能有轻微影响，建议人工确认' },
  high: { color: 'red', label: '高风险', desc: '会产生财务或业务影响，必须人工审批' },
  critical: { color: 'purple', label: '严重', desc: '会产生重大影响，需多重确认' },
};

interface ConfirmAction {
  id: number;
  title: string;
  description?: string;
  agent_id: string;
  squad_id?: string;
  action_type: string;
  risk_level: string;
  status: string;
  confidence?: number | null;
  proposed_by: string;
  proposed_at?: string;
  trace_id?: string;
  execution_mode?: string;
  rejection_reason?: string | null;
  payload: Record<string, unknown>;
}

interface ActionConfirmModalProps {
  action: ConfirmAction | null;
  open: boolean;
  mode: 'approve' | 'reject' | 'execute' | null;
  loading?: boolean;
  onClose: () => void;
  onConfirm: (action: ConfirmAction, reason: string) => void;
}

export default function ActionConfirmModal({
  action,
  open,
  mode,
  loading,
  onClose,
  onConfirm,
}: ActionConfirmModalProps) {
  const [reason, setReason] = useState('');
  const [confirmText, setConfirmText] = useState('');

  if (!action) return null;

  const normalizedRisk = action.risk_level?.toLowerCase() ?? '';
  const HIGH_RISK_LEVELS = new Set(['high', 'critical']);
  const risk = RISK_LABELS[normalizedRisk] || RISK_LABELS.medium;
  const isHighRisk = HIGH_RISK_LEVELS.has(normalizedRisk);
  const isExecuteMode = mode === 'execute';
  const isApproveMode = mode === 'approve';

  const title = isExecuteMode
    ? '确认执行操作'
    : isApproveMode
    ? '确认批准操作'
    : mode === 'reject'
    ? '确认拒绝操作'
    : '操作确认';

  const canConfirm = isExecuteMode
    ? (!isHighRisk || confirmText === action.title)
    : true;

  const handleConfirm = () => {
    if (!action) return;
    if (isExecuteMode && isHighRisk && confirmText !== action.title) return;
    onConfirm(action, reason);
  };

  return (
    <Modal
      title={
        <Space>
          {isHighRisk && <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />}
          {title}
        </Space>
      }
      open={open}
      onCancel={onClose}
      width={640}
      footer={
        mode === null
          ? [
              <Button key="close" type="primary" onClick={onClose}>
                关闭
              </Button>,
            ]
          : mode === 'reject'
          ? [
              <Button key="cancel" onClick={onClose}>
                取消
              </Button>,
              <Button
                key="reject"
                danger
                icon={<CloseCircleOutlined />}
                onClick={handleConfirm}
                loading={loading}
              >
                确认拒绝
              </Button>,
            ]
          : [
              <Button key="cancel" onClick={onClose}>
                取消
              </Button>,
              <Button
                key="confirm"
                type="primary"
                icon={isExecuteMode ? <PlayCircleOutlined /> : <CheckCircleOutlined />}
                onClick={handleConfirm}
                loading={loading}
                disabled={!canConfirm}
                danger={isExecuteMode && isHighRisk}
              >
                {isExecuteMode ? '确认执行' : isApproveMode ? '确认批准' : '确认'}
              </Button>,
            ]
      }
    >
      {/* Risk level banner */}
      <Alert
        type={isHighRisk ? 'warning' : 'info'}
        showIcon
        icon={<SafetyCertificateOutlined />}
        message={
          <Space>
            <Tag color={risk.color} style={{ marginRight: 4 }}>
              {risk.label}
            </Tag>
            {risk.desc}
          </Space>
        }
        style={{ marginBottom: 16 }}
      />

      {/* Action details */}
      <Descriptions column={2} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label="Action ID" span={2}>
          <Text code>{action.id}</Text>
        </Descriptions.Item>
        <Descriptions.Item label="标题" span={2}>
          <Text strong>{action.title}</Text>
        </Descriptions.Item>
        <Descriptions.Item label="操作类型">
          <Tag>{action.action_type}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="模式">
          <Tag color={action.execution_mode === 'dry_run' ? 'orange' : 'blue'}>
            {action.execution_mode || 'production'}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Agent">
          <Tag>{action.agent_id}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="置信度">
          {action.confidence
            ? (action.confidence * 100).toFixed(0) + '%'
            : '-'}
        </Descriptions.Item>
        {action.trace_id && (
          <Descriptions.Item label="审计追踪 ID" span={2}>
            <Text code copyable style={{ fontSize: 12 }}>
              {action.trace_id}
            </Text>
          </Descriptions.Item>
        )}
        {action.description && (
          <Descriptions.Item label="描述" span={2}>
            <Paragraph style={{ margin: 0 }}>{action.description}</Paragraph>
          </Descriptions.Item>
        )}
        <Descriptions.Item label="提案人" span={2}>
          {action.proposed_by || '-'}
        </Descriptions.Item>
      </Descriptions>

      {/* Payload summary */}
      {action.payload && Object.keys(action.payload).length > 0 && (
        <>
          <Title level={5}>
            <ExclamationCircleOutlined style={{ marginRight: 8 }} />
            操作内容
          </Title>
          <pre
            style={{
              background: '#f5f5f5',
              padding: 12,
              borderRadius: 4,
              maxHeight: 200,
              overflow: 'auto',
              fontSize: 12,
              marginBottom: 16,
            }}
          >
            {JSON.stringify(action.payload, null, 2)}
          </pre>
        </>
      )}

      {/* Reason input */}
      <Input.TextArea
        placeholder={
          mode === 'reject'
            ? '请输入拒绝原因（必填）'
            : isExecuteMode
            ? '备注（非必填）'
            : '审批备注（非必填）'
        }
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        rows={2}
        style={{ marginBottom: 12 }}
      />

      {/* High-risk confirmation */}
      {isExecuteMode && isHighRisk && (
        <Alert
          type="error"
          showIcon
          message="高风险操作确认"
          description={
            <div>
              <Text>
                请输入操作标题 <Text code>{action.title}</Text> 以确认：
              </Text>
              <Input
                placeholder="请粘贴标题以确认"
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                style={{ marginTop: 8 }}
              />
            </div>
          }
        />
      )}

      {/* Rollback notice */}
      {isExecuteMode && (
        <Alert
          type="info"
          showIcon
          message="可回滚操作"
          description="此操作可以回滚。如需撤销，请在 operation_log 中查找此 Action ID，或联系系统管理员。"
          style={{ marginTop: 8 }}
        />
      )}
    </Modal>
  );
}
