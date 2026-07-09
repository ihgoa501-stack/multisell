'use client';

import { useParams } from 'next/navigation';
import { Badge, Card, Col, Descriptions, Row, Spin, Statistic, Table, Tag, Timeline, Typography } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import { fmtDate } from '@/components/crud/CrudListPage';

const { Title, Text } = Typography;

const LIFECYCLE_COLORS: Record<string, string> = {
  idea: 'default', researching: 'blue', sampling: 'orange',
  approved: 'cyan', costed: 'geekblue', ready_to_list: 'purple',
  listed: 'processing', active: 'success', sunset: 'warning', archived: 'default',
};
const LIFECYCLE_LABELS: Record<string, string> = {
  idea: '创意', researching: '调研中', sampling: '打样中', approved: '已确认',
  costed: '已核算成本', ready_to_list: '待上架', listed: '已上架',
  active: '销售中', sunset: '衰退中', archived: '已归档',
};

interface ProductHubProfile {
  master: Record<string, unknown>;
  variants: Array<Record<string, unknown>>;
  concept: Record<string, unknown> | null;
  latest_cost: Record<string, unknown> | null;
  cost_history: Array<Record<string, unknown>>;
  suppliers: Array<Record<string, unknown>>;
  samples: Array<Record<string, unknown>>;
  timeline: Array<Record<string, unknown>>;
}

const STATUS_COLORS: Record<string, string> = { list: 'success', skip: 'error', cautious: 'warning' };
const MODE_TAGS: Record<string, string> = { dry_run: 'default', sandbox: 'orange', production: 'red' };
const APPROVAL_COLORS: Record<string, string> = { pending: 'gold', approved: 'success', rejected: 'error' };
const TASK_COLORS: Record<string, string> = {
  completed: 'success', executing: 'processing', failed: 'error',
  approved: 'blue', pending_approval: 'gold', blocked: 'default', rejected: 'error',
};

function EvdSection({ data }: { data: Record<string, unknown> }) {
  const ci = data?.candidate_info as Record<string, unknown> || {};
  const comp = data?.completeness as Record<string, unknown> || {};
  const ps = data?.profit_summary as Record<string, unknown> || {};
  const lr = data?.listing_recommendation as Record<string, unknown> || {};
  const approvals = (data?.approval_requests || []) as Array<Record<string, unknown>>;
  const tasks = (data?.listing_tasks || []) as Array<Record<string, unknown>>;
  const records = (data?.listing_records || []) as Array<Record<string, unknown>>;
  const complete = data?.complete_chain as boolean;

  return (
    <Card title={
      <span>证据链 <Tag color={complete ? 'success' : 'error'}>{complete ? '闭环完成' : '链路不完整'}</Tag></span>
    } size="small" style={{ marginTop: 16 }}>
      <Timeline
        items={[
          { color: 'blue', children: <><Text strong>候选商品</Text><br /><Text>{ci?.title as string || '-'}（来源: {ci?.source as string || '-'}）</Text></> },
          { color: comp?.score != null ? 'green' : 'gray', children: <><Text strong>完整度评分</Text><br /><Text>{comp?.score != null ? `${comp.score}/100` : '未评估'}{comp?.missing_items ? ` — 缺失: ${(comp.missing_items as string[])?.join(', ') || '无'}` : ''}</Text></> },
          { color: ps?.profit_margin ? (ps.profit_margin as number) > 0 ? 'green' : 'red' : 'gray', children: <><Text strong>利润评估</Text><br /><Text>预估利润 ${ps?.estimated_profit as number ?? '-'}（利润率 {ps?.profit_margin as number ?? '-'}%）</Text></> },
          { color: lr?.decision ? (lr.decision as string) === 'list' ? 'green' : 'orange' : 'gray', children: <><Text strong>上架建议</Text><br /><Tag color={STATUS_COLORS[lr?.decision as string] || 'default'}>{lr?.decision as string || '-'}</Tag> {lr?.reason as string || ''}</> },
          ...approvals.map((a, i) => ({
            color: (a?.status as string) === 'approved' ? 'green' : (a?.status as string) === 'rejected' ? 'red' : 'orange', children: <><Text strong>审批</Text><br /><Tag color={APPROVAL_COLORS[a?.status as string] || 'default'}>{a?.status as string || '-'}</Tag></>,
            key: i,
          })),
          ...tasks.map((t, i) => {
            const mode = t?.execution_mode as number ?? 0;
            const modeKey = mode === 1 ? 'sandbox' : mode === 3 ? 'production' : 'dry_run';
            return {
              color: TASK_COLORS[t?.status as string] || 'default',
              children: <><Text strong>上架任务 #{t?.id as number ?? ''}</Text><br /><Tag color={TASK_COLORS[t?.status as string] || 'default'}>{t?.status as string || '-'}</Tag><Tag color={MODE_TAGS[modeKey]}>{modeKey}</Tag>{t?.last_error ? <Text type="danger">: {t.last_error as string}</Text> : null}</>,
              key: i,
            };
          }),
          ...records.map((r, i) => ({
            color: 'green',
            children: <><Text strong>平台记录</Text><br /><Text>ID: {r?.platform_product_id as string || '-'} {r?.platform_url as string ? <a href={r.platform_url as string} target="_blank">查看</a> : ''}</Text></>,
            key: i,
          })),
        ]}
      />
      {!complete ? <Text type="warning">证据链不完整，部分环节数据缺失。</Text> : null}
    </Card>
  );
}

function ReviewSection({ productId }: { productId: string }) {
  const { data, isLoading } = useQuery<Record<string, unknown>>({
    queryKey: ['evidence-trace-summary', productId],
    queryFn: async () => (await apiClient.get<Record<string, unknown>>(`/v1/product-hub/${productId}/evidence`)).data ?? {},
    enabled: !!productId,
  });
  if (isLoading) return null;
  const lr = data?.listing_recommendation as Record<string, unknown> || {};
  const feedback = lr?.feedback_status as string;
  if (!feedback) return null;
  return (
    <Card title="推荐复盘" size="small" style={{ marginTop: 16 }}>
      <Descriptions column={2} size="small">
        <Descriptions.Item label="AI 建议">{lr?.decision as string || '-'}</Descriptions.Item>
        <Descriptions.Item label="Owner 反馈">
          <Tag color={feedback === 'adopted' ? 'success' : feedback === 'rejected' ? 'error' : feedback === 'executed' ? 'blue' : feedback === 'execution_failed' ? 'red' : 'default'}>{feedback}</Tag>
        </Descriptions.Item>
        {feedback === 'execution_failed' ? <Descriptions.Item label="失败原因">{lr?.feedback_note as string || '-'}</Descriptions.Item> : null}
        <Descriptions.Item label="采纳/拒绝理由">{lr?.feedback_note as string || '-'}</Descriptions.Item>
      </Descriptions>
    </Card>
  );
}

function PlatformComparisonSection({ productId }: { productId: string }) {
  const { data, isLoading } = useQuery<Array<Record<string, unknown>>>({
    queryKey: ['platform-comparison', productId],
    queryFn: async () => {
      const res = await apiClient.get<{ items: Array<Record<string, unknown>> }>(`/v1/listing/products/${productId}/platform-comparison`);
      return res.data?.items ?? [];
    },
    enabled: !!productId,
  });
  if (isLoading || !data || (data as Array<unknown>).length === 0) return null;
  return (
    <Card title="平台对比建议" size="small" style={{ marginTop: 16 }}>
      <Table
        dataSource={data as Array<Record<string, unknown>>}
        rowKey="_k"
        size="small"
        pagination={false}
        columns={[
          { title: '平台', dataIndex: 'platform_name', key: 'platform_name', render: (v) => (v as string) || '-' },
          { title: '预估利润', dataIndex: 'estimated_profit', key: 'estimated_profit', render: (v) => `$${(v as number ?? 0).toFixed(2)}` },
          { title: '利润率', dataIndex: 'profit_margin', key: 'profit_margin', render: (v) => `${(v as number ?? 0).toFixed(1)}%` },
          { title: '风险', dataIndex: 'risk_level', key: 'risk_level', render: (v) => <Tag color={(v as string) === 'high' ? 'red' : (v as string) === 'medium' ? 'orange' : 'green'}>{v as string || '-'}</Tag> },
          { title: '建议上架', dataIndex: 'suggested', key: 'suggested', render: (v) => v ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : null },
          { title: '状态', dataIndex: 'listing_status', key: 'listing_status', render: (v) => <Tag color={TASK_COLORS[v as string] || 'default'}>{v as string || '-'}</Tag> },
        ]}
      />
    </Card>
  );
}

export default function ProductHubDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data, isLoading } = useQuery<ProductHubProfile>({
    queryKey: ['product-hub', id],
    queryFn: async (): Promise<ProductHubProfile> => {
      const res = await apiClient.get<ProductHubProfile>(`/v1/product-hub/${id}/hub`);
      if (!res.data) throw new Error('Product not found');
      return res.data;
    },
  });

  const { data: evd, isLoading: evdLoading } = useQuery<Record<string, unknown>>({
    queryKey: ['evidence-trace', id],
    queryFn: async () => (await apiClient.get<Record<string, unknown>>(`/v1/product-hub/${id}/evidence`)).data ?? {},
    enabled: !!id,
  });

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!data) return <PageContainer title="产品档案"><Text type="danger">产品未找到</Text></PageContainer>;

  const { master, variants, concept, latest_cost, suppliers, samples, timeline } = data;

  return (
    <PageContainer title={((master?.name as string) || '产品档案')}>
      <Title level={3}>
        {(master?.name as string) || '-'}
        <Tag color={LIFECYCLE_COLORS[master?.lifecycle_status as string] || 'default'} style={{ marginLeft: 12 }}>
          {LIFECYCLE_LABELS[master?.lifecycle_status as string] || (master?.lifecycle_status as string) || '-'}
        </Tag>
      </Title>

      <Row gutter={[16, 16]}>
        <Col span={16}>
          <Card title="基本信息" size="small">
            <Descriptions column={2} size="small">
              <Descriptions.Item label="产品编号">{(master?.product_code as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="业务模式">{master?.business_model as string || '-'}</Descriptions.Item>
              <Descriptions.Item label="目标市场">{(master?.target_market as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="目标售价">¥{master?.target_price as number ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="负责人ID">{master?.owner_id as number ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="描述">{(master?.description as string) || '-'}</Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>
        <Col span={8}>
          <Card title="最新成本" size="small">
            {latest_cost ? (
              <>
                <Statistic title="到仓成本" value={latest_cost?.landed_cost as number ?? 0} prefix="¥" precision={2} />
                <Statistic title="建议售价" value={latest_cost?.recommended_price as number ?? 0} prefix="¥" precision={2} style={{ marginTop: 8 }} />
                <Statistic title="毛利率" value={latest_cost?.gross_margin as number ?? 0} suffix="%" precision={1} style={{ marginTop: 8 }} />
              </>
            ) : (<Text type="secondary">暂无成本数据</Text>)}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 'var(--space-lg)' }}>
        <Col span={12}>
          <Card title="产品创意" size="small">
            {concept ? (<Descriptions column={1} size="small">
              <Descriptions.Item label="简述">{(concept?.brief as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="目标客户">{(concept?.target_customer as string) || '-'}</Descriptions.Item>
              <Descriptions.Item label="解决痛点">{(concept?.pain_point as string) || '-'}</Descriptions.Item>
            </Descriptions>) : (<Text type="secondary">暂无创意信息</Text>)}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="供应商报价" size="small">
            {suppliers?.length ? (<Table dataSource={suppliers as Array<Record<string, unknown>>} rowKey={(_, idx) => String(idx)} size="small" pagination={false} columns={[
              { title: '供应商', dataIndex: 'supplier_name', key: 'name', render: (v) => (v as string) || '-' },
              { title: '单价', key: 'cost', render: (_, r) => { const o = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>; return o ? `¥${o.unit_cost}` : '-'; } },
              { title: 'MOQ', key: 'moq', render: (_, r) => { const o = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>; return o?.moq ?? '-'; } },
              { title: '优选', key: 'preferred', render: (_, r) => { const o = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>; return o?.is_preferred ? '★' : '-'; } },
            ]} />) : (<Text type="secondary">暂无供应商报价</Text>)}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title="打样记录" size="small">
            {samples?.length ? (<Table dataSource={samples as Array<Record<string, unknown>>} rowKey="id" size="small" pagination={false} columns={[
              { title: '状态', dataIndex: 'status', key: 'status' },
              { title: '评分', dataIndex: 'quality_score', key: 'score', render: (v) => (v as number ?? '-') },
              { title: '结论', dataIndex: 'decision', key: 'decision', render: (v) => (v as string) || '-' },
            ]} />) : (<Text type="secondary">暂无打样记录</Text>)}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="生命周期时间线" size="small">
            {timeline?.length ? (<Timeline items={(timeline as Array<Record<string, unknown>>).map((t) => ({ children: <>{t.summary as string}<Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(t.created_at)}</Text></> }))} />) : (<Text type="secondary">暂无事件</Text>)}
          </Card>
        </Col>
      </Row>

      <Card title="变体 / SKU" size="small" style={{ marginTop: 16 }}>
        {variants?.length ? (<Table dataSource={variants as Array<Record<string, unknown>>} rowKey="id" size="small" pagination={false} columns={[
          { title: 'SKU编码', dataIndex: 'sku_code', key: 'code', render: (v) => (v as string) || '-' },
          { title: '规格', dataIndex: 'variant_label', key: 'label', render: (v) => (v as string) || '-' },
          { title: '重量(kg)', dataIndex: 'weight', key: 'weight', render: (v) => (v as number ?? '-') },
          { title: '尺寸', dataIndex: 'dimensions', key: 'dimensions', render: (v) => (v as string) || '-' },
          { title: '条形码', dataIndex: 'barcode', key: 'barcode', render: (v) => (v as string) || '-' },
          { title: '原产国', dataIndex: 'country_of_origin', key: 'origin', render: (v) => (v as string) || '-' },
        ]} />) : (<Text type="secondary">暂无变体</Text>)}
      </Card>

      {/* Phase 2: Evidence Trace */}
      {!evdLoading && evd ? <EvdSection data={evd} /> : null}

      {/* Phase 5: Review Summary */}
      <ReviewSection productId={id} />

      {/* Phase 6: Platform Comparison */}
      <PlatformComparisonSection productId={id} />
    </PageContainer>
  );
}
