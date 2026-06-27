'use client';

import { Button, Card, Descriptions, Image, message, Progress, Skeleton, Table, Tabs, Tag, Result } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';

// ----- Types -----

interface Certificate {
  id: number;
  name: string;
  type: string;
  expiry_date: string;
  is_expiring: boolean;
}

interface DecisionTraceItem {
  agent_id: string;
  action: string;
  reasoning: string;
  confidence: number;
  created_at: string;
}

interface Product360 {
  product_id: number;
  name: string;
  status: string;
  category_name: string;
  brand_name: string;
  sku_count: number;
  main_image: string;
  compliance: Record<string, string>;
  certificates: Certificate[];
  hs_code: string;
  listings: Record<string, string>;
  price: Record<string, number>;
  profit_margin: Record<string, number>;
  inventory: number;
  safety_stock: number;
  stock_status: string;
  health_score: number;
  cost_price: number;
  profit_score: number;
  demand_score: number | null;
  competition_score: number | null;
  lifecycle_status: string;
  decision_trace: DecisionTraceItem[];
  _metadata: {
    complete: boolean;
    missing_domains: string[];
    generated_at: string;
  };
}

// ----- Color & label helpers -----

const STATUS_COLORS: Record<string, string> = {
  draft: 'default',
  active: 'success',
  inactive: 'warning',
};

const STOCK_COLORS: Record<string, string> = {
  in_stock: 'success',
  low_stock: 'warning',
  out_of_stock: 'error',
  overstock: 'geekblue',
};

const LIFECYCLE_TAG_COLORS: Record<string, string> = {
  active: 'green',
  declining: 'orange',
  end_of_life: 'red',
  new: 'blue',
};

const COMPLIANCE_COLORS: Record<string, string> = {
  ok: 'success',
  warning: 'warning',
  fail: 'error',
  pending: 'processing',
};

function label(s: string, map: Record<string, string>): string {
  return map[s] ?? s;
}

const statusLabel = (s: string) => label(s, { draft: '草稿', active: '上架', inactive: '下架' });
const stockLabel = (s: string) => label(s, { in_stock: '有库存', low_stock: '库存不足', out_of_stock: '缺货', overstock: '库存过剩' });
const lifecycleLabel = (s: string) => label(s, { active: '活跃', declining: '衰退', end_of_life: '停产', new: '新品' });
const complianceLabel = (s: string) => label(s, { ok: '合规', warning: '警告', fail: '不合规', pending: '审核中' });

// Competition & demand helpers
function competitionScoreInfo(score: number | null): { color: string; label: string } {
  if (score == null) return { color: 'default', label: '-' };
  if (score >= 70) return { color: 'red', label: '高' };
  if (score >= 40) return { color: 'orange', label: '中' };
  return { color: 'green', label: '低' };
}

function demandScoreColor(score: number | null): string {
  if (score == null) return '#d9d9d9';
  if (score >= 70) return '#34D399';
  if (score >= 40) return '#FBBF24';
  return '#F87171';
}

// ----- Component -----

export default function Product360Page() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['product360', id],
    queryFn: async () => {
      const res = await apiClient.get<Product360>(`/v1/products/360/${id}`);
      return res.data;
    },
    retry: false,
  });

  // --- Loading state ---
  if (isLoading) {
    return (
      <PageContainer title="商品详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')} style={{ marginBottom: 16 }}>
          返回列表
        </Button>
        <Card>
          <Skeleton active paragraph={{ rows: 10 }} />
        </Card>
      </PageContainer>
    );
  }

  // --- Error state ---
  if (isError || !data) {
    return (
      <PageContainer title="商品详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')} style={{ marginBottom: 16 }}>
          返回列表
        </Button>
        <Card>
          <Result
            status="error"
            title="加载失败"
            subTitle="无法获取商品360详情，请检查网络后重试"
            extra={<Button onClick={() => refetch()}>重试</Button>}
          />
        </Card>
      </PageContainer>
    );
  }

  // --- Data derived from Product360 ---

  // Collect unique platform keys across all records
  const platformKeys = Array.from(
    new Set([
      ...Object.keys(data.listings ?? {}),
      ...Object.keys(data.price ?? {}),
      ...Object.keys(data.profit_margin ?? {}),
      ...Object.keys(data.compliance ?? {}),
    ]),
  );

  // Tab 4: Listing status table
  const listingColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    {
      title: '刊登状态',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => <Tag color={STATUS_COLORS[v] ?? 'default'}>{statusLabel(v)}</Tag>,
    },
    {
      title: '售价',
      dataIndex: 'price',
      key: 'price',
      render: (v: number | undefined) => (v != null ? `${v.toFixed(2)}` : '-'),
    },
    {
      title: '利润率',
      dataIndex: 'profit_margin',
      key: 'profit_margin',
      render: (v: number | undefined) => (v != null ? `${v.toFixed(1)}%` : '-'),
    },
  ];

  const listingData = platformKeys.map((p) => ({
    key: p,
    platform: p,
    status: data.listings[p] ?? '-',
    price: data.price[p],
    profit_margin: data.profit_margin[p],
  }));

  // Tab 5: Cost & profit
  const profitColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    {
      title: '售价',
      dataIndex: 'price',
      key: 'price',
      render: (v: number | undefined) => (v != null ? `${v.toFixed(2)}` : '-'),
    },
    {
      title: '利润率',
      dataIndex: 'profit_margin',
      key: 'profit_margin',
      render: (v: number | undefined) => (v != null ? `${v.toFixed(1)}%` : '-'),
    },
    {
      title: '成本价',
      dataIndex: 'cost_price',
      key: 'cost_price',
      render: (v: number) => fmtMoney(v),
    },
  ];

  const profitData = platformKeys.map((p) => ({
    key: p,
    platform: p,
    price: data.price[p],
    profit_margin: data.profit_margin[p],
    cost_price: data.cost_price,
  }));

  // Tab 6: Compliance
  const complianceColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    {
      title: '合规状态',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => <Tag color={COMPLIANCE_COLORS[v] ?? 'default'}>{complianceLabel(v)}</Tag>,
    },
  ];

  const complianceData = platformKeys.map((p) => ({
    key: p,
    platform: p,
    status: data.compliance[p] ?? 'pending',
  }));

  // Tab 6: Certificates
  const certColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '类型', dataIndex: 'type', key: 'type' },
    {
      title: '到期日',
      dataIndex: 'expiry_date',
      key: 'expiry_date',
      render: (v: string) => v ?? '-',
    },
    {
      title: '状态',
      dataIndex: 'is_expiring',
      key: 'is_expiring',
      render: (v: boolean) =>
        v ? <Tag color="warning">即将到期</Tag> : <Tag color="success">正常</Tag>,
    },
  ];

  // Tab 8: Decision trace
  const decisionColumns = [
    { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 80 },
    { title: '动作', dataIndex: 'action', key: 'action', width: 140 },
    {
      title: '决策理由',
      dataIndex: 'reasoning',
      key: 'reasoning',
      render: (v: string) => (v ? (v.length > 60 ? `${v.slice(0, 60)}...` : v) : '-'),
    },
    {
      title: '置信度',
      dataIndex: 'confidence',
      key: 'confidence',
      width: 100,
      render: (v: number) => (v != null ? `${(v * 100).toFixed(0)}%` : '-'),
    },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: fmtDate },
  ];

  const decisionData = (data.decision_trace ?? []).map((item, i) => ({ ...item, key: i }));

  // --- Tab definitions ---

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: (
        <Card>
          {data.main_image && (
            <div style={{ marginBottom: 16 }}>
              <Image src={data.main_image} alt={data.name} width={200} />
            </div>
          )}
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="商品名称">{data.name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="品牌">{data.brand_name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="分类">{data.category_name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="SKU 数量">{data.sku_count ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="HS 编码">{data.hs_code ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={STATUS_COLORS[data.status] ?? 'default'}>{statusLabel(data.status)}</Tag>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'content',
      label: '内容',
      children: (
        <Card>
          <div style={{ textAlign: 'center', padding: '48px 0', color: 'var(--text-secondary)' }}>
            <p style={{ marginBottom: 16, fontSize: 16 }}>多语言内容管理 (coming in Phase 4)</p>
            <Button type="primary" onClick={() => message.info('编辑功能将在 Phase 4 上线')}>
              编辑内容
            </Button>
          </div>
        </Card>
      ),
    },
    {
      key: 'images',
      label: '图片',
      children: (
        <Card>
          <div style={{ textAlign: 'center', padding: '48px 0', color: 'var(--text-secondary)' }}>
            <p>图片管理 (coming in Phase 4)</p>
          </div>
        </Card>
      ),
    },
    {
      key: 'listings',
      label: '刊登状态',
      children: (
        <Card title="各平台刊登状态">
          <Table rowKey="key" dataSource={listingData} columns={listingColumns} pagination={false} size="small" />
        </Card>
      ),
    },
    {
      key: 'profit',
      label: '成本与利润',
      children: (
        <>
          <Card style={{ marginBottom: 16 }}>
            <Descriptions bordered column={3} size="small">
              <Descriptions.Item label="成本价">{data.cost_price != null ? fmtMoney(data.cost_price) : '-'}</Descriptions.Item>
              <Descriptions.Item label="利润评分">{data.profit_score != null ? `${data.profit_score}/100` : '-'}</Descriptions.Item>
              <Descriptions.Item label="需求评分">{data.demand_score != null ? `${data.demand_score}/100` : '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
          <Card title="各平台利润明细">
            <Table rowKey="key" dataSource={profitData} columns={profitColumns} pagination={false} size="small" />
          </Card>
        </>
      ),
    },
    {
      key: 'compliance',
      label: '合规与认证',
      children: (
        <>
          <Card style={{ marginBottom: 16 }}>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="HS 编码">{data.hs_code ?? '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
          <Card title="各平台合规状态" style={{ marginBottom: 16 }}>
            <Table rowKey="key" dataSource={complianceData} columns={complianceColumns} pagination={false} size="small" />
          </Card>
          <Card title="认证信息">
            {(data.certificates ?? []).length > 0 ? (
              <Table rowKey="id" dataSource={data.certificates} columns={certColumns} pagination={false} size="small" />
            ) : (
              <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-secondary)' }}>暂无认证信息</div>
            )}
          </Card>
        </>
      ),
    },
    {
      key: 'inventory',
      label: '库存与供应链',
      children: (
        <Card>
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="当前库存">{data.inventory ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="安全库存">{data.safety_stock ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="库存状态">
              <Tag color={STOCK_COLORS[data.stock_status] ?? 'default'}>{stockLabel(data.stock_status)}</Tag>
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'decisions',
      label: '决策历史',
      children: (
        <Card title="Agent 决策记录">
          {(data.decision_trace ?? []).length > 0 ? (
            <Table rowKey="key" dataSource={decisionData} columns={decisionColumns} pagination={false} size="small" />
          ) : (
            <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-secondary)' }}>暂无决策记录</div>
          )}
        </Card>
      ),
    },
  ];

  return (
    <PageContainer title={data.name ?? '商品详情'}>
      <div style={{ marginBottom: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')}>
          返回列表
        </Button>
      </div>

      {/* Header: lifecycle badge + health score */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
          <div>
            <h1 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>{data.name}</h1>
            <div style={{ marginTop: 8, display: 'flex', gap: 8, alignItems: 'center' }}>
              <Tag color={LIFECYCLE_TAG_COLORS[data.lifecycle_status] ?? 'default'}>
                {lifecycleLabel(data.lifecycle_status)}
              </Tag>
              <Tag color={STATUS_COLORS[data.status] ?? 'default'}>{statusLabel(data.status)}</Tag>
            </div>
          </div>
          <div style={{ marginLeft: 'auto', minWidth: 200 }}>
            <div style={{ marginBottom: 4, color: 'var(--text-secondary)', fontSize: 13 }}>健康评分</div>
            <Progress
              percent={data.health_score}
              size="small"
              strokeColor={data.health_score >= 80 ? '#34D399' : data.health_score >= 60 ? '#FBBF24' : '#F87171'}
            />
          </div>
        </div>
      </Card>

      <Tabs items={tabItems} />
    </PageContainer>
  );
}
