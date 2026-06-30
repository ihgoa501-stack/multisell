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
  Tabs,
  Tag,
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
  CloseOutlined,
  MinusCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import type { TrustScore } from '@/types/api';

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
  feedback_status: string;
  feedback_note?: string;
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

const feedbackStatusLabel = (s: string): string => {
  if (s === 'adopted') return '已采纳';
  if (s === 'rejected') return '已拒绝';
  if (s === 'executed') return '已执行';
  if (s === 'execution_failed') return '执行失败';
  if (s === 'pending') return '待处理';
  return s;
};

const feedbackStatusColor = (s: string): string => {
  if (s === 'adopted') return 'green';
  if (s === 'rejected') return 'red';
  if (s === 'executed') return 'green';
  if (s === 'execution_failed') return 'red';
  return 'default';
};

// ---------- Page ----------
export default function OwnerPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [activeTab, setActiveTab] = useState<string>('queue');
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
      const res = await apiClient.get<Suggestion[]>('/v1/owner/suggestions', { limit: '100' });
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

  // Trust Scores (Agent Evaluation tab)
  const { data: trustScores, isLoading: trustScoresLoading } = useQuery({
    queryKey: ['owner-trust-scores'],
    queryFn: async () => {
      const res = await apiClient.get<TrustScore[]>('/v1/trust-scores');
      return res.data ?? [];
    },
    enabled: activeTab === 'agents',
  });

  const refreshAll = () => {
    qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
    qc.invalidateQueries({ queryKey: ['owner-pending-approvals'] });
    qc.invalidateQueries({ queryKey: ['owner-platform-sync'] });
    qc.invalidateQueries({ queryKey: ['owner-trust-scores'] });
  };

  const pendingPublishApprovals = (approvals ?? []).filter(
    (a: ApprovalRequest) => a.status === 'pending' && a.request_type === 'publish'
  );

  const listReady = useMemo(
    () => (suggestions ?? []).filter((s) => s.decision === 'list').length,
    [suggestions]
  );

  // Queue: pending suggestions
  const queueSuggestions = useMemo(() => {
    const pending = (suggestions ?? []).filter((s) => s.feedback_status === 'pending');
    if (!suggestionFilter) return pending;
    return pending.filter((s) => s.decision === suggestionFilter);
  }, [suggestions, suggestionFilter]);

  // History: processed suggestions (adopted / rejected / executed / execution_failed)
  const historySuggestions = useMemo(() => {
    return (suggestions ?? []).filter(
      (s) => s.feedback_status !== 'pending'
    ).reverse();
  }, [suggestions]);

  const filteredSuggestions = useMemo(() => {
    if (!suggestionFilter) return suggestions ?? [];
    return (suggestions ?? []).filter((s) => s.decision === suggestionFilter);
  }, [suggestions, suggestionFilter]);

  // Computed stats for Agent evaluation
  const agentStats = useMemo(() => {
    if (!suggestions) return [];
    const map = new Map<string, { total: number; adopted: number; rejected: number }>();
    for (const s of suggestions) {
      const key = s.agent_source;
      if (!map.has(key)) {
        map.set(key, { total: 0, adopted: 0, rejected: 0 });
      }
      const stats = map.get(key)!;
      stats.total++;
      if (s.feedback_status === 'adopted' || s.feedback_status === 'executed') {
        stats.adopted++;
      } else if (s.feedback_status === 'rejected') {
        stats.rejected++;
      }
    }
    return Array.from(map.entries()).map(([agent, stats]) => ({
      agent,
      total: stats.total,
      adopted: stats.adopted,
      rejected: stats.rejected,
      rate: stats.total > 0 ? ((stats.adopted / stats.total) * 100).toFixed(1) : '0.0',
    }));
  }, [suggestions]);

  // ---------- Render queue tab ----------
  const renderQueueTab = () => (
    <>
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
                {queueSuggestions.length === 0 && !suggestionsLoading ? (
                  <Empty description="暂无待处理建议" />
                ) : (
                  <Table
                    rowKey="id"
                    dataSource={queueSuggestions}
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
    </>
  );

  // ---------- Render history tab ----------
  const renderHistoryTab = () => (
    <Spin spinning={suggestionsLoading}>
      {historySuggestions.length === 0 && !suggestionsLoading ? (
        <Empty description="暂无审批历史" />
      ) : (
        <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
          <div style={{
            padding: '12px 16px', borderBottom: '1px solid var(--bd)',
            display: 'flex', alignItems: 'center', gap: 8,
            fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
          }}>
            审批历史 (已处理建议)
          </div>
          <div style={{ padding: 16 }}>
            <Table
              rowKey="id"
              dataSource={historySuggestions}
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
                  width: 100,
                  render: (v: string) => <Tag color={suggestionColor(v)}>{v}</Tag>,
                },
                {
                  title: '采纳/拒绝',
                  dataIndex: 'feedback_status',
                  width: 110,
                  render: (v: string) => (
                    <Tag color={feedbackStatusColor(v)}>
                      {v === 'adopted' && <CheckOutlined style={{ marginRight: 4 }} />}
                      {v === 'rejected' && <CloseOutlined style={{ marginRight: 4 }} />}
                      {v === 'executed' && <CheckOutlined style={{ marginRight: 4 }} />}
                      {v === 'execution_failed' && <MinusCircleOutlined style={{ marginRight: 4 }} />}
                      {feedbackStatusLabel(v)}
                    </Tag>
                  ),
                },
                {
                  title: '审批状态',
                  dataIndex: 'decision',
                  width: 100,
                  render: (v: string) => <Tag color={decisionColor(v)}>{decisionLabel(v)}</Tag>,
                },
                {
                  title: '执行状态',
                  dataIndex: 'feedback_note',
                  width: 160,
                  ellipsis: true,
                  render: (v: string | undefined) =>
                    v ? <Text type="secondary" ellipsis={{ tooltip: v }}>{v}</Text> : <Text type="secondary">-</Text>,
                },
                {
                  title: '备注',
                  dataIndex: 'feedback_note',
                  width: 200,
                  ellipsis: true,
                  render: (v: string | undefined) =>
                    v ? <Text type="secondary" ellipsis={{ tooltip: v }}>{v}</Text> : <Text type="secondary">-</Text>,
                },
                {
                  title: '时间',
                  dataIndex: 'created_at',
                  width: 140,
                },
                {
                  title: '操作',
                  key: 'actions',
                  width: 120,
                  fixed: 'right' as const,
                  render: (_: unknown, record: Suggestion) => (
                    <Space size="small">
                      {record.listing_task_id && (
                        <Button
                          size="small"
                          onClick={() => router.push(`/listing-tasks/${record.listing_task_id}`)}
                        >
                          查看任务
                        </Button>
                      )}
                    </Space>
                  ),
                },
              ]}
            />
          </div>
        </div>
      )}
    </Spin>
  );

  // ---------- Render agent evaluation tab ----------
  const renderAgentEvalTab = () => (
    <Spin spinning={trustScoresLoading}>
      {trustScores && trustScores.length > 0 ? (
        <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
          <div style={{
            padding: '12px 16px', borderBottom: '1px solid var(--bd)',
            display: 'flex', alignItems: 'center', gap: 8,
            fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
          }}>
            Agent 信任分与采纳率
          </div>
          <div style={{ padding: 16 }}>
            <Table
              rowKey="agent_id"
              dataSource={trustScores}
              size="small"
              scroll={{ x: 'max-content' }}
              pagination={false}
              columns={[
                {
                  title: 'Agent ID',
                  dataIndex: 'agent_id',
                  width: 100,
                  render: (v: string) => <Tag color="blue">{v}</Tag>,
                },
                {
                  title: '名称',
                  dataIndex: 'agent_name',
                  width: 140,
                  render: (v: string) => <Text strong>{v}</Text>,
                },
                {
                  title: '分队',
                  dataIndex: 'squad_id',
                  width: 100,
                  render: (v: string) => <Tag>{v}</Tag>,
                },
                {
                  title: '信任分',
                  dataIndex: 'trust_score',
                  width: 90,
                  render: (v: number) => (
                    <Text strong style={{ color: v >= 0.8 ? 'var(--g4)' : v >= 0.55 ? 'var(--y4)' : 'var(--r4)' }}>
                      {v.toFixed(2)}
                    </Text>
                  ),
                },
                {
                  title: '自主等级',
                  dataIndex: 'autonomy_level',
                  width: 100,
                  render: (v: string) => {
                    const color = v === 'autonomous' ? 'green' : v === 'supervised' ? 'blue' : v === 'guided' ? 'orange' : 'default';
                    return <Tag color={color}>{v}</Tag>;
                  },
                },
                {
                  title: '总建议数',
                  dataIndex: 'total_actions',
                  width: 90,
                },
                {
                  title: '已采纳',
                  dataIndex: 'adopted_actions',
                  width: 80,
                },
                {
                  title: '已拒绝',
                  dataIndex: 'rejected_actions',
                  width: 80,
                },
                {
                  title: '采纳率',
                  dataIndex: 'adoption_rate',
                  width: 90,
                  render: (v: number) => (
                    <Tag color={v >= 0.8 ? 'green' : v >= 0.5 ? 'orange' : 'red'}>
                      {(v * 100).toFixed(1)}%
                    </Tag>
                  ),
                },
              ]}
            />
          </div>
        </div>
      ) : trustScoresLoading ? null : (
        <Empty description="暂无评估数据" />
      )}
    </Spin>
  );

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
      </div>

      {/* Tab navigation */}
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          { key: 'queue', label: '决策队列' },
          { key: 'history', label: '审批历史' },
          { key: 'agents', label: 'Agent 评估' },
        ]}
        style={{ marginBottom: 0 }}
      />

      {/* Tab content */}
      <div style={{ marginTop: 16 }}>
        {activeTab === 'queue' && renderQueueTab()}
        {activeTab === 'history' && renderHistoryTab()}
        {activeTab === 'agents' && renderAgentEvalTab()}
      </div>
    </div>
  );
}
