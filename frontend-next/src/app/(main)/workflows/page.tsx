'use client';

import { useState } from 'react';
import { Card, Table, Tag, Button, Modal, Form, Input, InputNumber, message, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';
import type { PageResult } from '@/types/api';

interface WorkflowDef {
  id: number;
  name: string;
  status: string;
  trigger: string;
  last_run_at?: string;
  created_at: string;
}

export default function WorkflowsPage() {
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  const { data, isLoading } = useQuery({
    queryKey: ['workflows', page],
    queryFn: async () => {
      const res = await apiClient.getPage<WorkflowDef>(`/v1/workflows?page=${page}&size=20`);
      return res;
    },
  });

  const create = useMutation({
    mutationFn: async (values: { name: string }) => {
      await apiClient.post('/v1/workflows', values);
    },
    onSuccess: () => {
      message.success('工作流已创建');
      setModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['workflows'] });
    },
  });

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name', render: (v: string, r: WorkflowDef) => <a href={`/workflows/${r.id}`}>{v}</a> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 100, render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag> },
    { title: '触发器', dataIndex: 'trigger', key: 'trigger', width: 100 },
    { title: '最后运行', dataIndex: 'last_run_at', key: 'last_run_at', width: 160, render: (v?: string) => v ? v.slice(0, 16).replace('T', ' ') : '-' },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 160, render: (v: string) => v.slice(0, 16).replace('T', ' ') },
  ];

  return (
    <div style={{ padding: 'var(--space-xl)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--space-lg)' }}>
        <Typography.Title level={3} style={{ margin: 0 }}>工作流管理</Typography.Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>新建工作流</Button>
      </div>

      <Card style={{ borderRadius: 'var(--r4)' }}>
        <Table
          rowKey="id"
          loading={isLoading}
          dataSource={data?.data ?? []}
          columns={columns}
          pagination={{
            current: page,
            pageSize: 20,
            total: data?.total ?? 0,
            onChange: setPage,
          }}
          size="middle"
        />
      </Card>

      <Modal title="新建工作流" open={modalOpen} onOk={() => form.submit()} onCancel={() => setModalOpen(false)}>
        <Form form={form} layout="vertical" onFinish={(v) => create.mutate(v)}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
