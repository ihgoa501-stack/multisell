'use client';

import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Button, Card, Checkbox, Col, Empty, Form, Image, Input, InputNumber, List, Row, Select, Space, Spin, Tag, Typography, Upload,
} from 'antd';
import type { UploadProps } from 'antd';
import { CloudUploadOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import PageHeader from '@/components/ui/PageHeader';
import {
  createCostEntry, createFiveAxisReview, createImageJob, createProductImageSet, createRightsGrant, executeImageJob, fetchImageOutput, freezeProductImageSet, getImageProcessorCapabilities, listImageJobs, newImageIdempotencyKey, uploadSourceImage,
} from './api';
import type { CreateCostEntryInput, CreateRightsGrantInput, FiveAxisReviewInput, ImageGateStatus, ImageProcessorCapability, ProductImageAsset, ProductImageJob, ProductImageJobStatus, ProductImageRole, ProductImageSet } from './types';

const statusPresentation: Record<ProductImageJobStatus, { label: string; color: string }> = {
  pending: { label: '待执行', color: 'default' }, queued: { label: '排队中', color: 'processing' },
  created: { label: '待执行', color: 'default' },
  running: { label: '处理中', color: 'processing' }, completed: { label: '已完成', color: 'success' },
  succeeded: { label: '已完成', color: 'success' }, failed: { label: '失败', color: 'error' }, reconcile_required: { label: '待对账', color: 'warning' },
  QUEUED: { label: '排队中', color: 'processing' }, RUNNING: { label: '处理中', color: 'processing' }, READY: { label: '已完成', color: 'success' }, FAILED: { label: '失败', color: 'error' },
};

function errorText(error: unknown) {
  return error instanceof Error ? error.message : '发生未知错误';
}

function ProtectedOutput({ jobId }: { jobId: number }) {
  const output = useQuery({ queryKey: ['product-images', 'output', jobId], queryFn: () => fetchImageOutput(jobId), staleTime: Infinity });
  const [url, setURL] = useState<string>();
  useEffect(() => { if (!output.data) return; const next = URL.createObjectURL(output.data); setURL(next); return () => URL.revokeObjectURL(next); }, [output.data]);
  if (output.isLoading) return <Spin />;
  if (output.error || !url) return <Alert type="error" message="候选图片读取失败" description={errorText(output.error)} />;
  return <Image src={url} alt="图片处理候选" width="100%" />;
}

export default function ProductImagesWorkspace() {
  const client = useQueryClient();
  const [source, setSource] = useState<ProductImageAsset>();
  const [selected, setSelected] = useState<number[]>([]);
  const [roles, setRoles] = useState<Record<number, ProductImageRole>>({});
  const [imageSet, setImageSet] = useState<ProductImageSet>();
  const [governanceJobId, setGovernanceJobId] = useState<number>();
  const [rightsRecorded, setRightsRecorded] = useState(false);
  const [reviewRecorded, setReviewRecorded] = useState(false);
  const [setForm] = Form.useForm<{ listing_id: number; channel: string; locale: string }>();
  const [rightsForm] = Form.useForm<CreateRightsGrantInput>();
  const [reviewForm] = Form.useForm<FiveAxisReviewInput>();
  const [costForm] = Form.useForm<CreateCostEntryInput>();
  const jobs = useQuery({
    queryKey: ['product-images', 'jobs'], queryFn: listImageJobs, retry: false,
    refetchInterval: (query) => query.state.data?.some((job) => ['queued', 'running', 'QUEUED', 'RUNNING'].includes(job.status)) ? 2000 : false,
  });
  const capabilities = useQuery({
    queryKey: ['product-images', 'capabilities'], queryFn: getImageProcessorCapabilities, retry: false,
  });
  const deterministic = capabilities.data?.find((capability) => capability.code === 'deterministic');
  const deterministicAvailable = deterministic?.configured === true;
  const refresh = () => client.invalidateQueries({ queryKey: ['product-images', 'jobs'] });
  const upload = useMutation({ mutationFn: uploadSourceImage, onSuccess: setSource });
  const create = useMutation({
    mutationFn: () => createImageJob({
      asset_id: source!.id, idempotency_key: newImageIdempotencyKey(`product-image-create-${source!.id}`),
      operation: 'DETERMINISTIC_RESIZE', width: 1200, height: 1200, format: 'png',
    }),
    onSuccess: refresh,
  });
  const execute = useMutation({ mutationFn: executeImageJob, onSuccess: refresh });
  const customRequest: UploadProps['customRequest'] = ({ file, onError, onSuccess }) => {
    upload.mutate(file as File, { onSuccess: (asset) => onSuccess?.(asset), onError: (error) => onError?.(error) });
  };
  const completed = useMemo(() => (jobs.data ?? []).filter((job) => job.status === 'READY' && job.output_blob_id), [jobs.data]);
  const governanceJob = completed.find((job) => job.id === governanceJobId);
  useEffect(() => {
    if (!governanceJobId && completed[0]) setGovernanceJobId(completed[0].id);
  }, [completed, governanceJobId]);
  const rights = useMutation({ mutationFn: async () => {
    const values = await rightsForm.validateFields();
    return createRightsGrant({ ...values, asset_sha256: governanceJob!.output_blob_id!, valid_from: new Date().toISOString(), idempotency_key: newImageIdempotencyKey(`image-rights-${governanceJob!.id}`), expected_version: 1 });
  }, onSuccess: () => setRightsRecorded(true) });
  const review = useMutation({ mutationFn: async () => {
    const values = await reviewForm.validateFields();
    return createFiveAxisReview(governanceJob!.id, { ...values, asset_sha256: governanceJob!.output_blob_id!, idempotency_key: newImageIdempotencyKey(`image-review-${governanceJob!.id}`), expected_version: governanceJob!.version ?? 1 });
  }, onSuccess: () => setReviewRecorded(true) });
  const cost = useMutation({ mutationFn: async () => {
    const values = await costForm.validateFields();
    return createCostEntry(governanceJob!.id, { ...values, observed_at: new Date().toISOString(), idempotency_key: newImageIdempotencyKey(`image-cost-${governanceJob!.id}`), expected_version: governanceJob!.version ?? 1 });
  } });
  const createSet = useMutation({
    mutationFn: async () => {
      const scope = await setForm.validateFields();
      const ordered = completed.filter((job) => selected.includes(job.id));
      return createProductImageSet({ ...scope, items: ordered.map((job, index) => ({ task_id: job.id, role: roles[job.id] ?? (index === 0 ? 'main' : 'gallery'), ordinal: index + 1 })) });
    },
    onSuccess: setImageSet,
  });
  const freezeSet = useMutation({ mutationFn: freezeProductImageSet, onSuccess: setImageSet });
  const mutationError = upload.error ?? create.error ?? execute.error ?? rights.error ?? review.error ?? cost.error ?? createSet.error ?? freezeSet.error;

  const capabilityLabel = (item: ImageProcessorCapability) => {
    if (item.code === 'openai') return `${item.name} · 适配器合同已实现，但付费执行未启用（需 Owner 批准门禁）`;
    return `${item.name} · ${item.configured ? '可用' : '未配置'}`;
  };

  return <div style={{ padding: '16px 20px', minHeight: '100%' }}>
    <PageHeader title="商品图片工作室" subtitle="上传真实商品原图，生成候选图片；完成并不代表图片已获准发布" />
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon title={deterministicAvailable ? '当前可用闭环：确定性尺寸处理' : '图片处理能力尚未可用'} description="能力状态来自凌镜后端。外部 Provider 还必须满足配置、预算和 Owner 批准门禁，不会用模拟图片冒充结果。" />
      {(jobs.error || mutationError) && <Alert type="error" showIcon title="图片工作台操作失败" description={errorText(jobs.error ?? mutationError)} />}
      <Card title="处理方案">
        {capabilities.isLoading ? <Space><Spin size="small" /><Typography.Text type="secondary">正在读取真实能力状态…</Typography.Text></Space>
          : capabilities.error ? <Alert type="error" showIcon title="能力状态读取失败" description={`${errorText(capabilities.error)}；为避免误执行，所有处理入口已关闭。`} action={<Button size="small" onClick={() => capabilities.refetch()}>重试</Button>} />
            : <Space wrap>{(capabilities.data ?? []).map((item) => <Tag key={item.code} color={item.code !== 'openai' && item.configured ? 'success' : 'default'}>
              {capabilityLabel(item)}
            </Tag>)}</Space>}
        {!capabilities.isLoading && !capabilities.error && !deterministicAvailable && <Alert style={{ marginTop: 12 }} type="warning" showIcon title="确定性处理未配置" description={deterministic?.reason || '请先配置并启动 Image Service；在后端确认可用前，不能创建或执行图片任务。'} />}
      </Card>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={9}><Card title="1. 上传真实原图">
          <Upload accept="image/png,image/jpeg" maxCount={1} showUploadList={false} customRequest={customRequest}>
            <Button icon={<CloudUploadOutlined />} loading={upload.isPending}>选择并上传图片</Button>
          </Upload>
          {source && <Space orientation="vertical" style={{ marginTop: 16 }}>
            <Typography.Text strong>{source.filename}</Typography.Text>
            <Typography.Text type="secondary">SHA-256：{source.sha256}</Typography.Text>
          </Space>}
        </Card></Col>
        <Col xs={24} lg={15}><Card title="2. 创建处理任务">
          <Typography.Paragraph>将原图确定性处理为 1200 × 1200 PNG。此路径不生成新商品结构和文字。</Typography.Paragraph>
          <Button type="primary" disabled={!source || !deterministicAvailable} loading={create.isPending} onClick={() => create.mutate()}>创建确定性图片任务</Button>
        </Card></Col>
      </Row>
      <Card title="任务" extra={<Button icon={<ReloadOutlined />} loading={jobs.isFetching} onClick={() => jobs.refetch()}>刷新</Button>}>
        {jobs.isLoading ? <Spin /> : !(jobs.data?.length) ? <Empty description="还没有图片处理任务" /> : <List dataSource={jobs.data} renderItem={(job: ProductImageJob) => {
          const status = statusPresentation[job.status] ?? { label: job.status, color: 'default' };
          return <List.Item actions={['pending', 'created'].includes(job.status) ? [<Button key="execute" disabled={!deterministicAvailable} icon={<PlayCircleOutlined />} loading={execute.isPending && execute.variables === job.id} onClick={() => execute.mutate(job.id)}>执行</Button>] : []}>
            <List.Item.Meta title={<Space><Typography.Text code>{job.id}</Typography.Text><Tag color={status.color}>{status.label}</Tag></Space>} description={<Space orientation="vertical" size={0}><span>{job.operation} · {job.width} × {job.height} {job.format}</span>{job.error_code && <Typography.Text type="danger">错误代码：{job.error_code}</Typography.Text>}</Space>} />
          </List.Item>;
        }} />}
      </Card>
      <Card title="3. 权利、五类审核与成本">
        {!completed.length ? <Empty description="处理成功并取得受控输出后才能登记权利与审核" /> : <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="warning" showIcon title="这些记录是发布前门禁，不是自动判断" description="必须绑定精确输出 SHA-256。只有 Owner 明确核对权利证据和五类检查后，图片集合才允许冻结。" />
          <Select<number> value={governanceJobId} style={{ width: '100%' }} onChange={(id) => { setGovernanceJobId(id); setRightsRecorded(false); setReviewRecorded(false); }} options={completed.map((job) => ({ value: job.id, label: `任务 ${job.id} · ${job.output_blob_id}` }))} />
          {governanceJob && <>
            <Typography.Text code>输出 SHA-256：{governanceJob.output_blob_id}</Typography.Text>
            <Row gutter={[16, 16]}>
              <Col xs={24} xl={12}><Card size="small" title="A. 精确图片权利授权">
                <Form form={rightsForm} layout="vertical" initialValues={{ purpose: 'listing_main', jurisdiction: '*', channel: 'ozon', provider: 'deterministic', region: 'local', can_copy: false, can_modify: false, can_third_party_ai: false, can_cross_border: false, can_commercial_publish: false, can_platform_sublicense: false, trademark_cleared: false, likeness_cleared: false, owner_verified: false }}>
                  <Row gutter={8}><Col span={12}><Form.Item name="purpose" label="用途" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Row gutter={8}><Col span={12}><Form.Item name="jurisdiction" label="法域" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="region" label="处理地区" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Row gutter={8}><Col span={12}><Form.Item name="provider" label="处理方" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="grantor" label="授权人" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Form.Item name="rights_chain" label="权利链说明" rules={[{ required: true }]}><Input.TextArea rows={2} /></Form.Item>
                  <Form.Item name="evidence_sha256" label="权利证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/, message: '必须是64位小写SHA-256' }]}><Input /></Form.Item>
                  <Space wrap>{['can_copy', 'can_modify', 'can_third_party_ai', 'can_cross_border', 'can_commercial_publish', 'can_platform_sublicense', 'trademark_cleared', 'likeness_cleared'].map((name) => <Form.Item key={name} name={name} valuePropName="checked" noStyle><Checkbox>{name}</Checkbox></Form.Item>)}</Space>
                  <Form.Item name="owner_verified" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须由Owner明确核验')) }]}><Checkbox>我已核对上述授权和精确证据</Checkbox></Form.Item>
                  <Button type="primary" disabled={!governanceJob} loading={rights.isPending} onClick={() => rights.mutate()}>保存权利授权</Button>
                  {rightsRecorded && <Tag color="success">权利记录已保存</Tag>}
                </Form>
              </Card></Col>
              <Col xs={24} xl={12}><Card size="small" title="B. 五类逐项审核">
                <Form form={reviewForm} layout="vertical" initialValues={{ purpose: 'listing_main', channel: 'ozon', evidence_truth: 'quoted', product_authenticity: 'unknown', rights: 'unknown', channel_rules: 'unknown', claims_scene: 'unknown', technical_visual: 'unknown' }}>
                  <Row gutter={8}><Col span={12}><Form.Item name="purpose" label="用途" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  {([['product_authenticity', '商品真实性'], ['rights', '图片权利'], ['channel_rules', '渠道规则'], ['claims_scene', '声明与场景'], ['technical_visual', '技术视觉']] as const).map(([name, label]) => <Form.Item key={name} name={name} label={label} rules={[{ required: true }]}><Select<ImageGateStatus> options={['passed', 'blocked', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>)}
                  <Form.Item name="evidence_sha256" label="审核证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/, message: '必须是64位小写SHA-256' }]}><Input /></Form.Item>
                  <Form.Item name="evidence_truth" label="证据真实性"><Select options={['quoted', 'inferred', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>
                  <Form.Item name="notes" label="说明"><Input.TextArea rows={2} /></Form.Item>
                  <Button type="primary" disabled={!governanceJob} loading={review.isPending} onClick={() => review.mutate()}>保存五类审核</Button>
                  {reviewRecorded && <Tag color="success">五类审核已保存</Tag>}
                </Form>
              </Card></Col>
            </Row>
            <Card size="small" title="C. 成本记录（确定性处理可选，外部付费处理必填）">
              <Form form={costForm} layout="inline" initialValues={{ kind: 'estimated', category: 'provider', provider: governanceJob.processor ?? 'deterministic', amount: '0', currency: 'USD', exchange_rate: '1', exchange_rate_source: 'Owner entered', billing_status: 'estimated' }}>
                <Form.Item name="kind" label="类型"><Select style={{ width: 110 }} options={['estimated', 'actual'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Form.Item name="category" label="类别" rules={[{ required: true }]}><Input style={{ width: 120 }} /></Form.Item>
                <Form.Item name="provider" label="处理方" rules={[{ required: true }]}><Input style={{ width: 120 }} /></Form.Item>
                <Form.Item name="amount" label="金额" rules={[{ required: true }, { pattern: /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/ }]}><Input style={{ width: 100 }} /></Form.Item>
                <Form.Item name="currency" label="币种"><Select style={{ width: 90 }} options={['USD', 'EUR', 'CNY', 'GBP', 'JPY'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Form.Item name="exchange_rate" label="汇率" rules={[{ required: true }]}><Input style={{ width: 90 }} /></Form.Item>
                <Form.Item name="exchange_rate_source" label="汇率来源" rules={[{ required: true }]}><Input style={{ width: 140 }} /></Form.Item>
                <Form.Item name="billing_status" label="账单状态"><Select style={{ width: 120 }} options={['estimated', 'pending', 'invoiced', 'paid', 'reconciled', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Button loading={cost.isPending} onClick={() => cost.mutate()}>保存成本</Button>
              </Form>
            </Card>
          </>}
        </Space>}
      </Card>
      <Card title="候选图片">
        {!completed.length ? <Empty description="只有 READY 且已有受控输出字节的任务才能成为候选" /> : <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="info" showIcon title="创建 Listing 图片集合" description="必须填写真实 Listing ID；集合创建后仍是草稿，需再次冻结 Owner 选择。" />
          <Form form={setForm} layout="vertical" initialValues={{ channel: 'ozon', locale: 'ru-RU' }}>
            <Row gutter={12}>
              <Col xs={24} md={8}><Form.Item name="listing_id" label="真实 Listing ID" rules={[{ required: true, message: '请填写真实 Listing ID，不会自动生成或使用 mock' }]}><InputNumber min={1} precision={0} style={{ width: '100%' }} placeholder="必须填写" /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input placeholder="例如 ozon" /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="locale" label="语言/地区" rules={[{ required: true }]}><Input placeholder="例如 ru-RU" /></Form.Item></Col>
            </Row>
          </Form>
          <Row gutter={[12, 12]}>{completed.map((job) => {
            const checked = selected.includes(job.id);
            const order = selected.indexOf(job.id);
            return <Col key={job.id} xs={24} sm={12} lg={8}><Card size="small" title={<Checkbox checked={checked} onChange={(event) => setSelected((current) => event.target.checked ? [...current, job.id] : current.filter((id) => id !== job.id))}>任务 {job.id}</Checkbox>}>
              <ProtectedOutput jobId={job.id} />
              <Space orientation="vertical" style={{ width: '100%', marginTop: 8 }}>
                <Select<ProductImageRole> disabled={!checked} value={roles[job.id] ?? (order === 0 ? 'main' : 'gallery')} style={{ width: '100%' }} onChange={(role) => setRoles((current) => ({ ...current, [job.id]: role }))} options={['main', 'gallery', 'detail', 'size', 'packaging', 'ad_cover'].map((value) => ({ value, label: value }))} />
                <Typography.Text type="secondary">{job.width} × {job.height} {job.format} · 顺序 {order >= 0 ? order + 1 : '未选择'}</Typography.Text>
              </Space>
            </Card></Col>;
          })}</Row>
          <Space>
            <Button type="primary" disabled={!selected.length || Boolean(imageSet)} loading={createSet.isPending} onClick={() => createSet.mutate()}>创建图片集合</Button>
            <Button danger disabled={!imageSet || imageSet.status === 'frozen'} loading={freezeSet.isPending} onClick={() => imageSet && freezeSet.mutate(imageSet.id)}>冻结 Owner 选择</Button>
          </Space>
          {imageSet && <Alert type={imageSet.status === 'frozen' ? 'success' : 'warning'} showIcon title={`图片集合 #${imageSet.id} · ${imageSet.status === 'frozen' ? '已冻结' : '草稿'}`} description={imageSet.status === 'frozen' ? `最终字节清单 SHA-256：${imageSet.manifest_sha256}` : '尚未冻结，不得作为发布放行依据。'} />}
        </Space>}
      </Card>
    </Space>
  </div>;
}
