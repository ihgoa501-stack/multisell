'use client';

import { useParams } from 'next/navigation';
import { Card, Col, Descriptions, Row, Spin, Tag, Typography, Table, Statistic, Timeline } from 'antd';
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
            ) : (
              <Text type="secondary">暂无成本数据</Text>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title="产品创意" size="small">
            {concept ? (
              <Descriptions column={1} size="small">
                <Descriptions.Item label="简述">{(concept?.brief as string) || '-'}</Descriptions.Item>
                <Descriptions.Item label="目标客户">{(concept?.target_customer as string) || '-'}</Descriptions.Item>
                <Descriptions.Item label="解决痛点">{(concept?.pain_point as string) || '-'}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Text type="secondary">暂无创意信息</Text>
            )}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="供应商报价" size="small">
            {suppliers && suppliers.length > 0 ? (
              <Table
                dataSource={suppliers as Array<Record<string, unknown>>}
                rowKey={(_, idx) => String(idx)}
                size="small"
                pagination={false}
                columns={[
                  { title: '供应商', dataIndex: 'supplier_name', key: 'name', render: (v) => (v as string) || '-' },
                  { title: '单价', key: 'cost', render: (_, r) => {
                    const offer = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>;
                    return offer ? `¥${offer.unit_cost}` : '-';
                  }},
                  { title: 'MOQ', key: 'moq', render: (_, r) => {
                    const offer = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>;
                    return offer?.moq ?? '-';
                  }},
                  { title: '优选', key: 'preferred', render: (_, r) => {
                    const offer = (r as Record<string, unknown>).supplier_offer as Record<string, unknown>;
                    return offer?.is_preferred ? '★' : '-';
                  }},
                ]}
              />
            ) : (
              <Text type="secondary">暂无供应商报价</Text>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={12}>
          <Card title="打样记录" size="small">
            {samples && samples.length > 0 ? (
              <Table
                dataSource={samples as Array<Record<string, unknown>>}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  { title: '状态', dataIndex: 'status', key: 'status' },
                  { title: '评分', dataIndex: 'quality_score', key: 'score', render: (v) => (v as number ?? '-') },
                  { title: '结论', dataIndex: 'decision', key: 'decision', render: (v) => (v as string) || '-' },
                ]}
              />
            ) : (
              <Text type="secondary">暂无打样记录</Text>
            )}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="生命周期时间线" size="small">
            {timeline && timeline.length > 0 ? (
              <Timeline
                items={(timeline as Array<Record<string, unknown>>).map((t) => ({
                  children: (
                    <>
                      {t.summary as string}
                      <Text type="secondary" style={{ marginLeft: 8 }}>{fmtDate(t.created_at)}</Text>
                    </>
                  ),
                }))}
              />
            ) : (
              <Text type="secondary">暂无事件</Text>
            )}
          </Card>
        </Col>
      </Row>

      <Card title="变体 / SKU" size="small" style={{ marginTop: 16 }}>
        {variants && variants.length > 0 ? (
          <Table
            dataSource={variants as Array<Record<string, unknown>>}
            rowKey="id"
            size="small"
            pagination={false}
            columns={[
              { title: 'SKU编码', dataIndex: 'sku_code', key: 'code', render: (v) => (v as string) || '-' },
              { title: '规格', dataIndex: 'variant_label', key: 'label', render: (v) => (v as string) || '-' },
              { title: '重量(kg)', dataIndex: 'weight', key: 'weight', render: (v) => (v as number ?? '-') },
              { title: '尺寸', dataIndex: 'dimensions', key: 'dimensions', render: (v) => (v as string) || '-' },
              { title: '条形码', dataIndex: 'barcode', key: 'barcode', render: (v) => (v as string) || '-' },
              { title: '原产国', dataIndex: 'country_of_origin', key: 'origin', render: (v) => (v as string) || '-' },
            ]}
          />
        ) : (
          <Text type="secondary">暂无变体</Text>
        )}
      </Card>
    </PageContainer>
  );
}
