'use client';

import { useMemo, useState } from 'react';
import { Button, Card, Col, Row, Space, Table, Tag, Typography, message } from 'antd';
import { CheckOutlined, CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const { Title, Text } = Typography;

interface NudgeItem {
  id: number;
  agent_id: string;
  current_level: string;
  target_level: string;
  trust_score: number;
  status: string; // pending | accepted | dismissed
  message: string;
  created_at: string;
  decided_at?: string;
}

const LEVEL_COLORS: Record<string, string> = {
  advisory: 'default',
  guided: 'blue',
  supervised: 'orange',
  autonomous: 'green',
};

const STATUS_COLORS: Record<string, string> = {
  pending: 'blue',
  accepted: 'green',
  dismissed: 'default',
};

const FILTER_TABS = [
  { key: '', label: '全部' },
  { key: 'pending', label: '待处理' },
  { key: 'accepted', label: '已接受' },
  { key: 'dismissed', label: '已忽略' },
];

export default function EvolutionPage() {
  const qc = useQueryClient();
  const [filterStatus, setFilterStatus] = useState<string>('pending');

  // ---------- List query ----------
  const { data, isLoading } = useQuery<Result<NudgeItem[]>>({
    queryKey: ['evolution-nudges', filterStatus],
    queryFn: () => {
      const params: Record<string, string> = {};
      if (filterStatus) params.status = filterStatus;
      return apiClient.get<NudgeItem[]>('/evolution/nudges', params);
    },
  });

  const nudges = data?.data || [];

  // ---------- Stats query (no filter, for summary cards) ----------
  const { data: statsData } = useQuery<Result<NudgeItem[]>>({
    queryKey: ['evolution-nudges-stats'],
    queryFn: () => apiClient.get<NudgeItem[]>('/evolution/nudges', {}),
  });

  const stats = useMemo(() => {
    const all = statsData?.data || [];
    return {
      total: all.length,
      pending: all.filter((n) => n.status === 'pending').length,
      accepted: all.filter((n) => n.status === 'accepted').length,
    };
  }, [statsData]);

  // ---------- Mutations ----------
  const evaluateMut = useMutation({
    mutationFn: () => apiClient.post('/v1/evolution/nudges/evaluate'),
    onSuccess: () => {
      message.success('评估完成');
      qc.invalidateQueries({ queryKey: ['evolution-nudges'] });
    },
    onError: () => message.error('评估失败'),
  });

  const acceptMut = useMutation({
    mutationFn: (id: number) => apiClient.post(`/evolution/nudges/${id}/accept`),
    onSuccess: () => {
      message.success('已接受升级');
      qc.invalidateQueries({ queryKey: ['evolution-nudges'] });
    },
    onError: () => message.error('操作失败'),
  });

  const dismissMut = useMutation({
    mutationFn: (id: number) => apiClient.post(`/evolution/nudges/${id}/dismiss`),
    onSuccess: () => {
      message.success('已忽略');
      qc.invalidateQueries({ queryKey: ['evolution-nudges'] });
    },
    onError: () => message.error('操作失败'),
  });

  // ---------- Table columns ----------
  const columns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    {
      title: 'Agent',
      dataIndex: 'agent_id',
      width: 80,
      render: (id: string) => <Text strong>{id}</Text>,
    },
    {
      title: '当前级别',
      dataIndex: 'current_level',
      width: 100,
      render: (l: string) => <Tag color={LEVEL_COLORS[l] || 'default'}>{l}</Tag>,
    },
    {
      title: '目标级别',
      dataIndex: 'target_level',
      width: 100,
      render: (l: string) => <Tag color={LEVEL_COLORS[l] || 'default'}>{l}</Tag>,
    },
    {
      title: '信任分',
      dataIndex: 'trust_score',
      width: 80,
      render: (v: number) => (v * 100).toFixed(0) + '%',
    },
    { title: '消息', dataIndex: 'message', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (s: string) => <Tag color={STATUS_COLORS[s] || 'default'}>{s}</Tag>,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, r: NudgeItem) => {
        if (r.status !== 'pending') return <Tag color="default">已处理</Tag>;
        return (
          <Space size="small">
            <Button
              type="primary"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => acceptMut.mutate(r.id)}
              loading={acceptMut.isPending}
            >
              接受
            </Button>
            <Button
              size="small"
              icon={<CloseOutlined />}
              onClick={() => dismissMut.mutate(r.id)}
              loading={dismissMut.isPending}
            >
              忽略
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* Header */}
      <Space
        style={{ marginBottom: 16, justifyContent: 'space-between', width: '100%' }}
        align="start"
      >
        <div>
          <Title level={3} style={{ margin: 0 }}>
            Agent 进化
          </Title>
          <Text type="secondary">
            系统根据信任分、采纳率等多维指标评估 Agent 升级机会，推送 Nudge 升级建议
          </Text>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => evaluateMut.mutate()}
            loading={evaluateMut.isPending}
          >
            检测升级机会
          </Button>
        </Space>
      </Space>

      {/* Summary cards */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Text type="secondary">总建议数</Text>
            <div style={{ fontSize: 28, fontWeight: 600 }}>{stats.total}</div>
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Text type="secondary">待处理</Text>
            <div style={{ fontSize: 28, fontWeight: 600, color: '#1677ff' }}>{stats.pending}</div>
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Text type="secondary">已接受</Text>
            <div style={{ fontSize: 28, fontWeight: 600, color: '#52c41a' }}>{stats.accepted}</div>
          </Card>
        </Col>
      </Row>

      {/* Filter tabs */}
      <Card style={{ marginBottom: 16, padding: '4px 16px' }} styles={{ body: { padding: '8px 0' } }}>
        <Space>
          {FILTER_TABS.map((tab) => (
            <Button
              key={tab.key}
              type={filterStatus === tab.key ? 'primary' : 'default'}
              size="small"
              onClick={() => setFilterStatus(tab.key)}
            >
              {tab.label}
            </Button>
          ))}
        </Space>
      </Card>

      {/* Nudge table */}
      <Table
        dataSource={nudges}
        columns={columns}
        rowKey="id"
        loading={isLoading}
        pagination={{
          pageSize: 20,
          showTotal: (t: number) => `共 ${t} 条`,
        }}
        size="middle"
      />
    </div>
  );
}
