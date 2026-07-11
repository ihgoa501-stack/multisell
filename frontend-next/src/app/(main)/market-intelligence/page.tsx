'use client';

import { useState } from 'react';
import type { ReactElement } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Input, Spin, Empty, Typography, Space, Button, Tabs } from 'antd';
import { SearchOutlined, ShoppingOutlined, ArrowUpOutlined, ArrowDownOutlined, MinusOutlined, RiseOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

const { Title, Text } = Typography;

/* ─── Types ─── */

interface CategorySummary {
  category: string;
  product_count: number;
  avg_rank: number;
  avg_price_min: number;
  avg_price_max: number;
  avg_rating: number;
  total_reviews: number;
  demand_score: number;
}

interface MarketTrendItem {
  source: string;
  category?: string;
  rank?: number;
  product_title?: string;
  price_range?: string;
  review_count?: number;
  avg_rating?: number;
  keyword?: string;
  search_volume?: number;
  competition_level?: string;
  trend_direction?: string;
}

interface MarketOverview {
  source: string;
  categories: CategorySummary[];
}

/* ─── Helpers ─── */

function demandColor(score: number): string {
  if (score >= 7) return '#52c41a';     // green
  if (score >= 5) return '#faad14';     // yellow
  return '#ff4d4f';                     // red
}

function fmtCurrency(v: number): string {
  return `¥${v.toFixed(0)}`;
}

function scoreBadge(score: number): ReactElement {
  const c = demandColor(score);
  return <Tag color={c} style={{ fontSize: 14, fontWeight: 600 }}>{score.toFixed(1)}</Tag>;
}

function trendIcon(dir: string): ReactElement {
  switch (dir) {
    case 'up': return <ArrowUpOutlined style={{ color: '#52c41a' }} />;
    case 'down': return <ArrowDownOutlined style={{ color: '#ff4d4f' }} />;
    default: return <MinusOutlined style={{ color: '#999' }} />;
  }
}

/* ─── Keyword Trends Section ─── */

function KeywordTab({ keyword }: { keyword: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['keyword-trends', keyword],
    queryFn: async () => {
      const res = await apiClient.get<{ items: MarketTrendItem[] }>('/v1/sourcing/keyword-trends', { keyword });
      return res.data?.items ?? [];
    },
  });

  if (isLoading) return <Spin style={{ display: 'block', margin: '40px auto' }} />;
  if (!data?.length) return <Empty description="无数据" />;

  const columns = [
    { title: '关键词', dataIndex: 'keyword', width: 200 },
    { title: '搜索量', dataIndex: 'search_volume', width: 100, render: (v: number) => v?.toLocaleString() },
    {
      title: '竞争', dataIndex: 'competition_level', width: 100,
      render: (v: string) => <Tag color={v === 'high' ? 'red' : v === 'medium' ? 'gold' : 'green'}>{v}</Tag>,
    },
    {
      title: '趋势', dataIndex: 'trend_direction', width: 80,
      render: (v: string) => trendIcon(v),
    },
  ];

  return (
    <Table
      dataSource={data}
      columns={columns}
      rowKey={(r) => r.keyword ?? ''}
      size="small"
      pagination={{ pageSize: 8 }}
    />
  );
}

/* ─── Products In Category Tab ─── */

function ProductsTab({ category }: { category: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['market-trends', category],
    queryFn: async () => {
      const res = await apiClient.get<{ items: MarketTrendItem[] }>('/v1/sourcing/market-trends', { category });
      return res.data?.items ?? [];
    },
  });

  if (isLoading) return <Spin style={{ display: 'block', margin: '40px auto' }} />;
  if (!data?.length) return <Empty description={`"${category}" 暂无产品数据`} />;

  const columns = [
    { title: '排名', dataIndex: 'rank', width: 60 },
    { title: '产品', dataIndex: 'product_title', ellipsis: true },
    { title: '价格区间', dataIndex: 'price_range', width: 120 },
    { title: '评论数', dataIndex: 'review_count', width: 80, render: (v: number) => v?.toLocaleString() },
    { title: '评分', dataIndex: 'avg_rating', width: 60, render: (v: number) => v?.toFixed(1) },
  ];

  return (
    <Table
      dataSource={data}
      columns={columns}
      rowKey={(r) => r.product_title ?? ''}
      size="small"
      pagination={{ pageSize: 5 }}
    />
  );
}

/* ─── Category Card ─── */

function CategoryCard({ cat, onClick }: { cat: CategorySummary; onClick: () => void }) {
  return (
    <Card
      hoverable
      style={{ borderRadius: 8, borderLeft: `4px solid ${demandColor(cat.demand_score)}` }}
      onClick={onClick}
    >
      <Space direction="vertical" size="small" style={{ width: '100%' }}>
        <Space>
          <ShoppingOutlined style={{ fontSize: 20, color: '#1890ff' }} />
          <Title level={5} style={{ margin: 0 }}>{cat.category}</Title>
        </Space>
        <Space style={{ justifyContent: 'space-between', width: '100%' }}>
          <Statistic title="需求评分" value={cat.demand_score.toFixed(1)} valueStyle={{ color: demandColor(cat.demand_score), fontSize: 22 }} suffix="/ 10" />
          <Statistic title="产品数" value={cat.product_count} />
        </Space>
        <Row gutter={8}>
          <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>均价</Text><br /><Text>{fmtCurrency(cat.avg_price_min)}-{fmtCurrency(cat.avg_price_max)}</Text></Col>
          <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>平均排名</Text><br /><Text>{cat.avg_rank.toFixed(0)}</Text></Col>
          <Col span={8}><Text type="secondary" style={{ fontSize: 12 }}>平均评分</Text><br /><Text>{cat.avg_rating.toFixed(1)}</Text></Col>
        </Row>
      </Space>
    </Card>
  );
}

/* ─── Main Page ─── */

export default function MarketIntelligencePage() {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [searchKeyword, setSearchKeyword] = useState('');

  const { data: overview, isLoading } = useQuery({
    queryKey: ['market-overview'],
    queryFn: async () => {
      const res = await apiClient.get<MarketOverview>('/v1/sourcing/market-overview');
      return res.data;
    },
  });

  if (isLoading) return <PageContainer title="市场情报"><Spin style={{ display: 'block', margin: '80px auto' }} /></PageContainer>;

  const categories = overview?.categories ?? [];

  // If a category is selected, show its detail tabs
  if (selectedCategory) {
    const cat = categories.find((c) => c.category === selectedCategory);
    return (
      <PageContainer
        title=""
        extra={<Button onClick={() => setSelectedCategory(null)}>← 返回品类概览</Button>}
      >
        <Card style={{ marginBottom: 16, borderLeft: `4px solid ${demandColor(cat?.demand_score ?? 0)}` }}>
          <Space direction="vertical" size="small">
            <Title level={4} style={{ margin: 0 }}>{selectedCategory}</Title>
            {cat && (
              <Space wrap>
                <Tag>{cat.product_count} 个产品</Tag>
                <Tag color="blue">均价 {fmtCurrency(cat.avg_price_min)}-{fmtCurrency(cat.avg_price_max)}</Tag>
                <Tag color="purple">评分 {cat.avg_rating.toFixed(1)}</Tag>
                <Tag color={demandColor(cat.demand_score)}>需求评分 {cat.demand_score.toFixed(1)}</Tag>
              </Space>
            )}
          </Space>
        </Card>
        <Tabs
          items={[
            { key: 'products', label: 'BSR 产品榜', children: <ProductsTab category={selectedCategory} /> },
            { key: 'keywords', label: '相关关键词', children: <KeywordTab keyword={selectedCategory} /> },
          ]}
        />
      </PageContainer>
    );
  }

  // Overview: show category cards
  return (
    <PageContainer
      title="市场情报"
      subtitle="基于 Amazon BSR 的需求信号 — 发现高潜力品类"
      extra={
        <Input
          prefix={<SearchOutlined />}
          placeholder="搜索品类..."
          style={{ width: 240 }}
          value={searchKeyword}
          onChange={(e) => setSearchKeyword(e.target.value)}
        />
      }
    >
      {categories.length === 0 ? (
        <Empty description="暂无市场数据" />
      ) : (
        <>
          <Row gutter={[16, 16]}>
            {categories
              .filter((c) => !searchKeyword || c.category.includes(searchKeyword))
              .map((cat) => (
                <Col key={cat.category} xs={24} sm={12} md={8} lg={6}>
                  <CategoryCard cat={cat} onClick={() => setSelectedCategory(cat.category)} />
                </Col>
              ))}
          </Row>
          <Card title="品类需求评分说明" style={{ marginTop: 16 }}>
            <Space direction="vertical" size="small">
              <Text type="secondary">需求评分 = 评论数(30%) + BSR排名优势(30%) + 评分水平(20%) + 价格区间(20%)，满分 10 分</Text>
              <Space>
                <Tag color="#52c41a">≥7 强需求</Tag>
                <Tag color="#faad14">5-7 中需求</Tag>
                <Tag color="#ff4d4f">&lt;5 弱需求</Tag>
              </Space>
            </Space>
          </Card>
        </>
      )}
    </PageContainer>
  );
}
