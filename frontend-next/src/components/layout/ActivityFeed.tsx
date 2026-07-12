'use client';

import { useEffect } from 'react';
import { useAppStore } from '@/stores/app-store';

export default function ActivityFeed() {
  const { activityFeedOpen, markActivitiesRead } = useAppStore();

  useEffect(() => {
    if (activityFeedOpen) {
      markActivitiesRead();
    }
  }, [activityFeedOpen, markActivitiesRead]);

  if (!activityFeedOpen) return null;

  return (
    <div
      style={{
        position: 'absolute',
        top: '100%',
        right: 0,
        width: 320,
        maxHeight: 380,
        background: 'var(--s1)',
        border: '1px solid var(--bd)',
        borderRadius: 12,
        boxShadow: '0 8px 24px rgba(0,0,0,0.12)',
        overflow: 'hidden',
        zIndex: 1000,
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        style={{
          padding: '10px 12px',
          fontSize: '0.78rem',
          fontWeight: 600,
          color: 'var(--t1)',
          borderBottom: '1px solid var(--bd)',
        }}
      >
        活动通知
      </div>
      <div style={{ overflowY: 'auto', flex: 1 }}>
        <div style={{ padding: '24px 12px', textAlign: 'center', fontSize: '0.72rem', color: 'var(--t3)' }}>
          暂无真实活动
        </div>
      </div>
    </div>
  );
}
