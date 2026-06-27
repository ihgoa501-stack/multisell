'use client';

import {
  Badge, Card, Col, Descriptions, Empty, Row, Spin, Statistic, Table, Tag,
} from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, LinkOutlined, RobotOutlined } from '@ant-design/icons';
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
  platform_connections?: PlatformConnection[];
  agent_statuses?: AgentStatus[];
}

interface PlatformConnection {
  platform_id: number;
  platform_code: string;
  platform_name: string;
  store_name: string;
  status: string;
  sync_status: string;
  last_sync_at?: string;
  last_error?: string;
}

interface AgentStatus {
  agent_id: string;
  name: string;
  status: string;
  last_activity?: string;
}

interface ExceptionItem {
  type: string;
  count: number;
}

interface RejectionReasonStat {
  agent_id: string;
  rejection_reason: string;
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

const PLATFORM_COLORS: Record<string, string> = {
  ozon: 'blue',
  shopee: 'orange',
};

const AGENT_COLORS: Record<string, string> = {
  A4: 'green', A5: 'red', A6: 'purple',
  A8: 'geekblue', A10: 'cyan', A9: 'gold',
  G0: 'volcano', G1: 'lime', G3: 'orange',
};

function Money({ value }: { value: number | undefined }) {
  return (
    <Statistic value={value ?? 0} precision={2} prefix="¥" />
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

  const { data: rejectionReasons, isLoading: rejectionLoading } = useQuery({
    queryKey: ['dashboard', 'rejection-reasons'],
    queryFn: async () => {
      const res = await apiClient.get<RejectionReasonStat[]>('/v1/dashboard/rejection-reasons');
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
  const platforms = o.platform_connections ?? [];
  const agents = o.agent_statuses ?? [];

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', marginBottom: 24 }}>Dashboard</h1>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card><Statistic title="订单总数" value={o.order_total ?? 0} /></Card>
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
          <Card><Statistic title="SKU 总数" value={o.sku_total ?? 0} /></Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic title="低库存" value={o.low_stock_count ?? 0}
              valueStyle={{ color: (o.low_stock_count ?? 0) > 0 ? '#fa8c16' : undefined }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={4}>
          <Card>
            <Statistic title="异常" value={o.exception_open_count ?? 0}
              valueStyle={{ color: (o.exception_open_count ?? 0) > 0 ? '#cf1322' : undefined }} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title={<><LinkOutlined /> 平台对接</>}>
            {platforms.length === 0 ? (
              <Empty description="暂无已对接平台">
                <a href="/platform-integrations">连接 Ozon 店铺</a>
              </Empty>
            ) : platforms.map((p) => (
              <Card key={p.platform_id} size="small" style={{ marginBottom: 8, borderLeft: p.status === 'active' ? '3px solid #52c41a' : '3px solid #ff4d4f' }}>
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="平台">
                    <Tag color={PLATFORM_COLORS[p.platform_code] || 'default'}>{p.platform_name}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="状态">
                    {p.status === 'active' ? <Badge status="success" text="已连接" /> : <Badge status="error" text="断开" />}
                  </Descriptions.Item>
                  <Descriptions.Item label="店铺">{p.store_name || '-'}</Descriptions.Item>
                  <Descriptions.Item label="同步"><Tag>{p.sync_status}</Tag></Descriptions.Item>
                  {p.last_sync_at && (
                    <Descriptions.Item label="上次同步" span={2}>{p.last_sync_at?.slice(0, 16).replace('T', ' ')}</Descriptions.Item>
                  )}
                </Descriptions>
              </Card>
            ))}
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title={<><RobotOutlined /> AI Agent 状态</>}>
            {agents.length === 0 ? (
              <Empty description="Agent 暂无活动"><span style={{ color: '#999' }}>系统启动中...</span></Empty>
            ) : (
              <Row gutter={[12, 12]}>
                {agents.map((a) => (
                  <Col key={a.agent_id} xs={12} md={8}>
                    <Card size="small">
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
                        <Tag color={AGENT_COLORS[a.agent_id] || 'default'}>{a.agent_id}</Tag>
                        {a.status === 'active' ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : <CloseCircleOutlined style={{ color: '#999' }} />}
                      </div>
                      <div style={{ fontWeight: 500, fontSize: 13 }}>{a.name}</div>
                      <div style={{ color: '#999', fontSize: 11, marginTop: 4 }}>
                        {a.last_activity ? a.last_activity.slice(0, 16).replace('T', ' ') : '等待运行'}
                      </div>
                    </Card>
                  </Col>
                ))}
              </Row>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="订单状态分布">
            {statusEntries.length === 0 ? <Empty description="暂无数据" /> : (
              <Row gutter={[12, 12]}>
                {statusEntries.map(([status, count]) => (
                  <Col key={status} xs={12} md={8}>
                    <Statistic title={<Tag color={STATUS_COLORS[status] ?? 'default'}>{status}</Tag>} value={count} />
                  </Col>
                ))}
              </Row>
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="当月收入 vs 成本">
            <Row gutter={16}>
              <Col span={12}><Money value={o.month_revenue} /><div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>收入</div></Col>
              <Col span={12}><Statistic title="成本" value={o.month_cost ?? 0} precision={2} prefix="¥" valueStyle={{ color: '#cf1322' }} /></Col>
            </Row>
          </Card>
        </Col>
      </Row>

      <Card title="异常分布" style={{ marginTop: 16 }}>
        <Table rowKey="type" loading={exceptionsLoading} dataSource={exceptions} pagination={false} size="small"
          columns={[
            { title: '异常类型', dataIndex: 'type' },
            { title: '数量', dataIndex: 'count', width: 120 },
          ]} />
      </Card>

      <Card title="拒绝原因分析" style={{ marginTop: 16 }}>
        <Table rowKey={(r) => `${r.agent_id}-${r.rejection_reason}`}
          loading={rejectionLoading}
          dataSource={rejectionReasons}
          pagination={false}
          size="small"
          columns={[
            { title: 'Agent ID', dataIndex: 'agent_id', width: 100 },
            { title: '拒绝原因', dataIndex: 'rejection_reason' },
            { title: '次数', dataIndex: 'count', width: 80 },
          ]} />
      </Card>
    </div>
  );
}
