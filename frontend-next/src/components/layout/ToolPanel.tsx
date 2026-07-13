'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useAppStore } from '@/stores/app-store';
import apiClient from '@/lib/api-client';
import {
  ShoppingOutlined,
  SendOutlined,
  BarChartOutlined,
  DollarOutlined,
  BulbOutlined,
} from '@ant-design/icons';

/* ─── API response types ─── */

interface SkuItem {
  id: number;
  product_id: number;
  code: string;
  barcode: string;
  spec_desc: string;
  price: number;
  cost_price: number;
  market_price: number;
  stock: number;
  warning_stock: number;
  status: number;
}

interface DashboardOverview {
  order_total: number;
  order_by_status: Record<string, number>;
  order_revenue: number;
  order_profit: number;
  sku_total: number;
  low_stock_count: number;
  out_of_stock_count: number;
  listing_active_count: number;
  aftersales_pending_count: number;
  exception_open_count: number;
  month_revenue: number;
  month_cost: number;
}

interface ListingTaskItem {
  id: number;
  product_id: number;
  platform_id: number;
  sku_id: number | null;
  source_type: string;
  status: string;
  last_error: string;
  created_at: string;
}

/* ─── Tool definitions ─── */

const tools = [
  { id: 'products', icon: <ShoppingOutlined />, label: '商品管理', badge: '未核验' },
  { id: 'publish', icon: <SendOutlined />, label: '平台发布', badge: '未核验' },
  { id: 'analytics', icon: <BarChartOutlined />, label: '数据分析', badge: undefined },
  { id: 'pricing', icon: <DollarOutlined />, label: '价格监控', badge: undefined },
  { id: 'ai-copy', icon: <BulbOutlined />, label: 'AI 文案', badge: undefined },
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
  if (status === '在售' || status === 'completed') bg = 'var(--g4)';
  else if (status === '低库存' || status === '待发布' || status === 'pending') bg = 'var(--y4)';
  else if (status === '缺货' || status === '失败' || status === 'failed' || status === 'processing') bg = 'var(--r4)';
  else if (status === '已发布' || status === 'online') bg = 'var(--g4)';
  else if (status === 'blocked') bg = 'var(--t4)';
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

function FilterBtn({
  label,
  count,
  active,
  onClick,
}: {
  label: string;
  count?: number;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        ...btnBase,
        background: active ? 'rgba(99,102,241,0.08)' : 'transparent',
        color: active ? 'var(--i4)' : 'var(--t3)',
        border: active ? '1px solid var(--i4)' : '1px solid var(--bd)',
        fontWeight: active ? 600 : 400,
      }}
    >
      {label}
      {count != null ? ` (${count})` : ''}
    </button>
  );
}

/* ─── Loading / Error placeholders ─── */

function LoadingState() {
  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '0.75rem',
        fontFamily: 'var(--body)',
        color: 'var(--t3)',
      }}
    >
      加载中...
    </div>
  );
}

function ErrorState({ message }: { message: string }) {
  return (
    <div
      style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        fontSize: '0.7rem',
        fontFamily: 'var(--body)',
        color: 'var(--r4)',
        padding: '12px',
        textAlign: 'center',
      }}
    >
      {message}
    </div>
  );
}

/* ─── Tool sections ─── */

/* ── Products ── */

/* Uses GET /v1/skus since the Sku model provides stock, price,
   SKU code, and variant description — fields needed by this panel.
   The product-level endpoint (GET /v1/products) returns only product
   metadata (name, dates) without stock or price. */
function ProductsContent() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedProducts, setSelectedProducts] = useState<string[]>([]);

  const { data: pageResult, isLoading, error } = useQuery({
    queryKey: ['toolpanel-products', searchQuery],
    queryFn: () =>
      apiClient.getPage<SkuItem>('/v1/skus', {
        page: '1',
        size: '5',
        ...(searchQuery ? { search: searchQuery } : {}),
      }),
  });

  const products = pageResult?.data ?? [];
  const total = pageResult?.total ?? 0;

  const filteredProducts = searchQuery
    ? products.filter((p) =>
        p.code.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (p.spec_desc && p.spec_desc.toLowerCase().includes(searchQuery.toLowerCase())),
      )
    : products;

  const toggleProduct = (skuCode: string) => {
    setSelectedProducts((prev) =>
      prev.includes(skuCode) ? prev.filter((s) => s !== skuCode) : [...prev, skuCode],
    );
  };

  const toggleAllProducts = () => {
    if (selectedProducts.length === filteredProducts.length) {
      setSelectedProducts([]);
    } else {
      setSelectedProducts(filteredProducts.map((p) => p.code));
    }
  };

  const estimatedTotal = selectedProducts
    .reduce((sum, code) => {
      const p = products.find((pr) => pr.code === code);
      return sum + (p ? p.market_price || p.price : 0);
    }, 0)
    .toFixed(2);

  function stockStatus(sku: SkuItem): string {
    if (sku.stock <= 0) return '缺货';
    if (sku.warning_stock > 0 && sku.stock <= sku.warning_stock) return '低库存';
    return '在售';
  }

  const gridCols = '20px 1fr 60px 44px 56px 56px 52px';

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState message={`加载失败: ${(error as Error).message}`} />;

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
        {'✦'} AI 助理已筛选 {total.toLocaleString()} 件商品，展示 Top {products.length}
      </div>

      {/* Search input */}
      <input
        type="text"
        placeholder="搜索 SKU 名称..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        style={{
          marginBottom: 8,
          padding: '4px 8px',
          border: '1px solid var(--bd)',
          borderRadius: 4,
          background: 'var(--bg)',
          color: 'var(--t1)',
          fontFamily: 'var(--body)',
          fontSize: '0.7rem',
          outline: 'none',
        }}
      />

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
        <input
          type="checkbox"
          checked={
            filteredProducts.length > 0 &&
            selectedProducts.length === filteredProducts.length
          }
          onChange={toggleAllProducts}
          style={{ accentColor: 'var(--i4)' }}
        />
        <span
          style={{ ...cell, cursor: 'pointer', color: 'var(--i4)' }}
          onClick={() => router.push('/products')}
        >
          商品管理
        </span>
        <span style={cell}>SKU</span>
        <span style={cell}>库存</span>
        <span style={cell}>当前价</span>
        <span style={{ ...cell, color: 'var(--c4)' }}>建议价</span>
        <span style={cell}>状态</span>
      </div>

      {/* Rows */}
      {filteredProducts.map((p, i) => {
        const status = stockStatus(p);
        return (
          <div
            key={p.code}
            style={{
              display: 'grid',
              gridTemplateColumns: gridCols,
              gap: 2,
              padding: '5px 0',
              borderBottom:
                i < filteredProducts.length - 1
                  ? '1px solid var(--bd2)'
                  : 'none',
              fontSize: '0.7rem',
              fontFamily: 'var(--body)',
              color: 'var(--t2)',
              alignItems: 'center',
            }}
          >
            <input
              type="checkbox"
              checked={selectedProducts.includes(p.code)}
              onChange={() => toggleProduct(p.code)}
              style={{ accentColor: 'var(--i4)' }}
            />
            <span style={cell}>{p.spec_desc || p.code}</span>
            <span style={{ ...cell, color: 'var(--t3)' }}>{p.code}</span>
            <span
              style={{ ...cell, color: stockColor(p.stock), fontWeight: 600 }}
            >
              {p.stock}
            </span>
            <span style={cell}>${p.price.toFixed(2)}</span>
            <span style={{ ...cell, color: 'var(--c4)', fontWeight: 600 }}>
              ${(p.market_price || p.price).toFixed(2)}
            </span>
            <span style={cell}>{statusBadge(status)}</span>
          </div>
        );
      })}

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
        <span>
          已选 {selectedProducts.length} 件 · 预估 ${estimatedTotal}
        </span>
        <button
          onClick={() => alert('AI 确认发布已触发')}
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

/* ── Publish ── */

function PublishContent() {
  const router = useRouter();
  const [selectedPublish, setSelectedPublish] = useState<string[]>([]);
  const [activePublishFilter, setActivePublishFilter] = useState('pending');

  const { data: pageResult, isLoading, error } = useQuery({
    queryKey: ['toolpanel-publish', activePublishFilter],
    queryFn: () =>
      apiClient.getPage<ListingTaskItem>('/v1/listing-tasks', {
        page: '1',
        size: '10',
        ...(activePublishFilter !== 'all'
          ? { status: activePublishFilter }
          : {}),
      }),
  });

  const tasks = pageResult?.data ?? [];

  const togglePublishItem = (id: string) => {
    setSelectedPublish((prev) =>
      prev.includes(id) ? prev.filter((n) => n !== id) : [...prev, id],
    );
  };

  const toggleAllPublish = () => {
    if (selectedPublish.length === tasks.length) {
      setSelectedPublish([]);
    } else {
      setSelectedPublish(tasks.map((t) => String(t.id)));
    }
  };

  function statusLabel(s: string): string {
    const map: Record<string, string> = {
      pending: '待发布',
      processing: '处理中',
      completed: '已发布',
      failed: '失败',
      blocked: '阻塞',
    };
    return map[s] || s;
  }

  function platformLabel(pid: number): string {
    /* In a full integration this would resolve platform names from
       the platform registry. For now show the numeric ID as a fallback. */
    const map: Record<number, string> = {
      1: 'Shopee',
      2: 'Lazada',
      3: 'Ozon',
    };
    return map[pid] || `平台#${pid}`;
  }

  const gridCols = '20px 1fr 56px 60px 44px';

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState message={`加载失败: ${(error as Error).message}`} />;

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Filter buttons */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap' }}>
        <FilterBtn
          label="全部"
          active={activePublishFilter === 'all'}
          onClick={() => setActivePublishFilter('all')}
        />
        <FilterBtn
          label="待发布"
          active={activePublishFilter === 'pending'}
          onClick={() => setActivePublishFilter('pending')}
        />
        <FilterBtn
          label="处理中"
          active={activePublishFilter === 'processing'}
          onClick={() => setActivePublishFilter('processing')}
        />
        <FilterBtn
          label="失败"
          active={activePublishFilter === 'failed'}
          onClick={() => setActivePublishFilter('failed')}
        />
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
        <input
          type="checkbox"
          checked={tasks.length > 0 && selectedPublish.length === tasks.length}
          onChange={toggleAllPublish}
          style={{ accentColor: 'var(--i4)' }}
        />
        <span style={cell}>商品</span>
        <span style={cell}>平台</span>
        <span style={cell}>状态</span>
        <span style={cell}>操作</span>
      </div>

      {/* Rows */}
      {tasks.map((item, i) => (
        <div
          key={item.id}
          onClick={() => router.push('/products')}
          style={{
            display: 'grid',
            gridTemplateColumns: gridCols,
            gap: 2,
            padding: '5px 0',
            borderBottom:
              i < tasks.length - 1 ? '1px solid var(--bd2)' : 'none',
            fontSize: '0.7rem',
            fontFamily: 'var(--body)',
            color: 'var(--t2)',
            alignItems: 'center',
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={selectedPublish.includes(String(item.id))}
            onChange={() => togglePublishItem(String(item.id))}
            onClick={(e) => e.stopPropagation()}
            style={{ accentColor: 'var(--i4)' }}
          />
          <span style={cell}>
            SKU #{item.product_id}
            {item.source_type ? ` (${item.source_type})` : ''}
          </span>
          <span style={{ ...cell, color: 'var(--t3)' }}>
            {platformLabel(item.platform_id)}
          </span>
          <span style={cell}>{statusBadge(statusLabel(item.status))}</span>
          <span style={cell}>
            {item.status === 'pending' && (
              <button
                style={{ ...btnBase, background: 'var(--i4)', color: '#fff' }}
              >
                发布
              </button>
            )}
            {item.status === 'failed' && (
              <button
                style={{
                  ...btnBase,
                  background: 'var(--bg)',
                  color: 'var(--t3)',
                  border: '1px solid var(--bd)',
                }}
              >
                重试
              </button>
            )}
            {item.status === 'completed' && (
              <span style={{ color: 'var(--t3)' }}>{'—'}</span>
            )}
            {item.status === 'processing' && (
              <span style={{ color: 'var(--t3)', fontSize: '0.65rem' }}>
                处理中
              </span>
            )}
          </span>
        </div>
      ))}

      {/* Bottom bar */}
      <div
        style={{
          marginTop: 'auto',
          borderTop: '1px solid var(--bd)',
          padding: '8px 0 0',
          display: 'flex',
          justifyContent: 'flex-end',
          alignItems: 'center',
          fontSize: '0.7rem',
          fontFamily: 'var(--body)',
          color: 'var(--t2)',
        }}
      >
        <button
          onClick={() => alert('AI 批量发布已确认')}
          style={{
            ...btnBase,
            background: 'linear-gradient(135deg, var(--i5), var(--c4))',
            color: '#fff',
            fontWeight: 500,
            padding: '3px 10px',
          }}
        >
          {'✦'} AI 批量发布
        </button>
      </div>
    </div>
  );
}

/* ── Analytics ── */

function AnalyticsContent() {
  const [activeDateFilter, setActiveDateFilter] = useState('近7天');

  const { data: result, isLoading, error } = useQuery({
    queryKey: ['toolpanel-analytics', activeDateFilter],
    queryFn: () => apiClient.get<DashboardOverview>('/v1/dashboard/overview'),
  });

  const overview = result?.data;

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState message={`加载失败: ${(error as Error).message}`} />;

  const kpis = overview
    ? [
        {
          label: '总销售额',
          value: `$${(overview.order_revenue || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
          change: `SKU 总数 ${overview.sku_total || 0}`,
        },
        {
          label: '总利润',
          value: `$${(overview.order_profit || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
          change: `订单总数 ${overview.order_total || 0}`,
        },
        {
          label: '本月营收',
          value: `$${(overview.month_revenue || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
          change: overview.listing_active_count
            ? `在线 Listing ${overview.listing_active_count}`
            : undefined,
        },
        {
          label: '本月成本',
          value: `$${(overview.month_cost || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
          change: overview.exception_open_count
            ? `异常 ${overview.exception_open_count}`
            : undefined,
        },
      ]
    : [];

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* KPI cards */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 8,
          marginBottom: 10,
        }}
      >
        {kpis.map((kpi) => (
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
            {kpi.change && (
              <div
                style={{
                  fontFamily: 'var(--body)',
                  fontSize: '0.65rem',
                  color: 'var(--t3)',
                  marginTop: 2,
                }}
              >
                {kpi.change}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Date filters */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8 }}>
        <FilterBtn
          label="近7天"
          active={activeDateFilter === '近7天'}
          onClick={() => setActiveDateFilter('近7天')}
        />
        <FilterBtn
          label="近30天"
          active={activeDateFilter === '近30天'}
          onClick={() => setActiveDateFilter('近30天')}
        />
        <FilterBtn
          label="自定义"
          active={activeDateFilter === '自定义'}
          onClick={() => setActiveDateFilter('自定义')}
        />
      </div>

      {/* Summary rows */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 8,
        }}
      >
        <div
          style={{
            background: 'var(--bg)',
            borderRadius: 6,
            padding: '8px',
            border: '1px solid var(--bd)',
            fontFamily: 'var(--body)',
            fontSize: '0.7rem',
            color: 'var(--t2)',
          }}
        >
          <span style={{ color: 'var(--t3)' }}>低库存 SKU</span>
          <div style={{ fontWeight: 600, fontSize: '0.85rem', marginTop: 4 }}>
            {overview?.low_stock_count ?? '--'}
          </div>
        </div>
        <div
          style={{
            background: 'var(--bg)',
            borderRadius: 6,
            padding: '8px',
            border: '1px solid var(--bd)',
            fontFamily: 'var(--body)',
            fontSize: '0.7rem',
            color: 'var(--t2)',
          }}
        >
          <span style={{ color: 'var(--t3)' }}>缺货 SKU</span>
          <div
            style={{
              fontWeight: 600,
              fontSize: '0.85rem',
              marginTop: 4,
              color:
                (overview?.out_of_stock_count ?? 0) > 0
                  ? 'var(--r4)'
                  : 'var(--t2)',
            }}
          >
            {overview?.out_of_stock_count ?? '--'}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Pricing ── */

function PricingContent() {
  const [activePricingFilter, setActivePricingFilter] = useState('全部');

  const priceComparisons = [
    { name: '蓝牙耳机', current: 25.99, avg: 27.50, suggested: 27.50, diff: -1.51 },
    { name: '充电宝', current: 18.50, avg: 22.00, suggested: 19.99, diff: -3.50 },
    { name: '运动手环', current: 39.99, avg: 44.00, suggested: 42.00, diff: -4.01 },
  ];

  const gridCols = '1fr 56px 56px 56px 48px';
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      {/* Filter buttons */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 8, flexWrap: 'wrap' }}>
        <FilterBtn
          label="全部"
          active={activePricingFilter === '全部'}
          onClick={() => setActivePricingFilter('全部')}
        />
        <FilterBtn
          label="低于均价"
          active={activePricingFilter === '低于均价'}
          onClick={() => setActivePricingFilter('低于均价')}
        />
        <FilterBtn
          label="需调价"
          active={activePricingFilter === '需调价'}
          onClick={() => setActivePricingFilter('需调价')}
        />
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
              borderBottom:
                i < priceComparisons.length - 1
                  ? '1px solid var(--bd2)'
                  : 'none',
              fontSize: '0.7rem',
              fontFamily: 'var(--body)',
              color: 'var(--t2)',
              alignItems: 'center',
            }}
          >
            <span style={cell}>{p.name}</span>
            <span style={cell}>${p.current.toFixed(2)}</span>
            <span style={{ ...cell, color: 'var(--t3)' }}>
              ${p.avg.toFixed(2)}
            </span>
            <span style={{ ...cell, color: 'var(--c4)', fontWeight: 600 }}>
              ${p.suggested.toFixed(2)}
            </span>
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
        <span
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: 'var(--r4)',
          }}
        />
        低于行业均价 $3.01
      </div>
    </div>
  );
}

/* ── AI Copy ── */

function AICopyContent() {
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
          onClick={() => alert('AI 文案已全部应用')}
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
        transition:
          'width 0.4s cubic-bezier(0.22, 1, 0.36, 1), border 0.4s ease',
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
