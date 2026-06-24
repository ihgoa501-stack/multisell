'use client';

import { useState } from 'react';
import {
  Card, Table, Tag, Typography, Button, Modal, Descriptions, Space, Switch,
} from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const { Title } = Typography;

interface PolicyRule {
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

const OUTCOME_COLORS: Record<string, string> = {
  auto_approve: 'green',
  escalate: 'orange',
  block: 'red',
};

export default function PolicySettingsPage() {
  const [detailModal, setDetailModal] = useState<PolicyRule | null>(null);

  const { data: rulesData, isLoading } = useQuery<Result<PolicyRule[]>>({
    queryKey: ['policy-rules'],
    queryFn: () => apiClient.get<PolicyRule[]>('/policy/rules'),
  });

  const rules = rulesData?.data || [];

  const columns = [
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a: PolicyRule, b: PolicyRule) => b.priority - a.priority,
    },
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: PolicyRule) => (
        <a onClick={() => setDetailModal(record)}>{name}</a>
      ),
    },
    {
      title: '风险等级',
      dataIndex: 'risk_level',
      key: 'risk_level',
      width: 100,
      render: (level: string) => level ? <Tag>{level}</Tag> : <Tag>*</Tag>,
    },
    {
      title: '动作类型',
      dataIndex: 'action_type',
      key: 'action_type',
      width: 120,
      render: (t: string) => t === '*' ? <Tag>全部</Tag> : <Tag>{t}</Tag>,
    },
    {
      title: 'Agent',
      dataIndex: 'agent_id',
      key: 'agent_id',
      width: 80,
      render: (id: string | null) => id === '*' || !id ? <Tag>全部</Tag> : <Tag>{id}</Tag>,
    },
    {
      title: '阈值',
      key: 'thresholds',
      width: 200,
      render: (_: unknown, r: PolicyRule) => {
        const parts: string[] = [];
        if (r.max_amount) parts.push('¥' + r.max_amount);
        if (r.max_quantity) parts.push(r.max_quantity + '件');
        if (r.min_confidence) parts.push('置信>' + (r.min_confidence * 100).toFixed(0) + '%');
        return parts.length > 0 ? parts.join(', ') : '-';
      },
    },
    {
      title: '结果',
      dataIndex: 'outcome',
      key: 'outcome',
      width: 120,
      render: (o: string) => <Tag color={OUTCOME_COLORS[o]}>{o === 'auto_approve' ? '自动批准' : o === 'escalate' ? '人工审批' : '阻止'}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (e: boolean) => e ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>审批策略</Title>
      <p style={{ color: '#666', marginBottom: 16 }}>
        审批策略决定 AI Agent 的动作是否需要人工审批、自动批准或阻止执行。
        策略按优先级顺序评估，最严格的结果生效。
      </p>

      <Card>
        <Table
          dataSource={rules}
          columns={columns}
          rowKey="id"
          loading={isLoading}
          pagination={false}
          size="small"
        />
      </Card>

      <Modal
        title={detailModal?.name || '规则详情'}
        open={!!detailModal}
        onCancel={() => setDetailModal(null)}
        footer={<Button onClick={() => setDetailModal(null)}>关闭</Button>}
        width={560}
      >
        {detailModal && (
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="名称" span={2}>{detailModal.name}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>{detailModal.description || '-'}</Descriptions.Item>
            <Descriptions.Item label="风险等级">{detailModal.risk_level || '全部'}</Descriptions.Item>
            <Descriptions.Item label="动作类型">{detailModal.action_type || '全部'}</Descriptions.Item>
            <Descriptions.Item label="Agent">{detailModal.agent_id || '全部'}</Descriptions.Item>
            <Descriptions.Item label="业务对象">{detailModal.business_object_type || '全部'}</Descriptions.Item>
            <Descriptions.Item label="最大金额">{detailModal.max_amount ? '¥' + detailModal.max_amount : '无限制'}</Descriptions.Item>
            <Descriptions.Item label="最大数量">{detailModal.max_quantity ? detailModal.max_quantity + '件' : '无限制'}</Descriptions.Item>
            <Descriptions.Item label="最低置信度">{detailModal.min_confidence ? (detailModal.min_confidence * 100).toFixed(0) + '%' : '无限制'}</Descriptions.Item>
            <Descriptions.Item label="结果">
              <Tag color={OUTCOME_COLORS[detailModal.outcome]}>
                {detailModal.outcome === 'auto_approve' ? '自动批准' : detailModal.outcome === 'escalate' ? '人工审批' : '阻止'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="优先级">{detailModal.priority}</Descriptions.Item>
            <Descriptions.Item label="是否启用">{detailModal.enabled ? '是' : '否'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}
