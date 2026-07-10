'use client';

import { useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Input,
  Modal,
  Space,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlayCircleOutlined,
  ExclamationCircleOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';

const { TextArea } = Input;
const { Text } = Typography;

export interface ConfirmAction {
  id: number;
  title: string;
  description?: string;
  agent_id: string;
  action_type: string;
  risk_level: string;
  status: string;
  confidence?: number | null;
  proposed_by: string;
  trace_id?: string;
  execution_mode?: string;
  rejection_reason?: string | null;
  payload: Record<string, unknown>;
}

interface Props {
  action: ConfirmAction | null;
  open: boolean;
  mode: 'approve' | 'reject' | 'execute' | null;
  loading: boolean;
  onClose: () => void;
  onConfirm: (action: ConfirmAction, reason: string) => void;
}

const RISK_CONFIG: Record<string, { color: string; label: string; description: string }> = {
  low: { color: 'green', label: '低风险', description: '无财务影响' },
  medium: { color: 'orange', label: '中风险', description: '轻微影响' },
  high: { color: 'red', label: '高风险', description: '财务/业务影响，需审批' },
  critical: { color: 'purple', label: '严重', description: '重大影响，需多重确认' },
};

const MODE_TITLES: Record<string, string> = {
  approve: '确认批准',
  reject: '确认拒绝',
  execute: '确认执行',
};

export default function ActionConfirmModal({ action, open, mode, loading, onClose, onConfirm }: Props) {
  const [reason, setReason] = useState('');
  const [titleConfirm, setTitleConfirm] = useState('');

  const isHighRisk = action?.risk_level === 'high' || action?.risk_level === 'critical';
  const title = mode ? MODE_TITLES[mode] : '操作确认';
  const needsReason = mode === 'reject';
  const needsTitleConfirm = isHighRisk && mode === 'execute';

  const handleConfirm = () => {
    if (!action) return;
    onConfirm(action, reason);
  };

  const risk = action ? (RISK_CONFIG[action.risk_level] || RISK_CONFIG.low) : RISK_CONFIG.low;

  const footerEls: React.ReactNode[] = [];
  if (!mode) {
    footerEls.push(<Button key="close" onClick={onClose}>关闭</Button>);
  } else if (mode === 'reject') {
    footerEls.push(<Button key="cancel" onClick={onClose}>取消</Button>);
    footerEls.push(
      <Button
        key="reject"
        danger
        type="primary"
        icon={<CloseCircleOutlined />}
        onClick={handleConfirm}
        loading={loading}
        disabled={needsReason && !reason.trim()}
      >
        拒绝
    </Button>
    );
  } else {
    footerEls.push(<Button key="cancel" onClick={onClose}>取消</Button>);
    const confirmLabel = mode === 'approve' ? '确认批准' : '确认执行';
    const confirmIcon = mode === 'approve' ? <CheckCircleOutlined /> : <PlayCircleOutlined />;
    footerEls.push(
      <Button
        key="confirm"
        type="primary"
        icon={confirmIcon}
        onClick={handleConfirm}
        loading={loading}
        disabled={needsTitleConfirm && titleConfirm !== action?.title}
      >
        {confirmLabel}
      </Button>
    );
  }

  return (
    <Modal
      title={
        <Space>
          {isHighRisk && <ExclamationCircleOutlined style={{ color: risk.color }} />}
          {title}
        </Space>
      }
      open={open}
      onCancel={onClose}
      afterOpenChange={(visible) => {
        if (!visible) {
          setReason('');
          setTitleConfirm('');
        }
      }}
      footer={footerEls}
      width={640}
    >
      {action && (
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Alert
            message={
              <Space>
                {isHighRisk && <SafetyCertificateOutlined />}
                <Text strong style={{ color: risk.color }}>{risk.label}</Text>
                <Text>- {risk.description}</Text>
              </Space>
            }
            type={isHighRisk ? 'warning' : 'info'}
            showIcon
          />

          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="ID">{action.id}</Descriptions.Item>
            <Descriptions.Item label="标题">{action.title}</Descriptions.Item>
            <Descriptions.Item label="操作类型">{action.action_type}</Descriptions.Item>
            <Descriptions.Item label="模式">{action.execution_mode || '-'}</Descriptions.Item>
            <Descriptions.Item label="Agent">{action.agent_id}</Descriptions.Item>
            <Descriptions.Item label="置信度">
              {action.confidence != null ? (action.confidence * 100).toFixed(0) + '%' : '-'}
            </Descriptions.Item>
            {action.trace_id && (
              <Descriptions.Item label="审计追踪ID" span={2}>{action.trace_id}</Descriptions.Item>
            )}
            {action.description && (
              <Descriptions.Item label="描述" span={2}>{action.description}</Descriptions.Item>
            )}
            <Descriptions.Item label="提案人" span={2}>{action.proposed_by}</Descriptions.Item>
          </Descriptions>

          <div>
            <Text strong style={{ marginBottom: 8, display: 'block' }}>载荷 (Payload)</Text>
            <pre
              style={{
                background: '#f5f5f5',
                padding: 12,
                borderRadius: 4,
                maxHeight: 300,
                overflow: 'auto',
                fontSize: 12,
                margin: 0,
              }}
            >
              {JSON.stringify(action.payload, null, 2)}
            </pre>
          </div>

          {mode === 'execute' && (
            <Alert
              message="执行后如需回滚，请手动操作。此操作不可撤回。"
              type="warning"
              showIcon
            />
          )}

          {(needsReason || mode === 'approve' || mode === 'execute') && (
            <div>
              <Text strong style={{ marginBottom: 4, display: 'block' }}>
                原因{needsReason ? '（必填）' : '（可选）'}
              </Text>
              <TextArea
                rows={3}
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder={needsReason ? '请输入拒绝原因...' : '请输入备注（可选）...'}
              />
            </div>
          )}

          {needsTitleConfirm && (
            <div>
              <Text strong style={{ marginBottom: 4, display: 'block' }}>
                请输入操作标题以确认严重操作：
              </Text>
              <Input
                value={titleConfirm}
                onChange={(e) => setTitleConfirm(e.target.value)}
                placeholder={action.title}
              />
            </div>
          )}
        </Space>
      )}
    </Modal>
  );
}
