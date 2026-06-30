'use client';

import { Card, Empty, Spin, Table, Tag } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeftOutlined, ShoppingOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import apiClient from '@/lib/api-client';

interface OzonProduct {
  offer_id: string;
  name: string;
  price: number;
  stock: number;
  category_id: number;
  state: string;
  image_url?: string;
}

const STATE_MAP: Record<string, { label: string; color: string }> = {
  imported: { label: '已导入', color: 'processing' },
  processed: { label: '已处理', color: 'success' },
  processing: { label: '处理中', color: 'processing' },
  created: { label: '已创建', color: 'default' },
  failed: { label: '失败', color: 'error' },
  rejected: { label: '被拒', color: 'error' },
};

export default function OzonProductsPage() {
  const params = useParams();
  const router = useRouter();
  const accountId = params.id as string;

  const { data, isLoading, error } = useQuery({
    queryKey: ['ozon-products', accountId],
    queryFn: async () => {
      const res = await apiClient.get<OzonProduct[]>(`/v1/platform-integrations/${accountId}/ozon-products`);
      return res.data ?? [];
    },
  });

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/platform-integrations')}>返回</Button>
        <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', margin: 0 }}><ShoppingOutlined /> Ozon 商品列表</h1>
      </div>

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin size="large" tip="正在从 Ozon 拉取商品..." /></div>
      ) : error ? (
        <Card><Empty description="无法加载 Ozon 商品数据" /></Card>
      ) : !data || data.length === 0 ? (
        <Card>
          <Empty description="Ozon 店铺暂无商品">
            <span style={{ color: '#999' }}>去 Ozon 上架第一个商品，或者创建本地商品后发布到 Ozon</span>
          </Empty>
        </Card>
      ) : (
        <Table rowKey="offer_id" dataSource={data} pagination={{ pageSize: 20 }} size="small"
          columns={[
            { title: 'Offter ID', dataIndex: 'offer_id', width: 140 },
            { title: '商品名称', dataIndex: 'name', ellipsis: true },
            { title: '价格', dataIndex: 'price', width: 100, render: (v: number) => `₽${v.toFixed(2)}` },
            { title: '库存', dataIndex: 'stock', width: 80 },
            { title: '状态', dataIndex: 'state', width: 100,
              render: (s: string) => {
                const m = STATE_MAP[s];
                return <Tag color={m?.color || 'default'}>{m?.label || s}</Tag>;
              },
            },
          ]}
        />
      )}
    </div>
  );
}
