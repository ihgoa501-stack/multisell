'use client';

import { useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Col,
  Empty,
  message,
  Modal,
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
  AlertOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  DatabaseOutlined,
  ShoppingOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  ArrowRightOutlined,
  BranchesOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';
import StatCard from '@/components/ui/StatCard';
import SectionCard from '@/components/ui/SectionCard';

const { Text } = Typography;

// ---------- Types ----------
interface RiskSummary {
  low_profit_products: number;
  missing_data_products: number;
  pending_approvals: number;
  sync_errors: number;
  total_candidates: number;
  total_recommendations: number;
  list_ready_products: number;
}

interface Suggestion {
  id: number;
  product_id: number;
  product_title: string;
  agent_source: string;
  suggestion: string;
  decision: string;
  reason: string;
  confidence: number;
  risk_level: string;
  created_at: string;
  listing_task_id?: number | null;
}

interface PlatformSync {
  platform_id: number;
  platform_name: string;
  mode: string;
  orders_sync: string;
  products_sync: string;
  fees_sync: string;
  settlements_sync: string;
  last_sync_time: string;
}

interface AgentActivity {
  currently_running: number;
  completed_today: number;
  failed_today: number;
  recent_events: Array<{
    id: number;
    agent_id: string;
    title: string;
    status: string;
    risk_level: string;
    created_at: string;
    summary: string;
  }>;
}

interface PipelineChain {
  chains: Array<{
    name: string;
    steps: Array<{
      agent_id: string;
      description: string;
      status: string;
    }>;
    overall_health: string;
  }>;
}

// ---------- Color helpers ----------
const suggestionColor = (s: string): string => {
  if (s === '建议上架') return 'green';
  if (s === '谨慎上架') return 'orange';
  if (s === '不建议上架') return 'red';
  return 'blue';
};

const riskColor = (level: string): string => {
  if (level === 'high') return 'red';
  if (level === 'medium') return 'orange';
  if (level === 'low') return 'green';
  return 'blue';
};

const modeColor = (mode: string): string => {
  if (mode === 'mock') return 'orange';
  if (mode === 'sandbox') return 'blue';
  if (mode === 'production') return 'red';
  return 'default';
};

const modeLabel = (mode: string): string => {
  if (mode === 'mock') return '模拟';
  if (mode === 'sandbox') return '沙箱';
  if (mode === 'production') return '生产';
  return mode;
};

const decisionColor = (d: string): string => {
  if (d === 'list') return 'green';
  if (d === 'cautious') return 'orange';
  if (d === 'skip') return 'red';
  return 'blue';
};

const decisionLabel = (d: string): string => {
  if (d === 'list') return '推荐上架';
  if (d === 'cautious') return '谨慎';
  if (d === 'skip') return '跳过';
  return d;
};

const confidenceColor = (v: number): string => {
  if (v >= 0.8) return 'green';
  if (v >= 0.5) return 'orange';
  return 'red';
};

const statusColor = (status: string): string => {
  const s = status.toLowerCase();
  if (s === 'completed' || s === 'success') return 'green';
  if (s === 'running' || s === 'in_progress' || s === 'processing') return 'blue';
  if (s === 'failed' || s === 'error') return 'red';
  if (s === 'pending' || s === 'queued') return 'default';
  if (s === 'blocked' || s === 'warning') return 'orange';
  return 'blue';
};

// ---------- Page ----------
export default function OwnerPage() {
  const qc = useQueryClient();
  const [suggestionFilter, setSuggestionFilter] = useState<string>('');
  const [approvalModal, setApprovalModal] = useState<Suggestion | null>(null);
  const [approvalAction, setApprovalAction] = useState<'approve' | 'reject' | null>(null);

  // Risk summary
  const { data: riskSummary } = useQuery({
    queryKey: ['owner-risk-summary'],
    queryFn: async () => {
      const res = await apiClient.get<RiskSummary>('/v1/owner/risk-summary');
      return res.data;
    },
  });

  // Suggestions
  const { data: suggestions, isLoading: suggestionsLoading } = useQuery({
    queryKey: ['owner-suggestions'],
    queryFn: async () => {
      const res = await apiClient.get<Suggestion[]>('/v1/owner/suggestions', { limit: '50' });
      return res.data ?? [];
    },
  });

  // Platform sync
  const { data: platformSync, isLoading: syncLoading } = useQuery({
    queryKey: ['owner-platform-sync'],
    queryFn: async () => {
      const res = await apiClient.get<PlatformSync[]>('/v1/owner/platform-sync');
      return res.data ?? [];
    },
  });

  // Agent activity
  const { data: agentActivity, isLoading: activityLoading } = useQuery({
    queryKey: ['owner-agent-activity'],
    queryFn: async () => {
      const res = await apiClient.get<AgentActivity>('/v1/owner/agent-activity');
      return res.data;
    },
  });

  // Pipeline chain
  const { data: pipelineChain, isLoading: pipelineLoading } = useQuery({
    queryKey: ['owner-pipeline-chain'],
    queryFn: async () => {
      const res = await apiClient.get<PipelineChain>('/v1/owner/pipeline-chain');
      return res.data;
    },
  });

  // Listing task approval
  const approveListingTask = useMutation({
    mutationFn: async (params: { taskId: number }) => {
      await apiClient.put(`/v1/listing-tasks/${params.taskId}`, {
        status: 'approved',
        updated_by: getCurrentOperator(),
      });
      return params;
    },
    onSuccess: () => {
      message.success('已批准上架');
      setApprovalModal(null);
      qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
      qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectListingTask = useMutation({
    mutationFn: async (params: { taskId: number }) => {
      await apiClient.put(`/v1/listing-tasks/${params.taskId}`, {
        status: 'rejected',
        updated_by: getCurrentOperator(),
      });
      return params;
    },
    onSuccess: () => {
      message.success('已拒绝上架');
      setApprovalModal(null);
      qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
      qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const refreshAll = () => {
    qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
    qc.invalidateQueries({ queryKey: ['owner-platform-sync'] });
    qc.invalidateQueries({ queryKey: ['owner-agent-activity'] });
    qc.invalidateQueries({ queryKey: ['owner-pipeline-chain'] });
  };

  // Compute stats
  const listReady = useMemo(
    () => (suggestions ?? []).filter((s) => s.decision === 'list').length,
    [suggestions]
  );

  // Filter suggestions
  const filteredSuggestions = useMemo(() => {
    if (!suggestionFilter) return suggestions ?? [];
    return (suggestions ?? []).filter((s) => s.decision === suggestionFilter);
  }, [suggestions, suggestionFilter]);

  // Check if a suggestion has a listing task we can approve
  const handleApprove = async (s: Suggestion) => {
    setApprovalModal(s);
    setApprovalAction('approve');
  };

  const handleReject = (s: Suggestion) => {
    setApprovalModal(s);
    setApprovalAction('reject');
  };

  const confirmApproval = async () => {
    if (!approvalModal) return;
    const taskId = approvalModal.listing_task_id;
    if (!taskId) {
      message.error('该建议没有对应的刊登任务，无法审批');
      setApprovalModal(null);
      return;
    }
    if (approvalAction === 'approve') {
      approveListingTask.mutate({ taskId });
    } else {
      rejectListingTask.mutate({ taskId });
    }
  };

  // ---------- Render ----------
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%', fontFamily: 'var(--body)' }}>
      {/* Page header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 'var(--space-lg)',
        }}
      >
        <h1 style={{
          fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)',
          color: 'var(--t1)', margin: 0,
        }}>
          Owner 经营总控台
        </h1>
        <Button icon={<ReloadOutlined />} onClick={refreshAll}>
          刷新
        </Button>
      </div>

      {/* System status banner */}
      <div
        style={{
          display: 'flex', alignItems: 'center', gap: 12, padding: '10px 16px',
          marginBottom: 'var(--space-lg)', borderRadius: 8,
          background: 'var(--y1)', border: '1px solid var(--y3)',
        }}
      >
        <div
          style={{
            width: 8, height: 8, borderRadius: '50%',
            background: 'var(--y4)', flexShrink: 0,
          }}
        />
        <div style={{ fontSize: '0.78rem', fontWeight: 600, color: 'var(--y6)', flexShrink: 0 }}>
          模拟环境 · Mock Data
        </div>
        <div style={{ fontSize: '0.7rem', color: 'var(--y5)', flex: 1 }}>
          当前页面展示的是本地模拟数据，不涉及真实交易。平台集成处于沙箱模式。
        </div>
        <Tag color="orange">Mock</Tag>
        <Tag color="blue">沙箱</Tag>
        <Tag color="default" style={{ opacity: 0.5 }}>生产</Tag>
      </div>

      {/* P0 local closed-loop flow */}
      <div
        style={{
          display: 'flex', alignItems: 'center', gap: 8, padding: '12px 16px',
          marginBottom: 'var(--space-lg)', borderRadius: 8,
          background: 'var(--s1)', border: '1px solid var(--bd)',
        }}
      >
        <div style={{ fontSize: '0.72rem', fontWeight: 600, color: 'var(--t2)', flexShrink: 0 }}>
          本地闭环流程 →
        </div>
        {[
          { label: '候选商品', href: '/candidates', icon: '📦', color: 'var(--i4)' },
          { label: '完整性评估', icon: '🔍', color: 'var(--c4)' },
          { label: '上架建议', icon: '💡', color: 'var(--g4)' },
          { label: 'Owner 审批', href: '/approval', icon: '✅', color: 'var(--g4)' },
        ].map((step, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            {i > 0 && <ArrowRightOutlined style={{ fontSize: '0.6rem', color: 'var(--t4)' }} />}
            {step.href ? (
              <a
                href={step.href}
                style={{
                  display: 'flex', alignItems: 'center', gap: 4,
                  padding: '3px 10px', borderRadius: 6, cursor: 'pointer',
                  fontSize: '0.72rem', fontWeight: 500,
                  background: 'var(--s2)', color: step.color,
                  border: '1px solid var(--bd)', textDecoration: 'none',
                }}
              >
                <span style={{ fontSize: '0.75rem' }}>{step.icon}</span>
                {step.label}
              </a>
            ) : (
              <span
                style={{
                  display: 'flex', alignItems: 'center', gap: 4,
                  padding: '3px 10px', borderRadius: 6,
                  fontSize: '0.72rem', fontWeight: 500,
                  background: 'var(--s2)', color: 'var(--t2)',
                }}
              >
                <span style={{ fontSize: '0.75rem' }}>{step.icon}</span>
                {step.label}
              </span>
            )}
          </div>
        ))}
      </div>

      {/* Risk summary cards */}
      <Row gutter={16} style={{ marginBottom: 'var(--space-lg)' }}>
        <Col xs={12} sm={6}>
          <StatCard title="待审批上架" value={riskSummary?.pending_approvals ?? '-'}
            prefix={<ClockCircleOutlined style={{ color: 'var(--y4)' }} />}
            valueStyle={{ color: (riskSummary?.pending_approvals ?? 0) > 0 ? 'var(--y4)' : 'var(--g4)' }} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="低利润商品" value={riskSummary?.low_profit_products ?? '-'}
            prefix={<WarningOutlined style={{ color: 'var(--r4)' }} />}
            valueStyle={{ color: (riskSummary?.low_profit_products ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)' }} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="资料不完整商品" value={riskSummary?.missing_data_products ?? '-'}
            prefix={<AlertOutlined style={{ color: 'var(--y4)' }} />}
            valueStyle={{ color: (riskSummary?.missing_data_products ?? 0) > 0 ? 'var(--y4)' : 'var(--g4)' }} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="同步异常" value={riskSummary?.sync_errors ?? '-'}
            prefix={<ApiOutlined style={{ color: (riskSummary?.sync_errors ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)' }} />}
            valueStyle={{ color: (riskSummary?.sync_errors ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)' }} />
        </Col>
      </Row>

      {/* Secondary summary row */}
      <Row gutter={16} style={{ marginBottom: 'var(--space-lg)' }}>
        <Col xs={12} sm={6}>
          <StatCard title="候选商品总数" value={riskSummary?.total_candidates ?? '-'}
            prefix={<ShoppingOutlined style={{ color: 'var(--i4)' }} />} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="评估建议总数" value={riskSummary?.total_recommendations ?? '-'}
            prefix={<ThunderboltOutlined style={{ color: 'var(--i4)' }} />} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="推荐上架" value={riskSummary?.list_ready_products ?? listReady}
            prefix={<CheckOutlined style={{ color: 'var(--g4)' }} />}
            valueStyle={{ color: 'var(--g4)' }} />
        </Col>
        <Col xs={12} sm={6}>
          <StatCard title="演示模式" value="Mock / 模拟"
            prefix={<DatabaseOutlined style={{ color: 'var(--y4)' }} />} />
        </Col>
      </Row>

      <Row gutter={16}>
        {/* Left: Agent suggestions */}
        <Col xs={24} lg={17}>
          <SectionCard title={<><RobotOutlined /> Agent 上架建议 <Tag color="orange" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>Mock</Tag></>}>
              <Space style={{ marginBottom: 'var(--space-md)' }}>
                <Button
                  size="small"
                  onClick={() => setSuggestionFilter('')}
                  type={suggestionFilter === '' ? 'primary' : 'default'}
                >
                  全部
                </Button>
                <Button
                  size="small"
                  type={suggestionFilter === 'list' ? 'primary' : 'default'}
                  style={suggestionFilter === 'list' ? { backgroundColor: 'var(--g4)', borderColor: 'var(--g4)' } : {}}
                  onClick={() => setSuggestionFilter('list')}
                >
                  推荐上架
                </Button>
                <Button
                  size="small"
                  type={suggestionFilter === 'cautious' ? 'primary' : 'default'}
                  style={suggestionFilter === 'cautious' ? { backgroundColor: 'var(--y4)', borderColor: 'var(--y4)' } : {}}
                  onClick={() => setSuggestionFilter('cautious')}
                >
                  谨慎
                </Button>
                <Button
                  size="small"
                  danger
                  type={suggestionFilter === 'skip' ? 'primary' : 'default'}
                  onClick={() => setSuggestionFilter('skip')}
                >
                  不建议
                </Button>
              </Space>
              <Spin spinning={suggestionsLoading}>
                {filteredSuggestions.length === 0 && !suggestionsLoading ? (
                  <Empty description="暂无建议数据" />
                ) : (
                  <Table
                    rowKey="id"
                    dataSource={filteredSuggestions}
                    size="small"
                    scroll={{ x: 'max-content' }}
                    pagination={{ pageSize: 10, showSizeChanger: false }}
                    columns={[
                      {
                        title: '商品',
                        dataIndex: 'product_title',
                        width: 160,
                        ellipsis: true,
                        render: (v: string) => <Text strong>{v || '-'}</Text>,
                      },
                      {
                        title: '建议',
                        dataIndex: 'suggestion',
                        width: 110,
                        render: (v: string) => <Tag color={suggestionColor(v)}>{v}</Tag>,
                      },
                      {
                        title: '评估结论',
                        dataIndex: 'decision',
                        width: 100,
                        render: (v: string) => <Tag color={decisionColor(v)}>{decisionLabel(v)}</Tag>,
                      },
                      {
                        title: '理由',
                        dataIndex: 'reason',
                        width: 280,
                        ellipsis: true,
                        render: (v: string) => <Text type="secondary" ellipsis={{ tooltip: v }}>{v}</Text>,
                      },
                      {
                        title: '置信度',
                        dataIndex: 'confidence',
                        width: 80,
                        render: (v: number) => (
                          <Tag color={confidenceColor(v)}>{(v * 100).toFixed(0)}%</Tag>
                        ),
                      },
                      {
                        title: '风险',
                        dataIndex: 'risk_level',
                        width: 80,
                        render: (v: string) => <Tag color={riskColor(v)}>{v}</Tag>,
                      },
                      {
                        title: '时间',
                        dataIndex: 'created_at',
                        width: 140,
                      },
                      {
                        title: '操作',
                        key: 'actions',
                        width: 130,
                        fixed: 'right' as const,
                        render: (_: unknown, record: Suggestion) => (
                          <Space size="small">
                            <Button
                              size="small"
                              type="primary"
                              icon={<CheckOutlined />}
                              disabled={record.decision !== 'list'}
                              onClick={() => handleApprove(record)}
                            >
                              批准
                            </Button>
                            <Button
                              size="small"
                              danger
                              icon={<CloseOutlined />}
                              onClick={() => handleReject(record)}
                            >
                              拒绝
                            </Button>
                          </Space>
                        ),
                      },
                    ]}
                  />
                )}
              </Spin>
          </SectionCard>
        </Col>

        {/* Right: Platform sync status */}
        <Col xs={24} lg={7}>
          <SectionCard title={<><ApiOutlined /> 平台同步状态 <Tag color="blue" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>沙箱</Tag></>}>
              <Spin spinning={syncLoading}>
                {(platformSync ?? []).length === 0 && !syncLoading ? (
                  <Empty description="暂无数据" />
                ) : (
                  <Space direction="vertical" style={{ width: '100%' }} size="small">
                    {(platformSync ?? []).map((p) => (
                      <div key={p.platform_id} style={{
                        background: 'var(--s1)', border: '1px solid var(--bd)',
                        borderRadius: 8, padding: 12,
                      }}>
                        <div style={{
                          display: 'flex', justifyContent: 'space-between',
                          alignItems: 'center', marginBottom: 'var(--space-sm)',
                        }}>
                          <Text strong>{p.platform_name}</Text>
                          <Tag color={modeColor(p.mode)}>{modeLabel(p.mode)}</Tag>
                        </div>
                        <Space direction="vertical" style={{ width: '100%' }} size={4}>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 'var(--text-small)' }}>订单</Text>
                            <Badge status={p.orders_sync === 'success' ? 'success' : 'error'} text={p.orders_sync} />
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 'var(--text-small)' }}>商品</Text>
                            <Badge status={p.products_sync === 'success' ? 'success' : 'error'} text={p.products_sync} />
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 'var(--text-small)' }}>费用</Text>
                            <Badge status={p.fees_sync === 'success' ? 'success' : 'error'} text={p.fees_sync} />
                          </div>
                          {p.last_sync_time && (
                            <div style={{ display: 'flex', justifyContent: 'space-between', paddingTop: 4 }}>
                              <Text type="secondary" style={{ fontSize: 'var(--text-small)' }}>最后同步</Text>
                              <Text style={{ fontSize: 'var(--text-small)' }}>{p.last_sync_time}</Text>
                            </div>
                          )}
                        </Space>
                      </div>
                    ))}
                  </Space>
                )}
              </Spin>
          </SectionCard>
        </Col>
      </Row>

      {/* Agent Activity Feed */}
      <div
        style={{
          background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
          marginBottom: 'var(--space-lg)',
        }}
      >
        <div style={{
          padding: '12px 16px', borderBottom: '1px solid var(--bd)',
          display: 'flex', alignItems: 'center', gap: 8,
          fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
        }}>
          <RobotOutlined /> Agent 活动摘要
          <Tag color="purple" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>Live</Tag>
        </div>
        <div style={{ padding: 16 }}>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col xs={8}>
              <div style={{ background: 'var(--bg)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
                <Statistic
                  title="正在运行"
                  value={agentActivity?.currently_running ?? '-'}
                  valueStyle={{ color: 'var(--i4)' }}
                  prefix={<ThunderboltOutlined />}
                  loading={activityLoading}
                />
              </div>
            </Col>
            <Col xs={8}>
              <div style={{ background: 'var(--bg)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
                <Statistic
                  title="今日完成"
                  value={agentActivity?.completed_today ?? '-'}
                  valueStyle={{ color: 'var(--g4)' }}
                  prefix={<CheckCircleOutlined />}
                  loading={activityLoading}
                />
              </div>
            </Col>
            <Col xs={8}>
              <div style={{ background: 'var(--bg)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
                <Statistic
                  title="今日失败"
                  value={agentActivity?.failed_today ?? '-'}
                  valueStyle={{
                    color: (agentActivity?.failed_today ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)',
                  }}
                  prefix={<CloseCircleOutlined />}
                  loading={activityLoading}
                />
              </div>
            </Col>
          </Row>
          <Spin spinning={activityLoading}>
            {(agentActivity?.recent_events ?? []).length === 0 && !activityLoading ? (
              <Empty description="暂无活动记录" />
            ) : (
              <Table
                rowKey="id"
                dataSource={agentActivity?.recent_events ?? []}
                size="small"
                pagination={false}
                columns={[
                  { title: 'Agent', dataIndex: 'agent_id', width: 100 },
                  { title: '标题', dataIndex: 'title', ellipsis: true, render: (v: string) => <Text strong>{v || '-'}</Text> },
                  { title: '状态', dataIndex: 'status', width: 100, render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag> },
                  { title: '风险', dataIndex: 'risk_level', width: 80, render: (v: string) => <Tag color={riskColor(v)}>{v}</Tag> },
                  { title: '时间', dataIndex: 'created_at', width: 150 },
                ]}
              />
            )}
          </Spin>
        </div>
      </div>

      {/* Pipeline Chain Status */}
      <div
        style={{
          background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
          marginBottom: 'var(--space-lg)',
        }}
      >
        <div style={{
          padding: '12px 16px', borderBottom: '1px solid var(--bd)',
          display: 'flex', alignItems: 'center', gap: 8,
          fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
        }}>
          <BranchesOutlined /> Agent 流程管道
          <Tag color="purple" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>Live</Tag>
        </div>
        <div style={{ padding: 16 }}>
          <Spin spinning={pipelineLoading}>
            {(pipelineChain?.chains ?? []).length === 0 && !pipelineLoading ? (
              <Empty description="暂无管道数据" />
            ) : (
              <Space direction="vertical" style={{ width: '100%' }} size="middle">
                {(pipelineChain?.chains ?? []).map((chain, idx) => (
                  <div key={idx} style={{
                    background: 'var(--bg)', border: '1px solid var(--bd)',
                    borderRadius: 8, padding: 16,
                  }}>
                    <div style={{
                      display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16,
                    }}>
                      <span style={{ fontWeight: 600, fontSize: '0.85rem', color: 'var(--t1)' }}>{chain.name}</span>
                      <Tag color={chain.overall_health === 'ok' ? 'green' : chain.overall_health === 'warn' ? 'orange' : 'red'}>
                        {chain.overall_health === 'ok' ? '正常' : chain.overall_health === 'warn' ? '警告' : '严重'}
                      </Tag>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                      {chain.steps.map((step, si) => (
                        <div key={si} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                          {si > 0 && <ArrowRightOutlined style={{ color: 'var(--t4)', fontSize: '0.7rem' }} />}
                          <div style={{
                            display: 'flex', alignItems: 'center', gap: 6,
                            padding: '6px 12px', borderRadius: 6,
                            background:
                              step.status === 'completed' ? 'var(--g1)'
                              : step.status === 'running' ? 'var(--i1)'
                              : step.status === 'failed' ? 'var(--r1)'
                              : step.status === 'blocked' ? 'var(--y1)'
                              : 'var(--s2)',
                            border: `1px solid ${
                              step.status === 'completed' ? 'var(--g3)'
                              : step.status === 'running' ? 'var(--i3)'
                              : step.status === 'failed' ? 'var(--r3)'
                              : step.status === 'blocked' ? 'var(--y3)'
                              : 'var(--bd)'
                            }`,
                          }}>
                            <div style={{
                              width: 6, height: 6, borderRadius: '50%', flexShrink: 0,
                              background:
                                step.status === 'completed' ? 'var(--g4)'
                                : step.status === 'running' ? 'var(--i4)'
                                : step.status === 'failed' ? 'var(--r4)'
                                : step.status === 'blocked' ? 'var(--y4)'
                                : 'var(--t4)',
                            }} />
                            <span style={{ fontWeight: 500, fontSize: '0.72rem', color: 'var(--t1)' }}>
                              {step.agent_id}
                            </span>
                            {step.description && (
                              <span style={{ fontSize: '0.68rem', color: 'var(--t2)' }}>
                                ({step.description})
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </Space>
            )}
          </Spin>
        </div>
      </div>

      {/* Approval confirmation modal */}
      <Modal
        title={approvalAction === 'approve' ? '确认批准上架' : '确认拒绝上架'}
        open={!!approvalModal}
        onCancel={() => { setApprovalModal(null); setApprovalAction(null); }}
        onOk={confirmApproval}
        confirmLoading={approveListingTask.isPending || rejectListingTask.isPending}
        okText={approvalAction === 'approve' ? '批准' : '拒绝'}
        cancelText="取消"
        okButtonProps={{ danger: approvalAction === 'reject' }}
      >
        <p>
          {approvalAction === 'approve'
            ? `确定批准商品 "${approvalModal?.product_title || `ID:${approvalModal?.product_id}`}" 上架？`
            : `确定拒绝商品 "${approvalModal?.product_title || `ID:${approvalModal?.product_id}`}" 上架？`}
        </p>
        {approvalModal && (
          <div style={{ background: 'var(--bg)', padding: 12, borderRadius: 6, marginTop: 8 }}>
            <Text type="secondary">{approvalModal.reason}</Text>
          </div>
        )}
      </Modal>
    </div>
  );
}
