'use client';

import { Card, Descriptions, Empty, Spin, Tag } from 'antd';
import { useQuery } from '@tanstack/react-query';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

interface EntropyData {
  rule_count?: number;
  conflict_count?: number;
  entropy_score?: number;
  health?: 'ok' | 'warn' | 'critical' | string;
  warnings?: string[];
}

const HEALTH_COLORS: Record<string, string> = {
  ok: 'green',
  warn: 'orange',
  critical: 'red',
};

export default function AgentsEntropyPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['agents', 'entropy'],
    queryFn: async () => {
      const res = await apiClient.get<EntropyData>('/v1/agents/entropy');
      return res.data;
    },
    retry: false,
  });

  return (
    <PageContainer title="Agent 熵监测">
      {isLoading ? (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin tip="加载中..." />
          </div>
        </Card>
      ) : !data ? (
        <Card>
          <Empty description="暂无熵数据" />
        </Card>
      ) : (
        <>
          <Card title="健康概览" style={{ marginBottom: 16 }}>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label="健康状态">
                <Tag color={HEALTH_COLORS[data.health ?? ''] ?? 'default'}>
                  {data.health ?? '-'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="熵分数">
                {data.entropy_score ?? '-'}
              </Descriptions.Item>
              <Descriptions.Item label="规则数">{data.rule_count ?? 0}</Descriptions.Item>
              <Descriptions.Item label="冲突数">{data.conflict_count ?? 0}</Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title="告警">
            {!data.warnings || data.warnings.length === 0 ? (
              <Empty description="无告警" />
            ) : (
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {data.warnings.map((w, i) => (
                  <li key={i} style={{ marginBottom: 4 }}>
                    <Tag color="orange">warn</Tag>
                    {w}
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </>
      )}
    </PageContainer>
  );
}
