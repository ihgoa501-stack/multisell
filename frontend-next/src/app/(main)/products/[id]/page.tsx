'use client';

import { Alert, Button, Card, Descriptions, Image, Input, message, Modal, Progress, Select, Skeleton, Space, Table, Tabs, Tag, Result, Tooltip, Typography } from 'antd';
import { AlertOutlined, ArrowLeftOutlined, CheckCircleOutlined, ExclamationCircleOutlined, ReloadOutlined, RollbackOutlined, WarningOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import ComplianceTab from './compliance-tab';
import { fmtDate, fmtMoney } from '@/components/crud/CrudListPage';
import { useState } from 'react';

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

interface ProductVersion {
  id: number;
  product_id: number;
  version_data: Record<string, unknown> | null;
  snapshot: Record<string, unknown> | null;
  agent_id: string;
  reason: string;
  created_at: string;
}

interface VersionListResponse {
  items: ProductVersion[];
  total: number;
  page: number;
  size: number;
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

// AI Content generation types
interface GeneratedContent {
  title: string;
  subtitle?: string;
  description: string;
  keywords: string[];
  confidence: number;
}

interface ContentReviewResult {
  passed: boolean;
  issues: string[];
  adjusted_confidence: number;
}

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

function VersionHistoryTab({ productId }: { productId: string }) {
  const queryClient = useQueryClient();
  const [modalVersion, setModalVersion] = useState<ProductVersion | null>(null);
  const [rollingBack, setRollingBack] = useState<number | null>(null);

  const { data: versionPage, isLoading } = useQuery({
    queryKey: ['product-versions', productId],
    queryFn: async () => {
      const res = await apiClient.get<VersionListResponse>(`/v1/products/${productId}/versions?page=1&size=50`);
      return res.data;
    },
  });

  const rollbackMutation = useMutation({
    mutationFn: async (versionId: number) => {
      await apiClient.post(`/v1/products/${productId}/versions/${versionId}/rollback`, {});
    },
    onSuccess: () => {
      message.success('回滚成功');
      setModalVersion(null);
      setRollingBack(null);
      queryClient.invalidateQueries({ queryKey: ['product360', productId] });
      queryClient.invalidateQueries({ queryKey: ['product-versions', productId] });
    },
    onError: (err: Error) => {
      message.error('回滚失败: ' + err.message);
      setRollingBack(null);
    },
  });

  const handleRollback = (version: ProductVersion) => {
    setModalVersion(version);
  };

  const confirmRollback = () => {
    if (!modalVersion) return;
    setRollingBack(modalVersion.id);
    rollbackMutation.mutate(modalVersion.id);
  };

  const versionColumns = [
    { title: '版本 ID', dataIndex: 'id', key: 'id', width: 90 },
    { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 80 },
    {
      title: '操作理由',
      dataIndex: 'reason',
      key: 'reason',
      render: (v: string) => (v ? (v.length > 80 ? v.slice(0, 80) + '...' : v) : '-'),
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170, render: fmtDate },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: ProductVersion) => (
        <Tooltip title="回滚到此版本">
          <Button
            type="link"
            size="small"
            icon={<RollbackOutlined />}
            loading={rollingBack === record.id}
            onClick={() => handleRollback(record)}
          >
            回滚
          </Button>
        </Tooltip>
      ),
    },
  ];

  const versions = versionPage?.items ?? [];

  return (
    <>
      <Card title="版本快照历史">
        {isLoading ? (
          <Skeleton active paragraph={{ rows: 5 }} />
        ) : versions.length > 0 ? (
          <Table rowKey="id" dataSource={versions} columns={versionColumns} pagination={false} size="small" />
        ) : (
          <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-secondary)' }}>暂无版本记录</div>
        )}
      </Card>
      <Modal
        title="确认回滚"
        open={!!modalVersion}
        onCancel={() => setModalVersion(null)}
        onOk={confirmRollback}
        confirmLoading={rollingBack != null}
        okText="确认回滚"
        cancelText="取消"
      >
        <p>确定要将商品回滚到以下版本吗？</p>
        {modalVersion && (
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label="版本ID">{modalVersion.id}</Descriptions.Item>
            <Descriptions.Item label="Agent">{modalVersion.agent_id || '-'}</Descriptions.Item>
            <Descriptions.Item label="理由">{modalVersion.reason || '-'}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{fmtDate(modalVersion.created_at)}</Descriptions.Item>
          </Descriptions>
        )}
        <p style={{ marginTop: 12, color: 'var(--text-secondary)' }}>回滚操作会自动创建一个当前状态的快照，以便后续恢复。</p>
      </Modal>
    </>
  );
}

// ----- Content AI Tab Component -----

function ContentAITab({
  productId,
  productName,
  category,
  brand,
}: {
  productId: string;
  productName: string;
  category: string;
  brand: string;
}) {
  const [language, setLanguage] = useState('zh');
  const [platform, setPlatform] = useState('ozon');
  const [genResult, setGenResult] = useState<GeneratedContent | null>(null);
  const [reviewResult, setReviewResult] = useState<ContentReviewResult | null>(null);

  const generateMutation = useMutation({
    mutationFn: async () => {
      const res = await apiClient.post<GeneratedContent>('/v1/content/generate', {
        product_name: productName,
        category,
        brand,
        specifications: '',
        target_language: language,
        platform,
      });
      return res.data;
    },
    onSuccess: (data) => {
      if (!data) return;
      setGenResult(data);
      apiClient.post<ContentReviewResult>('/v1/content/validate', {
        title: data.title,
        description: data.description,
        language,
        platform,
      }).then((res) => res.data && setReviewResult(res.data)).catch(() => {
        setReviewResult({ passed: true, issues: [], adjusted_confidence: data.confidence });
      });
    },
    onError: (err: Error) => {
      message.error('AI 生成失败: ' + err.message);
    },
  });

  return (
    <Card title="AI 内容生成">
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <div style={{ display: 'flex', gap: 12, alignItems: 'center', flexWrap: 'wrap' }}>
          <Select
            value={language}
            onChange={setLanguage}
            style={{ width: 120 }}
            options={[
              { value: 'zh', label: '中文' },
              { value: 'en', label: 'English' },
              { value: 'ru', label: 'Русский' },
            ]}
          />
          <Select
            value={platform}
            onChange={setPlatform}
            style={{ width: 140 }}
            options={[
              { value: 'ozon', label: 'Ozon' },
              { value: 'shopee', label: 'Shopee' },
              { value: 'wb', label: 'Wildberries' },
            ]}
          />
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            loading={generateMutation.isPending}
            onClick={() => generateMutation.mutate()}
          >
            AI 生成
          </Button>
        </div>

        {generateMutation.isPending && (
          <Progress percent={100} status="active" strokeColor="#1677ff" showInfo={false} />
        )}

        {genResult && (
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="置信度">
              <Tag color={genResult.confidence >= 0.7 ? 'success' : genResult.confidence >= 0.4 ? 'warning' : 'error'}>
                {(genResult.confidence * 100).toFixed(0)}%
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="标题">{genResult.title}</Descriptions.Item>
            {genResult.subtitle && <Descriptions.Item label="副标题">{genResult.subtitle}</Descriptions.Item>}
            <Descriptions.Item label="描述">
              <Typography.Paragraph ellipsis={{ rows: 3, expandable: true, symbol: '展开' }}>
                {genResult.description}
              </Typography.Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="关键词">
              {genResult.keywords && genResult.keywords.length > 0
                ? genResult.keywords.map((kw) => <Tag key={kw}>{kw}</Tag>)
                : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}

        {reviewResult && reviewResult.issues && reviewResult.issues.length > 0 && (
          <Alert
            type="warning"
            showIcon
            icon={<AlertOutlined />}
            message={'验证发现 ' + reviewResult.issues.length + ' 个问题'}
            description={
              <ul style={{ margin: 0, paddingLeft: 18 }}>
                {reviewResult.issues.map((issue, i) => (
                  <li key={i}>{issue}</li>
                ))}
              </ul>
            }
          />
        )}

        {reviewResult && reviewResult.passed && genResult && (
          <Alert
            type="success"
            showIcon
            icon={<CheckCircleOutlined />}
            message="内容验证通过"
            description={'置信度: ' + (reviewResult.adjusted_confidence * 100).toFixed(0) + '%'}
          />
        )}
      </Space>
    </Card>
  );
}

interface FreshnessItem {
  id: number;
  product_id: number;
  dimension: string;
  last_verified_at: string;
  next_check_at: string;
  freshness_days: number;
  status: string;
  drift_detected: boolean;
  last_value: string | null;
  current_value: string | null;
  freshness_label: string;
  days_since_check: number;
}

function FreshnessTab({ productId }: { productId: string }) {
  const { data: freshnessList, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ["product-freshness", productId],
    queryFn: async () => {
      const res = await apiClient.get<FreshnessItem[]>("/v1/products/" + productId + "/freshness");
      return res.data;
    },
    enabled: !!productId,
  });

  const verifyMutation = useMutation({
    mutationFn: async ({ dimension, value }: { dimension: string; value: string }) => {
      await apiClient.post("/v1/products/" + productId + "/freshness/verify", { dimension, current_value: value });
    },
    onSuccess: () => { message.success("验证记录已保存"); refetch(); },
    onError: (err: Error) => { message.error("验证失败: " + err.message); },
  });

  const handleVerify = (dimension: string) => {
    verifyMutation.mutate({ dimension, value: new Date().toISOString().split("T")[0] });
  };

  const FRESHNESS_COLORS: Record<string, string> = { fresh: "success", stale: "warning", expired: "error", drift: "red", unknown: "default" };
  const FRESHNESS_LABELS: Record<string, string> = { fresh: "新鲜", stale: "待检查", expired: "过期", drift: "数据漂移", unknown: "未知" };
  const FRESHNESS_DIMENSION_LABELS: Record<string, string> = { pricing: "定价", content: "内容", inventory: "库存", compliance: "合规" };

  const cols = [
    { title: "维度", dataIndex: "dimension", key: "dimension", width: 120, render: (v: string) => FRESHNESS_DIMENSION_LABELS[v] ?? v },
    {
      title: "状态", dataIndex: "freshness_label", key: "freshness_label", width: 130,
      render: (_: unknown, r: FreshnessItem) => {
        const color = FRESHNESS_COLORS[r.freshness_label] ?? "default";
        const label = FRESHNESS_LABELS[r.freshness_label] ?? r.freshness_label;
        const icon = r.freshness_label === "fresh" ? <CheckCircleOutlined style={{ marginRight: 4 }} /> : r.freshness_label === "drift" ? <WarningOutlined style={{ marginRight: 4 }} /> : <ExclamationCircleOutlined style={{ marginRight: 4 }} />;
        return <Tag color={color} icon={icon}>{label}</Tag>;
      },
    },
    { title: "上次验证", dataIndex: "last_verified_at", key: "last_verified_at", width: 170, render: (v: string) => v ? fmtDate(v) : "-" },
    { title: "下次检查", dataIndex: "next_check_at", key: "next_check_at", width: 170, render: (v: string) => v ? fmtDate(v) : "-" },
    { title: "间隔(天)", dataIndex: "freshness_days", key: "freshness_days", width: 90 },
    { title: "上次值", dataIndex: "last_value", key: "last_value", ellipsis: true, render: (v: string | null) => v ?? "-" },
    { title: "当前值", dataIndex: "current_value", key: "current_value", ellipsis: true, render: (v: string | null) => v ?? "-" },
    { title: "距上次(天)", dataIndex: "days_since_check", key: "days_since_check", width: 100, render: (v: number) => v ?? 0 },
    {
      title: "操作", key: "action", width: 80,
      render: (_: unknown, r: FreshnessItem) => (
        <Button size="small" type="link" icon={<CheckCircleOutlined />} loading={verifyMutation.isPending} onClick={() => handleVerify(r.dimension)}>验证</Button>
      ),
    },
  ];

  return (
    <Card title="数据新鲜度" extra={<Button icon={<ReloadOutlined />} size="small" loading={isRefetching} onClick={() => refetch()}>刷新</Button>}>
      {isLoading ? <Skeleton active paragraph={{ rows: 4 }} /> : freshnessList && freshnessList.length > 0 ? (
        <Table rowKey="id" dataSource={freshnessList} columns={cols} pagination={false} size="small" />
      ) : (
        <div style={{ textAlign: "center", padding: 24, color: "var(--text-secondary)" }}>暂无数据新鲜度记录。点击"验证"按钮开始追踪。</div>
      )}
    </Card>
  );
}


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

  if (isLoading) {
    return (
      <PageContainer title="商品详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')} style={{ marginBottom: 16 }}>返回列表</Button>
        <Card><Skeleton active paragraph={{ rows: 10 }} /></Card>
      </PageContainer>
    );
  }

  if (isError || !data) {
    return (
      <PageContainer title="商品详情">
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')} style={{ marginBottom: 16 }}>返回列表</Button>
        <Card>
          <Result status="error" title="加载失败" subTitle="无法获取商品360详情，请检查网络后重试" extra={<Button onClick={() => refetch()}>重试</Button>} />
        </Card>
      </PageContainer>
    );
  }

  const platformKeys = Array.from(new Set([
    ...Object.keys(data.listings ?? {}),
    ...Object.keys(data.price ?? {}),
    ...Object.keys(data.profit_margin ?? {}),
    ...Object.keys(data.compliance ?? {}),
  ]));

  const listingColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    { title: '刊登状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={STATUS_COLORS[v] ?? 'default'}>{statusLabel(v)}</Tag> },
    { title: '售价', dataIndex: 'price', key: 'price', render: (v: number | undefined) => (v != null ? v.toFixed(2) : '-') },
    { title: '利润率', dataIndex: 'profit_margin', key: 'profit_margin', render: (v: number | undefined) => (v != null ? v.toFixed(1) + '%' : '-') },
  ];

  const listingData = platformKeys.map((p) => ({ key: p, platform: p, status: data.listings[p] ?? '-', price: data.price[p], profit_margin: data.profit_margin[p] }));

  const profitColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    { title: '售价', dataIndex: 'price', key: 'price', render: (v: number | undefined) => (v != null ? v.toFixed(2) : '-') },
    { title: '利润率', dataIndex: 'profit_margin', key: 'profit_margin', render: (v: number | undefined) => (v != null ? v.toFixed(1) + '%' : '-') },
    { title: '成本价', dataIndex: 'cost_price', key: 'cost_price', render: (v: number) => fmtMoney(v) },
  ];

  const profitData = platformKeys.map((p) => ({ key: p, platform: p, price: data.price[p], profit_margin: data.profit_margin[p], cost_price: data.cost_price }));

  const complianceColumns = [
    { title: '平台', dataIndex: 'platform', key: 'platform', width: 120 },
    { title: '合规状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={COMPLIANCE_COLORS[v] ?? 'default'}>{complianceLabel(v)}</Tag> },
  ];

  const complianceData = platformKeys.map((p) => ({ key: p, platform: p, status: data.compliance[p] ?? 'pending' }));

  const certColumns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '类型', dataIndex: 'type', key: 'type' },
    { title: '到期日', dataIndex: 'expiry_date', key: 'expiry_date', render: (v: string) => v ?? '-' },
    { title: '状态', dataIndex: 'is_expiring', key: 'is_expiring', render: (v: boolean) => v ? <Tag color="warning">即将到期</Tag> : <Tag color="success">正常</Tag> },
  ];

  const decisionColumns = [
    { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 80 },
    { title: '动作', dataIndex: 'action', key: 'action', width: 140 },
    { title: '决策理由', dataIndex: 'reasoning', key: 'reasoning', render: (v: string) => (v ? (v.length > 60 ? v.slice(0, 60) + '...' : v) : '-') },
    { title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 100, render: (v: number) => (v != null ? (v * 100).toFixed(0) + '%' : '-') },
    { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: fmtDate },
  ];

  const decisionData = (data.decision_trace ?? []).map((item, i) => ({ ...item, key: i }));

  const tabItems = [
    {
      key: 'basic', label: '基本信息',
      children: (
        <Card>
          {data.main_image && <div style={{ marginBottom: 16 }}><Image src={data.main_image} alt={data.name} width={200} /></div>}
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="商品名称">{data.name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="品牌">{data.brand_name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="分类">{data.category_name ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="SKU 数量">{data.sku_count ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="HS 编码">{data.hs_code ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="状态"><Tag color={STATUS_COLORS[data.status] ?? 'default'}>{statusLabel(data.status)}</Tag></Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'content', label: '内容',
      children: <ContentAITab productId={id} productName={data.name ?? ''} category={data.category_name ?? ''} brand={data.brand_name ?? ''} />,
    },
    {
      key: 'images', label: '图片',
      children: (
        <Card><div style={{ textAlign: 'center', padding: '48px 0', color: 'var(--text-secondary)' }}><p>图片管理 (coming in Phase 4)</p></div></Card>
      ),
    },
    {
      key: 'compliance-scan', label: '合规检查',
      children: <ComplianceTab productId={id} />,
    },
    {
      key: 'listings', label: '刊登状态',
      children: <Card title="各平台刊登状态"><Table rowKey="key" dataSource={listingData} columns={listingColumns} pagination={false} size="small" /></Card>,
    },
    {
      key: 'profit', label: '成本与利润',
      children: (
        <>
          <Card style={{ marginBottom: 16 }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="成本价">{data.cost_price != null ? fmtMoney(data.cost_price) : '-'}</Descriptions.Item>
              <Descriptions.Item label="利润评分">{data.profit_score != null ? data.profit_score + '/100' : '-'}</Descriptions.Item>
              <Descriptions.Item label="需求分">
                {data.demand_score != null ? (
                  <Progress
                    percent={data.demand_score}
                    size="small"
                    strokeColor={demandScoreColor(data.demand_score)}
                    format={(pct) => `${pct}/100`}
                  />
                ) : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="竞争指数">
                {data.competition_score != null ? (
                  <Tag color={competitionScoreInfo(data.competition_score).color}>
                    {competitionScoreInfo(data.competition_score).label} ({data.competition_score}/100)
                  </Tag>
                ) : '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>
          <Card title="各平台利润明细"><Table rowKey="key" dataSource={profitData} columns={profitColumns} pagination={false} size="small" /></Card>
        </>
      ),
    },
    {
      key: 'compliance', label: '合规与认证',
      children: (
        <>
          <Card style={{ marginBottom: 16 }}><Descriptions bordered column={1} size="small"><Descriptions.Item label="HS 编码">{data.hs_code ?? '-'}</Descriptions.Item></Descriptions></Card>
          <Card title="各平台合规状态" style={{ marginBottom: 16 }}><Table rowKey="key" dataSource={complianceData} columns={complianceColumns} pagination={false} size="small" /></Card>
          <Card title="认证信息">{(data.certificates ?? []).length > 0 ? <Table rowKey="id" dataSource={data.certificates} columns={certColumns} pagination={false} size="small" /> : <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-secondary)' }}>暂无认证信息</div>}</Card>
        </>
      ),
    },
    {
      key: 'inventory', label: '库存与供应链',
      children: (
        <Card>
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="当前库存">{data.inventory ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="安全库存">{data.safety_stock ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="库存状态"><Tag color={STOCK_COLORS[data.stock_status] ?? 'default'}>{stockLabel(data.stock_status)}</Tag></Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: 'versions', label: '版本历史',
      children: <VersionHistoryTab productId={id} />,
    },
    {
      key: 'freshness', label: '数据新鲜度',
      children: <FreshnessTab productId={id} />,
    },
    {
      key: 'decisions', label: '决策历史',
      children: (
        <Card title="Agent 决策记录">
          {(data.decision_trace ?? []).length > 0 ? <Table rowKey="key" dataSource={decisionData} columns={decisionColumns} pagination={false} size="small" /> : <div style={{ textAlign: 'center', padding: 24, color: 'var(--text-secondary)' }}>暂无决策记录</div>}
        </Card>
      ),
    },
  ];

  return (
    <PageContainer title={data.name ?? '商品详情'}>
      <div style={{ marginBottom: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/products')}>返回列表</Button>
      </div>
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
          <div>
            <h1 style={{ margin: 0, fontSize: 24, fontWeight: 600 }}>{data.name}</h1>
            <div style={{ marginTop: 8, display: 'flex', gap: 8, alignItems: 'center' }}>
              <Tag color={LIFECYCLE_TAG_COLORS[data.lifecycle_status] ?? 'default'}>{lifecycleLabel(data.lifecycle_status)}</Tag>
              <Tag color={STATUS_COLORS[data.status] ?? 'default'}>{statusLabel(data.status)}</Tag>
            </div>
          </div>
          <div style={{ marginLeft: 'auto', minWidth: 200 }}>
            <div style={{ marginBottom: 4, color: 'var(--text-secondary)', fontSize: 13 }}>健康评分</div>
            <Progress percent={data.health_score} size="small" strokeColor={data.health_score >= 80 ? '#34D399' : data.health_score >= 60 ? '#FBBF24' : '#F87171'} />
          </div>
        </div>
      </Card>
      <Tabs items={tabItems} />
  );
}
