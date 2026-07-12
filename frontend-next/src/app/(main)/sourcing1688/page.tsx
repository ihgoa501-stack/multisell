'use client';

import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, App as AntdApp, Button, Card, Checkbox, Col, Collapse, Descriptions, Divider, Drawer, Form, Image, Input, InputNumber,
  Modal, Row, Select, Space, Table, Tag, Typography, Upload,
} from 'antd';
import { DeleteOutlined, ExclamationCircleOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, UploadOutlined } from '@ant-design/icons';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import { getToken } from '@/lib/auth';

const { Text, Paragraph } = Typography;
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

function protectedContentURL(path: string) {
  if (/^https?:\/\//i.test(path)) return path;
  return `${API_BASE.replace(/\/api\/?$/, '')}${path.startsWith('/') ? path : `/${path}`}`;
}

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
  lifecycle_status?: string;
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
type IdentityHistory = { snapshots?: unknown[]; changes?: unknown[]; duplicates?: Array<{ id: number; source_product_id: number; matched_product_id: number; status: string; match_type: string }> };
type AcceptanceItem = { number: number; code: string; title: string; status: 'passed' | 'blocked' | 'unknown'; summary: string; blockers: string[]; evidence: unknown[] };
type AcceptanceReport = { sourcing_product_id: number; generated_at: string; ready: boolean; status: 'passed' | 'blocked' | 'unknown'; items: AcceptanceItem[]; disclaimer: string };
type PublishAttempt = {
  id: number;
  sourcing_product_id: number;
  draft_id: number;
  product_id: number;
  listing_id: number;
  platform_id: number;
  platform_account_id: number;
  experiment_id: string;
  idempotency_key: string;
  request_sha256: string;
  status: string;
  error_message?: string;
  approval_id?: number;
  request_payload: unknown;
  adapter_request_payload?: unknown;
  response_payload?: unknown;
  response_sha256?: string;
  requested_by: number;
  requested_at: string;
  approved_by?: number;
  approved_at?: string;
  executed_at?: string;
  completed_at?: string;
};

type DraftSKU = { code: string; spec_desc?: string };

export const publishStatusMeta: Record<string, { color: string; label: string; description: string }> = {
  pending_approval: { color: 'orange', label: '等待独立审批', description: '请求已冻结，但尚未调用平台。' },
  approved: { color: 'blue', label: '已批准，待手动执行', description: '批准本身不会发布；Owner 仍需再次点击执行。' },
  rejected: { color: 'red', label: '已拒绝', description: '没有调用平台，可修正后用新的幂等键重新请求。' },
  executing: { color: 'processing', label: '平台调用进行中', description: '不要重复执行；等待结果或超时后对账。' },
  submitted: { color: 'cyan', label: '平台已接收请求', description: '这不等于商品已真实上线，仍需后续平台状态同步确认。' },
  reconcile_required: { color: 'gold', label: '结果不明确，需对账', description: '调用可能已到达平台，禁止重试；请用 actual 平台证据确认结果。' },
  succeeded: { color: 'green', label: '平台状态已确认', description: '表示后续同步已确认平台状态，不由 submitted 自动推断。' },
  failed: { color: 'red', label: '发布失败', description: '未证明上线；查看安全错误分类后决定是否重新请求。' },
};

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

const required = [{ required: true, message: '此项必填' }];
const truthOptions = [
  { value: 'actual', label: 'actual · Owner 实际核验' },
  { value: 'quoted', label: 'quoted · 来源原文' },
  { value: 'estimated', label: 'estimated · 有依据估算' },
];
const supplierCheckTypes = [
  ['identity', '供应商身份'], ['operating_history', '经营年限'], ['transaction_history', '成交记录'], ['moq', '起订量'],
  ['mixed_batch', '混批条件'], ['lead_time', '交期'], ['sample', '样品条件'], ['returns', '退换货条件'],
] as const;
const complianceCheckTypes = [
  ['brand_ip', '品牌与知识产权'], ['patent', '专利'], ['certification', '认证'], ['dangerous_goods', '危险品'],
  ['material', '材质限制'], ['labeling_instructions', '标签与说明书'],
] as const;
const costTypes = [
  ['purchase', '采购'], ['domestic_shipping', '国内运费'], ['packaging', '包装'], ['cross_border_shipping', '跨境物流'],
  ['platform_fee', '平台费'], ['payment_fee', '支付费'], ['advertising', '广告'], ['tax', '税费'], ['duty', '关税'], ['return_loss', '退货损失'],
] as const;

type StructuredRow = Record<string, unknown>;
type DraftFormValues = Record<string, unknown> & {
  sku_variants?: StructuredRow[];
  media?: StructuredRow[];
  costs?: StructuredRow[];
  supplier_assessment?: StructuredRow[];
  compliance_checks?: StructuredRow[];
  exchange_rates?: StructuredRow[];
  category_attributes?: Array<{ name?: string; value?: string }>;
  localized_attributes?: Array<{ name?: string; value?: string }>;
};

function localDateTime(value: unknown = new Date()) {
  const now = value instanceof Date ? value : new Date(String(value));
  if (Number.isNaN(now.getTime())) return '';
  return new Date(now.getTime() - now.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

function iso(value: unknown) {
  return new Date(String(value)).toISOString();
}

function rowsToObject(rows: Array<{ name?: string; value?: string }> = []) {
  return Object.fromEntries(rows.filter((row) => row.name?.trim()).map((row) => [row.name!.trim(), row.value ?? '']));
}

export function buildPublishRequestPayload(values: Record<string, unknown>) {
  const inventoryRows = (values.inventory_rows ?? []) as Array<{ sku_code?: string; quantity?: number }>;
  return {
    platform_account_id: values.platform_account_id,
    idempotency_key: String(values.idempotency_key ?? '').trim(),
    reason: String(values.reason ?? '').trim(),
    inventories: Object.fromEntries(inventoryRows.map((row) => [String(row.sku_code ?? '').trim(), row.quantity ?? 0])),
  };
}

export function buildReconcilePayload(values: Record<string, unknown>) {
  return {
    outcome: values.outcome,
    evidence_uri: String(values.evidence_uri ?? '').trim(),
    observed_at: iso(values.observed_at),
    truth_status: 'actual',
    platform_result: {
      platform_product_id: String(values.platform_product_id ?? '').trim(),
      platform_sku: String(values.platform_sku ?? '').trim(),
      platform_url: String(values.platform_url ?? '').trim(),
      published_data: rowsToObject((values.published_data as Array<{ name?: string; value?: string }>) ?? []),
      sync_message: String(values.sync_message ?? '').trim(),
    },
  };
}

function newPublishIdempotencyKey(sourceID: number) {
  const random = globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2);
  return `sourcing-${sourceID}-${Date.now()}-${random}`;
}

function formatPayload(value: unknown) {
  if (value == null || value === '') return '—';
  if (typeof value === 'string') {
    try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
  }
  return JSON.stringify(value, null, 2);
}

function evidenceRows(types: ReadonlyArray<readonly [string, string]>, actualOnly = false) {
  return types.map(([check_type]) => ({ check_type, value: '', result: 'pass', truth_status: actualOnly ? 'actual' : 'quoted', source_uri: '', observed_at: localDateTime(), notes: '' }));
}

function defaultCosts() {
  return costTypes.map(([cost_type]) => ({ cost_type, amount: 0, currency: 'CNY', truth_status: 'quoted', source_uri: '', observed_at: localDateTime() }));
}

function processedMediaDefaults(processed?: Record<string, unknown> | null) {
  if (!processed?.record_id) return [];
  return [{
    processing_record_id: processed.record_id,
    source_url: processed.source_url,
    processed_url: processed.content_url,
    media_role: 'main', rights_status: 'verified', rights_evidence_uri: processed.rights_evidence_uri,
    rights_observed_at: localDateTime(processed.rights_observed_at), channel_rule_uri: processed.channel_rule_uri,
    rights_observed_at_exact: processed.rights_observed_at,
    content_sha256: processed.processed_sha256, width: processed.width, height: processed.height,
    background: 'white', cropped: true, clarity_score: 1, no_watermark: false, no_chinese_text: false, no_brand_mark: false,
    verification_observed_at: localDateTime(),
  }];
}

function defaultDraftValues(processed?: Record<string, unknown> | null): DraftFormValues {
  return {
    unit: '件', target_locale: '', currency: 'CNY', category_observed_at: localDateTime(),
    sku_variants: [{ truth_status: 'quoted', observed_at: localDateTime(), color: '', size: '', material: '', packaging: '' }],
    media: processedMediaDefaults(processed), costs: defaultCosts(),
    supplier_assessment: evidenceRows(supplierCheckTypes), compliance_checks: evidenceRows(complianceCheckTypes, true),
    localized_bullet_points: [], localized_keywords: [], localized_attributes: [], allowed_scripts: ['cyrillic'], prohibited_words: [],
    localization_rule_truth_status: 'quoted', localization_rule_observed_at: localDateTime(), min_title_length: 1, max_title_length: 200,
    min_bullet_points: 0, max_bullet_length: 500, min_keywords: 0,
    category_attributes: [], variant_dimensions: ['color', 'size', 'material', 'packaging'],
    required_category_attributes: [], required_variant_dimensions: ['color', 'size', 'material', 'packaging'],
    allowed_variant_dimensions: ['color', 'size', 'material', 'packaging'], channel_rule_truth_status: 'quoted',
    min_images: 1, max_images: 9, min_image_width: 1000, min_image_height: 1000,
    image_rule_truth_status: 'quoted', image_rule_observed_at: localDateTime(), allowed_backgrounds: ['white'],
    require_crop: true, min_clarity_score: 0.8, image_rule_min_images: 1, image_rule_max_images: 9,
    sku_rule_truth_status: 'quoted', sku_rule_observed_at: localDateTime(), exchange_rates: [],
    revenue_truth_status: 'estimated', revenue_observed_at: localDateTime(), revenue_currency: 'CNY',
  };
}

function mediaOperations(row: StructuredRow) {
  return [
    { operation: 'center_crop' },
    { operation: 'resize', width: row.width, height: row.height },
    { operation: 'white_background' },
  ];
}

/** Maps Owner-friendly structured fields back to the unchanged backend contract. */
export function buildDraftPayload(values: DraftFormValues) {
  const skus = values.sku_variants ?? [];
  const media = values.media ?? [];
  const costs = values.costs ?? [];
  const categoryAttributes = rowsToObject(values.category_attributes);
  const localizedAttributes = rowsToObject(values.localized_attributes);
  const categoryObservedAt = iso(values.category_observed_at);
  const categorySchemaURI = String(values.category_schema_uri ?? '');
  const mappedSKUs = skus.map((row) => ({
    supplier_sku: row.supplier_sku, internal_sku: row.internal_sku, channel_sku: row.channel_sku,
    color: row.color, size: row.size, material: row.material, packaging: row.packaging,
    spec_desc: [row.color, row.size, row.material, row.packaging].filter(Boolean).join(' / '),
    spec_values: { color: row.color, size: row.size, material: row.material, packaging: row.packaging },
    cost_price: row.cost_price, price: row.price, weight: row.weight ?? 0, image: row.image ?? '',
  }));
  const mappedMedia = media.map((row) => ({
    processing_record_id: row.processing_record_id, source_url: row.source_url, processed_url: row.processed_url,
    media_role: row.media_role, rights_status: 'verified', rights_evidence_uri: row.rights_evidence_uri,
    rights_observed_at: iso(row.rights_observed_at_exact ?? row.rights_observed_at), operations: mediaOperations(row), content_sha256: row.content_sha256,
    width: row.width, height: row.height, has_watermark: !row.no_watermark, has_chinese_text: !row.no_chinese_text,
    has_brand_mark: !row.no_brand_mark, channel_rule_uri: row.channel_rule_uri,
  }));
  const mappedCosts: StructuredRow[] = costs.map((row) => ({ ...row, observed_at: iso(row.observed_at) }));
  const mappedChecks = (rows: StructuredRow[] = []) => rows.map((row) => ({ ...row, observed_at: iso(row.observed_at) }));
  const validationCosts = mappedCosts.map((row) => {
    const { cost_type, ...rest } = row;
    return { ...rest, type: cost_type };
  });
  const validationImages = media.map((row) => ({
    role: row.media_role, width: row.width, height: row.height, background: row.background, cropped: row.cropped,
    clarity_score: row.clarity_score, has_watermark: !row.no_watermark, has_chinese_text: !row.no_chinese_text,
    has_brand_mark: !row.no_brand_mark, truth_status: 'actual', source_uri: `sha256:${row.content_sha256}`,
    observed_at: iso(row.verification_observed_at),
  }));
  const ruleEvidence = (prefix: string) => ({
    truth_status: values[`${prefix}_truth_status`], source_uri: values[`${prefix}_source_uri`],
    observed_at: iso(values[`${prefix}_observed_at`]),
  });
  const payload = {
    platform_id: values.platform_id, category_id: values.category_id, currency: values.currency,
    title: values.title, description: values.description, platform_sku: values.platform_sku, unit: values.unit,
    localized_title: values.localized_title, localized_description: values.localized_description, target_locale: values.target_locale,
    sku_variants: mappedSKUs, media: mappedMedia, costs: mappedCosts,
    supplier_assessment: mappedChecks(values.supplier_assessment), compliance_checks: mappedChecks(values.compliance_checks),
    category_schema_uri: categorySchemaURI, category_observed_at: categoryObservedAt,
    listing_payload: { attributes: categoryAttributes, variant_dimensions: values.variant_dimensions, shipping_template_id: values.shipping_template_id },
    shipping_template_id: values.shipping_template_id,
    validation: {
      localization: { locale: values.target_locale, title: values.localized_title, description: values.localized_description, bullet_points: values.localized_bullet_points ?? [], keywords: values.localized_keywords ?? [], attributes: localizedAttributes, unit: values.unit },
      localization_rules: { evidence: ruleEvidence('localization_rule'), locale: values.target_locale, allowed_scripts: values.allowed_scripts ?? [], min_title_length: values.min_title_length, max_title_length: values.max_title_length, min_bullet_points: values.min_bullet_points, max_bullet_length: values.max_bullet_length, min_keywords: values.min_keywords, allowed_units: [values.unit], prohibited_words: values.prohibited_words ?? [] },
      channel: { platform_id: values.platform_id, category_id: String(values.category_id), category_schema_uri: categorySchemaURI, category_observed_at: categoryObservedAt, attributes: categoryAttributes, variant_dimensions: values.variant_dimensions ?? [], image_count: mappedMedia.length, image_widths: mappedMedia.map((row) => row.width), image_heights: mappedMedia.map((row) => row.height), shipping_template_id: values.shipping_template_id },
      channel_rules: { evidence: { truth_status: values.channel_rule_truth_status, source_uri: categorySchemaURI, observed_at: categoryObservedAt }, platform_id: values.platform_id, category_id: String(values.category_id), required_attributes: values.required_category_attributes ?? [], required_variant_dimensions: values.required_variant_dimensions ?? [], allowed_variant_dimensions: values.allowed_variant_dimensions ?? [], min_images: values.min_images, max_images: values.max_images, min_image_width: values.min_image_width, min_image_height: values.min_image_height, allowed_shipping_template_ids: [values.shipping_template_id] },
      costs: { target_currency: values.currency, costs: validationCosts, exchange_rates: (values.exchange_rates ?? []).map((row) => ({ ...row, observed_at: iso(row.observed_at) })), revenue: { amount: values.revenue_amount, currency: values.revenue_currency, truth_status: values.revenue_truth_status, source_uri: values.revenue_source_uri, observed_at: iso(values.revenue_observed_at) } },
      images: validationImages,
      image_rules: { evidence: ruleEvidence('image_rule'), min_main_width: values.min_image_width, min_main_height: values.min_image_height, allowed_backgrounds: values.allowed_backgrounds ?? [], require_crop: values.require_crop, min_clarity_score: values.min_clarity_score, min_images: values.image_rule_min_images, max_images: values.image_rule_max_images },
      skus: skus.map((row) => ({ supplier_sku: row.supplier_sku, internal_sku: row.internal_sku, channel_sku: row.channel_sku, color: row.color, size: row.size, material: row.material, packaging: row.packaging, truth_status: row.truth_status, source_uri: row.source_uri, observed_at: iso(row.observed_at) })),
      sku_rules: { evidence: ruleEvidence('sku_rule'), require_color: true, require_size: true, require_material: true, require_packaging: true },
    },
  };
  return payload;
}

function EvidenceChecklist({ name, types, actualOnly = false }: { name: string; types: ReadonlyArray<readonly [string, string]>; actualOnly?: boolean }) {
  return <Form.List name={name}>{(fields) => <Space orientation="vertical" style={{ width: '100%' }}>
    {fields.map((field, index) => <Card key={field.key} size="small" title={types[index]?.[1] ?? `核验 ${index + 1}`}>
      <Form.Item name={[field.name, 'check_type']} hidden><Input /></Form.Item>
      <Form.Item name={[field.name, 'value']} label="核验到的事实值" rules={required}><Input placeholder="例如经营 6 年、近 90 天成交 320 单、交期 7 天或不适用及理由" /></Form.Item>
      <Row gutter={12}>
        <Col xs={24} md={5}><Form.Item name={[field.name, 'result']} label="结论" rules={required}><Select options={[{ value: 'pass', label: '通过' }]} /></Form.Item></Col>
        <Col xs={24} md={7}><Form.Item name={[field.name, 'truth_status']} label="证据等级" rules={required}><Select options={actualOnly ? truthOptions.slice(0, 1) : truthOptions.slice(0, 2)} /></Form.Item></Col>
        <Col xs={24} md={12}><Form.Item name={[field.name, 'observed_at']} label="核验时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
      </Row>
      <Form.Item name={[field.name, 'source_uri']} label="证据来源" rules={required}><Input placeholder="原链接、合同、平台规则或文件位置" /></Form.Item>
      <Form.Item name={[field.name, 'notes']} label="说明"><Input.TextArea rows={2} /></Form.Item>
    </Card>)}
  </Space>}</Form.List>;
}

function KeyValueList({ name, addText }: { name: string; addText: string }) {
  return <Form.List name={name}>{(fields, { add, remove }) => <Space orientation="vertical" style={{ width: '100%' }}>
    {fields.map((field) => <Space key={field.key} align="start" style={{ display: 'flex' }}>
      <Form.Item name={[field.name, 'name']} rules={required}><Input placeholder="属性名" /></Form.Item>
      <Form.Item name={[field.name, 'value']} rules={required}><Input placeholder="属性值" /></Form.Item>
      <Button aria-label="删除属性" danger type="text" icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
    </Space>)}
    <Button type="dashed" icon={<PlusOutlined />} onClick={() => add()}>{addText}</Button>
  </Space>}</Form.List>;
}

export default function Sourcing1688Page() {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
  const [fetchOpen, setFetchOpen] = useState(false);
  const [captureOpen, setCaptureOpen] = useState(false);
  const [reviewing, setReviewing] = useState<SourceRecord | null>(null);
  const [converting, setConverting] = useState<SourceRecord | null>(null);
  const [preview, setPreview] = useState<DraftResult | null>(null);
  const [evidence, setEvidence] = useState<Snapshot | null>(null);
  const [approvalTarget, setApprovalTarget] = useState<SourceRecord | null>(null);
  const [decisionTarget, setDecisionTarget] = useState<{ record: SourceRecord; approvalId: number } | null>(null);
  const [imageTarget, setImageTarget] = useState<SourceRecord | null>(null);
  const [processedImage, setProcessedImage] = useState<Record<string, unknown> | null>(null);
  const [processedImagePreviewURL, setProcessedImagePreviewURL] = useState<string | null>(null);
  const [processedImagePreviewError, setProcessedImagePreviewError] = useState<string | null>(null);
  const [identityHistory, setIdentityHistory] = useState<{ sourceId: number; data: IdentityHistory } | null>(null);
  const [acceptanceReport, setAcceptanceReport] = useState<AcceptanceReport | null>(null);
  const [publishTarget, setPublishTarget] = useState<SourceRecord | null>(null);
  const [publishDecisionTarget, setPublishDecisionTarget] = useState<PublishAttempt | null>(null);
  const [publishExecuteTarget, setPublishExecuteTarget] = useState<PublishAttempt | null>(null);
  const [publishReconcileTarget, setPublishReconcileTarget] = useState<PublishAttempt | null>(null);
  const [captureForm] = Form.useForm();
  const [fetchForm] = Form.useForm();
  const [reviewForm] = Form.useForm();
  const [convertForm] = Form.useForm();
  const [decisionForm] = Form.useForm();
  const [imageForm] = Form.useForm();
  const [publishRequestForm] = Form.useForm();
  const [publishDecisionForm] = Form.useForm();
  const [publishExecuteForm] = Form.useForm();
  const [publishReconcileForm] = Form.useForm();
  const publishInventoryRows = Form.useWatch('inventory_rows', publishRequestForm) ?? [];
  const reconcileOutcome = Form.useWatch('outcome', publishReconcileForm);

  useEffect(() => () => {
    if (processedImagePreviewURL) URL.revokeObjectURL(processedImagePreviewURL);
  }, [processedImagePreviewURL]);

  const list = useQuery({
    queryKey: ['sourcing-1688-controlled'],
    queryFn: () => apiClient.getPage<SourceRecord>('/v1/sourcing-1688', { page: '1', size: '100' }),
  });

  const publishAttempts = useQuery({
    queryKey: ['sourcing-1688-publish-requests', publishTarget?.id],
    enabled: !!publishTarget,
    queryFn: () => apiClient.get<PublishAttempt[]>(`/v1/sourcing-1688/${publishTarget!.id}/publish-requests`),
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

  const controlledFetch = useMutation({
    mutationFn: (v: Record<string, unknown>) => apiClient.post('/v1/sourcing-1688/fetch', v),
    onSuccess: (result) => {
      message.success(result.message || '1688 页面已采集并保存为不可变来源快照');
      setFetchOpen(false);
      fetchForm.resetFields();
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (e: Error) => message.error(`URL 采集失败：${e.message}`),
  });

  const review = useMutation({
    mutationFn: (v: Record<string, unknown>) => apiClient.post(`/v1/sourcing-1688/${reviewing?.id}/review`, v),
    onSuccess: () => {
      message.success('Owner 复核已记录'); setReviewing(null); reviewForm.resetFields(); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (e: Error) => message.error(`复核失败：${e.message}`),
  });

  const rejectSource = useMutation({
    mutationFn: (notes: string) => apiClient.post(`/v1/sourcing-1688/${reviewing?.id}/review-decision`, { action: 'reject', notes }),
    onSuccess: () => { message.success('来源已淘汰并保存理由'); setReviewing(null); reviewForm.resetFields(); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] }); },
    onError: (e: Error) => message.error(`淘汰失败：${e.message}`),
  });

  const submitApproval = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) => apiClient.post(`/v1/sourcing-1688/${id}/submit-draft-approval`, { reason }),
    onSuccess: () => { message.success('草稿已提交 Owner 审批，仍未发布'); setApprovalTarget(null); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] }); },
    onError: (e: Error) => message.error(`提交审批失败：${e.message}`),
  });

  const decideApproval = useMutation({
    mutationFn: ({ action, note }: { action: string; note: string }) => apiClient.post(`/v1/sourcing-1688/${decisionTarget?.record.id}/approvals/${decisionTarget?.approvalId}/decision`, { action, note }),
    onSuccess: () => { message.success('审批决定已保存；没有触发外部发布'); setDecisionTarget(null); void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] }); },
    onError: (e: Error) => message.error(`审批失败：${e.message}`),
  });

  const processImage = useMutation({
    mutationFn: (v: Record<string, unknown>) => apiClient.post<Record<string, unknown>>('/v1/sourcing-1688/processed-images', { ...v, sourcing_product_id: imageTarget?.id, rights_observed_at: new Date(v.rights_observed_at as string).toISOString() }),
    onSuccess: async (result) => {
      const processed: Record<string, unknown> = { ...(result.data ?? {}), source_url: imageForm.getFieldValue('source_url'), source_id: imageTarget?.id };
      setProcessedImage(processed);
      setProcessedImagePreviewError(null);
      try {
        const token = getToken();
        const response = await fetch(protectedContentURL(String(processed.content_url ?? '')), {
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const objectURL = URL.createObjectURL(await response.blob());
        setProcessedImagePreviewURL(objectURL);
        message.success('图片已处理并加载预览；请实际目检后再填写草稿核验项');
      } catch (error) {
        setProcessedImagePreviewURL(null);
        setProcessedImagePreviewError((error as Error).message);
        message.error('处理记录已保存，但预览加载失败；不得在未目检时确认图片合格');
      }
    },
    onError: (e: Error) => message.error(`图片处理失败：${e.message}`),
  });

  const resolveDuplicate = useMutation({
    mutationFn: ({ id, decision }: { id: number; decision: string }) => apiClient.post(`/v1/sourcing-1688/duplicates/${id}/resolve`, { decision }),
    onSuccess: async () => { if (!identityHistory) return; const res = await apiClient.get<IdentityHistory>(`/v1/sourcing-1688/${identityHistory.sourceId}/identity-history`); setIdentityHistory({ sourceId: identityHistory.sourceId, data: res.data ?? {} }); message.success('疑似同款裁决已保存'); },
    onError: (e: Error) => message.error(`同款裁决失败：${e.message}`),
  });

  const convert = useMutation({
    mutationFn: async (v: Record<string, unknown>) => {
      const payload = buildDraftPayload(v as DraftFormValues);
      if (converting?.product_id) return apiClient.put<DraftResult>(`/v1/sourcing-1688/${converting.id}/draft`, payload);
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

  const requestPublish = useMutation({
    mutationFn: (values: Record<string, unknown>) => apiClient.post<PublishAttempt>(`/v1/sourcing-1688/${publishTarget?.id}/publish-requests`, buildPublishRequestPayload(values)),
    onSuccess: () => {
      message.success('发布请求已冻结并进入独立审批；没有调用外部平台');
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布请求创建失败：${e.message}`),
  });

  const decidePublish = useMutation({
    mutationFn: ({ action, note }: { action: 'approve' | 'reject'; note: string }) => apiClient.post<PublishAttempt>(`/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishDecisionTarget?.id}/decision`, { action, note }),
    onSuccess: (_, variables) => {
      message.success(variables.action === 'approve' ? '独立发布审批已批准；仍未调用平台' : '发布请求已拒绝；没有调用平台');
      setPublishDecisionTarget(null);
      publishDecisionForm.resetFields();
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布审批失败：${e.message}`),
  });

  const executePublish = useMutation({
    mutationFn: () => apiClient.post<PublishAttempt>(`/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishExecuteTarget?.id}/execute`, {}),
    onSuccess: () => {
      message.warning('平台调用已结束；请根据最新状态判断，submitted 仍不代表真实上线');
      setPublishExecuteTarget(null);
      publishExecuteForm.resetFields();
      void publishAttempts.refetch();
    },
    onError: (e: Error) => {
      message.error(`外部发布调用未确认成功：${e.message}`);
      void publishAttempts.refetch();
    },
  });

  const reconcilePublish = useMutation({
    mutationFn: (values: Record<string, unknown>) => apiClient.post<PublishAttempt>(`/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishReconcileTarget?.id}/reconcile`, buildReconcilePayload(values)),
    onSuccess: () => {
      message.success('actual 平台证据已用于后置对账；submitted 仍需后续状态同步确认上线');
      setPublishReconcileTarget(null);
      publishReconcileForm.resetFields();
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布结果对账失败：${e.message}`),
  });

  const openDraftEditor = (record: SourceRecord) => {
    convertForm.resetFields();
    const matchingImage = processedImage?.source_id === record.id ? processedImage : null;
    convertForm.setFieldsValue(defaultDraftValues(matchingImage));
    setConverting(record);
  };

  const loadEvidence = async (record: SourceRecord) => {
    try {
      const res = await apiClient.get<Snapshot>(`/v1/sourcing-1688/${record.id}/snapshot`);
      setEvidence(res.data ?? null);
    } catch (error) {
      message.error(`来源证据读取失败：${(error as Error).message}`);
    }
  };

  const loadIdentityHistory = async (record: SourceRecord) => {
    try {
      const res = await apiClient.get<IdentityHistory>(`/v1/sourcing-1688/${record.id}/identity-history`);
      setIdentityHistory({ sourceId: record.id, data: res.data ?? {} });
    } catch (error) {
      message.error(`变化/同款读取失败：${(error as Error).message}`);
    }
  };

  const loadAcceptanceReport = async (record: SourceRecord) => {
    try {
      const res = await apiClient.get<AcceptanceReport>(`/v1/sourcing-1688/${record.id}/acceptance-report`);
      setAcceptanceReport(res.data ?? null);
    } catch (error) {
      message.error(`15 项验收读取失败：${(error as Error).message}`);
    }
  };

  const openPublishSafety = async (record: SourceRecord) => {
    setPublishTarget(record);
    publishRequestForm.resetFields();
    publishRequestForm.setFieldsValue({
      platform_account_id: undefined,
      idempotency_key: newPublishIdempotencyKey(record.id),
      reason: '',
      inventory_rows: [],
    });
    try {
      const draft = await apiClient.get<{ skus?: DraftSKU[] }>(`/v1/sourcing-1688/${record.id}/draft`);
      publishRequestForm.setFieldValue('inventory_rows', (draft.data?.skus ?? []).map((sku) => ({ sku_code: sku.code, quantity: 0 })));
    } catch (error) {
      message.error(`无法读取冻结库存所需的 SKU：${(error as Error).message}`);
    }
  };

  const records = list.data?.data ?? [];
  const attempts = publishAttempts.data?.data ?? [];
  const canCreatePublishRequest = publishAttempts.isSuccess && !attempts.some((attempt) => !['rejected', 'failed'].includes(attempt.status));
  return (
    <PageContainer
      title="1688 采集库 → 待上架草稿"
      subtitle="仅处理已批准的市场、渠道与商品机会。草稿与真实发布分别审批，任何步骤都不会自动串联。"
      extra={<Space wrap><Button icon={<ReloadOutlined />} onClick={() => void list.refetch()}>刷新</Button><Button onClick={() => setCaptureOpen(true)}>高级证据导入</Button><Button type="primary" icon={<PlusOutlined />} onClick={() => setFetchOpen(true)}>从 1688 URL 采集</Button></Space>}
    >
      <Alert type="warning" showIcon icon={<SafetyCertificateOutlined />} title="外部发布受独立审批保护" description="草稿批准不会发布。只有 approved_draft 才显示发布安全区；请求发布、Owner 独立批准、再次执行和异常对账必须分别手动完成。" style={{ marginBottom: 16 }} />
      {list.isError ? <Alert type="error" showIcon title="采集库加载失败" description={(list.error as Error).message} /> : (
        <Table<SourceRecord>
          rowKey="id" loading={list.isLoading} dataSource={records} scroll={{ x: 1300 }}
          pagination={{ pageSize: 20, showTotal: (n) => `共 ${n} 条` }}
          columns={[
            { title: '商品 / 供应商', width: 220, render: (_, r) => <><Text strong>{r.title || '未解析标题'}</Text><br /><Text type="secondary">{r.supplier_name || '供应商待核验'}</Text></> },
            { title: '来源证据', width: 230, render: (_, r) => <><a href={r.source_url} target="_blank" rel="noreferrer">1688 原链接</a><br /><Text type="secondary">快照 #{r.snapshot_id ?? '缺失'}</Text></> },
            { title: '市场 / 实验', width: 190, render: (_, r) => <><Text>候选市场 #{r.demand_case_id ?? '缺失'}</Text><br /><Text type="secondary">实验 {r.experiment_id || '缺失'}</Text></> },
            { title: '采购信息', width: 130, render: (_, r) => <><Text>¥{r.price ?? '未知'}</Text><br /><Text type="secondary">MOQ {r.moq}</Text></> },
            { title: '生命周期', width: 170, render: (_, r) => <><Tag color={statusColor[r.status]}>{r.lifecycle_status || statusLabel[r.status] || r.status}</Tag><br /><Text type="secondary">{r.reviewed_at ? `Owner #${r.reviewed_by}` : '尚未复核'}</Text></> },
            { title: '追溯', width: 130, render: (_, r) => <><Text>采集 #{r.id}</Text><br /><Text type="secondary">产品 #{r.product_id ?? '未创建'}</Text></> },
            { title: '操作', fixed: 'right', width: 240, render: (_, r) => <Space wrap>
              <Button size="small" disabled={!r.snapshot_id} onClick={() => void loadEvidence(r)}>查看证据</Button>
              <Button size="small" disabled={!r.snapshot_id} onClick={() => void loadIdentityHistory(r)}>变化/同款</Button>
              <Button size="small" onClick={() => void loadAcceptanceReport(r)}>15项验收</Button>
              <Button size="small" disabled={!r.snapshot_id || !['collected', 'pending_review'].includes(r.status)} onClick={() => { setReviewing(r); reviewForm.setFieldsValue({ notes: '' }); }}>Owner 复核</Button>
              <Button size="small" type="primary" disabled={!r.reviewed_at || !['ready_for_product', 'editing'].includes(r.lifecycle_status || '')} onClick={() => openDraftEditor(r)}>{r.product_id ? '编辑草稿' : '转待上架草稿'}</Button>
              <Button size="small" disabled={!r.reviewed_at} onClick={() => { setImageTarget(r); setProcessedImage(null); setProcessedImagePreviewURL(null); setProcessedImagePreviewError(null); }}>处理图片</Button>
              <Button size="small" disabled={r.lifecycle_status !== 'editing'} onClick={() => setApprovalTarget(r)}>提交审批</Button>
              <Button size="small" disabled={r.lifecycle_status !== 'pending_approval'} onClick={async () => { const res = await apiClient.get<{ approval_id?: number }>(`/v1/sourcing-1688/${r.id}/lifecycle`); const approvalId = res.data?.approval_id; if (approvalId) setDecisionTarget({ record: r, approvalId }); else message.error('未找到草稿审批记录'); }}>审批草稿</Button>
              {r.lifecycle_status === 'approved_draft' && <Button size="small" danger icon={<SafetyCertificateOutlined />} onClick={() => void openPublishSafety(r)}>发布安全</Button>}
            </Space> },
          ]}
        />
      )}

      <Modal title="从 URL 采集 1 个真实 1688 商品" open={fetchOpen} width={680} onCancel={() => setFetchOpen(false)} onOk={() => fetchForm.validateFields().then((v) => controlledFetch.mutate(v))} confirmLoading={controlledFetch.isPending} okText="采集并保存证据">
        <Alert type="info" showIcon title="先打开同一个 1688 商品页面并确认扩展已连接" description="浏览器中必须已登录 1688，并打开与下方完全相同的商品 URL；凌镜扩展状态必须为 connected。随后系统才会检查已批准市场、active 实验和商品机会，并保存真实页面快照。" style={{ marginBottom: 16 }} />
        <Form form={fetchForm} layout="vertical">
          <Row gutter={12}>
            <Col xs={24} md={10}><Form.Item name="demand_case_id" label="已批准候选市场 ID" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={14}><Form.Item name="experiment_id" label="商品实验 ID" rules={required}><Input /></Form.Item></Col>
          </Row>
          <Form.Item name="source_url" label="1688 商品原始链接" rules={[...required, { type: 'url', message: '请输入完整 URL' }]}><Input placeholder="https://detail.1688.com/offer/..." /></Form.Item>
          <Paragraph type="secondary">解析版本由受控采集器返回并随快照保存，不能由前端伪造。</Paragraph>
        </Form>
      </Modal>

      <Modal title="高级证据导入" open={captureOpen} width={760} onCancel={() => setCaptureOpen(false)} onOk={() => captureForm.validateFields().then((v) => capture.mutate(v))} confirmLoading={capture.isPending} okText="导入来源快照">
        <Alert type="warning" showIcon title="仅用于导入已有原始证据" description="优先使用“从 1688 URL 采集”。这里要求你自行提供采集驱动、解析版本和原始 payload，适合已由其他只读采集器取得且需要完整留痕的证据。" style={{ marginBottom: 16 }} />
        <Form form={captureForm} layout="vertical" initialValues={{ driver: 'manual_owner', moq: 1, collected_at: new Date().toISOString().slice(0, 16), images: '[]', sku_variants: '[]', raw_payload: '{}' }}>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="demand_case_id" label="已批准候选市场 ID" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item><Form.Item name="experiment_id" label="商品实验 ID" rules={[{ required: true }]}><Input style={{ width: 240 }} /></Form.Item></Space>
          <Form.Item name="source_url" label="1688 原始链接" rules={[{ required: true }, { type: 'url' }]}><Input /></Form.Item>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="collected_at" label="实际采集时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item><Form.Item name="driver" label="采集驱动" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="parser_version" label="解析版本" rules={[{ required: true }]}><Input placeholder="例如 1688-parser@1.0.0" /></Form.Item></Space>
          <Space align="start" style={{ display: 'flex' }}><Form.Item name="title" label="商品标题"><Input style={{ width: 260 }} /></Form.Item><Form.Item name="supplier_name" label="供应商名称"><Input /></Form.Item><Form.Item name="supplier_business_id" label="供应商企业/店铺 ID" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="price" label="采购单价"><InputNumber min={0} precision={2} /></Form.Item><Form.Item name="moq" label="起订量"><InputNumber min={1} /></Form.Item></Space>
          <Form.Item name="raw_payload" label="原始页面数据 JSON（不可变证据）" rules={[requiredJSON('原始页面数据')]}><Input.TextArea rows={5} /></Form.Item>
          <Form.Item name="images" label="原始图片列表 JSON" rules={[requiredJSON('图片列表')]}><Input.TextArea rows={2} /></Form.Item>
          <Form.Item name="sku_variants" label="供应商 SKU/变体 JSON" rules={[requiredJSON('SKU/变体')]}><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`真实图片处理 · 采集 #${imageTarget?.id ?? ''}`} open={!!imageTarget} width={720} onCancel={() => setImageTarget(null)} onOk={() => imageForm.validateFields().then((v) => processImage.mutate(v))} confirmLoading={processImage.isPending} okText="处理并保存版本">
        <Alert type="warning" showIcon title="不会自动去水印或品牌" description="系统执行中心裁切、缩放和白底合成。图片使用权必须已有证据；仍含水印、中文或品牌标识的图片不能通过后续草稿门禁。" style={{ marginBottom: 16 }} />
        <Form form={imageForm} layout="vertical" initialValues={{ width: 1200, height: 1200, format: 'jpeg', quality: 90, rights_truth_status: 'actual', rights_observed_at: new Date().toISOString().slice(0, 16) }}>
          <Form.Item label="选择原图" required><Upload maxCount={1} accept="image/png,image/jpeg" beforeUpload={(file) => { const reader = new FileReader(); reader.onload = () => imageForm.setFieldValue('source_base64', String(reader.result).split(',')[1]); reader.readAsDataURL(file); return false; }}><Button icon={<UploadOutlined />}>选择 JPG/PNG（最大 10 MiB）</Button></Upload></Form.Item>
          <Form.Item name="source_base64" hidden rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="source_url" label="原图来源 URL" rules={[{ required: true }]}><Input /></Form.Item>
          <Space align="start"><Form.Item name="width" label="输出宽度" rules={[{ required: true }]}><InputNumber min={100} max={4000} /></Form.Item><Form.Item name="height" label="输出高度" rules={[{ required: true }]}><InputNumber min={100} max={4000} /></Form.Item><Form.Item name="format" label="格式" rules={[{ required: true }]}><Select style={{ width: 110 }} options={[{ value: 'jpeg' }, { value: 'png' }]} /></Form.Item><Form.Item name="quality" label="JPEG质量"><InputNumber min={60} max={100} /></Form.Item></Space>
          <Form.Item name="rights_evidence_uri" label="图片使用权证据" rules={[{ required: true }]}><Input /></Form.Item>
          <Space><Form.Item name="rights_truth_status" label="权利证据状态" rules={[{ required: true }]}><Select style={{ width: 120 }} options={[{ value: 'actual', label: 'Owner 已核验' }]} /></Form.Item><Form.Item name="rights_observed_at" label="权利核验时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item></Space>
          <Form.Item name="channel_rule_uri" label="渠道图片规则来源" rules={[{ required: true }]}><Input /></Form.Item>
        </Form>
        {processedImage && <Space orientation="vertical" style={{ width: '100%' }}>
          {processedImagePreviewURL ? <Card size="small" title="处理后图片（必须实际目检）"><Image src={processedImagePreviewURL} alt="处理后商品图预览" style={{ maxHeight: 420, objectFit: 'contain' }} /></Card> : <Alert type="error" showIcon title="处理后图片尚未成功显示" description={processedImagePreviewError ? `预览读取失败：${processedImagePreviewError}` : '请等待受保护图片载入；未看到图片前不能确认无水印、无中文或无品牌标识。'} />}
          <Alert type="success" showIcon title="处理记录已保存" description={<Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{JSON.stringify(processedImage, null, 2)}</Paragraph>} />
        </Space>}
      </Modal>

      <Modal title={`Owner 复核采集 #${reviewing?.id ?? ''}`} open={!!reviewing} onCancel={() => setReviewing(null)} onOk={() => reviewForm.validateFields().then((v) => review.mutate(v))} confirmLoading={review.isPending} okText="通过复核" footer={(_, { OkBtn, CancelBtn }) => <><CancelBtn /><Button danger loading={rejectSource.isPending} onClick={() => reviewForm.validateFields().then((v) => rejectSource.mutate(v.notes))}>淘汰来源</Button><OkBtn /></>}>
        <Alert type="warning" showIcon title="请对照原链接与快照" description="确认商品、供应商、价格、MOQ 和变体来自同一次观察。复核不会发布，也不代表合规通过。" style={{ marginBottom: 16 }} />
        <Form form={reviewForm} layout="vertical"><Form.Item name="notes" label="复核说明" rules={[{ required: true }]}><Input.TextArea rows={4} placeholder="写明核对项、疑点和结论" /></Form.Item></Form>
      </Modal>

      <Modal title="提交草稿审批" open={!!approvalTarget} onCancel={() => setApprovalTarget(null)} footer={null}>
        <Form layout="vertical" onFinish={(v) => approvalTarget && submitApproval.mutate({ id: approvalTarget.id, reason: v.reason })}><Form.Item name="reason" label="提交理由" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item><Button htmlType="submit" type="primary" loading={submitApproval.isPending}>提交 Owner 审批（不发布）</Button></Form>
      </Modal>

      <Modal title="Owner 草稿审批" open={!!decisionTarget} onCancel={() => setDecisionTarget(null)} footer={null}>
        <Form form={decisionForm} layout="vertical"><Form.Item name="note" label="审批说明" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item><Space><Button type="primary" loading={decideApproval.isPending} onClick={() => decisionForm.validateFields().then(({ note }) => decideApproval.mutate({ action: 'approve', note }))}>批准内部草稿</Button><Button danger loading={decideApproval.isPending} onClick={() => decisionForm.validateFields().then(({ note }) => decideApproval.mutate({ action: 'reject', note }))}>退回编辑</Button></Space></Form>
      </Modal>

      <Modal forceRender title={`生成待上架草稿 · 采集 #${converting?.id ?? ''}`} open={!!converting} width={1120} onCancel={() => setConverting(null)} onOk={() => convertForm.validateFields().then((v) => convert.mutate(v))} confirmLoading={convert.isPending} okText="保存并预览草稿">
        <Alert type="warning" showIcon title="不会自动发布" description="此动作只创建产品记录和渠道草稿。任何平台发布必须走单独的 Owner 批准。" style={{ marginBottom: 16 }} />
        <Form form={convertForm} layout="vertical">
          <Collapse defaultActiveKey={['basic', 'sku']} items={[
            { key: 'basic', label: '1. 产品、本地化与渠道', children: <>
              <Row gutter={12}>
                <Col xs={24} md={6}><Form.Item name="platform_id" label="已批准销售渠道 ID" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item name="category_id" label="渠道类目 ID" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item name="platform_sku" label="渠道主 SKU" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item name="unit" label="计量单位" rules={required}><Input /></Form.Item></Col>
              </Row>
              <Form.Item name="title" label="产品库中文标题" rules={required}><Input /></Form.Item>
              <Form.Item name="description" label="产品库中文说明" rules={required}><Input.TextArea rows={2} /></Form.Item>
              <Row gutter={12}>
                <Col xs={24} md={8}><Form.Item name="target_locale" label="目标语言/地区" rules={required}><Input placeholder="例如 ru-RU" /></Form.Item></Col>
                <Col xs={24} md={16}><Form.Item name="localized_title" label="本地化标题" rules={required}><Input /></Form.Item></Col>
              </Row>
              <Form.Item name="localized_description" label="本地化说明" rules={required}><Input.TextArea rows={3} /></Form.Item>
              <Form.Item name="localized_bullet_points" label="本地化卖点"><Select mode="tags" tokenSeparators={[',']} placeholder="输入一条卖点后回车" /></Form.Item>
              <Form.Item name="localized_keywords" label="本地化关键词"><Select mode="tags" tokenSeparators={[',']} placeholder="输入关键词后回车" /></Form.Item>
              <Text strong>本地化属性</Text><KeyValueList name="localized_attributes" addText="添加本地化属性" />
              <Divider>本地化规则证据</Divider>
              <Row gutter={12}>
                <Col xs={24} md={8}><Form.Item name="localization_rule_source_uri" label="规则来源" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="localization_rule_truth_status" label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="localization_rule_observed_at" label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
              </Row>
              <Row gutter={12}>
                <Col xs={24} md={6}><Form.Item name="allowed_scripts" label="允许文字脚本" rules={required}><Select mode="tags" options={[{ value: 'cyrillic' }, { value: 'latin' }, { value: 'han' }, { value: 'arabic' }]} /></Form.Item></Col>
                <Col xs={12} md={3}><Form.Item name="min_title_length" label="标题最短" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={3}><Form.Item name="max_title_length" label="标题最长" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={3}><Form.Item name="min_bullet_points" label="卖点最少" rules={required}><InputNumber min={0} /></Form.Item></Col>
                <Col xs={12} md={3}><Form.Item name="max_bullet_length" label="卖点最长" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={3}><Form.Item name="min_keywords" label="关键词最少" rules={required}><InputNumber min={0} /></Form.Item></Col>
                <Col xs={24} md={3}><Form.Item name="prohibited_words" label="禁用词"><Select mode="tags" /></Form.Item></Col>
              </Row>
            </> },
            { key: 'sku', label: '2. SKU 三段映射与变体', children: <>
              <Alert type="info" showIcon title="一行代表一个可销售变体" description="供应商 SKU、内部 SKU、渠道 SKU 必须唯一；颜色、尺寸、材质、包装会自动写入 SKU 属性和校验快照。" style={{ marginBottom: 12 }} />
              <Form.List name="sku_variants">{(fields, { add, remove }) => <Space orientation="vertical" style={{ width: '100%' }}>
                {fields.map((field, index) => <Card key={field.key} size="small" title={`SKU ${index + 1}`} extra={fields.length > 1 && <Button danger type="text" aria-label={`删除 SKU ${index + 1}`} icon={<DeleteOutlined />} onClick={() => remove(field.name)} />}>
                  <Row gutter={12}>
                    <Col xs={24} md={8}><Form.Item name={[field.name, 'supplier_sku']} label="供应商 SKU" rules={required}><Input /></Form.Item></Col>
                    <Col xs={24} md={8}><Form.Item name={[field.name, 'internal_sku']} label="凌镜内部 SKU" rules={required}><Input /></Form.Item></Col>
                    <Col xs={24} md={8}><Form.Item name={[field.name, 'channel_sku']} label="销售渠道 SKU" rules={required}><Input /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'color']} label="颜色" rules={required}><Input /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'size']} label="尺寸" rules={required}><Input /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'material']} label="材质" rules={required}><Input /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'packaging']} label="包装" rules={required}><Input /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'cost_price']} label="采购成本" rules={required}><InputNumber min={0.01} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'price']} label="计划售价" rules={required}><InputNumber min={0.01} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'weight']} label="重量（kg）"><InputNumber min={0} precision={3} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'image']} label="变体图片 URL"><Input /></Form.Item></Col>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'truth_status']} label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                    <Col xs={24} md={10}><Form.Item name={[field.name, 'source_uri']} label="SKU 来源" rules={required}><Input /></Form.Item></Col>
                    <Col xs={24} md={8}><Form.Item name={[field.name, 'observed_at']} label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                  </Row>
                </Card>)}
                <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ truth_status: 'quoted', observed_at: localDateTime() })}>添加 SKU</Button>
              </Space>}</Form.List>
              <Divider>SKU 规则证据</Divider>
              <Row gutter={12}>
                <Col xs={24} md={8}><Form.Item name="sku_rule_source_uri" label="规则来源" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="sku_rule_truth_status" label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="sku_rule_observed_at" label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
              </Row>
            </> },
            { key: 'media', label: '3. 图片权利、处理版本与质量', children: <>
              <Alert type="warning" showIcon title="必须先在“处理图片”中生成真实版本" description="这里记录处理结果和 Owner 目检。勾选“确认无水印/中文/品牌”表示你已实际看过处理后图片。" style={{ marginBottom: 12 }} />
              <Form.List name="media">{(fields, { add, remove }) => <Space orientation="vertical" style={{ width: '100%' }}>
                {fields.map((field, index) => <Card key={field.key} size="small" title={`图片 ${index + 1}`} extra={<Button danger type="text" aria-label={`删除图片 ${index + 1}`} icon={<DeleteOutlined />} onClick={() => remove(field.name)} />}>
                  <Row gutter={12}>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'processing_record_id']} label="处理记录 ID" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'media_role']} label="图片角色" rules={required}><Select options={[{ value: 'main', label: '主图' }, { value: 'gallery', label: '附图' }]} /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'width']} label="宽度" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={12} md={6}><Form.Item name={[field.name, 'height']} label="高度" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
                  </Row>
                  <Form.Item name={[field.name, 'source_url']} label="原图 URL" rules={required}><Input /></Form.Item>
                  <Form.Item name={[field.name, 'processed_url']} label="处理后内容 URL" rules={required}><Input /></Form.Item>
                  <Form.Item name={[field.name, 'content_sha256']} label="处理后 SHA-256" rules={required}><Input /></Form.Item>
                  <Form.Item name={[field.name, 'rights_observed_at_exact']} hidden><Input /></Form.Item>
                  <Row gutter={12}>
                    <Col xs={24} md={12}><Form.Item name={[field.name, 'rights_evidence_uri']} label="图片使用权证据" rules={required}><Input /></Form.Item></Col>
                    <Col xs={24} md={12}><Form.Item name={[field.name, 'rights_observed_at']} label="权利核验时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                    <Col xs={24} md={12}><Form.Item name={[field.name, 'channel_rule_uri']} label="渠道图片规则来源" rules={required}><Input /></Form.Item></Col>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'background']} label="背景" rules={required}><Select options={[{ value: 'white', label: '白底' }, { value: 'transparent', label: '透明' }]} /></Form.Item></Col>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'clarity_score']} label="清晰度 0-1" rules={required}><InputNumber min={0} max={1} step={0.05} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={24} md={12}><Form.Item name={[field.name, 'verification_observed_at']} label="Owner 目检时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                    <Col xs={24} md={12}><Form.Item name={[field.name, 'cropped']} valuePropName="checked"><Checkbox>已裁切并核对构图</Checkbox></Form.Item></Col>
                  </Row>
                  <Space wrap>
                    <Form.Item name={[field.name, 'no_watermark']} valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须实际确认无水印')) }]}><Checkbox>确认无水印</Checkbox></Form.Item>
                    <Form.Item name={[field.name, 'no_chinese_text']} valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须实际确认无中文')) }]}><Checkbox>确认无中文</Checkbox></Form.Item>
                    <Form.Item name={[field.name, 'no_brand_mark']} valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须实际确认无品牌标识')) }]}><Checkbox>确认无品牌标识</Checkbox></Form.Item>
                  </Space>
                </Card>)}
                <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ media_role: fields.length ? 'gallery' : 'main', background: 'white', cropped: true, clarity_score: 1, verification_observed_at: localDateTime() })}>添加处理记录</Button>
              </Space>}</Form.List>
              <Divider>图片标准规则</Divider>
              <Row gutter={12}>
                <Col xs={24} md={8}><Form.Item name="image_rule_source_uri" label="图片规则来源" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="image_rule_truth_status" label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="image_rule_observed_at" label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item name="allowed_backgrounds" label="允许背景" rules={required}><Select mode="tags" options={[{ value: 'white' }, { value: 'transparent' }]} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="min_clarity_score" label="最低清晰度" rules={required}><InputNumber min={0} max={1} step={0.05} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="image_rule_min_images" label="最少图片" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="image_rule_max_images" label="最多图片" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={6}><Form.Item name="require_crop" valuePropName="checked"><Checkbox>渠道要求裁切记录</Checkbox></Form.Item></Col>
              </Row>
            </> },
            { key: 'cost', label: '4. 完整成本、汇率与预计收入', children: <>
              <Alert type="info" showIcon title="10 项费用必须逐项留证" description="金额可以为 0，但来源和观察时间不能省略；汇率单独记录，不作为费用。" style={{ marginBottom: 12 }} />
              <Form.Item name="currency" label="统一核算币种" rules={required}><Select style={{ width: 160 }} options={[{ value: 'CNY' }, { value: 'RUB' }, { value: 'USD' }]} /></Form.Item>
              <Form.List name="costs">{(fields) => <Space orientation="vertical" style={{ width: '100%' }}>
                {fields.map((field, index) => <Card key={field.key} size="small" title={costTypes[index]?.[1] ?? `费用 ${index + 1}`}>
                  <Form.Item name={[field.name, 'cost_type']} hidden><Input /></Form.Item>
                  <Row gutter={12}>
                    <Col xs={12} md={4}><Form.Item name={[field.name, 'amount']} label="金额" rules={required}><InputNumber min={0} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
                    <Col xs={12} md={4}><Form.Item name={[field.name, 'currency']} label="币种" rules={required}><Select options={[{ value: 'CNY' }, { value: 'RUB' }, { value: 'USD' }]} /></Form.Item></Col>
                    <Col xs={24} md={5}><Form.Item name={[field.name, 'truth_status']} label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                    <Col xs={24} md={5}><Form.Item name={[field.name, 'observed_at']} label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                    <Col xs={24} md={6}><Form.Item name={[field.name, 'source_uri']} label="证据来源" rules={required}><Input /></Form.Item></Col>
                  </Row>
                </Card>)}
              </Space>}</Form.List>
              <Divider>汇率（跨币种时必填）</Divider>
              <Form.List name="exchange_rates">{(fields, { add, remove }) => <Space orientation="vertical" style={{ width: '100%' }}>
                {fields.map((field) => <Row key={field.key} gutter={12}>
                  <Col xs={12} md={3}><Form.Item name={[field.name, 'from_currency']} label="源币种" rules={required}><Input /></Form.Item></Col>
                  <Col xs={12} md={3}><Form.Item name={[field.name, 'to_currency']} label="目标币种" rules={required}><Input /></Form.Item></Col>
                  <Col xs={12} md={3}><Form.Item name={[field.name, 'rate']} label="汇率" rules={required}><InputNumber min={0.000001} style={{ width: '100%' }} /></Form.Item></Col>
                  <Col xs={12} md={4}><Form.Item name={[field.name, 'truth_status']} label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                  <Col xs={24} md={5}><Form.Item name={[field.name, 'source_uri']} label="来源" rules={required}><Input /></Form.Item></Col>
                  <Col xs={20} md={5}><Form.Item name={[field.name, 'observed_at']} label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                  <Col xs={4} md={1}><Button aria-label="删除汇率" danger type="text" icon={<DeleteOutlined />} onClick={() => remove(field.name)} /></Col>
                </Row>)}
                <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ truth_status: 'quoted', observed_at: localDateTime() })}>添加汇率</Button>
              </Space>}</Form.List>
              <Divider>预计销售收入</Divider>
              <Row gutter={12}>
                <Col xs={12} md={4}><Form.Item name="revenue_amount" label="收入金额" rules={required}><InputNumber min={0.01} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="revenue_currency" label="币种" rules={required}><Select options={[{ value: 'CNY' }, { value: 'RUB' }, { value: 'USD' }]} /></Form.Item></Col>
                <Col xs={24} md={5}><Form.Item name="revenue_truth_status" label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
                <Col xs={24} md={6}><Form.Item name="revenue_source_uri" label="定价依据" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={5}><Form.Item name="revenue_observed_at" label="观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
              </Row>
            </> },
            { key: 'supplier', label: '5. 供应商判断', children: <EvidenceChecklist name="supplier_assessment" types={supplierCheckTypes} /> },
            { key: 'compliance', label: '6. 合规检查', children: <>
              <Alert type="error" showIcon title="合规项只接受 actual" description="每一项都必须由 Owner 实际核验并留证，来源原文或 AI 推断不能替代合规结论。" style={{ marginBottom: 12 }} />
              <EvidenceChecklist name="compliance_checks" types={complianceCheckTypes} actualOnly />
            </> },
            { key: 'category', label: '7. 类目属性、变体规则与配送模板', children: <>
              <Row gutter={12}>
                <Col xs={24} md={8}><Form.Item name="category_schema_uri" label="渠道类目规则来源" rules={required}><Input /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="category_observed_at" label="规则观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="channel_rule_truth_status" label="证据等级" rules={required}><Select options={truthOptions} /></Form.Item></Col>
              </Row>
              <Text strong>渠道类目属性</Text><KeyValueList name="category_attributes" addText="添加类目属性" />
              <Row gutter={12} style={{ marginTop: 12 }}>
                <Col xs={24} md={8}><Form.Item name="required_category_attributes" label="规则要求的属性名"><Select mode="tags" placeholder="输入属性名后回车" /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="variant_dimensions" label="本商品变体维度" rules={required}><Select mode="tags" /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="required_variant_dimensions" label="规则要求的变体维度"><Select mode="tags" /></Form.Item></Col>
                <Col xs={24} md={8}><Form.Item name="allowed_variant_dimensions" label="规则允许的变体维度" rules={required}><Select mode="tags" /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="min_images" label="最少图片" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="max_images" label="最多图片" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="min_image_width" label="最小宽度" rules={required}><InputNumber min={1} /></Form.Item></Col>
                <Col xs={12} md={4}><Form.Item name="min_image_height" label="最小高度" rules={required}><InputNumber min={1} /></Form.Item></Col>
              </Row>
              <Form.Item name="shipping_template_id" label="已核验配送模板 ID" rules={required}><Input /></Form.Item>
              <Alert type="success" showIcon title="上架 payload 自动生成" description="系统会把上述类目属性、变体维度和配送模板组成渠道 payload，并生成同源校验快照，无需再维护 JSON。" />
            </> },
          ]} />
        </Form>
      </Modal>

      <Drawer
        title={`发布安全 · 采集 #${publishTarget?.id ?? ''}`}
        open={!!publishTarget}
        size={960}
        onClose={() => setPublishTarget(null)}
        extra={<Tag color="red">真实平台写入 · 高风险</Tag>}
      >
        <Alert
          type="error"
          showIcon
          icon={<ExclamationCircleOutlined />}
          title="四个步骤必须由 Owner 分别操作"
          description="①冻结请求与全部 SKU 库存 → ②独立批准或拒绝 → ③批准后再次点击执行外部写 → ④仅在结果不明确时用 actual 证据对账。任何成功提示都不会自动进入下一步。"
          style={{ marginBottom: 16 }}
        />

        {publishAttempts.isLoading ? <Card loading /> : publishAttempts.isError
          ? <Alert type="error" showIcon title="无法确认既有发布请求，已停止新建" description={(publishAttempts.error as Error).message} />
          : canCreatePublishRequest ? <Card size="small" title="步骤 1 · 冻结发布请求（不调用平台）">
          <Form form={publishRequestForm} layout="vertical" onFinish={(values) => requestPublish.mutate(values)}>
            <Row gutter={12}>
              <Col xs={24} md={8}><Form.Item name="platform_account_id" label="平台账号 ID" rules={required}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item></Col>
              <Col xs={24} md={16}><Form.Item name="idempotency_key" label="发布幂等键" extra="同一次逻辑发布永久使用同一个键；不要复制到其他商品。" rules={[...required, { max: 160 }]}><Input /></Form.Item></Col>
            </Row>
            <Form.Item name="reason" label="为什么现在要发布" rules={required}><Input.TextArea rows={2} placeholder="说明实验目标、渠道和本次发布边界" /></Form.Item>
            <Text strong>冻结全部内部 SKU 库存</Text>
            <Paragraph type="secondary">SKU 来自已批准草稿，不能增删；每个 SKU 都必须填写非负库存。</Paragraph>
            <Form.List name="inventory_rows">{(fields) => fields.length === 0
              ? <Alert type="warning" showIcon title="尚未读取到 SKU" description="请关闭后重新打开发布安全区；没有完整 SKU 时后端也会拒绝请求。" />
              : <Row gutter={12}>{fields.map((field) => <Col xs={24} md={12} key={field.key}>
                <Card size="small">
                  <Row gutter={8}>
                    <Col span={15}><Form.Item name={[field.name, 'sku_code']} label="内部 SKU" rules={required}><Input readOnly /></Form.Item></Col>
                    <Col span={9}><Form.Item name={[field.name, 'quantity']} label="发布库存" rules={required}><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                  </Row>
                </Card>
              </Col>)}</Row>}
            </Form.List>
            <Divider />
            <Button type="primary" htmlType="submit" disabled={publishInventoryRows.length === 0} loading={requestPublish.isPending}>冻结请求并提交独立审批</Button>
            <Text type="secondary" style={{ marginLeft: 12 }}>此按钮不会调用平台。</Text>
          </Form>
        </Card> : <Alert type="info" showIcon title="当前已有不可并行的新请求" description="只有既有请求明确 rejected 或 failed 后，才显示新的发布请求表单。" />}

        <Divider>冻结请求与结果</Divider>
        {publishAttempts.isLoading ? <Card loading /> : publishAttempts.isError ? <Alert type="error" showIcon title="发布记录加载失败" description={(publishAttempts.error as Error).message} /> : attempts.length === 0 ? <Alert type="info" showIcon title="还没有发布请求" description="先完成上方步骤 1。" /> : <Space orientation="vertical" style={{ width: '100%' }}>
          {attempts.map((attempt) => {
            const meta = publishStatusMeta[attempt.status] ?? { color: 'default', label: attempt.status, description: '未知状态，请停止操作并核对后端记录。' };
            return <Card
              key={attempt.id}
              size="small"
              title={<Space wrap><Text strong>发布请求 #{attempt.id}</Text><Tag color={meta.color}>{meta.label}</Tag></Space>}
              extra={<Space wrap>
                {attempt.status === 'pending_approval' && <Button size="small" onClick={() => { publishDecisionForm.resetFields(); setPublishDecisionTarget(attempt); }}>步骤 2 · 独立审批</Button>}
                {attempt.status === 'approved' && <Button size="small" danger onClick={() => { publishExecuteForm.resetFields(); setPublishExecuteTarget(attempt); }}>步骤 3 · 执行外部写</Button>}
                {attempt.status === 'reconcile_required' && <Button size="small" onClick={() => { publishReconcileForm.resetFields(); publishReconcileForm.setFieldsValue({ outcome: 'submitted', truth_status: 'actual', observed_at: localDateTime(), published_data: [] }); setPublishReconcileTarget(attempt); }}>步骤 4 · actual 对账</Button>}
              </Space>}
            >
              <Alert type={attempt.status === 'submitted' ? 'warning' : 'info'} showIcon title={meta.label} description={meta.description} style={{ marginBottom: 12 }} />
              <Descriptions bordered size="small" column={{ xs: 1, md: 2 }}>
                <Descriptions.Item label="平台 / 账号">#{attempt.platform_id} / #{attempt.platform_account_id}</Descriptions.Item>
                <Descriptions.Item label="审批记录">#{attempt.approval_id ?? '尚未创建'}</Descriptions.Item>
                <Descriptions.Item label="幂等键"><Text copyable>{attempt.idempotency_key}</Text></Descriptions.Item>
                <Descriptions.Item label="请求 SHA-256"><Text copyable>{attempt.request_sha256}</Text></Descriptions.Item>
                <Descriptions.Item label="请求时间">{attempt.requested_at}</Descriptions.Item>
                <Descriptions.Item label="执行 / 完成">{attempt.executed_at ?? '未执行'} / {attempt.completed_at ?? '未完成'}</Descriptions.Item>
                {attempt.error_message && <Descriptions.Item label="安全错误分类" span={2}><Text type="danger">{attempt.error_message}</Text></Descriptions.Item>}
                <Descriptions.Item label="冻结请求" span={2}><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0, maxHeight: 260, overflow: 'auto' }}>{formatPayload(attempt.request_payload)}</Paragraph></Descriptions.Item>
                <Descriptions.Item label="适配器请求快照" span={2}><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0, maxHeight: 260, overflow: 'auto' }}>{formatPayload(attempt.adapter_request_payload)}</Paragraph></Descriptions.Item>
                {(attempt.response_payload != null || attempt.response_sha256) && <Descriptions.Item label="平台返回 / 对账结果" span={2}>
                  <Space orientation="vertical" style={{ width: '100%' }}>
                    <Text>响应 SHA-256：<Text copyable>{attempt.response_sha256 || '—'}</Text></Text>
                    <Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0, maxHeight: 260, overflow: 'auto' }}>{formatPayload(attempt.response_payload)}</Paragraph>
                  </Space>
                </Descriptions.Item>}
              </Descriptions>
            </Card>;
          })}
        </Space>}
      </Drawer>

      <Modal title={`步骤 2 · 独立 Owner 审批发布请求 #${publishDecisionTarget?.id ?? ''}`} open={!!publishDecisionTarget} onCancel={() => setPublishDecisionTarget(null)} footer={null}>
        <Alert type="warning" showIcon title="批准不会调用平台" description="请先核对冻结请求、平台账号、所有 SKU 库存和请求 SHA。批准后仍需回到发布安全区，再次手动执行。" style={{ marginBottom: 16 }} />
        <Form form={publishDecisionForm} layout="vertical">
          <Form.Item name="note" label="审批说明" rules={required}><Input.TextArea rows={3} /></Form.Item>
          <Space>
            <Button type="primary" loading={decidePublish.isPending} onClick={() => publishDecisionForm.validateFields().then(({ note }) => decidePublish.mutate({ action: 'approve', note }))}>批准发布请求（仍不执行）</Button>
            <Button danger loading={decidePublish.isPending} onClick={() => publishDecisionForm.validateFields().then(({ note }) => decidePublish.mutate({ action: 'reject', note }))}>拒绝发布请求</Button>
          </Space>
        </Form>
      </Modal>

      <Modal title={`步骤 3 · 执行真实平台写入 #${publishExecuteTarget?.id ?? ''}`} open={!!publishExecuteTarget} onCancel={() => setPublishExecuteTarget(null)} footer={null}>
        <Alert
          type="error"
          showIcon
          icon={<ExclamationCircleOutlined />}
          title="下一次点击会调用外部平台"
          description="这可能创建真实商品并占用平台幂等键。网络超时不代表失败，系统不会自动重试；结果不明确时必须走后置对账。"
          style={{ marginBottom: 16 }}
        />
        <Descriptions bordered size="small" column={1}>
          <Descriptions.Item label="平台账号">#{publishExecuteTarget?.platform_account_id}</Descriptions.Item>
          <Descriptions.Item label="幂等键"><Text copyable>{publishExecuteTarget?.idempotency_key}</Text></Descriptions.Item>
          <Descriptions.Item label="冻结请求 SHA"><Text copyable>{publishExecuteTarget?.request_sha256}</Text></Descriptions.Item>
        </Descriptions>
        <Divider />
        <Form form={publishExecuteForm} layout="vertical" onFinish={() => executePublish.mutate()}>
          <Form.Item name="confirmed" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须明确确认外部写风险')) }]}>
            <Checkbox>我已核对独立审批和冻结请求，确认现在执行真实平台发布</Checkbox>
          </Form.Item>
          <Button danger type="primary" htmlType="submit" loading={executePublish.isPending}>确认执行外部发布</Button>
        </Form>
      </Modal>

      <Modal title={`步骤 4 · 发布结果后置对账 #${publishReconcileTarget?.id ?? ''}`} open={!!publishReconcileTarget} width={760} onCancel={() => setPublishReconcileTarget(null)} footer={null}>
        <Alert type="warning" showIcon title="只记录已在平台实际观察到的结果" description="必须提供 actual 证据。选择 submitted 只表示平台侧存在可识别商品，不等于已经审核通过、可见或可销售。" style={{ marginBottom: 16 }} />
        <Form form={publishReconcileForm} layout="vertical" onFinish={(values) => reconcilePublish.mutate(values)}>
          <Row gutter={12}>
            <Col xs={24} md={8}><Form.Item name="outcome" label="实际观察结果" rules={required}><Select options={[{ value: 'submitted', label: '平台存在商品记录' }, { value: 'failed', label: '平台确认失败/不存在' }]} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name="truth_status" label="证据等级"><Select disabled options={[{ value: 'actual', label: 'actual · Owner 已核验' }]} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name="observed_at" label="平台观察时间" rules={required}><Input type="datetime-local" /></Form.Item></Col>
          </Row>
          <Form.Item name="evidence_uri" label="平台证据 URI" rules={required}><Input placeholder="平台后台 URL、截图证据或审计文件位置" /></Form.Item>
          {reconcileOutcome === 'submitted' && <>
            <Row gutter={12}>
              <Col xs={24} md={12}><Form.Item name="platform_product_id" label="平台商品 ID" rules={required}><Input /></Form.Item></Col>
              <Col xs={24} md={12}><Form.Item name="platform_sku" label="平台主 SKU"><Input /></Form.Item></Col>
            </Row>
            <Form.Item name="platform_url" label="平台商品 URL"><Input /></Form.Item>
            <Form.Item name="sync_message" label="平台返回说明"><Input.TextArea rows={2} /></Form.Item>
            <Text strong>平台返回字段（可选）</Text>
            <KeyValueList name="published_data" addText="添加平台返回字段" />
          </>}
          <Divider />
          <Button type="primary" htmlType="submit" loading={reconcilePublish.isPending}>保存 actual 对账结果</Button>
        </Form>
      </Modal>

      <Drawer title="待上架草稿预览（未发布）" open={!!preview} size={720} onClose={() => setPreview(null)} extra={<Tag color="orange">不会自动发布</Tag>}>
        <Alert type="success" showIcon title="草稿已保存" description="下方内容用于 Owner 人工验收；它不证明平台已接受，也没有触发外部发布。" />
        <Divider />
        <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="追溯链">采集 → 快照 → 市场/实验 → 产品 → 渠道草稿</Descriptions.Item>
          <Descriptions.Item label="后端返回"><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{JSON.stringify(preview, null, 2)}</Paragraph></Descriptions.Item>
        </Descriptions>
        <Divider />
        <Button icon={<EyeOutlined />} disabled>仅预览；发布必须从列表“发布安全”进入</Button>
      </Drawer>

      <Drawer title="不可变采集证据" open={!!evidence} size={720} onClose={() => setEvidence(null)}>
        {evidence && <Descriptions bordered column={1} size="small">
          <Descriptions.Item label="原链接"><a href={evidence.source_url} target="_blank" rel="noreferrer">{evidence.source_url}</a></Descriptions.Item>
          <Descriptions.Item label="采集时间">{evidence.collected_at}</Descriptions.Item>
          <Descriptions.Item label="驱动 / 解析版本">{evidence.driver} / {evidence.parser_version}</Descriptions.Item>
          <Descriptions.Item label="本次解析字段">{evidence.observed_title || '—'} / {evidence.observed_supplier || '—'} / ¥{evidence.observed_price ?? '未知'} / MOQ {evidence.observed_moq ?? '未知'}</Descriptions.Item>
          <Descriptions.Item label="SHA-256"><Text copyable>{evidence.raw_sha256}</Text></Descriptions.Item>
          <Descriptions.Item label="原始 payload"><Paragraph copyable style={{ whiteSpace: 'pre-wrap', margin: 0 }}>{JSON.stringify(evidence.raw_payload, null, 2)}</Paragraph></Descriptions.Item>
        </Descriptions>}
      </Drawer>

      <Drawer title="版本变化与疑似同款" open={!!identityHistory} size={760} onClose={() => setIdentityHistory(null)}>
        <Alert type="info" showIcon title="疑似同款必须由 Owner 裁决" description="系统按 1688 offer ID 和内容指纹发现重复、换链接或跨供应商同款；不会自动合并商品。" style={{ marginBottom: 16 }} />
        <Table rowKey="id" pagination={false} dataSource={identityHistory?.data.duplicates ?? []} columns={[{ title: '来源', render: (_, r) => `#${r.source_product_id} ↔ #${r.matched_product_id}` }, { title: '匹配方式', dataIndex: 'match_type' }, { title: '状态', dataIndex: 'status' }, { title: 'Owner 裁决', render: (_, r) => r.status === 'pending_review' ? <Space><Button size="small" loading={resolveDuplicate.isPending} onClick={() => resolveDuplicate.mutate({ id: r.id, decision: 'same_product' })}>确认同款</Button><Button size="small" onClick={() => resolveDuplicate.mutate({ id: r.id, decision: 'different_product' })}>不是同款</Button></Space> : '已裁决' }]} />
        <Divider />
        <Paragraph copyable style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify({ snapshots: identityHistory?.data.snapshots, changes: identityHistory?.data.changes }, null, 2)}</Paragraph>
      </Drawer>

      <Drawer
        title={`真实商品 15 项验收 · 采集 #${acceptanceReport?.sourcing_product_id ?? ''}`}
        open={!!acceptanceReport}
        size={920}
        onClose={() => setAcceptanceReport(null)}
        extra={<Tag color={acceptanceReport?.ready ? 'green' : acceptanceReport?.status === 'blocked' ? 'red' : 'gold'}>{acceptanceReport?.ready ? '全部通过' : acceptanceReport?.status === 'blocked' ? '存在阻断' : '证据未知'}</Tag>}
      >
        {acceptanceReport && <>
          <Alert type={acceptanceReport.ready ? 'success' : 'warning'} showIcon title={acceptanceReport.ready ? '该商品的持久化真实证据链已逐项通过' : '不能宣布真实商品闭环完成'} description={acceptanceReport.disclaimer} style={{ marginBottom: 16 }} />
          <Table<AcceptanceItem>
            rowKey="code"
            pagination={false}
            dataSource={acceptanceReport.items}
            columns={[
              { title: '#', dataIndex: 'number', width: 48 },
              { title: '验收项', dataIndex: 'title', width: 180 },
              { title: '状态', dataIndex: 'status', width: 100, render: (status: AcceptanceItem['status']) => <Tag color={status === 'passed' ? 'green' : status === 'blocked' ? 'red' : 'gold'}>{status}</Tag> },
              { title: '事实与阻断', render: (_, item) => <><Text>{item.summary}</Text>{item.blockers?.length > 0 && <ul style={{ marginBottom: 0 }}>{item.blockers.map((blocker) => <li key={blocker}><Text type="danger">{blocker}</Text></li>)}</ul>}</> },
            ]}
          />
          <Paragraph type="secondary" style={{ marginTop: 16 }}>报告生成时间：{acceptanceReport.generated_at}</Paragraph>
        </>}
      </Drawer>
    </PageContainer>
  );
}
