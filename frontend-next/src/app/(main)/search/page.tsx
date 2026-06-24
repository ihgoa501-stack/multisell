'use client';

import { useState } from 'react';
import { Empty, Input, List, Spin, Tag } from 'antd';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

interface SearchResult {
  type: string;
  id: string | number;
  title: string;
  subtitle?: string;
  url: string;
}

const TYPE_COLORS: Record<string, string> = {
  product: 'blue',
  sku: 'cyan',
  order: 'green',
  aftersales: 'orange',
  exception: 'red',
  settlement: 'purple',
};

export default function SearchPage() {
  const router = useRouter();
  const [q, setQ] = useState('');

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['search', q],
    queryFn: async () => {
      const res = await apiClient.get<SearchResult[]>('/v1/search', {
        q,
        limit: '20',
      });
      return res.data ?? [];
    },
    enabled: false,
  });

  const handleSearch = (value: string) => {
    setQ(value);
    if (value.trim()) {
      refetch();
    }
  };

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontSize: 24, fontWeight: 600, marginBottom: 16 }}>全局搜索</h1>

      <Input.Search
        size="large"
        placeholder="搜索商品 / SKU / 订单 / 售后 / 异常 / 结算..."
        enterButton="搜索"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        onSearch={handleSearch}
        style={{ maxWidth: 720, marginBottom: 24 }}
      />

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 48 }}>
          <Spin tip="搜索中..." />
        </div>
      ) : data && data.length > 0 ? (
        <List
          bordered
          dataSource={data}
          renderItem={(item) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => item.url && router.push(item.url)}
            >
              <List.Item.Meta
                avatar={<Tag color={TYPE_COLORS[item.type] ?? 'default'}>{item.type}</Tag>}
                title={item.title}
                description={item.subtitle}
              />
            </List.Item>
          )}
        />
      ) : (
        q && !isLoading && <Empty description="未找到相关结果" />
      )}
    </div>
  );
}
