'use client';

import { useState } from 'react';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  message,
  Row,
  Space,
  Spin,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import {
  ReloadOutlined,
  CheckOutlined,
  CloseOutlined,
  ThunderboltOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';

const { Text, Paragraph } = Typography;

// ---------- Types ----------
interface TraceEvent {
  seq: number;
  event_type: string;
  content: string;
  payload?: Record<string, unknown>;
  timestamp?: string;
}

interface EvidenceItem {
  source_type: string;
  source_id: string;
  title: string;
  summary: string;
  payload?: Record<string, unknown>;
}

interface UnifiedAction {
  id: string;
  title: string;
  agent_id: string;
  risk_level: string;
  confidence: number;
  status: string;
  trace_id?: string;
  payload?: Record<string, unknown>;
  proposed_at?: string;
}

interface TraceDetail {
  trace: TraceMeta;
  events: TraceEvent[];
  evidence: EvidenceItem[];
  actions: UnifiedAction[];
}

interface TraceMeta {
  trace_id: string;
  agent_id: string;
  decision_point: string;
  status: string;
  model_name?: string;
  confidence?: number;
  risk_level?: string;
  latency_ms?: number;
  started_at?: string;
  completed_at?: string;
}

// ---------- Color helpers ----------
const eventColor = (type: string): string => {
  if (type === 'prompt_start') return 'blue';
  if (type === 'tool_call') return 'cyan';
  if (type === 'reasoning') return 'purple';
  if (type === 'done') return 'green';
  if (type === 'error') return 'red';
  return 'blue';
};

const riskColor = (level: string): string => {
  if (level === 'high' || level === 'critical') return 'red';
  if (level === 'medium') return 'orange';
  if (level === 'low') return 'green';
  return 'blue';
};

const statusColor = (status: string): string => {
  if (status === 'suggested') return 'blue';
  if (status === 'approved' || status === 'executed') return 'green';
  if (status === 'executing') return 'cyan';
  if (status === 'rejected' || status === 'failed') return 'red';
  if (status === 'reviewed') return 'default';
  return 'blue';
};

// ---------- Page ----------
export default function AgentTracePage() {
  const params = useParams<{ id: string; traceId: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const agentId = params?.id ?? '';
  const traceId = params?.traceId ?? '';
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const [activeEvidence, setActiveEvidence] = useState<EvidenceItem | null>(null);

  const { data: detail, isLoading } = useQuery({
    queryKey: ['ai-trace', traceId],
    queryFn: async () => {
      const res = await apiClient.get<TraceDetail>(`/v1/ai/traces/${traceId}`);
      return res.data;
    },
    enabled: !!traceId,
  });

  const approveMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/approve`, {
        operator: getCurrentOperator(),
      }),
    onSuccess: () => {
      message.success('已批准');
      qc.invalidateQueries({ queryKey: ['ai-trace', traceId] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/reject`, {
        operator: getCurrentOperator(),
        reason: 'manual reject',
      }),
    onSuccess: () => {
      message.success('已拒绝');
      qc.invalidateQueries({ queryKey: ['ai-trace', traceId] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const executeMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/execute`, {
        operator: getCurrentOperator(),
      }),
    onSuccess: () => {
      message.success('已执行');
      qc.invalidateQueries({ queryKey: ['ai-trace', traceId] });
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  const trace = detail?.trace;
  const events = detail?.events ?? [];
  const evidence = detail?.evidence ?? [];
  const actions = detail?.actions ?? [];

  const sortedEvents = [...events].sort((a, b) => a.seq - b.seq);

  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
        }}
      >
        <Space>
          <h1 style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: 'var(--text-h1)', margin: 0 }}>
            Trace 回放
          </h1>
          <Text type="secondary">{traceId}</Text>
        </Space>
        <Space>
          <Button
            onClick={() => router.push(`/agents/${agentId}`)}
          >
            返回 Agent
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => qc.invalidateQueries({ queryKey: ['ai-trace', traceId] })}
          >
            刷新
          </Button>
        </Space>
      </div>

      <Spin spinning={isLoading}>
        {!detail && !isLoading ? (
          <Empty description="未找到 Trace" />
        ) : (
          <>
            {/* 上部：trace 元信息卡片 */}
            <Card style={{ marginBottom: 16 }}>
              <Descriptions column={{ xs: 1, sm: 2, md: 3 }} size="small">
                <Descriptions.Item label="Agent">
                  {trace?.agent_id}
                </Descriptions.Item>
                <Descriptions.Item label="决策点">
                  {trace?.decision_point}
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  {trace?.status ? (
                    <Tag color={statusColor(trace.status)}>{trace.status}</Tag>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="模型">
                  {trace?.model_name ?? '-'}
                </Descriptions.Item>
                <Descriptions.Item label="置信度">
                  {trace?.confidence !== undefined ? (
                    <Tag
                      color={
                        trace.confidence >= 0.8
                          ? 'green'
                          : trace.confidence >= 0.5
                            ? 'orange'
                            : 'red'
                      }
                    >
                      {(trace.confidence * 100).toFixed(0)}%
                    </Tag>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="风险等级">
                  {trace?.risk_level ? (
                    <Tag color={riskColor(trace.risk_level)}>
                      {trace.risk_level}
                    </Tag>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="耗时">
                  {trace?.latency_ms !== undefined
                    ? `${trace.latency_ms} ms`
                    : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="开始时间">
                  {trace?.started_at
                    ? dayjs(trace.started_at).format('YYYY-MM-DD HH:mm:ss')
                    : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="完成时间">
                  {trace?.completed_at
                    ? dayjs(trace.completed_at).format('YYYY-MM-DD HH:mm:ss')
                    : '-'}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 中部：推理时间线 */}
            <Card title="推理时间线" style={{ marginBottom: 16 }}>
              {sortedEvents.length === 0 ? (
                <Empty description="无事件" />
              ) : (
                <Timeline
                  items={sortedEvents.map((ev) => ({
                    color: eventColor(ev.event_type),
                    children: (
                      <div>
                        <Space size={6} style={{ marginBottom: 4 }}>
                          <Tag color={eventColor(ev.event_type)}>
                            {ev.event_type}
                          </Tag>
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            seq #{ev.seq}
                          </Text>
                          {ev.timestamp && (
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {dayjs(ev.timestamp).format('HH:mm:ss.SSS')}
                            </Text>
                          )}
                        </Space>
                        <div style={{ marginBottom: 4 }}>
                          <Text>{ev.content}</Text>
                        </div>
                        {ev.payload && Object.keys(ev.payload).length > 0 && (
                          <pre
                            style={{
                              background: '#f6f6f6',
                              padding: 8,
                              borderRadius: 4,
                              fontSize: 12,
                              margin: 0,
                              overflow: 'auto',
                              maxHeight: 200,
                            }}
                          >
                            {JSON.stringify(ev.payload, null, 2)}
                          </pre>
                        )}
                      </div>
                    ),
                  }))}
                />
              )}
            </Card>

            <Row gutter={16}>
              {/* 左下：Evidence */}
              <Col xs={24} lg={12}>
                <Card title={`Evidence (${evidence.length})`}>
                  {evidence.length === 0 ? (
                    <Empty description="无 Evidence" />
                  ) : (
                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      {evidence.map((ev, idx) => (
                        <Card
                          key={`${ev.source_type}-${ev.source_id}-${idx}`}
                          size="small"
                          hoverable
                          onClick={() => {
                            setActiveEvidence(ev);
                            setEvidenceOpen(true);
                          }}
                        >
                          <Space size={6} wrap>
                            <Tag color="blue">{ev.source_type}</Tag>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {ev.source_id}
                            </Text>
                          </Space>
                          <div style={{ marginTop: 4 }}>
                            <Text strong>{ev.title}</Text>
                          </div>
                          <div style={{ marginTop: 4 }}>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {ev.summary}
                            </Text>
                          </div>
                          <div style={{ marginTop: 4 }}>
                            <Button
                              type="link"
                              size="small"
                              icon={<EyeOutlined />}
                            >
                              查看 Payload
                            </Button>
                          </div>
                        </Card>
                      ))}
                    </Space>
                  )}
                </Card>
              </Col>

              {/* 右下：关联 Action */}
              <Col xs={24} lg={12}>
                <Card title={`关联 Action (${actions.length})`}>
                  {actions.length === 0 ? (
                    <Empty description="无关联 Action" />
                  ) : (
                    <Space direction="vertical" style={{ width: '100%' }} size="small">
                      {actions.map((action) => (
                        <Card
                          key={action.id}
                          size="small"
                          style={{
                            borderLeft: `4px solid ${
                              action.risk_level === 'high' ||
                              action.risk_level === 'critical'
                                ? '#f5222d'
                                : action.risk_level === 'medium'
                                  ? '#fa8c16'
                                  : '#52c41a'
                            }`,
                          }}
                        >
                          <div
                            style={{
                              display: 'flex',
                              justifyContent: 'space-between',
                              alignItems: 'flex-start',
                              marginBottom: 8,
                            }}
                          >
                            <Text strong style={{ flex: 1 }}>
                              {action.title}
                            </Text>
                            <Tag color={riskColor(action.risk_level)}>
                              {action.risk_level}
                            </Tag>
                          </div>
                          <Space size={6} style={{ marginBottom: 8 }}>
                            <Tag color={statusColor(action.status)}>
                              {action.status}
                            </Tag>
                            <Tag
                              color={
                                action.confidence >= 0.8
                                  ? 'green'
                                  : action.confidence >= 0.5
                                    ? 'orange'
                                    : 'red'
                              }
                            >
                              {(action.confidence * 100).toFixed(0)}%
                            </Tag>
                          </Space>
                          <Space size="small">
                            <Button
                              size="small"
                              type="primary"
                              icon={<CheckOutlined />}
                              loading={
                                approveMutation.isPending &&
                                approveMutation.variables === action.id
                              }
                              onClick={() => approveMutation.mutate(action.id)}
                            >
                              批准
                            </Button>
                            <Button
                              size="small"
                              danger
                              icon={<CloseOutlined />}
                              loading={
                                rejectMutation.isPending &&
                                rejectMutation.variables === action.id
                              }
                              onClick={() => rejectMutation.mutate(action.id)}
                            >
                              拒绝
                            </Button>
                            <Button
                              size="small"
                              icon={<ThunderboltOutlined />}
                              loading={
                                executeMutation.isPending &&
                                executeMutation.variables === action.id
                              }
                              onClick={() => executeMutation.mutate(action.id)}
                            >
                              执行
                            </Button>
                            <Button
                              size="small"
                              type="link"
                              onClick={() => router.push(`/actions/${action.id}`)}
                            >
                              详情
                            </Button>
                          </Space>
                        </Card>
                      ))}
                    </Space>
                  )}
                </Card>
              </Col>
            </Row>
          </>
        )}
      </Spin>

      {/* Evidence 抽屉 */}
      <Drawer
        title="Evidence Payload"
        open={evidenceOpen}
        onClose={() => setEvidenceOpen(false)}
        width={520}
      >
        {activeEvidence && (
          <div>
            <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="Source Type">
                <Tag color="blue">{activeEvidence.source_type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Source ID">
                {activeEvidence.source_id}
              </Descriptions.Item>
              <Descriptions.Item label="Title">
                {activeEvidence.title}
              </Descriptions.Item>
              <Descriptions.Item label="Summary">
                {activeEvidence.summary}
              </Descriptions.Item>
            </Descriptions>
            <Paragraph type="secondary" style={{ marginTop: 8 }}>
              Payload:
            </Paragraph>
            <pre
              style={{
                background: '#f6f6f6',
                padding: 12,
                borderRadius: 6,
                fontSize: 12,
                overflow: 'auto',
                maxHeight: 400,
              }}
            >
              {JSON.stringify(activeEvidence.payload ?? {}, null, 2)}
            </pre>
          </div>
        )}
      </Drawer>
    </div>
  );
}
