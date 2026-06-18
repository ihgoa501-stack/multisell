<template>
  <div>
    <n-page-header subtitle="运营数据总览与关键指标">
      <template #title>📊 运营驾驶舱</template>
    </n-page-header>

    <!-- ═══════ KPI 卡片行 ═══════ -->
    <n-grid :cols="6" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="总收入" :value="fmt(stats.finance?.total_revenue)">
            <template #suffix>元</template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">利润 {{ fmt(stats.finance?.total_profit) }} 元</div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="利润率" :value="stats.finance?.profit_margin || 0">
            <template #suffix>%</template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">
            <n-progress :height="6" :percentage="Math.min(Math.abs(stats.finance?.profit_margin || 0), 100)" :color="(stats.finance?.profit_margin || 0) >= 0 ? '#18a058' : '#d03050'" />
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="订单总数" :value="stats.orders?.total || 0">
            <template #suffix>单</template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">已支付 {{ stats.orders?.paid || 0 }}</div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="商品 / SKU" :value="stats.products?.total || 0">
            <template #suffix>/ {{ stats.products?.skus || 0 }}</template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">
            <n-tag size="tiny" type="success" style="margin-right:2px;">上架{{ stats.products?.on_shelf||0 }}</n-tag>
            <n-tag size="tiny" type="warning">草稿{{ stats.products?.draft||0 }}</n-tag>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="库存健康" :value="`${stats.inventory?.health_pct || 100}%`">
            <template #suffix></template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">
            <n-progress :height="6" :percentage="stats.inventory?.health_pct || 100" :color="(stats.inventory?.health_pct || 100) > 70 ? '#18a058' : '#d03050'" />
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="true" size="small">
          <n-statistic label="结算净收入" :value="fmt(stats.settlements?.net_revenue)">
            <template #suffix>元</template>
          </n-statistic>
          <div style="font-size:12px;color:#888;">已对账 {{ stats.settlements?.reconciled || 0 }}/{{ stats.settlements?.total || 0 }}</div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- ═══════ 第二行：订单趋势 + 快捷操作 ═══════ -->
    <n-grid :cols="3" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item :span="2">
        <n-card title="📈 近30天订单趋势" :bordered="true" size="small">
          <n-empty v-if="!orderTrend.length" description="暂无订单数据" />
          <div v-else style="display:flex;align-items:flex-end;gap:2px;height:130px;padding:8px 0;">
            <div v-for="d in orderTrend" :key="d.date" style="flex:1;display:flex;flex-direction:column;align-items:center;">
              <span style="font-size:10px;color:#888;margin-bottom:2px;">{{ d.orders }}</span>
              <div :style="{height: Math.max(d.barH, 2) + 'px', width: '100%', background: '#18a058', borderRadius: '2px 2px 0 0', opacity: 0.8}"></div>
              <span style="font-size:9px;color:#bbb;margin-top:2px;writing-mode:vertical-lr;overflow:hidden;text-overflow:ellipsis;max-height:1.2em;">{{ d.label }}</span>
            </div>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="⚡ 快捷操作" :bordered="true" size="small">
          <n-space vertical>
            <n-button size="small" type="primary" block @click="router.push('/products/create')">＋ 新增商品</n-button>
            <n-button size="small" block @click="router.push('/order-import')">📥 导入订单</n-button>
            <n-button size="small" block @click="router.push('/settlements')">💰 结算对账</n-button>
            <n-button size="small" block @click="router.push('/finance')">📊 利润分析</n-button>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- ═══════ 第三行：热销排行 + 订单分布 ═══════ -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="🏆 热销商品 Top 10" :bordered="true" size="small">
          <n-empty v-if="!stats.top_products?.length" description="暂无销售数据" />
          <n-data-table v-else :columns="topProductColumns" :data="topProductsData" :bordered="false" :single-line="true" size="tiny" :max-height="280" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="📋 订单状态分布" :bordered="true" size="small">
          <n-empty v-if="!orderStatusKeys.length" description="暂无订单" />
          <div v-else style="display:flex;flex-direction:column;gap:8px;padding:4px 0;">
            <div v-for="s in orderStatusKeys" :key="s.key" style="display:flex;align-items:center;gap:8px;">
              <n-tag size="tiny" :type="statusColor(s.key)" style="width:60px;text-align:center;">{{ s.label }}</n-tag>
              <div style="flex:1;height:20px;background:#f0f0f0;border-radius:4px;overflow:hidden;">
                <div :style="{width: statusPct(s.key) + '%', height: '100%', background: statusBarColor(s.key), borderRadius: '4px', transition: 'width 0.3s'}"></div>
              </div>
              <span style="width:50px;text-align:right;font-size:13px;">{{ s.count }}</span>
              <span style="width:40px;text-align:right;font-size:11px;color:#999;">{{ statusPct(s.key).toFixed(1) }}%</span>
            </div>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- ═══════ 第四行：平台发布 + 库存健康 ═══════ -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="🌐 平台发布概况" :bordered="true" size="small">
          <n-empty v-if="!stats.platforms?.detail?.length" description="暂无平台发布" />
          <div v-else style="display:flex;flex-direction:column;gap:8px;padding:4px 0;">
            <div v-for="p in stats.platforms?.detail || []" :key="p.code" style="display:flex;align-items:center;gap:8px;">
              <n-tag size="tiny" :color="{color: platformColor(p.code), textColor: '#fff'}" style="width:90px;">{{ p.name }}</n-tag>
              <div style="flex:1;height:20px;background:#f0f0f0;border-radius:4px;overflow:hidden;">
                <div :style="{width: Math.min(p.count / maxPlatformCount * 100, 100) + '%', height: '100%', background: platformColor(p.code), borderRadius: '4px'}"></div>
              </div>
              <span style="width:60px;text-align:right;font-size:13px;">{{ p.count }} 个</span>
            </div>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="📦 库存健康" :bordered="true" size="small">
          <n-empty v-if="!stats.inventory?.total" description="暂无库存数据" />
          <div v-else>
            <div style="display:flex;gap:12px;margin-bottom:12px;">
              <n-card size="small" :bordered="false" style="flex:1;text-align:center;">
                <n-statistic label="正常" :value="stats.inventory?.healthy || 0"><template #suffix>个</template></n-statistic>
              </n-card>
              <n-card size="small" :bordered="false" style="flex:1;text-align:center;">
                <n-statistic label="预警" :value="stats.inventory?.low_stock || 0"><template #suffix>个</template></n-statistic>
              </n-card>
              <n-card size="small" :bordered="false" style="flex:1;text-align:center;">
                <n-statistic label="缺货" :value="stats.inventory?.out_of_stock || 0"><template #suffix>个</template></n-statistic>
              </n-card>
            </div>
            <n-progress :percentage="stats.inventory?.health_pct || 100" :color="(stats.inventory?.health_pct || 100) > 70 ? '#18a058' : '#d03050'" :height="16" />
            <div style="text-align:center;font-size:12px;color:#888;margin-top:4px;">库存健康率 {{ stats.inventory?.health_pct || 100 }}%</div>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- ═══════ 第五行：结算 + 财务 ═══════ -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="💰 结算概览" :bordered="true" size="small">
          <n-empty v-if="!stats.settlements?.total" description="暂无结算数据" />
          <div v-else style="display:flex;flex-direction:column;gap:8px;">
            <div style="display:flex;justify-content:space-around;text-align:center;">
              <div><div style="font-size:22px;font-weight:700;">{{ stats.settlements?.total || 0 }}</div><div style="font-size:12px;color:#888;">总结算单</div></div>
              <div><div style="font-size:22px;font-weight:700;color:#18a058;">{{ stats.settlements?.reconciled || 0 }}</div><div style="font-size:12px;color:#888;">已对账</div></div>
              <div><div style="font-size:22px;font-weight:700;color:#d03050;">{{ stats.settlements?.pending || 0 }}</div><div style="font-size:12px;color:#888;">待对账</div></div>
            </div>
            <n-progress :percentage="reconcilePct" :height="12" color="#18a058" />
            <div style="text-align:center;font-size:12px;color:#888;">对账完成率 {{ reconcilePct.toFixed(1) }}%</div>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="🏦 财务账户" :bordered="true" size="small">
          <n-empty v-if="!stats.accounts?.total" description="暂无财务账户" />
          <div v-else>
            <div style="display:flex;justify-content:space-around;text-align:center;margin-bottom:8px;">
              <div><div style="font-size:28px;font-weight:700;">{{ stats.accounts?.total || 0 }}</div><div style="font-size:12px;color:#888;">账户总数</div></div>
              <div><div style="font-size:28px;font-weight:700;color:#18a058;">¥{{ fmt(stats.finance?.total_balance) }}</div><div style="font-size:12px;color:#888;">总余额</div></div>
            </div>
            <n-button size="small" block @click="router.push('/finance')">查看财务明细 →</n-button>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- ═══════ 第六行：近期动态 ═══════ -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="📝 近期操作" :bordered="true" size="small">
          <n-list v-if="stats.recent_logs?.items?.length" style="max-height:200px;overflow-y:auto;">
            <n-list-item v-for="log in recentLogs" :key="log.id">
              <template #prefix><n-tag size="tiny" :type="log.tagType">{{ log.action }}</n-tag></template>
              <span style="font-size:12px;">{{ log.content || log.module }}</span>
              <template #suffix><span style="font-size:11px;color:#999;white-space:nowrap;">{{ log.time }}</span></template>
            </n-list-item>
          </n-list>
          <n-empty v-else description="暂无操作记录" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="📌 系统概览" :bordered="true" size="small">
          <n-descriptions :column="2" size="small">
            <n-descriptions-item label="品牌数">{{ stats.brands?.total || 0 }}</n-descriptions-item>
            <n-descriptions-item label="供应商">{{ stats.suppliers?.total || 0 }}</n-descriptions-item>
            <n-descriptions-item label="平台数">{{ stats.platforms?.total || 0 }}</n-descriptions-item>
            <n-descriptions-item label="7天操作量">{{ stats.recent_logs?.total_7days || 0 }}</n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-grid-item>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const stats = ref<any>({})

// ── 格式化 ──
function fmt(v: number | undefined | null): string {
  return (v ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

// ── 订单趋势 ──
const orderTrend = computed(() => {
  const raw = stats.value?.orders?.trend_30d || []
  const maxRev = Math.max(...raw.map((d: any) => d.revenue || 0), 1)
  return raw.map((d: any) => ({
    date: d.date,
    orders: d.orders,
    revenue: d.revenue,
    barH: (d.revenue || 0) / maxRev * 120,
    label: d.date ? d.date.slice(5) : '',
  }))
})

// ── 热销商品 ──
const topProductColumns = [
  { title: '#', key: 'rank', width: 36, render: (_: any, i: number) => i + 1 },
  { title: '商品名称', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '销量', key: 'sold_count', width: 60 },
  { title: '销售额', key: 'revenue', width: 100, render: (r: any) => `¥${fmt(r.revenue)}` },
]
const topProductsData = computed(() => stats.value?.top_products || [])

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
    pct: (v as number) / total * 100,
    label: statusMap[k]?.label || k,
    color: statusMap[k]?.color || 'default',
    barColor: statusMap[k]?.barColor || '#bbb',
  }))
})
function statusColor(k: string) { return (statusMap[k]?.color || 'default') as any }
function statusBarColor(k: string) { return statusMap[k]?.barColor || '#bbb' }
function statusPct(k: string) {
  const item = orderStatusKeys.value.find(s => s.key === k)
  return item ? item.pct : 0
}

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

// ── 结算 ──
const reconcilePct = computed(() => {
  const s = stats.value?.settlements
  if (!s?.total) return 0
  return (s.reconciled || 0) / s.total * 100
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

async function fetchStats() {
  try {
    const res: any = await http.get('/dashboard/stats')
    const body = res.data
    stats.value = body?.data || body || {}
  } catch {
    message.error('加载统计数据失败')
  }
}

onMounted(fetchStats)
</script>

<style scoped>
.dashboard-page {
  padding: 0;
}

/* 页面标题 */
:deep(.n-page-header__title) {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-neutral-900, #171717);
}

:deep(.n-page-header__subtitle) {
  color: var(--color-neutral-500, #737373);
  font-size: 14px;
}

/* KPI 卡片 */
:deep(.n-card) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.n-card:hover) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

/* 统计数字高亮 */
:deep(.n-statistic__value) {
  font-weight: 700;
  color: var(--color-neutral-900, #171717);
}

/* 进度条颜色 */
:deep(.n-progress) {
  margin-top: 4px;
}

/* 表格标题 */
:deep(.n-card-header__main) {
  font-weight: 600;
  font-size: 15px;
}

/* 空状态 */
:deep(.n-empty) {
  padding: 40px 0;
}

/* 网格间距 */
:deep(.n-grid) {
  margin-top: 16px;
}

/* 卡片内边距优化 */
:deep(.n-card__content) {
  padding: 16px;
}

/* 标签样式优化 */
:deep(.n-tag--tiny) {
  font-weight: 500;
}

/* 按钮样式优化 */
:deep(.n-button--small-type) {
  font-weight: 500;
}
</style>
