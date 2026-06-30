'use client';

import {
  Button, Card, Empty, Form, Input, InputNumber, Modal, Select, Space, Spin, Table, Tag, message,
} from 'antd';
import { PlusOutlined, ReloadOutlined, StepForwardOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import { useState } from 'react';

interface OrchestrationConfig {
  id: number;
  name: string;
  steps: string;
  failure_action: string;
  auto_approve_pct: number;
  auto_retry_count: number;
  created_at: string;
}

interface LifecycleStep {
  id: number;
  product_id: number;
  step: string;
  agent_id: string;
  status: string;
  started_at?: string;
  completed_at?: string;
  duration_ms: number;
  result?: string;
  error?: string;
  created_at: string;
}

const STEP_ORDER = ['sourcing', 'enrichment', 'compliance', 'pricing', 'listing', 'monitoring', 'delisting'];

const STEP_LABELS: Record<string, string> = {
  sourcing: '选品',
  enrichment: '内容生成',
  compliance: '合规检查',
  pricing: '定价',
  listing: '刊登',
  monitoring: '监控',
  delisting: '下架',
};

const STEP_AGENTS: Record<string, string> = {
  sourcing: 'A8',
  enrichment: 'Content AI',
  compliance: 'G3',
  pricing: 'A3',
  listing: 'A2',
  monitoring: 'A5',
  delisting: '定时任务',
};

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  completed: 'success',
  failed: 'error',
  skipped: 'warning',
};

const FAILURE_ACTIONS = ['stop', 'skip', 'retry'];

export default function OrchestrationPage() {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [productIdInput, setProductIdInput] = useState('');
  const [viewProductId, setViewProductId] = useState<number | null>(null);

  // Fetch configs
  const { data: configs, isLoading: configsLoading } = useQuery({
    queryKey: ['orchestration', 'configs'],
    queryFn: async () => {
      const res = await apiClient.get<OrchestrationConfig[]>('/v1/orchestration/pipeline/config');
      return res.data ?? [];
    },
  });

  // Fetch pipeline for a specific product
  const { data: pipelineSteps, isLoading: pipelineLoading } = useQuery({
    queryKey: ['orchestration', 'pipeline', viewProductId],
    queryFn: async () => {
      if (!viewProductId) return [];
      const res = await apiClient.get<LifecycleStep[]>(`/v1/orchestration/products/${viewProductId}/pipeline`);
      return res.data ?? [];
    },
    enabled: !!viewProductId,
  });

  // Start pipeline mutation
  const startPipeline = useMutation({
    mutationFn: async (productId: number) => {
      await apiClient.post(`/v1/orchestration/products/${productId}/pipeline/start`);
    },
    onSuccess: (_data, productId) => {
      message.success(`产品 ${productId} 生命周期已启动`);
      setViewProductId(productId);
      queryClient.invalidateQueries({ queryKey: ['orchestration', 'pipeline', productId] });
    },
    onError: (err: Error) => {
      message.error(`启动失败: ${err.message}`);
    },
  });

  // Create config mutation
  const createConfig = useMutation({
    mutationFn: async (values: {
      name: string;
      steps: string;
      failure_action: string;
      auto_approve_pct: number;
      auto_retry_count: number;
    }) => {
      await apiClient.post('/v1/orchestration/pipeline/config', values);
    },
    onSuccess: () => {
      message.success('配置已创建');
      setModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['orchestration', 'configs'] });
    },
    onError: (err: Error) => {
      message.error(`创建失败: ${err.message}`);
    },
  });

  const handleStartPipeline = () => {
    const pid = parseInt(productIdInput, 10);
    if (!pid || pid <= 0) {
      message.warning('请输入有效的产品 ID');
      return;
    }
    startPipeline.mutate(pid);
  };

  const configColumns = [
    { title: 'ID', dataIndex: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', width: 180 },
    {
      title: '步骤', dataIndex: 'steps', width: 300,
      render: (val: string) => {
        try {
          const steps = JSON.parse(val);
          return steps.map((s: string) => <Tag key={s} style={{ marginBottom: 4 }}>{STEP_LABELS[s] || s}</Tag>);
        } catch { return <Tag>默认流水线</Tag>; }
      },
    },
    {
      title: '失败策略', dataIndex: 'failure_action', width: 120,
      render: (val: string) => (
        <Tag color={val === 'stop' ? 'red' : val === 'skip' ? 'orange' : 'blue'}>{val}</Tag>
      ),
    },
    {
      title: '自动批准阈值(%)', dataIndex: 'auto_approve_pct', width: 140,
    },
    { title: '重试次数', dataIndex: 'auto_retry_count', width: 100 },
    {
      title: '创建时间', dataIndex: 'created_at', width: 160,
      render: (val: string) => val?.slice(0, 16).replace('T', ' '),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', marginBottom: 24 }}>
        产品生命周期编排
      </h1>

      {/* Start Pipeline Card */}
      <Card title="启动新产品生命周期" style={{ marginBottom: 24 }}>
        <Space>
          <Input
            placeholder="输入产品 ID"
            value={productIdInput}
            onChange={(e) => setProductIdInput(e.target.value)}
            style={{ width: 200 }}
            onPressEnter={handleStartPipeline}
          />
          <Button type="primary" icon={<StepForwardOutlined />} onClick={handleStartPipeline} loading={startPipeline.isPending}>
            启动流水线
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              if (viewProductId) {
                queryClient.invalidateQueries({ queryKey: ['orchestration', 'pipeline', viewProductId] });
              }
            }}
          >
            刷新
          </Button>
        </Space>
      </Card>

      {/* Pipeline Status Card */}
      {viewProductId && (
        <Card title={`产品 #${viewProductId} 生命周期`} style={{ marginBottom: 24 }}>
          <Spin spinning={pipelineLoading}>
            {!pipelineSteps || pipelineSteps.length === 0 ? (
              <Empty description="暂无生命周期数据" />
            ) : (
              <div>
                {/* Stepped Progress Bar */}
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 0, marginBottom: 24, overflowX: 'auto', padding: '16px 0' }}>
                  {pipelineSteps.map((step, index) => {
                    const isLast = index === pipelineSteps.length - 1;
                    const isActive = step.status === 'running';
                    const isCompleted = step.status === 'completed';
                    const isFailed = step.status === 'failed';
                    const isSkipped = step.status === 'skipped';

                    let stepColor = '#d9d9d9';
                    if (isCompleted) stepColor = '#52c41a';
                    else if (isActive) stepColor = '#1677ff';
                    else if (isFailed) stepColor = '#ff4d4f';
                    else if (isSkipped) stepColor = '#faad14';

                    return (
                      <div key={step.id} style={{ display: 'flex', alignItems: 'center', flex: 1, minWidth: 120 }}>
                        <div style={{ textAlign: 'center', flex: 1 }}>
                          <div style={{
                            width: 40, height: 40, borderRadius: '50%',
                            backgroundColor: stepColor, color: '#fff',
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                            margin: '0 auto', fontWeight: 700, fontSize: 14,
                            border: isActive ? '3px solid #91caff' : 'none',
                            boxShadow: isActive ? '0 0 0 3px #e6f4ff' : 'none',
                          }}>
                            {isCompleted ? '✓' : isFailed ? '✗' : index + 1}
                          </div>
                          <div style={{ marginTop: 8, fontWeight: 500, fontSize: 13 }}>{STEP_LABELS[step.step] || step.step}</div>
                          <div style={{ fontSize: 11, color: '#999' }}>{STEP_AGENTS[step.step] || step.agent_id}</div>
                          <div style={{ marginTop: 4 }}>
                            <Tag color={STATUS_COLORS[step.status] || 'default'}>{step.status}</Tag>
                          </div>
                          {step.duration_ms > 0 && (
                            <div style={{ fontSize: 11, color: '#999' }}>{Math.round(step.duration_ms / 1000)}s</div>
                          )}
                          {isFailed && step.error && (
                            <div style={{ fontSize: 11, color: '#ff4d4f', marginTop: 4, maxWidth: 150, wordBreak: 'break-word' }}>
                              {step.error}
                            </div>
                          )}
                        </div>
                        {!isLast && (
                          <div style={{
                            flex: 1, height: 2, backgroundColor: stepColor, margin: '0 4px', marginTop: -40,
                          }} />
                        )}
                      </div>
                    );
                  })}
                </div>

                {/* Detail Table */}
                <Table
                  rowKey="id"
                  dataSource={pipelineSteps}
                  pagination={false}
                  size="small"
                  columns={[
                    { title: '步骤', dataIndex: 'step', render: (val: string) => STEP_LABELS[val] || val },
                    { title: 'Agent', dataIndex: 'agent_id' },
                    {
                      title: '状态', dataIndex: 'status',
                      render: (val: string) => <Tag color={STATUS_COLORS[val] || 'default'}>{val}</Tag>,
                    },
                    { title: '耗时(ms)', dataIndex: 'duration_ms' },
                    {
                      title: '开始时间', dataIndex: 'started_at',
                      render: (val: string) => val ? val.slice(0, 16).replace('T', ' ') : '-',
                    },
                    {
                      title: '完成时间', dataIndex: 'completed_at',
                      render: (val: string) => val ? val.slice(0, 16).replace('T', ' ') : '-',
                    },
                    {
                      title: '操作', width: 100,
                      render: (_: unknown, record: LifecycleStep) => (
                        record.status === 'failed' ? (
                          <Button type="link" size="small" onClick={async () => {
                            try {
                              await apiClient.post(`/v1/orchestration/products/${record.product_id}/pipeline/step/${record.step}/retry`);
                              message.success('重试已触发');
                              queryClient.invalidateQueries({ queryKey: ['orchestration', 'pipeline', viewProductId] });
                            } catch (err: unknown) {
                              const e = err as Error;
                              message.error(`重试失败: ${e.message}`);
                            }
                          }}>
                            重试
                          </Button>
                        ) : null
                      ),
                    },
                  ]}
                />
              </div>
            )}
          </Spin>
        </Card>
      )}

      {/* Pipeline Configs Card */}
      <Card
        title="流水线配置"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
            新建配置
          </Button>
        }
      >
        <Table
          rowKey="id"
          loading={configsLoading}
          dataSource={configs}
          columns={configColumns}
          pagination={false}
          locale={{ emptyText: <Empty description="暂无配置（将使用默认流水线）" /> }}
        />
      </Card>

      {/* Create Config Modal */}
      <Modal
        title="新建流水线配置"
        open={modalOpen}
        onOk={() => form.submit()}
        onCancel={() => { setModalOpen(false); form.resetFields(); }}
        confirmLoading={createConfig.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values) => {
            createConfig.mutate({
              ...values,
              steps: JSON.stringify(values.steps || STEP_ORDER),
            });
          }}
          initialValues={{ failure_action: 'stop', auto_approve_pct: 80, auto_retry_count: 3, steps: STEP_ORDER }}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input placeholder="例如：默认流水线" />
          </Form.Item>
          <Form.Item name="steps" label="步骤顺序">
            <Select mode="multiple" placeholder="选择步骤" options={STEP_ORDER.map((s) => ({ label: STEP_LABELS[s] || s, value: s }))} />
          </Form.Item>
          <Form.Item name="failure_action" label="失败策略">
            <Select options={FAILURE_ACTIONS.map((s) => ({ label: s, value: s }))} />
          </Form.Item>
          <Form.Item name="auto_approve_pct" label="自动批准阈值 (%)">
            <InputNumber min={0} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="auto_retry_count" label="自动重试次数">
            <InputNumber min={0} max={10} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
