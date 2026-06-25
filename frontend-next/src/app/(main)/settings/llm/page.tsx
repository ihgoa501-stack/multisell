'use client';

import { Card, Empty, Spin, Table, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import apiClient from '@/lib/api-client';

interface LlmConfig {
  agent_id?: string;
  model_hint?: string;
  provider?: string;
  temperature?: number;
  status?: string;
}

export default function SettingsLlmPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['settings', 'llm'],
    queryFn: async () => {
      const res = await apiClient.get<LlmConfig[] | { agents?: LlmConfig[] }>('/v1/settings/llm');
      const d = res.data;
      if (Array.isArray(d)) return d;
      return d?.agents ?? [];
    },
    retry: false,
  });

  return (
    <div style={{ padding: '16px 20px', background: 'var(--bg)', minHeight: '100%' }}>
      <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '1rem', color: 'var(--t1)', margin: '0 0 16px 0' }}>LLM 配置</h1>
      <Card>
        {isLoading ? (
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        ) : !data || data.length === 0 ? (
          <Empty description="暂无 LLM 配置" />
        ) : (
          <Table
            rowKey="agent_id"
            dataSource={data}
            pagination={false}
            columns={[
              { title: 'Agent ID', dataIndex: 'agent_id', width: 200 },
              { title: '模型', dataIndex: 'model_hint', width: 220 },
              { title: 'Provider', dataIndex: 'provider', width: 140 },
              { title: 'Temperature', dataIndex: 'temperature', width: 120 },
              {
                title: '状态',
                dataIndex: 'status',
                width: 110,
                render: (v: string) =>
                  v ? <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag> : '-',
              },
            ]}
          />
        )}
      </Card>
    </div>
  );
}
