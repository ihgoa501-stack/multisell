'use client';

import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { Alert, Button, Card, Descriptions, Empty, Skeleton, Space, Tag, Timeline, Typography } from 'antd';
import { useParams, useRouter } from 'next/navigation';
import PageContainer from '@/components/ui/PageContainer';
import apiClient from '@/lib/api-client';

const { Paragraph, Text } = Typography;

interface TraceEvent {
  seq: number;
  event_type: string;
  content: string;
  payload?: Record<string, unknown>;
  created_at?: string;
}

interface TraceEvidence {
  id: number;
  source_type: string;
  source_id: string;
  title: string;
  summary?: string;
  payload?: Record<string, unknown>;
}

interface TraceDetail {
  trace: {
    trace_id: string;
    agent_id: string;
    decision_point: string;
    status: string;
    model_provider?: string;
    model_name?: string;
    prompt_version?: string;
    token_count?: number;
    final_output?: Record<string, unknown>;
    started_at?: string;
    completed_at?: string;
  };
  events: TraceEvent[];
  evidence: TraceEvidence[];
}

const eventLabels: Record<string, string> = {
  model_turn_started: '模型回合开始',
  model_turn_completed: '模型回合完成',
  tool_requested: '模型请求工具',
  tool_denied: '工具请求被拒绝',
  capability_call: '能力执行成功',
  capability_failed: '能力执行失败',
  tool_result: '工具结果返回模型',
  model_response: '模型生成最终回答',
  run_stopped: '运行已安全停止',
  provider_error: '模型服务失败',
};

function payloadText(payload?: Record<string, unknown>) {
  return payload && Object.keys(payload).length > 0 ? JSON.stringify(payload, null, 2) : '';
}

export default function XiaoQTracePage() {
  const { traceId = '' } = useParams<{ traceId: string }>();
  const router = useRouter();
  const query = useQuery({
    queryKey: ['xiao-q-trace', traceId],
    queryFn: async () => (await apiClient.get<TraceDetail>(`/v1/xiao-q/traces/${traceId}`)).data,
    enabled: Boolean(traceId),
    retry: false,
  });

  const detail = query.data;
  const events = [...(detail?.events ?? [])].sort((a, b) => a.seq - b.seq);

  return (
    <PageContainer title="小Q 执行记录">
      <Space wrap style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/xiaoq')}>返回小Q</Button>
        <Button icon={<ReloadOutlined />} onClick={() => query.refetch()} loading={query.isFetching}>刷新</Button>
        <Text code copyable>{traceId}</Text>
      </Space>

      {query.isLoading && <Card aria-busy="true"><Skeleton active paragraph={{ rows: 6 }} /></Card>}
      {query.isError && <Alert type="error" showIcon message="无法读取执行记录" description="记录不存在、无权访问，或服务暂时不可用。" />}
      {!query.isLoading && !query.isError && !detail && <Empty description="没有找到执行记录" />}

      {detail && (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Card title="运行边界">
            <Descriptions size="small" bordered column={{ xs: 1, md: 2 }}>
              <Descriptions.Item label="Agent"><Text code>{detail.trace.agent_id}</Text></Descriptions.Item>
              <Descriptions.Item label="状态"><Tag>{detail.trace.status}</Tag></Descriptions.Item>
              <Descriptions.Item label="决策点">{detail.trace.decision_point}</Descriptions.Item>
              <Descriptions.Item label="Prompt版本">{detail.trace.prompt_version ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="Provider">{detail.trace.model_provider ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="模型">{detail.trace.model_name ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="Token">{detail.trace.token_count ?? 0}</Descriptions.Item>
            </Descriptions>
          </Card>

          {detail.trace.final_output && (
            <Card title="最终结果">
              {'answer' in detail.trace.final_output ? (
                <Paragraph style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{String(detail.trace.final_output.answer)}</Paragraph>
              ) : (
                <Paragraph code copyable style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{JSON.stringify(detail.trace.final_output, null, 2)}</Paragraph>
              )}
            </Card>
          )}

          <Card title="实际执行顺序">
            {events.length === 0 ? <Empty description="本次运行没有事件" /> : (
              <Timeline items={events.map((event) => ({
                color: event.event_type.includes('failed') || event.event_type === 'provider_error' ? 'red' : event.event_type === 'tool_denied' || event.event_type === 'run_stopped' ? 'orange' : 'blue',
                children: (
                  <div>
                    <Space wrap><Tag>{event.seq}</Tag><Text strong>{eventLabels[event.event_type] ?? event.event_type}</Text><Text code>{event.content}</Text></Space>
                    {payloadText(event.payload) && <Paragraph code copyable style={{ whiteSpace: 'pre-wrap', marginTop: 8 }}>{payloadText(event.payload)}</Paragraph>}
                  </div>
                ),
              }))} />
            )}
          </Card>

          <Card title="证据引用">
            {(detail.evidence ?? []).length === 0 ? <Empty description="本次回答没有读取领域证据" /> : (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {detail.evidence.map((item) => (
                  <Card key={item.id} size="small">
                    <Space wrap><Text strong>{item.title}</Text><Tag>{item.source_type}</Tag><Text code>{item.source_id}</Text></Space>
                    {item.summary && <Paragraph type="secondary">{item.summary}</Paragraph>}
                  </Card>
                ))}
              </Space>
            )}
          </Card>
        </Space>
      )}
    </PageContainer>
  );
}
