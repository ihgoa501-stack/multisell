'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import ActionRiskConfirmDialog, {
  ActionRiskConfirmMode,
  RiskConfirmAction,
} from '@/components/actions/ActionRiskConfirmDialog';
import apiClient from '@/lib/api-client';
import { getToken } from '@/lib/auth';
import { useAppStore } from '@/stores/app-store';
import { useAIWebSocket, SSEEventData } from '@/lib/realtime';

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
  description?: string;
  agent_id: string;
  action_type?: string;
  risk_level: string;
  confidence: number;
  status: string;
  trace_id?: string;
  requires_approval?: boolean;
  execution_mode?: string;
  before_snapshot?: Record<string, unknown> | null;
  after_snapshot?: Record<string, unknown> | null;
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
  output: string | Record<string, unknown>;
  confidence: number;
  risk_level: string;
  action?: UnifiedAction;
}

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  streaming?: boolean;
  trace_id?: string;
  agent_id?: string;
  confidence?: number;
  risk_level?: string;
  actions?: UnifiedAction[];
}

// ---------- Helpers ----------
function formatOutput(output: string | Record<string, unknown>): string {
  if (typeof output === 'string') return output;
  try {
    return JSON.stringify(output, null, 2);
  } catch {
    return String(output);
  }
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

const ASSISTANT_ICON = (
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
);

// ---------- Page ----------
export default function AICommandPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [command, setCommand] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const conversationRef = useRef<HTMLDivElement>(null);
  const streamingAbortRef = useRef<AbortController | null>(null);
  const [streaming, setStreaming] = useState(false);
  const [confirmAction, setConfirmAction] = useState<UnifiedAction | null>(null);
  const [confirmMode, setConfirmMode] = useState<ActionRiskConfirmMode | null>(null);

  // Keep a reference to the app store
  useAppStore();

  // Auto-scroll conversation on new messages
  useEffect(() => {
    if (conversationRef.current) {
      conversationRef.current.scrollTop = conversationRef.current.scrollHeight;
    }
  }, [messages]);

  // ---------- SSE Streaming ----------
  const streamChat = useCallback(
    async (msg: string): Promise<boolean> => {
      const token = getToken();
      const apiBase =
        process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

      // Add empty assistant message
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: '', streaming: true },
      ]);
      setStreaming(true);

      const abortController = new AbortController();
      streamingAbortRef.current = abortController;

      try {
        const response = await fetch(`${apiBase}/v1/ai/chat`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
          },
          body: JSON.stringify({ message: msg, stream: true }),
          signal: abortController.signal,
        });

        if (!response.ok || !response.body) {
          throw new Error(`SSE request failed: ${response.status}`);
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let fullContent = '';
        let finalTraceId = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });

          // SSE events are separated by double newline
          const parts = buffer.split('\n\n');
          buffer = parts.pop() || '';

          for (const part of parts) {
            const lines = part.split('\n');
            let eventType = '';
            let data = '';

            for (const line of lines) {
              if (line.startsWith('event: ')) {
                eventType = line.slice(7).trim();
              } else if (line.startsWith('data: ')) {
                data = line.slice(6).trim();
              }
            }

            if (eventType === 'token' && data) {
              try {
                const parsed = JSON.parse(data);
                if (parsed.data?.text) {
                  fullContent += parsed.data.text;
                  setMessages((prev) => {
                    const msgs = [...prev];
                    const last = msgs[msgs.length - 1];
                    if (last && last.role === 'assistant') {
                      msgs[msgs.length - 1] = {
                        ...last,
                        content: fullContent,
                        streaming: true,
                      };
                    }
                    return msgs;
                  });
                }
              } catch {
                // ignore parse errors on individual tokens
              }
            } else if (eventType === 'done' && data) {
              try {
                const parsed = JSON.parse(data);
                finalTraceId = parsed.trace_id || '';
              } catch {
                // ignore
              }
            }
          }
        }

        // Finalize the assistant message
        setStreaming(false);
        setMessages((prev) => {
          const msgs = [...prev];
          const last = msgs[msgs.length - 1];
          if (last && last.role === 'assistant') {
            msgs[msgs.length - 1] = {
              ...last,
              streaming: false,
              trace_id: finalTraceId,
            };
          }
          return msgs;
        });

        qc.invalidateQueries({ queryKey: ['ai-traces-recent'] });
        message.success('AI 已回复');
        return true;
      } catch (err) {
        setStreaming(false);

        // If aborted, don't add a fallback message
        if (err instanceof DOMException && err.name === 'AbortError') {
          // Remove the empty assistant message
          setMessages((prev) => prev.slice(0, -1));
          return false;
        }

        // SSE failed — remove the empty assistant so the fallback mutation can add a proper one
        setMessages((prev) => prev.slice(0, -1));
        return false;
      }
    },
    [qc],
  );

  // ---------- WebSocket ----------
  const handleWSEvent = useCallback(
    (event: SSEEventData) => {
      // Invalidate queries when action state changes
      if (
        event.event === 'action_approved' ||
        event.event === 'action_rejected' ||
        event.event === 'action_executed'
      ) {
        qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
        qc.invalidateQueries({ queryKey: ['ai-traces-recent'] });
      }
    },
    [qc],
  );

  useAIWebSocket(handleWSEvent, true);

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
      // Add assistant response to conversation
      setMessages((prev) => {
        // If the last message is already an assistant response (from streaming
        // fallback case), don't duplicate it. Only add if last is user.
        const last = prev[prev.length - 1];
        if (last && last.role === 'assistant') return prev;
        return [
          ...prev,
          {
            role: 'assistant' as const,
            content: data.answer,
            trace_id: data.trace_id,
            agent_id: data.agent_id,
            confidence: data.confidence,
            risk_level: data.risk_level,
            actions: data.actions,
          },
        ];
      });
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
      setMessages((prev) => [
        ...prev,
        {
          role: 'user',
          content: `运行 Agent: ${data.agent_id}`,
        },
        {
          role: 'assistant',
          content: formatOutput(data.output),
          trace_id: data.trace_id,
          agent_id: data.agent_id,
          confidence: data.confidence,
          risk_level: data.risk_level,
          actions: data.action ? [data.action] : undefined,
        },
      ]);
      message.success(`Agent ${data.agent_id} 已执行`);
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
      qc.invalidateQueries({ queryKey: ['ai-traces-recent'] });
    },
    onError: (e: Error) => message.error(`Agent 执行失败: ${e.message}`),
  });

  const approveMutation = useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason?: string }) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/approve`, { reason }),
    onSuccess: () => {
      message.success('已批准');
      setConfirmAction(null);
      setConfirmMode(null);
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
    },
    onError: (e: Error) => message.error(`批准失败: ${e.message}`),
  });

  const rejectMutation = useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason?: string }) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/reject`, {
        reason: reason?.trim() || 'manual reject',
      }),
    onSuccess: () => {
      message.success('已拒绝');
      setConfirmAction(null);
      setConfirmMode(null);
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
    },
    onError: (e: Error) => message.error(`拒绝失败: ${e.message}`),
  });

  const executeMutation = useMutation({
    mutationFn: async ({ id, reason }: { id: string; reason?: string }) =>
      apiClient.post<unknown>(`/v1/ai/actions/${id}/execute`, { reason }),
    onSuccess: () => {
      message.success('已执行');
      setConfirmAction(null);
      setConfirmMode(null);
      qc.invalidateQueries({ queryKey: ['ai-actions-suggested'] });
    },
    onError: (e: Error) => message.error(`执行失败: ${e.message}`),
  });

  // ---------- Handlers ----------
  const handleSend = async () => {
    if (!command.trim()) {
      message.warning('请输入命令');
      return;
    }
    const msg = command.trim();
    setCommand('');

    // Add user message to conversation
    setMessages((prev) => [...prev, { role: 'user', content: msg }]);

    // Try streaming first, fall back to non-streaming mutation
    const success = await streamChat(msg);
    if (!success) {
      chatMutation.mutate(msg);
    }
  };

  const isPending =
    chatMutation.isPending || runMutation.isPending || streaming;

  const openActionConfirm = (action: UnifiedAction, mode: ActionRiskConfirmMode) => {
    setConfirmAction(action);
    setConfirmMode(mode);
  };

  const handleActionConfirm = (action: RiskConfirmAction, reason?: string) => {
    const id = String(action.id);
    if (confirmMode === 'approve') approveMutation.mutate({ id, reason });
    if (confirmMode === 'reject') rejectMutation.mutate({ id, reason });
    if (confirmMode === 'execute') executeMutation.mutate({ id, reason });
  };

  // ---------- Render: message components ----------
  const renderUserMessage = (m: ChatMessage) => (
    <div key={`user-${m.content.slice(0, 20)}`} style={{ display: 'flex', justifyContent: 'flex-end' }}>
      <div
        style={{
          maxWidth: '80%',
          fontSize: '0.82rem',
          lineHeight: 1.5,
          color: 'white',
          background: 'var(--i5)',
          padding: '8px 12px',
          borderRadius: '8px 8px 2px 8px',
          whiteSpace: 'pre-wrap',
        }}
      >
        {m.content}
      </div>
    </div>
  );

  const renderAssistantMessage = (m: ChatMessage, idx: number) => {
    const key = m.trace_id || `assistant-${idx}`;
    return (
      <div key={key} style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
        {ASSISTANT_ICON}
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
            {m.content}
            {m.streaming && (
              <span
                style={{
                  display: 'inline-block',
                  width: 6,
                  height: 14,
                  marginLeft: 2,
                  background: 'var(--t1)',
                  animation: 'blink 0.8s step-end infinite',
                  verticalAlign: 'text-bottom',
                }}
              />
            )}
          </div>
          {/* Metadata row */}
         {(m.confidence !== undefined || m.risk_level || m.trace_id) && (
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
              {m.confidence !== undefined && (
                <span
                  style={{
                    color:
                      m.confidence >= 0.8
                        ? 'var(--g4)'
                        : m.confidence >= 0.5
                          ? 'var(--y4)'
                          : 'var(--r4)',
                  }}
                >
                  {confidenceLabel(m.confidence)}
                </span>
              )}
              {m.risk_level && (
                <span style={{ color: riskVar(m.risk_level) }}>
                  {m.risk_level}
                </span>
              )}
              {m.trace_id && (
                <code
                  style={{
                    fontFamily: 'var(--mono)',
                    fontSize: '0.6rem',
                    color: 'var(--t3)',
                  }}
                >
                  {m.trace_id.slice(0, 12)}...
                </code>
              )}
              {m.streaming && (
                <span style={{ color: 'var(--i4)', fontStyle: 'italic' }}>
                  正在生成...
                </span>
              )}
            </div>
          )}
          {/* Pending actions from this response */}
          {m.actions && m.actions.length > 0 && (
            <div style={{ display: 'flex', gap: 4, marginTop: 6 }}>
              {m.actions.map((a) => (
                <span
                  key={a.id}
                  style={{
                    fontSize: '0.62rem',
                    padding: '2px 6px',
                    borderRadius: 3,
                    background: riskVar(a.risk_level) + '22',
                    color: riskVar(a.risk_level),
                    border: '1px solid ' + riskVar(a.risk_level) + '44',
                  }}
                >
                  {a.title}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  };

  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        padding: '16px 20px',
        overflow: 'hidden',
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
          flexShrink: 0,
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

      {/* ===== Content sections (hidden when conversation is active) ===== */}
      {messages.length === 0 && (
        <>
          {/* ===== Agent greeting message ===== */}
          <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start', flexShrink: 0 }}>
            {ASSISTANT_ICON}
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
                flexShrink: 0,
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
                flexShrink: 0,
              }}
            >
              暂无 Agent
            </div>
          ) : (
            <div style={{ flexShrink: 0 }}>
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
                            '1px solid ' +
                            autonomyTextColor(agent.autonomy_level),
                          color: autonomyTextColor(agent.autonomy_level),
                          lineHeight: '1.2',
                        }}
                      >
                        {agent.autonomy_level}
                      </span>
                      {agent.pending_count > 0 && (
                        <span
                          style={{ fontSize: '0.6rem', color: 'var(--y4)' }}
                        >
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
            </div>
          )}

          {/* ===== Pending actions ===== */}
          {actionsLoading ? (
            <div
              style={{
                fontSize: '0.72rem',
                color: 'var(--t3)',
                padding: '4px 0',
                flexShrink: 0,
              }}
            >
              loading actions...
            </div>
          ) : (actionsData ?? []).length === 0 ? (
            <div
              style={{
                fontSize: '0.72rem',
                color: 'var(--t3)',
                padding: '4px 0',
                flexShrink: 0,
              }}
            >
              暂无待审批动作
            </div>
          ) : (
            <div style={{ flexShrink: 0 }}>
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
                        onClick={() => openActionConfirm(action, 'approve')}
                        disabled={
                          approveMutation.isPending &&
                          approveMutation.variables?.id === action.id
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
                        approveMutation.variables?.id === action.id
                          ? '...'
                          : '批准'}
                      </button>
                      <button
                        onClick={() => openActionConfirm(action, 'reject')}
                        disabled={
                          rejectMutation.isPending &&
                          rejectMutation.variables?.id === action.id
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
                        rejectMutation.variables?.id === action.id
                          ? '...'
                          : '拒绝'}
                      </button>
                      <button
                        onClick={() => openActionConfirm(action, 'execute')}
                        disabled={
                          executeMutation.isPending &&
                          executeMutation.variables?.id === action.id
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
                        executeMutation.variables?.id === action.id
                          ? '...'
                          : '执行'}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* ===== Recent traces ===== */}
          {tracesLoading ? (
            <div
              style={{
                fontSize: '0.72rem',
                color: 'var(--t3)',
                padding: '4px 0',
                flexShrink: 0,
              }}
            >
              loading traces...
            </div>
          ) : (tracesData ?? []).length === 0 ? (
            <div
              style={{
                fontSize: '0.72rem',
                color: 'var(--t3)',
                padding: '4px 0',
                flexShrink: 0,
              }}
            >
              暂无最近 Trace
            </div>
          ) : (
            <div style={{ flexShrink: 0 }}>
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
            </div>
          )}
        </>
      )}

      {/* ===== Conversation area ===== */}
      {messages.length > 0 && (
        <div
          ref={conversationRef}
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
            overflow: 'auto',
            minHeight: 0,
          }}
        >
          {messages.map((m, idx) =>
            m.role === 'user'
              ? renderUserMessage(m)
              : renderAssistantMessage(m, idx),
          )}
        </div>
      )}

      <ActionRiskConfirmDialog
        action={confirmAction}
        mode={confirmMode}
        open={!!confirmAction && !!confirmMode}
        loading={approveMutation.isPending || rejectMutation.isPending || executeMutation.isPending}
        onCancel={() => {
          setConfirmAction(null);
          setConfirmMode(null);
        }}
        onConfirm={handleActionConfirm}
      />

      {/* ===== Input bar ===== */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          alignItems: 'center',
          padding: '6px 0 0',
          marginTop: 4,
          flexShrink: 0,
        }}
      >
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
          placeholder="输入自然语言命令，例如：检查库存异常并建议补货方案"
          disabled={isPending}
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
          disabled={isPending || !command.trim()}
          style={{
            width: 32,
            height: 32,
            borderRadius: 6,
            background: isPending || !command.trim() ? 'var(--s3)' : 'var(--i5)',
            border: 'none',
            color: 'white',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: isPending ? 'not-allowed' : 'pointer',
            fontSize: '0.85rem',
            flexShrink: 0,
            transition: 'background 80ms',
          }}
        >
          {isPending ? '...' : '↵'}
        </button>
      </div>

      {/* ===== Blink animation keyframes ===== */}
      <style jsx>{`
        @keyframes blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0; }
        }
      `}</style>
    </div>
  );
}
