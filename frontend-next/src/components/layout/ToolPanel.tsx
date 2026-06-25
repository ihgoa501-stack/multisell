'use client';

import { useAppStore } from '@/stores/app-store';

const tools = [
  { id: 'products', icon: '📦', label: '商品管理', badge: '2,847' },
  { id: 'publish', icon: '📤', label: '平台发布', badge: '8 待处理' },
  { id: 'analytics', icon: '📊', label: '数据分析', badge: undefined },
  { id: 'pricing', icon: '💰', label: '价格监控', badge: undefined },
  { id: 'ai-copy', icon: '✦', label: 'AI 文案', badge: undefined },
];

/* ─── Data ─── */

const products = [
  { name: '蓝牙耳机', sku: 'BH-1001', stock: 1568, price: 25.99, suggested: 27.50, status: '在售' },
  { name: '充电宝', sku: 'PB-2002', stock: 892, price: 18.50, suggested: 19.99, status: '在售' },
  { name: '运动手环 Pro', sku: 'FB-3003', stock: 23, price: 39.99, suggested: 42.00, status: '低库存' },
  { name: 'USB-C 扩展坞', sku: 'UD-4004', stock: 0, price: 32.00, suggested: 34.50, status: '缺货' },
  { name: '降噪头戴耳机', sku: 'NC-5005', stock: 447, price: 89.99, suggested: 92.00, status: '在售' },
];

const publishItems = [
  { name: '蓝牙耳机 — Shopee', platform: 'Shopee', status: '待发布' },
  { name: '蓝牙耳机 — Lazada', platform: 'Lazada', status: '待发布' },
  { name: '充电宝 — Shopee', platform: 'Shopee', status: '已发布' },
  { name: '充电宝 — Lazada', platform: 'Lazada', status: '失败' },
];

const analyticsKpis = [
  { label: '总销售额', value: '$12,847', change: '+12.3%' },
  { label: '订单数', value: '342', change: '+8.1%' },
];

const analyticsRows = [
  { date: '06/21', sales: '$2,847.50', orders: 68, platform: 'Shopee' },
  { date: '06/20', sales: '$2,103.20', orders: 55, platform: 'Lazada' },
  { date: '06/19', sales: '$1,998.00', orders: 47, platform: 'Shopee' },
  { date: '06/18', sales: '$3,241.80', orders: 82, platform: 'Shopee' },
  { date: '06/17', sales: '$2,656.50', orders: 90, platform: 'Lazada' },
];

const priceComparisons = [
  { name: '蓝牙耳机', current: 25.99, avg: 27.50, suggested: 27.50, diff: -1.51 },
  { name: '充电宝', current: 18.50, avg: 22.00, suggested: 19.99, diff: -3.50 },
  { name: '运动手环', current: 39.99, avg: 44.00, suggested: 42.00, diff: -4.01 },
];

const aiOptimizations = [
  {
    name: '蓝牙耳机',
    old: 'Wireless Bluetooth 5.3 Earphones HiFi Stereo',
    new: 'Bluetooth 5.3 Earphones | HiFi Stereo | 30H Battery Life',
  },
  {
    name: '充电宝',
    old: '20000mAh Portable Power Bank Fast Charging',
    new: '20000mAh Power Bank | 65W PD Fast Charge | 2 Devices',
  },
];

/* ─── Shared style helpers ─── */

const cell: React.CSSProperties = {
  padding: '0 4px',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};

const btnBase: React.CSSProperties = {
  border: 'none',
  borderRadius: 4,
  cursor: 'pointer',
  fontFamily: 'var(--body)',
  fontSize: '0.7rem',
  lineHeight: '1.6',
  padding: '2px 8px',
};

function stockColor(v: number) {
  if (v === 0) return 'var(--r4)';
  if (v < 50) return 'var(--y4)';
  return 'var(--g4)';
}

function statusBadge(status: string) {
  let bg: string;
  if (status === '在售') bg = 'var(--g4)';
  else if (status === '低库存') bg = 'var(--y4)';
  else if (status === '缺货' || status === '失败') bg = 'var(--r4)';
  else if (status === '待发布') bg = 'var(--y4)';
  else if (status === '已发布') bg = 'var(--g4)';
  else bg = 'var(--t3)';

  return (
    <span
      style={{
        display: 'inline-block',
        padding: '0 6px',
        borderRadius: 3,
        background: bg,
        color: '#fff',
        fontSize: '0.65rem',
        fontWeight: 500,
        lineHeight: '1.6',
        fontFamily: 'var(--body)',
      }}
    >
      {status}
    </span>
  );
}

/* ─── Filter bar btn ─── */

function FilterBtn({ label, count, active }: { label: string; count?: number; active?: boolean }) {
  return (
    <button
      style={{
        ...btnBase,
        background: active ? 'var(--i4)' : 'var(--bg)',
        color: active ? '#fff' : 'var(--t3)',
        border: active ? 'none' : '1px solid var(--bd)',
        fontWeight: active ? 600 : 400,
      }}
    >
      {label}{count != null ? ` (${count})` : ''}
    </button>
  );
}

/* ─── Tool sections ─── */

function ProductsContent() {
  const gridCols = '20px 1fr 60px 44px 56px 56px 52px';
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Agent banner */}
      <div
        style={{
          margin: '0 0 8px',
          padding: '6px 10px',
          background: 'var(--bg)',
          borderRadius: 6,
          fontSize: '0.7rem',
          fontFamily: 'var(--body)',
          color: 'var(--t2)',
          border: '1px solid var(--bd)',
        }}
      >
        {'✦'} AI 助理已筛选 2,847 件商品，展示 Top 5
      </div>

      {/* Header row */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: gridCols,
          gap: 2,
          padding: '6px 0',
          borderBottom: '1px solid var(--bd)',
          fontSize: '0.7rem',
          color: 'var(--t4)',
          fontFamily: 'var(--body)',
        }}
      >
        <input type="checkbox" style={{ accentColor: 'var(--i4)' }} />
        <span style={cell}>名称</span>
        <span style={cell}>SKU</span>
        <span style={cell}>库存</span>
        <span style={cell}>当前价</span>
        <span style={{ ...cell, color: 'var(--c4)' }}>建议价</span>
        <span style={cell}>状态</span>
      </div>

      {/* Rows */}
      {products.map((p, i) => (
        <div
          key={p.sku}
          style={{
            display: 'grid',
            gridTemplateColumns: gridCols,
            gap: 2,
            padding: '5px 0',
            borderBottom: i < products.length - 1 ? '1px solid var(--bd2)' : 'none',
            fontSize: '0.7rem',
            fontFamily: 'var(--body)',
            color: 'var(--t2)',
            alignItems: 'center',
          }}
        >
          <input type="checkbox" defaultChecked={i < 3} style={{ accentColor: 'var(--i4)' }} />
          <span style={cell}>{p.name}</span>
          <span style={{ ...cell, color: 'var(--t3)' }}>{p.sku}</span>
          <span style={{ ...cell, color: stockColor(p.stock), fontWeight: 600 }}>{p.stock}</span>
          <span style={cell}>${p.price.toFixed(2)}</span>
          <span style={{ ...cell, color: 'var(--c4)', fontWeight: 600 }}>${p.suggested.toFixed(2)}</span>
          <span style={cell}>{statusBadge(p.status)}</span>
        </div>
      ))}

      {/* Bottom bar */}
      <div
        style={{
          marginTop: 'auto',
          borderTop: '1px solid var(--bd)',
          padding: '8px 0 0',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: '0.7rem',
          fontFamily: 'var(--body)',
          color: 'var(--t2)',
        }}
      >
        <span>已选 3 件 · 预估 $100.48</span>
        <button
          style={{
            ...btnBase,
            background: 'linear-gradient(135deg, var(--i5), var(--c4))',
            color: '#fff',
            fontWeight: 500,
            padding: '3px 10px',
          }}
        >
          {'✦'} AI 确认发布
        </button>
      </div>
    </div>
  );
}

function PublishContent() {
  const gridCols = '20px 1fr 56px 60px 44px';
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Filter buttons */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap' }}>
        <FilterBtn label="待发布" count={8} active />
        <FilterBtn label="已发布" count={24} />
        <FilterBtn label="失败" count={2} />
      </div>

      {/* Header */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: gridCols,
          gap: 2,
          padding: '6px 0',
          borderBottom: '1px solid var(--bd)',
          fontSize: '0.7rem',
          color: 'var(--t4)',
          fontFamily: 'var(--body)',
        }}
      >
        <input type="checkbox" style={{ accentColor: 'var(--i4)' }} />
        <span style={cell}>名称</span>
        <span style={cell}>平台</span>
        <span style={cell}>状态</span>
        <span style={cell}>操作</span>
      </div>

      {/* Rows */}
      {publishItems.map((item, i) => (
        <div
          key={i}
          style={{
            display: 'grid',
            gridTemplateColumns: gridCols,
            gap: 2,
            padding: '5px 0',
            borderBottom: i < publishItems.length - 1 ? '1px solid var(--bd2)' : 'none',
            fontSize: '0.7rem',
            fontFamily: 'var(--body)',
            color: 'var(--t2)',
            alignItems: 'center',
          }}
        >
          <input type="checkbox" defaultChecked={item.status === '待发布'} style={{ accentColor: 'var(--i4)' }} />
          <span style={cell}>{item.name}</span>
          <span style={{ ...cell, color: 'var(--t3)' }}>{item.platform}</span>
          <span style={cell}>{statusBadge(item.status)}</span>
          <span style={cell}>
            {item.status === '待发布' && (
              <button style={{ ...btnBase, background: 'var(--i4)', color: '#fff' }}>发布</button>
            )}
            {item.status === '失败' && (
              <button style={{ ...btnBase, background: 'var(--bg)', color: 'var(--t3)', border: '1px solid var(--bd)' }}>
                重试
              </button>
            )}
            {item.status === '已发布' && (
              <span style={{ color: 'var(--t3)' }}>{'—'}</span>
            )}
          </span>
        </div>
      ))}
    </div>
  );
}

function AnalyticsContent() {
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* KPI cards */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 10 }}>
        {analyticsKpis.map((kpi) => (
          <div
            key={kpi.label}
            style={{
              background: 'var(--bg)',
              borderRadius: 6,
              padding: '10px 8px',
              border: '1px solid var(--bd)',
            }}
          >
            <div
              style={{
                fontFamily: 'var(--body)',
                fontSize: '0.65rem',
                color: 'var(--t3)',
                marginBottom: 4,
              }}
            >
              {kpi.label}
            </div>
            <div
              style={{
                fontFamily: 'var(--ds)',
                fontSize: '1rem',
                fontWeight: 600,
                color: 'var(--t1)',
                lineHeight: 1.3,
              }}
            >
              {kpi.value}
            </div>
            <div
              style={{
                fontFamily: 'var(--body)',
                fontSize: '0.65rem',
                color: 'var(--g4)',
                marginTop: 2,
              }}
            >
              {kpi.change}
            </div>
          </div>
        ))}
      </div>

      {/* Date filters */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
        <FilterBtn label="近7天" active />
        <FilterBtn label="近30天" />
        <FilterBtn label="自定义" />
      </div>

      {/* Table header */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '50px 1fr 44px 56px',
          gap: 2,
          padding: '6px 0',
          borderBottom: '1px solid var(--bd)',
          fontSize: '0.7rem',
          color: 'var(--t4)',
          fontFamily: 'var(--body)',
        }}
      >
        <span style={cell}>日期</span>
        <span style={cell}>销售额</span>
        <span style={cell}>订单</span>
        <span style={cell}>平台</span>
      </div>

      {/* Rows */}
      {analyticsRows.map((row, i) => (
        <div
          key={i}
          style={{
            display: 'grid',
            gridTemplateColumns: '50px 1fr 44px 56px',
            gap: 2,
            padding: '5px 0',
            borderBottom: i < analyticsRows.length - 1 ? '1px solid var(--bd2)' : 'none',
            fontSize: '0.7rem',
            fontFamily: 'var(--body)',
            color: 'var(--t2)',
            alignItems: 'center',
          }}
        >
          <span style={{ ...cell, color: 'var(--t3)' }}>{row.date}</span>
          <span style={{ ...cell, fontWeight: 500 }}>{row.sales}</span>
          <span style={cell}>{row.orders}</span>
          <span style={{ ...cell, color: 'var(--t3)' }}>{row.platform}</span>
        </div>
      ))}
    </div>
  );
}

function PricingContent() {
  const gridCols = '1fr 56px 56px 56px 48px';
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: gridCols,
          gap: 2,
          padding: '6px 0',
          borderBottom: '1px solid var(--bd)',
          fontSize: '0.7rem',
          color: 'var(--t4)',
          fontFamily: 'var(--body)',
        }}
      >
        <span style={cell}>名称</span>
        <span style={cell}>当前价</span>
        <span style={cell}>行业均价</span>
        <span style={{ ...cell, color: 'var(--c4)' }}>建议价</span>
        <span style={cell}>价差</span>
      </div>

      {/* Rows */}
      {priceComparisons.map((p, i) => {
        const diffColor = p.diff < -2 ? 'var(--r4)' : 'var(--y4)';
        return (
          <div
            key={p.name}
            style={{
              display: 'grid',
              gridTemplateColumns: gridCols,
              gap: 2,
              padding: '5px 0',
              borderBottom: i < priceComparisons.length - 1 ? '1px solid var(--bd2)' : 'none',
              fontSize: '0.7rem',
              fontFamily: 'var(--body)',
              color: 'var(--t2)',
              alignItems: 'center',
            }}
          >
            <span style={cell}>{p.name}</span>
            <span style={cell}>${p.current.toFixed(2)}</span>
            <span style={{ ...cell, color: 'var(--t3)' }}>${p.avg.toFixed(2)}</span>
            <span style={{ ...cell, color: 'var(--c4)', fontWeight: 600 }}>${p.suggested.toFixed(2)}</span>
            <span style={{ ...cell, color: diffColor, fontWeight: 600 }}>
              {p.diff > 0 ? '+' : ''}${Math.abs(p.diff).toFixed(2)}
            </span>
          </div>
        );
      })}

      {/* Info bar */}
      <div
        style={{
          marginTop: 'auto',
          borderTop: '1px solid var(--bd)',
          padding: '8px 0 0',
          fontSize: '0.65rem',
          fontFamily: 'var(--body)',
          color: 'var(--t3)',
          display: 'flex',
          gap: 6,
          alignItems: 'center',
        }}
      >
        <span style={{ display: 'inline-block', width: 8, height: 8, borderRadius: '50%', background: 'var(--r4)' }} />
        低于行业均价 $3.01
      </div>
    </div>
  );
}

function AICopyContent() {
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Cards */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {aiOptimizations.map((item, idx) => (
          <div
            key={idx}
            style={{
              background: 'var(--bg)',
              borderRadius: 6,
              border: '1px solid var(--bd)',
              overflow: 'hidden',
            }}
          >
            {/* Card header */}
            <div
              style={{
                fontFamily: 'var(--ds)',
                fontSize: '0.75rem',
                fontWeight: 600,
                color: 'var(--t1)',
                padding: '8px 10px',
                borderBottom: '1px solid var(--bd)',
              }}
            >
              {item.name}
            </div>

            {/* Old title */}
            <div style={{ padding: '6px 10px' }}>
              <div
                style={{
                  fontSize: '0.65rem',
                  color: 'var(--t3)',
                  fontFamily: 'var(--body)',
                  marginBottom: 2,
                }}
              >
                原标题
              </div>
              <div
                style={{
                  fontSize: '0.7rem',
                  color: 'var(--t3)',
                  fontFamily: 'var(--body)',
                  textDecoration: 'line-through',
                  opacity: 0.6,
                }}
              >
                {item.old}
              </div>
            </div>

            {/* New title */}
            <div style={{ padding: '0 10px 8px' }}>
              <div
                style={{
                  fontSize: '0.65rem',
                  color: 'var(--c4)',
                  fontFamily: 'var(--body)',
                  marginBottom: 2,
                }}
              >
                优化后
              </div>
              <div
                style={{
                  fontSize: '0.7rem',
                  color: 'var(--c4)',
                  fontWeight: 700,
                  fontFamily: 'var(--body)',
                }}
              >
                {item.new}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Bottom bar */}
      <div
        style={{
          marginTop: 'auto',
          borderTop: '1px solid var(--bd)',
          padding: '8px 0 0',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          fontSize: '0.7rem',
          fontFamily: 'var(--body)',
          color: 'var(--t2)',
        }}
      >
        <span>生成中: 8/15 件</span>
        <button
          style={{
            ...btnBase,
            background: 'linear-gradient(135deg, var(--i5), var(--c4))',
            color: '#fff',
            fontWeight: 500,
            padding: '3px 10px',
          }}
        >
          {'✦'} 全部应用
        </button>
      </div>
    </div>
  );
}

/* ─── Panel component ─── */

export default function ToolPanel() {
  const { toolPanelOpen, activeTool, toggleToolPanel } = useAppStore();

  const activeToolMeta = tools.find((t) => t.id === activeTool) ?? tools[0];

  function renderContent() {
    switch (activeTool) {
      case 'products':
        return <ProductsContent />;
      case 'publish':
        return <PublishContent />;
      case 'analytics':
        return <AnalyticsContent />;
      case 'pricing':
        return <PricingContent />;
      case 'ai-copy':
        return <AICopyContent />;
      default:
        return <ProductsContent />;
    }
  }

  return (
    <aside
      style={{
        width: toolPanelOpen ? 360 : 0,
        overflow: 'hidden',
        borderLeft: toolPanelOpen ? '1px solid var(--bd)' : 'none',
        background: 'var(--s1)',
        display: 'flex',
        flexDirection: 'column',
        flexShrink: 0,
        transition: 'width 0.4s cubic-bezier(0.22, 1, 0.36, 1), border 0.4s ease',
      }}
    >
      {toolPanelOpen && (
        <>
          {/* Panel header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '6px 12px',
              borderBottom: '1px solid var(--bd)',
              flexShrink: 0,
            }}
          >
            <span
              style={{
                fontFamily: 'var(--ds)',
                fontWeight: 600,
                fontSize: '0.8rem',
                display: 'flex',
                alignItems: 'center',
                gap: 6,
              }}
            >
              {activeToolMeta.icon}
              {' '}
              {activeToolMeta.label}
            </span>
            <button
              onClick={toggleToolPanel}
              style={{
                background: 'transparent',
                border: '1px solid var(--bd)',
                borderRadius: 4,
                color: 'var(--t3)',
                cursor: 'pointer',
                fontSize: '0.75rem',
                padding: '2px 6px',
                fontFamily: 'var(--body)',
                lineHeight: '1.6',
              }}
            >
              {'✕'}
            </button>
          </div>

          {/* Panel body */}
          <div
            style={{
              flex: '1 1 0',
              overflow: 'auto',
              padding: 12,
              display: 'flex',
              flexDirection: 'column',
              fontFamily: 'var(--body)',
            }}
          >
            {renderContent()}
          </div>
        </>
      )}
    </aside>
  );
}
