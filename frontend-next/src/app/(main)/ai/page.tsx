'use client';

import { useMemo, useState } from 'react';
import {
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Input,
  message,
  Row,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import {
  PlayCircleOutlined,
  ReloadOutlined,
  SendOutlined,
  CheckOutlined,
  CloseOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';
import type { PageResult, Result } from '@/types/api';

const { Text, Paragraph } = Typography;

// ---------- Types ----------
interface AiAgent {
  agent_id: string;
  name: string;
  squad: string;
  decision_point: string;
  autonomy_level: string;
  trace_count: number;
  action_count: number;
  pending_count: number;
  avg_confidence: number;
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

interface TraceListItem {
  trace_id: string;
  agent_id: string;
  decision_point: string;
  status: string;
  model_name?: string;
  confidence?: number;
  risk_level?: string;
  started_at?: string;
  completed_at?: string;
  latency_ms?: number;
}

interface ChatResponse {
  trace_id: string;
  agent_id: string;
  answer: string;
  confidence: number;
  risk_level: string;
  actions?: UnifiedAction[];
}

interface RunResponse {
  trace_id: string;
  agent_id: string;
  output: string;
  confidence: number;
  risk_level: string;
  action?: UnifiedAction;
}

// ---------- Color helpers ----------
const squadColor = (squad: string): string => {
  if (squad === 'autonomous') return 'blue';
  if (squad === 'governance') return 'purple';
  if (squad === 'ops') return 'gold';
  return 'default';
};

const autonomyColor = (level: string): string => {
  if (level === 'advisory') return 'green';
  if (level === 'guided') return 'blue';
  if (level === 'autonomous') return 'cyan';
  if (level === 'supervised') return 'orange';
  return 'default';
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

const confidenceColor = (c: number): string => {
  if (c >= 0.8) return 'green';
  if (c >= 0.5) return 'orange';
  return 'red';
};

// ---------- Page ----------
export default function AICommandPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [command, setCommand] = useState('');
  const [chatResult, setChatResult] = useState<ChatResponse | null>(null);
  const [runResult, setRunResult] = useState<RunResponse | null>(null);

  // Agents
  const { data: agentsData, isLoading: agentsLoading } = useQuery({
    queryKey: ['ai-agents'],
    queryFn: async () => {
      const res = await apiClient.get<AiAgent[]>('/v1/ai/agents');
      return res.data ?? [];
    },
  });

  // Pending actions (suggested)
  const { data: actionsData, isLoading: actionsLoading } = useQuery({
    queryKey: ['ai-actions-suggested'],
    queryFn: async () => {
      const res = await apiClient.getPage<UnifiedAction>('/v1/ai/actions', {
        status: 'suggested',
        size: '20',
      });
      return res.data ?? [];
    },
  });

  // Recent traces
  const { data: tracesData, isLoading: tracesLoading } = useQuery({
    queryKey: ['ai-traces-recent'],
    queryFn: async () => {
      const res = await apiClient.getPage<TraceListItem>('/v1/ai/traces', {
        size: '10',
      });
      return res.data ?? [];
    },
  });

  // Group agents by squad
  const agentsBySquad = useMemo(() => {
    const map = new Map<string, AiAgent[]>();
    (agentsData ?? []).forEach((a) => {
      const list = map.get(a.squad) ?? [];
      list.push(a);
      map.set(a.squad, list);
    });
    return Array.from(map.entries());
  }, [agentsData]);

  // Chat mutation
  const chatMutation = useMutation({
    mutationFn: async (msg: string) => {
      const res = await apiClient.post<ChatResponse>('/v1/ai/chat', {
        message: msg,
        stream: false,
      });
      return res.data as ChatResponse;
    },
    onSuccess: (data) => {
      setChatResult(data);
      setRunResult(null);
      message.success('AI 已回复');
      qc.invalidateQueries({ queryKey: ['ai-traces-recent'] });
    },
    onError: (e: Error) => message.error(`命令失败: ${e.message}`),
  });

  // Run agent mutation
  const runMutation = useMutation({
    mutationFn: async (agentId: string) => {
      const res = await apiClient.post<RunResponse>('/v1/ai/run', {
        agent_id: agentId,
        decision_point: 'default',
        context: {},
        stream: false,
      });
      return res.data as RunResponse;
    },
    onSuccess: (data) => {
      setRunResult(data);
      setChatResult(null);
      message.success(`Agent ${data.agent_id} 已执行`);
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
      qc.invalidateQueries({ queryKey: ['ai-traces-recent'] });
    },
    onError: (e: Error) => message.error(`Agent 执行失败: ${e.message}`),
  });

  // Action operations
  const approveMutation = useMutation({
    mutationFn: async (id: string) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/approve`, {
        operator: getCurrentOperator(),
      }),
    onSuccess: () => {
      message.success('已批准');
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
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
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
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
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  const handleSend = () => {
    if (!command.trim()) {
      message.warning('请输入命令');
      return;
    }
    chatMutation.mutate(command.trim());
  };

  const traceColumns = [
    {
      title: 'Trace ID',
      dataIndex: 'trace_id',
      width: 180,
      render: (v: string) => <Text code>{v}</Text>,
    },
    { title: 'Agent', dataIndex: 'agent_id', width: 140 },
    { title: '决策点', dataIndex: 'decision_point', width: 140 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    {
      title: '风险',
      dataIndex: 'risk_level',
      width: 90,
      render: (v: string) => (v ? <Tag color={riskColor(v)}>{v}</Tag> : '-'),
    },
    {
      title: '置信度',
      dataIndex: 'confidence',
      width: 90,
      render: (v?: number) =>
        v !== undefined && v !== null ? (
          <Tag color={confidenceColor(v)}>{(v * 100).toFixed(0)}%</Tag>
        ) : (
          '-'
        ),
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      width: 160,
      render: (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <h1 style={{ fontSize: 24, fontWeight: 600, marginBottom: 16 }}>
        AI 指挥中心
      </h1>

      {/* 顶部命令栏 */}
      <Card style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="输入自然语言命令，例如：检查库存异常并建议补货方案"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onSearch={handleSend}
          enterButton={
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={chatMutation.isPending}
            >
              发送
            </Button>
          }
          size="large"
          allowClear
        />
      </Card>

      {/* AI 回复展示 */}
      {(chatResult || runResult) && (
        <Card style={{ marginBottom: 16 }}>
          {chatResult && (
            <>
              <Space style={{ marginBottom: 8 }}>
                <Tag color="blue">AI 回复</Tag>
                <Tag color={confidenceColor(chatResult.confidence)}>
                  置信度 {(chatResult.confidence * 100).toFixed(0)}%
                </Tag>
                <Tag color={riskColor(chatResult.risk_level)}>
                  风险 {chatResult.risk_level}
                </Tag>
                <Text type="secondary">trace: {chatResult.trace_id}</Text>
              </Space>
              <Paragraph style={{ margin: 0 }}>{chatResult.answer}</Paragraph>
            </>
          )}
          {runResult && (
            <>
              <Space style={{ marginBottom: 8 }}>
                <Tag color="cyan">Agent 执行</Tag>
                <Tag color={confidenceColor(runResult.confidence)}>
                  置信度 {(runResult.confidence * 100).toFixed(0)}%
                </Tag>
                <Tag color={riskColor(runResult.risk_level)}>
                  风险 {runResult.risk_level}
                </Tag>
                <Text type="secondary">trace: {runResult.trace_id}</Text>
              </Space>
              <Paragraph style={{ margin: 0 }}>{runResult.output}</Paragraph>
            </>
          )}
        </Card>
      )}

      <Row gutter={16}>
        {/* 左侧：Agent 名册 */}
        <Col xs={24} lg={16}>
          <Card
            title="Agent 名册"
            extra={
              <Button
                icon={<ReloadOutlined />}
                size="small"
                onClick={() => qc.invalidateQueries({ queryKey: ['ai-agents'] })}
              >
                刷新
              </Button>
            }
            style={{ marginBottom: 16 }}
          >
            <Spin spinning={agentsLoading}>
              {agentsBySquad.length === 0 && !agentsLoading ? (
                <Empty description="暂无 Agent" />
              ) : (
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  {agentsBySquad.map(([squad, agents]) => (
                    <div key={squad}>
                      <div style={{ marginBottom: 8 }}>
                        <Tag color={squadColor(squad)}>{squad}</Tag>
                        <Text type="secondary" style={{ marginLeft: 8 }}>
                          {agents.length} 个 agent
                        </Text>
                      </div>
                      <Row gutter={[12, 12]}>
                        {agents.map((agent) => (
                          <Col xs={24} sm={12} key={agent.agent_id}>
                            <Card
                              size="small"
                              hoverable
                              onClick={() => runMutation.mutate(agent.agent_id)}
                              style={{ height: '100%' }}
                            >
                              <div
                                style={{
                                  display: 'flex',
                                  justifyContent: 'space-between',
                                  alignItems: 'flex-start',
                                }}
                              >
                                <div style={{ flex: 1, minWidth: 0 }}>
                                  <Space size={4} wrap>
                                    <Text strong>{agent.name}</Text>
                                    <Tag color={autonomyColor(agent.autonomy_level)}>
                                      {agent.autonomy_level}
                                    </Tag>
                                  </Space>
                                  <div style={{ marginTop: 4 }}>
                                    <Text
                                      type="secondary"
                                      style={{ fontSize: 12 }}
                                      ellipsis
                                    >
                                      {agent.decision_point}
                                    </Text>
                                  </div>
                                  <div style={{ marginTop: 6 }}>
                                    <Space size={8}>
                                      <Badge
                                        count={`待办 ${agent.pending_count}`}
                                        style={{
                                          backgroundColor:
                                            agent.pending_count > 0
                                              ? '#fa8c16'
                                              : '#52c41a',
                                        }}
                                      />
                                      <Tag color={confidenceColor(agent.avg_confidence)}>
                                        置信 {(agent.avg_confidence * 100).toFixed(0)}%
                                      </Tag>
                                    </Space>
                                  </div>
                                </div>
                                <Button
                                  type="link"
                                  icon={<PlayCircleOutlined />}
                                  loading={
                                    runMutation.isPending &&
                                    runMutation.variables === agent.agent_id
                                  }
                                  size="small"
                                />
                              </div>
                            </Card>
                          </Col>
                        ))}
                      </Row>
                    </div>
                  ))}
                </Space>
              )}
            </Spin>
          </Card>
        </Col>

        {/* 右侧：实时决策流 */}
        <Col xs={24} lg={8}>
          <Card
            title="实时决策流"
            extra={
              <Button
                icon={<ReloadOutlined />}
                size="small"
                onClick={() =>
                  qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] })
                }
              >
                刷新
              </Button>
            }
          >
            <Spin spinning={actionsLoading}>
              {(actionsData ?? []).length === 0 && !actionsLoading ? (
                <Empty description="暂无待审批动作" />
              ) : (
                <Space direction="vertical" style={{ width: '100%' }} size="small">
                  {(actionsData ?? []).map((action) => (
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
                          marginBottom: 4,
                        }}
                      >
                        <Text strong style={{ flex: 1 }}>
                          {action.title}
                        </Text>
                        <Tag color={riskColor(action.risk_level)}>
                          {action.risk_level}
                        </Tag>
                      </div>
                      <Space size={4} style={{ marginBottom: 8 }}>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {action.agent_id}
                        </Text>
                        <Tag color={confidenceColor(action.confidence)}>
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
                      </Space>
                    </Card>
                  ))}
                </Space>
              )}
            </Spin>
          </Card>
        </Col>
      </Row>

      {/* 底部：最近 trace 列表 */}
      <Card
        title="最近 Trace"
        style={{ marginTop: 16 }}
        extra={
          <Button
            icon={<ReloadOutlined />}
            size="small"
            onClick={() => qc.invalidateQueries({ queryKey: ['ai-traces-recent'] })}
          >
            刷新
          </Button>
        }
      >
        <Table
          rowKey="trace_id"
          loading={tracesLoading}
          dataSource={tracesData ?? []}
          columns={traceColumns}
          size="small"
          pagination={false}
          scroll={{ x: 'max-content' }}
          onRow={(record) => ({
            onClick: () =>
              router.push(`/agents/${record.agent_id}/trace/${record.trace_id}`),
            style: { cursor: 'pointer' },
          })}
        />
      </Card>
    </div>
  );
}
