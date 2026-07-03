'use client';

import { useQuery } from '@tanstack/react-query';
import {
  Card, Col, Row, Table, Tabs, Tag, Typography, Alert, Space, Descriptions,
} from 'antd';
import {
  DollarOutlined, WarningOutlined, CarryOutOutlined, BarChartOutlined,
} from '@ant-design/icons';
import PageContainer from '@/components/ui/PageContainer';
import StatusTag from '@/components/ui/StatusTag';
import apiClient from '@/lib/api-client';
import type { ColumnsType } from 'antd/es/table';

const { Text, Title } = Typography;

// ---- Types matching backend API responses ----

interface ShippingSnapshot {
  id: number;
  order_id: number;
  provider_name: string;
  channel_name: string;
  destination_country: string;
  total_shipping_fee: number;
  base_shipping_fee: number;
  surcharge_fee: number;
  fuel_surcharge_fee: number;
  currency: string;
  actual_weight_kg: number;
  volumetric_weight_kg: number;
  chargeable_weight_kg: number;
  rule_version: number;
  source_trigger: string;
  quoted_by: string;
  created_at: string;
}

interface BillAnomaly {
  id: number;
  batch_id: number;
  row_number: number;
  tracking_number: string;
  order_no: string;
  provider_name: string;
  channel_name: string;
  destination_country: string;
  actual_shipping_fee: number | null;
  snapshot_shipping_fee: number | null;
  variance_amount: number | null;
  variance_pct: number | null;
  anomaly_type: string;
  review_status: string;
  note: string;
  created_at: string;
}

interface FulfillmentTrackingRecord {
  id: number;
  order_id: number;
  tracking_number: string;
  carrier_code: string;
  carrier_name: string;
  status: string;
  tracking_events: TrackingEvent[];
  estimated_delivery: string | null;
  delivered_at: string | null;
  is_lost: boolean;
  is_returned: boolean;
  is_damaged: boolean;
  note: string;
  created_at: string;
}

interface TrackingEvent {
  timestamp: string;
  status: string;
  location: string;
  message: string;
}

interface CarrierPerformance {
  provider_name: string;
  total_orders: number;
  on_time_count: number;
  on_time_rate: number;
  lost_count: number;
  lost_rate: number;
  returned_count: number;
  returned_rate: number;
  damaged_count: number;
  damaged_rate: number;
  avg_variance_pct: number;
  score: number;
}

interface BillBatch {
  id: number;
  source_filename: string;
  currency: string;
  row_count: number;
  matched_count: number;
  mismatch_count: number;
  unmatched_count: number;
  status: string;
  created_by: string;
  created_at: string;
}

// ---- Snapshot Tab ----

const snapshotColumns: ColumnsType<ShippingSnapshot> = [
  { title: '订单ID', dataIndex: 'order_id', width: 80 },
  { title: '物流商', dataIndex: 'provider_name', width: 120 },
  { title: '渠道', dataIndex: 'channel_name', width: 120 },
  { title: '目的国', dataIndex: 'destination_country', width: 80 },
  {
    title: '计费重 (kg)', dataIndex: 'chargeable_weight_kg', width: 100,
    render: (v: number) => v?.toFixed(2),
  },
  {
    title: '基础运费', dataIndex: 'base_shipping_fee', width: 100,
    render: (v: number, r: ShippingSnapshot) => <Text strong>{v?.toFixed(2)} {r.currency}</Text>,
  },
  {
    title: '附加费', dataIndex: 'surcharge_fee', width: 80,
    render: (v: number) => v?.toFixed(2),
  },
  {
    title: '燃油费', dataIndex: 'fuel_surcharge_fee', width: 80,
    render: (v: number) => v?.toFixed(2),
  },
  {
    title: '运费总计', dataIndex: 'total_shipping_fee', width: 120,
    render: (v: number, r: ShippingSnapshot) => (
      <Text strong style={{ color: 'var(--g4)' }}>{v?.toFixed(2)} {r.currency}</Text>
    ),
  },
  { title: '规则版本', dataIndex: 'rule_version', width: 80 },
  {
    title: '来源', dataIndex: 'source_trigger', width: 80,
    render: (v: string) => <StatusTag status={v} />,
  },
  {
    title: '创建时间', dataIndex: 'created_at', width: 160,
    render: (v: string) => v?.slice(0, 19).replace('T', ' '),
  },
];

function SnapshotTab() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['shipping-snapshots', 1],
    queryFn: () => apiClient.getPage<ShippingSnapshot>('/v1/shipping/snapshots', { page: '1', size: '50' }),
  });

  if (error) return <Alert type="error" message="加载运费快照失败" showIcon />;

  const snapshots = data?.data ?? [];

  return (
    <div>
      <Alert
        message="订单运费快照在生成后不可修改。历史订单的运费不受后续费率变更影响。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Table<ShippingSnapshot>
        rowKey="id"
        dataSource={snapshots}
        columns={snapshotColumns}
        loading={isLoading}
        size="small"
        scroll={{ x: 1300 }}
        pagination={{
          pageSize: 20,
          total: data?.total ?? 0,
          showTotal: (t) => `共 ${t} 条`,
        }}
      />
    </div>
  );
}

// ---- Bill Anomalies Tab ----

const batchColumns: ColumnsType<BillBatch> = [
  { title: '批次ID', dataIndex: 'id', width: 70 },
  { title: '文件名', dataIndex: 'source_filename', width: 200 },
  { title: '币种', dataIndex: 'currency', width: 60 },
  { title: '总行数', dataIndex: 'row_count', width: 70 },
  { title: '已匹配', dataIndex: 'matched_count', width: 70 },
  { title: '差异', dataIndex: 'mismatch_count', width: 70 },
  { title: '未匹配', dataIndex: 'unmatched_count', width: 70 },
  {
    title: '状态', dataIndex: 'status', width: 100,
    render: (v: string) => <StatusTag status={v} />,
  },
  { title: '创建人', dataIndex: 'created_by', width: 100 },
  {
    title: '创建时间', dataIndex: 'created_at', width: 160,
    render: (v: string) => v?.slice(0, 19).replace('T', ' '),
  },
];

const anomalyColumns: ColumnsType<BillAnomaly> = [
  { title: '行号', dataIndex: 'row_number', width: 60 },
  { title: '运单号', dataIndex: 'tracking_number', width: 140 },
  { title: '订单号', dataIndex: 'order_no', width: 100 },
  { title: '物流商', dataIndex: 'provider_name', width: 120 },
  { title: '渠道', dataIndex: 'channel_name', width: 120 },
  { title: '目的国', dataIndex: 'destination_country', width: 70 },
  {
    title: '预估运费', dataIndex: 'snapshot_shipping_fee', width: 90,
    render: (v: number | null) => v != null ? v.toFixed(2) : '-',
  },
  {
    title: '实际运费', dataIndex: 'actual_shipping_fee', width: 90,
    render: (v: number | null) => v != null ? v.toFixed(2) : '-',
  },
  {
    title: '差异', dataIndex: 'variance_amount', width: 90,
    render: (v: number | null) => v != null ? (
      <Text style={{ color: v > 0 ? 'var(--r4)' : 'var(--g4)' }}>
        {v > 0 ? '+' : ''}{v.toFixed(2)}
      </Text>
    ) : '-',
  },
  {
    title: '差异%', dataIndex: 'variance_pct', width: 80,
    render: (v: number | null) => v != null ? `${v > 0 ? '+' : ''}${v.toFixed(1)}%` : '-',
  },
  {
    title: '异常类型', dataIndex: 'anomaly_type', width: 100,
    render: (v: string) => {
      if (!v) return '-';
      const color = v === 'overcharge' ? 'red' : v === 'undercharge' ? 'orange' : 'default';
      return <Tag color={color}>{v === 'overcharge' ? '多收' : v === 'undercharge' ? '少收' : v}</Tag>;
    },
  },
  {
    title: '复核状态', dataIndex: 'review_status', width: 100,
    render: (v: string) => {
      const map: Record<string, string> = { pending: '待复核', resolved: '已解决', confirmed: '已确认' };
      return <StatusTag status={v} label={map[v] ?? v} />;
    },
  },
  { title: '备注', dataIndex: 'note', width: 120 },
];

function BillAnomaliesTab() {
  const { data: batchData, isLoading: batchLoading } = useQuery({
    queryKey: ['bill-batches'],
    queryFn: () => apiClient.getPage<BillBatch>('/v1/shipping/bill-batches', { page: '1', size: '20' }),
  });

  // Load anomalies for the first batch by default
  const firstBatchId = (batchData?.data ?? [])[0]?.id;
  const { data: anomalyData, isLoading: anomalyLoading } = useQuery({
    queryKey: ['bill-anomalies', firstBatchId],
    queryFn: () => apiClient.get<BillAnomaly[]>(`/v1/shipping/bill-batches/${firstBatchId}/anomalies`),
    enabled: !!firstBatchId,
  });

  const batches = batchData?.data ?? [];
  const anomalies = anomalyData?.data ?? [];

  return (
    <div>
      <Alert
        message="账单对账比较物流商账单与订单运费快照，标记超出 ±5% 的差异为异常。当前为内部对账，非物流商 production API。"
        type="warning"
        showIcon
        icon={<WarningOutlined />}
        style={{ marginBottom: 16 }}
      />
      <Title level={5} style={{ marginBottom: 8 }}>账单批次</Title>
      <Table<BillBatch>
        rowKey="id"
        dataSource={batches}
        columns={batchColumns}
        loading={batchLoading}
        size="small"
        scroll={{ x: 1000 }}
        pagination={{ pageSize: 10, total: batchData?.total ?? 0 }}
        style={{ marginBottom: 24 }}
      />
      {firstBatchId ? (
        <>
          <Title level={5} style={{ marginBottom: 8 }}>
            批次异常明细 (批次 #{firstBatchId})
            <Tag style={{ marginLeft: 8 }}>{anomalies.length} 条</Tag>
          </Title>
          <Table<BillAnomaly>
            rowKey="id"
            dataSource={anomalies}
            columns={anomalyColumns}
            loading={anomalyLoading}
            size="small"
            scroll={{ x: 1500 }}
          />
        </>
      ) : (
        <Alert message="暂无账单批次数据，导入账单后自动对账并标记异常。" type="info" showIcon />
      )}
    </div>
  );
}

// ---- Fulfillment Tracking Tab ----

const trackingColumns: ColumnsType<FulfillmentTrackingRecord> = [
  { title: '运单ID', dataIndex: 'id', width: 70 },
  { title: '订单ID', dataIndex: 'order_id', width: 80 },
  { title: '运单号', dataIndex: 'tracking_number', width: 160 },
  { title: '承运商', dataIndex: 'carrier_name', width: 120 },
  {
    title: '物流状态', dataIndex: 'status', width: 100,
    render: (v: string) => <StatusTag status={v} />,
  },
  {
    title: '异常标记', dataIndex: 'is_lost', width: 180,
    render: (_: unknown, r: FulfillmentTrackingRecord) => (
      <Space size={4}>
        {r.is_lost && <Tag color="red">丢件</Tag>}
        {r.is_returned && <Tag color="orange">退件</Tag>}
        {r.is_damaged && <Tag color="volcano">破损</Tag>}
        {!r.is_lost && !r.is_returned && !r.is_damaged && <Text type="secondary">正常</Text>}
      </Space>
    ),
  },
  {
    title: '预计送达', dataIndex: 'estimated_delivery', width: 120,
    render: (v: string) => v ? v.slice(0, 19).replace('T', ' ') : '-',
  },
  {
    title: '实际送达', dataIndex: 'delivered_at', width: 120,
    render: (v: string) => v ? v.slice(0, 19).replace('T', ' ') : '-',
  },
  { title: '备注', dataIndex: 'note', width: 120 },
  {
    title: '创建时间', dataIndex: 'created_at', width: 160,
    render: (v: string) => v?.slice(0, 19).replace('T', ' '),
  },
];

function TrackingTab() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['fulfillment-tracking', 1],
    queryFn: () => apiClient.getPage<FulfillmentTrackingRecord>('/v1/shipping/tracking', { page: '1', size: '50' }),
  });

  if (error) return <Alert type="error" message="加载履约追踪数据失败" showIcon />;

  const records = data?.data ?? [];

  return (
    <div>
      <Alert
        message="运单数据来源：系统内部记录。目前为 mock/sandbox 模式，未接入真实物流商 API。"
        type="info"
        showIcon
        icon={<CarryOutOutlined />}
        style={{ marginBottom: 16 }}
      />
      <Table<FulfillmentTrackingRecord>
        rowKey="id"
        dataSource={records}
        columns={trackingColumns}
        loading={isLoading}
        size="small"
        scroll={{ x: 1300 }}
        pagination={{
          pageSize: 20,
          total: data?.total ?? 0,
          showTotal: (t) => `共 ${t} 条`,
        }}
      />
    </div>
  );
}

// ---- Carrier Performance Tab ----

function CarrierPerformanceTab() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['carrier-performance'],
    queryFn: () => apiClient.get<CarrierPerformance[]>('/v1/shipping/carrier-performance'),
  });

  if (error) return <Alert type="error" message="加载承运商绩效数据失败" showIcon />;

  const stats = data?.data ?? [];

  const perfColumns: ColumnsType<CarrierPerformance> = [
    { title: '承运商', dataIndex: 'provider_name', width: 120 },
    { title: '总订单', dataIndex: 'total_orders', width: 80 },
    {
      title: '准时率', dataIndex: 'on_time_rate', width: 80,
      render: (v: number) => `${(v * 100).toFixed(1)}%`,
    },
    {
      title: '丢件率', dataIndex: 'lost_rate', width: 80,
      render: (v: number) => <Text style={{ color: v > 0.02 ? 'var(--r4)' : undefined }}>{(v * 100).toFixed(1)}%</Text>,
    },
    {
      title: '退件率', dataIndex: 'returned_rate', width: 80,
      render: (v: number) => `${(v * 100).toFixed(1)}%`,
    },
    {
      title: '破损率', dataIndex: 'damaged_rate', width: 80,
      render: (v: number) => `${(v * 100).toFixed(1)}%`,
    },
    {
      title: '账单偏差率', dataIndex: 'avg_variance_pct', width: 100,
      render: (v: number) => (
        <Text style={{ color: v > 5 ? 'var(--r4)' : v > 2 ? 'var(--a4)' : 'var(--g4)' }}>
          {v.toFixed(1)}%
        </Text>
      ),
    },
    {
      title: '综合评分', dataIndex: 'score', width: 80,
      render: (v: number) => (
        <Text strong style={{
          color: v >= 80 ? 'var(--g4)' : v >= 60 ? 'var(--a4)' : 'var(--r4)',
        }}>
          {v.toFixed(0)}
        </Text>
      ),
    },
  ];

  if (stats.length === 0) {
    return (
      <Alert
        message="暂无承运商绩效数据。订单发货后将自动累积绩效评分。"
        type="info"
        showIcon
      />
    );
  }

  return (
    <div>
      <Alert
        message="承运商绩效基于系统内运单数据和账单偏差计算。左移为低风险建议，非生产发货决策。"
        type="warning"
        showIcon
        icon={<BarChartOutlined />}
        style={{ marginBottom: 16 }}
      />

      {/* KPI row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {stats.slice(0, 4).map((s) => (
          <Col key={s.provider_name} xs={24} sm={12} lg={6}>
            <Card size="small">
              <StatCardItem
                name={s.provider_name}
                onTimeRate={s.on_time_rate}
                score={s.score}
                avgVariance={s.avg_variance_pct}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Table<CarrierPerformance>
        rowKey="provider_name"
        dataSource={stats}
        columns={perfColumns}
        loading={isLoading}
        size="small"
        scroll={{ x: 1000 }}
      />
    </div>
  );
}

function StatCardItem({ name, onTimeRate, score, avgVariance }: {
  name: string;
  onTimeRate: number;
  score: number;
  avgVariance: number;
}) {
  return (
    <div>
      <Text strong>{name}</Text>
      <div style={{ marginTop: 8 }}>
        <Descriptions size="small" column={1}>
          <Descriptions.Item label="综合评分">
            <Text strong style={{
              fontSize: 20,
              color: score >= 80 ? 'var(--g4)' : score >= 60 ? 'var(--a4)' : 'var(--r4)',
            }}>
              {score.toFixed(0)}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item label="准时率">{(onTimeRate * 100).toFixed(1)}%</Descriptions.Item>
          <Descriptions.Item label="账单偏差">
            <Text style={{ color: avgVariance > 5 ? 'var(--r4)' : 'var(--g4)' }}>
              {avgVariance.toFixed(1)}%
            </Text>
          </Descriptions.Item>
        </Descriptions>
      </div>
    </div>
  );
}

// ---- Main Cockpit Page ----

const tabItems = [
  {
    key: 'snapshots',
    label: <span><DollarOutlined /> 订单运费快照</span>,
    children: <SnapshotTab />,
  },
  {
    key: 'bill-anomalies',
    label: <span><WarningOutlined /> 账单对账异常</span>,
    children: <BillAnomaliesTab />,
  },
  {
    key: 'tracking',
    label: <span><CarryOutOutlined /> 履约追踪</span>,
    children: <TrackingTab />,
  },
  {
    key: 'carrier-perf',
    label: <span><BarChartOutlined /> 承运商绩效</span>,
    children: <CarrierPerformanceTab />,
  },
];

export default function FulfillmentPage() {
  return (
    <PageContainer
      title="履约中枢"
      subtitle="查看运费快照、账单对账、运单追踪和承运商绩效 — 当前为仅供查看的运营工作台，不自动修改订单和物流"
    >
      <Tabs items={tabItems} size="large" />
    </PageContainer>
  );
}
