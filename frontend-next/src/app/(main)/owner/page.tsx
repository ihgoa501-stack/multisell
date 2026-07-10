'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Button,
  Col,
  Empty,
  Input,
  message,
  Row,
  Select,
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
  CloseOutlined,
  AlertOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  DatabaseOutlined,
  ShoppingOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  ArrowRightOutlined,
  SearchOutlined,
  BulbOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';
import HighRiskConfirmDialog from '@/components/ui/HighRiskConfirmDialog';

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

interface DecisionQueueItem {
  id: number;
  product_id: number;
  product_title: string;
  listing_task_id: number | null;
  agent_source: string;
  suggestion: string;
  decision: string;
  reason: string;
  confidence: number;
  risk_flags: string;
  risk_level: string;
  created_at: string;
  task_status?: string;
  task_error?: string;
  approval_status?: string;
  agent_feedback_status?: string | null;
  blocking_reasons?: string[];
  status?: string;
  execution_mode?: number;
  expected_outcome?: string;
  can_approve?: boolean;
  display_status?: string;
  target_sale_price?: number;
  completeness_score?: number;
  estimated_profit?: number;
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

interface ApprovalRequestResponse {
  id: number;
}

/** Response shape from GET /v1/owner/suggestions */
interface SuggestionResponse {
  id: number;
  product_id: number;
  product_title: string;
  listing_task_id: number | null;
  decision: string;
  reason: string;
  confidence: number;
  risk_flags: string;
  risk_level: string;
  created_at: string;
  task_status: string;
  approval_status: string;
  feedback_status: string | null;
  completeness_score: number;
  profit_margin: number;
  estimated_profit: number;
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

const taskStatusColor = (s: string | undefined): string => {
  if (s === 'blocked') return 'red';
  if (s === 'pending') return 'orange';
  if (s === 'executing') return 'blue';
  if (s === 'completed') return 'green';
  return 'default';
};

const taskStatusLabel = (s: string | undefined): string => {
  if (s === 'blocked') return '已阻断';
  if (s === 'pending') return '待处理';
  if (s === 'executing') return '执行中';
  if (s === 'completed') return '已完成';
  return s ?? '-';
};

const approvalStatusColor = (s: string | undefined): string => {
  if (s === 'approved') return 'green';
  if (s === 'rejected') return 'red';
  if (s === 'pending') return 'orange';
  return 'default';
};

const approvalStatusLabel = (s: string | undefined): string => {
  if (s === 'approved') return '已批准';
  if (s === 'rejected') return '已拒绝';
  if (s === 'pending') return '待审批';
  return s ?? '-';
};

const feedbackStatusColor = (s: string | null | undefined): string => {
  if (s === 'accepted') return 'green';
  if (s === 'rejected') return 'red';
  return 'default';
};

const feedbackStatusLabel = (s: string | null | undefined): string => {
  if (s === 'accepted') return '已接受';
  if (s === 'rejected') return '已拒绝';
  return s ?? '-';
};

// ---------- Page ----------
export default function OwnerPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [suggestionFilter, setSuggestionFilter] = useState<string>('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [sortBy, setSortBy] = useState<string>('');
  const [searchText, setSearchText] = useState<string>('');
  const [approvalModal, setApprovalModal] = useState<DecisionQueueItem | null>(null);
  const [approvalAction, setApprovalAction] = useState<'approve' | 'reject' | null>(null);

  // Risk summary
  const { data: riskSummary } = useQuery({
    queryKey: ['owner-risk-summary'],
    queryFn: async () => {
      const res = await apiClient.get<RiskSummary>('/v1/owner/risk-summary');
      return res.data;
    },
  });

  // Decision queue (maps from /v1/owner/suggestions)
  const { data: decisions, isLoading: decisionsLoading } = useQuery({
    queryKey: ['owner-decision-queue'],
    queryFn: async () => {
      const res = await apiClient.get<SuggestionResponse[]>('/v1/owner/suggestions', { limit: '50' });
      const items = res.data ?? [];
      return items.map((s): DecisionQueueItem => ({
        id: s.id,
        product_id: s.product_id,
        product_title: s.product_title,
        listing_task_id: s.listing_task_id,
        agent_source: '',
        suggestion: '',
        decision: s.decision,
        reason: s.reason,
        confidence: s.confidence,
        risk_flags: s.risk_flags,
        risk_level: s.risk_level,
        created_at: s.created_at,
        task_status: s.task_status,
        task_error: '',
        approval_status: s.approval_status,
        agent_feedback_status: s.feedback_status,
        blocking_reasons: [],
        display_status: s.task_status || '',
        execution_mode: 0,
        expected_outcome: '',
        can_approve: s.task_status === 'blocked' || s.task_status === 'pending_approval',
      }));
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

  // ---- Approval flow mutations ----

  const approveFlow = useMutation({
    mutationFn: async (params: { productId: number; taskId: number; reason: string; executionMode?: number }) => {
      const operator = getCurrentOperator();

      // 1. Create approval request
      const approvalResult = await apiClient.post<ApprovalRequestResponse>('/v1/approval', {
        product_id: params.productId,
        request_type: 'listing_task',
        requester: operator,
        reason: params.reason,
      });

      const approvalId = approvalResult.data?.id;
      if (!approvalId) {
        throw new Error('创建审批记录失败');
      }

      // 2. Review (approve) the created approval
      await apiClient.put(`/v1/approval/${approvalId}/review`, {
        action: 'approve',
        reviewer: operator,
        review_note: 'Owner approved via decision queue',
      });

      // 3. Submit agent feedback as accepted
      await apiClient.post(`/v1/listing-task/${params.taskId}/feedback`, {
        status: 'accepted',
        note: 'Owner approved listing task',
      });

      // 4. Navigate to listing task detail page for execution
      router.push(`/listing-tasks/${params.taskId}`);
    },
    onSuccess: (_, variables) => {
      if (variables.executionMode === 1) {
        message.success('已批准上架，沙盒环境调度并执行成功');
      } else {
        message.success('已批准上架并开始执行');
      }
      setApprovalModal(null);
      setApprovalAction(null);
      qc.invalidateQueries({ queryKey: ['owner-decision-queue'] });
      qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectFlow = useMutation({
    mutationFn: async (params: { productId: number; taskId: number; reason: string }) => {
      const operator = getCurrentOperator();

      // 1. Create approval request
      const approvalResult = await apiClient.post<ApprovalRequestResponse>('/v1/approval', {
        product_id: params.productId,
        request_type: 'listing_task',
        requester: operator,
        reason: params.reason,
      });

      const approvalId = approvalResult.data?.id;

      // 2. Review (reject) the created approval
      if (approvalId) {
        await apiClient.put(`/v1/approval/${approvalId}/review`, {
          action: 'reject',
          reviewer: operator,
          review_note: 'Owner rejected via decision queue',
        });
      }

      // 3. Submit agent feedback as rejected
      await apiClient.post(`/v1/listing-task/${params.taskId}/feedback`, {
        status: 'rejected',
        note: 'Owner rejected listing task',
      });
    },
    onSuccess: () => {
      message.success('已拒绝上架');
      setApprovalModal(null);
      setApprovalAction(null);
      qc.invalidateQueries({ queryKey: ['owner-decision-queue'] });
      qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const handleRefresh = () => {
    qc.invalidateQueries({ queryKey: ['owner-risk-summary'] });
    qc.invalidateQueries({ queryKey: ['owner-decision-queue'] });
    qc.invalidateQueries({ queryKey: ['owner-platform-sync'] });
  };

  const filteredDecisions = useMemo(() => {
    let result = decisions ?? [];
    if (suggestionFilter) result = result.filter((s) => s.decision === suggestionFilter);
    if (statusFilter) result = result.filter((s) => s.status === statusFilter || s.approval_status === statusFilter || s.task_status === statusFilter);
    if (searchText) result = result.filter((s) => s.product_title?.toLowerCase().includes(searchText.toLowerCase()));
    if (sortBy === 'confidence') result = [...result].sort((a, b) => (b.confidence ?? 0) - (a.confidence ?? 0));
    return result;
  }, [decisions, suggestionFilter, statusFilter, searchText, sortBy]);

  const handleApprove = (s: DecisionQueueItem) => {
    setApprovalModal(s);
    setApprovalAction('approve');
  };

  const handleReject = (s: DecisionQueueItem) => {
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
      approveFlow.mutate({
        productId: approvalModal.product_id,
        taskId,
        reason: approvalModal.reason,
        executionMode: approvalModal.execution_mode,
      });
    } else {
      rejectFlow.mutate({
        productId: approvalModal.product_id,
        taskId,
        reason: approvalModal.reason,
      });
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
          marginBottom: 16,
        }}
      >
        <h1 style={{
          fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)',
          color: 'var(--t1)', margin: 0,
        }}>
          Owner 经营总控台
        </h1>
        <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
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
          { label: '候选商品', href: '/candidates', icon: <ShoppingOutlined />, color: 'var(--i4)' },
          { label: '完整性评估', icon: <SearchOutlined />, color: 'var(--c4)' },
          { label: '上架建议', icon: <BulbOutlined />, color: 'var(--g4)' },
          { label: 'Owner 审批', href: '/approval', icon: <CheckCircleOutlined />, color: 'var(--g4)' },
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
                <span style={{ fontSize: '0.75rem', display: 'flex' }}>{step.icon}</span>
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
                <span style={{ fontSize: '0.75rem', display: 'flex' }}>{step.icon}</span>
                {step.label}
              </span>
            )}
          </div>
        ))}
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
        {/* Left: Decision queue table */}
        <Col xs={24} lg={17}>
          <div style={{ background: 'var(--s1)', border: '1px solid var(--bd)', borderRadius: 8 }}>
            <div style={{
              padding: '12px 16px', borderBottom: '1px solid var(--bd)',
              display: 'flex', alignItems: 'center', gap: 8,
              fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.875rem', color: 'var(--t1)',
            }}>
              Agent 决策队列
              <Tag color="orange" style={{ fontSize: '0.6rem', lineHeight: '1.4' }}>Mock</Tag>
            </div>
            <div style={{ padding: 16 }}>
              <Space style={{ marginBottom: 8, flexWrap: 'wrap' }}>
                <Button size="small" onClick={() => setSuggestionFilter('')} type={suggestionFilter === '' ? 'primary' : 'default'}>全部</Button>
                <Button size="small" type={suggestionFilter === 'list' ? 'primary' : 'default'} style={suggestionFilter === 'list' ? { backgroundColor: 'var(--g4)', borderColor: 'var(--g4)' } : {}} onClick={() => setSuggestionFilter('list')}>推荐上架</Button>
                <Button size="small" type={suggestionFilter === 'cautious' ? 'primary' : 'default'} style={suggestionFilter === 'cautious' ? { backgroundColor: 'var(--y4)', borderColor: 'var(--y4)' } : {}} onClick={() => setSuggestionFilter('cautious')}>谨慎</Button>
                <Button size="small" danger type={suggestionFilter === 'skip' ? 'primary' : 'default'} onClick={() => setSuggestionFilter('skip')}>不建议</Button>
                <Select allowClear placeholder="按状态筛选" style={{ width: 140 }} value={statusFilter || undefined} onChange={(v) => setStatusFilter(v || '')}
                  options={[
                    { value: 'waiting_data', label: '等待数据' },
                    { value: 'ready_for_decision', label: '待决策' },
                    { value: 'pending_approval', label: '待审批' },
                    { value: 'executing', label: '执行中' },
                    { value: 'completed', label: '已完成' },
                    { value: 'failed', label: '失败' },
                  ]}
                />
                <Select allowClear placeholder="排序" style={{ width: 120 }} value={sortBy || undefined} onChange={(v) => setSortBy(v || '')}
                  options={[{ value: 'confidence', label: '置信度' }, { value: 'created_at', label: '创建时间' }]}
                />
                <Input.Search allowClear placeholder="搜索商品标题" style={{ width: 200 }} value={searchText} onChange={(e) => setSearchText(e.target.value)} onSearch={(v) => setSearchText(v)} />
              </Space>
              {/* Summary stats */}
              <div style={{ display: 'flex', gap: 24, marginBottom: 12, padding: '8px 0', borderBottom: '1px solid var(--bd)' }}>
                <span><Tag>等待数据</Tag> {(decisions ?? []).filter(d => d.status === 'waiting_data' || (!d.task_status && !d.approval_status)).length}</span>
                <span><Tag color="blue">待决策</Tag> {(decisions ?? []).filter(d => d.status === 'ready_for_decision' || (d.decision && !d.approval_status)).length}</span>
                <span><Tag color="orange">待审批</Tag> {(decisions ?? []).filter(d => d.approval_status === 'pending' || d.status === 'pending_approval').length}</span>
                <span><Tag color="green">已完成</Tag> {(decisions ?? []).filter(d => d.task_status === 'completed' || d.status === 'completed').length}</span>
              </div>
              <Spin spinning={decisionsLoading}>
                {filteredDecisions.length === 0 && !decisionsLoading ? (
                  <div style={{ textAlign: 'center', padding: '24px 0' }}>
                    <Empty description="暂无决策数据" />
                    <Button type="link" icon={<BulbOutlined />} onClick={() => router.push('/candidates')}>
                      前往候选商品生成决策
                    </Button>
                  </div>
                ) : (
                  <Table
                    rowKey="id"
                    dataSource={filteredDecisions}
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
                        title: '建议',
                        dataIndex: 'suggestion',
                        width: 100,
                        render: (v: string) => <Tag color={suggestionColor(v)}>{v}</Tag>,
                      },
                      {
                        title: '评估结论',
                        dataIndex: 'decision',
                        width: 90,
                        render: (v: string) => <Tag color={decisionColor(v)}>{decisionLabel(v)}</Tag>,
                      },
                      {
                        title: '理由',
                        dataIndex: 'reason',
                        width: 200,
                        ellipsis: true,
                        render: (v: string) => <Text type="secondary" ellipsis={{ tooltip: v }}>{v}</Text>,
                      },
                      {
                        title: '风险',
                        dataIndex: 'risk_level',
                        width: 70,
                        render: (v: string) => <Tag color={v === 'high' ? 'red' : v === 'medium' ? 'orange' : 'green'}>{v || '-'}</Tag>,
                      },
                      {
                        title: '状态',
                        dataIndex: 'display_status',
                        width: 90,
                        render: (v: string) => {
                          const sc: Record<string, string> = {'waiting_data':'default','ready_for_decision':'green','pending_approval':'gold','executing':'blue','completed':'success','failed':'error'};
                          const sl: Record<string, string> = {'waiting_data':'等待数据','ready_for_decision':'待决策','pending_approval':'审批中','executing':'执行中','completed':'已完成','failed':'失败'};
                          return <Tag color={sc[v] || 'default'}>{sl[v] || v || '-'}</Tag>;
                        },
                      },
                      {
                        title: '置信度',
                        dataIndex: 'confidence',
                        width: 70,
                        render: (v: number) => (
                          <Tag color={confidenceColor(v)}>{(v * 100).toFixed(0)}%</Tag>
                        ),
                      },
                      {
                        title: '风险',
                        dataIndex: 'risk_level',
                        width: 60,
                        render: (v: string) => <Tag color={v === 'high' ? 'red' : v === 'medium' ? 'orange' : 'green'}>{v === 'high' ? '高' : v === 'medium' ? '中' : '低'}</Tag>,
                      },
                      {
                        title: '模式',
                        dataIndex: 'execution_mode',
                        width: 65,
                        render: (v: number) => {
                          const labels: Record<number, string> = { 0: '模拟', 1: '沙箱', 2: '需审批', 3: '生产' };
                          const colors: Record<number, string> = { 0: 'default', 1: 'orange', 2: 'purple', 3: 'red' };
                          return <Tag color={colors[v] ?? 'default'}>{labels[v] ?? '未知'}</Tag>;
                        },
                      },
                      {
                        title: '任务状态',
                        dataIndex: 'task_status',
                        width: 180,
                        render: (v: string | undefined, record: DecisionQueueItem) => {
                          const isSandbox = record.execution_mode === 1;
                          if (isSandbox && v === 'executing') {
                            return (
                              <Space>
                                <Spin size="small" />
                                <span style={{ fontSize: 12, color: 'var(--t3)' }}>
                                  正在调度沙盒环境并执行...
                                </span>
                              </Space>
                            );
                          }
                          if (isSandbox && (v === 'completed' || v === 'failed')) {
                            return (
                              <Space direction="vertical" size={4} style={{ display: 'flex' }}>
                                <Space>
                                  <Tag color={taskStatusColor(v)}>{taskStatusLabel(v)}</Tag>
                                  <Button
                                    size="small"
                                    type="link"
                                    style={{ padding: 0, height: 'auto', fontSize: 12 }}
                                    href={`/tmp/reports/pr-${record.id}/playwright-report`}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                  >
                                    下载报告
                                  </Button>
                                </Space>
                                {record.task_error && (
                                  <Text type="danger" style={{ fontSize: 11 }} ellipsis={{ tooltip: record.task_error }}>
                                    {record.task_error}
                                  </Text>
                                )}
                              </Space>
                            );
                          }
                          return <Tag color={taskStatusColor(v)}>{taskStatusLabel(v)}</Tag>;
                        },
                      },
                      {
                        title: '审批状态',
                        dataIndex: 'approval_status',
                        width: 90,
                        render: (v: string | undefined) => (
                          <Tag color={approvalStatusColor(v)}>{approvalStatusLabel(v)}</Tag>
                        ),
                      },
                      {
                        title: '阻断原因',
                        dataIndex: 'blocking_reasons',
                        width: 140,
                        render: (reasons: string[] | undefined) => {
                          if (!reasons || reasons.length === 0) return <Text type="secondary">-</Text>;
                          const display = reasons.slice(0, 2);
                          const rest = reasons.slice(2);
                          return (
                            <Tooltip
                              title={
                                <ul style={{ margin: 0, paddingLeft: 16 }}>
                                  {reasons.map((r, i) => <li key={i}>{r}</li>)}
                                </ul>
                              }
                            >
                              <Space size={4} wrap>
                                {display.map((r, i) => (
                                  <Tag key={i} color="red" style={{ margin: 0 }}>{r}</Tag>
                                ))}
                                {rest.length > 0 && (
                                  <Text type="secondary">+{rest.length}</Text>
                                )}
                              </Space>
                            </Tooltip>
                          );
                        },
                      },
                      {
                        title: 'Agent反馈',
                        dataIndex: 'agent_feedback_status',
                        width: 90,
                        render: (v: string | null | undefined) => {
                          if (!v) return <Text type="secondary">-</Text>;
                          return (
                            <Tag color={feedbackStatusColor(v)}>
                              {feedbackStatusLabel(v)}
                            </Tag>
                          );
                        },
                      },
                      {
                        title: '时间',
                        dataIndex: 'created_at',
                        width: 130,
                      },
                      {
                        title: '操作',
                        key: 'actions',
                        width: 130,
                        fixed: 'right' as const,
                        render: (_: unknown, record: DecisionQueueItem) => (
                          <Space size="small">
                            <Tooltip title={!record.can_approve ? '当前状态不可操作' : '批准后将创建审批、记录Agent反馈并触发执行上架任务'}>
                              <Button
                                size="small"
                                type="primary"
                                icon={<CheckOutlined />}
                                disabled={record.decision !== 'list' || !record.can_approve}
                                onClick={() => handleApprove(record)}
                              >
                                批准
                              </Button>
                            </Tooltip>
                            <Tooltip title={!record.can_approve ? '当前状态不可操作' : '拒绝后将创建审批记录并反馈Agent建议被拒绝'}>
                              <Button
                                size="small"
                                danger
                                icon={<CloseOutlined />}
                                disabled={!record.can_approve}
                                onClick={() => handleReject(record)}
                              >
                                拒绝
                              </Button>
                            </Tooltip>
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
                  <Empty description="暂无数据">
                    <Button type="link" onClick={() => router.push('/platform-integrations')}>
                      前往平台集成配置
                    </Button>
                  </Empty>
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

      {/* Approval confirmation dialog */}
      <HighRiskConfirmDialog
        open={!!approvalModal}
        actionName={approvalAction === 'approve' ? '批准上架' : '拒绝上架'}
        riskLevel={approvalModal?.risk_level === 'high' ? 'high' : approvalModal?.risk_level === 'medium' ? 'medium' : 'medium'}
        detail={approvalModal ? {
          targetLabel: approvalModal.product_title || `ID:${approvalModal.product_id}`,
          afterValue: approvalAction === 'approve' ? '创建审批 → 执行上架任务' : '不创建上架任务',
        } : undefined}
        environmentMode={
          approvalModal?.execution_mode === 1
            ? 'sandbox'
            : approvalModal?.execution_mode === 0
            ? 'dry_run'
            : 'production'
        }
        expectedConsequence={
          approvalAction === 'approve'
            ? '将创建审批记录、触发上架任务执行，商品将在 Ozon 平台可见'
            : '不创建上架任务，商品保留在候选列表中'
        }
        auditDestination="操作已记录至 operation_log 表，可追溯至操作人、时间、原因"
        rollbackNote="上架后无法自动回滚，如需下线请在对应平台手动操作"
        confirmLoading={approveFlow.isPending || rejectFlow.isPending}
        confirmText={approvalAction === 'approve' ? '批准上架' : '拒绝上架'}
        showReason
        reasonPlaceholder="补充说明（选填）"
        onConfirm={() => confirmApproval()}
        onCancel={() => { setApprovalModal(null); setApprovalAction(null); }}
      />
    </div>
  );
}
