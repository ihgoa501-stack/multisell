'use client';

import { useParams } from 'next/navigation';
import { Badge, Card, Col, Descriptions, Row, Spin, Tag, Typography, Table, Statistic, Timeline } from 'antd';
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

interface EvidenceTrace {
  candidate_info?: Record<string, unknown>;
  completeness?: Record<string, unknown>;
  profit_summary?: Record<string, unknown>;
  listing_recommendation?: Record<string, unknown>;
  approval_requests?: Array<Record<string, unknown>>;
  listing_tasks: Array<Record<string, unknown>>;
  execution_results: Array<Record<string, unknown>>;
  listing_records: Array<Record<string, unknown>>;
  complete_chain?: boolean;
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

  const { data: evidence } = useQuery<EvidenceTrace>({
    queryKey: ['product-hub-evidence', id],
    queryFn: async () => {
      const res = await apiClient.get<EvidenceTrace>(`/v1/product-hub/${id}/evidence`);
      return res.data ?? ({} as EvidenceTrace);
    },
    enabled: !!data,
  });

  if (isLoading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  if (!data) return <PageContainer title="产品档案"><Text type="danger">产品未找到</Text></PageContainer>;

  const { master, variants, concept, latest_cost, suppliers, samples, timeline } = data;
  const latestApproval = evidence?.approval_requests?.[0];

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

      {/* ===== Phase 2: Evidence Trace ===== */}
      {evidence && (
        <Card title="决策证据链" size="small" style={{ marginTop: 16 }}>
          <Row gutter={16}>
            <Col span={18}>
              <Timeline items={[
                { color: 'blue', children: <><Text strong>候选商品</Text> — {(evidence.candidate_info?.source as string) || '-'} / {(evidence.candidate_info?.status as string) || '-'} / {(evidence.candidate_info?.created_date as string)?.slice(0, 10) || '-'}</> },
                { color: (evidence.completeness?.score as number ?? 0) >= 80 ? 'green' : 'orange', children: <><Text strong>完整性</Text> — {(evidence.completeness?.score as number ?? 0).toFixed(0)}/100 {(evidence.completeness?.missing_items as string[])?.length ? <Tag color="red">缺: {(evidence.completeness?.missing_items as string[]).join(', ')}</Tag> : <Tag color="green">完整</Tag>}</> },
                { color: (evidence.profit_summary?.profit_status as string) === 'profitable' ? 'green' : 'red', children: <><Text strong>利润</Text> — 预估: ${(evidence.profit_summary?.estimated_profit as number ?? 0).toFixed(2)} / 利润率: {(evidence.profit_summary?.profit_margin as number ?? 0).toFixed(1)}% / {(evidence.profit_summary?.profit_status as string) || '-'}</> },
                { color: (evidence.listing_recommendation?.decision as string) === 'list' ? 'green' : (evidence.listing_recommendation?.decision as string) === 'cautious' ? 'orange' : 'red', children: <><Text strong>上架建议</Text> — <Tag color={(evidence.listing_recommendation?.decision as string) === 'list' ? 'green' : (evidence.listing_recommendation?.decision as string) === 'cautious' ? 'orange' : 'red'}>{(evidence.listing_recommendation?.decision as string) || '-'}</Tag> / {((evidence.listing_recommendation?.confidence as number ?? 0) * 100).toFixed(0)}% / {(evidence.listing_recommendation?.reason as string) || '-'}</> },
                { color: (latestApproval?.status as string) === 'approved' ? 'green' : 'orange', children: <><Text strong>审批</Text> — <Tag color={(latestApproval?.status as string) === 'approved' ? 'green' : (latestApproval?.status as string) === 'rejected' ? 'red' : 'orange'}>{(latestApproval?.status as string) || 'pending'}</Tag></> },
                { color: (evidence.listing_tasks as Array<Record<string, unknown>>)?.length ? 'blue' : 'gray', children: <><Text strong>刊登任务</Text> — {(evidence.listing_tasks as Array<Record<string, unknown>>)?.length ?? 0} 个 / {(evidence.listing_tasks as Array<Record<string, unknown>>)?.[0]?.execution_mode as number === 3 ? <Tag color="red">Production</Tag> : (evidence.listing_tasks as Array<Record<string, unknown>>)?.[0]?.execution_mode as number === 2 ? <Tag color="purple">Approval Required</Tag> : (evidence.listing_tasks as Array<Record<string, unknown>>)?.[0]?.execution_mode as number === 1 ? <Tag color="orange">Sandbox</Tag> : <Tag>Dry-Run</Tag>}</> },
                { color: (evidence.execution_results as Array<Record<string, unknown>>)?.some(r => r.status === 'success') ? 'green' : 'red', children: <><Text strong>执行结果</Text> — {(evidence.execution_results as Array<Record<string, unknown>>)?.filter(r => r.status === 'success').length ?? 0} 成功 / {(evidence.execution_results as Array<Record<string, unknown>>)?.filter(r => r.status === 'failed').length ?? 0} 失败</> },
                { color: (evidence.listing_records as Array<Record<string, unknown>>)?.length ? 'green' : 'gray', children: <><Text strong>上架记录</Text> — {(evidence.listing_records as Array<Record<string, unknown>>)?.length ?? 0} 条 / {(evidence.listing_records as Array<Record<string, unknown>>)?.map(r => r.platform_product_id as string)?.filter(Boolean)?.join(', ') || '-'}</> },
              ]} />
            </Col>
            <Col span={6}>
              <Card size="small" title="全链状态">
                <div style={{ textAlign: 'center', padding: '16px 0' }}>
                  <Badge status={evidence.complete_chain ? 'success' : 'error'} />
                  <span style={{ fontSize: 16, fontWeight: 700, marginLeft: 8, color: evidence.complete_chain ? 'var(--g4)' : 'var(--r4)' }}>
                    {evidence.complete_chain ? '全链完成' : '未完成'}
                  </span>
                </div>
              </Card>
            </Col>
          </Row>
        </Card>
      )}

      {/* ===== Phase 5: Review Summary ===== */}
      <Card title="审核摘要" size="small" style={{ marginTop: 16 }}>
        <Descriptions column={1} size="small">
          <Descriptions.Item label="AI建议">
            <Tag color={(evidence?.listing_recommendation?.decision as string) === 'list' ? 'green' : (evidence?.listing_recommendation?.decision as string) === 'cautious' ? 'orange' : 'default'}>
              {(evidence?.listing_recommendation?.decision as string) || '-'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="Owner决策">
            {(latestApproval?.status as string) === 'approved' ? <Tag color="green">已采纳</Tag> : (latestApproval?.status as string) === 'rejected' ? <Tag color="red">已拒绝</Tag> : <Tag>待审批</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="执行结果">
            {(evidence?.execution_results as Array<Record<string, unknown>>)?.some(r => r.status === 'success') ? <Tag color="green">成功</Tag> : (evidence?.execution_results as Array<Record<string, unknown>>)?.some(r => r.status === 'failed') ? <Tag color="red">失败</Tag> : <Tag>未执行</Tag>}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* ===== Phase 6: Platform Comparison ===== */}
      <Card title="平台对比" size="small" style={{ marginTop: 16 }}>
        {(evidence?.listing_records as Array<Record<string, unknown>>)?.length ? (
          <Table
            dataSource={evidence?.listing_records as Array<Record<string, unknown>>}
            rowKey={(_, i) => String(i)}
            size="small"
            pagination={false}
            columns={[
              { title: '平台', dataIndex: 'platform_name', key: 'platform', render: (v: string) => v || '-' },
              { title: '平台商品ID', dataIndex: 'platform_product_id', key: 'pid', render: (v: string) => v || '-' },
              { title: 'URL', dataIndex: 'url', key: 'url', render: (v: string) => v ? <a href={v} target="_blank" rel="noopener noreferrer">查看</a> : '-' },
              { title: '状态', key: 'status', render: () => <Tag color="green">已发布</Tag> },
            ]}
          />
        ) : (
          <Text type="secondary">暂无平台对比数据</Text>
        )}
      </Card>
    </PageContainer>
  );
}
