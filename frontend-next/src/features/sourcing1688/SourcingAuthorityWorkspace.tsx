'use client';

import { useEffect, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Card, Col, Collapse, DatePicker, Descriptions, Divider, Form, Input, InputNumber, Radio, Row, Select, Space, Table, Tag, Typography, message } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import apiClient from '@/lib/api-client';
import MaterialAssetWorkspace from './MaterialAssetWorkspace';

const { Text } = Typography;

export const COST_TYPES = [
  'purchase', 'domestic_shipping', 'packaging', 'cross_border_shipping', 'platform_fee',
  'payment_fee', 'advertising', 'tax', 'duty', 'return_loss',
] as const;
export const COMPLIANCE_TYPES = ['brand_ip', 'patent', 'certification', 'dangerous_goods', 'material', 'labeling_instructions'] as const;

const costLabels: Record<string, string> = {
  purchase: '采购', domestic_shipping: '境内运费', packaging: '包装', cross_border_shipping: '跨境运费', platform_fee: '平台费',
  payment_fee: '支付费', advertising: '广告', tax: '税费', duty: '关税', return_loss: '退货损耗',
};
const complianceLabels: Record<string, string> = {
  brand_ip: '品牌/IP', patent: '专利', certification: '认证', dangerous_goods: '危险品', material: '材质', labeling_instructions: '标签/说明书',
};

type Mapping = { id: number; product_id: number; internal_sku_id: number; supplier_sku: string; internal_sku: string; channel_sku: string; snapshot_id: number; version: number };
type CostLine = { id: number; cost_type: string; amount_minor: number; currency: string; normalized_amount_minor: number; exchange_rate_decimal?: string; exchange_rate_source_uri?: string; truth_status: string; source_uri: string; observed_at: string };
type CostVersion = { version: { id: number; task_link_id: number; sku_mapping_id: number; version: number; target_currency: string; total_minor: number; revenue_minor: number; contribution_profit_minor: number; pricing_basis: string; quantity_tier_min: number; quantity_tier_max?: number; purchase_line_owner_confirmed: boolean; content_hash: string; created_at: string }; lines: CostLine[] };
type DraftAuthority = { draft?: { cost_version_id?: number; cost_version_content_hash?: string; approval_status?: string } };
type ComplianceEvidence = { id: number; requirement_code: string; requirement_text: string; evidence_source: string; truth_status: string; scope: string; country_code: string; channel_code: string; observed_at: string; expires_at?: string; revoked_at?: string; revocation_reason?: string; review_status: string; review_notes?: string; internal_sku_id?: number };
type CostFormLine = { cost_type: string; amount_minor: string; currency: string; normalized_amount_minor: string; exchange_rate_decimal?: string; exchange_rate_source_uri?: string; exchange_rate_observed_at?: Dayjs; truth_status: string; source_uri: string; observed_at: Dayjs };
type SKUWorkspaceMapping = Mapping & { platform_id?: number; listing_id?: number };
export type SKUWorkspaceCombination = {
  key: string;
  supplier_sku: string;
  spec: string;
  values: Record<string, string>;
  quoted_price?: number;
  stock_status: 'observed' | 'unknown';
  quoted_stock?: number;
  issues: string[];
  duplicate: boolean;
  mapping?: SKUWorkspaceMapping;
};
export type SKUWorkspace = {
  source_id: number;
  task_link_id: number;
  snapshot_id: number;
  observed_at: string;
  target: { sales_channel: string; target_locale: string; product_opportunity_id: number; platform_ids: number[] };
  dimensions: Array<{ name: string; values: string[]; source: 'declared' | 'derived' }>;
  combinations: SKUWorkspaceCombination[];
  duplicate_combinations: Array<{ key: string; indexes: number[] }>;
  missing_price: string[];
  missing_stock: string[];
  missing_combinations: { status: 'calculated' | 'unknown'; combinations?: Array<Record<string, string>>; reason: string };
  canonical_mappings: SKUWorkspaceMapping[];
  status: 'ready' | 'needs_attention' | 'no_detail_observation';
  blockers: string[];
};

export type SourcingAuthorityWorkspaceProps = { sourceID: number; taskLinkID: number; snapshotID?: number; productID?: number };

function integerMinor(value: string, label: string): number {
  const normalized = String(value ?? '').trim();
  if (!/^\d+$/.test(normalized)) throw new Error(`${label}必须是非负整数最小货币单位`);
  const parsed = Number(normalized);
  if (!Number.isSafeInteger(parsed)) throw new Error(`${label}超出浏览器可安全提交范围`);
  return parsed;
}

export function complianceBlocker(row: ComplianceEvidence, now = new Date()) {
  if (row.revoked_at) return `已撤销：${row.revocation_reason || '未填写原因'}`;
  if (row.expires_at && new Date(row.expires_at) <= now) return '已过期';
  if (row.truth_status !== 'actual') return `${row.truth_status} 不是 actual，不能通过`;
  if (row.review_status !== 'approved') return row.review_status === 'rejected' ? 'Owner 已拒绝' : '等待 Owner 审核';
  return '';
}

export function skuCombinationFacts(row: SKUWorkspaceCombination) {
  return {
    price: row.quoted_price == null ? 'unknown' : `¥${row.quoted_price}`,
    stock: row.stock_status !== 'observed' || row.quoted_stock == null ? 'unknown' : String(row.quoted_stock),
    mapping: row.mapping ? `${row.mapping.supplier_sku} → ${row.mapping.internal_sku} → ${row.mapping.channel_sku}` : 'unmapped',
  };
}

export function normalizedMinorHalfUp(amountMinor: string, decimalRate: string) {
  if (!/^\d+$/.test(amountMinor) || !/^\d+(?:\.\d+)?$/.test(decimalRate)) return null;
  const [whole, fraction = ''] = decimalRate.split('.');
  const denominator = BigInt(10) ** BigInt(fraction.length);
  const numerator = BigInt(whole + fraction);
  const product = BigInt(amountMinor) * numerator;
  return ((product * BigInt(2) + denominator) / (BigInt(2) * denominator)).toString();
}

export default function SourcingAuthorityWorkspace({ sourceID, taskLinkID, snapshotID, productID }: SourcingAuthorityWorkspaceProps) {
  const qc = useQueryClient();
  const [costForm] = Form.useForm();
  const [costApprovalForm] = Form.useForm();
  const [complianceForm] = Form.useForm();
  const path = `/v1/sourcing-1688/${sourceID}/task-links/${taskLinkID}`;
  const mappings = useQuery({ queryKey: ['sourcing-sku-mappings', sourceID, taskLinkID], queryFn: () => apiClient.get<Mapping[]>(`${path}/sku-mappings`) });
  const skuWorkspace = useQuery({ queryKey: ['sourcing-sku-workspace', sourceID, taskLinkID], queryFn: () => apiClient.get<SKUWorkspace>(`${path}/sku-workspace`) });
  const costs = useQuery({ queryKey: ['sourcing-cost-versions', sourceID], queryFn: () => apiClient.get<CostVersion[]>(`/v1/sourcing-1688/${sourceID}/cost-versions`) });
  const compliance = useQuery({ queryKey: ['sourcing-compliance', sourceID, taskLinkID], queryFn: () => apiClient.get<ComplianceEvidence[]>(`${path}/compliance-evidence`) });
  const draftAuthority = useQuery({ queryKey: ['sourcing-draft-authority', sourceID, taskLinkID], queryFn: () => apiClient.get<DraftAuthority>(`${path}/draft`), retry: false });
  const mappingRows = useMemo(() => mappings.data?.data ?? [], [mappings.data?.data]);
  const exactCosts = (costs.data?.data ?? []).filter((item) => item.version.task_link_id === taskLinkID);
  const complianceRows = compliance.data?.data ?? [];
  const latestMappings = useMemo(() => {
    const latest = Math.max(0, ...mappingRows.map((row) => row.version));
    return mappingRows.filter((row) => row.version === latest);
  }, [mappingRows]);
  useEffect(() => {
    if (latestMappings[0] && !costForm.getFieldValue('sku_mapping_id')) costForm.setFieldValue('sku_mapping_id', latestMappings[0].id);
  }, [costForm, latestMappings]);

  const invalidate = async () => Promise.all([
    qc.invalidateQueries({ queryKey: ['sourcing-cost-versions', sourceID] }),
    qc.invalidateQueries({ queryKey: ['sourcing-compliance', sourceID, taskLinkID] }),
  ]);

  const createCost = useMutation({
    mutationFn: async (values: { sku_mapping_id: number; target_currency: string; revenue_minor: string; quantity_tier_min: number; quantity_tier_max?: number; purchase_line_owner_confirmed: boolean; lines: CostFormLine[] }) => {
      if (!snapshotID) throw new Error('缺少不可变来源快照，不能创建成本版本');
      const target = values.target_currency.trim().toUpperCase();
      const lines = values.lines.map((line) => {
        const currency = line.currency.trim().toUpperCase();
        const crossCurrency = currency !== target;
        return {
          cost_type: line.cost_type,
          amount_minor: integerMinor(line.amount_minor, `${costLabels[line.cost_type]}原币金额`),
          currency,
          normalized_amount_minor: integerMinor(line.normalized_amount_minor, `${costLabels[line.cost_type]}目标币金额`),
          truth_status: line.truth_status,
          source_uri: line.source_uri.trim(),
          observed_at: line.observed_at.toISOString(),
          ...(crossCurrency ? {
            exchange_rate_decimal: line.exchange_rate_decimal?.trim(),
            exchange_rate_source_uri: line.exchange_rate_source_uri?.trim(),
            exchange_rate_observed_at: line.exchange_rate_observed_at?.toISOString(),
          } : {}),
        };
      });
      return apiClient.post(`/v1/sourcing-1688/${sourceID}/cost-versions`, { task_link_id: taskLinkID, source_snapshot_id: snapshotID, sku_mapping_id: values.sku_mapping_id, target_currency: target, revenue_minor: integerMinor(values.revenue_minor, '预计收入'), pricing_basis: 'owner_confirmed_listing_price', quantity_tier_min: values.quantity_tier_min, quantity_tier_max: values.quantity_tier_max, purchase_line_owner_confirmed: values.purchase_line_owner_confirmed === true, lines });
    },
    onSuccess: async () => { message.success('不可变精确成本版本已保存'); costForm.resetFields(); await invalidate(); },
    onError: (error: Error) => message.error(error.message),
  });

  const createCompliance = useMutation({
    mutationFn: (values: { requirement_code: string; requirement_text: string; evidence_source: string; truth_status: string; scope: string; country_code: string; channel_code: string; internal_sku_id?: number; observed_at: Dayjs; issued_at?: Dayjs; expires_at?: Dayjs }) => {
      const resolvedProductID = productID ?? latestMappings[0]?.product_id;
      if (!resolvedProductID) throw new Error('缺少已转换产品身份，不能创建合规证据');
      return apiClient.post(`${path}/compliance-evidence`, { ...values, product_id: resolvedProductID, observed_at: values.observed_at.toISOString(), issued_at: values.issued_at?.toISOString(), expires_at: values.expires_at?.toISOString() });
    },
    onSuccess: async () => { message.success('独立合规证据已保存，仍需 Owner 审核'); complianceForm.resetFields(); await invalidate(); },
    onError: (error: Error) => message.error(error.message),
  });
  const submitCostApproval = useMutation({
    mutationFn: (values: { cost_version_id: number; reason: string }) => apiClient.post(`${path}/submit-draft-approval`, { cost_version_id: values.cost_version_id, reason: values.reason.trim() }),
    onSuccess: async () => { message.success('草稿审批已冻结所选精确成本版本'); await qc.invalidateQueries({ queryKey: ['sourcing-draft-authority', sourceID, taskLinkID] }); },
    onError: (error: Error) => message.error(error.message),
  });
  const reviewCompliance = useMutation({ mutationFn: ({ id, decision }: { id: number; decision: 'approved' | 'rejected' }) => apiClient.post(`${path}/compliance-evidence/${id}/review`, { decision, notes: decision === 'approved' ? 'Owner 已核验签发方、范围、时效和原始来源' : 'Owner 判断证据不足或范围不匹配' }), onSuccess: invalidate, onError: (error: Error) => message.error(error.message) });
  const revokeCompliance = useMutation({ mutationFn: (id: number) => apiClient.post(`${path}/compliance-evidence/${id}/revoke`, { reason: 'Owner 确认该证据已失效或不再适用' }), onSuccess: invalidate, onError: (error: Error) => message.error(error.message) });

  const initialLines = COST_TYPES.map((cost_type) => ({ cost_type, amount_minor: '0', currency: 'USD', normalized_amount_minor: '0', truth_status: 'quoted', source_uri: '', observed_at: dayjs() }));
  const missingCodes = COMPLIANCE_TYPES.filter((code) => !complianceRows.some((row) => row.requirement_code === code && !complianceBlocker(row)));
  const skuData = skuWorkspace.data?.data;
  const frozenDraft = draftAuthority.data?.data?.draft;
  const isApprovalReadOnly = frozenDraft?.approval_status === 'pending' || frozenDraft?.approval_status === 'approved';

  return <Space orientation="vertical" size={16} style={{ width: '100%' }}>
    <Alert type="info" showIcon title={`精确任务 #${taskLinkID}`} description="成本绑定 exact task + 不可变快照 + canonical SKU；全部金额使用最小货币单位整数。合规只有未过期、未撤销、Owner批准的 actual 证据才能通过。" />
    <MaterialAssetWorkspace sourceID={sourceID} taskLinkID={taskLinkID} snapshotID={snapshotID} mappings={latestMappings} readOnly={isApprovalReadOnly} />
    <Card title="SKU 整理工作台" extra={<Button size="small" loading={skuWorkspace.isFetching} onClick={() => void skuWorkspace.refetch()}>刷新观察</Button>}>
      {skuWorkspace.isLoading && <Text type="secondary">正在读取详情页 SKU 观察…</Text>}
      {skuWorkspace.isError && <Alert type="error" showIcon title="SKU 工作台读取失败" description={(skuWorkspace.error as Error).message} />}
      {skuData && <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <Alert
          type={skuData.status === 'ready' ? 'success' : 'warning'}
          showIcon
          title={skuData.status === 'ready' ? 'SKU 组合和 canonical 映射可继续核验' : skuData.status === 'no_detail_observation' ? '缺少详情页 SKU 观察' : 'SKU 组合需要整理'}
          description={skuData.blockers.length > 0 ? skuData.blockers.join('；') : `快照 #${skuData.snapshot_id} · ${new Date(skuData.observed_at).toLocaleString('zh-CN')}；页面价格与库存均为 quoted 声明。`}
        />
        <Descriptions size="small" bordered column={1}>
          <Descriptions.Item label="矩阵维度">
            {skuData.dimensions.length > 0 ? <Space wrap>{skuData.dimensions.map((dimension) => <Tag key={dimension.name} color={dimension.source === 'declared' ? 'blue' : 'default'}>{dimension.name}：{dimension.values.join(' / ')}（{dimension.source === 'declared' ? '页面声明' : '由规格拆分'}）</Tag>)}</Space> : <Text type="secondary">unknown · 页面没有可靠维度</Text>}
          </Descriptions.Item>
          <Descriptions.Item label="缺失组合">
            {skuData.missing_combinations.status === 'unknown'
              ? <Text type="secondary">unknown · {skuData.missing_combinations.reason}</Text>
              : (skuData.missing_combinations.combinations?.length ?? 0) > 0
                ? <Space wrap>{skuData.missing_combinations.combinations?.map((item, index) => <Tag color="gold" key={`${JSON.stringify(item)}-${index}`}>{Object.entries(item).map(([name, value]) => `${name}:${value}`).join(' / ')}</Tag>)}</Space>
                : <Tag color="green">未发现缺组合</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="重复组合">{skuData.duplicate_combinations.length > 0 ? <Tag color="red">{skuData.duplicate_combinations.length} 组重复</Tag> : <Tag color="green">未发现重复</Tag>}</Descriptions.Item>
          <Descriptions.Item label="字段缺口"><Space wrap><Tag color={skuData.missing_price.length ? 'gold' : 'green'}>缺价格 {skuData.missing_price.length}</Tag><Tag color={skuData.missing_stock.length ? 'gold' : 'green'}>缺库存 {skuData.missing_stock.length}</Tag><Tag color={skuData.combinations.filter((item) => !item.mapping).length ? 'gold' : 'green'}>未映射 {skuData.combinations.filter((item) => !item.mapping).length}</Tag></Space></Descriptions.Item>
        </Descriptions>
        <Table
          rowKey="__row_index"
          pagination={false}
          scroll={{ x: 1050 }}
          dataSource={skuData.combinations.map((row, index) => ({ ...row, __row_index: index }))}
          columns={[
            { title: '规格组合', width: 210, render: (_, row) => <Space orientation="vertical" size={0}><Text>{Object.entries(row.values).map(([name, value]) => `${name}: ${value}`).join(' / ') || row.spec || '规格未取得'}</Text><Text type="secondary">{row.spec || '无原始规格说明'}</Text></Space> },
            { title: '供应商 SKU', width: 140, dataIndex: 'supplier_sku', render: (value) => value ? <Text copyable>{value}</Text> : <Tag color="gold">未取得</Tag> },
            { title: '页面报价', width: 120, render: (_, row) => row.quoted_price == null ? <Tag>unknown</Tag> : <Space orientation="vertical" size={0}><Text>¥{row.quoted_price}</Text><Text type="secondary">quoted</Text></Space> },
            { title: '库存声明', width: 120, render: (_, row) => row.stock_status !== 'observed' || row.quoted_stock == null ? <Tag>unknown</Tag> : <Space orientation="vertical" size={0}><Text>{row.quoted_stock}</Text><Text type="secondary">quoted</Text></Space> },
            { title: '组合检查', width: 180, render: (_, row) => row.duplicate || row.issues.length > 0 ? <Space wrap>{row.duplicate && <Tag color="red">重复</Tag>}{row.issues.map((issue) => <Tag color="gold" key={issue}>{issue}</Tag>)}</Space> : <Tag color="green">未发现结构问题</Tag> },
            { title: 'Canonical 映射', width: 280, render: (_, row) => row.mapping ? <Space orientation="vertical" size={0}><Text>{row.mapping.supplier_sku} → {row.mapping.internal_sku} → {row.mapping.channel_sku}</Text><Text type="secondary">mapping #{row.mapping.id} · v{row.mapping.version} · snapshot #{row.mapping.snapshot_id}</Text></Space> : <Tag color="gold">未建立映射</Tag> },
          ]}
        />
      </Space>}
    </Card>
    <Card title="精确成本版本">
      {latestMappings.length === 0 && <Alert type="warning" showIcon title="缺少 canonical SKU mapping" description="请先为此任务生成草稿并冻结供应商 → 内部 → 渠道 SKU 映射。不得手填或猜测 mapping ID。" />}
      {isApprovalReadOnly && <Alert type="warning" showIcon title="草稿审批已冻结精确成本" description={`cost version #${frozenDraft?.cost_version_id ?? '未知'} · ${frozenDraft?.cost_version_content_hash ?? '缺少哈希'}；退回编辑前不能创建或切换成本。`} style={{ marginBottom: 12 }} />}
      <Form form={costForm} disabled={isApprovalReadOnly} layout="vertical" initialValues={{ target_currency: 'USD', sku_mapping_id: latestMappings[0]?.id, revenue_minor: '0', quantity_tier_min: 1, purchase_line_owner_confirmed: false, lines: initialLines }} onValuesChange={(changed, all) => {
        if (!changed.lines) return;
        const target = String(all.target_currency ?? '').toUpperCase();
        const next = (all.lines as CostFormLine[]).map((line) => {
          if (String(line.currency).toUpperCase() === target) return { ...line, normalized_amount_minor: line.amount_minor };
          const calculated = normalizedMinorHalfUp(String(line.amount_minor ?? ''), String(line.exchange_rate_decimal ?? ''));
          return calculated == null ? line : { ...line, normalized_amount_minor: calculated };
        });
        if (JSON.stringify(next) !== JSON.stringify(all.lines)) costForm.setFieldValue('lines', next);
      }} onFinish={(values) => createCost.mutate(values)}>
        <Row gutter={12}>
          <Col xs={24} md={12}><Form.Item name="sku_mapping_id" label="Exact SKU mapping" rules={[{ required: true }]}><Select options={latestMappings.map((row) => ({ value: row.id, label: `#${row.id} · ${row.supplier_sku} → ${row.internal_sku} → ${row.channel_sku}` }))} /></Form.Item></Col>
          <Col xs={24} md={12}><Form.Item name="target_currency" label="目标币种" rules={[{ required: true, pattern: /^[A-Za-z]{3,8}$/ }]}><Input maxLength={8} /></Form.Item></Col>
        </Row>
        <Row gutter={12}>
          <Col xs={24} md={6}><Form.Item name="revenue_minor" label="预计收入（目标币整数）" rules={[{ required: true, pattern: /^\d+$/ }]}><Input inputMode="numeric" /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="quantity_tier_min" label="适用数量下限" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="quantity_tier_max" label="适用数量上限（可空）"><InputNumber min={1} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="purchase_line_owner_confirmed" label="采购价确认" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须由Owner确认1688 quoted采购价')) }]}><Radio.Group options={[{ value: true, label: 'Owner已确认' }, { value: false, label: '尚未确认' }]} /></Form.Item></Col>
        </Row>
        <Form.List name="lines">{(fields) => <Collapse items={fields.map((field, index) => ({ key: COST_TYPES[index], label: `${index + 1}. ${costLabels[COST_TYPES[index]]}`, children: <>
          <Form.Item name={[field.name, 'cost_type']} hidden><Input /></Form.Item>
          <Row gutter={12}>
            <Col xs={24} md={6}><Form.Item name={[field.name, 'amount_minor']} label="原币金额（整数）" rules={[{ required: true, pattern: /^\d+$/ }]}><Input inputMode="numeric" /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item name={[field.name, 'currency']} label="原币" rules={[{ required: true, pattern: /^[A-Za-z]{3,8}$/ }]}><Input /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item name={[field.name, 'normalized_amount_minor']} label="目标币金额（整数）" rules={[{ required: true, pattern: /^\d+$/ }]}><Input inputMode="numeric" /></Form.Item></Col>
            <Col xs={24} md={6}><Form.Item name={[field.name, 'truth_status']} label="事实等级" rules={[{ required: true }]}><Select options={['actual', 'quoted', 'estimated'].map((value) => ({ value }))} /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name={[field.name, 'source_uri']} label="金额证据 URI" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={24} md={12}><Form.Item name={[field.name, 'observed_at']} label="金额观察时间" rules={[{ required: true }]}><DatePicker showTime style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name={[field.name, 'exchange_rate_decimal']} label="汇率十进制字符串（跨币种必填）"><Input placeholder="例如 7.2500" /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name={[field.name, 'exchange_rate_source_uri']} label="汇率来源（跨币种必填）"><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name={[field.name, 'exchange_rate_observed_at']} label="汇率观察时间"><DatePicker showTime style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
        </> }))} />}</Form.List>
        <Alert type="info" showIcon title="跨币种会按十进制汇率自动换算" description="前端只提供half-up预览；后端会用精确有理数重新计算并拒绝差异。1688价格仍是quoted候选，必须由Owner确认后才能保存。" style={{ marginBottom: 12 }} />
        <Button type="primary" htmlType="submit" disabled={isApprovalReadOnly || !snapshotID || latestMappings.length === 0} loading={createCost.isPending}>保存新的不可变成本版本</Button>
      </Form>
      <Divider />
      <Table rowKey={(row) => row.version.id} pagination={false} dataSource={exactCosts} expandable={{ expandedRowRender: (row) => <Table rowKey="id" pagination={false} size="small" dataSource={row.lines} columns={[{ title: '项目', dataIndex: 'cost_type', render: (v) => costLabels[v] ?? v }, { title: '原币整数', render: (_, v) => `${v.amount_minor} ${v.currency}` }, { title: '目标币整数', dataIndex: 'normalized_amount_minor' }, { title: '等级', dataIndex: 'truth_status', render: (v) => <Tag color={v === 'actual' ? 'green' : 'orange'}>{v}</Tag> }, { title: '证据', dataIndex: 'source_uri' }]} /> }} columns={[{ title: '版本', render: (_, row) => `v${row.version.version}` }, { title: 'SKU mapping', render: (_, row) => `#${row.version.sku_mapping_id}` }, { title: '总成本 / 收入', render: (_, row) => `${row.version.total_minor} / ${row.version.revenue_minor} ${row.version.target_currency}` }, { title: '贡献利润', render: (_, row) => <Tag color={row.version.contribution_profit_minor < 0 ? 'red' : 'green'}>{row.version.contribution_profit_minor} {row.version.target_currency}{row.version.contribution_profit_minor < 0 ? ' · 负利润' : ''}</Tag> }, { title: '审批选择', render: (_, row) => frozenDraft?.cost_version_id === row.version.id ? <Tag color="blue">当前审批冻结版本</Tag> : isApprovalReadOnly ? <Text type="secondary">不可切换</Text> : <Text type="secondary">提交审批时选择此ID</Text> }, { title: '内容哈希', render: (_, row) => <Text code>{row.version.content_hash.slice(0, 12)}…</Text> }, { title: '创建时间', render: (_, row) => new Date(row.version.created_at).toLocaleString('zh-CN') }]} />
      <Divider />
      <Form form={costApprovalForm} disabled={isApprovalReadOnly} layout="inline" onFinish={(values) => submitCostApproval.mutate(values)}>
        <Form.Item name="cost_version_id" label="本次审批冻结成本版本" rules={[{ required: true }]}><Select style={{ width: 260 }} options={exactCosts.map((item) => ({ value: item.version.id, label: `v${item.version.version} · #${item.version.id} · 利润 ${item.version.contribution_profit_minor}` }))} /></Form.Item>
        <Form.Item name="reason" label="提交理由" rules={[{ required: true }]}><Input style={{ width: 300 }} /></Form.Item>
        <Button type="primary" htmlType="submit" disabled={isApprovalReadOnly || exactCosts.length === 0} loading={submitCostApproval.isPending}>冻结此版本并提交草稿审批</Button>
      </Form>
    </Card>

    <Card title="独立合规证据">
      {missingCodes.length > 0 ? <Alert type="error" showIcon title={`发布阻塞：${missingCodes.length} 类证据未通过`} description={missingCodes.map((code) => complianceLabels[code]).join('、')} /> : <Alert type="success" showIcon title="六类证据当前均有 approved actual 记录" description="仍会在发布时重新核验过期和撤销状态。" />}
      <Form form={complianceForm} layout="vertical" initialValues={{ requirement_code: 'brand_ip', truth_status: 'quoted', observed_at: dayjs() }} onFinish={(values) => createCompliance.mutate(values)} style={{ marginTop: 16 }}>
        <Row gutter={12}>
          <Col xs={24} md={8}><Form.Item name="requirement_code" label="要求类型" rules={[{ required: true }]}><Select options={COMPLIANCE_TYPES.map((value) => ({ value, label: complianceLabels[value] }))} /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="country_code" label="冻结市场国家/地区" rules={[{ required: true }]}><Input /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="channel_code" label="冻结销售渠道" rules={[{ required: true }]}><Input /></Form.Item></Col>
          <Col xs={24} md={12}><Form.Item name="requirement_text" label="具体要求" rules={[{ required: true }]}><Input /></Form.Item></Col>
          <Col xs={24} md={12}><Form.Item name="evidence_source" label="原始证据 URI" rules={[{ required: true }]}><Input /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="truth_status" label="事实等级" rules={[{ required: true }]}><Radio.Group options={['actual', 'quoted', 'estimated', 'unknown'].map((value) => ({ label: value, value }))} /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="scope" label="适用范围" rules={[{ required: true }]}><Input placeholder="product / SKU / country / channel" /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="internal_sku_id" label="内部 SKU（可选）"><Select allowClear options={latestMappings.map((row) => ({ value: row.internal_sku_id, label: row.internal_sku }))} /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="observed_at" label="观察时间" rules={[{ required: true }]}><DatePicker showTime style={{ width: '100%' }} /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="issued_at" label="签发时间"><DatePicker showTime style={{ width: '100%' }} /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name="expires_at" label="到期时间"><DatePicker showTime style={{ width: '100%' }} /></Form.Item></Col>
        </Row>
        <Alert type="warning" showIcon title="quoted / estimated 只能留作线索" description="即使 Owner 点击审核，后端也会拒绝将非 actual 或已过期证据批准为通过。" style={{ marginBottom: 12 }} />
        <Button type="primary" htmlType="submit" loading={createCompliance.isPending}>保存合规证据</Button>
      </Form>
      <Divider />
      <Table rowKey="id" pagination={false} dataSource={complianceRows} columns={[
        { title: '类型', dataIndex: 'requirement_code', render: (v) => complianceLabels[v] ?? v },
        { title: '等级', dataIndex: 'truth_status', render: (v) => <Tag color={v === 'actual' ? 'green' : 'orange'}>{v}</Tag> },
        { title: '范围', dataIndex: 'scope' },
        { title: '审核', dataIndex: 'review_status', render: (v) => <Tag color={v === 'approved' ? 'green' : v === 'rejected' ? 'red' : 'default'}>{v}</Tag> },
        { title: '当前 blocker', render: (_, row) => { const blocker = complianceBlocker(row); return blocker ? <Text type="danger">{blocker}</Text> : <Tag color="green">可用 actual</Tag>; } },
        { title: '操作', render: (_, row) => <Space wrap>{row.review_status === 'pending' && <><Button size="small" disabled={row.truth_status !== 'actual' || !!complianceBlocker({ ...row, review_status: 'approved' })} onClick={() => reviewCompliance.mutate({ id: row.id, decision: 'approved' })}>Owner批准</Button><Button size="small" danger onClick={() => reviewCompliance.mutate({ id: row.id, decision: 'rejected' })}>拒绝</Button></>}{!row.revoked_at && <Button size="small" danger onClick={() => revokeCompliance.mutate(row.id)}>撤销</Button>}</Space> },
      ]} />
    </Card>
  </Space>;
}
