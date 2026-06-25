'use client';

import { useState, useRef, useEffect } from 'react';
import { Drawer, Button, Input, Space, Tag, Typography, Spin, message } from 'antd';
import { RobotOutlined, SendOutlined, UserOutlined } from '@ant-design/icons';
import { useAppStore } from '@/stores/app-store';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  agent_id?: string;
  confidence?: number;
  risk_level?: string;
  trace_id?: string;
}

interface ChatResponse {
  trace_id: string;
  agent_id: string;
  answer: string;
  confidence: number;
  risk_level: string;
}

const confidenceColor = (c?: number): string => {
  if (c === undefined) return 'default';
  if (c >= 0.8) return 'green';
  if (c >= 0.5) return 'orange';
  return 'red';
};

const riskColor = (level?: string): string => {
  if (level === 'high' || level === 'critical') return 'red';
  if (level === 'medium') return 'orange';
  if (level === 'low') return 'green';
  return 'default';
};

function getAuthHeaders(): HeadersInit {
  const headers: HeadersInit = { 'Content-Type': 'application/json' };
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

interface CopilotPanelProps {
  open: boolean;
}

export default function CopilotPanel({ open }: CopilotPanelProps) {
  const { setCopilotOpen } = useAppStore();
  const [inputValue, setInputValue] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      role: 'assistant',
      content: 'Hi! I\'m your AI Copilot. Ask me about inventory, sales, listings, or anything about your business.',
    },
  ]);
  const [loading, setLoading] = useState(false);
  const [streamingText, setStreamingText] = useState('');
  const [streamingMeta, setStreamingMeta] = useState<{
    agent_id?: string;
    confidence?: number;
    risk_level?: string;
    trace_id?: string;
  } | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingText]);

  const handleSend = async () => {
    const msg = inputValue.trim();
    if (!msg || loading) return;
    setInputValue('');

    // Add user message
    setMessages((prev) => [...prev, { role: 'user', content: msg }]);
    setLoading(true);
    setStreamingText('');
    setStreamingMeta(null);

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      // Try SSE streaming first
      const response = await fetch(`${API_BASE}/v1/ai/chat`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), Accept: 'text/event-stream' },
        body: JSON.stringify({ message: msg, stream: true }),
        signal: controller.signal,
      });

      if (!response.ok) throw new Error(`HTTP ${response.status}`);

      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let buffer = '';
      let fullText = '';
      const meta: {
        agent_id?: string;
        confidence?: number;
        risk_level?: string;
        trace_id?: string;
      } = {};

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('event: token')) continue;
          if (line.startsWith('data: ')) {
            try {
              const chunk: Record<string, unknown> = JSON.parse(line.slice(6));
              const text = typeof chunk.text === 'string' ? chunk.text : '';
              if (text) {
                fullText += text;
                setStreamingText(fullText);
              }
              const aid = typeof chunk.agent_id === 'string' ? chunk.agent_id : undefined;
              const conf = typeof chunk.confidence === 'number' ? chunk.confidence : undefined;
              const rl = typeof chunk.risk_level === 'string' ? chunk.risk_level : undefined;
              const tid = typeof chunk.trace_id === 'string' ? chunk.trace_id : undefined;
              if (aid || conf !== undefined || rl) {
                meta.agent_id = aid || meta.agent_id || '';
                meta.confidence = conf ?? meta.confidence;
                meta.risk_level = rl || meta.risk_level || '';
                meta.trace_id = tid || meta.trace_id || '';
                setStreamingMeta({ ...meta });
              }
              if (chunk.done) {
                const answer = typeof chunk.answer === 'string' && chunk.answer ? chunk.answer : fullText;
                if (answer) fullText = answer;
                meta.agent_id = aid || meta.agent_id || '';
                meta.confidence = conf ?? meta.confidence;
                meta.risk_level = rl || meta.risk_level || '';
                meta.trace_id = tid || meta.trace_id || '';
              }
            } catch {
              // ignore parse errors on partial chunks
            }
          }
        }
      }

      // Add assistant message
      if (fullText) {
        setMessages((prev) => [
          ...prev,
          {
            role: 'assistant',
            content: fullText || '',
            ...(meta.agent_id || meta.confidence !== undefined ? {
              agent_id: meta.agent_id,
              confidence: meta.confidence,
              risk_level: meta.risk_level,
              trace_id: meta.trace_id,
            } : {}),
          },
        ]);
      }
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      // Fallback: use non-streaming API
      try {
        const fallbackRes = await fetch(`${API_BASE}/v1/ai/chat`, {
          method: 'POST',
          headers: getAuthHeaders(),
          body: JSON.stringify({ message: msg, stream: false }),
        });
        if (fallbackRes.ok) {
          const json = await fallbackRes.json();
          const data = json.data as ChatResponse | undefined;
          if (data?.answer) {
            setMessages((prev) => [
              ...prev,
              {
                role: 'assistant',
                content: data.answer,
                agent_id: data.agent_id,
                confidence: data.confidence,
                risk_level: data.risk_level,
                trace_id: data.trace_id,
              },
            ]);
          }
        }
      } catch {
        message.error('Failed to get AI response');
      }
    } finally {
      setLoading(false);
      setStreamingText('');
      setStreamingMeta(null);
      abortRef.current = null;
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <Drawer
      title={
        <Space>
          <RobotOutlined />
          <span>AI Copilot</span>
          {loading && <Spin size="small" />}
        </Space>
      }
      placement="right"
      open={open}
      onClose={() => setCopilotOpen(false)}
      width={420}
      styles={{
        body: {
          padding: 0,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        },
      }}
    >
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        }}
      >
        {/* Messages area */}
        <div
          style={{
            flex: 1,
            padding: 16,
            overflowY: 'auto',
            display: 'flex',
            flexDirection: 'column',
            gap: 12,
          }}
        >
          {messages.map((msg, idx) => (
            <div
              key={idx}
              style={{
                display: 'flex',
                flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
                gap: 8,
              }}
            >
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 14,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: msg.role === 'user' ? '#1890ff' : '#f0f0f0',
                  flexShrink: 0,
                }}
              >
                {msg.role === 'user' ? (
                  <UserOutlined style={{ color: '#fff', fontSize: 14 }} />
                ) : (
                  <RobotOutlined style={{ color: '#595959', fontSize: 14 }} />
                )}
              </div>
              <div
                style={{
                  maxWidth: '80%',
                  padding: '10px 14px',
                  borderRadius: 12,
                  background: msg.role === 'user' ? '#1890ff' : '#f6f6f6',
                  color: msg.role === 'user' ? '#fff' : '#262626',
                  fontSize: 14,
                  lineHeight: 1.6,
                }}
              >
                <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'inherit' }}>
                  {msg.content}
                </Paragraph>
                {msg.role === 'assistant' && (msg.agent_id || msg.confidence !== undefined) && (
                  <div style={{ marginTop: 8, display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    {msg.agent_id && (
                      <Tag style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}>
                        {msg.agent_id}
                      </Tag>
                    )}
                    {msg.confidence !== undefined && (
                      <Tag
                        color={confidenceColor(msg.confidence)}
                        style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}
                      >
                        {(msg.confidence * 100).toFixed(0)}%
                      </Tag>
                    )}
                    {msg.risk_level && (
                      <Tag
                        color={riskColor(msg.risk_level)}
                        style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}
                      >
                        {msg.risk_level}
                      </Tag>
                    )}
                    {msg.trace_id && (
                      <Text
                        code
                        style={{ fontSize: 10, color: '#8c8c8c' }}
                      >
                        {msg.trace_id.slice(0, 8)}
                      </Text>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}

          {/* Streaming message */}
          {loading && streamingText && (
            <div style={{ display: 'flex', gap: 8 }}>
              <div
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: 14,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: '#f0f0f0',
                  flexShrink: 0,
                }}
              >
                <RobotOutlined style={{ color: '#595959', fontSize: 14 }} />
              </div>
              <div
                style={{
                  maxWidth: '80%',
                  padding: '10px 14px',
                  borderRadius: 12,
                  background: '#f6f6f6',
                  color: '#262626',
                  fontSize: 14,
                  lineHeight: 1.6,
                }}
              >
                <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
                  {streamingText}
                  <span className="blinking-cursor" style={{ animation: 'blink 1s step-end infinite' }}>
                    |
                  </span>
                </Paragraph>
                {streamingMeta && (
                  <div style={{ marginTop: 8, display: 'flex', gap: 4, flexWrap: 'wrap' }}>
                    {streamingMeta.agent_id && (
                      <Tag style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}>
                        {streamingMeta.agent_id}
                      </Tag>
                    )}
                    {streamingMeta.confidence !== undefined && streamingMeta.confidence !== null && (
                      <Tag
                        color={confidenceColor(streamingMeta.confidence)}
                        style={{ fontSize: 11, lineHeight: '18px', margin: 0 }}
                      >
                        {(streamingMeta.confidence * 100).toFixed(0)}%
                      </Tag>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Input area */}
        <div style={{ padding: '12px 16px', borderTop: '1px solid #f0f0f0' }}>
          <Space.Compact style={{ width: '100%' }}>
            <TextArea
              placeholder="Ask about inventory, sales, listings..."
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              rows={2}
              style={{ flex: 1, resize: 'none' }}
              disabled={loading}
            />
            <Button
              type="primary"
              icon={loading ? <Spin size="small" /> : <SendOutlined />}
              onClick={handleSend}
              disabled={loading || !inputValue.trim()}
              style={{ height: 'auto' }}
            />
          </Space.Compact>
        </div>
      </div>

      <style jsx global>{`
        @keyframes blink {
          from, to { opacity: 1; }
          50% { opacity: 0; }
        }
      `}</style>
    </Drawer>
  );
}
