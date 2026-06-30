'use client';

import {
  Badge,
  Button,
  Card,
  Form,
  Input,
  message,
  Modal,
  Select,
  Space,
  Table,
} from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import apiClient from '@/lib/api-client';

interface WebhookConfig {
  platform: string;
  webhook_url: string;
  is_configured: boolean;
  last_event_at?: string;
  last_event_type?: string;
}

export default function WebhooksPage() {
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [form] = Form.useForm();
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['platform-webhooks-config'],
    queryFn: async () => {
      const res = await apiClient.get<WebhookConfig[]>('/v1/platform-webhooks/config');
      return res.data;
    },
  });

  const testMut = useMutation({
    mutationFn: async (vals: { platform: string; event_type: string; payload?: Record<string, unknown> }) => {
      return apiClient.post('/v1/platform-webhooks/test-event', vals);
    },
    onSuccess: () => {
      message.success('测试事件已发送');
      setTestModalOpen(false);
      form.resetFields();
      qc.invalidateQueries({ queryKey: ['platform-webhooks-config'] });
    },
    onError: (e: Error) => {
      message.error(
        (e as { response?: { data?: { message?: string } } })?.response?.data?.message || '发送失败',
      );
    },
  });

  const columns = [
    {
      title: '平台',
      dataIndex: 'platform',
      key: 'platform',
      render: (code: string) => (
        <span style={{ textTransform: 'uppercase' }}>{code}</span>
      ),
    },
    {
      title: 'Webhook 地址',
      dataIndex: 'webhook_url',
      key: 'webhook_url',
      render: (url: string) => (
        <code style={{ fontSize: 12 }}>{url}</code>
      ),
    },
    {
      title: '配置状态',
      dataIndex: 'is_configured',
      key: 'is_configured',
      render: (v: boolean) => v ? <Badge status="success" text="已配置" /> : <Badge status="default" text="未配置" />,
    },
    {
      title: '最近事件',
      dataIndex: 'last_event_type',
      key: 'last_event_type',
      render: (v: string) => v || '-',
    },
    {
      title: '最近接收时间',
      dataIndex: 'last_event_at',
      key: 'last_event_at',
      render: (v: string) => v ? new Date(v).toLocaleString() : '-',
    },
  ];

  const items = data ?? [];

  return (
    <Card title="平台 Webhooks 管理">
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" onClick={() => setTestModalOpen(true)}>
          发送测试事件
        </Button>
      </Space>

      <Table
        dataSource={items}
        columns={columns}
        rowKey="platform"
        loading={isLoading}
        pagination={false}
      />

      <Modal
        title="发送测试 Webhook 事件"
        open={testModalOpen}
        onCancel={() => { setTestModalOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={testMut.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(vals) => {
            const payload: Record<string, unknown> = {};
            if (vals.product_id) payload.product_id = vals.product_id;
            if (vals.sku_code) payload.sku_code = vals.sku_code;
            testMut.mutate({
              platform: vals.platform,
              event_type: vals.event_type,
              payload: Object.keys(payload).length > 0 ? payload : undefined,
            });
          }}
        >
          <Form.Item
            name="platform"
            label="平台"
            rules={[{ required: true, message: '请选择平台' }]}
          >
            <Select
              options={[
                { value: 'ozon', label: 'Ozon' },
                { value: 'shopee', label: 'Shopee' },
              ]}
            />
          </Form.Item>

          <Form.Item
            name="event_type"
            label="事件类型"
            rules={[{ required: true, message: '请选择事件类型' }]}
          >
            <Select
              options={[
                { value: 'listing_blocked', label: 'listing_blocked - 商品被屏蔽' },
                { value: 'listing_live', label: 'listing_live - 商品上架成功' },
                { value: 'price_changed', label: 'price_changed - 价格变更' },
                { value: 'inventory_changed', label: 'inventory_changed - 库存变更' },
              ]}
            />
          </Form.Item>

          <Form.Item name="product_id" label="商品 ID（可选）">
            <Input placeholder="平台商品 ID" />
          </Form.Item>

          <Form.Item name="sku_code" label="SKU 编码（可选）">
            <Input placeholder="SKU 编码" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
