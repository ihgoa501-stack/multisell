'use client';

import { useState } from 'react';
import {
  Button,
  Card,
  Col,
  Input,
  message,
  Modal,
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
  const [selectedAction, setSelectedAction] = useState<UnifiedAction | null>(null);
  const [decisionReason, setDecisionReason] = useState('');

  const { data: actionsData, isLoading } = useQuery<PageResult<UnifiedAction>>({
    queryKey: ['actions', statusFilter, riskFilter, agentFilter, search, page],
    queryFn: () => {
      const params: Record<string, string> = { page: String(page), size: '20' };
      if (statusFilter) params.status = statusFilter;
      if (riskFilter) params.risk_level = riskFilter;
      if (agentFilter) params.agent_id = agentFilter;
      if (search) params.search = search;
      return apiClient.getPage<UnifiedAction>('/ai/actions', params);
    },
  });

  const approveMutation = useMutation({
    mutationFn: (id: number) =>
      apiClient.post('/ai/actions/' + id + '/approve', { operator: 'user', reason: decisionReason }),
    onSuccess: () => {
      message.success('已批准');
      setModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('批准失败'),
  });

  const rejectMutation = useMutation({
    mutationFn: (id: number) =>
      apiClient.post('/ai/actions/' + id + '/reject', { operator: 'user', reason: decisionReason }),
    onSuccess: () => {
      message.success('已拒绝');
      setModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('拒绝失败'),
  });

  const executeMutation = useMutation({
    mutationFn: (id: number) =>
      apiClient.post('/ai/actions/' + id + '/execute', { operator: 'user' }),
    onSuccess: () => {
      message.success('已执行');
      queryClient.invalidateQueries({ queryKey: ['actions'] });
    },
    onError: () => message.error('执行失败'),
  });

  const openModal = (action: UnifiedAction) => {
    setSelectedAction(action);
    setDecisionReason('');
    setModalOpen(true);
  };

  const handleDecision = () => {
    if (!selectedAction) return;
    if (selectedAction.status === 'suggested' || selectedAction.status === 'pending') {
      approveMutation.mutate(selectedAction.id);
    } else {
      executeMutation.mutate(selectedAction.id);
    }
  };

  const columns = [
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      render: (t: string, r: UnifiedAction) => (
        <Text strong style={{ cursor: 'pointer' }} onClick={() => openModal(r)}>{t}</Text>
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
              onClick={() => executeMutation.mutate(record.id)}
              loading={executeMutation.isPending}
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
                onClick={() => openModal(record)}
              >
                批准
              </Button>
              <Button
                danger
                size="small"
                icon={<CloseCircleOutlined />}
                onClick={() => openModal(record)}
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
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <Title level={3} style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)' }}>Action Center</Title>

      <Card style={{ marginBottom: 16, background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
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

      <Modal
        title={selectedAction?.title || 'Action 详情'}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={
          selectedAction?.status === 'approved'
            ? [
                <Button key="cancel" onClick={() => setModalOpen(false)}>取消</Button>,
                <Button key="execute" type="primary" icon={<PlayCircleOutlined />} onClick={handleDecision}>
                  执行
                </Button>,
              ]
            : selectedAction?.status === 'rejected'
            ? null
            : [
                <Button key="reject" danger onClick={() => rejectMutation.mutate(selectedAction!.id)}>
                  拒绝
                </Button>,
                <Button key="approve" type="primary" onClick={handleDecision}>
                  批准
                </Button>,
              ]
        }
        width={640}
      >
        {selectedAction && (
          <div>
            <p><Text strong>描述：</Text>{selectedAction.description || '-'}</p>
            <p><Text strong>Agent：</Text>{selectedAction.agent_id}</p>
            <p><Text strong>类型：</Text>{selectedAction.action_type}</p>
            <p>
              <Text strong>风险：</Text>
              <Tag color={RISK_COLORS[selectedAction.risk_level]}>{selectedAction.risk_level}</Tag>
            </p>
            <p><Text strong>提案人：</Text>{selectedAction.proposed_by || '-'}</p>
            {selectedAction.rejection_reason && (
              <p><Text strong>拒绝原因：</Text><Text type="danger">{selectedAction.rejection_reason}</Text></p>
            )}
            <pre style={{ background: 'var(--s2)', padding: 12, borderRadius: 'var(--r2)', maxHeight: 300, overflow: 'auto', fontSize: 12 }}>
              {JSON.stringify(selectedAction.payload, null, 2)}
            </pre>
          </div>
        )}
      </Modal>
    </div>
  );
}
