'use client';

import { useState } from 'react';
import { Button, Input, Modal, Space, Table, Tag, Form, InputNumber, Select, message, Statistic, Row, Col } from 'antd';
import { PlusOutlined, DollarOutlined, LineChartOutlined, DeleteOutlined } from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';

interface Competitor {
  id: number;
  name: string;
  platform: string;
  product_url: string;
  sku_code: string;
  category: string;
  brand: string;
  status: number;
  created_at: string;
}

interface PriceSnapshot {
  id: number;
  price: number;
  currency: string;
  snapshot_date: string;
  sales_last_30d: number;
  rating: number;
  review_count: number;
  is_in_stock: boolean;
}

interface PriceTrend {
  competitor_id: number;
  current_price: number;
  min_price: number;
  max_price: number;
  avg_price: number;
  price_change_7d: number;
  price_change_30d: number;
  snapshots: PriceSnapshot[];
}

interface CompetitorFormValues {
  name: string;
  platform: string;
  sku_code?: string;
  category?: string;
  brand?: string;
}

interface PriceFormValues {
  competitor_id: number;
  price: number;
  currency?: string;
  sales_last_30d?: number;
  rating?: number;
  review_count?: number;
}

export default function CompetitorsPage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [priceOpen, setPriceOpen] = useState<{ id: number; name: string } | null>(null);
  const [trendOpen, setTrendOpen] = useState<{ id: number; name: string } | null>(null);
  const [form] = Form.useForm();

  const { data: listRes, isLoading } = useQuery({
    queryKey: ['competitors', search],
    queryFn: async () => {
      const res = await apiClient.getPage<Competitor>('/v1/competitors', { search, page: '1', size: '50' });
      return res;
    },
  });

  const competitors: Competitor[] = listRes?.data ?? [];

  const createMut = useMutation({
    mutationFn: (values: CompetitorFormValues) => apiClient.post('/v1/competitors', values),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['competitors'] }); message.success('竞品已添加'); setCreateOpen(false); form.resetFields(); },
  });

  const deleteMut = useMutation({
    mutationFn: (id: number) => apiClient.delete(`/v1/competitors/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['competitors'] }); message.success('已删除'); },
  });

  const priceMut = useMutation({
    mutationFn: (values: PriceFormValues) => apiClient.post(`/v1/competitors/${values.competitor_id}/prices`, values),
    onSuccess: () => { message.success('价格已记录'); setPriceOpen(null); },
  });

  const { data: trendData } = useQuery({
    queryKey: ['competitor-trend', trendOpen?.id],
    queryFn: async () => { const id = trendOpen?.id; if (!id) return null; const res = await apiClient.get<PriceTrend>('/v1/competitors/'+id+'/trend'); return res.data ?? null; },
    enabled: !!trendOpen?.id,
  });

  const trend: PriceTrend | null = trendData ?? null;

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '平台', dataIndex: 'platform', key: 'platform', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'SKU编码', dataIndex: 'sku_code', key: 'sku_code' },
    { title: '品类', dataIndex: 'category', key: 'category' },
    { title: '品牌', dataIndex: 'brand', key: 'brand' },
    { title: '操作', key: 'actions', render: (_: unknown, r: Competitor) => (
        <Space>
          <Button size="small" icon={<DollarOutlined />} onClick={() => setPriceOpen({ id: r.id, name: r.name })}>记录价格</Button>
          <Button size="small" icon={<LineChartOutlined />} onClick={() => setTrendOpen({ id: r.id, name: r.name })}>趋势</Button>
          <Button size="small" danger icon={<DeleteOutlined />} onClick={() => { if (confirm('确认删除?')) deleteMut.mutate(r.id); }} />
        </Space>
    )},
  ];

  return (
    <PageContainer title="竞品监控" subtitle="P3: 竞品价格监控 (#199)">
      <Space style={{ marginBottom: 16 }}>
        <Input.Search placeholder="搜索竞品..." value={search} onChange={e => setSearch(e.target.value)} style={{ width: 300 }} allowClear />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>添加竞品</Button>
      </Space>

      <Table dataSource={competitors} columns={columns} rowKey="id" loading={isLoading} pagination={{ pageSize: 20 }} size="small" />

      <Modal title="添加竞品" open={createOpen} onOk={() => form.submit()} onCancel={() => setCreateOpen(false)} confirmLoading={createMut.isPending}>
        <Form form={form} layout="vertical" onFinish={values => createMut.mutate(values)}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="platform" label="平台" rules={[{ required: true }]}><Select options={[{ value: 'ozon', label: 'Ozon' }, { value: 'shopee', label: 'Shopee' }, { value: 'lazada', label: 'Lazada' }, { value: 'amazon', label: 'Amazon' }]} /></Form.Item>
          <Form.Item name="sku_code" label="SKU"><Input /></Form.Item>
          <Form.Item name="category" label="品类"><Input /></Form.Item>
          <Form.Item name="brand" label="品牌"><Input /></Form.Item>
        </Form>
      </Modal>

      <Modal title={`记录价格 - ${priceOpen?.name}`} open={!!priceOpen} onCancel={() => setPriceOpen(null)} footer={null}>
        <Form layout="vertical" onFinish={values => priceMut.mutate({ ...values, competitor_id: priceOpen?.id })}>
          <Form.Item name="price" label="价格" rules={[{ required: true }]}><InputNumber style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="currency" label="币种" initialValue="CNY"><Input /></Form.Item>
          <Form.Item name="sales_last_30d" label="近30天销量"><InputNumber style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="rating" label="评分"><InputNumber style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="review_count" label="评论数"><InputNumber style={{ width: '100%' }} /></Form.Item>
          <Button type="primary" htmlType="submit" loading={priceMut.isPending} block>记录</Button>
        </Form>
      </Modal>

      <Modal title={`价格趋势 - ${trendOpen?.name}`} open={!!trendOpen} onCancel={() => setTrendOpen(null)} footer={null} width={700}>
        {trend && (
          <>
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={6}><Statistic title="当前价格" value={trend.current_price} prefix="¥" /></Col>
              <Col span={6}><Statistic title="最低价" value={trend.min_price} prefix="¥" /></Col>
              <Col span={6}><Statistic title="最高价" value={trend.max_price} prefix="¥" /></Col>
              <Col span={6}><Statistic title="均价" value={trend.avg_price} prefix="¥" /></Col>
            </Row>
            {trend.price_change_7d !== 0 && <Tag color={trend.price_change_7d > 0 ? 'red' : 'green'}>7日变化: {trend.price_change_7d.toFixed(1)}%</Tag>}
            {trend.price_change_30d !== 0 && <Tag color={trend.price_change_30d > 0 ? 'red' : 'green'} style={{ marginLeft: 8 }}>30日变化: {trend.price_change_30d.toFixed(1)}%</Tag>}
            <Table dataSource={trend.snapshots.slice(0, 20)} rowKey="id" size="small" style={{ marginTop: 16 }}
              columns={[
                { title: '日期', dataIndex: 'snapshot_date', key: 'snapshot_date', render: (v: string) => dayjs(v).format('YYYY-MM-DD') },
                { title: '价格', dataIndex: 'price', key: 'price', render: (v: number) => `¥${v.toFixed(2)}` },
                { title: '销量', dataIndex: 'sales_last_30d', key: 'sales_last_30d' },
                { title: '评分', dataIndex: 'rating', key: 'rating' },
                { title: '库存', dataIndex: 'is_in_stock', key: 'is_in_stock', render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? '有货' : '缺货'}</Tag> },
              ]}
            />
          </>
        )}
      </Modal>
    </PageContainer>
  );
}
