'use client';

import { useState, useEffect, useCallback } from 'react';
import WidgetButton from './WidgetButton';
import apiClient from '@/lib/api-client';

interface WidgetConfig {
  projectSlug?: string;
  projectId?: number;
  primaryColor?: string;
  position?: 'right' | 'left';
}

declare global {
  interface Window {
    __feedbackWidgetConfig?: WidgetConfig;
    __feedbackWidgetReady?: boolean;
  }
}

export default function Widget() {
  const [open, setOpen] = useState(false);
  const [config, setConfig] = useState<WidgetConfig>({});
  const [projectId, setProjectId] = useState<number | null>(null);

  // Form state
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [feedbackType, setFeedbackType] = useState('feature');
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  useEffect(() => {
    // Read config from global or data-* attributes
    const cfg = window.__feedbackWidgetConfig || {};
    setConfig(cfg);

    // Auto-detect project
    async function init() {
      try {
        const res = await apiClient.get<any[]>('/v1/feedback/projects');
        if (res.code === 0 && res.data && res.data.length > 0) {
          const p = cfg.projectId
            ? res.data.find((p: any) => p.id === cfg.projectId)
            : cfg.projectSlug
              ? res.data.find((p: any) => p.slug === cfg.projectSlug)
              : res.data[0];
          if (p) setProjectId(p.id);
        }
      } catch { /* ignore */ }
    }
    init();
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!projectId || !title.trim()) return;
    setSubmitting(true);
    try {
      await apiClient.post('/v1/feedback/submissions', {
        project_id: projectId,
        title: title.trim(),
        description: description.trim(),
        feedback_type: feedbackType,
        url: window.location.href,
        user_agent: navigator.userAgent,
      });
      setSubmitted(true);
      setTimeout(() => { setOpen(false); setSubmitted(false); setTitle(''); setDescription(''); }, 2000);
    } catch { /* ignore */ } finally { setSubmitting(false); }
  }, [projectId, title, description, feedbackType]);

  if (!projectId) return null;

  const theme = { primary: config.primaryColor, position: config.position };

  return (
    <>
      <WidgetButton onClick={() => setOpen(true)} theme={theme} />
      {open && (
        <div style={{
          position: 'fixed', bottom: 90, right: config.position === 'left' ? undefined : 24,
          left: config.position === 'left' ? 24 : undefined,
          width: 360, maxHeight: 500, background: '#fff', borderRadius: 12,
          boxShadow: '0 8px 30px rgba(0,0,0,0.18)', zIndex: 999999,
          display: 'flex', flexDirection: 'column', overflow: 'hidden',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
        }}>
          {/* Header */}
          <div style={{
            padding: '14px 16px', background: theme.primary || '#1677ff', color: '#fff',
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          }}>
            <span style={{ fontWeight: 600, fontSize: 15 }}>提交反馈</span>
            <button onClick={() => setOpen(false)} style={{
              background: 'none', border: 'none', color: '#fff', cursor: 'pointer', fontSize: 18, padding: 0,
            }}>✕</button>
          </div>

          {/* Body */}
          {submitted ? (
            <div style={{ padding: 40, textAlign: 'center', color: '#52c41a' }}>
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                <polyline points="22 4 12 14.01 9 11.01" />
              </svg>
              <div style={{ marginTop: 12, fontWeight: 500 }}>感谢你的反馈！</div>
              <div style={{ marginTop: 4, fontSize: 13, color: '#666' }}>我们会认真审阅每一条建议</div>
            </div>
          ) : (
            <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
              <input
                placeholder="一句话描述..."
                value={title} onChange={(e) => setTitle(e.target.value)}
                maxLength={500}
                style={{
                  width: '100%', padding: '10px 12px', border: '1px solid #d9d9d9',
                  borderRadius: 6, fontSize: 14, outline: 'none', boxSizing: 'border-box',
                }}
              />
              <textarea
                placeholder="详细描述你的想法、建议或遇到的问题..."
                value={description} onChange={(e) => setDescription(e.target.value)}
                rows={4}
                style={{
                  width: '100%', padding: '10px 12px', border: '1px solid #d9d9d9',
                  borderRadius: 6, fontSize: 14, resize: 'none', outline: 'none', boxSizing: 'border-box',
                  fontFamily: 'inherit',
                }}
              />
              <select
                value={feedbackType} onChange={(e) => setFeedbackType(e.target.value)}
                style={{
                  width: '100%', padding: '10px 12px', border: '1px solid #d9d9d9',
                  borderRadius: 6, fontSize: 14, outline: 'none', background: '#fff', boxSizing: 'border-box',
                }}
              >
                <option value="feature">功能需求</option>
                <option value="improvement">改进建议</option>
                <option value="bug">Bug 报告</option>
                <option value="question">问题咨询</option>
                <option value="other">其他</option>
              </select>
              <button
                onClick={handleSubmit}
                disabled={!title.trim() || submitting}
                style={{
                  width: '100%', padding: '10px', border: 'none', borderRadius: 6,
                  background: !title.trim() ? '#d9d9d9' : (theme.primary || '#1677ff'),
                  color: '#fff', fontSize: 14, cursor: !title.trim() ? 'not-allowed' : 'pointer',
                  fontWeight: 500,
                }}
              >
                {submitting ? '提交中...' : '提交反馈'}
              </button>
            </div>
          )}
        </div>
      )}
    </>
  );
}
