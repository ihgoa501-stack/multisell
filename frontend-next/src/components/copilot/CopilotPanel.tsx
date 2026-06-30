'use client';

import { useState, useRef, useEffect } from 'react';
import { message } from 'antd';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  agent_id?: string;
  confidence?: number;
  risk_level?: string;
  trace_id?: string;
}

function getAuthHeaders(): HeadersInit {
  const headers: HeadersInit = { 'Content-Type': 'application/json' };
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

function StreamingDots() {
  return (
    <span style={{ display: 'inline-flex', gap: 3, alignItems: 'center', marginLeft: 2 }}>
      {[0, 1, 2].map((i) => (
        <span
          key={i}
          style={{
            width: 4,
            height: 4,
            borderRadius: '50%',
            background: 'var(--c4)',
            animation: `pulse-dot 1.4s ease-in-out ${i * 0.2}s infinite`,
          }}
        />
      ))}
    </span>
  );
}

export default function CopilotPanel() {
  const [inputValue, setInputValue] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      role: 'assistant',
      content: '你好！我是凌镜 AI 助理，可以帮你管理商品、分析库存、优化刊登。有什么需要帮忙的吗？',
    },
  ]);
  const [loading, setLoading] = useState(false);
  const [streamingText, setStreamingText] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingText]);

  const handleSend = async () => {
    const msg = inputValue.trim();
    if (!msg || loading) return;
    setInputValue('');
    setMessages((prev) => [...prev, { role: 'user', content: msg }]);
    setLoading(true);
    setStreamingText('');

    try {
      const response = await fetch(`${API_BASE}/v1/ai/chat`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), Accept: 'text/event-stream' },
        body: JSON.stringify({ message: msg, stream: true }),
      });

      if (!response.ok) throw new Error(`HTTP ${response.status}`);

      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let buffer = '';
      let fullText = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const chunk = JSON.parse(line.slice(6));
              const text = typeof chunk.text === 'string' ? chunk.text : '';
              if (text) {
                fullText += text;
                setStreamingText(fullText);
              }
            } catch { /* ignore */ }
          }
        }
      }

      if (fullText) {
        setMessages((prev) => [...prev, { role: 'assistant', content: fullText }]);
      }
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      message.error('AI 响应失败');
    } finally {
      setLoading(false);
      setStreamingText('');
      inputRef.current?.focus();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <>
      <style>{`
        @keyframes pulse-dot {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.3; transform: scale(0.8); }
        }
      `}</style>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', fontFamily: 'var(--body)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '6px 12px', borderBottom: '1px solid var(--bd)', flexShrink: 0 }}>
          <div style={{ width: 18, height: 18, borderRadius: 6, background: 'linear-gradient(135deg, var(--i5), var(--c5))', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: '0.6rem', fontWeight: 700 }}>AI</div>
          <span style={{ fontFamily: 'var(--ds)', fontWeight: 600, fontSize: '0.78rem', color: 'var(--t1)', flex: 1 }}>AI 对话</span>
          {loading && <StreamingDots />}
        </div>

        <div style={{ flex: 1, padding: 10, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 10 }}>
          {messages.map((msg, idx) => (
            <div key={idx} style={{ display: 'flex', flexDirection: msg.role === 'user' ? 'row-reverse' : 'row', gap: 6, alignItems: 'flex-start' }}>
              <div style={{ width: 24, height: 24, borderRadius: 6, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, fontSize: '0.55rem', fontWeight: 700, fontFamily: 'var(--ds)', background: msg.role === 'user' ? 'var(--i5)' : 'linear-gradient(135deg, var(--i5), var(--c5))', color: '#fff' }}>
                {msg.role === 'user' ? 'U' : 'AI'}
              </div>
              <div style={{ maxWidth: '85%', minWidth: 0 }}>
                <div style={{ padding: '8px 10px', borderRadius: msg.role === 'user' ? '8px 8px 2px 8px' : '8px 8px 8px 2px', background: msg.role === 'user' ? 'var(--i5)' : 'var(--s2)', color: msg.role === 'user' ? '#fff' : 'var(--t1)', fontSize: '0.78rem', lineHeight: 1.55, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                  {msg.content}
                </div>
              </div>
            </div>
          ))}
          {loading && streamingText && (
            <div style={{ display: 'flex', gap: 6, alignItems: 'flex-start' }}>
              <div style={{ width: 24, height: 24, borderRadius: 6, background: 'linear-gradient(135deg, var(--i5), var(--c5))', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, fontSize: '0.55rem', fontWeight: 700, fontFamily: 'var(--ds)', color: '#fff' }}>AI</div>
              <div style={{ maxWidth: '85%' }}>
                <div style={{ padding: '8px 10px', borderRadius: '8px 8px 8px 2px', background: 'var(--s2)', color: 'var(--t1)', fontSize: '0.78rem', lineHeight: 1.55, whiteSpace: 'pre-wrap' }}>
                  {streamingText}<StreamingDots />
                </div>
              </div>
            </div>
          )}
          {loading && !streamingText && (
            <div style={{ display: 'flex', gap: 6, alignItems: 'flex-start' }}>
              <div style={{ width: 24, height: 24, borderRadius: 6, background: 'linear-gradient(135deg, var(--i5), var(--c5))', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, fontSize: '0.55rem', fontWeight: 700, fontFamily: 'var(--ds)', color: '#fff' }}>AI</div>
              <div style={{ padding: '8px 10px', borderRadius: '8px 8px 8px 2px', background: 'var(--s2)', display: 'flex', alignItems: 'center' }}><StreamingDots /></div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        <div style={{ padding: '8px 10px', borderTop: '1px solid var(--bd)', flexShrink: 0 }}>
          <div style={{ display: 'flex', gap: 6, alignItems: 'flex-end', background: 'var(--s2)', border: '1px solid var(--bd2)', borderRadius: 8, padding: '6px 8px' }}>
            <textarea ref={inputRef} placeholder="输入消息，Enter 发送..." value={inputValue} onChange={(e) => setInputValue(e.target.value)} onKeyDown={handleKeyDown} rows={2} disabled={loading} style={{ flex: 1, border: 'none', background: 'transparent', color: 'var(--t1)', fontFamily: 'var(--body)', fontSize: '0.78rem', lineHeight: 1.5, resize: 'none', outline: 'none', padding: 0 }} />
            <button onClick={handleSend} disabled={loading || !inputValue.trim()} style={{ width: 28, height: 28, borderRadius: 6, border: 'none', cursor: loading || !inputValue.trim() ? 'default' : 'pointer', background: loading || !inputValue.trim() ? 'var(--s3)' : 'linear-gradient(135deg, var(--i5), var(--c5))', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, opacity: loading || !inputValue.trim() ? 0.5 : 1 }}>
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" style={{ transform: 'rotate(-30deg)' }}><path d="M2 10L10 6L6 2V8L2 10Z" fill="#fff" /></svg>
            </button>
          </div>
          <div style={{ fontSize: '0.55rem', color: 'var(--t4)', marginTop: 4, textAlign: 'center' }}>AI 助理可能不准确，请核实重要信息</div>
        </div>
      </div>
    </>
  );
}
