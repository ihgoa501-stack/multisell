'use client';

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Alert, Button, Card, Col, Descriptions, Form, Image, Input, InputNumber, Row, Select, Space, Table, Tag, Typography, message } from 'antd';
import apiClient from '@/lib/api-client';

const { Text } = Typography;

type Mapping = { id: number; supplier_sku: string; internal_sku: string; channel_sku: string };
type RightsEvidence = { id: number; version: number; status: string; effective_status: string; license_scope: string; countries: string[]; channels: string[]; licensor: string; source_uri: string; observed_at: string; valid_until?: string; reviewed_by?: number; reviewed_at?: string; review_note?: string };
type Rendition = { id: number; content_url?: string; processed_url?: string; content_sha256?: string; width?: number; height?: number };
export type MaterialAsset = {
  id: number; role: 'main' | 'gallery' | 'sku' | 'detail' | 'video'; ordinal: number; canonical_sku_mapping_id?: number;
  source_url: string; source_sha256: string; mime_type: string; media_type: 'image' | 'video'; width?: number; height?: number;
	  duration_ms?: number; byte_size: number; processing_status: 'ready' | 'pending' | 'blocked'; blocker?: string; archived_at?: string;
	  used_at?: string;
  preview_url?: string;
  latest_rights?: RightsEvidence; rights_versions: RightsEvidence[]; renditions: Rendition[];
};
type MaterialResponse = { assets: MaterialAsset[] };

const roleLabels = { main: '主图', gallery: '附图', sku: 'SKU图', detail: '详情图', video: '视频' } as const;

export function materialBlockers(asset: MaterialAsset) {
  const blockers: string[] = [];
  if (asset.archived_at) blockers.push('素材已归档');
  if (asset.media_type === 'video') blockers.push(asset.blocker || '当前没有视频处理器，禁止进入草稿');
  else if (asset.processing_status !== 'ready') blockers.push(asset.blocker || '图片处理尚未完成');
  if (!asset.latest_rights) blockers.push('缺少权利证据');
  else if (asset.latest_rights.effective_status !== 'approved') blockers.push(`权利状态：${asset.latest_rights.effective_status}`);
  if (asset.role === 'sku' && !asset.canonical_sku_mapping_id) blockers.push('SKU图未绑定canonical SKU');
  return blockers;
}

function previewURL(asset: MaterialAsset) {
  const rendition = asset.renditions?.[0];
  return asset.preview_url || rendition?.content_url || rendition?.processed_url || asset.source_url;
}

export default function MaterialAssetWorkspace({ sourceID, taskLinkID, snapshotID, mappings, readOnly = false }: { sourceID: number; taskLinkID: number; snapshotID?: number; mappings: Mapping[]; readOnly?: boolean }) {
  const qc = useQueryClient();
	  const [assetForm] = Form.useForm();
	  const [rightsForm] = Form.useForm();
	  const [processingRecordIDs, setProcessingRecordIDs] = useState<Record<number, number | null>>({});
  const path = `/v1/sourcing-1688/${sourceID}/task-links/${taskLinkID}/material-assets`;
  const query = useQuery({ queryKey: ['sourcing-material-assets', sourceID, taskLinkID], queryFn: () => apiClient.get<MaterialResponse>(path) });
  const assets = query.data?.data?.assets ?? [];
  const invalidate = () => qc.invalidateQueries({ queryKey: ['sourcing-material-assets', sourceID, taskLinkID] });
  const create = useMutation({ mutationFn: (values: Record<string, unknown>) => {
    if (!snapshotID) throw new Error('缺少不可变来源快照');
    return apiClient.post(path, { ...values, snapshot_id: snapshotID, countries: undefined, channels: undefined });
  }, onSuccess: async () => { message.success('素材来源已保存；权利和处理仍需分别核验'); assetForm.resetFields(); await invalidate(); }, onError: (error: Error) => message.error(error.message) });
  const order = useMutation({ mutationFn: ({ id, ordinal }: { id: number; ordinal: number }) => apiClient.patch(`${path}/${id}/order`, { ordinal }), onSuccess: invalidate, onError: (error: Error) => message.error(error.message) });
  const addRights = useMutation({ mutationFn: ({ assetID, values }: { assetID: number; values: Record<string, unknown> }) => apiClient.post(`${path}/${assetID}/rights-evidence`, { ...values, countries: String(values.countries ?? '').split(',').map((v) => v.trim()).filter(Boolean), channels: String(values.channels ?? '').split(',').map((v) => v.trim()).filter(Boolean), observed_at: new Date(String(values.observed_at)).toISOString(), valid_until: values.valid_until ? new Date(String(values.valid_until)).toISOString() : undefined }), onSuccess: async () => { rightsForm.resetFields(); await invalidate(); }, onError: (error: Error) => message.error(error.message) });
	  const review = useMutation({ mutationFn: ({ assetID, evidenceID, decision }: { assetID: number; evidenceID: number; decision: 'approved' | 'rejected' }) => apiClient.post(`${path}/${assetID}/rights-evidence/${evidenceID}/review`, { decision, review_note: decision === 'approved' ? 'Owner已核验许可方、范围、渠道与有效期' : 'Owner判断权利证据不足' }), onSuccess: invalidate, onError: (error: Error) => message.error(error.message) });
	  const attachRendition = useMutation({ mutationFn: ({ assetID, imageProcessingRecordID }: { assetID: number; imageProcessingRecordID: number }) => apiClient.post(`${path}/${assetID}/renditions`, { image_processing_record_id: imageProcessingRecordID }), onSuccess: async () => { message.success('受控图片处理记录已绑定'); await invalidate(); }, onError: (error: Error) => message.error(error.message) });
	  const markUsed = useMutation({ mutationFn: (assetID: number) => apiClient.post(`${path}/${assetID}/mark-used`, {}), onSuccess: async () => { message.success('素材已明确加入当前任务草稿候选'); await invalidate(); }, onError: (error: Error) => message.error(error.message) });
	  const archive = useMutation({ mutationFn: (assetID: number) => apiClient.post(`${path}/${assetID}/archive`, {}), onSuccess: invalidate, onError: (error: Error) => message.error(error.message) });

  return <Card title="Owner 素材工作台">
    <Alert type="info" showIcon title="素材来源、权利、处理结果分开保存" description="图片可预览、排序并绑定canonical SKU；视频在处理器缺失期间明确阻塞，不会伪装成可发布素材。" style={{ marginBottom: 12 }} />
    {readOnly && <Alert type="warning" showIcon title="草稿已提交审批，素材只读" description="如需更换素材，应退回编辑并产生新的草稿审批指纹。" style={{ marginBottom: 12 }} />}
    {!readOnly && <Form form={assetForm} layout="vertical" initialValues={{ role: 'gallery', ordinal: assets.length + 1, media_type: 'image', mime_type: 'image/jpeg', byte_size: 0 }} onFinish={(values) => create.mutate(values)}>
      <Row gutter={12}>
        <Col xs={24} md={6}><Form.Item name="role" label="素材分类" rules={[{ required: true }]}><Select options={Object.entries(roleLabels).map(([value, label]) => ({ value, label }))} /></Form.Item></Col>
        <Col xs={24} md={6}><Form.Item name="ordinal" label="排序" rules={[{ required: true }]}><InputNumber min={1} /></Form.Item></Col>
        <Col xs={24} md={6}><Form.Item name="media_type" label="媒体类型" rules={[{ required: true }]}><Select options={[{ value: 'image', label: '图片' }, { value: 'video', label: '视频（当前阻塞）' }]} /></Form.Item></Col>
        <Col xs={24} md={6}><Form.Item name="canonical_sku_mapping_id" label="绑定canonical SKU"><Select allowClear options={mappings.map((m) => ({ value: m.id, label: `${m.supplier_sku} → ${m.internal_sku} → ${m.channel_sku}` }))} /></Form.Item></Col>
      </Row>
      <Form.Item name="source_url" label="来源URL" rules={[{ required: true }, { type: 'url' }]}><Input /></Form.Item>
      <Row gutter={12}><Col span={12}><Form.Item name="source_sha256" label="来源SHA-256" rules={[{ required: true, len: 64 }]}><Input /></Form.Item></Col><Col span={6}><Form.Item name="mime_type" label="MIME" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={6}><Form.Item name="byte_size" label="字节数" rules={[{ required: true }]}><InputNumber min={0} style={{ width: '100%' }} /></Form.Item></Col></Row>
      <Button type="primary" htmlType="submit" disabled={!snapshotID} loading={create.isPending}>保存素材来源</Button>
    </Form>}
	    <Table rowKey="id" pagination={false} dataSource={assets} expandable={{ expandedRowRender: (asset) => <Space orientation="vertical" style={{ width: '100%' }}>
	      <Descriptions bordered size="small" column={1}><Descriptions.Item label="来源哈希"><Text copyable>{asset.source_sha256}</Text></Descriptions.Item><Descriptions.Item label="权利版本">{asset.rights_versions?.length ?? 0}</Descriptions.Item></Descriptions>
	      {!readOnly && <Form form={rightsForm} layout="inline" onFinish={(values) => addRights.mutate({ assetID: asset.id, values })}><Form.Item name="license_scope" rules={[{ required: true }]}><Input placeholder="许可范围" /></Form.Item><Form.Item name="countries" rules={[{ required: true }]}><Input placeholder="国家,逗号分隔" /></Form.Item><Form.Item name="channels" rules={[{ required: true }]}><Input placeholder="渠道,逗号分隔" /></Form.Item><Form.Item name="licensor" rules={[{ required: true }]}><Input placeholder="许可方" /></Form.Item><Form.Item name="source_uri" rules={[{ required: true }]}><Input placeholder="权利证据URI" /></Form.Item><Form.Item name="observed_at" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item><Button htmlType="submit">追加权利版本</Button></Form>}
	      {!readOnly && asset.media_type === 'image' && asset.renditions.length === 0 && <Space wrap>
	        <InputNumber aria-label={`素材${asset.id}图片处理记录ID`} min={1} placeholder="图片处理记录ID" value={processingRecordIDs[asset.id]} onChange={(value) => setProcessingRecordIDs((current) => ({ ...current, [asset.id]: value }))} />
	        <Button disabled={!processingRecordIDs[asset.id]} loading={attachRendition.isPending} onClick={() => attachRendition.mutate({ assetID: asset.id, imageProcessingRecordID: processingRecordIDs[asset.id]! })}>绑定受控处理结果</Button>
	      </Space>}
	    </Space> }} columns={[
      { title: '预览', width: 100, render: (_, asset) => asset.media_type === 'image' ? <Image width={72} height={72} style={{ objectFit: 'cover' }} src={previewURL(asset)} alt={`${roleLabels[asset.role]}预览`} /> : <Tag color="red">视频blocked</Tag> },
      { title: '分类 / 排序', width: 150, render: (_, asset) => <Space orientation="vertical"><Tag>{roleLabels[asset.role]}</Tag><InputNumber size="small" min={1} value={asset.ordinal} disabled={readOnly} onChange={(value) => value && order.mutate({ id: asset.id, ordinal: value })} /></Space> },
      { title: 'SKU绑定', width: 170, render: (_, asset) => asset.canonical_sku_mapping_id ? <Text>mapping #{asset.canonical_sku_mapping_id}</Text> : asset.role === 'sku' ? <Tag color="gold">未绑定</Tag> : <Text type="secondary">不适用</Text> },
      { title: '权利状态', width: 220, render: (_, asset) => asset.latest_rights ? <Space orientation="vertical"><Tag color={asset.latest_rights.effective_status === 'approved' ? 'green' : 'gold'}>{asset.latest_rights.effective_status}</Tag><Text type="secondary">{asset.latest_rights.licensor} · {asset.latest_rights.license_scope}</Text>{!readOnly && asset.latest_rights.status === 'pending' && <Space><Button size="small" onClick={() => review.mutate({ assetID: asset.id, evidenceID: asset.latest_rights!.id, decision: 'approved' })}>Owner批准</Button><Button size="small" danger onClick={() => review.mutate({ assetID: asset.id, evidenceID: asset.latest_rights!.id, decision: 'rejected' })}>拒绝</Button></Space>}</Space> : <Tag color="gold">缺权利证据</Tag> },
	      { title: '处理 / blocker', render: (_, asset) => { const blockers = materialBlockers(asset); return <Space orientation="vertical">{blockers.length ? <Space wrap>{blockers.map((item) => <Tag color="red" key={item}>{item}</Tag>)}</Space> : asset.used_at ? <Tag color="green">已加入草稿候选</Tag> : <Tag color="blue">处理与权利已通过，待明确加入</Tag>}{!readOnly && !asset.archived_at && <Space><Button size="small" type="primary" disabled={blockers.length > 0 || !!asset.used_at} loading={markUsed.isPending} onClick={() => markUsed.mutate(asset.id)}>加入草稿候选</Button><Button size="small" danger loading={archive.isPending} onClick={() => archive.mutate(asset.id)}>归档</Button></Space>}</Space>; } },
	    ]} />
  </Card>;
}
