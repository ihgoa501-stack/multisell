'use client';

import { useEffect, useState } from 'react';
import { useAppStore } from '@/stores/app-store';

interface Activity {
  id: string;
  icon: string;
  message: string;
  timestamp: Date;
}

const activitiesData: Activity[] = [
  {
    id: '1',
    icon: '📦',
    message: 'Agent 补货 Shopee 完成 (3 件)',
    timestamp: new Date(Date.now() - 3 * 60 * 1000),
  },
  {
    id: '2',
    icon: '✏️',
    message: 'Agent 标题优化 更新 (5/12)',
    timestamp: new Date(Date.now() - 15 * 60 * 1000),
  },
  {
    id: '3',
    icon: '🔔',
    message: 'Agent 价格监控 发现 3 件预警',
    timestamp: new Date(Date.now() - 60 * 60 * 1000),
  },
];

function relativeTime(date: Date): string {
  const diff = Date.now() - date.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return '刚刚';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export default function ActivityFeed() {
  const { activityFeedOpen, markActivitiesRead } = useAppStore();
  const [activities] = useState(activitiesData);

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
        {activities.length === 0 ? (
          <div
            style={{
              padding: '24px 12px',
              textAlign: 'center',
              fontSize: '0.72rem',
              color: 'var(--t3)',
            }}
          >
            暂无活动
          </div>
        ) : (
          activities.map((activity) => (
            <div
              key={activity.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: 8,
                padding: '8px 12px',
                fontSize: '0.72rem',
                color: 'var(--t2)',
                borderBottom: '1px solid var(--bd)',
              }}
            >
              <span style={{ flexShrink: 0, fontSize: '0.85rem' }}>
                {activity.icon}
              </span>
              <span style={{ flex: 1, lineHeight: 1.4 }}>
                {activity.message}
              </span>
              <span
                style={{
                  flexShrink: 0,
                  fontSize: '0.6rem',
                  color: 'var(--t4)',
                  whiteSpace: 'nowrap',
                }}
              >
                {relativeTime(activity.timestamp)}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
