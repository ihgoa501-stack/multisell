'use client';

import { useState, useEffect } from 'react';
import { Card, Table, Input, Button, Space, Tag, Spin, message, Badge } from 'antd';
import { SearchOutlined, LinkOutlined } from '@ant-design/icons';
import apiClient from '@/lib/api-client';

interface Recommendation {
  id: number;
  source_url: string;
  title: string;
  supplier_name: string;
  price: number;
  score: number;
  status: string;
  product_id_1688: string;
  image_url: string;
  recommend_reason: string;
  created_at: string;
}

interface PageData {
  code: number;
  message: string;
  data: Recommendation[];
  total: number;
  page: number;
  size: number;
}

const statusColorMap: Record<string, string> = {
  recommended: 'green',
  pending: 'gold',
  low_quality: 'red',
  imported: 'blue',
  rejected: 'default',
};

const statusLabelMap: Record<string, string> = {
  recommended: '推荐',
  pending: '待处理',
  low_quality: '低质量',
  imported: '已导入',
  rejected: '已拒绝',
};

const scoreColor = (score: number): string => {
  if (score >= 7) return 'green';
  if (score >= 4) return 'gold';
  return 'red';
};

export default function SourcingPage() {
  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [data, setData] = useState<Recommendation[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const loadRecommendations = async () => {
    setLoading(true);
    try {
      const res = await apiClient.get<PageData>('/v1/sourcing/recommendations', {
        page: String(page),
        size: String(pageSize),
      });
      if (res.data) {
        setData(res.data.data || []);
        setTotal(res.data.total || 0);
      }
    } catch (err) {
      message.error('加载推荐列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRecommendations();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize]);

  const handleFetch = async () => {
    if (!url.trim()) {
      message.warning('请输入 1688 商品链接');
      return;
    }
    setFetching(true);
    try {
      const res = await apiClient.post<{
        product_id: number;
        title: string;
        price: number;
        score: number;
        status: string;
      }>('/v1/sourcing/fetch', { url: url.trim() });
      if (res.code === 0) {
        message.success(`采集成功！评分：${res.data?.score}/10`);
        setUrl('');
        // Reload list to show the new recommendation
        await loadRecommendations();
      } else {
        message.error(res.message || '采集失败');
      }
    } catch (err) {
      message.error('采集请求失败，请检查链接是否正确');
    } finally {
      setFetching(false);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: '商品标题',
      dataIndex: 'title',
      ellipsis: true,
      render: (title: string, record: Recommendation) => (
        <a href={record.source_url} target="_blank" rel="noopener noreferrer">
          {title || record.source_url.slice(0, 50) + '...'}
        </a>
      ),
    },
    {
      title: '供应商',
      dataIndex: 'supplier_name',
      width: 140,
      ellipsis: true,
    },
    {
      title: '价格 (CNY)',
      dataIndex: 'price',
      width: 120,
      render: (price: number) => `¥${price.toFixed(2)}`,
    },
    {
      title: '评分',
      dataIndex: 'score',
      width: 80,
      render: (score: number) => (
        <Badge
          count={score}
          showZero
          color={scoreColor(score)}
          style={{ fontSize: 12, fontWeight: 600 }}
          overflowCount={10}
        />
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColorMap[status] || 'default'}>
          {statusLabelMap[status] || status}
        </Tag>
      ),
    },
    {
      title: '采集时间',
      dataIndex: 'created_at',
      width: 160,
    },
    {
      title: '操作',
      width: 80,
      render: (_: unknown, record: Recommendation) => (
        <Button
          type="link"
          size="small"
          icon={<LinkOutlined />}
          href={record.source_url}
          target="_blank"
        >
          查看
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 600,
          fontSize: '1rem',
          color: 'var(--t1)',
          margin: '0 0 16px 0',
        }}
      >
        1688 选品采集
      </h1>

      {/* URL input card */}
      <Card
        size="small"
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: '16px 20px' } }}
      >
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder="粘贴 1688 商品链接，例如 https://detail.1688.com/offer/xxx.html"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            onPressEnter={handleFetch}
            prefix={<LinkOutlined />}
            size="large"
            disabled={fetching}
          />
          <Button
            type="primary"
            icon={fetching ? <Spin size="small" /> : <SearchOutlined />}
            onClick={handleFetch}
            loading={fetching}
            size="large"
          >
            采集分析
          </Button>
        </Space.Compact>
      </Card>

      {/* Recommendations table */}
      <Card
        size="small"
        title={`推荐列表 (${total})`}
        styles={{ body: { padding: 0 } }}
      >
        <Table<Recommendation>
          rowKey="id"
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 900 }}
        />
      </Card>
    </div>
  );
}
