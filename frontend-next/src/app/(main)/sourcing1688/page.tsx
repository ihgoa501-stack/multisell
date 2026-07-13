'use client';

import { useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, App as AntdApp, Button, Card, Checkbox, Col, Collapse, Descriptions, Divider, Drawer, Form, Image, Input, InputNumber,
  Modal, Popconfirm, Row, Select, Space, Table, Tag, Timeline, Typography, Upload,
} from 'antd';
import { DeleteOutlined, ExclamationCircleOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, UploadOutlined } from '@ant-design/icons';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import { getToken } from '@/lib/auth';
import SourcingAuthorityWorkspace from '@/features/sourcing1688/SourcingAuthorityWorkspace';
import SourceWatchWorkspace from '@/features/sourcing1688/SourceWatchWorkspace';

const { Text, Paragraph } = Typography;
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

function protectedContentURL(path: string) {
  if (/^https?:\/\//i.test(path)) return path;
  return `${API_BASE.replace(/\/api\/?$/, '')}${path.startsWith('/') ? path : `/${path}`}`;
}

type SourceRecord = {
  id: number;
  owner_id: number;
  source_url: string;
  title?: string;
  price?: number;
  moq: number;
  supplier_name?: string;
  supplier_id?: number;
  status: string;
  demand_case_id?: number;
  experiment_id?: string;
  snapshot_id?: number;
  reviewed_by?: number;
  reviewed_at?: string;
  review_notes?: string;
  product_id?: number;
	lifecycle_status?: string;
	field_statuses?: Record<string, string>;
	observation_count?: number;
	task_link_count?: number;
	latest_page_kind?: CollectionPageKind;
	latest_observed_at?: string;
	created_at: string;
	updated_at: string;
};

type CollectionPageKind = 'list_lead' | 'detail_observation' | 'controlled_fetch';
type QualityFieldObservation = {
	field: string;
	status: 'observed' | 'unknown' | 'parse_failed' | 'no_sku';
	value_summary?: unknown;
	source: { page_kind: CollectionPageKind; snapshot_id: number; observed_at: string; parser: string };
};
type QualityObservation = {
	snapshot_id: number;
	page_kind: CollectionPageKind;
	observed_at: string;
	parser: string;
	fields: Record<string, QualityFieldObservation>;
};
type CollectionQuality = {
	sourcing_product_id: number;
	source_url: string;
	observations: QualityObservation[];
	latest_list_observation?: QualityObservation;
	latest_detail_observation?: QualityObservation;
	latest_controlled_observation?: QualityObservation;
	best_fields: Record<string, QualityFieldObservation>;
	conflicts: Array<{ field: string; values: QualityFieldObservation[]; message: string }>;
	missing: string[];
	recapture_action: { kind: 'none' | 'open_detail_page' | 'retry_detail_collection'; url: string; reason: string };
};
type PreciseCostVersion = { version: { id: number; task_link_id: number; version: number; contribution_profit_minor: number; target_currency: string; content_hash: string } };

const pageKindMeta: Record<CollectionPageKind, { label: string; color: string }> = {
	list_lead: { label: '列表线索', color: 'default' },
	detail_observation: { label: '详情观察', color: 'blue' },
	controlled_fetch: { label: '受控详情', color: 'purple' },
};

export function collectionRecordIDFromSearch(search: string) {
	const value = Number(new URLSearchParams(search).get('record_id'));
	return Number.isInteger(value) && value > 0 ? value : null;
}

export function safe1688DetailURL(value?: string) {
	try {
		const url = new URL(value ?? '');
		return url.protocol === 'https:' && url.hostname === 'detail.1688.com' && /^\/offer\/\d+\.html$/i.test(url.pathname)
			? url.toString()
			: null;
	} catch {
		return null;
	}
}

export function collectionQualityRows(quality?: CollectionQuality) {
	if (!quality) return [];
	const fields = Array.from(new Set([
		...Object.keys(collectionFieldLabels),
		...Object.keys(quality.latest_list_observation?.fields ?? {}),
		...Object.keys(quality.latest_detail_observation?.fields ?? {}),
		...Object.keys(quality.best_fields ?? {}),
		...(quality.missing ?? []),
	]));
	return fields.map((field) => ({
		field,
		label: collectionFieldLabels[field] ?? field,
		list: quality.latest_list_observation?.fields?.[field],
		detail: quality.latest_detail_observation?.fields?.[field],
		best: quality.best_fields?.[field],
		conflict: quality.conflicts?.find((item) => item.field === field),
		missing: quality.missing?.includes(field) || !quality.best_fields?.[field],
	}));
}

function qualityValue(item?: QualityFieldObservation) {
	if (!item) return <Text type="secondary">未观察</Text>;
	if (item.status !== 'observed' && item.status !== 'no_sku') {
		return <Tag color={item.status === 'parse_failed' ? 'red' : 'default'}>{item.status === 'parse_failed' ? '解析失败' : '未知'}</Tag>;
	}
	const value = item.status === 'no_sku' ? '页面明确无 SKU' : item.value_summary;
	return <Space orientation="vertical" size={0}>
		<Text>{typeof value === 'string' || typeof value === 'number' ? String(value) : JSON.stringify(value ?? '已取得')}</Text>
		<Text type="secondary">快照 #{item.source.snapshot_id} · {new Date(item.source.observed_at).toLocaleString('zh-CN')}</Text>
	</Space>;
}

type EligibleTask = {
  experiment_id: string;
  demand_case_id: number;
  label: string;
  region: string;
  consumer: string;
  need_scenario: string;
  sales_channel: string;
	product_opportunity_id: number;
	opportunity_title: string;
};

type TaskLink = {
  id: number;
  experiment_id: string;
  demand_case_id: number;
  product_opportunity_id?: number;
  opportunity_decision_id?: number;
  status: string;
  is_primary: boolean;
  current_status: string;
  current_blocker?: string;
  workflow_status: string;
  draft_id?: number;
  workflow_updated_at?: string;
  sample_policy?: 'required' | 'waived';
  sample_waiver_reason?: string;
  sample_waived_by?: number;
  sample_waived_at?: string;
  label: string;
  created_at: string;
};

export type SampleStatus = 'request' | 'approved_to_order' | 'ordered' | 'received' | 'evaluated' | 'accepted' | 'rejected';
type SampleEvent = {
  id: number;
  from_status: string;
  to_status: SampleStatus;
  order_amount?: number;
  currency?: string;
  external_credential_uri?: string;
  observed_at?: string;
  truth_status: string;
  note?: string;
  created_at: string;
};
type Sample = {
  id: number;
  task_link_id: number;
  product_opportunity_id: number;
  opportunity_decision_id: number;
  supplier_id: number;
  snapshot_id: number;
  supplier_sku?: string;
  quantity: number;
  status: SampleStatus;
  order_amount?: number;
  currency?: string;
  external_credential_uri?: string;
  observed_at?: string;
  truth_status: string;
  evaluation?: string;
  created_at: string;
};
type SampleDetail = { sample: Sample; events: SampleEvent[] };

const sampleStatusMeta: Record<SampleStatus, { label: string; color: string }> = {
  request: { label: '样品申请', color: 'default' },
  approved_to_order: { label: 'Owner已批准下单（尚未下单）', color: 'blue' },
  ordered: { label: '已下单（actual凭证）', color: 'cyan' },
  received: { label: '已收货（actual凭证）', color: 'geekblue' },
  evaluated: { label: '已评估（actual凭证）', color: 'purple' },
  accepted: { label: '样品通过', color: 'green' },
  rejected: { label: '样品淘汰', color: 'red' },
};

export function sampleNextStatuses(status: SampleStatus): SampleStatus[] {
  return ({
    request: ['approved_to_order'],
    approved_to_order: ['ordered'],
    ordered: ['received'],
    received: ['evaluated'],
    evaluated: ['accepted', 'rejected'],
    accepted: [],
    rejected: [],
  } satisfies Record<SampleStatus, SampleStatus[]>)[status];
}

export function buildSampleTransitionPayload(status: SampleStatus, values: Record<string, unknown>) {
  const payload: Record<string, unknown> = { to_status: status, note: String(values.note ?? '').trim() };
  if (status !== 'approved_to_order') {
    payload.truth_status = 'actual';
    payload.external_credential_uri = String(values.external_credential_uri ?? '').trim();
    payload.observed_at = iso(values.observed_at);
  }
  if (status === 'ordered') {
    payload.order_amount = values.order_amount;
    payload.currency = String(values.currency ?? '').trim().toUpperCase();
  }
  return payload;
}

type DraftResult = {
  draft?: Record<string, unknown>;
  listing_draft?: Record<string, unknown>;
  product?: Record<string, unknown>;
  trace?: Record<string, unknown>;
  editable_payload?: Record<string, unknown>;
  editable_version?: number;
  editable_sha256?: string;
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
  task_link_id?: number;
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
	unverified_lead: '未验证线索', needs_review: '工作副本待复核', collected: '待复核', pending_review: '待复核', reviewed: '已复核', rejected: '已淘汰',
	ready_for_draft: '可转草稿', converting: '正在转草稿', converted_to_draft: '已转草稿',
	blocked: '已阻塞', conversion_failed: '转换失败', reconcile_required: '需要对账', archived: '已归档',
	converted: '已转产品', draft_created: '草稿已生成',
};
const statusColor: Record<string, string> = {
	unverified_lead: 'default', needs_review: 'gold', collected: 'orange', pending_review: 'orange', reviewed: 'blue', ready_for_draft: 'cyan', converting: 'processing', converted_to_draft: 'green', blocked: 'red', conversion_failed: 'red', reconcile_required: 'gold', archived: 'default', rejected: 'red', converted: 'cyan', draft_created: 'green',
};

const collectionFieldLabels: Record<string, string> = {
	title: '标题', price: '价格', moq: '起订量', supplier: '供应商身份', images: '主图', sku: 'SKU', attributes: '商品属性',
};

export function collectionCompleteness(fieldStatuses?: Record<string, string>) {
		const statuses = fieldStatuses ?? {};
		const fields = ['title', 'price', 'moq', 'supplier', 'images', 'sku'];
	const complete = fields.filter((field) => statuses[field] === 'observed' || (field === 'sku' && statuses[field] === 'no_sku'));
	const missing = fields.filter((field) => !complete.includes(field)).map((field) => ({
		field,
		label: collectionFieldLabels[field],
		status: statuses[field] || 'unknown',
	}));
	return { complete: complete.length, total: fields.length, missing, isComplete: missing.length === 0 };
}

const taskLinkStatusMeta: Record<string, { color: string; label: string }> = {
  needs_review: { color: 'gold', label: '任务待复核' },
  ready_for_draft: { color: 'cyan', label: '可转草稿' },
  converting: { color: 'processing', label: '正在转草稿' },
  converted_to_draft: { color: 'blue', label: '草稿已生成' },
  editing: { color: 'blue', label: '草稿编辑中' },
  pending_approval: { color: 'orange', label: '草稿待审批' },
  approved_draft: { color: 'green', label: '草稿已批准' },
  publish_pending: { color: 'orange', label: '发布待审批' },
  publish_approved: { color: 'blue', label: '发布已批准' },
  publishing: { color: 'processing', label: '发布执行中' },
  submitted: { color: 'cyan', label: '平台已接收' },
  succeeded: { color: 'green', label: '平台状态已确认' },
  publish_failed: { color: 'red', label: '发布失败' },
  conversion_failed: { color: 'red', label: '转换失败' },
  reconcile_required: { color: 'gold', label: '需要对账' },
  linked: { color: 'blue', label: '已关联' },
  active: { color: 'processing', label: '进行中' },
  blocked: { color: 'red', label: '已阻塞' },
  completed: { color: 'green', label: '已完成' },
  archived: { color: 'default', label: '已归档' },
};

export function taskLinkStatus(link: Pick<TaskLink, 'status' | 'current_status' | 'current_blocker'> & { workflow_status?: string }) {
  const status = link.workflow_status || link.current_status || link.status || 'linked';
  const blocked = status === 'blocked' || Boolean(link.current_blocker?.trim());
  if (blocked) return { color: 'red', label: '已阻塞', status };
  const meta = taskLinkStatusMeta[status] ?? { color: 'default', label: status };
  return { ...meta, status };
}

export function taskLinkAvailableActions(link: Pick<TaskLink, 'workflow_status' | 'draft_id' | 'current_blocker'>) {
  const status = link.workflow_status || 'needs_review';
  const blocked = status === 'blocked' || Boolean(link.current_blocker?.trim());
  return {
    convert: !blocked && status === 'ready_for_draft' && !link.draft_id,
    edit: !blocked && Boolean(link.draft_id) && ['converted_to_draft', 'editing'].includes(status),
    submitApproval: !blocked && Boolean(link.draft_id) && ['converted_to_draft', 'editing'].includes(status),
    decideApproval: !blocked && Boolean(link.draft_id) && status === 'pending_approval',
    publish: !blocked && Boolean(link.draft_id) && ['approved_draft', 'publish_pending', 'publish_approved', 'publishing', 'submitted', 'publish_failed', 'reconcile_required', 'succeeded'].includes(status),
  };
}

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

function objectToRows(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, unknown>).map(([name, item]) => ({ name, value: String(item ?? '') }));
}

/** Restores the exact Owner form from the server's canonical editable payload. */
export function editableDraftToForm(input: Record<string, unknown>): DraftFormValues {
  const record = (value: unknown): Record<string, unknown> => value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {};
  const rows = (value: unknown): Array<Record<string, unknown>> => Array.isArray(value) ? value.map(record) : [];
  const validation = record(input.validation);
  const localization = record(validation.localization);
  const localizationRules = record(validation.localization_rules);
  const channel = record(validation.channel);
  const channelRules = record(validation.channel_rules);
  const costValidation = record(validation.costs);
  const imageRules = record(validation.image_rules);
  const skuRules = record(validation.sku_rules);
  const listingPayload = record(input.listing_payload);
  const validationSKUs = rows(validation.skus);
  const validationImages = rows(validation.images);
  const evidenceFields = (prefix: string, value: Record<string, unknown>) => ({
    [`${prefix}_truth_status`]: value?.truth_status,
    [`${prefix}_source_uri`]: value?.source_uri,
    [`${prefix}_observed_at`]: localDateTime(value?.observed_at),
  });
  const skuVariants = rows(input.sku_variants).map((row, index) => ({
    ...row,
    ...(validationSKUs[index] ?? {}),
    observed_at: localDateTime(validationSKUs[index]?.observed_at),
  }));
  const media = rows(input.media).map((row, index) => {
    const checked = validationImages[index] ?? {};
    return {
      ...row,
      rights_observed_at: localDateTime(row.rights_observed_at),
      rights_observed_at_exact: row.rights_observed_at,
      background: checked.background,
      cropped: checked.cropped,
      clarity_score: checked.clarity_score,
      no_watermark: !row.has_watermark,
      no_chinese_text: !row.has_chinese_text,
      no_brand_mark: !row.has_brand_mark,
      verification_observed_at: localDateTime(checked.observed_at),
    };
  });
  return {
    ...input,
    currency: costValidation.target_currency,
    category_observed_at: localDateTime(input.category_observed_at),
    sku_variants: skuVariants,
    media,
    costs: rows(input.costs).map((row) => ({ ...row, observed_at: localDateTime(row.observed_at) })),
    supplier_assessment: rows(input.supplier_assessment).map((row) => ({ ...row, observed_at: localDateTime(row.observed_at) })),
    compliance_checks: rows(input.compliance_checks).map((row) => ({ ...row, observed_at: localDateTime(row.observed_at) })),
    localized_bullet_points: localization.bullet_points ?? [],
    localized_keywords: localization.keywords ?? [],
    localized_attributes: objectToRows(localization.attributes),
    allowed_scripts: localizationRules.allowed_scripts ?? [],
    prohibited_words: localizationRules.prohibited_words ?? [],
    min_title_length: localizationRules.min_title_length,
    max_title_length: localizationRules.max_title_length,
    min_bullet_points: localizationRules.min_bullet_points,
    max_bullet_length: localizationRules.max_bullet_length,
    min_keywords: localizationRules.min_keywords,
    ...evidenceFields('localization_rule', record(localizationRules.evidence)),
    category_attributes: objectToRows(listingPayload.attributes ?? channel.attributes),
    variant_dimensions: listingPayload.variant_dimensions ?? channel.variant_dimensions ?? [],
    required_category_attributes: channelRules.required_attributes ?? [],
    required_variant_dimensions: channelRules.required_variant_dimensions ?? [],
    allowed_variant_dimensions: channelRules.allowed_variant_dimensions ?? [],
    min_images: channelRules.min_images,
    max_images: channelRules.max_images,
    min_image_width: channelRules.min_image_width,
    min_image_height: channelRules.min_image_height,
    channel_rule_truth_status: record(channelRules.evidence).truth_status,
    exchange_rates: rows(costValidation.exchange_rates).map((row) => ({ ...row, observed_at: localDateTime(row.observed_at) })),
    revenue_amount: record(costValidation.revenue).amount,
    revenue_currency: record(costValidation.revenue).currency,
    revenue_truth_status: record(costValidation.revenue).truth_status,
    revenue_source_uri: record(costValidation.revenue).source_uri,
    revenue_observed_at: localDateTime(record(costValidation.revenue).observed_at),
    allowed_backgrounds: imageRules.allowed_backgrounds ?? [],
    require_crop: imageRules.require_crop,
    min_clarity_score: imageRules.min_clarity_score,
    image_rule_min_images: imageRules.min_images,
    image_rule_max_images: imageRules.max_images,
    ...evidenceFields('image_rule', record(imageRules.evidence)),
    ...evidenceFields('sku_rule', record(skuRules.evidence)),
    shipping_template_id: input.shipping_template_id ?? listingPayload.shipping_template_id,
  };
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

function newConversionRequestID(sourceID: number) {
	const random = globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2);
	return `convert_${sourceID}_${Date.now()}_${random}`;
}

export function taskWorkflowPath(sourceID: number, taskLinkID: number, suffix: string) {
  if (!Number.isInteger(sourceID) || sourceID <= 0 || !Number.isInteger(taskLinkID) || taskLinkID <= 0) {
    throw new Error('invalid sourcing task identity');
  }
  const normalized = suffix.replace(/^\/+/, '');
  return `/v1/sourcing-1688/${sourceID}/task-links/${taskLinkID}${normalized ? `/${normalized}` : ''}`;
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
	conversion_request_id: values.conversion_request_id,
    editable_version: values.editable_version,
    editable_sha256: values.editable_sha256,
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

type TaskLinkCardProps = {
  link: TaskLink;
  onDraft?: (link: TaskLink) => void;
  onSubmitApproval?: (link: TaskLink) => void;
  onDecideApproval?: (link: TaskLink) => void;
  onPublish?: (link: TaskLink) => void;
  onSamples?: (link: TaskLink) => void;
  onAuthority?: (link: TaskLink) => void;
};

function TaskLinkCard({ link, onDraft, onSubmitApproval, onDecideApproval, onPublish, onSamples, onAuthority }: TaskLinkCardProps) {
  const meta = taskLinkStatus(link);
  const actions = taskLinkAvailableActions(link);
  return (
    <Card
      size="small"
      style={{ marginBottom: 8 }}
      title={<Text>{link.label || `任务 ${link.experiment_id}`}</Text>}
      extra={<Space wrap>{link.is_primary && <Tag color="blue">主任务</Tag>}<Tag color={meta.color}>{meta.label}</Tag></Space>}
    >
      <Descriptions size="small" column={{ xs: 1, sm: 2 }}>
        <Descriptions.Item label="任务 ID">{link.experiment_id}</Descriptions.Item>
        <Descriptions.Item label="候选市场">#{link.demand_case_id}</Descriptions.Item>
        <Descriptions.Item label="商品机会">{link.product_opportunity_id ? `#${link.product_opportunity_id}` : '历史关联（无授权）'}</Descriptions.Item>
        <Descriptions.Item label="冻结批准">{link.opportunity_decision_id ? `#${link.opportunity_decision_id}` : '无'}</Descriptions.Item>
        <Descriptions.Item label="关联状态">{link.status || 'linked'}</Descriptions.Item>
        <Descriptions.Item label="独立工作流">{meta.label}</Descriptions.Item>
        <Descriptions.Item label="任务草稿">{link.draft_id ? `#${link.draft_id}` : '尚未生成'}</Descriptions.Item>
        <Descriptions.Item label="工作流更新时间">{link.workflow_updated_at ? new Date(link.workflow_updated_at).toLocaleString('zh-CN') : '未知'}</Descriptions.Item>
        <Descriptions.Item label="关联时间" span={2}>{link.created_at ? new Date(link.created_at).toLocaleString('zh-CN') : '未知'}</Descriptions.Item>
      </Descriptions>
      {link.current_blocker && (
        <Alert
          type="error"
          showIcon
          title="当前阻塞原因"
          description={link.current_blocker}
          style={{ marginTop: 12 }}
        />
      )}
      <Divider style={{ margin: '12px 0' }} />
      <Space wrap>
        {onSamples && <Button size="small" onClick={() => onSamples(link)}>样品事实链</Button>}
        {onAuthority && <Button size="small" onClick={() => onAuthority(link)}>精确成本 / 合规</Button>}
        {actions.convert && onDraft && <Button size="small" type="primary" onClick={() => onDraft(link)}>为此任务转草稿</Button>}
        {actions.edit && onDraft && <Button size="small" onClick={() => onDraft(link)}>编辑此任务草稿</Button>}
        {actions.submitApproval && onSubmitApproval && <Button size="small" onClick={() => onSubmitApproval(link)}>提交此任务审批</Button>}
        {actions.decideApproval && onDecideApproval && <Button size="small" onClick={() => onDecideApproval(link)}>审批此任务草稿</Button>}
        {actions.publish && onPublish && <Button size="small" danger icon={<SafetyCertificateOutlined />} onClick={() => onPublish(link)}>此任务发布安全</Button>}
        {!onSamples && !Object.values(actions).some(Boolean) && <Text type="secondary">当前状态没有可执行动作；请先处理阻塞或完成前一步。</Text>}
      </Space>
    </Card>
  );
}

function latestQualityObservation(quality?: CollectionQuality) {
	return [...(quality?.observations ?? [])].sort((a, b) => Date.parse(b.observed_at) - Date.parse(a.observed_at))[0];
}

function QualitySummaryCell({ record, onOpen }: { record: SourceRecord; onOpen: () => void }) {
	const completeness = collectionCompleteness(record.field_statuses);
	return <Space orientation="vertical" size={2}>
		<Space wrap size={[4, 4]}>
			<Tag color={completeness.isComplete ? 'green' : 'gold'}>{completeness.isComplete ? '关键字段已取得' : `${completeness.complete}/${completeness.total} 已取得`}</Tag>
			{(record.observation_count ?? 0) > 1 && <Tag color="blue">{record.observation_count} 次历史观察</Tag>}
		</Space>
		<Text type="secondary">详情质量与历史变化按需读取</Text>
		<Button size="small" onClick={onOpen}>查看质量</Button>
	</Space>;
}

export default function Sourcing1688Page() {
  const { message } = AntdApp.useApp();
  const qc = useQueryClient();
	  const [fetchOpen, setFetchOpen] = useState(false);
	  const [collectionStatus, setCollectionStatus] = useState<string>();
	const [requestedRecordID, setRequestedRecordID] = useState<number | null>();
  const [captureOpen, setCaptureOpen] = useState(false);
  const [reviewing, setReviewing] = useState<SourceRecord | null>(null);
  const [linkingTask, setLinkingTask] = useState<SourceRecord | null>(null);
	const [editingPrivate, setEditingPrivate] = useState<SourceRecord | null>(null);
  const [taskLinksTarget, setTaskLinksTarget] = useState<SourceRecord | null>(null);
  const [samplesTarget, setSamplesTarget] = useState<{ source: SourceRecord; link: TaskLink } | null>(null);
  const [authorityTarget, setAuthorityTarget] = useState<{ source: SourceRecord; link: TaskLink } | null>(null);
  const [sampleTransitionTarget, setSampleTransitionTarget] = useState<{ detail: SampleDetail; toStatus: SampleStatus } | null>(null);
  const [converting, setConverting] = useState<SourceRecord | null>(null);
  const [convertingTaskLink, setConvertingTaskLink] = useState<TaskLink | null>(null);
  const [preview, setPreview] = useState<DraftResult | null>(null);
  const [evidence, setEvidence] = useState<Snapshot | null>(null);
  const [approvalTarget, setApprovalTarget] = useState<SourceRecord | null>(null);
  const [approvalTaskLink, setApprovalTaskLink] = useState<TaskLink | null>(null);
  const [decisionTarget, setDecisionTarget] = useState<{ record: SourceRecord; approvalId: number } | null>(null);
  const [decisionTaskLink, setDecisionTaskLink] = useState<TaskLink | null>(null);
  const [imageTarget, setImageTarget] = useState<SourceRecord | null>(null);
  const [processedImage, setProcessedImage] = useState<Record<string, unknown> | null>(null);
  const [processedImagePreviewURL, setProcessedImagePreviewURL] = useState<string | null>(null);
  const [processedImagePreviewError, setProcessedImagePreviewError] = useState<string | null>(null);
  const [identityHistory, setIdentityHistory] = useState<{ sourceId: number; data: IdentityHistory } | null>(null);
	const [qualityTarget, setQualityTarget] = useState<SourceRecord | null>(null);
	const [watchTarget, setWatchTarget] = useState<SourceRecord | null>(null);
  const [acceptanceReport, setAcceptanceReport] = useState<AcceptanceReport | null>(null);
  const [publishTarget, setPublishTarget] = useState<SourceRecord | null>(null);
  const [publishTaskLink, setPublishTaskLink] = useState<TaskLink | null>(null);
  const [publishDecisionTarget, setPublishDecisionTarget] = useState<PublishAttempt | null>(null);
  const [publishExecuteTarget, setPublishExecuteTarget] = useState<PublishAttempt | null>(null);
  const [publishReconcileTarget, setPublishReconcileTarget] = useState<PublishAttempt | null>(null);
  const [captureForm] = Form.useForm();
  const [fetchForm] = Form.useForm();
  const [reviewForm] = Form.useForm();
  const [taskLinkForm] = Form.useForm();
  const [sampleRequestForm] = Form.useForm();
  const [sampleTransitionForm] = Form.useForm();
  const [sampleWaiverForm] = Form.useForm();
	const [privateWorkcopyForm] = Form.useForm();
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

	useEffect(() => {
		setRequestedRecordID(collectionRecordIDFromSearch(window.location.search));
	}, []);

	  const list = useQuery({
	    queryKey: ['sourcing-1688-controlled', collectionStatus, requestedRecordID],
		enabled: requestedRecordID !== undefined,
	    queryFn: () => apiClient.getPage<SourceRecord>('/v1/sourcing-1688', { page: '1', size: '100', ...(collectionStatus ? { lifecycle_status: collectionStatus } : {}), ...(requestedRecordID ? { record_id: String(requestedRecordID) } : {}) }),
	  });

  const eligibleTasks = useQuery({
    queryKey: ['sourcing-1688-eligible-tasks'],
    queryFn: () => apiClient.get<EligibleTask[]>('/v1/sourcing-1688/eligible-tasks'),
  });

  const taskLinks = useQuery({
    queryKey: ['sourcing-1688-task-links', taskLinksTarget?.id],
    enabled: !!taskLinksTarget,
    queryFn: () => apiClient.get<TaskLink[]>(`/v1/sourcing-1688/${taskLinksTarget!.id}/task-links`),
  });

	const collectionQuality = useQuery({
		queryKey: ['sourcing-1688-collection-quality', qualityTarget?.id],
		enabled: !!qualityTarget,
		queryFn: () => apiClient.get<CollectionQuality>(`/v1/sourcing-1688/${qualityTarget!.id}/collection-quality`),
	});
	const approvalCosts = useQuery({
		queryKey: ['sourcing-cost-versions', approvalTarget?.id],
		enabled: !!approvalTarget,
		queryFn: () => apiClient.get<PreciseCostVersion[]>(`/v1/sourcing-1688/${approvalTarget!.id}/cost-versions`),
	});
	const approvalCostOptions = (approvalCosts.data?.data ?? []).filter((item) => !approvalTaskLink || item.version.task_link_id === approvalTaskLink.id);

  const samples = useQuery({
    queryKey: ['sourcing-1688-samples', samplesTarget?.source.id],
    enabled: !!samplesTarget,
    queryFn: () => apiClient.get<SampleDetail[]>(`/v1/sourcing-1688/${samplesTarget!.source.id}/samples`),
  });

  const publishAttempts = useQuery({
    queryKey: ['sourcing-1688-publish-requests', publishTarget?.id, publishTaskLink?.id],
    enabled: !!publishTarget,
    queryFn: () => apiClient.get<PublishAttempt[]>(publishTaskLink
      ? taskWorkflowPath(publishTarget!.id, publishTaskLink.id, 'publish-requests')
      : `/v1/sourcing-1688/${publishTarget!.id}/publish-requests`),
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

  const linkTask = useMutation({
    mutationFn: (opportunityID: number) => {
      const task = eligibleTasks.data?.data?.find((item) => item.product_opportunity_id === opportunityID);
      if (!task) throw new Error('选品任务已不可用，请刷新后重试');
	  return apiClient.post(`/v1/sourcing-1688/${linkingTask?.id}/task-links`, { demand_case_id: task.demand_case_id, experiment_id: task.experiment_id, product_opportunity_id: task.product_opportunity_id });
    },
    onSuccess: () => {
      message.success('已关联选品任务，现在可以进入Owner复核');
      const sourceID = linkingTask?.id;
      setLinkingTask(null);
      taskLinkForm.resetFields();
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
      if (sourceID) void qc.invalidateQueries({ queryKey: ['sourcing-1688-task-links', sourceID] });
    },
    onError: (e: Error) => message.error(`关联选品任务失败：${e.message}`),
  });

  const createSample = useMutation({
    mutationFn: (values: { supplier_sku?: string; quantity: number }) => {
      if (!samplesTarget?.source.supplier_id || !samplesTarget.source.snapshot_id) {
        throw new Error('来源缺少权威供应商或不可变快照，不能创建样品申请');
      }
      return apiClient.post<SampleDetail>(`/v1/sourcing-1688/${samplesTarget.source.id}/samples`, {
        task_link_id: samplesTarget.link.id,
        supplier_id: samplesTarget.source.supplier_id,
        snapshot_id: samplesTarget.source.snapshot_id,
        supplier_sku: String(values.supplier_sku ?? '').trim() || undefined,
        quantity: values.quantity,
      });
    },
    onSuccess: () => {
      message.success('样品申请已建立；系统没有向供应商下单');
      sampleRequestForm.resetFields();
      void samples.refetch();
    },
    onError: (error: Error) => message.error(`样品申请失败：${error.message}`),
  });

  const transitionSample = useMutation({
    mutationFn: (values: Record<string, unknown>) => {
      if (!samplesTarget || !sampleTransitionTarget) throw new Error('样品状态已变化，请刷新后重试');
      return apiClient.post<SampleDetail>(
        `/v1/sourcing-1688/${samplesTarget.source.id}/samples/${sampleTransitionTarget.detail.sample.id}/transitions`,
        buildSampleTransitionPayload(sampleTransitionTarget.toStatus, values),
      );
    },
    onSuccess: () => {
      message.success(sampleTransitionTarget?.toStatus === 'approved_to_order'
        ? 'Owner批准已保存；系统仍未向供应商下单'
        : 'actual样品凭证和状态已保存');
      setSampleTransitionTarget(null);
      sampleTransitionForm.resetFields();
      void samples.refetch();
    },
    onError: (error: Error) => message.error(`样品状态保存失败：${error.message}`),
  });

  const waiveSample = useMutation({
    mutationFn: (values: { reason: string }) => {
      if (!samplesTarget) throw new Error('样品任务已变化，请刷新后重试');
      return apiClient.post<TaskLink>(`/v1/sourcing-1688/${samplesTarget.source.id}/task-links/${samplesTarget.link.id}/sample-waiver`, values);
    },
    onSuccess: (result) => {
      message.success('样品要求已由Owner明确豁免；风险理由已冻结');
      if (result.data) setSamplesTarget((current) => current ? { ...current, link: result.data! } : current);
      sampleWaiverForm.resetFields();
      void taskLinks.refetch();
    },
    onError: (error: Error) => message.error(`样品豁免失败：${error.message}`),
  });

	const updatePrivateWorkcopy = useMutation({
		mutationFn: (values: { title: string; price?: number; moq?: number; supplier_name?: string; notes?: string }) =>
			apiClient.patch(`/v1/sourcing-1688/${editingPrivate?.id}/private-workcopy`, { ...values, expected_updated_at: editingPrivate?.updated_at }),
		onSuccess: () => {
			message.success('私人工作副本已保存；原始页面快照没有改变');
			setEditingPrivate(null); privateWorkcopyForm.resetFields();
			void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
		},
		onError: (error: Error) => message.error(`保存失败：${error.message}`),
	});

  const setPrivateArchive = useMutation({
    mutationFn: ({ id, archived }: { id: number; archived: boolean }) =>
      apiClient.post(`/v1/sourcing-1688/${id}/${archived ? 'private-archive' : 'private-restore'}`, {}),
    onSuccess: (_result, variables) => {
      message.success(variables.archived ? '私人收藏已归档；原始观察仍然保留' : '私人收藏已恢复为待复核');
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
    },
    onError: (error: Error) => message.error(`操作失败：${error.message}`),
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
	    mutationFn: ({ id, reason, cost_version_id }: { id: number; reason: string; cost_version_id: number }) => apiClient.post(approvalTaskLink
	      ? taskWorkflowPath(id, approvalTaskLink.id, 'submit-draft-approval')
	      : `/v1/sourcing-1688/${id}/submit-draft-approval`, { reason, cost_version_id }),
    onSuccess: () => {
      message.success('草稿已提交 Owner 审批，仍未发布');
      const sourceID = approvalTarget?.id;
      setApprovalTarget(null); setApprovalTaskLink(null);
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
      if (sourceID) void qc.invalidateQueries({ queryKey: ['sourcing-1688-task-links', sourceID] });
    },
    onError: (e: Error) => message.error(`提交审批失败：${e.message}`),
  });

  const decideApproval = useMutation({
    mutationFn: ({ action, note }: { action: string; note: string }) => apiClient.post(decisionTaskLink
      ? taskWorkflowPath(decisionTarget!.record.id, decisionTaskLink.id, `approvals/${decisionTarget!.approvalId}/decision`)
      : `/v1/sourcing-1688/${decisionTarget?.record.id}/approvals/${decisionTarget?.approvalId}/decision`, { action, note }),
    onSuccess: () => {
      message.success('审批决定已保存；没有触发外部发布');
      const sourceID = decisionTarget?.record.id;
      setDecisionTarget(null); setDecisionTaskLink(null);
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
      if (sourceID) void qc.invalidateQueries({ queryKey: ['sourcing-1688-task-links', sourceID] });
    },
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
      if (convertingTaskLink) {
        const taskPath = taskWorkflowPath(converting!.id, convertingTaskLink.id, '');
        return convertingTaskLink.draft_id
          ? apiClient.put<DraftResult>(`${taskPath}/draft`, payload)
          : apiClient.post<DraftResult>(`${taskPath}/convert-to-draft`, payload);
      }
      if (converting?.product_id) return apiClient.put<DraftResult>(`/v1/sourcing-1688/${converting.id}/draft`, payload);
      return apiClient.post<DraftResult>(`/v1/sourcing-1688/${converting?.id}/convert-to-draft`, payload);
    },
    onSuccess: async (result) => {
      message.success('产品和待上架草稿已保存，未向平台发布');
      const sourceID = converting?.id;
      if (sourceID) {
        const detail = await apiClient.get<DraftResult>(convertingTaskLink
          ? taskWorkflowPath(sourceID, convertingTaskLink.id, 'draft')
          : `/v1/sourcing-1688/${sourceID}/draft`);
        setPreview(detail.data ?? result.data ?? {});
      } else setPreview(result.data ?? {});
      setConverting(null); setConvertingTaskLink(null); convertForm.resetFields();
      void qc.invalidateQueries({ queryKey: ['sourcing-1688-controlled'] });
      if (sourceID) void qc.invalidateQueries({ queryKey: ['sourcing-1688-task-links', sourceID] });
    },
    onError: (e: Error) => message.error(`生成草稿失败：${e.message}`),
  });

  const requestPublish = useMutation({
    mutationFn: (values: Record<string, unknown>) => apiClient.post<PublishAttempt>(publishTaskLink
      ? taskWorkflowPath(publishTarget!.id, publishTaskLink.id, 'publish-requests')
      : `/v1/sourcing-1688/${publishTarget?.id}/publish-requests`, buildPublishRequestPayload(values)),
    onSuccess: () => {
      message.success('发布请求已冻结并进入独立审批；没有调用外部平台');
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布请求创建失败：${e.message}`),
  });

  const decidePublish = useMutation({
    mutationFn: ({ action, note }: { action: 'approve' | 'reject'; note: string }) => apiClient.post<PublishAttempt>(publishTaskLink
      ? taskWorkflowPath(publishTarget!.id, publishTaskLink.id, `publish-requests/${publishDecisionTarget!.id}/decision`)
      : `/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishDecisionTarget?.id}/decision`, { action, note }),
    onSuccess: (_, variables) => {
      message.success(variables.action === 'approve' ? '独立发布审批已批准；仍未调用平台' : '发布请求已拒绝；没有调用平台');
      setPublishDecisionTarget(null);
      publishDecisionForm.resetFields();
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布审批失败：${e.message}`),
  });

  const executePublish = useMutation({
    mutationFn: () => {
      if (!publishExecuteTarget?.approval_id) throw new Error('发布请求缺少已批准的一次性审批记录');
      return apiClient.postApproved<PublishAttempt>(publishTaskLink
        ? taskWorkflowPath(publishTarget!.id, publishTaskLink.id, `publish-requests/${publishExecuteTarget.id}/execute`)
        : `/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishExecuteTarget.id}/execute`, {}, {
        approvalId: publishExecuteTarget.approval_id,
        idempotencyKey: publishExecuteTarget.idempotency_key,
      });
    },
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
    mutationFn: (values: Record<string, unknown>) => apiClient.post<PublishAttempt>(publishTaskLink
      ? taskWorkflowPath(publishTarget!.id, publishTaskLink.id, `publish-requests/${publishReconcileTarget!.id}/reconcile`)
      : `/v1/sourcing-1688/${publishTarget?.id}/publish-requests/${publishReconcileTarget?.id}/reconcile`, buildReconcilePayload(values)),
    onSuccess: () => {
      message.success('actual 平台证据已用于后置对账；submitted 仍需后续状态同步确认上线');
      setPublishReconcileTarget(null);
      publishReconcileForm.resetFields();
      void publishAttempts.refetch();
    },
    onError: (e: Error) => message.error(`发布结果对账失败：${e.message}`),
  });

  const openDraftEditor = async (record: SourceRecord, taskLink: TaskLink | null = null) => {
    convertForm.resetFields();
    const matchingImage = processedImage?.source_id === record.id ? processedImage : null;
    if (taskLink?.draft_id || (!taskLink && record.product_id)) {
      try {
        const detail = await apiClient.get<DraftResult>(taskLink
          ? taskWorkflowPath(record.id, taskLink.id, 'draft')
          : `/v1/sourcing-1688/${record.id}/draft`);
        if (!detail.data?.editable_payload || !detail.data.editable_sha256 || !detail.data.editable_version) {
          throw new Error('服务端未返回可核验的完整可编辑载荷');
        }
        convertForm.setFieldsValue(editableDraftToForm({
          ...detail.data.editable_payload,
          editable_version: detail.data.editable_version,
          editable_sha256: detail.data.editable_sha256,
        }));
      } catch (error) {
        message.error(`为防止覆盖既有草稿，已停止编辑：${(error as Error).message}`);
        return;
      }
    } else {
      convertForm.setFieldsValue({ ...defaultDraftValues(matchingImage), conversion_request_id: newConversionRequestID(record.id) });
    }
    setConvertingTaskLink(taskLink);
    setConverting(record);
  };

  const openTaskApprovalDecision = async (record: SourceRecord, taskLink: TaskLink) => {
    try {
      const res = await apiClient.get<{ draft?: { approval_id?: number } }>(taskWorkflowPath(record.id, taskLink.id, 'draft'));
      const approvalId = res.data?.draft?.approval_id;
      if (!approvalId) {
        message.error('此任务没有待处理的草稿审批记录');
        return;
      }
      setDecisionTaskLink(taskLink);
      setDecisionTarget({ record, approvalId });
    } catch (error) {
      message.error(`无法读取此任务的审批记录：${(error as Error).message}`);
    }
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

  const openPublishSafety = async (record: SourceRecord, taskLink: TaskLink | null = null) => {
    setPublishTaskLink(taskLink);
    setPublishTarget(record);
    publishRequestForm.resetFields();
    publishRequestForm.setFieldsValue({
      platform_account_id: undefined,
      idempotency_key: newPublishIdempotencyKey(record.id),
      reason: '',
      inventory_rows: [],
    });
    try {
      const draft = await apiClient.get<{ skus?: DraftSKU[] }>(taskLink
        ? taskWorkflowPath(record.id, taskLink.id, 'draft')
        : `/v1/sourcing-1688/${record.id}/draft`);
      publishRequestForm.setFieldValue('inventory_rows', (draft.data?.skus ?? []).map((sku) => ({ sku_code: sku.code, quantity: 0 })));
    } catch (error) {
      message.error(`无法读取冻结库存所需的 SKU：${(error as Error).message}`);
    }
  };

  const records = list.data?.data ?? [];
	const openedRecordFromURL = useRef<number | null>(null);
	useEffect(() => {
		if (!requestedRecordID || openedRecordFromURL.current === requestedRecordID || !list.isSuccess) return;
		const record = list.data?.data?.find((item) => item.id === requestedRecordID);
		if (!record) {
			openedRecordFromURL.current = requestedRecordID;
			message.warning(`没有找到私人采集记录 #${requestedRecordID}，或该记录不属于当前Owner`);
			return;
		}
		openedRecordFromURL.current = requestedRecordID;
		message.success(`已定位私人采集记录 #${requestedRecordID}`);
		void apiClient.get<Snapshot>(`/v1/sourcing-1688/${record.id}/snapshot`)
			.then((res) => setEvidence(res.data ?? null))
			.catch((error) => message.error(`来源证据读取失败：${(error as Error).message}`));
	}, [list.data, list.isSuccess, requestedRecordID, message]);
  const attempts = publishAttempts.data?.data ?? [];
  const allTaskLinks = taskLinks.data?.data ?? [];
  const primaryTaskLinks = allTaskLinks.filter((link) => link.is_primary);
  const additionalTaskLinks = allTaskLinks.filter((link) => !link.is_primary);
  const selectedTaskSamples = (samples.data?.data ?? []).filter((detail) => detail.sample.task_link_id === samplesTarget?.link.id);
  const taskCard = (link: TaskLink) => <TaskLinkCard
    key={link.id}
    link={link}
    onDraft={(selected) => { if (taskLinksTarget) void openDraftEditor(taskLinksTarget, selected); }}
    onSubmitApproval={(selected) => {
      if (!taskLinksTarget) return;
      setApprovalTaskLink(selected);
      setApprovalTarget(taskLinksTarget);
    }}
    onDecideApproval={(selected) => { if (taskLinksTarget) void openTaskApprovalDecision(taskLinksTarget, selected); }}
    onPublish={(selected) => { if (taskLinksTarget) void openPublishSafety(taskLinksTarget, selected); }}
    onSamples={(selected) => {
      if (!taskLinksTarget) return;
      sampleRequestForm.setFieldsValue({ quantity: 1 });
      setSamplesTarget({ source: taskLinksTarget, link: selected });
    }}
    onAuthority={(selected) => {
      if (taskLinksTarget) setAuthorityTarget({ source: taskLinksTarget, link: selected });
    }}
  />;
  const canCreatePublishRequest = publishAttempts.isSuccess && !attempts.some((attempt) => !['rejected', 'failed'].includes(attempt.status));
  return (
    <PageContainer
      title="1688 私人采集箱 → 受控草稿"
      subtitle="先保存Owner私人未验证线索；决定继续研究后，再关联选品任务并进入受控草稿。"
      extra={<Space wrap><Button icon={<ReloadOutlined />} onClick={() => void list.refetch()}>刷新</Button><Button onClick={() => setCaptureOpen(true)}>高级证据导入</Button><Button icon={<PlusOutlined />} onClick={() => setFetchOpen(true)}>旧版受控URL采集</Button></Space>}
    >
	      <Alert type="info" showIcon title="在1688商品页点击“采集到凌镜”" description="商品会直接进入Owner私人采集箱，不要求提前建立选品任务；未关联前只表示页面线索，不代表商品机会或可信货源。" style={{ marginBottom: 16 }} />
	      {requestedRecordID && <Alert type="success" showIcon title={`正在定位采集记录 #${requestedRecordID}`} description="只会显示当前Owner自己的记录；找到后会自动打开来源证据。" style={{ marginBottom: 16 }} />}
	      <Alert type="warning" showIcon icon={<SafetyCertificateOutlined />} title="外部发布受独立审批保护" description="草稿批准不会发布。只有 approved_draft 才显示发布安全区；请求发布、Owner 独立批准、再次执行和异常对账必须分别手动完成。" style={{ marginBottom: 16 }} />
	      <Space wrap style={{ marginBottom: 16 }}>
	        <Text strong>按状态筛选</Text>
	        <Select
	          aria-label="按采集箱状态筛选"
	          allowClear
	          placeholder="全部状态"
	          value={collectionStatus}
	          onChange={setCollectionStatus}
	          style={{ width: 190 }}
	          options={Object.entries(statusLabel).map(([value, label]) => ({ value, label }))}
	        />
	        {collectionStatus && <Button size="small" onClick={() => setCollectionStatus(undefined)}>清除筛选</Button>}
	      </Space>
	      {list.isError ? <Alert type="error" showIcon title="采集库加载失败" description={(list.error as Error).message} /> : (
        <Table<SourceRecord>
          rowKey="id" loading={list.isLoading} dataSource={records} scroll={{ x: 1300 }}
          pagination={{ pageSize: 20, showTotal: (n) => `共 ${n} 条` }}
          columns={[
            { title: '商品 / 供应商', width: 220, render: (_, r) => <><Text strong>{r.title || '未解析标题'}</Text><br /><Text type="secondary">{r.supplier_name || '供应商待核验'}</Text></> },
	            { title: '采集来源', width: 230, render: (_, r) => { const meta = r.latest_page_kind ? pageKindMeta[r.latest_page_kind] : null; return <><Tag color={meta?.color}>{meta?.label ?? '来源待确认'}</Tag><Text type="secondary">{r.latest_page_kind === 'controlled_fetch' ? '受控复核记录' : '1688页面声明，尚未核验'}</Text><br /><a href={r.source_url} target="_blank" rel="noreferrer">查看1688原页面</a></>; } },
	            { title: '任务关联', width: 220, render: (_, r) => (r.task_link_count ?? (r.experiment_id ? 1 : 0)) > 0 ? <><Tag color="blue">已关联 {r.task_link_count ?? 1} 个任务</Tag>{r.experiment_id && <><br /><Text type="secondary">主工作流：{r.experiment_id}</Text></>}<br /><Text type="secondary">点击查看每个任务的状态</Text></> : <><Tag>尚未关联任务</Tag><br /><Text type="secondary">私人收藏，不代表商品机会</Text></> },
	            { title: '采购信息', width: 130, render: (_, r) => <><Text>{r.field_statuses?.price === 'observed' && r.price != null ? `¥${r.price}` : '价格未取得'}</Text><br /><Text type="secondary">{r.field_statuses?.moq === 'observed' ? `MOQ ${r.moq}` : '起订量未取得'}</Text></> },
	            { title: '采集完整度', width: 250, render: (_, r) => { const completeness = collectionCompleteness(r.field_statuses); return <Space orientation="vertical" size={2}><Tag color={completeness.isComplete ? 'green' : 'gold'}>{completeness.isComplete ? '关键字段已取得' : `已取得 ${completeness.complete}/${completeness.total}`}</Tag>{completeness.missing.length > 0 && <Space wrap size={[4, 4]}>{completeness.missing.map((item) => <Tag key={item.field} color={item.status === 'parse_failed' ? 'red' : 'default'}>{item.label}：{item.status === 'parse_failed' ? '解析失败' : '未取得'}</Tag>)}</Space>}</Space>; } },
	            { title: '线索 / 最近观察 / 冲突', width: 245, render: (_, r) => <QualitySummaryCell record={r} onOpen={() => setQualityTarget(r)} /> },
	            { title: '状态 / 观察', width: 180, render: (_, r) => { const state = r.lifecycle_status || r.status; const observations = r.observation_count ?? (r.snapshot_id ? 1 : 0); const observedAt = r.latest_observed_at || r.created_at; return <><Tag color={statusColor[state] || statusColor[r.status]}>{statusLabel[state] || state}</Tag><br />{observations > 1 ? <Tag color="orange">共 {observations} 次观察</Tag> : <Text type="secondary">首次观察</Text>}<br /><Text type="secondary">最近观察 {observedAt ? new Date(observedAt).toLocaleString('zh-CN') : '未知'}</Text></>; } },
            { title: '追溯', width: 130, render: (_, r) => <><Text>采集 #{r.id}</Text><br /><Text type="secondary">产品 #{r.product_id ?? '未创建'}</Text></> },
            { title: '操作', fixed: 'right', width: 240, render: (_, r) => <Space wrap>
              <Button size="small" icon={<EyeOutlined />} onClick={() => setTaskLinksTarget(r)}>全部任务关联</Button>
			  {!r.experiment_id && (r.task_link_count ?? 0) === 0 && ['unverified_lead', 'needs_review'].includes(r.lifecycle_status || r.status) && <Button size="small" onClick={() => {
				setEditingPrivate(r); privateWorkcopyForm.setFieldsValue({ title: r.title, price: r.price, moq: r.moq, supplier_name: r.supplier_name, notes: r.review_notes });
			  }}>整理/备注</Button>}
	              {!r.experiment_id && (r.task_link_count ?? 0) === 0 && ['unverified_lead', 'needs_review'].includes(r.lifecycle_status || r.status) &&
                <Popconfirm title="归档这条私人收藏？" description="原始采集快照会保留；已关联任务或草稿的记录不能在这里归档。" okText="归档" cancelText="取消" onConfirm={() => setPrivateArchive.mutate({ id: r.id, archived: true })}>
	                  <Button size="small" loading={setPrivateArchive.isPending} icon={<DeleteOutlined />}>归档</Button>
	                </Popconfirm>}
	              {!r.experiment_id && (r.task_link_count ?? 0) === 0 && (r.lifecycle_status || r.status) === 'archived' &&
	                <Button size="small" loading={setPrivateArchive.isPending} onClick={() => setPrivateArchive.mutate({ id: r.id, archived: false })}>恢复</Button>}
	              {(r.lifecycle_status || r.status) !== 'archived' && <Button size="small" type={r.experiment_id ? "default" : "primary"} onClick={() => { setLinkingTask(r); taskLinkForm.resetFields(); }}>
	                {r.experiment_id ? '再关联一个任务' : '关联选品任务'}
	              </Button>}
              <Button size="small" disabled={!r.snapshot_id} onClick={() => void loadEvidence(r)}>查看证据</Button>
              <Button size="small" disabled={!r.snapshot_id} onClick={() => void loadIdentityHistory(r)}>变化/同款</Button>
              <Button size="small" disabled={!r.snapshot_id} onClick={() => setWatchTarget(r)}>货源关注</Button>
              <Button size="small" onClick={() => void loadAcceptanceReport(r)}>15项验收</Button>
              <Button size="small" disabled={!r.snapshot_id || !['collected', 'pending_review'].includes(r.status)} onClick={() => { setReviewing(r); reviewForm.setFieldsValue({ notes: '' }); }}>Owner 复核</Button>
              <Button size="small" type="primary" disabled={!r.reviewed_at || !['ready_for_product', 'editing'].includes(r.lifecycle_status || '')} onClick={() => void openDraftEditor(r)}>{r.product_id ? '编辑主任务草稿（兼容）' : '主任务转草稿（兼容）'}</Button>
              <Button size="small" disabled={!r.reviewed_at} onClick={() => { setImageTarget(r); setProcessedImage(null); setProcessedImagePreviewURL(null); setProcessedImagePreviewError(null); }}>处理图片</Button>
              <Button size="small" disabled={r.lifecycle_status !== 'editing'} onClick={() => { setApprovalTaskLink(null); setApprovalTarget(r); }}>提交主任务审批（兼容）</Button>
              <Button size="small" disabled={r.lifecycle_status !== 'pending_approval'} onClick={async () => { const res = await apiClient.get<{ approval_id?: number }>(`/v1/sourcing-1688/${r.id}/lifecycle`); const approvalId = res.data?.approval_id; if (approvalId) { setDecisionTaskLink(null); setDecisionTarget({ record: r, approvalId }); } else message.error('未找到草稿审批记录'); }}>审批主任务草稿（兼容）</Button>
              {r.lifecycle_status === 'approved_draft' && <Button size="small" danger icon={<SafetyCertificateOutlined />} onClick={() => void openPublishSafety(r, null)}>主任务发布安全（兼容）</Button>}
            </Space> },
          ]}
        />
      )}

      <Drawer
        title={`全部任务关联 · 收藏 #${taskLinksTarget?.id ?? ''}`}
        open={!!taskLinksTarget}
        width={680}
        onClose={() => setTaskLinksTarget(null)}
        extra={<Button onClick={() => void taskLinks.refetch()} loading={taskLinks.isFetching} icon={<ReloadOutlined />}>刷新</Button>}
      >
        <Alert
          type="info"
          showIcon
          title="同一条货源可以服务多个选品任务"
          description="每个任务都有自己的工作流、草稿、审批和发布安全链。主任务标签只用于兼容旧入口，不会授权系统替你操作其他任务。"
          style={{ marginBottom: 16 }}
        />
        {taskLinks.isError ? (
          <Alert type="error" showIcon title="任务关联加载失败" description={(taskLinks.error as Error).message} />
        ) : taskLinks.isLoading ? (
          <Card loading title="正在读取任务关联" />
        ) : allTaskLinks.length === 0 ? (
          <Alert type="warning" showIcon title="尚未关联任何选品任务" description="这条记录仍是Owner私人收藏，不代表商品机会。" />
        ) : (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <div>
              <Text strong>主工作流</Text>
              <Paragraph type="secondary" style={{ marginBottom: 8 }}>旧商品级按钮只兼容这个任务；推荐始终在对应任务卡内操作。</Paragraph>
              {primaryTaskLinks.length ? primaryTaskLinks.map(taskCard) : <Alert type="warning" showIcon title="主任务缺失" description="仍可查看独立任务，但不要使用旧商品级操作入口。" />}
            </div>
            <Divider style={{ margin: 0 }} />
            <div>
              <Text strong>其他关联（{additionalTaskLinks.length}）</Text>
              <Paragraph type="secondary" style={{ marginBottom: 8 }}>这些任务共享原始货源观察，但各自拥有独立草稿与审批状态。</Paragraph>
              {additionalTaskLinks.length ? additionalTaskLinks.map(taskCard) : <Text type="secondary">暂无其他任务关联</Text>}
            </div>
          </Space>
        )}
        <Divider />
	        {(taskLinksTarget?.lifecycle_status || taskLinksTarget?.status) !== 'archived' && <Button type="primary" onClick={() => { if (taskLinksTarget) { setLinkingTask(taskLinksTarget); taskLinkForm.resetFields(); } }}>
	          再关联一个任务
	        </Button>}
      </Drawer>

      <Drawer
        title={`精确成本与合规 · 货源 #${authorityTarget?.source.id ?? ''} / 任务 #${authorityTarget?.link.id ?? ''}`}
        open={!!authorityTarget}
        width={1120}
        onClose={() => setAuthorityTarget(null)}
      >
        {authorityTarget && <SourcingAuthorityWorkspace
          sourceID={authorityTarget.source.id}
          taskLinkID={authorityTarget.link.id}
          snapshotID={authorityTarget.source.snapshot_id}
          productID={authorityTarget.source.product_id}
        />}
      </Drawer>

      <Drawer
        title={`样品事实链 · 货源 #${samplesTarget?.source.id ?? ''}`}
        open={!!samplesTarget}
        width={760}
        onClose={() => { setSamplesTarget(null); sampleRequestForm.resetFields(); sampleWaiverForm.resetFields(); }}
        extra={<Button icon={<ReloadOutlined />} loading={samples.isFetching} onClick={() => void samples.refetch()}>刷新</Button>}
      >
        <Alert
          type="warning"
          showIcon
          title="批准下单不等于已经下单"
          description="系统只保存Owner决定和现实凭证，不会自动向1688或供应商下单。只有录入订单凭证、金额和观察时间后，状态才是已下单 actual。"
          style={{ marginBottom: 16 }}
        />
        {samplesTarget && (
          <Descriptions size="small" column={{ xs: 1, sm: 2 }} bordered style={{ marginBottom: 16 }}>
            <Descriptions.Item label="选定任务关联">#{samplesTarget.link.id}</Descriptions.Item>
            <Descriptions.Item label="冻结商品机会">#{samplesTarget.link.product_opportunity_id ?? '缺失'}</Descriptions.Item>
            <Descriptions.Item label="权威供应商">#{samplesTarget.source.supplier_id ?? '缺失'}</Descriptions.Item>
            <Descriptions.Item label="不可变快照">#{samplesTarget.source.snapshot_id ?? '缺失'}</Descriptions.Item>
          </Descriptions>
        )}
        {samplesTarget?.link.sample_policy === 'waived' ? (
          <Alert type="warning" showIcon title="Owner已豁免样品门禁" description={samplesTarget.link.sample_waiver_reason || '已记录豁免'} style={{ marginBottom: 16 }} />
        ) : selectedTaskSamples.some((detail) => detail.sample.status === 'accepted') ? (
          <Alert type="success" showIcon title="已有通过样品" description="该任务满足草稿送审的样品门禁。" style={{ marginBottom: 16 }} />
        ) : (
          <Card size="small" title="样品门禁例外（可选）" style={{ marginBottom: 16 }}>
            <Alert type="warning" showIcon title="仅在明确承担风险时豁免" description="豁免理由一旦保存不可修改；不等于供应商或商品已验证。" style={{ marginBottom: 12 }} />
            <Form form={sampleWaiverForm} layout="vertical" onFinish={(values) => waiveSample.mutate(values)}>
              <Form.Item name="reason" label="Owner豁免理由" rules={[{ required: true, message: '必须说明为什么不做样品' }, { min: 8, message: '请写清具体风险判断' }]}>
                <Input.TextArea rows={3} placeholder="例如：标准低风险商品，本轮仅验证页面草稿；若进入采购前必须重新取样。" />
              </Form.Item>
              <Button htmlType="submit" danger loading={waiveSample.isPending}>确认豁免并冻结理由</Button>
            </Form>
          </Card>
        )}
        {(!samplesTarget?.source.supplier_id || !samplesTarget.source.snapshot_id || !samplesTarget.link.product_opportunity_id) ? (
          <Alert type="error" showIcon title="不能建立样品申请" description="必须先取得权威供应商、不可变来源快照和已批准商品机会任务关联。" />
        ) : (
          <Card size="small" title="建立样品申请（不会外部下单）" style={{ marginBottom: 16 }}>
            <Form form={sampleRequestForm} layout="vertical" initialValues={{ quantity: 1 }} onFinish={(values) => createSample.mutate(values)}>
              <Row gutter={12}>
                <Col xs={24} sm={14}><Form.Item name="supplier_sku" label="供应商SKU（可选）"><Input maxLength={200} /></Form.Item></Col>
                <Col xs={24} sm={10}><Form.Item name="quantity" label="样品数量" rules={[{ required: true }]}><InputNumber min={1} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
              </Row>
              <Text type="secondary">供应商、快照和任务关联由当前货源自动带入，不能手工改写；Owner ID 由登录身份确定。</Text>
              <div style={{ marginTop: 12 }}><Button htmlType="submit" type="primary" loading={createSample.isPending}>建立申请</Button></div>
            </Form>
          </Card>
        )}
        {samples.isError ? <Alert type="error" showIcon title="样品事实链加载失败" description={(samples.error as Error).message} /> : samples.isLoading ? (
          <Card loading />
        ) : selectedTaskSamples.length === 0 ? (
          <Alert type="info" showIcon title="这个任务还没有样品记录" description="建立申请只记录研究意图，不会触发外部采购。" />
        ) : selectedTaskSamples.map((detail) => {
          const meta = sampleStatusMeta[detail.sample.status];
          const next = sampleNextStatuses(detail.sample.status);
          return <Card key={detail.sample.id} size="small" title={`样品 #${detail.sample.id}`} extra={<Tag color={meta.color}>{meta.label}</Tag>} style={{ marginBottom: 12 }}>
            <Descriptions size="small" column={{ xs: 1, sm: 2 }}>
              <Descriptions.Item label="数量">{detail.sample.quantity}</Descriptions.Item>
              <Descriptions.Item label="供应商SKU">{detail.sample.supplier_sku || '未填写'}</Descriptions.Item>
              <Descriptions.Item label="事实等级"><Tag color={detail.sample.truth_status === 'actual' ? 'green' : 'default'}>{detail.sample.truth_status}</Tag></Descriptions.Item>
              <Descriptions.Item label="金额">{detail.sample.order_amount != null ? `${detail.sample.order_amount} ${detail.sample.currency ?? ''}` : '尚无已下单金额'}</Descriptions.Item>
              <Descriptions.Item label="当前凭证" span={2}>{detail.sample.external_credential_uri ? <a href={detail.sample.external_credential_uri} target="_blank" rel="noreferrer">查看外部凭证</a> : '尚无actual外部凭证'}</Descriptions.Item>
            </Descriptions>
            <Divider style={{ margin: '12px 0' }} />
            <Timeline items={detail.events.map((event) => ({
              color: event.truth_status === 'actual' ? 'green' : 'gray',
              children: <div><Space wrap><Text strong>{sampleStatusMeta[event.to_status]?.label ?? event.to_status}</Text><Tag>{event.truth_status}</Tag></Space><br /><Text type="secondary">{event.observed_at ? `观察于 ${new Date(event.observed_at).toLocaleString('zh-CN')}` : `记录于 ${new Date(event.created_at).toLocaleString('zh-CN')}`}</Text>{event.external_credential_uri && <><br /><a href={event.external_credential_uri} target="_blank" rel="noreferrer">外部凭证</a></>}{event.order_amount != null && <><br /><Text>{event.order_amount} {event.currency}</Text></>}{event.note && <Paragraph style={{ marginBottom: 0 }}>{event.note}</Paragraph>}</div>,
            }))} />
            <Space wrap>{next.map((toStatus) => <Button key={toStatus} type={toStatus === 'rejected' ? 'default' : 'primary'} danger={toStatus === 'rejected'} onClick={() => {
              sampleTransitionForm.resetFields();
              sampleTransitionForm.setFieldsValue({ observed_at: localDateTime(), currency: 'CNY' });
              setSampleTransitionTarget({ detail, toStatus });
            }}>{sampleStatusMeta[toStatus].label}</Button>)}</Space>
          </Card>;
        })}
      </Drawer>

      <Modal
        title={sampleTransitionTarget ? `${sampleStatusMeta[sampleTransitionTarget.toStatus].label} · 样品 #${sampleTransitionTarget.detail.sample.id}` : '更新样品事实'}
        open={!!sampleTransitionTarget}
        onCancel={() => { setSampleTransitionTarget(null); sampleTransitionForm.resetFields(); }}
        onOk={() => sampleTransitionForm.validateFields().then((values) => transitionSample.mutate(values))}
        confirmLoading={transitionSample.isPending}
        okText="保存本次状态"
      >
        {sampleTransitionTarget?.toStatus === 'approved_to_order' ? (
          <Alert type="info" showIcon title="这只是Owner批准" description="保存后仍不会调用供应商或创建外部订单。实际下单完成后必须再录入actual订单凭证。" style={{ marginBottom: 16 }} />
        ) : (
          <Alert type="warning" showIcon title="必须来自现实事件" description="以下信息将按actual保存，请只录入可核验的外部凭证和真实观察时间。" style={{ marginBottom: 16 }} />
        )}
        <Form form={sampleTransitionForm} layout="vertical">
          {sampleTransitionTarget?.toStatus !== 'approved_to_order' && <>
            <Form.Item name="external_credential_uri" label="外部凭证 URI" rules={[{ required: true, message: '必须提供外部凭证' }]}><Input placeholder="订单、物流、签收照片或评估记录的位置" /></Form.Item>
            <Form.Item name="observed_at" label="实际观察时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
          </>}
          {sampleTransitionTarget?.toStatus === 'ordered' && <Row gutter={12}>
            <Col span={14}><Form.Item name="order_amount" label="实际订单金额" rules={[{ required: true }]}><InputNumber min={0} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
            <Col span={10}><Form.Item name="currency" label="币种" rules={[{ required: true }, { pattern: /^[A-Za-z]{3,8}$/ }]}><Input maxLength={8} /></Form.Item></Col>
          </Row>}
          <Form.Item
            name="note"
            label={sampleTransitionTarget?.toStatus === 'evaluated' ? '样品评估' : 'Owner说明'}
            rules={['approved_to_order', 'evaluated', 'accepted', 'rejected'].includes(sampleTransitionTarget?.toStatus ?? '') ? [{ required: true, message: '此状态必须填写说明' }] : undefined}
          ><Input.TextArea rows={4} placeholder="记录批准理由、样品质量、偏差或最终决定" /></Form.Item>
        </Form>
      </Modal>

      <Modal
		title={`整理私人收藏 · #${editingPrivate?.id ?? ''}`}
		open={!!editingPrivate}
		onCancel={() => { setEditingPrivate(null); privateWorkcopyForm.resetFields(); }}
		onOk={() => privateWorkcopyForm.validateFields().then((values) => updatePrivateWorkcopy.mutate(values))}
		confirmLoading={updatePrivateWorkcopy.isPending}
		okText="保存工作副本"
	  >
		<Alert type="info" showIcon title="这里只整理你的工作副本" description="原始1688页面快照保持不变；关联选品任务后应进入受控复核和草稿流程。" style={{ marginBottom: 16 }} />
		<Form form={privateWorkcopyForm} layout="vertical">
		  <Form.Item name="title" label="工作标题" rules={[{ required: true, message: '标题不能为空' }, { max: 500 }]}><Input /></Form.Item>
		  <Row gutter={12}><Col span={12}><Form.Item name="price" label="参考价格（未知可留空）"><InputNumber min={0} precision={2} style={{ width: '100%' }} /></Form.Item></Col>
		  <Col span={12}><Form.Item name="moq" label="起订量（未知填0）"><InputNumber min={0} precision={0} style={{ width: '100%' }} /></Form.Item></Col></Row>
		  <Form.Item name="supplier_name" label="供应商显示名称"><Input /></Form.Item>
		  <Form.Item name="notes" label="Owner备注" rules={[{ max: 4000 }]}><Input.TextArea rows={4} placeholder="记录待确认材质、报价条件、风险等" /></Form.Item>
		</Form>
	  </Modal>

	  <Modal
        title={`关联选品任务 · 收藏 #${linkingTask?.id ?? ''}`}
        open={!!linkingTask}
        onCancel={() => setLinkingTask(null)}
        onOk={() => taskLinkForm.validateFields().then((values) => linkTask.mutate(values.product_opportunity_id))}
        confirmLoading={linkTask.isPending}
        okText="关联并进入待复核"
      >
        <Alert type="info" showIcon title="关联后才进入受控经营流程" description="私人收藏本身不代表商品机会。后端会再次检查候选市场、Owner和商品机会闸门。" style={{ marginBottom: 16 }} />
        <Form form={taskLinkForm} layout="vertical">
          <Form.Item name="product_opportunity_id" label="已批准商品机会" rules={[{ required: true, message: '请选择商品机会' }]}>
            <Select
              loading={eligibleTasks.isLoading}
              placeholder={eligibleTasks.data?.data?.length ? '选择一个已满足条件的选品任务' : '目前没有可关联的选品任务'}
              options={(eligibleTasks.data?.data ?? []).map((task) => ({ value: task.product_opportunity_id, label: `${task.label} · 机会 #${task.product_opportunity_id}` }))}
              notFoundContent="目前没有已通过商品机会闸门的选品任务"
            />
          </Form.Item>
        </Form>
      </Modal>

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

	      <Modal title={approvalTaskLink ? `提交任务草稿审批 · ${approvalTaskLink.label || approvalTaskLink.experiment_id}` : '提交主任务草稿审批（兼容入口）'} open={!!approvalTarget} onCancel={() => { setApprovalTarget(null); setApprovalTaskLink(null); }} footer={null}>
	        {approvalTaskLink && <Alert type="info" showIcon title={`仅操作任务关联 #${approvalTaskLink.id}`} description={`草稿 #${approvalTaskLink.draft_id ?? '未知'}；不会提交其他任务的草稿。`} style={{ marginBottom: 12 }} />}
	        <Form layout="vertical" onFinish={(v) => approvalTarget && submitApproval.mutate({ id: approvalTarget.id, reason: v.reason, cost_version_id: v.cost_version_id })}>
	          <Form.Item name="cost_version_id" label="本次审批冻结的精确成本版本" rules={[{ required: true, message: '必须明确选择一个精确成本版本' }]}>
	            <Select loading={approvalCosts.isLoading} placeholder="选择 exact task 的不可变成本版本" options={approvalCostOptions.map((item) => ({ value: item.version.id, label: `v${item.version.version} · #${item.version.id} · 利润 ${item.version.contribution_profit_minor} ${item.version.target_currency}` }))} />
	          </Form.Item>
	          {approvalCosts.isError && <Alert type="error" showIcon title="精确成本版本读取失败" description={(approvalCosts.error as Error).message} style={{ marginBottom: 12 }} />}
	          {!approvalCosts.isLoading && !approvalCosts.isError && approvalCostOptions.length === 0 && <Alert type="warning" showIcon title="没有可冻结的精确成本版本" description="请先在“精确成本 / 合规”中保存版本；审批不会自动选择 latest。" style={{ marginBottom: 12 }} />}
	          <Form.Item name="reason" label="提交理由" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item>
	          <Button htmlType="submit" type="primary" disabled={approvalCostOptions.length === 0} loading={submitApproval.isPending}>提交 Owner 审批（不发布）</Button>
	        </Form>
	      </Modal>

      <Modal title={decisionTaskLink ? `Owner 审批任务草稿 · ${decisionTaskLink.label || decisionTaskLink.experiment_id}` : 'Owner 审批主任务草稿（兼容入口）'} open={!!decisionTarget} onCancel={() => { setDecisionTarget(null); setDecisionTaskLink(null); }} footer={null}>
        {decisionTaskLink && <Alert type="warning" showIcon title={`仅审批任务关联 #${decisionTaskLink.id}`} description={`草稿 #${decisionTaskLink.draft_id ?? '未知'}；批准不会发布，也不会影响其他任务。`} style={{ marginBottom: 12 }} />}
        <Form form={decisionForm} layout="vertical"><Form.Item name="note" label="审批说明" rules={[{ required: true }]}><Input.TextArea rows={3} /></Form.Item><Space><Button type="primary" loading={decideApproval.isPending} onClick={() => decisionForm.validateFields().then(({ note }) => decideApproval.mutate({ action: 'approve', note }))}>批准内部草稿</Button><Button danger loading={decideApproval.isPending} onClick={() => decisionForm.validateFields().then(({ note }) => decideApproval.mutate({ action: 'reject', note }))}>退回编辑</Button></Space></Form>
      </Modal>

      <Modal forceRender title={convertingTaskLink ? `${convertingTaskLink.draft_id ? '编辑' : '生成'}任务草稿 · ${convertingTaskLink.label || convertingTaskLink.experiment_id}` : `主任务草稿（兼容入口） · 采集 #${converting?.id ?? ''}`} open={!!converting} width={1120} onCancel={() => { setConverting(null); setConvertingTaskLink(null); }} onOk={() => convertForm.validateFields().then((v) => convert.mutate(v))} confirmLoading={convert.isPending} okText="保存并预览草稿">
        {convertingTaskLink && <Alert type="info" showIcon title={`当前只操作任务关联 #${convertingTaskLink.id}`} description={`独立状态：${taskLinkStatus(convertingTaskLink).label}；${convertingTaskLink.draft_id ? `草稿 #${convertingTaskLink.draft_id}` : '将为此任务生成独立草稿'}。`} style={{ marginBottom: 12 }} />}
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
        title={publishTaskLink ? `任务发布安全 · ${publishTaskLink.label || publishTaskLink.experiment_id}` : `主任务发布安全（兼容入口） · 采集 #${publishTarget?.id ?? ''}`}
        open={!!publishTarget}
        size={960}
        onClose={() => { setPublishTarget(null); setPublishTaskLink(null); }}
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
        {publishTaskLink && <Alert type="info" showIcon title={`本页只绑定任务关联 #${publishTaskLink.id}`} description={`草稿 #${publishTaskLink.draft_id ?? '未知'}；所有审批、执行和对账请求均携带该任务身份，不会回退到主任务。`} style={{ marginBottom: 16 }} />}

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
          <Descriptions.Item label="一次性审批 ID">#{publishExecuteTarget?.approval_id ?? '缺失（禁止执行）'}</Descriptions.Item>
          <Descriptions.Item label="幂等键"><Text copyable>{publishExecuteTarget?.idempotency_key}</Text></Descriptions.Item>
          <Descriptions.Item label="冻结请求 SHA"><Text copyable>{publishExecuteTarget?.request_sha256}</Text></Descriptions.Item>
        </Descriptions>
        <Divider />
        <Form form={publishExecuteForm} layout="vertical" onFinish={() => executePublish.mutate()}>
          <Form.Item name="confirmed" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须明确确认外部写风险')) }]}>
            <Checkbox>我已核对独立审批和冻结请求，确认现在执行真实平台发布</Checkbox>
          </Form.Item>
          <Button danger type="primary" htmlType="submit" disabled={!publishExecuteTarget?.approval_id} loading={executePublish.isPending}>确认执行外部发布</Button>
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

      <Drawer
		title={`采集质量中心 · 收藏 #${qualityTarget?.id ?? ''}`}
		open={!!qualityTarget}
		size={980}
		onClose={() => setQualityTarget(null)}
		extra={<Button icon={<ReloadOutlined />} loading={collectionQuality.isFetching} onClick={async () => {
			await Promise.all([collectionQuality.refetch(), list.refetch()]);
		}}>刷新新观察</Button>}
	  >
		{collectionQuality.isLoading && <Text type="secondary">正在汇总列表与详情观察…</Text>}
		{collectionQuality.isError && <Alert type="error" showIcon title="采集质量读取失败" description={(collectionQuality.error as Error).message} action={<Button size="small" onClick={() => void collectionQuality.refetch()}>重试</Button>} />}
		{collectionQuality.data?.data && (() => {
			const quality = collectionQuality.data.data;
			const latest = latestQualityObservation(quality);
			const detailURL = safe1688DetailURL(quality.recapture_action?.url || quality.source_url);
			return <Space orientation="vertical" size={16} style={{ width: '100%' }}>
				<Alert
					type={quality.conflicts.length > 0 ? 'warning' : quality.missing.length > 0 ? 'info' : 'success'}
					showIcon
						title={quality.conflicts.length > 0 ? `${quality.conflicts.length} 个字段存在历史变化` : quality.missing.length > 0 ? `${quality.missing.length} 个关键字段仍缺失` : '关键字段已有可追溯观察'}
					description={`共 ${quality.observations.length} 次观察；最近一次 ${latest ? `${pageKindMeta[latest.page_kind].label} · ${new Date(latest.observed_at).toLocaleString('zh-CN')}` : '未知'}。最佳值只是当前可用页面声明，不代表供应商事实已核验。`}
				/>
				{!quality.latest_detail_observation && <Alert
					type="warning"
					showIcon
					title="当前只有列表线索，缺少详情观察"
					description={quality.recapture_action?.reason || '请打开真实1688商品详情页，用凌镜插件补采价格、MOQ、供应商和SKU。'}
						action={detailURL ? <Button type="primary" href={detailURL} target="_blank" rel="noopener noreferrer">打开1688详情补采</Button> : <Button disabled>详情链接不安全或缺失</Button>}
				/>}
				<Table
					rowKey="field"
					pagination={false}
					dataSource={collectionQualityRows(quality)}
					columns={[
						{ title: '字段', dataIndex: 'label', width: 110 },
						{ title: '列表观察', dataIndex: 'list', render: qualityValue },
						{ title: '详情观察', dataIndex: 'detail', render: qualityValue },
						{ title: '当前最佳值', dataIndex: 'best', render: qualityValue },
							{ title: '质量判断', width: 190, render: (_, row) => row.conflict ? <Space orientation="vertical" size={2}><Tag color="blue">历史变化</Tag><Text type="secondary">请核对变化时间与来源</Text></Space> : row.missing ? <Tag color="gold">缺失</Tag> : <Tag color="green">可追溯</Tag> },
					]}
				/>
				{quality.conflicts.length > 0 && <Collapse items={quality.conflicts.map((conflict) => ({
					key: conflict.field,
					label: `${collectionFieldLabels[conflict.field] ?? conflict.field} · 历史观察值不同`,
					children: <Space orientation="vertical">{conflict.values.map((value, index) => <Card key={`${value.source.snapshot_id}-${index}`} size="small">{qualityValue(value)}</Card>)}</Space>,
				}))} />}
			</Space>;
		})()}
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
      {watchTarget && <SourceWatchWorkspace sourceID={watchTarget.id} sourceURL={watchTarget.source_url} open={!!watchTarget} onClose={() => setWatchTarget(null)} />}

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
