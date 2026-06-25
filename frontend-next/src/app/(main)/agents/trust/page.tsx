'use client';

import { useState } from 'react';
import { Button, Table, Tag, Typography, Space, Progress, Modal, Descriptions, message } from 'antd';
import { ReloadOutlined, ArrowUpOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const { Title, Text } = Typography;

interface TrustScoreItem {
  agent_id: string; agent_name: string; squad_id: string; description: string;
  capabilities: string[]; current_level: string; target_level: string;
  trust_score: number; adoption_rate: number; exec_success: number;
  avg_confidence: number; total_actions: number; next_threshold: number;
}

const LEVEL_COLORS: Record<string, string> = { advisory: 'default', guided: 'blue', supervised: 'orange', autonomous: 'green' };
const SQUAD_LABELS: Record<string, string> = { growth: '增长', fulfillment: '履约', risk: '风控', governance: '治理', settle: '结算' };

export default function TrustScoresPage() {
  const qc = useQueryClient();
  const { data, isLoading } = useQuery<Result<TrustScoreItem[]>>({
    queryKey: ['trust-summary'],
    queryFn: () => apiClient.get<TrustScoreItem[]>('/v1/trust-scores/summary'),
  });
  const [detail, setDetail] = useState<TrustScoreItem | null>(null);

  const recalcMut = useMutation({
    mutationFn: () => apiClient.post('/v1/trust-scores/recalculate'),
    onSuccess: () => { message.success('已重算'); qc.invalidateQueries({ queryKey: ['trust-summary'] }); },
  });

  const upgradeMut = useMutation({
    mutationFn: () => apiClient.post('/v1/trust-scores/auto-upgrade'),
    onSuccess: () => {
      message.success('升级请求已提交');
      qc.invalidateQueries({ queryKey: ['trust-summary'] });
    },
  });

  const scores = data?.data || [];

  const cols = [
    { title: 'Agent', dataIndex: 'agent_id', width: 60, render: (id: string) => <Text strong>{id}</Text> },
    { title: '名称', dataIndex: 'agent_name', width: 100 },
    { title: '分队', dataIndex: 'squad_id', width: 70, render: (s: string) => <Tag>{SQUAD_LABELS[s] || s}</Tag> },
    { title: '信任分', dataIndex: 'trust_score', width: 140, sorter: (a: TrustScoreItem, b: TrustScoreItem) => a.trust_score - b.trust_score,
      render: (v: number) => <Progress percent={Math.round(v*100)} size="small" status={v<0.3?'exception':v<0.55?'active':'success'} format={(p) => (p||0).toFixed(0)+'%'} /> },
    { title: '采纳率', dataIndex: 'adoption_rate', width: 70, render: (v: number) => (v*100).toFixed(0)+'%' },
    { title: '执行成功率', dataIndex: 'exec_success', width: 70, render: (v: number) => (v*100).toFixed(0)+'%' },
    { title: '置信度', dataIndex: 'avg_confidence', width: 70, render: (v: number) => (v*100).toFixed(0)+'%' },
    { title: '当前等级', dataIndex: 'current_level', width: 90, render: (l: string) => <Tag color={LEVEL_COLORS[l]}>{l}</Tag> },
    { title: '目标等级', dataIndex: 'target_level', width: 90, render: (l: string) => l ? <Tag color={LEVEL_COLORS[l]}>{l}</Tag> : '-' },
    { title: 'Action数', dataIndex: 'total_actions', width: 60 },
    { title: '', key: 'action', width: 50, render: (_: unknown, r: TrustScoreItem) => <Button size="small" onClick={() => setDetail(r)}>详情</Button> },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space style={{ marginBottom: 16, justifyContent: 'space-between', width: '100%' }}>
        <div>
          <Title level={3} style={{ margin: 0 }}>Agent 信任分与自主度</Title>
          <Text type="secondary">基于采纳率(40%) + 执行成功率(30%) + 置信度(30%) 的复合评分</Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => recalcMut.mutate()} loading={recalcMut.isPending}>重算</Button>
          <Button type="primary" icon={<ArrowUpOutlined />} onClick={() => upgradeMut.mutate()} loading={upgradeMut.isPending}>自动升级</Button>
        </Space>
      </Space>

      <Table dataSource={scores} columns={cols} rowKey="agent_id" loading={isLoading} pagination={false} size="small" />

      <Modal title={detail?.agent_name+' ('+detail?.agent_id+')'} open={!!detail} onCancel={() => setDetail(null)}
        footer={<Button onClick={() => setDetail(null)}>关闭</Button>} width={520}>
        {detail && (
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label="描述" span={2}>{detail.description}</Descriptions.Item>
            <Descriptions.Item label="能力" span={2}>{detail.capabilities?.join(', ') || '-'}</Descriptions.Item>
            <Descriptions.Item label="当前等级"><Tag color={LEVEL_COLORS[detail.current_level]}>{detail.current_level}</Tag></Descriptions.Item>
            <Descriptions.Item label="目标等级">{detail.target_level ? <Tag color={LEVEL_COLORS[detail.target_level]}>{detail.target_level}</Tag> : '-'}</Descriptions.Item>
            <Descriptions.Item label="信任分"><Progress percent={Math.round(detail.trust_score*100)} size="small" /></Descriptions.Item>
            <Descriptions.Item label="采纳率">{(detail.adoption_rate*100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label="执行成功率">{(detail.exec_success*100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label="平均置信度">{(detail.avg_confidence*100).toFixed(1)}%</Descriptions.Item>
            <Descriptions.Item label="总 Action 数">{detail.total_actions}</Descriptions.Item>
            <Descriptions.Item label="下一门槛">{detail.next_threshold > 0 ? (detail.next_threshold*100).toFixed(0)+'%' : '已满级'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}
