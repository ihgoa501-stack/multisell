'use client';

import { Button, Space, Avatar } from 'antd';
import { SearchOutlined, SunOutlined, BellOutlined } from '@ant-design/icons';
import { useAppStore } from '@/stores/app-store';
import ActivityFeed from '@/components/layout/ActivityFeed';

function toggleTheme() {
  const html = document.documentElement;
  const next = html.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
  html.setAttribute('data-theme', next);
  window.dispatchEvent(new CustomEvent('themechange', { detail: next }));
}

export default function AppHeader() {
  const { setCommandPaletteOpen, toggleActivityFeed, unseenCount, activityFeedOpen } = useAppStore();

  return (
    <>
      {/* Brand */}
      <span
        style={{
          fontFamily: 'var(--ds)',
          fontWeight: 700,
          fontSize: '0.9rem',
          color: 'var(--t1)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          flexShrink: 0,
        }}
      >
        <span
          style={{
            background: 'linear-gradient(135deg, var(--i4), var(--c4))',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
          }}
        >
          ◆
        </span>
        凌镜
      </span>

      {/* Agent status */}
      <span
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 5,
          padding: '2px 10px 2px 6px',
          borderRadius: 100,
          background: 'rgba(34,211,238,0.08)',
          border: '1px solid rgba(34,211,238,0.12)',
          fontSize: '0.68rem',
          color: 'var(--c4)',
          flexShrink: 0,
        }}
      >
        <span style={{
          width: 4, height: 4, borderRadius: '50%',
          background: 'var(--c4)',
          display: 'inline-block',
        }} />
        Agent 状态未核验
      </span>

      <div style={{ flex: 1 }} />

      {/* Right actions */}
      <Space size="small">
        <Button
          icon={<SearchOutlined />}
          onClick={() => setCommandPaletteOpen(true)}
          type="text"
          size="small"
          style={{ color: 'var(--t2)', fontSize: '0.8rem', minWidth: 44, minHeight: 44 }}
        >
          /
        </Button>
        <Button
          type="text"
          icon={<SunOutlined />}
          onClick={toggleTheme}
          size="small"
          style={{ color: 'var(--t2)', minWidth: 44, minHeight: 44 }}
        />
        <div style={{ position: 'relative' }}>
          <Button
            type="text"
            icon={<BellOutlined />}
            onClick={toggleActivityFeed}
            size="small"
            style={{ color: 'var(--t2)', position: 'relative', minWidth: 44, minHeight: 44 }}
          />
          {unseenCount > 0 && !activityFeedOpen && (
            <div
              style={{
                position: 'absolute',
                top: 2,
                right: 2,
                width: 6,
                height: 6,
                borderRadius: '50%',
                background: 'var(--r4)',
                pointerEvents: 'none',
              }}
            />
          )}
          <ActivityFeed />
        </div>
        <Avatar
          size={24}
          style={{
            backgroundColor: 'var(--i5)',
            fontFamily: 'var(--ds)',
            fontSize: '0.75rem',
            cursor: 'pointer',
          }}
        >
          L
        </Avatar>
      </Space>
    </>
  );
}
