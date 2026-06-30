'use client';

import { useQuery } from '@tanstack/react-query';
import {
  Card,
  Statistic,
  Table,
  Tag,
  Typography,
  Space,
  Button,
  Skeleton,
  Empty,
} from 'antd';
import {
  AppstoreOutlined,
  CheckCircleOutlined,
  EditOutlined,
  WarningOutlined,
  ExclamationCircleOutlined,
  ArrowRightOutlined,
} from '@ant-design/icons';
import Link from 'next/link';
import apiClient from '@/lib/api-client';
import type { Result } from '@/types/api';
import StatCard from '@/components/ui/StatCard';

const { Title } = Typography;

interface ProductDashboardSummary {
  total_products: number;
  active_products: number;
  draft_products: number;
  low_stock_products: number;
  expiring_certificates: number;
}

interface DecisionTrace {
  id: number;
  product_id: number;
  action: string;
  reason: string;
  created_at: string;
}

export default function ProductDashboardPage() {
  const {
    data: summaryData,
    isLoading: summaryLoading,
  } = useQuery({
    queryKey: ['products', '360', 'summary'],
    queryFn: async () => {
      const res = await apiClient.get<ProductDashboardSummary>('/v1/products/360/summary');
      return res.data;
    },
  });

  const {
    data: decisionData,
    isLoading: decisionLoading,
  } = useQuery({
    queryKey: ['products', 'decision'],
    queryFn: async () => {
      const res = await apiClient.get<DecisionTrace[]>('/v1/products/decision', {
        page: '1',
        size: '10',
      });
      return res.data ?? [];
    },
  });

  const summary = summaryData ?? ({} as ProductDashboardSummary);

  const decisionColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '商品',
      dataIndex: 'product_id',
      key: 'product_id',
      width: 100,
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 160,
      render: (action: string) => <Tag>{action}</Tag>,
    },
    {
      title: '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
    },
  ];

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Title level={3}>商品仪表盘</Title>

      {/* Summary Cards */}
      <div
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 16,
        }}
      >
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <StatCard
            title="总商品数"
            value={summary.total_products}
            loading={summaryLoading}
            icon={<AppstoreOutlined />}
            iconBgColor="#e6f7ff"
          />
        </div>
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <StatCard
            title="在售商品"
            value={summary.active_products}
            loading={summaryLoading}
            icon={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
            iconBgColor="#f6ffed"
          />
        </div>
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <StatCard
            title="草稿商品"
            value={summary.draft_products}
            loading={summaryLoading}
            icon={<EditOutlined style={{ color: '#faad14' }} />}
            iconBgColor="#fffbe6"
          />
        </div>
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <StatCard
            title="低库存"
            value={summary.low_stock_products}
            loading={summaryLoading}
            icon={<WarningOutlined style={{ color: summary.low_stock_products > 0 ? '#f5222d' : undefined }} />}
            iconBgColor={summary.low_stock_products > 0 ? '#fff2f0' : '#f6ffed'}
          />
        </div>
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <StatCard
            title="即将过期证书"
            value={summary.expiring_certificates}
            loading={summaryLoading}
            icon={<ExclamationCircleOutlined style={{ color: summary.expiring_certificates > 0 ? '#fa8c16' : undefined }} />}
            iconBgColor={summary.expiring_certificates > 0 ? '#fff7e6' : '#f6ffed'}
          />
        </div>
      </div>

      {/* Recent Decision Activity */}
      <Card title="最近决策活动">
        {decisionLoading ? (
          <Skeleton active paragraph={{ rows: 4 }} />
        ) : !decisionData || decisionData.length === 0 ? (
          <Empty description="暂无决策记录" />
        ) : (
          <Table
            dataSource={decisionData}
            columns={decisionColumns}
            rowKey="id"
            pagination={false}
            size="small"
          />
        )}
      </Card>

      {/* Quick Actions */}
      <Card title="快捷操作">
        <Space wrap>
          <Link href="/products">
            <Button type="primary" icon={<ArrowRightOutlined />}>
              查看商品列表
            </Button>
          </Link>
          <Link href="/settings">
            <Button icon={<ExclamationCircleOutlined />}>管理合规证书</Button>
          </Link>
          <Link href="/import-batches">
            <Button icon={<AppstoreOutlined />}>批量导入商品</Button>
          </Link>
        </Space>
      </Card>
    </Space>
  );
}
