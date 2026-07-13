'use client';

import { useState } from 'react';
import { Alert, Button, Card, Checkbox, Col, Descriptions, Empty, Form, Input, message, Modal, Popconfirm, Row, Select, Space, Statistic, Table, Tag, Typography } from 'antd';
import { ArrowLeftOutlined, AuditOutlined, CheckCircleOutlined, LinkOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import { formatBlocker, gateMeta, nextExperimentStage, stageGateCodes, stageLabels, truthMeta } from '@/lib/experiment-display';
import { experimentStages, type EvidenceTruth, type ExperimentDetail, type ExperimentOwnerSummary, type ExperimentStage, type GateResult } from '@/types/experiment';

const { Text } = Typography;
const truthOptions = (Object.keys(truthMeta) as EvidenceTruth[])
  .filter((value) => value !== 'actual')
  .map((value) => ({ value, label: truthMeta[value].label }));
const gateOptions = (Object.keys(gateMeta) as GateResult[]).map((value) => ({ value, label: gateMeta[value].label }));

function useExperimentPost(path: string, onSuccess: () => void) {
  return useMutation({
    mutationFn: (values: Record<string, unknown>) => apiClient.post(path, values),
    onSuccess,
    onError: (error: Error) => message.error(`操作失败：${error.message}`),
  });
}

export default function ExperimentDetailPage() {
  const { experimentId = '' } = useParams<{ experimentId: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const [gateOpen, setGateOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [evidenceForm] = Form.useForm();
  const [gateForm] = Form.useForm();
  const [linkForm] = Form.useForm();
  const detail = useQuery({ queryKey: ['experiment', experimentId], queryFn: async () => (await apiClient.get<ExperimentDetail>(`/v1/experiments/${experimentId}`)).data, enabled: !!experimentId });
  const summary = useQuery({ queryKey: ['experiment-summary', experimentId], queryFn: async () => (await apiClient.get<ExperimentOwnerSummary>(`/v1/experiments/${experimentId}/owner-summary`)).data, enabled: !!experimentId });
  const refresh = () => { void qc.invalidateQueries({ queryKey: ['experiment', experimentId] }); void qc.invalidateQueries({ queryKey: ['experiment-summary', experimentId] }); };
  const addEvidence = useExperimentPost(`/v1/experiments/${experimentId}/evidence`, () => { message.success('证据已写入案卷'); setEvidenceOpen(false); evidenceForm.resetFields(); refresh(); });
  const evaluateGate = useExperimentPost(`/v1/experiments/${experimentId}/gates/evaluate`, () => { message.success('闸门判断已记录'); setGateOpen(false); gateForm.resetFields(); refresh(); });
  const addLink = useExperimentPost(`/v1/experiments/${experimentId}/links`, () => { message.success('经营对象已关联'); setLinkOpen(false); linkForm.resetFields(); refresh(); });
  const verifyEvidence = useMutation({
    mutationFn: (evidenceId: number) => apiClient.post(`/v1/experiments/${experimentId}/evidence/${evidenceId}/verify`, {}),
    onSuccess: () => { message.success('Owner 已核验，证据升级为真实发生'); refresh(); },
    onError: (error: Error) => message.error(`核验失败：${error.message}`),
  });
  const updateCase = useMutation({
    mutationFn: (overrides: Record<string, unknown>) => {
      if (!detail.data) throw new Error('案件尚未加载');
      return apiClient.put(`/v1/experiments/${experimentId}`, { ...detail.data.case, ...overrides });
    },
    onSuccess: () => { message.success('案件状态已更新'); refresh(); },
    onError: (error: Error) => message.error(`推进失败：${error.message}`),
  });
  const data = detail.data;
  const owner = summary.data;
  const currentGate = data ? [...data.gates].reverse().find((gate) => gate.stage === data.case.stage) : undefined;
  const currentGatePassed = currentGate?.result === 'pass';
  const nextStage = data ? nextExperimentStage(data.case.stage) : null;

  const openCurrentGate = () => {
    if (!data) return;
    gateForm.setFieldsValue({ stage: data.case.stage, gate_code: stageGateCodes[data.case.stage], result: 'return', evidence_ids: [] });
    setGateOpen(true);
  };

  const advanceCase = () => {
    if (!data || !nextStage) return;
    updateCase.mutate({ stage: nextStage });
  };

  return (
    <PageContainer
      title={data?.case.name ?? '经营事实核验案卷'}
      subtitle={experimentId}
      loading={detail.isLoading}
      error={detail.isError}
      errorMsg={(detail.error as Error | undefined)?.message}
      onRetry={() => void detail.refetch()}
      extra={<Space><Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/experiments')}>全部案件</Button><Button icon={<ReloadOutlined />} onClick={refresh}>刷新</Button></Space>}
    >
      {data && <>
        <Alert style={{ marginBottom: 16 }} showIcon type="warning" title="此模块仅作历史经营事实追踪" description="对象关联、闸门通过、订单终态、最终利润或现金到账都不证明因果关系，也不能把案卷标记为经营反馈闭环完成。" />
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col xs={24} md={12}><Card><Statistic title="当前阶段" value={stageLabels[data.case.stage] ?? data.case.stage} /></Card></Col>
          <Col xs={24} md={12}><Card><Statistic title="已通过闸门" value={owner?.passed_gates ?? 0} suffix="项" /></Card></Col>
        </Row>

        <Card title="当前阻塞" style={{ marginBottom: 16 }}>
          {owner?.blockers?.length ? <Space wrap>{owner.blockers.map((item) => <Tag color="red" key={item}>{formatBlocker(item)}</Tag>)}</Space> : <Alert type="success" showIcon title="当前没有已记录的阻塞项" />}
        </Card>

        <Card title="当前阶段操作" style={{ marginBottom: 16 }}>
          {data.case.stage === 'decision' ? (
            <Alert type="error" showIcon title="历史决策阶段已冻结" description="该案卷不得作为最终决定或反馈闭环的权威来源。" />
          ) : !currentGatePassed ? (
            <Alert type="warning" showIcon title={`先完成当前闸门：${stageGateCodes[data.case.stage]}`} description="只有当前阶段最新判断为“通过”，案件才允许进入下一阶段。" action={<Button type="primary" onClick={openCurrentGate}>评估当前闸门</Button>} />
          ) : data.case.stage === 'cash' ? (
            <Alert type="info" showIcon title="交易事实已追踪至现金阶段" description="案卷在此停止；这不是因果结论、最终经营决定或反馈闭环完成。" />
          ) : (
            <Space orientation="vertical">
              <Alert type="success" showIcon title="当前阶段闸门已通过" />
              <Popconfirm title={`确认推进到“${nextStage ? stageLabels[nextStage] : ''}”？`} description="只推进历史事实追踪阶段，不会形成利润、现金或经营决定终局。" onConfirm={advanceCase}>
                <Button type="primary" icon={<CheckCircleOutlined />} loading={updateCase.isPending}>{`推进到${nextStage ? stageLabels[nextStage] : '下一阶段'}`}</Button>
              </Popconfirm>
            </Space>
          )}
        </Card>

        <Card title="事实核验阶段（非因果、非反馈闭环）" style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', gap: 8, overflowX: 'auto', paddingBottom: 4 }}>
            {experimentStages.map((stage, index) => <div key={stage} style={{ minWidth: 112, padding: 12, borderRadius: 8, border: stage === data.case.stage ? '2px solid #1677ff' : '1px solid var(--border)', background: stage === data.case.stage ? 'var(--brand-bg, #e6f4ff)' : 'var(--surface)' }}><Text type="secondary">{String(index + 1).padStart(2, '0')}</Text><div><Text strong={stage === data.case.stage}>{stageLabels[stage]}</Text></div></div>)}
          </div>
        </Card>

        <Row gutter={[16, 16]}>
          <Col xs={24} xl={14}>
            <Card title="证据账本" extra={<Button size="small" icon={<PlusOutlined />} onClick={() => setEvidenceOpen(true)}>添加证据</Button>}>
              <Alert style={{ marginBottom: 12 }} showIcon type="info" title="真实、报价、估算、未知、模拟和 AI 推断必须明确区分；模拟、未知和推断不能通过高风险闸门。" />
              <Table rowKey="id" size="small" pagination={false} dataSource={data.evidence} locale={{ emptyText: <Empty description="尚无证据，案件不能凭空前进" /> }} columns={[
                { title: '证据', dataIndex: 'title', render: (v, row) => <Space orientation="vertical" size={0}><Text strong>{v}</Text>{row.source_uri && <a href={row.source_uri} target="_blank" rel="noreferrer">查看来源</a>}{row.truth_status === 'unknown' && row.source_uri && row.observed_at && <Popconfirm title="确认你已亲自核验来源与观察时间？" description="核验后该记录将升级为“真实发生”，并留下 Owner 身份。报价、估算、模拟和 AI 推断不能通过核验改写成真实事件。" onConfirm={() => verifyEvidence.mutate(row.id)}><Button type="link" size="small" style={{ padding: 0 }}>Owner 核验为真实</Button></Popconfirm>}</Space> },
                { title: '作用', dataIndex: 'evidence_kind', width: 90, render: (v: string) => <Tag color={v === 'counter' ? 'red' : v === 'conflict' ? 'orange' : 'blue'}>{v === 'counter' ? '反证' : v === 'conflict' ? '冲突' : '支持'}</Tag> },
                { title: '真实性', dataIndex: 'truth_status', width: 110, render: (v: EvidenceTruth) => <Tag color={truthMeta[v]?.color}>{truthMeta[v]?.label ?? v}</Tag> },
                { title: '阶段', dataIndex: 'stage', width: 110, render: (v: ExperimentStage) => stageLabels[v] ?? v },
                { title: '观察时间', dataIndex: 'observed_at', width: 120, render: (v?: string) => v ? dayjs(v).format('YYYY-MM-DD') : '未记录' },
              ]} />
            </Card>
          </Col>
          <Col xs={24} xl={10}>
            <Card title="经营闸门" extra={<Button size="small" icon={<AuditOutlined />} onClick={openCurrentGate}>记录评估</Button>} style={{ marginBottom: 16 }}>
              {data.gates.length ? <Space orientation="vertical" style={{ width: '100%' }}>{data.gates.map((gate) => <div key={gate.id} style={{ borderBottom: '1px solid var(--border)', paddingBottom: 10, width: '100%' }}><Space><Text strong>{gate.gate_code}</Text><Tag color={gateMeta[gate.result]?.color}>{gateMeta[gate.result]?.label ?? gate.result}</Tag></Space><div><Text type="secondary">{gate.reason || '未记录判断理由'}</Text></div></div>)}</Space> : <Empty description="尚未评估经营闸门" />}
            </Card>
            <Card title="关联经营对象" extra={<Button size="small" icon={<LinkOutlined />} onClick={() => setLinkOpen(true)}>添加关联</Button>}>
              {data.object_links.length ? <Descriptions column={1} size="small" items={data.object_links.map((link) => ({ key: link.id, label: link.object_type, children: <Text copyable>{link.object_id}</Text> }))} /> : <Empty description="尚未关联商品、供应商、SKU、订单或结算对象" />}
            </Card>
          </Col>
        </Row>

        <Modal title="添加证据" open={evidenceOpen} onCancel={() => setEvidenceOpen(false)} onOk={() => evidenceForm.validateFields().then((v) => addEvidence.mutate(v))} confirmLoading={addEvidence.isPending} okText="写入证据账本">
          <Alert type="info" showIcon style={{ marginBottom: 16 }} title="普通录入不能直接标记为“真实发生”。填写来源和观察时间后，可在证据账本中由 Owner 独立核验。" />
          <Form form={evidenceForm} layout="vertical" initialValues={{ stage: data.case.stage, evidence_kind: 'support', truth_status: 'unknown' }}>
            <Form.Item name="title" label="证据说明" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item label="所属阶段"><Tag color="blue">{stageLabels[data.case.stage]}</Tag></Form.Item>
            <Form.Item name="stage" hidden><Input /></Form.Item>
            <Form.Item name="evidence_kind" label="证据作用" rules={[{ required: true }]}><Select options={[{ value: 'support', label: '支持证据' }, { value: 'counter', label: '反证' }, { value: 'conflict', label: '冲突证据' }]} /></Form.Item>
            <Form.Item name="truth_status" label="真实性" rules={[{ required: true }]}><Select options={truthOptions} /></Form.Item>
            <Form.Item name="source_uri" label="来源链接"><Input placeholder="可审计的网页、文件或系统记录地址" /></Form.Item>
            <Form.Item name="observed_at" label="观察时间（ISO 8601）"><Input placeholder="2026-07-11T10:00:00+08:00" /></Form.Item>
            <Form.Item name="expires_at" label="失效时间（可选）"><Input placeholder="证据过期后应重新核验" /></Form.Item>
          </Form>
        </Modal>
        <Modal title="评估经营闸门" open={gateOpen} onCancel={() => setGateOpen(false)} onOk={() => gateForm.validateFields().then((v) => evaluateGate.mutate(v))} confirmLoading={evaluateGate.isPending} okText="记录判断">
          <Alert type="warning" showIcon style={{ marginBottom: 16 }} title="这是判断记录，不会自动采购、发布、退款或移动资金。高风险阶段选择“通过”时必须勾选可信证据。" />
          <Form form={gateForm} layout="vertical" initialValues={{ stage: data.case.stage, gate_code: stageGateCodes[data.case.stage], result: 'return', evidence_ids: [] }}>
            <Form.Item label="当前闸门"><Text code>{stageGateCodes[data.case.stage]}</Text><Text type="secondary"> · {stageLabels[data.case.stage]}</Text></Form.Item>
            <Form.Item name="gate_code" hidden><Input /></Form.Item>
            <Form.Item name="stage" hidden><Input /></Form.Item>
            <Form.Item name="result" label="判断结果" rules={[{ required: true }]}><Select options={gateOptions} /></Form.Item>
            <Form.Item name="reason" label="理由"><Input.TextArea rows={3} /></Form.Item>
            <Form.Item name="evidence_ids" label="引用当前阶段证据"><Checkbox.Group options={data.evidence.filter((e) => e.stage === data.case.stage).map((e) => ({ value: e.id, label: `${e.title}（${truthMeta[e.truth_status].label}）` }))} /></Form.Item>
          </Form>
        </Modal>
        <Modal title="关联现有经营对象" open={linkOpen} onCancel={() => setLinkOpen(false)} onOk={() => linkForm.validateFields().then((v) => addLink.mutate(v))} confirmLoading={addLink.isPending} okText="建立关联">
          <Form form={linkForm} layout="vertical"><Form.Item name="object_type" label="对象类型" rules={[{ required: true }]}><Select options={['candidate', 'product', 'product_spec', 'supplier', 'sku', 'purchase', 'inventory_batch', 'order', 'shipment', 'aftersale', 'settlement', 'profit_record', 'cash_transaction'].map((value) => ({ value, label: value }))} /></Form.Item><Form.Item name="object_id" label="对象编号" rules={[{ required: true }]}><Input /></Form.Item></Form>
        </Modal>
      </>}
    </PageContainer>
  );
}
