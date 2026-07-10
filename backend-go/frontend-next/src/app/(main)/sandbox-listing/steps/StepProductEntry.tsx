'use client';

import { useState } from 'react';
import { Form, Input, InputNumber, Select, Button, message } from 'antd';
import { useSandboxListingStore } from '../store';
import apiClient from '@/lib/api-client';

export default function StepProductEntry() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { goNext, setCandidateId } = useSandboxListingStore();

  const onFinish = async (values: Record<string, unknown>) => {
    setLoading(true);
    try {
      const res = await apiClient.post<{ id: number }>('/v1/candidates', values);
      if (!res.data) {
        message.error('录入失败：服务器未返回商品 ID');
        return;
      }
      setCandidateId(res.data.id);
      message.success('商品录入成功');
      goNext();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '录入失败';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 600 }}>
      <Form.Item name="title" label="商品名称" rules={[{ required: true }]}>
        <Input placeholder="例: Wireless Bluetooth Earbuds" />
      </Form.Item>
      <Form.Item name="purchase_price" label="采购价 (CNY)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="8.50" />
      </Form.Item>
      <Form.Item name="purchase_currency" label="采购币种" initialValue="CNY">
        <Select>
          <Select.Option value="CNY">CNY</Select.Option>
          <Select.Option value="USD">USD</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="package_weight_kg" label="重量 (kg)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="0.5" />
      </Form.Item>
      <Form.Item name="package_length_cm" label="长 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="package_width_cm" label="宽 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="package_height_cm" label="高 (cm)">
        <InputNumber min={0} style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item name="destination_country" label="目标国家" initialValue="US">
        <Select>
          <Select.Option value="US">美国</Select.Option>
          <Select.Option value="RU">俄罗斯</Select.Option>
          <Select.Option value="DE">德国</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="target_platform_id" label="目标平台" rules={[{ required: true }]}>
        <Select placeholder="选择平台">
          <Select.Option value={1}>Ozon (Mock)</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item name="target_sale_price" label="目标售价 (USD)" rules={[{ required: true }]}>
        <InputNumber min={0} step={0.01} style={{ width: '100%' }} placeholder="29.99" />
      </Form.Item>
      <Form.Item name="source_url" label="供应商链接">
        <Input placeholder="https://..." />
      </Form.Item>
      <Form.Item name="main_image" label="商品图 URL">
        <Input placeholder="https://..." />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading}>
          提交并继续
        </Button>
      </Form.Item>
    </Form>
  );
}
