'use client';

import { useState } from 'react';
import {
  Button,
  Card,
  Col,
  Input,
  message,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlayCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import ActionConfirmModal, { ConfirmAction } from '@/components/actions/ActionConfirmModal';
import apiClient from '@/lib/api-client';
import type { PageResult } from '@/types/api';

const { Text, Title } = Typography;

interface UnifiedAction {
  id: number;
  agent_id: string;
  squad_id: string;
  action_type: string;
  title: string;
  description: string;
  risk_level: string;
  status: string;
  confidence: number | null;
  proposed_by: string;
  proposed_at: string;
  rejection_reason: string | null;
  payload: Record<string, unknown>;
}

const RISK_COLORS: Record<string, string> = {
  low: 'green',
  medium: 'orange',
  high: 'red',
  critical: 'purple',
};

const STATUS_COLORS: Record<string, string> = {
  suggested: 'blue',
  pending: 'orange',
  approved: 'cyan',
  rejected: 'red',
  executing: 'purple',
  executed: 'green',
  failed: 'red',
  reviewed: 'default',
};

export default function ActionsPage() {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilter] = useState<string>('pending');
  const [riskFilter, setRiskFilter] = useState<string>('');
  const [agentFilter, setAgentFilter] = useState<string>('');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [actionMode, setActionMode] = useState<'approve' | 'reject' | 'execute' | null>(null);
  const [selectedAction, setSelectedAction] = useState<UnifiedAction | null>(null);

  const { data: actionsData, isLoading } = useQuery<PageResult<UnifiedAction>>({
    queryKey: ['actions', statusFilter, riskFilter, agentFilter, search, page],
    queryFn: () => {
      const params: Record<string, string> = { page: String(page), size: '20' };
      if (statusFilter) params.status = statusFilter;
      if (riskFilter) params.risk_level = riskFilter;
      if (agentFilter) params.agent_id = agentFilter;
      if (search) params.search = search;
      return apiClient.getPage<UnifiedAction>('/v1/ai/actions', params);
    },
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      apiClient.post('/v1/ai/actions/' + id + '/approve', { operator: 'user', reason }),
    onSuccess: () => {
      message.success('已批准');
      setModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('批准失败'),
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      apiClient.post('/v1/ai/actions/' + id + '/reject', { operator: 'user', reason }),
    onSuccess: () => {
      message.success('已拒绝');
      setModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('拒绝失败'),
  });

  const executeMutation = useMutation({
    mutationFn: ({ id }: { id: number }) =>
      apiClient.post('/v1/ai/actions/' + id + '/execute', { operator: 'user' }),
    onSuccess: () => {
      message.success('已执行');
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('执行失败'),
  });

  const openModal = (action: UnifiedAction, mode: 'approve' | 'reject' | 'execute' | null) => {
    setSelectedAction(action);
    setActionMode(mode);
    setModalOpen(true);
  };

  const handleConfirm = (action: ConfirmAction, reason: string) => {
    if (actionMode === 'approve') {
      approveMutation.mutate({ id: action.id, reason });
    } else if (actionMode === 'reject') {
      rejectMutation.mutate({ id: action.id, reason });
    } else if (actionMode === 'execute') {
      executeMutation.mutate({ id: action.id });
    }
  };

  const columns = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      render: (t: string, r: UnifiedAction) => (
        <Text strong style={{ cursor: 'pointer' }} onClick={() => openModal(r, null)}>{t}</Text>
      ),
    },
    {
      title: 'Agent',
      dataIndex: 'agent_id',
      key: 'agent_id',
      width: 80,
      render: (id: string) => <Tag>{id}</Tag>,
    },
    {
      title: '类型',
      dataIndex: 'action_type',
      key: 'action_type',
      width: 120,
    },
    {
      title: '风险等级',
      dataIndex: 'risk_level',
      key: 'risk_level',
      width: 100,
      render: (level: string) => (
        <Tag color={RISK_COLORS[level] || 'default'}>{level}</Tag>
      ),
    },
    {
      title: '置信度',
      dataIndex: 'confidence',
      key: 'confidence',
      width: 90,
      render: (v: number | null) => v ? (v * 100).toFixed(0) + '%' : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => (
        <Tag color={STATUS_COLORS[s] || 'default'}>{s}</Tag>
      ),
    },
    {
      title: '建议时间',
      dataIndex: 'proposed_at',
      key: 'proposed_at',
      width: 160,
      render: (t: string) => new Date(t).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: unknown, record: UnifiedAction) => {
        if (record.status === 'approved') {
          return (
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => openModal(record, 'execute')}
            >
              执行
            </Button>
          );
        }
        if (record.status === 'suggested' || record.status === 'pending') {
          return (
            <Space>
              <Button
                type="primary"
                size="small"
                icon={<CheckCircleOutlined />}
                onClick={() => openModal(record, 'approve')}
              >
                批准
              </Button>
              <Button
                danger
                size="small"
                icon={<CloseCircleOutlined />}
                onClick={() => openModal(record, 'reject')}
              >
                拒绝
              </Button>
            </Space>
          );
        }
        return <Tag>{record.status}</Tag>;
      },
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>Action Center</Title>

      <Card style={{ marginBottom: 16 }}>
        <Row gutter={16} align="middle">
          <Col>
            <Select
              value={statusFilter}
              onChange={(v) => { setStatusFilter(v); setPage(1); }}
              style={{ width: 140 }}
              options={[
                { value: '', label: '全部状态' },
                { value: 'suggested', label: '待审批' },
                { value: 'pending', label: '审批中' },
                { value: 'approved', label: '已批准' },
                { value: 'rejected', label: '已拒绝' },
                { value: 'executed', label: '已执行' },
              ]}
            />
          </Col>
          <Col>
            <Select
              value={riskFilter}
              onChange={(v) => { setRiskFilter(v); setPage(1); }}
              style={{ width: 130 }}
              allowClear
              placeholder="风险等级"
              options={[
                { value: 'low', label: '低风险' },
                { value: 'medium', label: '中风险' },
                { value: 'high', label: '高风险' },
                { value: 'critical', label: '严重' },
              ]}
            />
          </Col>
          <Col>
            <Select
              value={agentFilter}
              onChange={(v) => { setAgentFilter(v); setPage(1); }}
              style={{ width: 120 }}
              allowClear
              placeholder="Agent"
              options={[
                { value: 'A1', label: 'A1 选品' },
                { value: 'A2', label: 'A2 Listing' },
                { value: 'A3', label: 'A3 广告' },
                { value: 'A4', label: 'A4 客服' },
                { value: 'A5', label: 'A5 库存' },
                { value: 'A6', label: 'A6 利润' },
                { value: 'A7', label: 'A7 合规' },
                { value: 'G1', label: 'G1 驾驶舱' },
                { value: 'G2', label: 'G2 仓储' },
                { value: 'G3', label: 'G3 折扣' },
              ]}
            />
          </Col>
          <Col flex="auto">
            <Input
              prefix={<SearchOutlined />}
              placeholder="搜索标题..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              allowClear
            />
          </Col>
          <Col>
            <Button onClick={() => queryClient.invalidateQueries({ queryKey: ['actions'] })}>
              刷新
            </Button>
          </Col>
        </Row>
      </Card>

      <Table
        dataSource={actionsData?.data || []}
        columns={columns}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize: 20,
          total: actionsData?.total || 0,
          onChange: setPage,
          showTotal: (t: number) => "共 " + t + " 条",
        }}
        size="middle"
      />

      <ActionConfirmModal
        action={selectedAction}
        open={modalOpen}
        mode={actionMode}
        loading={approveMutation.isPending || rejectMutation.isPending || executeMutation.isPending}
        onClose={() => setModalOpen(false)}
        onConfirm={handleConfirm}
      />
    </div>
  );
}
