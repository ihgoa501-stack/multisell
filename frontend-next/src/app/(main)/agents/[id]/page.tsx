'use client';

import { ArrowLeftOutlined } from '@ant-design/icons';
import { Button, Card, Descriptions, Empty, Spin, Table, Tag } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';
import { fmtDate } from '@/components/crud/CrudListPage';

interface AgentDetail {
  id?: string;
  name?: string;
  squad?: string;
  status?: string;
  model_hint?: string;
  description?: string;
  last_run?: string;
}

interface Trace {
  id: string | number;
  agent_id?: string;
  action?: string;
  status?: string;
  created_at?: string;
}

const SQUAD_COLORS: Record<string, string> = {
  autonomous: 'blue',
  governance: 'purple',
  ops: 'gold',
};

export default function AgentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const { data: agent, isLoading } = useQuery({
    queryKey: ['agent', id],
    queryFn: async () => {
      const res = await apiClient.get<AgentDetail>(`/v1/agents/${id}`);
      return res.data;
    },
    retry: false,
  });

  const { data: traces } = useQuery({
    queryKey: ['agent', id, 'traces'],
    queryFn: async () => {
      const res = await apiClient.get<Trace[]>('/v1/ai/traces', { agent_id: id, limit: '10' });
      return res.data ?? [];
    },
    retry: false,
  });

  return (
    <PageContainer title="Agent 详情">
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => router.push('/ai')}
        style={{ marginBottom: 16 }}
      >
        返回
      </Button>

      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : !agent ? (
        <Card>
          <Empty description="Agent 不存在或暂无数据" />
        </Card>
      ) : (
        <>
          <Card title="基本信息" style={{ marginBottom: 16 }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="ID">{agent.id ?? id}</Descriptions.Item>
              <Descriptions.Item label="名称">{agent.name ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="Squad">
                {agent.squad ? (
                  <Tag color={SQUAD_COLORS[agent.squad] ?? 'default'}>{agent.squad}</Tag>
                ) : (
                  '-'
                )}
              </Descriptions.Item>
              <Descriptions.Item label="状态">{agent.status ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="模型">{agent.model_hint ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="上次运行">{agent.last_run ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>
                {agent.description ?? '-'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="最近 Traces（前 10 条）">
            <Table
              rowKey="id"
              dataSource={traces ?? []}
              size="small"
              pagination={false}
              columns={[
                { title: 'ID', dataIndex: 'id', width: 70 },
                { title: 'Action', dataIndex: 'action' },
                { title: '状态', dataIndex: 'status', width: 120 },
                { title: '时间', dataIndex: 'created_at', width: 160, render: fmtDate },
              ]}
            />
          </Card>
        </>
      )}
    </PageContainer>
  );
}
