'use client';

import { Card, Descriptions, Empty, Spin, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

interface EvolutionData {
  enabled?: boolean;
  rules?: Array<{ id: string; name: string; description?: string }>;
  episodes?: number;
  last_run?: string;
}

export default function AgentsEvolutionPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['agents', 'evolution'],
    queryFn: async () => {
      const res = await apiClient.get<EvolutionData>('/v1/agents/evolution');
      return res.data;
    },
    retry: false,
  });

  return (
    <PageContainer title="Agent 进化">
      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : !data ? (
        <Card>
          <Empty description="暂无进化数据" />
        </Card>
      ) : (
        <>
          <Card title="概览" style={{ marginBottom: 16 }}>
            <Descriptions column={3} bordered size="small">
              <Descriptions.Item label="是否启用">
                <Tag color={data.enabled ? 'green' : 'default'}>
                  {data.enabled ? '启用' : '停用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Episodes">{data.episodes ?? 0}</Descriptions.Item>
              <Descriptions.Item label="上次运行">{data.last_run ?? '-'}</Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="进化规则">
            {!data.rules || data.rules.length === 0 ? (
              <Empty description="暂无规则" />
            ) : (
              <Descriptions column={1} bordered size="small">
                {data.rules.map((r) => (
                  <Descriptions.Item key={r.id} label={r.name}>
                    {r.description ?? '-'}
                  </Descriptions.Item>
                ))}
              </Descriptions>
            )}
          </Card>
        </>
      )}
    </PageContainer>
  );
}
