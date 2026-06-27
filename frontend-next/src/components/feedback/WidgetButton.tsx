'use client';

import { useState, useCallback } from 'react';

interface WidgetButtonProps {
  onClick: () => void;
  theme?: { primary?: string; position?: 'right' | 'left' };
}

export default function WidgetButton({ onClick, theme }: WidgetButtonProps) {
  const [hover, setHover] = useState(false);
  const pos = theme?.position === 'left' ? { left: 24 } : { right: 24 };
  const color = theme?.primary || '#1677ff';

  return (
    <div style={{
      position: 'fixed', bottom: 24, ...pos, zIndex: 999999,
    }}>
      <button
        onClick={onClick}
        onMouseEnter={() => setHover(true)}
        onMouseLeave={() => setHover(false)}
        style={{
          width: hover ? 120 : 56, height: 56, borderRadius: 28,
          background: `linear-gradient(135deg, ${color}, ${color}dd)`,
          border: 'none', cursor: 'pointer', display: 'flex',
          alignItems: 'center', justifyContent: 'center', gap: 6,
          boxShadow: '0 4px 14px rgba(0,0,0,0.25)',
          transition: 'all 0.2s ease', color: '#fff',
          fontSize: 14, fontWeight: 500, overflow: 'hidden',
          whiteSpace: 'nowrap',
        }}
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor"
          strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0 }}>
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
        {hover && <span>反馈建议</span>}
      </button>
    </div>
  );
}
