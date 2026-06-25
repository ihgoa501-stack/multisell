'use client';

import { useState } from 'react';
import { message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import apiClient from '@/lib/api-client';
import { getCurrentOperator } from '@/lib/user';
import { useAppStore } from '@/stores/app-store';

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

// ---------- CSS variable color helpers ----------
const squadVar = (squad: string): string => {
  if (squad === 'autonomous') return 'var(--i4)';
  if (squad === 'governance') return 'var(--c4)';
  if (squad === 'ops') return 'var(--y4)';
  return 'var(--t3)';
};

const riskVar = (level: string): string => {
  if (level === 'high' || level === 'critical') return 'var(--r4)';
  if (level === 'medium') return 'var(--y4)';
  if (level === 'low') return 'var(--g4)';
  return 'var(--i4)';
};

const statusVar = (status: string): string => {
  if (status === 'suggested') return 'var(--i4)';
  if (status === 'approved' || status === 'executed') return 'var(--g4)';
  if (status === 'executing') return 'var(--c4)';
  if (status === 'rejected' || status === 'failed') return 'var(--r4)';
  if (status === 'reviewed') return 'var(--t3)';
  return 'var(--i4)';
};

const autonomyTextColor = (level: string): string => {
  if (level === 'advisory') return 'var(--g4)';
  if (level === 'guided') return 'var(--i4)';
  if (level === 'autonomous') return 'var(--c4)';
  if (level === 'supervised') return 'var(--y4)';
  return 'var(--t3)';
};

const confidenceLabel = (c: number): string => (c * 100).toFixed(0) + '%';

// ---------- Reusable style blocks ----------
const sectionHeader: React.CSSProperties = {
  fontSize: '0.62rem',
  fontWeight: 600,
  letterSpacing: '0.07em',
  textTransform: 'uppercase',
  color: 'var(--t4)',
  marginBottom: 6,
};

// ---------- Page ----------
export default function AICommandPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [command, setCommand] = useState('');
  const [sentMessage, setSentMessage] = useState('');
  const [chatResult, setChatResult] = useState<ChatResponse | null>(null);
  const [runResult, setRunResult] = useState<RunResponse | null>(null);

  // Keep a reference to the app store
  useAppStore();

  // ---------- Queries ----------
  const { data: agentsData, isLoading: agentsLoading } = useQuery({
    queryKey: ['ai-agents'],
    queryFn: async () => {
      const res = await apiClient.get<AiAgent[]>('/v1/ai/agents');
      return res.data ?? [];
    },
  });

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

  const { data: tracesData, isLoading: tracesLoading } = useQuery({
    queryKey: ['ai-traces-recent'],
    queryFn: async () => {
      const res = await apiClient.getPage<TraceListItem>('/v1/ai/traces', {
        size: '10',
      });
      return res.data ?? [];
    },
  });

  // ---------- Mutations ----------
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

  // ---------- Handlers ----------
  const handleSend = () => {
    if (!command.trim()) {
      message.warning('请输入命令');
      return;
    }
    const msg = command.trim();
    setSentMessage(msg);
    setCommand('');
    chatMutation.mutate(msg);
  };

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        padding: '16px 20px',
        overflow: 'auto',
        background: 'var(--bg)',
        gap: 12,
      }}
    >
      {/* ===== Greeting bar ===== */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          paddingBottom: 8,
          borderBottom: '1px solid var(--bd)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              fontFamily: 'var(--ds)',
              fontWeight: 700,
              fontSize: '0.95rem',
              color: 'var(--t1)',
            }}
          >
            AI 指挥中心
          </span>
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            fontSize: '0.68rem',
            color: 'var(--t3)',
            fontWeight: 500,
          }}
        >
          <span
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: 'var(--c4)',
              display: 'inline-block',
            }}
          />
          <span>就绪</span>
        </div>
      </div>

      {/* ===== Agent greeting message ===== */}
      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
        <div
          style={{
            width: 30,
            height: 30,
            borderRadius: 8,
            flexShrink: 0,
            background: 'linear-gradient(135deg, var(--i5), var(--c5))',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '0.75rem',
            color: 'white',
            marginTop: 2,
          }}
        >
          ✦
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: '0.72rem',
              fontWeight: 600,
              color: 'var(--t2)',
              marginBottom: 2,
            }}
          >
            凌镜 Agent
          </div>
          <div
            style={{
              fontSize: '0.88rem',
              lineHeight: 1.6,
              color: 'var(--t1)',
              background: 'var(--s1)',
              padding: '10px 14px',
              borderRadius: 8,
              border: '1px solid var(--bd)',
            }}
          >
            ☀️ 早上好！需要我做什么？
          </div>
        </div>
      </div>

      {/* ===== Agent 名册 ===== */}
      {agentsLoading ? (
        <div
          style={{
            fontSize: '0.72rem',
            color: 'var(--t3)',
            padding: '4px 0',
          }}
        >
          loading agents...
        </div>
      ) : (agentsData ?? []).length === 0 ? (
        <div
          style={{
            fontSize: '0.72rem',
            color: 'var(--t3)',
            padding: '4px 0',
          }}
        >
          暂无 Agent
        </div>
      ) : (
        <>
          <div style={sectionHeader}>Agent 名册</div>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(170px, 1fr))',
              gap: 6,
            }}
          >
            {(agentsData ?? []).map((agent) => (
              <button
                key={agent.agent_id}
                onClick={() => runMutation.mutate(agent.agent_id)}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 4,
                  padding: '8px 10px',
                  borderRadius: 6,
                  background: 'var(--s1)',
                  border: '1px solid var(--bd)',
                  color: 'var(--t1)',
                  cursor: 'pointer',
                  fontFamily: 'var(--body)',
                  fontSize: '0.75rem',
                  textAlign: 'left',
                  transition: 'background 80ms',
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLButtonElement).style.background =
                    'var(--s2)';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLButtonElement).style.background =
                    'var(--s1)';
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 5,
                    flexWrap: 'wrap',
                  }}
                >
                  <span style={{ fontWeight: 500, fontSize: '0.75rem' }}>
                    {agent.name}
                  </span>
                  <span
                    style={{
                      fontSize: '0.6rem',
                      padding: '1px 5px',
                      borderRadius: 3,
                      background: squadVar(agent.squad),
                      color: 'white',
                      lineHeight: '1.2',
                    }}
                  >
                    {agent.squad}
                  </span>
                </div>
                <div
                  style={{
                    fontSize: '0.65rem',
                    color: 'var(--t3)',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {agent.decision_point}
                </div>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    marginTop: 2,
                  }}
                >
                  <span
                    style={{
                      fontSize: '0.6rem',
                      padding: '1px 4px',
                      borderRadius: 3,
                      border:
                        '1px solid ' + autonomyTextColor(agent.autonomy_level),
                      color: autonomyTextColor(agent.autonomy_level),
                      lineHeight: '1.2',
                    }}
                  >
                    {agent.autonomy_level}
                  </span>
                  {agent.pending_count > 0 && (
                    <span style={{ fontSize: '0.6rem', color: 'var(--y4)' }}>
                      · {agent.pending_count} 待办
                    </span>
                  )}
                </div>
                {runMutation.isPending &&
                  runMutation.variables === agent.agent_id && (
                    <div
                      style={{
                        fontSize: '0.6rem',
                        color: 'var(--i4)',
                        marginTop: 2,
                      }}
                    >
                      executing...
                    </div>
                  )}
              </button>
            ))}
          </div>
        </>
      )}

      {/* ===== Pending actions ===== */}
      {actionsLoading ? (
        <div
          style={{ fontSize: '0.72rem', color: 'var(--t3)', padding: '4px 0' }}
        >
          loading actions...
        </div>
      ) : (actionsData ?? []).length === 0 ? (
        <div
          style={{ fontSize: '0.72rem', color: 'var(--t3)', padding: '4px 0' }}
        >
          暂无待审批动作
        </div>
      ) : (
        <>
          <div style={sectionHeader}>待审批动作</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {(actionsData ?? []).map((action) => (
              <div
                key={action.id}
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 6,
                  padding: '8px 12px',
                  borderRadius: 6,
                  background: 'var(--s1)',
                  border: '1px solid var(--bd)',
                  borderLeft: '3px solid ' + riskVar(action.risk_level),
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                    gap: 8,
                  }}
                >
                  <span
                    style={{
                      flex: 1,
                      fontSize: '0.82rem',
                      fontWeight: 500,
                      color: 'var(--t1)',
                      lineHeight: 1.4,
                    }}
                  >
                    {action.title}
                  </span>
                  <span
                    style={{
                      fontSize: '0.6rem',
                      fontWeight: 600,
                      letterSpacing: '0.03em',
                      padding: '2px 6px',
                      borderRadius: 3,
                      background: riskVar(action.risk_level) + '22',
                      color: riskVar(action.risk_level),
                      flexShrink: 0,
                    }}
                  >
                    {action.risk_level}
                  </span>
                </div>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    fontSize: '0.65rem',
                    color: 'var(--t3)',
                  }}
                >
                  <span
                    style={{
                      fontFamily: 'var(--mono)',
                      fontSize: '0.62rem',
                    }}
                  >
                    {action.agent_id}
                  </span>
                  <span
                    style={{
                      color:
                        action.confidence >= 0.8
                          ? 'var(--g4)'
                          : action.confidence >= 0.5
                            ? 'var(--y4)'
                            : 'var(--r4)',
                    }}
                  >
                    {confidenceLabel(action.confidence)}
                  </span>
                </div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <button
                    onClick={() => approveMutation.mutate(action.id)}
                    disabled={
                      approveMutation.isPending &&
                      approveMutation.variables === action.id
                    }
                    style={{
                      fontSize: '0.65rem',
                      padding: '3px 8px',
                      borderRadius: 4,
                      border: '1px solid var(--g4)',
                      background: 'transparent',
                      color: 'var(--g4)',
                      cursor: 'pointer',
                      fontFamily: 'var(--body)',
                      fontWeight: 500,
                    }}
                  >
                    {approveMutation.isPending &&
                    approveMutation.variables === action.id
                      ? '...'
                      : '批准'}
                  </button>
                  <button
                    onClick={() => rejectMutation.mutate(action.id)}
                    disabled={
                      rejectMutation.isPending &&
                      rejectMutation.variables === action.id
                    }
                    style={{
                      fontSize: '0.65rem',
                      padding: '3px 8px',
                      borderRadius: 4,
                      border: '1px solid var(--r4)',
                      background: 'transparent',
                      color: 'var(--r4)',
                      cursor: 'pointer',
                      fontFamily: 'var(--body)',
                      fontWeight: 500,
                    }}
                  >
                    {rejectMutation.isPending &&
                    rejectMutation.variables === action.id
                      ? '...'
                      : '拒绝'}
                  </button>
                  <button
                    onClick={() => executeMutation.mutate(action.id)}
                    disabled={
                      executeMutation.isPending &&
                      executeMutation.variables === action.id
                    }
                    style={{
                      fontSize: '0.65rem',
                      padding: '3px 8px',
                      borderRadius: 4,
                      border: '1px solid var(--i4)',
                      background: 'transparent',
                      color: 'var(--i4)',
                      cursor: 'pointer',
                      fontFamily: 'var(--body)',
                      fontWeight: 500,
                    }}
                  >
                    {executeMutation.isPending &&
                    executeMutation.variables === action.id
                      ? '...'
                      : '执行'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* ===== Recent traces ===== */}
      {tracesLoading ? (
        <div
          style={{ fontSize: '0.72rem', color: 'var(--t3)', padding: '4px 0' }}
        >
          loading traces...
        </div>
      ) : (tracesData ?? []).length === 0 ? (
        <div
          style={{ fontSize: '0.72rem', color: 'var(--t3)', padding: '4px 0' }}
        >
          暂无最近 Trace
        </div>
      ) : (
        <>
          <div style={sectionHeader}>最近 Trace</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {(tracesData ?? []).map((trace) => (
              <div
                key={trace.trace_id}
                onClick={() =>
                  router.push(
                    `/agents/${trace.agent_id}/trace/${trace.trace_id}`,
                  )
                }
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '5px 8px',
                  borderRadius: 4,
                  cursor: 'pointer',
                  fontSize: '0.68rem',
                  color: 'var(--t2)',
                  background: 'transparent',
                  transition: 'background 80ms',
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLDivElement).style.background =
                    'var(--s1)';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLDivElement).style.background =
                    'transparent';
                }}
              >
                <code
                  style={{
                    fontFamily: 'var(--mono)',
                    fontSize: '0.62rem',
                    color: 'var(--t3)',
                    minWidth: 100,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {trace.trace_id.slice(0, 12)}...
                </code>
                <span
                  style={{
                    minWidth: 50,
                    color: 'var(--t1)',
                    fontWeight: 500,
                  }}
                >
                  {trace.agent_id}
                </span>
                <span
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    minWidth: 60,
                    color: statusVar(trace.status),
                  }}
                >
                  <span
                    style={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      background: statusVar(trace.status),
                      display: 'inline-block',
                    }}
                  />
                  {trace.status}
                </span>
                {trace.confidence !== undefined &&
                  trace.confidence !== null && (
                    <span
                      style={{
                        minWidth: 36,
                        fontSize: '0.62rem',
                        color:
                          trace.confidence >= 0.8
                            ? 'var(--g4)'
                            : trace.confidence >= 0.5
                              ? 'var(--y4)'
                              : 'var(--r4)',
                      }}
                    >
                      {confidenceLabel(trace.confidence)}
                    </span>
                  )}
                {trace.risk_level && (
                  <span
                    style={{
                      fontSize: '0.6rem',
                      padding: '1px 4px',
                      borderRadius: 3,
                      background: riskVar(trace.risk_level) + '22',
                      color: riskVar(trace.risk_level),
                    }}
                  >
                    {trace.risk_level}
                  </span>
                )}
                <span
                  style={{
                    marginLeft: 'auto',
                    fontSize: '0.62rem',
                    color: 'var(--t3)',
                  }}
                >
                  {trace.started_at
                    ? dayjs(trace.started_at).format('HH:mm:ss')
                    : '-'}
                </span>
              </div>
            ))}
          </div>
        </>
      )}

      {/* ===== Spacer ===== */}
      <div style={{ flex: 1 }} />

      {/* ===== Chat / Run result bubble ===== */}
      {(chatResult || runResult) && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {/* User message */}
          {sentMessage && (
            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <div
                style={{
                  maxWidth: '80%',
                  fontSize: '0.82rem',
                  lineHeight: 1.5,
                  color: 'white',
                  background: 'var(--i5)',
                  padding: '8px 12px',
                  borderRadius: '8px 8px 2px 8px',
                }}
              >
                {sentMessage}
              </div>
            </div>
          )}
          {/* AI response */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
            <div
              style={{
                width: 30,
                height: 30,
                borderRadius: 8,
                flexShrink: 0,
                background: 'linear-gradient(135deg, var(--i5), var(--c5))',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '0.75rem',
                color: 'white',
                marginTop: 2,
              }}
            >
              ✦
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: '0.72rem',
                  fontWeight: 600,
                  color: 'var(--t2)',
                  marginBottom: 2,
                }}
              >
                凌镜 Agent
              </div>
              <div
                style={{
                  fontSize: '0.88rem',
                  lineHeight: 1.6,
                  color: 'var(--t1)',
                  background: 'var(--s1)',
                  padding: '10px 14px',
                  borderRadius: 8,
                  border: '1px solid var(--bd)',
                  whiteSpace: 'pre-wrap',
                }}
              >
                {chatResult
                  ? chatResult.answer
                  : runResult
                    ? runResult.output
                    : ''}
              </div>
              {/* Result metadata */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  marginTop: 4,
                  fontSize: '0.62rem',
                  color: 'var(--t3)',
                }}
              >
                {chatResult && (
                  <>
                    <span
                      style={{
                        color:
                          chatResult.confidence >= 0.8
                            ? 'var(--g4)'
                            : chatResult.confidence >= 0.5
                              ? 'var(--y4)'
                              : 'var(--r4)',
                      }}
                    >
                      {confidenceLabel(chatResult.confidence)}
                    </span>
                    <span
                      style={{
                        color: riskVar(chatResult.risk_level),
                      }}
                    >
                      {chatResult.risk_level}
                    </span>
                    <code
                      style={{
                        fontFamily: 'var(--mono)',
                        fontSize: '0.6rem',
                        color: 'var(--t3)',
                      }}
                    >
                      {chatResult.trace_id.slice(0, 12)}...
                    </code>
                  </>
                )}
                {runResult && (
                  <>
                    <span
                      style={{
                        color:
                          runResult.confidence >= 0.8
                            ? 'var(--g4)'
                            : runResult.confidence >= 0.5
                              ? 'var(--y4)'
                              : 'var(--r4)',
                      }}
                    >
                      {confidenceLabel(runResult.confidence)}
                    </span>
                    <span style={{ color: riskVar(runResult.risk_level) }}>
                      {runResult.risk_level}
                    </span>
                    <code
                      style={{
                        fontFamily: 'var(--mono)',
                        fontSize: '0.6rem',
                        color: 'var(--t3)',
                      }}
                    >
                      {runResult.trace_id.slice(0, 12)}...
                    </code>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ===== Input bar ===== */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          alignItems: 'center',
          padding: '6px 0 0',
          marginTop: 4,
        }}
      >
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
          placeholder="输入自然语言命令，例如：检查库存异常并建议补货方案"
          style={{
            flex: 1,
            padding: '8px 14px',
            borderRadius: 10,
            background: 'var(--s2)',
            border: '1px solid var(--bd2)',
            fontFamily: 'var(--body)',
            fontSize: '0.82rem',
            color: 'var(--t1)',
            outline: 'none',
          }}
        />
        <button
          onClick={handleSend}
          disabled={chatMutation.isPending}
          style={{
            width: 32,
            height: 32,
            borderRadius: 6,
            background: chatMutation.isPending ? 'var(--s3)' : 'var(--i5)',
            border: 'none',
            color: 'white',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: chatMutation.isPending ? 'not-allowed' : 'pointer',
            fontSize: '0.85rem',
            flexShrink: 0,
            transition: 'background 80ms',
          }}
        >
          {chatMutation.isPending ? '...' : '↵'}
        </button>
      </div>
    </div>
  );
}
