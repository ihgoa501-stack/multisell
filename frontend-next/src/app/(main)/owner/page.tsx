'use client';

import { useMemo, useState } from 'react';
import {
  Button,
  Col,
  Empty,
  Input,
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
  AlertOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  DatabaseOutlined,
  ShoppingOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import type { SuggestionResponse } from '@/types/api';

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

// ---------- Helpers for new columns ----------

/** Color for completeness_score Tag */
const completenessColor = (v: number): string => {
  if (v >= 80) return 'green';
  if (v >= 50) return 'orange';
  return 'red';
};

/** Color for profit_margin Tag */
const profitColor = (v: number): string => {
  if (v >= 15) return 'green';
  if (v >= 0) return 'orange';
  return 'red';
};

/** Format a number as USD */
const formatUSD = (v: number): string => {
  if (v == null || isNaN(v)) return '-';
  return `$${v.toFixed(2)}`;
};

/** Icon and label config for feedback_status */
interface FeedbackStatusConfig {
  icon: React.ReactNode;
  label: string;
  color: string;
}

const feedbackStatusConfig = (status: string): FeedbackStatusConfig => {
  switch (status) {
    case 'pending':
      return { icon: <ClockCircleOutlined style={{ fontSize: 14 }} />, label: '待处理', color: 'default' };
    case 'adopted':
      return { icon: <CheckCircleOutlined style={{ color: 'var(--g4)', fontSize: 14 }} />, label: '已采纳', color: 'green' };
    case 'rejected':
      return { icon: <CloseCircleOutlined style={{ color: 'var(--r4)', fontSize: 14 }} />, label: '已拒绝', color: 'red' };
    case 'executed':
      return { icon: <ThunderboltOutlined style={{ color: 'var(--g4)', fontSize: 14 }} />, label: '已执行', color: 'green' };
    case 'execution_failed':
      return { icon: <WarningOutlined style={{ color: 'var(--r4)', fontSize: 14 }} />, label: '执行失败', color: 'red' };
    default:
      return { icon: null, label: status, color: 'default' };
  }
};

/** Approval status display when feedback_status=adopted and listing_task_id exists */
const approvalStatusLabel = (status: string | null): string => {
  if (!status) return '-';
  switch (status) {
    case 'pending': return '审批中';
    case 'approved': return '已批准';
    case 'rejected': return '已拒绝';
    default: return status;
  }
};

const approvalStatusColor = (status: string | null): string => {
  switch (status) {
    case 'pending': return 'blue';
    case 'approved': return 'green';
    case 'rejected': return 'red';
    default: return 'default';
  }
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
  const [approvalModal, setApprovalModal] = useState<SuggestionResponse | null>(null);
  const [approvalAction, setApprovalAction] = useState<'adopt' | 'reject' | null>(null);
  const [rejectNote, setRejectNote] = useState('');

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
      const res = await apiClient.get<SuggestionResponse[]>('/v1/owner/suggestions', { limit: '50' });
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

  // Provide feedback (adopt / reject) via the new API
  const provideFeedback = useMutation({
    mutationFn: async (params: { suggestionId: number; action: 'adopt' | 'reject'; note?: string }) => {
      await apiClient.post(`/v1/owner/suggestions/${params.suggestionId}/feedback`, {
        action: params.action,
        note: params.note || '',
      });
    },
    onSuccess: () => {
      message.success('操作成功');
      qc.invalidateQueries({ queryKey: ['owner-suggestions'] });
      qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
      setApprovalModal(null);
      setApprovalAction(null);
      setRejectNote('');
    },
    onError: (e: Error) => message.error(`操作失败: ${e.message}`),
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

  // Filter + sort suggestions (created_at descending)
  const filteredSuggestions = useMemo(() => {
    let data = suggestions ?? [];
    if (suggestionFilter) {
      data = data.filter((s) => s.decision === suggestionFilter);
    }
    return [...data].sort((a, b) => {
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }, [suggestions, suggestionFilter]);

  // Handlers
  const handleAdopt = (s: SuggestionResponse) => {
    setApprovalModal(s);
    setApprovalAction('adopt');
    setRejectNote('');
  };

  const handleReject = (s: SuggestionResponse) => {
    setApprovalModal(s);
    setApprovalAction('reject');
    setRejectNote('');
  };

  const confirmApproval = async () => {
    if (!approvalModal) return;
    const action = approvalAction === 'adopt' ? 'adopt' : 'reject';
    provideFeedback.mutate({
      suggestionId: approvalModal.id,
      action,
      note: action === 'reject' ? rejectNote : undefined,
    });
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
              Agent 上架建议 — 决策队列
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
                    columns={[
                      {
                        title: '商品',
                        dataIndex: 'product_title',
                        width: 140,
                        ellipsis: true,
                        render: (v: string) => <Text strong>{v || '-'}</Text>,
                      },
                      {
                        title: '完整度',
                        dataIndex: 'completeness_score',
                        width: 90,
                        sorter: (a, b) => a.completeness_score - b.completeness_score,
                        render: (v: number) => (
                          <Tag color={completenessColor(v)}>{v}%</Tag>
                        ),
                      },
                      {
                        title: '利润率',
                        dataIndex: 'profit_margin',
                        width: 90,
                        sorter: (a, b) => a.profit_margin - b.profit_margin,
                        render: (v: number) => (
                          <Tag color={profitColor(v)}>{v}%</Tag>
                        ),
                      },
                      {
                        title: '预计利润',
                        dataIndex: 'estimated_profit',
                        width: 100,
                        sorter: (a, b) => a.estimated_profit - b.estimated_profit,
                        render: (v: number) => (
                          <Text style={{ fontVariantNumeric: 'tabular-nums' }}>{formatUSD(v)}</Text>
                        ),
                      },
                      {
                        title: '评估结论',
                        dataIndex: 'decision',
                        width: 100,
                        render: (v: string) => <Tag color={decisionColor(v)}>{decisionLabel(v)}</Tag>,
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
                        title: '审批/任务',
                        key: 'approval_status',
                        width: 100,
                        render: (_: unknown, record: SuggestionResponse) => {
                          if (!record.listing_task_id) return <Text type="secondary">-</Text>;
                          if (record.feedback_status === 'adopted') {
                            return (
                              <Tag color={approvalStatusColor(record.approval_status)}>
                                {approvalStatusLabel(record.approval_status)}
                              </Tag>
                            );
                          }
                          if (record.feedback_status === 'executed') {
                            return <Tag color="green">已执行</Tag>;
                          }
                          if (record.feedback_status === 'execution_failed') {
                            return <Tag color="red">执行失败</Tag>;
                          }
                          return <Text type="secondary">-</Text>;
                        },
                      },
                      {
                        title: '反馈状态',
                        dataIndex: 'feedback_status',
                        width: 110,
                        render: (v: string) => {
                          const cfg = feedbackStatusConfig(v);
                          return (
                            <Space size={4}>
                              {cfg.icon}
                              <Text style={{ fontSize: '0.75rem' }}>{cfg.label}</Text>
                            </Space>
                          );
                        },
                      },
                      {
                        title: '理由',
                        dataIndex: 'reason',
                        width: 260,
                        ellipsis: true,
                        render: (v: string) => <Text type="secondary" ellipsis={{ tooltip: v }}>{v}</Text>,
                      },
                      {
                        title: '时间',
                        dataIndex: 'created_at',
                        width: 140,
                        sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
                        defaultSortOrder: 'descend',
                      },
                      {
                        title: '操作',
                        key: 'actions',
                        width: 180,
                        fixed: 'right' as const,
                        render: (_: unknown, record: SuggestionResponse) => (
                          <Space size="small">
                            <Button
                              size="small"
                              type="primary"
                              icon={<CheckOutlined />}
                              disabled={record.decision !== 'list' || record.feedback_status !== 'pending'}
                              onClick={() => handleAdopt(record)}
                            >
                              采纳
                            </Button>
                            <Button
                              size="small"
                              danger
                              icon={<CloseOutlined />}
                              disabled={record.feedback_status !== 'pending'}
                              onClick={() => handleReject(record)}
                            >
                              拒绝
                            </Button>
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

      {/* Approval confirmation modal */}
      <Modal
        title={approvalAction === 'adopt' ? '确认采纳建议' : '确认拒绝建议'}
        open={!!approvalModal}
        onCancel={() => { setApprovalModal(null); setApprovalAction(null); setRejectNote(''); }}
        onOk={confirmApproval}
        confirmLoading={provideFeedback.isPending}
        okText={approvalAction === 'adopt' ? '采纳' : '拒绝'}
        cancelText="取消"
        okButtonProps={{ danger: approvalAction === 'reject' }}
      >
        <p>
          商品："{approvalModal?.product_title || `ID:${approvalModal?.product_id}`}"
        </p>
        {approvalAction === 'adopt' ? (
          <div style={{ background: 'var(--g1)', padding: 12, borderRadius: 6, marginTop: 8, border: '1px solid var(--g3)' }}>
            <Text type="secondary" style={{ fontSize: '0.85rem', lineHeight: 1.6 }}>
              将会创建审批请求 → 审批通过后 → Agent 将执行上架
            </Text>
          </div>
        ) : (
          <div style={{ marginTop: 12 }}>
            <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>拒绝原因（选填）：</Text>
            <Input.TextArea
              rows={3}
              value={rejectNote}
              onChange={(e) => setRejectNote(e.target.value)}
              placeholder="请输入拒绝原因..."
            />
          </div>
        )}
      </Modal>
    </div>
  );
}
