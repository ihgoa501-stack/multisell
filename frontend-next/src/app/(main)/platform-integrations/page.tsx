'use client';

import { Button, Form, Input, message, Modal, Space, Table, Tag, Tooltip } from 'antd';
import { LinkOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import apiClient from '@/lib/api-client';

interface IntegrationAccount {
  id: number;
  platform_id: number;
  store_name: string;
  account_id: string;
  status: string;
  sync_status: string;
  last_sync_at?: string;
  last_error?: string;
  created_at: string;
}

export default function PlatformIntegrationsPage() {
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['platform-integrations'],
    queryFn: async () => {
      const res = await apiClient.get<{ items: IntegrationAccount[]; total: number }>(
        '/v1/platform-integrations?size=100'
      );
      return res.data;
    },
  });

  const createMut = useMutation({
    mutationFn: async (vals: { platform_id: number; store_name: string; access_token: string; config: string }) => {
      return apiClient.post('/v1/platform-integrations', {
        platform_id: vals.platform_id,
        store_name: vals.store_name,
        access_token: vals.access_token,
        config: JSON.stringify({ client_id: vals.config }),
        status: 'active',
      });
    },
    onSuccess: () => { message.success('Ozon 店铺已连接'); setModalOpen(false); form.resetFields(); qc.invalidateQueries({ queryKey: ['platform-integrations'] }); },
    onError: (e: any) => { message.error(e?.response?.data?.message || '连接失败'); },
  });

  const testMut = useMutation({
    mutationFn: (id: number) => apiClient.post(`/v1/platform-integrations/${id}/test`, {}),
    onSuccess: (res: any) => { message.success(res.data?.success ? '验证成功' : res.data?.message || '验证失败'); },
    onError: (e: any) => { message.error(e?.response?.data?.message || '验证失败'); },
  });

  const productsUrl = (id: number) => `/platform-integrations/${id}/ozon-products`;

  const items = data?.items ?? [];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h1 style={{ fontSize: 24, fontWeight: 600, margin: 0 }}>平台对接</h1>
        <Button type="primary" icon={<LinkOutlined />} onClick={() => setModalOpen(true)}>连接 Ozon 店铺</Button>
      </div>

      <Table rowKey="id" loading={isLoading} dataSource={items}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 60 },
          { title: '店铺名称', dataIndex: 'store_name', width: 180 },
          { title: '账号ID', dataIndex: 'account_id', width: 120 },
          { title: '状态', dataIndex: 'status', width: 90, render: (s: string) => <Tag color={s === 'active' ? 'success' : 'default'}>{s}</Tag> },
          { title: '同步状态', dataIndex: 'sync_status', width: 100, render: (s: string) => <Tag color={s === 'syncing' ? 'processing' : 'default'}>{s}</Tag> },
          { title: '上次同步', dataIndex: 'last_sync_at', width: 160, render: (s?: string) => s ? s.slice(0, 16).replace('T', ' ') : '-' },
          {
            title: '操作', width: 220,
            render: (_: any, r: IntegrationAccount) => (
              <Space>
                <Tooltip title="验证连接"><Button size="small" loading={testMut.isPending} onClick={() => testMut.mutate(r.id)}>验证</Button></Tooltip>
                <Button size="small" href={productsUrl(r.id)}>查看商品</Button>
              </Space>
            ),
          },
        ]}
        pagination={false} size="small"
      />

      <Modal title="连接 Ozon 店铺" open={modalOpen} onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()} confirmLoading={createMut.isPending}>
        <Form form={form} layout="vertical" onFinish={(v) => createMut.mutate(v)}>
          <Form.Item name="platform_id" initialValue={1} hidden><Input /></Form.Item>
          <Form.Item name="store_name" label="店铺名称" rules={[{ required: true }]}>
            <Input placeholder="例如：我的Ozon店" />
          </Form.Item>
          <Form.Item name="config" label="Client ID" rules={[{ required: true, message: '请输入Ozon Client ID' }]}>
            <Input placeholder="从Ozon Seller API获取" />
          </Form.Item>
          <Form.Item name="access_token" label="API Key" rules={[{ required: true, message: '请输入Ozon API Key' }]}>
            <Input.Password placeholder="从Ozon Seller API获取" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
