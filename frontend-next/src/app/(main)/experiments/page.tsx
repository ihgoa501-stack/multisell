'use client';

import { useState } from 'react';
import { Alert, Button, Card, Form, Input, message, Modal, Space, Table, Tag, Typography } from 'antd';
import { PlusOutlined, RightOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import { type ExperimentCase, type ExperimentStage } from '@/types/experiment';
import { stageLabels } from '@/lib/experiment-display';

const { Text } = Typography;

export default function ExperimentsPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<{ name: string }>();
  const query = useQuery({
    queryKey: ['experiments'],
    queryFn: async () => apiClient.getPage<ExperimentCase>('/v1/experiments', { page: '1', size: '100' }),
  });
  const create = useMutation({
    mutationFn: async (values: { name: string }) =>
      apiClient.post<ExperimentCase>('/v1/experiments', { ...values, stage: 'opportunity' }),
    onSuccess: (result) => {
      message.success('经营实验案件已创建');
      setOpen(false);
      form.resetFields();
      void qc.invalidateQueries({ queryKey: ['experiments'] });
      if (result.data?.experiment_id) router.push(`/experiments/${result.data.experiment_id}`);
    },
    onError: (error: Error) => message.error(`创建失败：${error.message}`),
  });

  return (
    <PageContainer
      title="经营实验案卷"
      subtitle="用一条可追溯事实链，把需求证据连接到最终利润、现金回收与退出决定。"
      loading={query.isLoading}
      error={query.isError}
      errorMsg={(query.error as Error | undefined)?.message}
      onRetry={() => void query.refetch()}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>创建真实案件</Button>}
    >
      <Card styles={{ body: { padding: 0 } }}>
        <Table
          rowKey="experiment_id"
          dataSource={query.data?.data ?? []}
          pagination={{ pageSize: 20, showTotal: (n) => `共 ${n} 个案件` }}
          onRow={(row) => ({ onClick: () => router.push(`/experiments/${row.experiment_id}`), style: { cursor: 'pointer' } })}
          columns={[
            { title: '实验案件', dataIndex: 'name', render: (name, row) => <Space orientation="vertical" size={0}><Text strong>{name}</Text><Text type="secondary" copyable>{row.experiment_id}</Text></Space> },
            { title: '当前阶段', dataIndex: 'stage', render: (stage: ExperimentStage) => <Tag color="blue">{stageLabels[stage] ?? stage}</Tag> },
            { title: '最终利润', dataIndex: 'final_profit_status', render: (v: string) => <Tag color={v === 'pending' ? 'default' : 'gold'}>{v || 'pending'}</Tag> },
            { title: '现金回收', dataIndex: 'cash_recovery_status', render: (v: string) => <Tag color={v === 'pending' ? 'default' : 'cyan'}>{v || 'pending'}</Tag> },
            { title: '最近更新', dataIndex: 'updated_at', render: (v: string) => v ? dayjs(v).format('MM-DD HH:mm') : '—' },
            { title: '', width: 40, render: () => <RightOutlined /> },
          ]}
        />
      </Card>
      <Modal title="创建经营实验案件" open={open} onCancel={() => setOpen(false)} onOk={() => form.validateFields().then((v) => create.mutate(v))} confirmLoading={create.isPending} okText="创建并进入案卷">
        <Alert type="info" showIcon style={{ marginBottom: 16 }} title="所有新案件从机会与需求阶段开始，不能跳过证据与反证。" />
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="案件名称" rules={[{ required: true, message: '请用业务语言描述本次实验' }]}><Input placeholder="例如：轻量宠物旅行水杯首轮实验" /></Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
}
