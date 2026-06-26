'use client';

import { useState } from 'react';
import { Button, Form, Input, InputNumber, message, Modal, Select, Tag } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import CrudListPage, { fmtDate } from '@/components/crud/CrudListPage';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';

const STATUS_MAP: Record<string, { label: string; color: string }> = {
  '0': { label: '草稿', color: 'default' },
  '1': { label: '上架', color: 'success' },
  '2': { label: '下架', color: 'warning' },
};

export default function ProductsPage() {
  const [publishOpen, setPublishOpen] = useState(false);
  const [selectedProductId, setSelectedProductId] = useState<number | null>(null);
  const [form] = Form.useForm();
  const qc = useQueryClient();

  // Fetch Ozon accounts for the publish selector
  const { data: accountsData } = useQuery({
    queryKey: ['platform-integrations', 'ozon'],
    queryFn: async () => {
      const res = await apiClient.get<{ items: Array<Record<string, unknown>>; total: number }>('/v1/platform-integrations?size=50');
      return res.data?.items ?? [];
    },
  });

  const publishMut = useMutation({
    mutationFn: async (vals: { product_id: number; account_id: number; price: number }): Promise<Result<{ platform_url?: string; message?: string }>> => {
      return apiClient.post('/v1/platform-integrations/publish-to-ozon', vals);
    },
    onSuccess: (res) => {
      const data = res.data;
      message.success(data?.platform_url ? `已发布！链接: ${data.platform_url}` : '已提交到 Ozon');
      setPublishOpen(false);
      qc.invalidateQueries({ queryKey: ['products'] });
    },
    onError: (e: Error) => { message.error((e as { response?: { data?: { message?: string } } })?.response?.data?.message || '发布失败'); },
  });

  const ozonAccounts = (accountsData ?? []).filter((a: Record<string, unknown>) => a.platform_id === 1);

  return (
    <>
      <CrudListPage
        resource="/products"
        title="商品"
        singular="商品"
        searchPlaceholder="搜索商品名称 / 编码 / 副标题..."
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '商品名称', dataIndex: 'name', width: 200 },
          { title: '副标题', dataIndex: 'subtitle', width: 200 },
          { title: '品牌ID', dataIndex: 'brand_id', width: 90 },
          { title: '分类ID', dataIndex: 'category_id', width: 90 },
          { title: '单位', dataIndex: 'unit', width: 80 },
          { title: '状态', dataIndex: 'status', width: 100,
            render: (s: unknown) => {
              const v = STATUS_MAP[String(s)] || { label: String(s), color: 'default' };
              return <Tag color={v.color}>{v.label}</Tag>;
            },
          },
          { title: '货品类型', dataIndex: 'cargo_type', width: 110 },
          { title: '创建时间', dataIndex: 'created_at', width: 160, render: fmtDate },
          {
            title: '操作', dataIndex: 'id', width: 150,
            render: (_: unknown, r: Record<string, unknown>) => (
              <Button type="link" size="small" icon={<ExportOutlined />}
                onClick={() => { setSelectedProductId(r.id as number); setPublishOpen(true); }}>
                发布到 Ozon
              </Button>
            ),
          },
        ]}
        fields={[
          { name: 'name', label: '商品名称', required: true },
          { name: 'subtitle', label: '副标题' },
          { name: 'brand_id', label: '品牌ID', type: 'number' },
          { name: 'category_id', label: '分类ID', type: 'number' },
          { name: 'unit', label: '单位' },
          { name: 'status', label: '状态', initialValue: '1' },
          { name: 'main_image', label: '主图URL' },
          { name: 'cargo_type', label: '货品类型' },
        ]}
      />

      <Modal title="发布到 Ozon" open={publishOpen} onCancel={() => setPublishOpen(false)}
        onOk={() => form.submit()} confirmLoading={publishMut.isPending}>
        <Form form={form} layout="vertical" initialValues={{ product_id: selectedProductId }}
          onFinish={(v) => {
            if (selectedProductId) publishMut.mutate({ product_id: selectedProductId, account_id: v.account_id, price: v.price });
          }}>
          <Form.Item name="product_id" label="商品ID">
            <Input disabled value={selectedProductId ?? ''} />
          </Form.Item>
          <Form.Item name="account_id" label="Ozon 店铺" rules={[{ required: true }]}>
            <Select placeholder="选择目标店铺">
              {ozonAccounts.map((a: Record<string, unknown>) => (
                <Select.Option key={a.id as number} value={a.id as number}>{a.store_name as string}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="price" label="Ozon 售价 (RUB)" rules={[{ required: true }]}>
            <InputNumber min={0} style={{ width: '100%' }} placeholder="输入卢布价格" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
