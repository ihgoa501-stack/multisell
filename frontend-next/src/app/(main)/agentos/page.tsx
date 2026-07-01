'use client';

import { useState } from 'react';
import {
  Badge,
  Button,
  Collapse,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  message,
  Row,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import {
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  ThunderboltOutlined,
  AlertOutlined,
  ClockCircleOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
  ApiOutlined,
  ToolOutlined,
  HeartOutlined,
  RobotOutlined,
  BranchesOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';

const { Text } = Typography;

// ---------- Types ----------
interface AgentOSOverview {
  squads: SquadInfo[];
  agents: AgentInfo[];
  pending_by_risk: Record<string, number>;
  pending_total: number;
  sla_breached: number;
  work_queue_len: number;
  avg_confidence?: number;
}

interface SquadInfo {
  squad_id: string;
  name: string;
  agent_count: number;
  trace_count: number;
  action_count: number;
  pending_count: number;
  health: string;
}

interface AgentInfo {
  agent_id: string;
  name: string;
  squad: string;
  status: string;
}

interface WorkItem {
  id: string;
  title: string;
  agent_id: string;
  squad_id?: string;
  risk_level: string;
  confidence: number;
  status: string;
  proposed_at?: string;
  trace_id?: string;
}

interface AutonomyEntry {
  agent_id: string;
  autonomy_level: string;
  requires_approval: boolean;
  max_actions_per_hour: number;
}

interface WorkItemsResponse {
  items: WorkItem[];
  total: number;
}

interface AgentTimelineEntry {
  agent_id: string;
  agent_name: string;
  recent_actions: Array<{
    id: number;
    title: string;
    status: string;
    risk_level: string;
    confidence: number | null;
    entity_type: string;
    entity_id: number | null;
    created_at: string;
  }>;
  status_summary: Record<string, number>;
}

interface WorkItemDetail {
  id: number;
  title: string;
  agent_id: string;
  squad_id: string;
  risk_level: string;
  status: string;
  confidence: number | null;
  proposed_at: string;
  decision_point: string;
  reason: string;
  input_summary: string;
  output_summary: string;
  entity_type: string;
  entity_id: number | null;
  entity_status: string;
  approval: {
    id: number;
    status: string;
    risk_level: string;
  } | null;
  trace_id: string | null;
  upstream_items: Array<{ id: number; type: string; title: string; status: string }>;
  downstream_items: Array<{ id: number; type: string; title: string; status: string }>;
  audit_logs: Array<{ id: number; action: string; content: string; operator: string; created_at: string }>;
}

// ---------- Color helpers ----------
const riskColor = (level: string): string => {
  if (level === 'high' || level === 'critical') return 'red';
  if (level === 'medium') return 'orange';
  if (level === 'low') return 'green';
  return 'blue';
};

const statusColor = (status: string): string => {
  if (status === 'suggested') return 'blue';
  if (status === 'approved' || status === 'executed') return 'green';
  if (status === 'executing') return 'cyan';
  if (status === 'rejected' || status === 'failed') return 'red';
  if (status === 'reviewed') return 'default';
  return 'blue';
};

const healthColor = (health: string): string => {
  if (health === 'ok') return 'green';
  if (health === 'warn') return 'orange';
  if (health === 'critical') return 'red';
  return 'blue';
};

const autonomyColor = (level: string): string => {
  if (level === 'advisory') return 'green';
  if (level === 'guided') return 'blue';
  if (level === 'autonomous') return 'cyan';
  if (level === 'supervised') return 'orange';
  return 'default';
};

// ---------- Page ----------
export default function AgentOSPage() {
  const qc = useQueryClient();
  const [workFilter, setWorkFilter] = useState<{
    status?: string;
    risk_level?: string;
  }>({});

  // Overview
  const { data: overview, isLoading: overviewLoading } = useQuery({
    queryKey: ['agentos-overview'],
    queryFn: async () => {
      const res = await apiClient.get<AgentOSOverview>('/v1/agentos');
      return res.data;
    },
  });

  // Work items
  const { data: workItemsData, isLoading: workLoading } = useQuery({
    queryKey: ['agentos-work-items', workFilter],
    queryFn: async () => {
      const params: Record<string, string> = { limit: '50' };
      if (workFilter.status) params.status = workFilter.status;
      if (workFilter.risk_level) params.risk_level = workFilter.risk_level;
      const res = await apiClient.get<WorkItemsResponse>('/v1/agentos/work-items', params);
      return res.data;
    },
  });

  // Autonomy
  const { data: autonomyData, isLoading: autonomyLoading } = useQuery({
    queryKey: ['agentos-autonomy'],
    queryFn: async () => {
      const res = await apiClient.get<AutonomyEntry[]>('/v1/agentos/autonomy');
      return res.data ?? [];
    },
  });

  // AIOS system health
  interface AIOSHealth {
    status: string;
    runtime: string;
    tools: number;
    guardrails: string;
    agents: number;
    observability: boolean;
  }
  const { data: aiosHealth, isLoading: healthLoading } = useQuery({
    queryKey: ['aios-health'],
    queryFn: async () => {
      const res = await apiClient.get<AIOSHealth>('/v1/aios/health');
      return res.data;
    },
  });

  // Action operations
  const approveMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/approve`, {
        operator: getCurrentOperator(),
      }),
    onSuccess: () => {
      message.success('已批准');
      qc.invalidateQueries({ queryKey: ['agentos-work-items'] });
      qc.invalidateQueries({ queryKey: ['agentos-overview'] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/reject`, {
        operator: getCurrentOperator(),
        reason: 'manual reject',
      }),
    onSuccess: () => {
      message.success('已拒绝');
      qc.invalidateQueries({ queryKey: ['agentos-work-items'] });
      qc.invalidateQueries({ queryKey: ['agentos-overview'] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const executeMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/execute`, {
        operator: getCurrentOperator(),
      }),
    onSuccess: () => {
      message.success('已执行');
      qc.invalidateQueries({ queryKey: ['agentos-work-items'] });
      qc.invalidateQueries({ queryKey: ['agentos-overview'] });
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  const refreshAll = () => {
    qc.invalidateQueries({ queryKey: ['agentos-overview'] });
    qc.invalidateQueries({ queryKey: ['agentos-work-items'] });
    qc.invalidateQueries({ queryKey: ['agentos-autonomy'] });
    qc.invalidateQueries({ queryKey: ['aios-health'] });
    qc.invalidateQueries({ queryKey: ['agentos-timeline'] });
  };

  // Work item detail drawer
  const [selectedWorkItemId, setSelectedWorkItemId] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const { data: workItemDetail, isLoading: detailLoading } = useQuery({
    queryKey: ['agentos-work-item-detail', selectedWorkItemId],
    queryFn: async () => {
      const res = await apiClient.get<WorkItemDetail>(`/v1/agentos/work-items/${selectedWorkItemId}`);
      return res.data;
    },
    enabled: !!selectedWorkItemId,
  });

  // Agent timeline
  const { data: timelineData, isLoading: timelineLoading } = useQuery({
    queryKey: ['agentos-timeline'],
    queryFn: async () => {
      const res = await apiClient.get<AgentTimelineEntry[]>('/v1/agentos/agent-timeline');
      return res.data ?? [];
    },
  });

  const workColumns = [
    {
      title: '标题',
      dataIndex: 'title',
      ellipsis: true,
      render: (v: string) => (
        <Space size={4}>
          <Text strong>{v}</Text>
        </Space>
      ),
    },
    { title: 'Agent', dataIndex: 'agent_id', width: 140 },
    {
      title: '风险',
      dataIndex: 'risk_level',
      width: 90,
      render: (v: string) => <Tag color={riskColor(v)}>{v}</Tag>,
    },
    {
      title: '置信度',
      dataIndex: 'confidence',
      width: 90,
      render: (v: number) => (
        <Tag color={v >= 0.8 ? 'green' : v >= 0.5 ? 'orange' : 'red'}>
          {(v * 100).toFixed(0)}%
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 240,
      fixed: 'right' as const,
      render: (_: unknown, record: WorkItem) => (
        <Space size="small">
          <Button
            size="small"
            type="primary"
            icon={<CheckOutlined />}
            loading={
              approveMutation.isPending && approveMutation.variables === record.id
            }
            onClick={(e) => {
              e.stopPropagation();
              approveMutation.mutate(record.id);
            }}
          >
            批准
          </Button>
          <Button
            size="small"
            danger
            icon={<CloseOutlined />}
            loading={
              rejectMutation.isPending && rejectMutation.variables === record.id
            }
            onClick={(e) => {
              e.stopPropagation();
              rejectMutation.mutate(record.id);
            }}
          >
            拒绝
          </Button>
          <Button
            size="small"
            icon={<ThunderboltOutlined />}
            loading={
              executeMutation.isPending && executeMutation.variables === record.id
            }
            onClick={(e) => {
              e.stopPropagation();
              executeMutation.mutate(record.id);
            }}
          >
            执行
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%', fontFamily: 'var(--body)' }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)', margin: 0 }}>
          AgentOS 驾驶舱
        </h1>
        <Button icon={<ReloadOutlined />} onClick={refreshAll}>
          刷新
        </Button>
      </div>

      {/* 顶部：统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="待审批总数"
              value={overview?.pending_total ?? 0}
              prefix={<ClockCircleOutlined style={{ color: 'var(--y4)' }} />}
              valueStyle={{ color: 'var(--y4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="SLA 超期"
              value={overview?.sla_breached ?? 0}
              prefix={<AlertOutlined style={{ color: 'var(--r4)' }} />}
              valueStyle={{ color: 'var(--r4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="工作队列长度"
              value={overview?.work_queue_len ?? 0}
              prefix={<TeamOutlined style={{ color: 'var(--i4)' }} />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="平均置信度"
              value={
                overview?.avg_confidence !== undefined
                  ? `${(overview.avg_confidence * 100).toFixed(0)}%`
                  : '-'
              }
              prefix={<SafetyCertificateOutlined style={{ color: 'var(--g4)' }} />}
              valueStyle={{ color: 'var(--g4)' }}
            />
          </div>
        </Col>
      </Row>

      {/* AIOS 系统指标 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="已注册 Agent"
              value={aiosHealth?.agents ?? 0}
              prefix={<ApiOutlined style={{ color: 'var(--i4)' }} />}
              loading={healthLoading}
            />
          </div>
        </Col>
        <Col xs={12} sm={8}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="已注册 Tool"
              value={aiosHealth?.tools ?? 0}
              prefix={<ToolOutlined style={{ color: 'var(--g4)' }} />}
              loading={healthLoading}
            />
          </div>
        </Col>
        <Col xs={12} sm={8}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16, height: '100%' }}>
            <Statistic
              title="系统健康"
              value={aiosHealth?.status === 'ok' ? '正常' : aiosHealth?.status ?? '-'}
              prefix={<HeartOutlined style={{ color: aiosHealth?.status === 'ok' ? 'var(--g4)' : 'var(--r4)' }} />}
              valueStyle={{ color: aiosHealth?.status === 'ok' ? 'var(--g4)' : 'var(--r4)' }}
              loading={healthLoading}
            />
          </div>
        </Col>
      </Row>

      <Row gutter={16}>
        {/* 左侧：Squad 健康地图 + 工作队列 */}
        <Col xs={24} lg={17}>
          {/* Squad 健康地图 */}
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, marginBottom: 16 }}>
            <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--bd)', fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)' }}>
              Squad 健康地图
            </div>
            <div style={{ padding: 16 }}>
              <Spin spinning={overviewLoading}>
                {(overview?.squads ?? []).length === 0 && !overviewLoading ? (
                  <Empty description="暂无 Squad 数据" />
                ) : (
                  <Space direction="vertical" style={{ width: '100%' }} size="small">
                    {(overview?.squads ?? []).map((squad) => (
                      <div
                        key={squad.squad_id}
                        style={{
                          background: 'var(--s1)',
                          border: '1px solid var(--bd)',
                          borderRadius: 8,
                          borderLeft: `4px solid ${
                            squad.health === 'ok'
                              ? 'var(--g4)'
                              : squad.health === 'warn'
                                ? 'var(--y4)'
                                : squad.health === 'critical'
                                  ? 'var(--r4)'
                                  : 'var(--i4)'
                          }`,
                          padding: '12px 16px',
                        }}
                      >
                        <Row gutter={16} align="middle">
                          <Col xs={24} sm={6}>
                            <Space>
                              <Text strong>{squad.name}</Text>
                              <Badge
                                status={
                                  squad.health === 'ok'
                                    ? 'success'
                                    : squad.health === 'warn'
                                      ? 'warning'
                                      : squad.health === 'critical'
                                        ? 'error'
                                        : 'processing'
                                }
                                text={
                                  <Tag color={healthColor(squad.health)}>
                                    {squad.health}
                                  </Tag>
                                }
                              />
                            </Space>
                          </Col>
                          <Col xs={6} sm={4}>
                            <Statistic
                              title="Agent"
                              value={squad.agent_count}
                              valueStyle={{ fontSize: 16 }}
                            />
                          </Col>
                          <Col xs={6} sm={4}>
                            <Statistic
                              title="Trace"
                              value={squad.trace_count}
                              valueStyle={{ fontSize: 16 }}
                            />
                          </Col>
                          <Col xs={6} sm={4}>
                            <Statistic
                              title="Action"
                              value={squad.action_count}
                              valueStyle={{ fontSize: 16 }}
                            />
                          </Col>
                          <Col xs={6} sm={6}>
                            <Statistic
                              title="待办"
                              value={squad.pending_count}
                              valueStyle={{
                                fontSize: 16,
                                color: squad.pending_count > 0 ? 'var(--y4)' : 'var(--g4)',
                              }}
                            />
                          </Col>
                        </Row>
                      </div>
                    ))}
                  </Space>
                )}
              </Spin>
            </div>
          </div>

          {/* 待审批工作队列 */}
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--bd)', fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)' }}>
              待审批工作队列
            </div>
            <div style={{ padding: 16 }}>
              <Space style={{ marginBottom: 12 }}>
                <Button
                  size="small"
                  onClick={() => setWorkFilter({})}
                  type={!workFilter.risk_level && !workFilter.status ? 'primary' : 'default'}
                >
                  全部
                </Button>
                <Button
                  size="small"
                  danger
                  type={workFilter.risk_level === 'high' ? 'primary' : 'default'}
                  onClick={() => setWorkFilter({ risk_level: 'high' })}
                >
                  高风险
                </Button>
                <Button
                  size="small"
                  type={workFilter.risk_level === 'medium' ? 'primary' : 'default'}
                  style={
                    workFilter.risk_level === 'medium'
                      ? { backgroundColor: 'var(--y4)', borderColor: 'var(--y4)' }
                      : {}
                  }
                  onClick={() => setWorkFilter({ risk_level: 'medium' })}
                >
                  中风险
                </Button>
                <Button
                  size="small"
                  type={workFilter.risk_level === 'low' ? 'primary' : 'default'}
                  style={
                    workFilter.risk_level === 'low'
                      ? { backgroundColor: 'var(--g4)', borderColor: 'var(--g4)' }
                      : {}
                  }
                  onClick={() => setWorkFilter({ risk_level: 'low' })}
                >
                  低风险
                </Button>
              </Space>
              <Table
                rowKey="id"
                loading={workLoading}
                dataSource={workItemsData?.items ?? []}
                columns={workColumns}
                size="small"
                scroll={{ x: 'max-content' }}
                pagination={{
                  pageSize: 10,
                  total: workItemsData?.total ?? 0,
                  showSizeChanger: false,
                }}
                onRow={(record) => ({
                  onClick: () => {
                    setSelectedWorkItemId(record.id);
                    setDrawerOpen(true);
                  },
                  style: { cursor: 'pointer' },
                })}
              />
            </div>
          </div>
        </Col>

        {/* 右侧：Autonomy 控制面板 */}
        <Col xs={24} lg={7}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{ padding: '12px 16px', borderBottom: '1px solid var(--bd)', fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)' }}>
              Autonomy 控制面板（只读）
            </div>
            <div style={{ padding: 16 }}>
              <Spin spinning={autonomyLoading}>
                {(autonomyData ?? []).length === 0 && !autonomyLoading ? (
                  <Empty description="暂无 Autonomy 配置" />
                ) : (
                  <Space direction="vertical" style={{ width: '100%' }} size="small">
                    {(autonomyData ?? []).map((entry) => (
                      <div
                        key={entry.agent_id}
                        style={{
                          background: 'var(--s1)',
                          border: '1px solid var(--bd)',
                          borderRadius: 8,
                          padding: 12,
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                          }}
                        >
                          <Text strong>{entry.agent_id}</Text>
                          <Tag color={autonomyColor(entry.autonomy_level)}>
                            {entry.autonomy_level}
                          </Tag>
                        </div>
                        <div style={{ marginTop: 8 }}>
                          <Space direction="vertical" size={4} style={{ width: '100%' }}>
                            <div
                              style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                              }}
                            >
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                需要审批
                              </Text>
                              <Tag color={entry.requires_approval ? 'orange' : 'green'}>
                                {entry.requires_approval ? '是' : '否'}
                              </Tag>
                            </div>
                            <div
                              style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                              }}
                            >
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                每小时上限
                              </Text>
                              <Text style={{ fontSize: 12 }}>
                                {entry.max_actions_per_hour}
                              </Text>
                            </div>
                          </Space>
                        </div>
                      </div>
                    ))}
                  </Space>
                )}
              </Spin>
            </div>
          </div>
        </Col>
      </Row>

      {/* Agent Timeline */}
      <div
        style={{
          background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
          marginTop: 16,
        }}
      >
        <div style={{
          padding: '12px 16px', borderBottom: '1px solid var(--bd)',
          display: 'flex', alignItems: 'center', gap: 8,
          fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
        }}>
          <RobotOutlined /> Agent 活动时间线
        </div>
        <div style={{ padding: 16 }}>
          <Spin spinning={timelineLoading}>
            {(timelineData ?? []).length === 0 && !timelineLoading ? (
              <Empty description="暂无时间线数据" />
            ) : (
              <Collapse
                size="small"
                items={(timelineData ?? []).map((entry) => ({
                  key: entry.agent_id,
                  label: (
                    <Space>
                      <Text strong>{entry.agent_id}</Text>
                      <Text type="secondary" style={{ fontSize: '0.75rem' }}>{entry.agent_name}</Text>
                      {Object.entries(entry.status_summary ?? {}).map(([status, count]) => (
                        <Tag key={status} color={statusColor(status)}>{status}: {count}</Tag>
                      ))}
                    </Space>
                  ),
                  children: (
                    <Table
                      rowKey="id"
                      dataSource={entry.recent_actions ?? []}
                      size="small"
                      pagination={false}
                      columns={[
                        { title: '标题', dataIndex: 'title', ellipsis: true, render: (v: string) => <Text strong>{v || '-'}</Text> },
                        { title: '类型', dataIndex: 'entity_type', width: 100 },
                        { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag> },
                        { title: '风险', dataIndex: 'risk_level', width: 80, render: (v: string) => <Tag color={riskColor(v)}>{v}</Tag> },
                        {
                          title: '置信度', dataIndex: 'confidence', width: 80,
                          render: (v: number | null) => v !== null && v !== undefined
                            ? <Tag color={v >= 0.8 ? 'green' : v >= 0.5 ? 'orange' : 'red'}>{(v * 100).toFixed(0)}%</Tag>
                            : '-',
                        },
                        { title: '时间', dataIndex: 'created_at', width: 150 },
                      ]}
                    />
                  ),
                }))}
              />
            )}
          </Spin>
        </div>
      </div>

      {/* Work Item Detail Drawer */}
      <Drawer
        title={workItemDetail?.title ?? '工作项详情'}
        open={drawerOpen}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedWorkItemId(null);
        }}
        width={640}
        loading={detailLoading}
      >
        {workItemDetail ? (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Descriptions size="small" column={2} bordered>
              <Descriptions.Item label="ID">{workItemDetail.id}</Descriptions.Item>
              <Descriptions.Item label="Agent">{workItemDetail.agent_id}</Descriptions.Item>
              <Descriptions.Item label="Squad">{workItemDetail.squad_id}</Descriptions.Item>
              <Descriptions.Item label="风险">
                <Tag color={riskColor(workItemDetail.risk_level)}>{workItemDetail.risk_level}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColor(workItemDetail.status)}>{workItemDetail.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="置信度">
                {workItemDetail.confidence !== null ? `${(workItemDetail.confidence * 100).toFixed(0)}%` : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="决策点">{workItemDetail.decision_point}</Descriptions.Item>
              <Descriptions.Item label="提议时间">{workItemDetail.proposed_at}</Descriptions.Item>
              <Descriptions.Item label="Trace ID" span={2}>{workItemDetail.trace_id ?? '-'}</Descriptions.Item>
            </Descriptions>

            {workItemDetail.reason && (
              <>
                <Divider>决策理由</Divider>
                <div style={{ background: 'var(--bg)', padding: 12, borderRadius: 6, whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>
                  {workItemDetail.reason}
                </div>
              </>
            )}

            {workItemDetail.input_summary && (
              <>
                <Divider>输入摘要</Divider>
                <div style={{ background: 'var(--bg)', padding: 12, borderRadius: 6, whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>
                  {workItemDetail.input_summary}
                </div>
              </>
            )}

            {workItemDetail.output_summary && (
              <>
                <Divider>输出摘要</Divider>
                <div style={{ background: 'var(--bg)', padding: 12, borderRadius: 6, whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}>
                  {workItemDetail.output_summary}
                </div>
              </>
            )}

            <Divider>实体信息</Divider>
            <Descriptions size="small" column={2} bordered>
              <Descriptions.Item label="实体类型">{workItemDetail.entity_type}</Descriptions.Item>
              <Descriptions.Item label="实体 ID">{workItemDetail.entity_id ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="实体状态">{workItemDetail.entity_status}</Descriptions.Item>
            </Descriptions>

            {workItemDetail.approval && (
              <>
                <Divider>审批信息</Divider>
                <Descriptions size="small" column={2} bordered>
                  <Descriptions.Item label="审批 ID">{workItemDetail.approval.id}</Descriptions.Item>
                  <Descriptions.Item label="审批状态">
                    <Tag color={statusColor(workItemDetail.approval.status)}>{workItemDetail.approval.status}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="审批风险">
                    <Tag color={riskColor(workItemDetail.approval.risk_level)}>{workItemDetail.approval.risk_level}</Tag>
                  </Descriptions.Item>
                </Descriptions>
              </>
            )}

            {(workItemDetail.upstream_items ?? []).length > 0 && (
              <>
                <Divider>上游工作项 ({workItemDetail.upstream_items.length})</Divider>
                <Table
                  rowKey="id"
                  dataSource={workItemDetail.upstream_items}
                  size="small"
                  pagination={false}
                  columns={[
                    { title: 'ID', dataIndex: 'id', width: 60 },
                    { title: '类型', dataIndex: 'type', width: 80 },
                    { title: '标题', dataIndex: 'title', ellipsis: true },
                    { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag> },
                  ]}
                />
              </>
            )}

            {(workItemDetail.downstream_items ?? []).length > 0 && (
              <>
                <Divider>下游工作项 ({workItemDetail.downstream_items.length})</Divider>
                <Table
                  rowKey="id"
                  dataSource={workItemDetail.downstream_items}
                  size="small"
                  pagination={false}
                  columns={[
                    { title: 'ID', dataIndex: 'id', width: 60 },
                    { title: '类型', dataIndex: 'type', width: 80 },
                    { title: '标题', dataIndex: 'title', ellipsis: true },
                    { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag> },
                  ]}
                />
              </>
            )}

            {(workItemDetail.audit_logs ?? []).length > 0 && (
              <>
                <Divider>审计日志</Divider>
                <Table
                  rowKey="id"
                  dataSource={workItemDetail.audit_logs}
                  size="small"
                  pagination={false}
                  columns={[
                    { title: '操作', dataIndex: 'action', width: 100 },
                    { title: '内容', dataIndex: 'content', ellipsis: true },
                    { title: '操作人', dataIndex: 'operator', width: 80 },
                    { title: '时间', dataIndex: 'created_at', width: 150 },
                  ]}
                />
              </>
            )}
          </Space>
        ) : (
          <Empty description={detailLoading ? '加载中...' : '无法加载工作项详情'} />
        )}
      </Drawer>
    </div>
  );
}
