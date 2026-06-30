'use client';

import { useParams, useRouter } from 'next/navigation';
import { Card, Descriptions, Spin, Tag, Timeline, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';

const { Text, Title } = Typography;

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  pending: { color: 'default', label: '待处理' },
  processing: { color: 'processing', label: '处理中' },
  completed: { color: 'success', label: '已完成' },
  failed: { color: 'error', label: '失败' },
  cancelled: { color: 'warning', label: '已取消' },
};

interface FlowRecord {
  id: string;
  source_type: string;
  source_id: string;
  status: string;
  context: Record<string, unknown> | null;
  carrier_summary: Record<string, unknown> | null;
  financial_summary: Record<string, unknown> | null;
  error_log: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

function formatTimestamp(ts: string): string {
  return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
}

function renderJson(data: Record<string, unknown> | null) {
  if (!data) return <Text type="secondary">暂无数据</Text>;
  return (
    <pre
      style={{
        background: '#f5f5f5',
        padding: 12,
        borderRadius: 6,
        fontSize: 13,
        maxHeight: 300,
        overflow: 'auto',
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-all',
      }}
    >
      {JSON.stringify(data, null, 2)}
    </pre>
  );
}

function buildTimelineItems(flow: FlowRecord) {
  const items = [];

  // Created event
  items.push({
    color: 'blue',
    children: (
      <div>
        <Text strong>流程创建</Text>
        <br />
        <Text type="secondary">{formatTimestamp(flow.created_at)}</Text>
        <br />
        <Text>
          来源: {flow.source_type} / {flow.source_id}
        </Text>
      </div>
    ),
  });

  // Status change events from context
  if (flow.context?.events && Array.isArray(flow.context.events)) {
    for (const evt of flow.context.events as Record<string, unknown>[]) {
      const time = evt.timestamp as string;
      const type = evt.type as string;
      const data = evt.data as Record<string, unknown> | undefined;
      items.push({
        color: type === 'error' ? 'red' : type === 'warning' ? 'orange' : 'green',
        children: (
          <div>
            <Text strong>{type ?? '事件'}</Text>
            {time && <Text type="secondary"> — {formatTimestamp(time)}</Text>}
            {data && (
              <pre
                style={{
                  marginTop: 4,
                  fontSize: 12,
                  background: '#fafafa',
                  padding: 8,
                  borderRadius: 4,
                  maxHeight: 150,
                  overflow: 'auto',
                }}
              >
                {JSON.stringify(data, null, 2)}
              </pre>
            )}
          </div>
        ),
      });
    }
  }

  // Updated-at as a marker
  if (flow.updated_at && flow.updated_at !== flow.created_at) {
    items.push({
      color: 'gray',
      children: (
        <div>
          <Text>最后更新</Text>
          <br />
          <Text type="secondary">{formatTimestamp(flow.updated_at)}</Text>
        </div>
      ),
    });
  }

  return items;
}

export default function SupplyChainDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params.id as string;

  const { data: flow, isLoading, error } = useQuery({
    queryKey: ['supplychain', 'flow', id],
    queryFn: async () => {
      const res = await apiClient.get<FlowRecord>(`/v1/supplychain/flows/${id}`);
      return res.data;
    },
  });

  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 400 }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (error || !flow) {
    return (
      <div style={{ padding: 24 }}>
        <Text type="danger">加载失败: {(error as Error)?.message ?? '未知错误'}</Text>
      </div>
    );
  }

  const statusCfg = STATUS_MAP[flow.status] ?? { color: 'default', label: flow.status };
  const timelineItems = buildTimelineItems(flow);

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      {/* Back button + title */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          marginBottom: 24,
          cursor: 'pointer',
        }}
        onClick={() => router.push('/supplychain')}
      >
        <ArrowLeftOutlined />
        <Title level={4} style={{ margin: 0 }}>
          供应链流详情
        </Title>
        <Tag color={statusCfg.color}>{statusCfg.label}</Tag>
      </div>

      {/* Basic info card */}
      <Card title="基本信息" style={{ marginBottom: 24 }}>
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="ID">{flow.id}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusCfg.color}>{statusCfg.label}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="来源类型">{flow.source_type}</Descriptions.Item>
          <Descriptions.Item label="来源ID">{flow.source_id}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{formatTimestamp(flow.created_at)}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{formatTimestamp(flow.updated_at)}</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Timeline card */}
      <Card title="时间线" style={{ marginBottom: 24 }}>
        <Timeline items={timelineItems} />
      </Card>

      {/* JSONB data cards */}
      {flow.context && (
        <Card title="上下文 (Context)" style={{ marginBottom: 16 }}>
          {renderJson(flow.context as Record<string, unknown>)}
        </Card>
      )}
      {flow.carrier_summary && (
        <Card title="物流摘要 (Carrier Summary)" style={{ marginBottom: 16 }}>
          {renderJson(flow.carrier_summary as Record<string, unknown>)}
        </Card>
      )}
      {flow.financial_summary && (
        <Card title="财务摘要 (Financial Summary)" style={{ marginBottom: 16 }}>
          {renderJson(flow.financial_summary as Record<string, unknown>)}
        </Card>
      )}
      {flow.error_log && (
        <Card title="错误日志 (Error Log)" style={{ marginBottom: 16 }}>
          {renderJson(flow.error_log as Record<string, unknown>)}
        </Card>
      )}
    </div>
  );
}
