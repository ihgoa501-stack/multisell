'use client';

import { useMemo, useState } from 'react';
import { Card, Col, Row, Statistic, Table, Tag, Select, DatePicker, Input, Empty } from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

interface ProfitSummary {
  id: number;
  product_id: number;
  purchase_cost: number;
  shipping_cost: number;
  platform_fee: number;
  tariff_cost: number;
  other_cost: number;
  total_cost: number;
  target_revenue: number;
  estimated_profit: number;
  profit_margin: number;
  status: string;
  currency: string;
  calculated_by: string;
  created_at: string;
  updated_at: string;
}

const statusLabels: Record<string, { color: string; label: string }> = {
  profitable: { color: 'green', label: '盈利' },
  marginal: { color: 'orange', label: '微利' },
  unprofitable: { color: 'red', label: '亏损' },
};

export default function ProfitPage() {
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(20);
  const [status, setStatus] = useState<string>('');
  const [dateRange, setDateRange] = useState<[unknown, unknown] | null>(null);
  const [search, setSearch] = useState('');

  const queryParams = useMemo(() => {
    const params: Record<string, string> = { page: String(page), size: String(size) };
    if (status) params.status = status;
    if (dateRange?.[0]) {
      params.start_date = dayjs(dateRange[0] as dayjs.Dayjs).format('YYYY-MM-DD');
    }
    if (dateRange?.[1]) {
      params.end_date = dayjs(dateRange[1] as dayjs.Dayjs).format('YYYY-MM-DD');
    }
    if (search) params.search = search;
    return params;
  }, [page, size, status, dateRange, search]);

  const { data, isLoading } = useQuery({
    queryKey: ['profit-summaries', queryParams],
    queryFn: () => apiClient.getPage<ProfitSummary>('/v1/profit/summaries', queryParams),
  });

  const items = data?.data || [];
  const total = data?.total || 0;

  const avgMargin =
    items.length > 0
      ? items.reduce((s, i) => s + i.profit_margin, 0) / items.length
      : 0;
  const unprofitableCount = items.filter((i) => i.status === 'unprofitable').length;

  const columns = [
    { title: 'SKU', dataIndex: 'product_id', width: 100 },
    {
      title: '采购成本',
      dataIndex: 'purchase_cost',
      width: 100,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
      sorter: (a: ProfitSummary, b: ProfitSummary) => a.purchase_cost - b.purchase_cost,
    },
    {
      title: '物流成本',
      dataIndex: 'shipping_cost',
      width: 100,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '平台费用',
      dataIndex: 'platform_fee',
      width: 100,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '关税',
      dataIndex: 'tariff_cost',
      width: 80,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '其他成本',
      dataIndex: 'other_cost',
      width: 80,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '总成本',
      dataIndex: 'total_cost',
      width: 100,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '售价',
      dataIndex: 'target_revenue',
      width: 100,
      render: (v: number) => `¥${(v ?? 0).toFixed(2)}`,
    },
    {
      title: '预估利润',
      dataIndex: 'estimated_profit',
      width: 110,
      render: (v: number) => {
        const val = v ?? 0;
        return (
          <span style={{ color: val >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 600 }}>
            ¥{val.toFixed(2)}
          </span>
        );
      },
      sorter: (a: ProfitSummary, b: ProfitSummary) => a.estimated_profit - b.estimated_profit,
    },
    {
      title: '利润率',
      dataIndex: 'profit_margin',
      width: 90,
      render: (v: number) => {
        const val = v ?? 0;
        let color = '#52c41a';
        if (val < 0) color = '#ff4d4f';
        else if (val < 15) color = '#faad14';
        return (
          <span style={{ color, fontWeight: 600 }}>{val.toFixed(1)}%</span>
        );
      },
      sorter: (a: ProfitSummary, b: ProfitSummary) => a.profit_margin - b.profit_margin,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (v: string) => {
        const s = statusLabels[v] ?? { color: 'default', label: v };
        return <Tag color={s.color}>{s.label}</Tag>;
      },
    },
  ];

  return (
    <div style={{ padding: '16px 20px', minHeight: '100%' }}>
      <h1
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: 'var(--text-h1)',
          color: 'var(--t1)',
          marginBottom: 16,
        }}
      >
        利润真相
      </h1>

      {/* Filter bar */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col>
          <Select
            allowClear
            placeholder="状态"
            style={{ width: 140 }}
            value={status || undefined}
            onChange={(v) => {
              setStatus(v || '');
              setPage(1);
            }}
            options={[
              { label: '全部', value: '' },
              { label: '盈利', value: 'profitable' },
              { label: '微利', value: 'marginal' },
              { label: '亏损', value: 'unprofitable' },
            ]}
          />
        </Col>
        <Col>
          <DatePicker.RangePicker
            onChange={(dates) => {
              setDateRange(dates as [unknown, unknown] | null);
              setPage(1);
            }}
          />
        </Col>
        <Col>
          <Input.Search
            placeholder="搜索商品..."
            allowClear
            onSearch={(v) => {
              setSearch(v);
              setPage(1);
            }}
            style={{ width: 200 }}
          />
        </Col>
      </Row>

      {/* Summary stats */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic title="总商品数" value={total} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="平均利润率" value={avgMargin} precision={1} suffix="%" />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="亏损商品数"
              value={unprofitableCount}
              valueStyle={{ color: unprofitableCount > 0 ? '#ff4d4f' : undefined }}
            />
          </Card>
        </Col>
      </Row>

      {/* Table */}
      <Table
        rowKey="id"
        loading={isLoading}
        dataSource={items}
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        columns={columns as any}
        scroll={{ x: 'max-content' }}
        locale={{ emptyText: <Empty description="暂无利润数据" /> }}
        pagination={{
          current: data?.page ?? page,
          pageSize: data?.size ?? size,
          total: data?.total ?? 0,
          showSizeChanger: true,
          pageSizeOptions: [10, 20, 50],
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, s) => {
            setPage(p);
            setSize(s);
          },
        }}
      />
    </div>
  );
}
