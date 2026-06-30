'use client';

import { useMemo, useState } from 'react';
import {
  Button,
  Col,
  Empty,
  Row,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  ReloadOutlined,
  CheckOutlined,
  AlertOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  DatabaseOutlined,
  ShoppingOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  ArrowRightOutlined,
} from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

// ---------- Types ----------
interface RiskSummary {
  low_profit_products: number;
  missing_data_products: number;
  pending_approvals: number;
  pending_approval_count?: number;
  blocked_listing_task_count?: number;
  recommended_listing_count?: number;
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
  // Enhanced feedback fields (optional, populated by backend when available)
  feedback_status?: 'pending' | 'adopted' | 'rejected' | 'executed' | 'execution_failed';
  feedback_note?: string;
  task_status?: string;
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

interface ApprovalRequest {
  id: number;
  product_id: number;
  request_type: string;
  requester: string;
  status: 'pending' | 'approved' | 'rejected';
  target_type?: string;
  target_id?: number;
  risk_level?: 'low' | 'medium' | 'high';
  reason?: string;
  created_at: string;
}

// ---------- Color helpers ----------
const suggestionColor = (s: string): string => {
  if (s === '建议上架') return 'green';
  if (s === '谨慎上架') return 'orange';
  if (s === '不建议上架') return 'red';
  return 'blue';
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

// ---------- Feedback status helpers ----------
const feedbackIconMap: Record<string, string> = {
  pending: '⌛',
  adopted: '✅',
  rejected: '❌',
  executed: '⚡',
  execution_failed: '⚠️',
};

const feedbackTooltipMap: Record<string, string> = {
  pending: '待执行：建议已提交，等待自动处理',
  adopted: '已采纳：建议已被 Owner 采纳',
  rejected: '已拒绝：建议已被 Owner 拒绝',
  executed: '已执行：建议已被成功执行',
  execution_failed: '执行失败：建议执行时发生错误，悬浮查看详情',
};

// ---------- Task status helpers ----------
const taskStatusColorMap: Record<string, string> = {
  pending: 'blue',
  blocked: 'red',
  pending_approval: 'orange',
  executing: 'processing',
  completed: 'success',
  failed: 'error',
  cancelled: 'default',
};

const taskStatusLabelMap: Record<string, string> = {
  pending: '待处理',
  blocked: '已阻断',
  pending_approval: '待审批',
  executing: '执行中',
  completed: '已完成',
  failed: '失败',
  cancelled: '已取消',
};

const taskStatusTooltipMap: Record<string, string> = {
  blocked: '该任务已被系统阻断，需查看审计日志了解原因',
  pending_approval: '该任务等待审批',
  pending: '任务等待处理',
  executing: '任务正在执行中',
  completed: '任务已完成',
  failed: '任务执行失败',
  cancelled: '任务已取消',
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

// ---------- Page ----------
export default function OwnerPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [suggestionFilter, setSuggestionFilter] = useState<string>('');

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

  // Pending approvals
  const { data: approvals } = useQuery({
    queryKey: ['owner-pending-approvals'],
    queryFn: async () => {
      const res = await apiClient.get<ApprovalRequest[]>('/v1/approval/my', { page: '1', size: '100' });
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

  const refreshAll = () => {
    qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
    qc.invalidateQueries({ queryKey: ['owner-pending-approvals'] });
    qc.invalidateQueries({ queryKey: ['owner-platform-sync'] });
  };

  const pendingPublishApprovals = (approvals ?? []).filter(
    (a: ApprovalRequest) => a.status === 'pending' && a.request_type === 'publish'
  );

  const listReady = useMemo(
    () => (suggestions ?? []).filter((s) => s.decision === 'list').length,
    [suggestions]
  );

  const filteredSuggestions = useMemo(() => {
    if (!suggestionFilter) return suggestions ?? [];
    return (suggestions ?? []).filter((s) => s.decision === suggestionFilter);
  }, [suggestions, suggestionFilter]);

  // ---------- Render ----------
  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%', fontFamily: 'var(--body)' }}>
      {/* Page header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
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

      {/* Inline styles for table row highlighting */}
      <style>{`
        .execution-failed-row {
          background: rgba(255, 77, 79, 0.04);
        }
        .execution-failed-row:hover td {
          background: rgba(255, 77, 79, 0.08) !important;
        }
      `}</style>

      {/* System status banner */}
      <div
        style={{
          display: 'flex', alignItems: 'center', gap: 12, padding: '10px 16px',
          marginBottom: 16, borderRadius: 8,
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
          marginBottom: 16, borderRadius: 8,
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
      {/* Next-action panel */}
      <div style={{
        background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
        padding: 16, marginBottom: 16,
      }}>
        {pendingPublishApprovals.length > 0 ? (
          <Space>
            <ClockCircleOutlined style={{ color: 'var(--y4)', fontSize: 18 }} />
            <Text strong>下一步：审核 {pendingPublishApprovals.length} 个上架请求</Text>
            <Button type="primary" onClick={() => router.push('/approval')}>去审批</Button>
          </Space>
        ) : (
          <Space>
            <CheckOutlined style={{ color: 'var(--g4)', fontSize: 18 }} />
            <Text strong>当前没有待审批上架请求</Text>
            <Button onClick={() => router.push('/candidates')}>去评估候选商品</Button>
          </Space>
        )}
      </div>
      {/* Next-action panel */}
      <div style={{
        background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8,
        padding: 16, marginBottom: 16,
      }}>
        {pendingPublishApprovals.length > 0 ? (
          <Space>
            <ClockCircleOutlined style={{ color: 'var(--y4)', fontSize: 18 }} />
            <Text strong>下一步：审核 {pendingPublishApprovals.length} 个上架请求</Text>
            <Button type="primary" onClick={() => router.push('/approval')}>去审批</Button>
          </Space>
        ) : (
          <Space>
            <CheckOutlined style={{ color: 'var(--g4)', fontSize: 18 }} />
            <Text strong>当前没有待审批上架请求</Text>
            <Button onClick={() => router.push('/candidates')}>去评估候选商品</Button>
          </Space>
        )}
      </div>

      {/* Risk summary cards */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="待审批动作"
              value={riskSummary?.pending_approval_count ?? riskSummary?.pending_approvals ?? '-'}
              prefix={<ClockCircleOutlined style={{ color: 'var(--y4)' }} />}
              valueStyle={{ color: (riskSummary?.pending_approval_count ?? 0) > 0 ? 'var(--y4)' : 'var(--g4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="被阻塞刊登任务"
              value={riskSummary?.blocked_listing_task_count ?? 0}
              prefix={<WarningOutlined style={{ color: 'var(--y4)' }} />}
              valueStyle={{ color: (riskSummary?.blocked_listing_task_count ?? 0) > 0 ? 'var(--y4)' : 'var(--g4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="建议上架商品"
              value={riskSummary?.recommended_listing_count ?? riskSummary?.list_ready_products ?? '-'}
              prefix={<CheckOutlined style={{ color: 'var(--g4)' }} />}
              valueStyle={{ color: 'var(--g4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="同步异常"
              value={riskSummary?.sync_errors ?? '-'}
              prefix={<ApiOutlined style={{ color: (riskSummary?.sync_errors ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)' }} />}
              valueStyle={{ color: (riskSummary?.sync_errors ?? 0) > 0 ? 'var(--r4)' : 'var(--g4)' }}
            />
          </div>
        </Col>
      </Row>

      {/* Secondary summary row */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="候选商品总数"
              value={riskSummary?.total_candidates ?? '-'}
              prefix={<ShoppingOutlined style={{ color: 'var(--i4)' }} />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="评估建议总数"
              value={riskSummary?.total_recommendations ?? '-'}
              prefix={<ThunderboltOutlined style={{ color: 'var(--i4)' }} />}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="低利润 / 不完整商品"
              value={((riskSummary?.low_profit_products ?? 0) + (riskSummary?.missing_data_products ?? 0)) || '-'}
              prefix={<AlertOutlined style={{ color: 'var(--r4)' }} />}
              valueStyle={{ color: ((riskSummary?.low_profit_products ?? 0) + (riskSummary?.missing_data_products ?? 0)) > 0 ? 'var(--r4)' : 'var(--g4)' }}
            />
          </div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8, padding: 16 }}>
            <Statistic
              title="演示模式"
              value="Mock / 模拟"
              prefix={<DatabaseOutlined style={{ color: 'var(--y4)' }} />}
            />
          </div>
        </Col>
      </Row>

      <Row gutter={16}>
        {/* Left: Agent suggestions */}
        <Col xs={24} lg={17}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{
              padding: '12px 16px', borderBottom: '1px solid var(--bd)',
              display: 'flex', alignItems: 'center', gap: 8,
              fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
            }}>
              Agent 上架建议
              <Tag color="orange" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>Mock</Tag>
            </div>
            <div style={{ padding: 16 }}>
              <Space style={{ marginBottom: 12 }}>
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
                    rowClassName={(record: Suggestion) =>
                      record.feedback_status === 'execution_failed' ? 'execution-failed-row' : ''
                    }
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
                        title: '反馈状态',
                        dataIndex: 'feedback_status',
                        width: 100,
                        render: (v: string | undefined, record: Suggestion) => {
                          if (!v) return <Text type="secondary" style={{ fontSize: 12 }}>-</Text>;
                          const icon = feedbackIconMap[v] || '?';
                          const tip = feedbackTooltipMap[v] || '';
                          if (v === 'execution_failed') {
                            return (
                              <Tooltip title={record.feedback_note || tip}>
                                <span style={{ cursor: 'pointer', whiteSpace: 'nowrap' }}>
                                  <span style={{ fontSize: '1rem' }}>{icon}</span>
                                  <Text type="danger" style={{ fontSize: 12, marginLeft: 4 }}>失败</Text>
                                </span>
                              </Tooltip>
                            );
                          }
                          return (
                            <Tooltip title={tip}>
                              <span style={{ fontSize: '1rem', cursor: 'default' }}>{icon}</span>
                            </Tooltip>
                          );
                        },
                      },
                      {
                        title: '任务状态',
                        dataIndex: 'task_status',
                        width: 100,
                        render: (v: string | undefined) => {
                          if (!v) return null;
                          const color = taskStatusColorMap[v] || 'default';
                          const label = taskStatusLabelMap[v] || v;
                          const tip = taskStatusTooltipMap[v] || '';
                          if (v === 'blocked') {
                            return (
                              <Tooltip title={tip}>
                                <Tag color={color} style={{ cursor: 'pointer' }}>{label}</Tag>
                              </Tooltip>
                            );
                          }
                          if (v === 'failed') {
                            return (
                              <Tooltip title="任务执行失败，可查看错误信息">
                                <Tag color={color}>{label}</Tag>
                              </Tooltip>
                            );
                          }
                          if (tip) {
                            return (
                              <Tooltip title={tip}>
                                <Tag color={color}>{label}</Tag>
                              </Tooltip>
                            );
                          }
                          return <Tag color={color}>{label}</Tag>;
                        },
                      },
                      {
                        title: '时间',
                        dataIndex: 'created_at',
                        width: 140,
                      },
                      {
                        title: '操作',
                        key: 'actions',
                        width: 180,
                        fixed: 'right' as const,
                        render: (_: unknown, record: Suggestion) => (
                          <Space size="small">
                            {record.decision === 'list' && (
                              <>
                                <Button
                                  size="small"
                                  type="primary"
                                  onClick={() => router.push('/approval')}
                                >
                                  查看审批
                                </Button>
                                {record.listing_task_id && (
                                  <Button
                                    size="small"
                                    onClick={() => router.push(`/listing-tasks/${record.listing_task_id}`)}
                                  >
                                    查看任务
                                  </Button>
                                )}
                              </>
                            )}
                          </Space>
                        ),
                      },
                    ]}
                  />
                )}
              </Spin>
            </div>
          </div>
        </Col>

        {/* Right: Platform sync status */}
        <Col xs={24} lg={7}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{
              padding: '12px 16px', borderBottom: '1px solid var(--bd)',
              display: 'flex', alignItems: 'center', gap: 8,
              fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
            }}>
              平台同步状态
              <Tag color="blue" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>沙箱</Tag>
            </div>
            <div style={{ padding: 16 }}>
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
                          alignItems: 'center', marginBottom: 8,
                        }}>
                          <Text strong>{p.platform_name}</Text>
                          <Tag color={modeColor(p.mode)}>{modeLabel(p.mode)}</Tag>
                        </div>
                        <Space direction="vertical" style={{ width: '100%' }} size={4}>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>订单</Text>
                            <Text>{p.orders_sync}</Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>商品</Text>
                            <Text>{p.products_sync}</Text>
                          </div>
                          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>费用</Text>
                            <Text>{p.fees_sync}</Text>
                          </div>
                          {p.last_sync_time && (
                            <div style={{ display: 'flex', justifyContent: 'space-between', paddingTop: 4 }}>
                              <Text type="secondary" style={{ fontSize: 11 }}>最后同步</Text>
                              <Text style={{ fontSize: 11 }}>{p.last_sync_time}</Text>
                            </div>
                          )}
                        </Space>
                      </div>
                    ))}
                  </Space>
                )}
              </Spin>
            </div>
          </div>
        </Col>
      </Row>
    </div>
  );
}
