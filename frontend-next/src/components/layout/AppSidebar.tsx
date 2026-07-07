 'use client';

import { useEffect, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { useAppStore } from '@/stores/app-store';
import { usePermissionStore } from '@/stores/permission-store';
import { menuGroups, statusLabels, type MenuItem } from '@/config/menu';
import { ApiClient } from '@/lib/api-client';
import {
  ShoppingOutlined,
  SendOutlined,
  BarChartOutlined,
  DollarOutlined,
  BulbOutlined,
} from '@ant-design/icons';

type SessionItem = {
  key: string;
  label: string;
  group: string;
  status?: string;
};

function elapsedStr(startedAt: Date): string {
  const diff = Date.now() - startedAt.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return '<1m';
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  const rem = mins % 60;
  return `${hours}h ${rem}m`;
}

export default function AppSidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { toggleToolPanel, setActiveTool } = useAppStore();
  const { fetchPermissions, hasPermission } = usePermissionStore();
  const [, setTick] = useState(0);

  useEffect(() => {
    fetchPermissions();

    // Register the 403 handler: when the backend returns 403, re-fetch permissions.
    ApiClient.setForbiddenHandler(() => {
      const { clearPermissions, fetchPermissions } = usePermissionStore.getState();
      clearPermissions();
      fetchPermissions();
    });

    return () => {
      ApiClient.setForbiddenHandler(null);
    };
  }, [fetchPermissions]);

  // Tick every 30s to update elapsed times
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 30000);
    return () => clearInterval(id);
  }, []);

  function isItemVisible(item: MenuItem): boolean {
    if (!item.permission) return true;
    return hasPermission(item.permission);
  }

  // Flatten visible menu items into session list preserving group labels
  const sessions: SessionItem[] = menuGroups
    .filter((g) => g.items.some(isItemVisible))
    .flatMap((g) =>
      g.items.filter(isItemVisible).map((item) => ({
        key: item.key,
        label: item.label,
        group: g.label,
        status: item.status,
      }))
    );

  // Track which session is active based on path prefix
  const activeSession =
    sessions.find((s) => pathname.startsWith(s.key))?.key ?? '';

  const [runningTasks] = useState(() => [
    { label: '补货 Shopee', status: '3/3', color: 'var(--i4)', startedAt: new Date(Date.now() - 3 * 60 * 1000) },
    { label: '标题优化', status: '5/12', color: 'var(--y4)', startedAt: new Date(Date.now() - 15 * 60 * 1000) },
    { label: 'Ozon 价格对比', status: '✓', color: 'var(--g4)', startedAt: new Date(Date.now() - 30 * 60 * 1000) },
  ]);

  const toolButtons = [
    { icon: <ShoppingOutlined />, label: '商品管理', badge: '2,847', tool: 'products' },
    { icon: <SendOutlined />, label: '平台发布', badge: '8 待处理', tool: 'publish' },
    { icon: <BarChartOutlined />, label: '数据分析', badge: null, tool: 'analytics' },
    { icon: <DollarOutlined />, label: '价格监控', badge: null, tool: 'price' },
    { icon: <BulbOutlined />, label: 'AI 文案', badge: null, tool: 'copywriting' },
  ];

  return (
    <div
      style={{
        width: 200,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        background: 'var(--s1)',
        borderRight: '1px solid var(--bd)',
        overflow: 'hidden',
        fontSize: '0.85rem',
        color: 'var(--t1)',
      }}
    >
      {/* Agent profile card */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 10px 6px',
          borderBottom: '1px solid var(--bd)',
          cursor: 'pointer',
        }}
        onClick={() => router.push('/ai')}
      >
        <div
          style={{
            position: 'relative',
            width: 30,
            height: 30,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 30,
              height: 30,
              borderRadius: 8,
              background: 'linear-gradient(135deg, var(--i5), var(--c5))',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#fff',
              fontSize: '0.85rem',
              fontWeight: 700,
              fontFamily: 'var(--mono)',
            }}
          >
            L
          </div>
          <div
            style={{
              position: 'absolute',
              bottom: -1,
              right: -1,
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: 'var(--g4)',
              border: '2px solid var(--s1)',
            }}
          />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: '0.72rem',
              fontWeight: 600,
              lineHeight: 1.2,
              color: 'var(--t1)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            凌镜 Agent
          </div>
          <div
            style={{
              fontSize: '0.6rem',
              color: 'var(--t3)',
              lineHeight: 1.2,
            }}
          >
            跨境电商运营
          </div>
        </div>
        <div
          style={{
            fontSize: '0.58rem',
            color: 'var(--g4)',
            whiteSpace: 'nowrap',
            fontWeight: 500,
          }}
        >
          3 运行中
        </div>
      </div>

      {/* Scrollable content area */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          overflowX: 'hidden',
        }}
      >
        {/* Running tasks section */}
        <div style={{ padding: '6px 10px 2px' }}>
          <style>{`@keyframes pulse-dot { 0%,100% { opacity:1 } 50% { opacity:0.3 } }`}</style>
          <div
            style={{
              fontSize: '0.58rem',
              letterSpacing: '0.07em',
              textTransform: 'uppercase',
              color: 'var(--t4)',
              marginBottom: 4,
              fontWeight: 600,
            }}
          >
            ▸ 正在运行
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {runningTasks.map((task) => {
              const isDone = task.status === '✓';
              return (
                <div
                  key={task.label}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 4,
                    padding: '2px 4px',
                    fontSize: '0.7rem',
                    borderRadius: 3,
                    cursor: 'default',
                    color: 'var(--t2)',
                    opacity: isDone ? 0.5 : 1,
                    transition: 'opacity var(--dur-micro)',
                  }}
                >
                  <div
                    style={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      background: task.color,
                      flexShrink: 0,
                      animation: isDone ? 'none' : 'pulse-dot 1.5s ease-in-out infinite',
                    }}
                  />
                  <span
                    style={{
                      flex: 1,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {task.label}
                  </span>
                  <span
                    style={{
                      color: isDone ? 'var(--g4)' : 'var(--t3)',
                      flexShrink: 0,
                    }}
                  >
                    {isDone ? task.status : `${task.status} · ${elapsedStr(task.startedAt)}`}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Trust score section */}
        <div style={{ padding: '6px 10px 8px' }}>
          <div
            style={{
              fontSize: '0.58rem',
              letterSpacing: '0.07em',
              textTransform: 'uppercase',
              color: 'var(--t4)',
              marginBottom: 6,
              fontWeight: 600,
            }}
          >
            ▸ 信任指数
          </div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
            }}
          >
            <div style={{ flex: 1 }}>
              <div
                style={{
                  height: 6,
                  borderRadius: 3,
                  background: 'var(--s2)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    width: '85%',
                    height: '100%',
                    borderRadius: 3,
                    background: 'linear-gradient(90deg, var(--i4), var(--c4))',
                    transition: 'width 0.5s ease',
                  }}
                />
              </div>
            </div>
            <span
              style={{
                fontSize: '1.1rem',
                fontWeight: 700,
                fontFamily: 'var(--mono)',
                color: 'var(--t1)',
                lineHeight: 1,
              }}
            >
              85
            </span>
          </div>
          <div
            style={{
              fontSize: '0.6rem',
              color: 'var(--t3)',
              marginTop: 2,
            }}
          >
            基于 127 次决策 · 良好
          </div>
        </div>

        {/* Tools section */}
        <div style={{ padding: '6px 10px 2px' }}>
          <div
            style={{
              fontSize: '0.58rem',
              letterSpacing: '0.07em',
              textTransform: 'uppercase',
              color: 'var(--t4)',
              marginBottom: 4,
              fontWeight: 600,
            }}
          >
            ▸ 常用工具
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            {toolButtons.map((btn) => (
              <div
                key={btn.tool}
                onClick={() => {
                  toggleToolPanel();
                  setActiveTool(btn.tool);
                }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 5,
                  padding: '3px 5px',
                  fontSize: '0.7rem',
                  borderRadius: 4,
                  cursor: 'pointer',
                  color: 'var(--t2)',
                  transition: 'background var(--dur-micro)',
                }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.background =
                    'var(--s2)';
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.background =
                    'transparent';
                }}
              >
                <span style={{ fontSize: '0.75rem', flexShrink: 0 }}>
                  {btn.icon}
                </span>
                <span
                  style={{
                    flex: 1,
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {btn.label}
                </span>
                {btn.badge && (
                  <span
                    style={{
                      fontSize: '0.55rem',
                      background: 'var(--r1)',
                      color: 'var(--r4)',
                      padding: '0 4px',
                      borderRadius: 6,
                      lineHeight: '1.3',
                      fontWeight: 600,
                      flexShrink: 0,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {btn.badge}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Sessions section */}
        <div style={{ padding: '6px 10px 2px' }}>
          <div
            style={{
              fontSize: '0.58rem',
              letterSpacing: '0.07em',
              textTransform: 'uppercase',
              color: 'var(--t4)',
              marginBottom: 4,
              fontWeight: 600,
            }}
          >
            ▸ 最近会话
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {sessions.map((session) => {
              const isActive = activeSession === session.key;
              return (
                <div
                  key={session.key}
                  onClick={() => router.push(session.key)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '4px 10px',
                    fontSize: '0.72rem',
                    borderRadius: 4,
                    cursor: 'pointer',
                    color: isActive ? 'var(--t1)' : 'var(--t2)',
                    fontWeight: isActive ? 600 : 400,
                    borderLeft: isActive
                      ? '2px solid var(--i4)'
                      : '2px solid transparent',
                    background: isActive ? 'var(--s2)' : 'transparent',
                    transition: 'background var(--dur-micro)',
                  }}
                  onMouseEnter={(e) => {
                    if (!isActive) {
                      (e.currentTarget as HTMLElement).style.background =
                        'var(--s2)';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!isActive) {
                      (e.currentTarget as HTMLElement).style.background =
                        'transparent';
                    }
                  }}
                >
                  <span
                    style={{
                      flex: 1,
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {session.label}
                  </span>
                  {session.status && (
                    <span
                      style={{
                        fontSize: '0.5rem',
                        padding: '0 3px',
                        borderRadius: 3,
                        background: session.status === 'mock' ? 'var(--y2)' : 'var(--b2)',
                        color: session.status === 'mock' ? 'var(--y5)' : 'var(--b5)',
                        fontWeight: 600,
                        flexShrink: 0,
                        lineHeight: '1.4',
                      }}
                    >
                      {statusLabels[session.status] ?? session.status}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* New session button */}
      <div
        style={{
          padding: '6px 10px',
          borderTop: '1px solid var(--bd)',
        }}
      >
        <div
          onClick={() => router.push('/ai')}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 4,
            padding: '4px 0',
            fontSize: '0.7rem',
            borderRadius: 5,
            cursor: 'pointer',
            color: 'var(--i4)',
            border: '1px dashed var(--i4)',
            transition: 'background var(--dur-micro)',
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLElement).style.background = 'var(--s2)';
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLElement).style.background = 'transparent';
          }}
        >
          + 新会话
        </div>
      </div>
    </div>
  );
}
