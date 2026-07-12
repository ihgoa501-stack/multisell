'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Button, Card, Descriptions, Divider, Drawer, Form, Input, InputNumber,
  Modal, Select, Space, Table, Tag, Typography, message,
} from 'antd';
import { EyeOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

const { Text, Paragraph } = Typography;

type SourceRecord = {
  id: number;
  source_url: string;
  title?: string;
  price?: number;
  moq: number;
  supplier_name?: string;
  status: string;
  demand_case_id?: number;
  experiment_id?: string;
  snapshot_id?: number;
  reviewed_by?: number;
  reviewed_at?: string;
  review_notes?: string;
  product_id?: number;
  created_at: string;
};

type DraftResult = {
  draft?: Record<string, unknown>;
  listing_draft?: Record<string, unknown>;
  product?: Record<string, unknown>;
  trace?: Record<string, unknown>;
  [key: string]: unknown;
};

type Snapshot = { id: number; source_url: string; collected_at: string; driver: string; parser_version: string; raw_sha256: string; raw_payload: unknown; observed_title?: string; observed_price?: number; observed_moq?: number; observed_supplier?: string };

const statusLabel: Record<string, string> = {
  collected: '待复核', pending_review: '待复核', reviewed: '已复核', rejected: '已淘汰',
  converted: '已转产品', draft_created: '草稿已生成',
};
const statusColor: Record<string, string> = {
  collected: 'orange', pending_review: 'orange', reviewed: 'blue', rejected: 'red', converted: 'cyan', draft_created: 'green',
};

function requiredJSON(label: string) {
  return {
    validator: (_: unknown, value?: string) => {
      if (!value?.trim()) return Promise.reject(new Error(`请输入${label}`));
      try { JSON.parse(value); return Promise.resolve(); } catch { return Promise.reject(new Error(`${label}必须是有效 JSON`)); }
    },
  };
}

function toJSON(value: string) {
  return JSON.parse(value);
}

export default function Sourcing1688Page() {
  const qc = useQueryClient();
  const [captureOpen, setCaptureOpen] = useState(false);
  const [reviewing, setReviewing] = useState<SourceRecord | null>(null);
  const [converting, setConverting] = useState<SourceRecord | null>(null);
  const [preview, setPreview] = useState<DraftResult | null>(null);
  const [evidence, setEvidence] = useState<Snapshot | null>(null);
  const [captureForm] = Form.useForm();
  const [reviewForm] = Form.useForm();
  const [convertForm] = Form.useForm();

  const list = useQuery({
    queryKey: ['sourcing-1688-controlled'],
    queryFn: () => apiClient.getPage<SourceRecord>('/v1/sourcing-1688', { page: '1', size: '100' }),
  });

  const capture = useMutation({
    mutationFn: async (v: Record<string, unknown>) => apiClient.post<SourceRecord>('/v1/sourcing-1688/capture', {
      ...v,
      raw_payload: toJSON(v.raw_payload as string),
      images: toJSON(v.images as string),
      sku_variants: toJSON(v.sku_variants as string),
      collected_at: new Date(v.collected_at as string).toISOString(),
    }),
    onSuccess: (result) => {
      message.success(result.message || '真实来源快照已保存；记录进入待复核状态');
      setCaptureOpen(false); captureForm.resetFields(); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (e: Error) => message.error(`采集录入失败：${e.message}`),
  });

  const review = useMutation({
    mutationFn: (v: Record<string, unknown>) => apiClient.post(`/v1/sourcing-1688/${reviewing?.id}/review`, v),
    onSuccess: () => {
      message.success('Owner 复核已记录'); setReviewing(null); reviewForm.resetFields(); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (e: Error) => message.error(`复核失败：${e.message}`),
  });

  const convert = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const payload = {
        ...v,
        sku_variants: toJSON(v.sku_variants as string),
        media: toJSON(v.media as string),
        costs: toJSON(v.costs as string),
        supplier_assessment: toJSON(v.supplier_assessment as string),
        compliance_checks: toJSON(v.compliance_checks as string),
        listing_payload: toJSON(v.listing_payload as string),
        category_observed_at: new Date(v.category_observed_at as string).toISOString(),
      };
      return apiClient.post<DraftResult>(`/v1/sourcing-1688/${converting?.id}/convert-to-draft`, payload);
    },
    onSuccess: async (result) => {
      message.success('产品和待上架草稿已保存，未向平台发布');
      const sourceID = converting?.id;
      if (sourceID) {
        const detail = await apiClient.get<DraftResult>(`/v1/sourcing-1688/${sourceID}/draft`);
        setPreview(detail.data ?? result.data ?? {});
      } else setPreview(result.data ?? {});
      setConverting(null); convertForm.resetFields(); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (e: Error) => message.error(`生成草稿失败：${e.message}`),
  });

  const records = list.data?.data ?? [];
  return (
    <PageContainer
      title="1688 采集库 → 待上架草稿"
      subtitle="仅处理已批准的市场、渠道与商品机会。保存来源证据、Owner 复核并生成草稿；本页不会自动发布。"
      extra={<Space><Button icon={<ReloadOutlined />} onClick={() => void list.refetch()}>刷新</Button><Button type="primary" icon={<PlusOutlined />} onClick={() => setCaptureOpen(true)}>录入真实采集</Button></Space>}
    >
      <Alert type="warning" showIcon icon={<SafetyCertificateOutlined />} title="外部发布已锁定" description="这里只生成可预览、可追溯的待上架草稿。没有真实 1688 来源快照、已批准市场/实验、Owner 复核、图片使用权或完整成本时，系统应拒绝转草稿。" style={{ marginBottom: 16 }} />
      {list.isError ? <Alert type="error" showIcon title="采集库加载失败" description={(list.error as Error).message} /> : (
        <Table<SourceRecord>
          rowKey="id" loading={list.isLoading} dataSource={records} scroll={{ x: 1300 }}
          pagination={{ pageSize: 20, showTotal: (n) => `共 ${n} 条` }}
          columns={[
            { title: '商品 / 供应商', width: 220, render: (_, r) => <><Text strong>{r.title || '未解析标题'}</Text><br /><Text type="secondary">{r.supplier_name || '供应商待核验'}</Text></> },
            { title: '来源证据', width: 230, render: (_, r) => <><a href={r.source_url} target="_blank" rel="noreferrer">1688 原链接</a><br /><Text type="secondary">快照 #{r.snapshot_id ?? '缺失'}</Text></> },
            { title: '市场 / 实验', width: 190, render: (_, r) => <><Text>候选市场 #{r.demand_case_id ?? '缺失'}</Text><br /><Text type="secondary">实验 {r.experiment_id || '缺失'}</Text></> },
            { title: '采购信息', width: 130, render: (_, r) => <><Text>¥{r.price ?? '未知'}</Text><br /><Text type="secondary">MOQ {r.moq}</Text></> },
            { title: '复核状态', width: 150, render: (_, r) => <><Tag color={statusColor[r.status]}>{statusLabel[r.status] || r.status}</Tag><br /><Text type="secondary">{r.reviewed_at ? `Owner #${r.reviewed_by}` : '尚未复核'}</Text></> },
            { title: '追溯', width: 130, render: (_, r) => <><Text>采集 #{r.id}</Text><br /><Text type="secondary">产品 #{r.product_id ?? '未创建'}</Text></> },
            { title: '操作', fixed: 'right', width: 210, render: (_, r) => <Space wrap>
              <Button size="small" disabled={!r.snapshot_id} onClick={async () => { const res = await apiClient.get<Snapshot>(`/v1/sourcing-1688/${r.id}/snapshot`); setEvidence(res.data ?? null); }}>查看证据</Button>
              <Button size="small" disabled={!r.snapshot_id || !['collected', 'pending_review'].includes(r.status)} onClick={() => { setReviewing(r); reviewForm.setFieldsValue({ notes: '' }); }}>Owner 复核</Button>
              <Button size="small" type="primary" disabled={!r.reviewed_at || !['reviewed', 'converted'].includes(r.status)} onClick={() => setConverting(r)}>转待上架草稿</Button>
            </Space> },
          ]}
        />
      )}

      <Modal title="录入 1 个真实 1688 商品" open={captureOpen} width={760} onCancel={() => setCaptureOpen(false)} onOk={() => captureForm.validateFields().then((v) => capture.mutate(v))} confirmLoading={capture.isPending} okText="保存来源快照">
        <Alert type="info" showIcon title="录入不是批准" description="必须填写已批准候选市场与实验。原始 payload 会生成不可变快照；重复来源由后端去重。" style={{ marginBottom: 16 }} />
        <Form form={captureForm} layout="vertical" initialValues={{ driver: 'manual_owner', moq: 1, collected_at: new Date().toISOString().slice(0, 16), images: '[]', sku_variants: '[]', raw_payload: '{}' }}>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="demand_case_id" label="已批准候选市场 ID" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item><Form.Item name="experiment_id" label="商品实验 ID" rules={[{ required: true }]}><Input style={{ width: 240 }} /></Form.Item></Space>
          <Form.Item name="source_url" label="1688 原始链接" rules={[{ required: true }, { type: 'url' }]}><Input /></Form.Item>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="collected_at" label="实际采集时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item><Form.Item name="driver" label="采集驱动" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="parser_version" label="解析版本" rules={[{ required: true }]}><Input placeholder="例如 1688-parser@1.0.0" /></Form.Item></Space>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="title" label="商品标题"><Input style={{ width: 260 }} /></Form.Item><Form.Item name="supplier_name" label="供应商名称"><Input /></Form.Item><Form.Item name="price" label="采购单价"><InputNumber min={0} precision={2} /></Form.Item><Form.Item name="moq" label="起订量"><InputNumber min={1} /></Form.Item></Space>
          <Form.Item name="raw_payload" label="原始页面数据 JSON（不可变证据）" rules={[requiredJSON('原始页面数据')]}><Input.TextArea rows={5} /></Form.Item>
          <Form.Item name="images" label="原始图片列表 JSON" rules={[requiredJSON('图片列表')]}><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="sku_variants" label="供应商 SKU/变体 JSON" rules={[requiredJSON('SKU/变体')]}><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`Owner 复核采集 #${reviewing?.id ?? ''}`} open={!!reviewing} onCancel={() => setReviewing(null)} onOk={() => reviewForm.validateFields().then((v) => review.mutate(v))} confirmLoading={review.isPending} okText="确认复核">
        <Alert type="warning" showIcon title="请对照原链接与快照" description="确认商品、供应商、价格、MOQ 和变体来自同一次观察。复核不会发布，也不代表合规通过。" style={{ marginBottom: 16 }} />
        <Form form={reviewForm} layout="vertical"><Form.Item name="notes" label="复核说明" rules={[{ required: true }]}><Input.TextArea rows={4} placeholder="写明核对项、疑点和结论" /></Form.Item></Form>
      </Modal>

      <Modal title={`生成待上架草稿 · 采集 #${converting?.id ?? ''}`} open={!!converting} width={840} onCancel={() => setConverting(null)} onOk={() => convertForm.validateFields().then((v) => convert.mutate(v))} confirmLoading={convert.isPending} okText="保存并预览草稿">
        <Alert type="warning" showIcon title="不会自动发布" description="此动作只创建产品记录和渠道草稿。任何平台发布必须走单独的 Owner 批准。" style={{ marginBottom: 16 }} />
        <Form form={convertForm} layout="vertical" initialValues={{ unit: '件', target_locale: 'ru-RU', sku_variants: '[]', media: '[]', costs: '[]', supplier_assessment: '[]', compliance_checks: '[]', listing_payload: '{}', currency: 'CNY', category_observed_at: new Date().toISOString().slice(0, 16) }}>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="platform_id" label="已批准销售渠道 ID" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item><Form.Item name="category_id" label="渠道类目 ID" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item><Form.Item name="currency" label="成本币种" rules={[{ required: true }]}><Select style={{ width: 110 }} options={[{ value: 'CNY' }, { value: 'RUB' }, { value: 'USD' }]} /></Form.Item></Space>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="title" label="产品库标题" rules={[{ required: true }]}><Input style={{ width: 280 }} /></Form.Item><Form.Item name="platform_sku" label="渠道主 SKU" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="unit" label="单位" rules={[{ required: true }]}><Input style={{ width: 80 }} /></Form.Item></Space>
          <Form.Item name="description" label="产品库说明" rules={[{ required: true }]}><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="localized_title" label="本地化标题" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="localized_description" label="本地化卖点与说明" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item>
          <Form.Item name="target_locale" label="目标语言/地区" rules={[{ required: true }]}><Input placeholder="例如 ru-RU" /></Form.Item>
          <Divider>SKU 与图片</Divider>
          <Form.Item name="sku_variants" label="供应商 SKU → 渠道 SKU 映射 JSON" rules={[requiredJSON('SKU 映射')]}><Input.TextArea rows={3} /></Form.Item>
          <Form.Item name="media" label="图片权利与处理记录 JSON 数组" extra="每张图应有来源/处理 URL、角色、权利证据、非空 operations、尺寸、渠道规则来源；水印/中文/品牌标记必须为 false。" rules={[requiredJSON('图片记录')]}><Input.TextArea rows={4} /></Form.Item>
          <Divider>成本、类目与合规</Divider>
          <Form.Item name="costs" label="完整成本明细 JSON 数组" extra="每项应有 cost_type、amount、currency、truth_status、source_uri、observed_at；覆盖采购、国内运费、包装、跨境物流、平台/支付/广告费、税/关税、退货损失和汇率。" rules={[requiredJSON('完整成本')]}><Input.TextArea rows={5} /></Form.Item>
          <Form.Item name="supplier_assessment" label="供应商核验 JSON" extra="必须覆盖身份、经营历史、成交、混批、交期、样品和退换条件，并带来源与观察时间。" rules={[requiredJSON('供应商核验')]}><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="compliance_checks" label="合规核验 JSON" extra="必须覆盖品牌知识产权、专利、认证、危险品、材质、标签与说明书，全部有证据并通过。" rules={[requiredJSON('合规核验')]}><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="category_schema_uri" label="渠道类目规则来源" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="category_observed_at" label="类目规则观察时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
          <Form.Item name="listing_payload" label="渠道类目必填属性与上架 payload JSON" rules={[requiredJSON('上架 payload')]}><Input.TextArea rows={4} /></Form.Item>
          <Form.Item name="shipping_template_id" label="配送模板 ID" rules={[{ required: true }]}><Input /></Form.Item>
        </Form>
      </Modal>

      <Drawer title="待上架草稿预览（未发布）" open={!!preview} width={720} onClose={() => setPreview(null)} extra={<Tag color="orange">不会自动发布</Tag>}>
        <Alert type="success" showIcon title="草稿已保存" description="下方内容用于 Owner 人工验收；它不证明平台已接受，也没有触发外部发布。" />
        <Divider />
        <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="追溯链">采集 → 快照 → 市场/实验 → 产品 → 渠道草稿</Descriptions.Item>
          <Descriptions.Item label="后端返回"><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{JSON.stringify(preview, null, 2)}</Paragraph></Descriptions.Item>
        </Descriptions>
        <Divider />
        <Button icon={<EyeOutlined />} disabled>仅预览；发布入口未开放</Button>
      </Drawer>

      <Drawer title="不可变采集证据" open={!!evidence} width={720} onClose={() => setEvidence(null)}>
        {evidence && <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="原链接"><a href={evidence.source_url} target="_blank" rel="noreferrer">{evidence.source_url}</a></Descriptions.Item>
          <Descriptions.Item label="采集时间">{evidence.collected_at}</Descriptions.Item>
          <Descriptions.Item label="驱动 / 解析版本">{evidence.driver} / {evidence.parser_version}</Descriptions.Item>
          <Descriptions.Item label="本次解析字段">{evidence.observed_title || '—'} / {evidence.observed_supplier || '—'} / ¥{evidence.observed_price ?? '未知'} / MOQ {evidence.observed_moq ?? '未知'}</Descriptions.Item>
          <Descriptions.Item label="SHA-256"><Text copyable>{evidence.raw_sha256}</Text></Descriptions.Item>
          <Descriptions.Item label="原始 payload"><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{JSON.stringify(evidence.raw_payload, null, 2)}</Paragraph></Descriptions.Item>
        </Descriptions>}
      </Drawer>
    </PageContainer>
  );
}
