<!-- ⚠️ 此文档引用旧栈（Python/FastAPI/Vue 3），已于 2026-06-30 迁移至 Go/Next.js。仅供参考，不可直接执行。 -->
<template>
  <div class="dashboard-container">
    <!-- ═══════ 页面标题区 ═══════ -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">运营驾驶舱</h1>
        <p class="page-subtitle">数据总览与关键指标 · 更新于 {{ currentTime }}</p>
      </div>
      <div class="page-header-right">
        <n-tag :bordered="false" type="info" size="small">
          <template #icon><n-icon :component="CloudOutline" /></template>
          数据实时同步
        </n-tag>
        <n-button size="small" @click="fetchStats" :loading="loading">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新数据
        </n-button>
      </div>
    </div>

    <!-- ═══════ AI 洞察提示栏 ═══════ -->
    <div class="ai-insight-bar" v-if="aiInsight">
      <n-icon :component="BulbOutline" size="18" />
      <span class="insight-text">{{ aiInsight }}</span>
      <n-button text size="tiny" @click="aiInsight = ''">不再提示</n-button>
    </div>

    <!-- ═══════ KPI 卡片行 ═══════ -->
    <div class="kpi-grid">
      <div class="kpi-card kpi-revenue" @click="router.push('/finance')">
        <div class="kpi-icon">
          <n-icon :component="CashOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">总收入</div>
          <div class="kpi-value">¥{{ fmt(stats.finance?.total_revenue) }}</div>
          <div class="kpi-footer">
            <n-icon :component="TrendingUpOutline" size="14" v-if="(stats.finance?.revenue_growth || 0) >= 0" />
            <n-icon :component="TrendingDownOutline" size="14" v-else />
            <span :class="(stats.finance?.revenue_growth || 0) >= 0 ? 'positive' : 'negative'">
              {{ (stats.finance?.revenue_growth || 0) >= 0 ? '+' : '' }}{{ stats.finance?.revenue_growth || 0 }}%
            </span>
            <span class="kpi-meta">vs 上月</span>
          </div>
        </div>
      </div>

      <div class="kpi-card kpi-profit" @click="router.push('/finance')">
        <div class="kpi-icon">
          <n-icon :component="BarChartOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">利润率</div>
          <div class="kpi-value">{{ (stats.finance?.profit_margin || 0).toFixed(1) }}%</div>
          <div class="kpi-progress">
            <div class="progress-bar" :style="{ width: Math.min(Math.abs(stats.finance?.profit_margin || 0), 100) + '%', background: (stats.finance?.profit_margin || 0) >= 0 ? 'var(--color-success)' : 'var(--color-error)' }"></div>
          </div>
        </div>
      </div>

      <div class="kpi-card kpi-orders" @click="router.push('/orders')">
        <div class="kpi-icon">
          <n-icon :component="CartOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">订单总数</div>
          <div class="kpi-value">{{ stats.orders?.total || 0 }} <span class="kpi-unit">单</span></div>
          <div class="kpi-footer">
            <n-tag size="tiny" type="success" round>已支付 {{ stats.orders?.paid || 0 }}</n-tag>
          </div>
        </div>
      </div>

      <div class="kpi-card kpi-products" @click="router.push('/products')">
        <div class="kpi-icon">
          <n-icon :component="CubeOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">商品 / SKU</div>
          <div class="kpi-value">{{ stats.products?.total || 0 }} <span class="kpi-unit">/ {{ stats.products?.skus || 0 }}</span></div>
          <div class="kpi-footer">
            <n-tag size="tiny" type="success" round>上架 {{ stats.products?.on_shelf || 0 }}</n-tag>
            <n-tag size="tiny" type="warning" round>草稿 {{ stats.products?.draft || 0 }}</n-tag>
          </div>
        </div>
      </div>

      <div class="kpi-card kpi-inventory" @click="router.push('/inventory')">
        <div class="kpi-icon">
          <n-icon :component="ArchiveOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">库存健康率</div>
          <div class="kpi-value">{{ stats.inventory?.health_pct || 100 }}%</div>
          <div class="kpi-progress">
            <div class="progress-bar" :style="{ width: (stats.inventory?.health_pct || 100) + '%', background: (stats.inventory?.health_pct || 100) > 70 ? 'var(--color-success)' : 'var(--color-error)' }"></div>
          </div>
        </div>
      </div>

      <div class="kpi-card kpi-settlement" @click="router.push('/settlements')">
        <div class="kpi-icon">
          <n-icon :component="WalletOutline" size="24" />
        </div>
        <div class="kpi-content">
          <div class="kpi-label">结算净收入</div>
          <div class="kpi-value">¥{{ fmt(stats.settlements?.net_revenue) }}</div>
          <div class="kpi-footer">
            <span class="kpi-meta">已对账 {{ stats.settlements?.reconciled || 0 }}/{{ stats.settlements?.total || 0 }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════ 第二行：订单趋势 + 快捷操作 + AI Agent 状态 ═══════ -->
    <div class="content-grid-3col">
      <div class="chart-card">
        <div class="card-header">
          <h3 class="card-title">📈 近 30 天订单趋势</h3>
          <n-radio-group v-model:value="trendMode" size="small">
            <n-radio-button value="orders">订单量</n-radio-button>
            <n-radio-button value="revenue">销售额</n-radio-button>
          </n-radio-group>
        </div>
        <div class="chart-container">
          <n-empty v-if="!orderTrend.length" description="暂无订单数据" />
          <div v-else class="bar-chart">
            <div v-for="d in orderTrend" :key="d.date" class="bar-item">
              <span class="bar-value">{{ trendMode === 'orders' ? d.orders : '¥' + fmt(d.revenue) }}</span>
              <div
                class="bar-fill"
                :style="{ height: Math.max(d.barH, 4) + 'px' }"
                :class="trendMode === 'revenue' ? 'bar-revenue' : 'bar-orders'"
              ></div>
              <span class="bar-label">{{ d.label }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="action-card">
        <div class="card-header">
          <h3 class="card-title">⚡ 快捷操作</h3>
        </div>
        <div class="action-list">
          <div class="action-item" @click="router.push('/products/create')">
            <div class="action-icon action-icon-primary">
              <n-icon :component="AddOutline" />
            </div>
            <div class="action-text">
              <div class="action-title">新增商品</div>
              <div class="action-desc">创建商品并生成 SKU</div>
            </div>
            <n-icon :component="ChevronForwardOutline" class="action-arrow" />
          </div>

          <div class="action-item" @click="router.push('/order-import')">
            <div class="action-icon action-icon-info">
              <n-icon :component="CloudDownloadOutline" />
            </div>
            <div class="action-text">
              <div class="action-title">导入订单</div>
              <div class="action-desc">从平台同步订单数据</div>
            </div>
            <n-icon :component="ChevronForwardOutline" class="action-arrow" />
          </div>

          <div class="action-item" @click="router.push('/listing-tasks')">
            <div class="action-icon action-icon-warning">
              <n-icon :component="SendOutline" />
            </div>
            <div class="action-text">
              <div class="action-title">批量刊登</div>
              <div class="action-desc">一键发布到多个平台</div>
            </div>
            <n-icon :component="ChevronForwardOutline" class="action-arrow" />
          </div>

          <div class="action-item" @click="router.push('/agent')">
            <div class="action-icon action-icon-success">
              <n-icon :component="SparklesOutline" />
            </div>
            <div class="action-text">
              <div class="action-title">AI Agent</div>
              <div class="action-desc">查看智能助手建议</div>
            </div>
            <n-icon :component="ChevronForwardOutline" class="action-arrow" />
          </div>
        </div>
      </div>

      <div class="agent-card">
        <div class="card-header">
          <h3 class="card-title">🤖 AI Agent 状态</h3>
          <n-tag size="tiny" :type="agentStatus.type" round>{{ agentStatus.label }}</n-tag>
        </div>
        <div class="agent-status-list">
          <div class="agent-item" v-for="agent in agentList" :key="agent.id">
            <div class="agent-avatar" :class="'agent-stage-' + agent.stage">
              {{ agent.name.charAt(0) }}
            </div>
            <div class="agent-info">
              <div class="agent-name">{{ agent.name }}</div>
              <div class="agent-role">{{ agent.role }}</div>
            </div>
            <div class="agent-stage">
              <n-tag size="tiny" :type="agent.stageType" round>{{ agent.stageLabel }}</n-tag>
            </div>
          </div>
        </div>
        <div class="agent-actions">
          <n-button size="small" block @click="router.push('/agent')">
            <template #icon><n-icon :component="SparklesOutline" /></template>
            查看所有 Agent
          </n-button>
        </div>
      </div>
    </div>

    <!-- ═══════ 第三行：热销排行 + 订单分布 ═══════ -->
    <div class="content-grid-2col">
      <div class="table-card">
        <div class="card-header">
          <h3 class="card-title">🏆 热销商品 Top 10</h3>
          <n-button text size="small" @click="router.push('/products')">查看全部 →</n-button>
        </div>
        <n-data-table
          v-if="stats.top_products?.length"
          :columns="topProductColumns"
          :data="stats.top_products"
          :bordered="false"
          :single-line="false"
          size="small"
          :max-height="320"
        />
        <n-empty v-else description="暂无销售数据" />
      </div>

      <div class="distribution-card">
        <div class="card-header">
          <h3 class="card-title">📋 订单状态分布</h3>
        </div>
        <div class="distribution-list" v-if="orderStatusKeys.length">
          <div v-for="s in orderStatusKeys" :key="s.key" class="distribution-item">
            <n-tag size="small" :type="s.color" round style="width: 70px; justify-content: center;">{{ s.label }}</n-tag>
            <div class="distribution-bar-container">
              <div class="distribution-bar" :style="{ width: s.pct + '%', background: s.barColor }"></div>
            </div>
            <span class="distribution-count">{{ s.count }}</span>
            <span class="distribution-pct">{{ s.pct.toFixed(1) }}%</span>
          </div>
        </div>
        <n-empty v-else description="暂无订单" />
      </div>
    </div>

    <!-- ═══════ 第四行：平台发布 + 库存健康 ═══════ -->
    <div class="content-grid-2col">
      <div class="platform-card">
        <div class="card-header">
          <h3 class="card-title">🌐 平台发布概况</h3>
          <n-button text size="small" @click="router.push('/listings')">管理平台 →</n-button>
        </div>
        <div class="platform-list" v-if="stats.platforms?.detail?.length">
          <div v-for="p in stats.platforms.detail" :key="p.code" class="platform-item">
            <div class="platform-badge" :style="{ background: platformColor(p.code) }">{{ p.code.toUpperCase() }}</div>
            <span class="platform-name">{{ p.name }}</span>
            <div class="platform-bar-container">
              <div class="platform-bar" :style="{ width: Math.min(p.count / maxPlatformCount * 100, 100) + '%', background: platformColor(p.code) }"></div>
            </div>
            <span class="platform-count">{{ p.count }} 个</span>
          </div>
        </div>
        <n-empty v-else description="暂无平台发布" />
      </div>

      <div class="inventory-card">
        <div class="card-header">
          <h3 class="card-title">📦 库存健康</h3>
          <n-button text size="small" @click="router.push('/inventory')">查看详情 →</n-button>
        </div>
        <div v-if="stats.inventory?.total" class="inventory-stats">
          <div class="inventory-stat-item inventory-healthy">
            <div class="inventory-stat-value">{{ stats.inventory.healthy || 0 }}</div>
            <div class="inventory-stat-label">正常</div>
          </div>
          <div class="inventory-stat-item inventory-warning">
            <div class="inventory-stat-value">{{ stats.inventory.low_stock || 0 }}</div>
            <div class="inventory-stat-label">预警</div>
          </div>
          <div class="inventory-stat-item inventory-danger">
            <div class="inventory-stat-value">{{ stats.inventory.out_of_stock || 0 }}</div>
            <div class="inventory-stat-label">缺货</div>
          </div>
        </div>
        <div v-if="stats.inventory?.total" class="inventory-progress">
          <div class="inventory-progress-bar">
            <div class="inventory-progress-fill" :style="{ width: (stats.inventory.health_pct || 100) + '%' }"></div>
          </div>
          <div class="inventory-progress-label">库存健康率 {{ stats.inventory.health_pct || 100 }}%</div>
        </div>
        <n-empty v-else description="暂无库存数据" />
      </div>
    </div>

    <!-- ═══════ 第五行：近期动态 + 系统概览 ═══════ -->
    <div class="content-grid-2col">
      <div class="activity-card">
        <div class="card-header">
          <h3 class="card-title">📝 近期操作</h3>
          <n-button text size="small" @click="router.push('/operation-logs')">查看全部 →</n-button>
        </div>
        <div class="activity-list" v-if="recentLogs.length">
          <div v-for="log in recentLogs" :key="log.id" class="activity-item">
            <div class="activity-tag">
              <n-tag size="tiny" :type="log.tagType" round>{{ log.action }}</n-tag>
            </div>
            <div class="activity-content">{{ log.content || log.module }}</div>
            <div class="activity-time">{{ log.time }}</div>
          </div>
        </div>
        <n-empty v-else description="暂无操作记录" />
      </div>

      <div class="system-card">
        <div class="card-header">
          <h3 class="card-title">📌 系统概览</h3>
        </div>
        <div class="system-grid">
          <div class="system-item">
            <div class="system-value">{{ stats.brands?.total || 0 }}</div>
            <div class="system-label">品牌数</div>
          </div>
          <div class="system-item">
            <div class="system-value">{{ stats.suppliers?.total || 0 }}</div>
            <div class="system-label">供应商</div>
          </div>
          <div class="system-item">
            <div class="system-value">{{ stats.platforms?.total || 0 }}</div>
            <div class="system-label">平台数</div>
          </div>
          <div class="system-item">
            <div class="system-value">{{ stats.recent_logs?.total_7days || 0 }}</div>
            <div class="system-label">7天操作量</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { Component } from 'vue'
import http from '@/api/http'

// Icons (需要安装 @vicons/ionicons5)
import {
  CashOutline,
  BarChartOutline,
  CartOutline,
  CubeOutline,
  ArchiveOutline,
  WalletOutline,
  TrendingUpOutline,
  TrendingDownOutline,
  AddOutline,
  CloudDownloadOutline,
  SendOutline,
  SparklesOutline,
  ChevronForwardOutline,
  RefreshOutline,
  CloudOutline,
  BulbOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const stats = ref<any>({})
const trendMode = ref<'orders' | 'revenue'>('orders')
const aiInsight = ref('💡 AI 洞察：上周 OZON 平台销售额增长 23%，建议增加该平台广告预算')
const currentTime = ref(new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))

// ── 格式化 ──
function fmt(v: number | undefined | null): string {
  return (v ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// ── 订单趋势 ──
const orderTrend = computed(() => {
  const raw = stats.value?.orders?.trend_30d || []
  const maxVal = Math.max(...raw.map((d: any) => trendMode.value === 'orders' ? d.orders : (d.revenue || 0)), 1)
  return raw.map((d: any) => ({
    date: d.date,
    orders: d.orders,
    revenue: d.revenue,
    barH: ((trendMode.value === 'orders' ? d.orders : (d.revenue || 0)) / maxVal) * 160,
    label: d.date ? d.date.slice(5) : '',
  }))
})

// ── 热销商品 ──
const topProductColumns = [
  { title: '#', key: 'rank', width: 40, render: (_: any, i: number) => i + 1 },
  { title: '商品名称', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '销量', key: 'sold_count', width: 70 },
  { title: '销售额', key: 'revenue', width: 110, render: (r: any) => `¥${fmt(r.revenue)}` },
]

// ── 订单状态 ──
const statusMap: Record<string, { label: string; color: string; barColor: string }> = {
  pending: { label: '待付款', color: 'warning', barColor: '#f0a020' },
  paid: { label: '待发货', color: 'info', barColor: '#2080f0' },
  shipped: { label: '已发货', color: 'primary', barColor: '#2b7f6e' },
  delivered: { label: '已签收', color: 'success', barColor: '#18a058' },
  completed: { label: '已完成', color: 'success', barColor: '#18a058' },
  cancelled: { label: '已取消', color: 'default', barColor: '#bbb' },
}
const orderStatusKeys = computed(() => {
  const dist = stats.value?.orders?.status_distribution || {}
  const total = Object.values(dist).reduce((a: any, b: any) => a + b, 0) || 1
  return Object.entries(dist).map(([k, v]) => ({
    key: k,
    count: v as number,
    pct: ((v as number) / total) * 100,
    label: statusMap[k]?.label || k,
    color: statusMap[k]?.color || 'default',
    barColor: statusMap[k]?.barColor || '#bbb',
  }))
})

// ── 平台颜色 ──
function platformColor(code: string): string {
  const m: Record<string, string> = {
    ozon: '#005bff', shopee: '#ee4d2d', wb: '#cb11ab',
    wildberries: '#cb11ab', aliexpress: '#e62e04', temu: '#e0120c',
  }
  return m[code.toLowerCase()] || '#2080f0'
}
const maxPlatformCount = computed(() => {
  const detail = stats.value?.platforms?.detail || []
  return Math.max(...detail.map((p: any) => p.count), 1)
})

// ── 近期操作 ──
const recentLogs = computed(() => {
  const items = stats.value?.recent_logs?.items || []
  return items.slice(0, 8).map((l: any) => ({
    ...l,
    tagType: l.action === 'create' || l.action === '创建' ? 'success'
      : l.action === 'delete' || l.action === '删除' ? 'error' : 'info',
    time: l.created_at ? l.created_at.slice(0, 16).replace('T', ' ') : '',
  }))
})

// ── AI Agent 状态（模拟数据）────
const agentStatus = computed(() => {
  // 根据实际 agent 数据计算状态
  return { type: 'success' as const, label: '运行正常' }
})

const agentList = ref([
  { id: 1, name: 'G1 总经理', role: '全局决策与资源调配', stage: 3, stageLabel: '半自主', stageType: 'warning' },
  { id: 2, name: 'A5 市场分析师', role: '市场趋势与选品建议', stage: 2, stageLabel: '建议', stageType: 'info' },
  { id: 3, name: 'A1 选品专家', role: '选品策略与优化', stage: 2, stageLabel: '建议', stageType: 'info' },
])

async function fetchStats() {
  loading.value = true
  try {
    const res: any = await http.get('/dashboard/stats')
    const body = res.data
    stats.value = body?.data || body || {}
    currentTime.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch {
    message.error('加载统计数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchStats)
</script>

<style scoped>
/* ═══════ 设计系统 Token 应用 ═══════ */
.dashboard-container {
  padding: 24px;
  max-width: 1440px;
  margin: 0 auto;
  background: #f8fafc;
  min-height: 100vh;
}

/* ═══════ 页面标题 ═══════ */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px 0;
  letter-spacing: -0.02em;
}
.page-subtitle {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
}
.page-header-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* ═══════ AI 洞察提示栏 ═══════ */
.ai-insight-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 1px solid #0ea5e9;
  border-radius: 8px;
  margin-bottom: 20px;
  color: #0c4a6e;
  font-size: 13px;
}
.insight-text {
  flex: 1;
}

/* ═══════ KPI 卡片网格 ═══════ */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.kpi-card {
  background: white;
  border-radius: 12px;
  padding: 18px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid #e2e8f0;
  position: relative;
  overflow: hidden;
}
.kpi-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  border-radius: 12px 12px 0 0;
}
.kpi-revenue::before { background: linear-gradient(90deg, #0ea5e9, #38bdf8); }
.kpi-profit::before { background: linear-gradient(90deg, #10b981, #34d399); }
.kpi-orders::before { background: linear-gradient(90deg, #f59e0b, #fbbf24); }
.kpi-products::before { background: linear-gradient(90deg, #8b5cf6, #a78bfa); }
.kpi-inventory::before { background: linear-gradient(90deg, #ec4899, #f472b6); }
.kpi-settlement::before { background: linear-gradient(90deg, #06b6d4, #22d3ee); }

.kpi-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
.kpi-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  color: white;
}
.kpi-revenue .kpi-icon { background: linear-gradient(135deg, #0ea5e9, #38bdf8); }
.kpi-profit .kpi-icon { background: linear-gradient(135deg, #10b981, #34d399); }
.kpi-orders .kpi-icon { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
.kpi-products .kpi-icon { background: linear-gradient(135deg, #8b5cf6, #a78bfa); }
.kpi-inventory .kpi-icon { background: linear-gradient(135deg, #ec4899, #f472b6); }
.kpi-settlement .kpi-icon { background: linear-gradient(135deg, #06b6d4, #22d3ee); }

.kpi-label {
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 6px;
  font-weight: 500;
}
.kpi-value {
  font-size: 22px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1.2;
}
.kpi-unit {
  font-size: 13px;
  font-weight: 500;
  color: #94a3b8;
}
.kpi-footer {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
}
.kpi-footer .positive { color: #10b981; font-weight: 600; }
.kpi-footer .negative { color: #ef4444; font-weight: 600; }
.kpi-meta { color: #94a3b8; }
.kpi-progress {
  height: 4px;
  background: #f1f5f9;
  border-radius: 2px;
  margin-top: 10px;
  overflow: hidden;
}
.kpi-progress .progress-bar {
  height: 100%;
  border-radius: 2px;
  transition: width 0.6s ease;
}

/* ═══════ 内容网格 ═══════ */
.content-grid-3col {
  display: grid;
  grid-template-columns: 1.5fr 0.8fr 0.7fr;
  gap: 16px;
  margin-bottom: 20px;
}
.content-grid-2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 20px;
}

/* ═══════ 卡片通用样式 ═══════ */
.chart-card,
.action-card,
.agent-card,
.table-card,
.distribution-card,
.platform-card,
.inventory-card,
.activity-card,
.system-card {
  background: white;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.card-title {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

/* ═══════ 柱状图 ═══════ */
.chart-container {
  height: 200px;
  display: flex;
  align-items: flex-end;
}
.bar-chart {
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 180px;
  width: 100%;
  padding: 8px 0;
}
.bar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
}
.bar-value {
  font-size: 9px;
  color: #64748b;
  margin-bottom: 4px;
  white-space: nowrap;
}
.bar-fill {
  width: 100%;
  border-radius: 3px 3px 0 0;
  transition: height 0.4s ease;
  min-height: 4px;
}
.bar-orders { background: linear-gradient(180deg, #0ea5e9, #38bdf8); }
.bar-revenue { background: linear-gradient(180deg, #10b981, #34d399); }
.bar-label {
  font-size: 9px;
  color: #94a3b8;
  margin-top: 4px;
  writing-mode: horizontal-tb;
}

/* ═══════ 快捷操作 ═══════ */
.action-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.action-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  border: 1px solid transparent;
}
.action-item:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}
.action-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}
.action-icon-primary { background: #0ea5e9; }
.action-icon-info { background: #6464fa; }
.action-icon-warning { background: #f59e0b; }
.action-icon-success { background: #10b981; }
.action-text { flex: 1; }
.action-title { font-size: 13px; font-weight: 600; color: #1e293b; }
.action-desc { font-size: 11px; color: #94a3b8; margin-top: 2px; }
.action-arrow { color: #cbd5e1; }

/* ═══════ AI Agent 卡片 ═══════ */
.agent-status-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 14px;
}
.agent-item {
  display: flex;
  align-items: center;
  gap: 10px;
}
.agent-avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: white;
  flex-shrink: 0;
}
.agent-stage-1 { background: linear-gradient(135deg, #0ea5e9, #38bdf8); }  /* 观察 */
.agent-stage-2 { background: linear-gradient(135deg, #6464fa, #818cf8); }  /* 建议 */
.agent-stage-3 { background: linear-gradient(135deg, #f59e0b, #fbbf24); }  /* 半自主 */
.agent-stage-4 { background: linear-gradient(135deg, #10b981, #34d399); }  /* 全自主 */
.agent-info { flex: 1; }
.agent-name { font-size: 13px; font-weight: 600; color: #1e293b; }
.agent-role { font-size: 11px; color: #94a3b8; }

/* ═══════ 表格卡片 ═══════ */
.table-card :deep(.n-data-table) {
  --n-border-color: transparent;
}

/* ═══════ 分布列表 ═══════ */
.distribution-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.distribution-item {
  display: flex;
  align-items: center;
  gap: 10px;
}
.distribution-bar-container {
  flex: 1;
  height: 18px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}
.distribution-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.5s ease;
  min-width: 4px;
}
.distribution-count {
  width: 45px;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}
.distribution-pct {
  width: 55px;
  text-align: right;
  font-size: 11px;
  color: #94a3b8;
}

/* ═══════ 平台列表 ═══════ */
.platform-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.platform-item {
  display: flex;
  align-items: center;
  gap: 10px;
}
.platform-badge {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 10px;
  font-weight: 800;
  flex-shrink: 0;
}
.platform-name {
  width: 90px;
  font-size: 13px;
  color: #1e293b;
  font-weight: 500;
}
.platform-bar-container {
  flex: 1;
  height: 18px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}
.platform-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.5s ease;
}
.platform-count {
  width: 60px;
  text-align: right;
  font-size: 13px;
  color: #1e293b;
}

/* ═══════ 库存健康 ═══════ */
.inventory-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 16px;
}
.inventory-stat-item {
  text-align: center;
  padding: 14px 8px;
  border-radius: 8px;
}
.inventory-healthy { background: #f0fdf4; }
.inventory-warning { background: #fffbeb; }
.inventory-danger { background: #fef2f2; }
.inventory-stat-value {
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 4px;
}
.inventory-healthy .inventory-stat-value { color: #16a34a; }
.inventory-warning .inventory-stat-value { color: #d97706; }
.inventory-danger .inventory-stat-value { color: #dc2626; }
.inventory-stat-label {
  font-size: 12px;
  color: #64748b;
}
.inventory-progress {
  margin-top: 8px;
}
.inventory-progress-bar {
  height: 8px;
  background: #fee2e2;
  border-radius: 4px;
  overflow: hidden;
}
.inventory-progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #34d399);
  border-radius: 4px;
  transition: width 0.6s ease;
}
.inventory-progress-label {
  text-align: center;
  font-size: 12px;
  color: #94a3b8;
  margin-top: 6px;
}

/* ═══════ 近期操作 ═══════ */
.activity-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 280px;
  overflow-y: auto;
}
.activity-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  transition: background 0.15s;
}
.activity-item:hover {
  background: #f8fafc;
}
.activity-content {
  flex: 1;
  font-size: 13px;
  color: #334155;
}
.activity-time {
  font-size: 11px;
  color: #94a3b8;
  white-space: nowrap;
}

/* ═══════ 系统概览 ═══════ */
.system-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.system-item {
  text-align: center;
  padding: 16px 8px;
  background: #f8fafc;
  border-radius: 8px;
}
.system-value {
  font-size: 28px;
  font-weight: 700;
  color: #0ea5e9;
  margin-bottom: 4px;
}
.system-label {
  font-size: 12px;
  color: #94a3b8;
}

/* ═══════ 响应式 ═══════ */
@media (max-width: 1280px) {
  .kpi-grid { grid-template-columns: repeat(3, 1fr); }
  .content-grid-3col { grid-template-columns: 1fr; }
}
@media (max-width: 960px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .content-grid-2col { grid-template-columns: 1fr; }
}
</style>
