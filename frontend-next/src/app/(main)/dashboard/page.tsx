'use client';

import { Card, Col, Empty, Row, Spin, Statistic, Table, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

interface OverviewData {
  order_total?: number;
  order_by_status?: Record<string, number>;
  order_revenue?: number;
  order_profit?: number;
  sku_total?: number;
  low_stock_count?: number;
  out_of_stock_count?: number;
  listing_active_count?: number;
  aftersales_pending_count?: number;
  exception_open_count?: number;
  month_revenue?: number;
  month_cost?: number;
}

interface ExceptionItem {
  type: string;
  count: number;
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'default',
  paid: 'blue',
  shipped: 'cyan',
  completed: 'green',
  cancelled: 'red',
  refunded: 'orange',
};

function Money({ value }: { value: number | undefined }) {
  return (
    <Statistic
      value={value ?? 0}
      precision={2}
      prefix="¥"
    />
  );
}

export default function DashboardPage() {
  const { data: overview, isLoading: overviewLoading } = useQuery({
    queryKey: ['dashboard', 'overview'],
    queryFn: async () => {
      const res = await apiClient.get<OverviewData>('/v1/dashboard/overview');
      return res.data;
    },
  });

  const { data: exceptions, isLoading: exceptionsLoading } = useQuery({
    queryKey: ['dashboard', 'exceptions'],
    queryFn: async () => {
      const res = await apiClient.get<ExceptionItem[]>('/v1/dashboard/exceptions');
      return res.data ?? [];
    },
  });

  if (overviewLoading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin tip="加载中..." />
      </div>
    );
  }

  const o = overview ?? {};
  const statusEntries = Object.entries(o.order_by_status ?? {});

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontSize: 24, fontWeight: 600, marginBottom: 24 }}>Dashboard</h1>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic title="订单总数" value={o.order_total ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Money value={o.order_revenue} />
            <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>订单收入</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Money value={o.order_profit} />
            <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>订单利润</div>
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic title="SKU 总数" value={o.sku_total ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic
              title="低库存数"
              value={o.low_stock_count ?? 0}
              valueStyle={{ color: (o.low_stock_count ?? 0) > 0 ? '#fa8c16' : undefined }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic
              title="未解决异常数"
              value={o.exception_open_count ?? 0}
              valueStyle={{ color: (o.exception_open_count ?? 0) > 0 ? '#cf1322' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="订单状态分布">
            {statusEntries.length === 0 ? (
              <Empty description="暂无数据" />
            ) : (
              <Row gutter={[12, 12]}>
                {statusEntries.map(([status, count]) => (
                  <Col key={status} xs={12} md={8}>
                    <Statistic
                      title={
                        <Tag color={STATUS_COLORS[status] ?? 'default'}>{status}</Tag>
                      }
                      value={count}
                    />
                  </Col>
                ))}
              </Row>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="当月收入 vs 成本">
            <Row gutter={16}>
              <Col span={12}>
                <Money value={o.month_revenue} />
                <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>当月收入</div>
              </Col>
              <Col span={12}>
                <Statistic
                  title="当月成本"
                  value={o.month_cost ?? 0}
                  precision={2}
                  prefix="¥"
                  valueStyle={{ color: '#cf1322' }}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>

      <Card title="异常分布" style={{ marginTop: 16 }}>
        <Table
          rowKey="type"
          loading={exceptionsLoading}
          dataSource={exceptions}
          pagination={false}
          size="small"
          columns={[
            { title: '异常类型', dataIndex: 'type' },
            { title: '数量', dataIndex: 'count', width: 120 },
          ]}
        />
      </Card>
    </div>
  );
}
