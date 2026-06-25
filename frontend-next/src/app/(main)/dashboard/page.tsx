'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { useAppStore } from '@/stores/app-store';
import apiClient from '@/lib/api-client';

interface OverviewData {
  order_total?: number;
  order_revenue?: number;
  order_profit?: number;
  sku_total?: number;
  low_stock_count?: number;
  out_of_stock_count?: number;
  listing_active_count?: number;
  aftersales_pending_count?: number;
  exception_open_count?: number;
  month_revenue?: number;
  month_cost?: number;
  order_by_status?: Record<string, number>;
}

export default function DashboardPage() {
  const router = useRouter();
  const { setActiveTool, toggleToolPanel } = useAppStore();
  const [input, setInput] = useState('');

  const { data: overview, isLoading } = useQuery<OverviewData>({
    queryKey: ['dashboard', 'overview'],
    queryFn: async () => {
      const res = await apiClient.get<OverviewData>('/v1/dashboard/overview');
      return res.data ?? {};
    },
  });

  const o = overview ?? {};

  const quickActions = [
    { label: '📦 商品管理', tool: 'products' as const, desc: `${o.sku_total ?? 0} 件商品` },
    { label: '📤 平台发布', tool: 'publish' as const, desc: `${o.listing_active_count ?? 0} 待处理` },
    { label: '📊 数据分析', tool: 'analytics' as const, desc: '查看运营报告' },
    { label: '💰 价格监控', tool: 'pricing' as const, desc: `${o.low_stock_count ?? 0} 件预警` },
  ];

  const openTool = (tool: string) => {
    setActiveTool(tool);
    toggleToolPanel();
  };

  const handleSend = () => {
    if (!input.trim()) return;
    // ponytail: navigate to /ai page with the command as context
    router.push('/ai');
    setInput('');
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
      {/* Agent greeting */}
      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
        <div
          style={{
            width: 30, height: 30, borderRadius: 8, flexShrink: 0,
            background: 'linear-gradient(135deg, var(--i5), var(--c5))',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: '0.75rem', color: 'white', marginTop: 2,
          }}
        >
          ✦
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: '0.72rem', fontWeight: 600, color: 'var(--t2)', marginBottom: 2 }}>
            凌镜 Agent
          </div>
          <div
            style={{
              fontSize: '0.88rem', lineHeight: 1.6, color: 'var(--t1)',
              background: 'var(--s1)', padding: '10px 14px',
              borderRadius: 8, border: '1px solid var(--bd)',
            }}
          >
            ☀️ 早上好！昨晚已完成数据同步。
            {!isLoading && (
              <>
                {' '}目前管理 <span style={{ color: 'var(--c4)', fontWeight: 500 }}>{o.sku_total ?? 0}</span> 件商品，
                <span style={{ color: 'var(--r4)', fontWeight: 500 }}>{o.low_stock_count ?? 0}</span> 件库存预警，
                本月收入 <span style={{ color: 'var(--g4)', fontWeight: 500 }}>¥{(o.month_revenue ?? 0).toLocaleString()}</span>。
              </>
            )}
          </div>
        </div>
      </div>

      {/* Stats row */}
      {!isLoading && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 8 }}>
          {[
            { label: '商品总数', value: o.sku_total ?? 0, color: 'var(--t1)' },
            { label: '订单收入', value: `¥${(o.order_revenue ?? 0).toLocaleString()}`, color: 'var(--g4)' },
            { label: '低库存', value: o.low_stock_count ?? 0, color: o.low_stock_count ? 'var(--r4)' : 'var(--g4)' },
            { label: '异常', value: o.exception_open_count ?? 0, color: o.exception_open_count ? 'var(--y4)' : 'var(--g4)' },
          ].map(s => (
            <div
              key={s.label}
              style={{
                background: 'var(--s1)', borderRadius: 8, padding: '10px 12px',
                border: '1px solid var(--bd)',
              }}
            >
              <div style={{ fontSize: '0.65rem', fontWeight: 600, letterSpacing: '0.05em', textTransform: 'uppercase', color: 'var(--t4)' }}>
                {s.label}
              </div>
              <div style={{ fontFamily: 'var(--ds)', fontWeight: 700, fontSize: '1.2rem', color: s.color, marginTop: 2 }}>
                {s.value}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Quick actions */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        <div style={{ fontSize: '0.62rem', fontWeight: 600, letterSpacing: '0.07em', textTransform: 'uppercase', color: 'var(--t4)' }}>
          ▸ 快捷操作
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(160px, 1fr))', gap: 6 }}>
          {quickActions.map(a => (
            <button
              key={a.tool}
              onClick={() => openTool(a.tool)}
              style={{
                display: 'flex', alignItems: 'center', gap: 8,
                padding: '8px 10px', borderRadius: 6,
                background: 'var(--s1)', border: '1px solid var(--bd)',
                color: 'var(--t2)', cursor: 'pointer',
                fontFamily: 'var(--body)', fontSize: '0.78rem',
                transition: 'background 80ms',
                textAlign: 'left',
              }}
            >
              <span style={{ fontSize: '1rem' }}>{a.label.split(' ')[0]}</span>
              <div style={{ minWidth: 0 }}>
                <div style={{ fontWeight: 500, color: 'var(--t1)' }}>{a.label.split(' ').slice(1).join(' ')}</div>
                <div style={{ fontSize: '0.65rem', color: 'var(--t3)' }}>{a.desc}</div>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {/* Input bar */}
      <div
        style={{
          display: 'flex', gap: 6, alignItems: 'center',
          padding: '6px 10px 8px',
          borderTop: '1px solid var(--bd)',
        }}
      >
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleSend()}
          placeholder="告诉凌镜做什么... '发布蓝牙耳机到 Ozon' '分析上周销售'"
          style={{
            flex: 1, padding: '8px 14px', borderRadius: 10,
            background: 'var(--s2)', border: '1px solid var(--bd2)',
            fontFamily: 'var(--body)', fontSize: '0.82rem',
            color: 'var(--t1)', outline: 'none',
          }}
        />
        <button
          onClick={handleSend}
          style={{
            width: 32, height: 32, borderRadius: 6,
            background: 'var(--i5)', border: 'none', color: 'white',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            cursor: 'pointer', fontSize: '0.85rem',
            flexShrink: 0,
          }}
        >
          ↵
        </button>
      </div>
    </div>
  );
}
