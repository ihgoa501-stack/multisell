'use client';

import { Card, Collapse, Empty, Row, Col, Statistic, Table, Tag, Tabs, Alert } from 'antd';
import {
  CheckCircleFilled, ExclamationCircleFilled,
  ExclamationCircleOutlined, FallOutlined,
  RiseOutlined, ShoppingCartOutlined, WarningFilled,
} from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

/* ─── Types ─── */

interface DailyBrief {
  today_profit: number;
  today_revenue: number;
  month_profit: number;
  month_revenue: number;
  month_cost: number;
  open_exception_count: number;
  low_stock_count: number;
  out_of_stock_count: number;
  negative_margin_count: number;
  pending_support_count: number;
  pending_aftersales_count: number;
  low_stock_skus: LowStockSku[];
  negative_margin_skus: NegativeMarginSku[];
  recent_exceptions: RecentException[];
  urgent_conversations: UrgentConversation[];
  platform_connections: PlatformConnection[];
}

interface LowStockSku {
  sku_id: number;
  product_id: number;
  code: string;
  spec_desc: string;
  stock: number;
  warning_stock: number;
}

interface NegativeMarginSku {
  product_id: number;
  sku_code: string;
  title: string;
  profit_margin: number;
  estimated_profit: number;
}

interface RecentException {
  id: number;
  severity: string;
  source_module: string;
  message: string;
  status: string;
  created_at: string;
}

interface UrgentConversation {
  id: number;
  customer_name: string;
  subject: string;
  priority: string;
  platform: string;
  last_message_at?: string;
}


/* ─── Overview (M6 Owner KPIs) ─── */

interface DashboardOverview {
  today_sales: number;
  pending_approvals: number;
  agent_suggestions: number;
  exception_open_count: number;
  recent_alerts: {
    id: number;
    severity: string;
    title: string;
    created_at: string;
  }[];
}

interface PlatformConnection {
  platform_id: number;
  platform_code: string;
  platform_name: string;
  store_name: string;
  status: string;
  sync_status: string;
  last_sync_at?: string;
  last_error?: string;
}

/* ─── Helpers ─── */

function fmt(n: number): string {
  return `¥${n.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`;
}

function profitColor(n: number): string {
  return n >= 0 ? 'var(--g4)' : 'var(--r4)';
}

/* ─── Sections ─── */

function ProfitCard({ data }: { data: DailyBrief }) {
  return (
    <Card style={{ borderRadius: 'var(--r4)', background: 'var(--s2)', border: '1px solid var(--bd)' }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 12, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 'clamp(1.8rem,5vw,2.6rem)', fontWeight: 700, fontFamily: 'var(--ds)', color: profitColor(data.today_profit) }}>
          {data.today_profit >= 0 ? <RiseOutlined /> : <FallOutlined />}
          {' '}{fmt(data.today_profit)}
        </span>
        <span style={{ color: 'var(--t3)', fontSize: 'var(--text-small)' }}>今日利润</span>
      </div>

      <Row gutter={[16, 8]} style={{ marginTop: 'var(--space-lg)' }}>
        <Col xs={12} sm={6}>
          <div style={{ fontSize: 'var(--text-small)', color: 'var(--t3)' }}>今日收入</div>
          <div style={{ fontWeight: 600, color: 'var(--t1)' }}>{fmt(data.today_revenue)}</div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ fontSize: 'var(--text-small)', color: 'var(--t3)' }}>本月利润</div>
          <div style={{ fontWeight: 600, color: profitColor(data.month_profit) }}>{fmt(data.month_profit)}</div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ fontSize: 'var(--text-small)', color: 'var(--t3)' }}>本月收入</div>
          <div style={{ fontWeight: 600, color: 'var(--t1)' }}>{fmt(data.month_revenue)}</div>
        </Col>
        <Col xs={12} sm={6}>
          <div style={{ fontSize: 'var(--text-small)', color: 'var(--t3)' }}>本月成本</div>
          <div style={{ fontWeight: 600, color: 'var(--r4)' }}>{fmt(data.month_cost)}</div>
        </Col>
      </Row>
    </Card>
  );
}

function ExceptionAlerts({ data }: { data: DailyBrief }) {
  const issues: Array<{ key: string; label: string; count: number; severity: 'critical' | 'warning'; href: string }> = [];

  if (data.low_stock_count > 0 || data.out_of_stock_count > 0) {
    issues.push({ key: 'stock', label: `低库存 ${data.low_stock_count} 条 / 缺货 ${data.out_of_stock_count} 条`, count: data.low_stock_count + data.out_of_stock_count, severity: data.out_of_stock_count > 0 ? 'critical' : 'warning', href: '/inventory' });
  }
  if (data.negative_margin_count > 0) {
    issues.push({ key: 'margin', label: `亏损 SKU ${data.negative_margin_count} 条`, count: data.negative_margin_count, severity: 'critical', href: '/profit' });
  }
  if (data.open_exception_count > 0) {
    issues.push({ key: 'exception', label: `异常 ${data.open_exception_count} 条`, count: data.open_exception_count, severity: data.open_exception_count > 5 ? 'critical' : 'warning', href: '/exceptions' });
  }
  if (data.pending_support_count > 0 || data.pending_aftersales_count > 0) {
    issues.push({ key: 'support', label: `待回复客服 ${data.pending_support_count} 条 / 售后 ${data.pending_aftersales_count} 条`, count: data.pending_support_count + data.pending_aftersales_count, severity: 'warning', href: '/support' });
  }

  if (issues.length === 0) {
    return (
      <Card style={{ borderRadius: 'var(--r4)', background: 'var(--s2)', border: '1px solid var(--bd)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--g4)', fontSize: 'var(--text-body)' }}>
          <CheckCircleFilled /> 一切正常
        </div>
      </Card>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {issues.map((issue) => (
        <Alert
          key={issue.key}
          type={issue.severity === 'critical' ? 'error' : 'warning'}
          showIcon
          icon={issue.severity === 'critical' ? <ExclamationCircleFilled /> : <WarningFilled />}
          message={
            <a href={issue.href} style={{ color: 'inherit', textDecoration: 'underline' }}>
              {issue.label}
            </a>
          }
          style={{ borderRadius: 'var(--r3)', marginBottom: 0 }}
        />
      ))}
    </div>
  );
}

const lowStockColumns = [
  { title: 'SKU 编码', dataIndex: 'code', key: 'code', width: 140 },
  { title: '规格', dataIndex: 'spec_desc', key: 'spec_desc', ellipsis: true },
  { title: '库存', dataIndex: 'stock', key: 'stock', width: 80, render: (v: number) => <span style={{ color: v <= 0 ? 'var(--r4)' : 'var(--y4)', fontWeight: 600 }}>{v}</span> },
  { title: '预警线', dataIndex: 'warning_stock', key: 'warning_stock', width: 80 },
  { title: '状态', key: 'status', width: 80, render: (_: unknown, r: LowStockSku) => r.stock <= 0 ? <Tag color="red">缺货</Tag> : <Tag color="orange">偏低</Tag> },
];

const negativeMarginColumns = [
  { title: 'SKU', dataIndex: 'sku_code', key: 'sku_code', width: 140 },
  { title: '商品', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '利润率', dataIndex: 'profit_margin', key: 'profit_margin', width: 90, render: (v: number) => <span style={{ color: 'var(--r4)', fontWeight: 600 }}>{v.toFixed(2)}%</span>, sorter: (a: NegativeMarginSku, b: NegativeMarginSku) => a.profit_margin - b.profit_margin },
  { title: '预计利润', dataIndex: 'estimated_profit', key: 'estimated_profit', width: 110, render: (v: number) => fmt(v) },
];

function DetailsTabs({ data }: { data: DailyBrief }) {
  const tabItems = [
    {
      key: 'stock',
      label: <span><ShoppingCartOutlined /> 库存告警</span>,
      children: data.low_stock_skus.length === 0
        ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: 'var(--g4)' }}><CheckCircleFilled /> 库存正常</span>} />
        : <Table rowKey="sku_id" dataSource={data.low_stock_skus} columns={lowStockColumns} pagination={false} size="small" style={{ marginTop: 'var(--space-sm)' }} />,
    },
    {
      key: 'margin',
      label: <span><FallOutlined /> 亏损 SKU</span>,
      children: data.negative_margin_skus.length === 0
        ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: 'var(--g4)' }}><CheckCircleFilled /> 无亏损 SKU</span>} />
        : <Table rowKey="sku_code" dataSource={data.negative_margin_skus} columns={negativeMarginColumns} pagination={false} size="small" style={{ marginTop: 'var(--space-sm)' }} />,
    },
    {
      key: 'support',
      label: <span><ExclamationCircleOutlined /> 紧急客服</span>,
      children: data.urgent_conversations.length === 0
        ? <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={<span style={{ color: 'var(--g4)' }}><CheckCircleFilled /> 无待处理会话</span>} />
        : data.urgent_conversations.map((c) => (
          <Card key={c.id} size="small" style={{ marginBottom: 8, borderLeft: c.priority === 'high' || c.priority === 'urgent' ? '3px solid var(--r4)' : '3px solid var(--y4)' }}>
            <Row justify="space-between" align="middle">
              <Col>
                <div style={{ fontWeight: 600, fontSize: 'var(--text-body)' }}>{c.customer_name}</div>
                <div style={{ color: 'var(--t3)', fontSize: 'var(--text-small)' }}>{c.subject}</div>
              </Col>
              <Col>
                <Tag color={c.priority === 'high' || c.priority === 'urgent' ? 'red' : 'orange'}>{c.priority}</Tag>
                <Tag>{c.platform}</Tag>
              </Col>
            </Row>
            {c.last_message_at && <div style={{ color: 'var(--t4)', fontSize: 'var(--text-label)', marginTop: 4 }}>{new Date(c.last_message_at).toLocaleString('zh-CN')}</div>}
          </Card>
        )),
    },
  ];

  return (
    <Card style={{ borderRadius: 'var(--r4)', background: 'var(--s2)', border: '1px solid var(--bd)' }}>
      <Tabs items={tabItems} />
    </Card>
  );
}

const platformColumns = [
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name', width: 100, render: (v: string) => <Tag>{v}</Tag> },
  { title: '店铺', dataIndex: 'store_name', key: 'store_name', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (v: string) => v === 'active' ? <Tag color="green">已连接</Tag> : <Tag color="red">断开</Tag> },
  { title: '同步', dataIndex: 'sync_status', key: 'sync_status', width: 80, render: (v: string) => <Tag>{v}</Tag> },
  { title: '上次同步', dataIndex: 'last_sync_at', key: 'last_sync_at', width: 140, render: (v?: string) => v ? v.slice(0, 16).replace('T', ' ') : '-' },
  { title: '错误', dataIndex: 'last_error', key: 'last_error', ellipsis: true, render: (v?: string) => v ? <span style={{ color: 'var(--r4)' }}>{v}</span> : '-' },
];

/* ─── Loading skeleton ─── */

function BriefSkeleton() {
  return (
    <div style={{ padding: 'var(--space-xl)', display: 'flex', flexDirection: 'column', gap: 'var(--space-lg)' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', color: 'var(--t1)' }}>Daily Brief</h1>
      <div style={{ height: 120, background: 'var(--s2)', borderRadius: 'var(--r4)' }} />
      <div style={{ height: 80, background: 'var(--s2)', borderRadius: 'var(--r3)' }} />
      <div style={{ height: 300, background: 'var(--s2)', borderRadius: 'var(--r4)' }} />
    </div>
  );
}


/* ─── KPI Overview Cards ─── */

function KpiRow({ data }: { data: DashboardOverview }) {
  const fmt = (n: number) => n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const items = [
    { title: '今日销售额', value: `$${fmt(data.today_sales)}`, icon: <ShoppingCartOutlined />, color: 'var(--b4)' },
    { title: '待审批', value: data.pending_approvals.toString(), icon: <WarningFilled />, color: 'var(--y4)' },
    { title: '未处理异常', value: data.exception_open_count.toString(), icon: <ExclamationCircleFilled />, color: 'var(--r4)' },
    { title: 'Agent 建议', value: data.agent_suggestions.toString(), icon: <RiseOutlined />, color: 'var(--g4)' },
  ];
  return (
    <Row gutter={16}>
      {items.map((item) => (
        <Col span={6} key={item.title}>
          <Card style={{ borderRadius: 'var(--r4)', border: '1px solid var(--bd)' }} size="small">
            <Statistic title={item.title} value={item.value} prefix={item.icon} valueStyle={{ color: item.color }} />
          </Card>
        </Col>
      ))}
    </Row>
  );
}
/* ─── Page ─── */

export default function DashboardPage() {
  const { data: brief, isLoading: briefLoading, error: briefError } = useQuery({
    queryKey: ['dashboard', 'brief'],
    queryFn: async () => {
      const res = await apiClient.get<DailyBrief>('/v1/dashboard/brief');
      return res.data;
    },
  });

  const { data: overview } = useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: async () => {
      const res = await apiClient.get<DashboardOverview>('/v1/dashboard/overview');
      return res.data;
    },
  });

  const data = brief;

  if (briefError) {
    return (
      <div style={{ padding: 'var(--space-xl)' }}>
        <Alert message="加载失败" description={briefError instanceof Error ? briefError.message : '未知错误'} type="error" showIcon />
      </div>
    );
  }

  if (briefLoading || !data) return <BriefSkeleton />;

  return (
    <div style={{ padding: 'var(--space-xl)', display: 'flex', flexDirection: 'column', gap: 'var(--space-xl)' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', margin: 0, color: 'var(--t1)' }}>Daily Brief</h1>

      {/* Top: profit */}
      <ProfitCard data={data} />

      {/* Middle: exceptions */}
      <ExceptionAlerts data={data} />

      {/* Bottom: detail tabs */}
      <DetailsTabs data={data} />

      {/* Collapsed: platform connections */}
      <Collapse
        items={[{
          key: 'platforms',
          label: `平台对接 (${data.platform_connections.length})`,
          children: data.platform_connections.length === 0
            ? <Empty description="暂无已对接平台" />
            : <Table rowKey="platform_id" dataSource={data.platform_connections} columns={platformColumns} pagination={false} size="small" />,
        }]}
        style={{ background: 'var(--s2)', border: '1px solid var(--bd)', borderRadius: 'var(--r4)' }}
      />
    </div>
  );
}
