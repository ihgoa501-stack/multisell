<template>
  <div>
    <n-page-header subtitle="商品数据多维分析">
      <template #title>📊 数据报表</template>
      <template #extra>
        <n-date-picker
          v-model:formatted-value="timeRange"
          type="datetimerange"
          value-format="yyyy-MM-dd HH:mm:ss"
          clearable
          style="width: 400px;"
          placeholder="选择时间范围"
          @update:formatted-value="onTimeChange"
        />
        <n-button quaternary style="margin-left: 8px;" @click="resetTime">重置</n-button>
      </template>
    </n-page-header>

    <!-- 加载状态 -->
    <n-spin :show="loading">
      <!-- 第一行：商品分布饼图 + 平台发布状态柱状图 -->
      <n-grid :cols="2" :x-gap="16" style="margin-top: 16px;">
        <!-- 商品分布饼图 -->
        <n-grid-item>
          <n-card title="📦 商品分布" :bordered="true" hoverable>
            <div style="display: flex; align-items: center; gap: 24px;">
              <SvgPieChart :data="productStats" :size="200" />
              <div style="flex: 1;">
                <div v-for="item in productStats" :key="item.label" style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px; font-size: 13px;">
                  <span :style="{ width: '10px', height: '10px', borderRadius: '50%', background: item.color, display: 'inline-block' }"></span>
                  <span style="flex: 1;">{{ item.label }}</span>
                  <span style="color: #666;">{{ item.value }}</span>
                  <span style="color: #999;">({{ item.percent }}%)</span>
                </div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <!-- 平台发布状态柱状图 -->
        <n-grid-item>
          <n-card title="🌐 平台发布状态" :bordered="true" hoverable>
            <div v-if="platformStats.length === 0" style="text-align: center; padding: 40px 0; color: #999;">
              暂无平台发布数据
            </div>
            <div v-else v-for="item in platformStats" :key="item.label" style="margin-bottom: 16px;">
              <div style="display: flex; justify-content: space-between; font-size: 13px; margin-bottom: 4px;">
                <span>{{ item.label }}</span>
                <span>{{ item.value }} / {{ item.total }}</span>
              </div>
              <n-progress
                type="line"
                :percentage="item.percent"
                :color="item.color"
                :height="20"
                :border-radius="4"
                indicator-placement="inside"
              >
                {{ item.percent }}%
              </n-progress>
            </div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 第二行：库存健康度环形图 + 综合指标 -->
      <n-grid :cols="2" :x-gap="16" style="margin-top: 16px;">
        <n-grid-item>
          <n-card title="🥨 库存健康度" :bordered="true" hoverable>
            <div style="display: flex; align-items: center; gap: 24px;">
              <SvgDonutChart :data="inventoryHealth" :size="180" :stroke-width="28" />
              <div style="flex: 1;">
                <div v-for="item in inventoryHealth" :key="item.label" style="display: flex; align-items: center; gap: 8px; margin-bottom: 10px; font-size: 13px;">
                  <span :style="{ width: '12px', height: '12px', borderRadius: '3px', background: item.color, display: 'inline-block' }"></span>
                  <span style="flex: 1;">{{ item.label }}</span>
                  <span style="font-weight: 600;">{{ item.value }}</span>
                </div>
                <n-divider style="margin: 12px 0;" />
                <div style="text-align: center; font-size: 12px; color: #999;">
                  库存总计 <strong style="font-size: 20px; color: #333;">{{ totalInventory }}</strong> 个 SKU
                </div>
              </div>
            </div>
          </n-card>
        </n-grid-item>

        <n-grid-item>
          <n-card title="📋 综合指标" :bordered="true" hoverable>
            <n-grid :cols="2" :x-gap="12" :y-gap="12">
              <n-grid-item v-for="(item, idx) in summaryMetrics" :key="idx">
                <n-statistic :label="item.label" :value="item.value">
                  <template #suffix v-if="item.unit">{{ item.unit }}</template>
                </n-statistic>
                <div v-if="item.sub" style="font-size: 12px; color: #999; margin-top: 2px;">
                  {{ item.sub }}
                </div>
              </n-grid-item>
            </n-grid>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, defineComponent, h } from 'vue'
import type { PropType } from 'vue'
import { useMessage } from 'naive-ui'
import http from '@/api/http'

// ========== 类型 ==========
interface PieItem {
  label: string
  value: number
  percent: number
  color: string
}

interface BarItem {
  label: string
  value: number
  total: number
  percent: number
  color: string
}

interface MetricItem {
  label: string
  value: string | number
  unit?: string
  sub?: string
}

// ========== 纯 SVG 饼图组件 — 用 stroke-dasharray 绘制扇形，不引入任何图表库 ==========
const SvgPieChart = defineComponent({
  name: 'SvgPieChart',
  props: {
    data: { type: Array as PropType<PieItem[]>, default: () => [] },
    size: { type: Number, default: 200 },
  },
  setup(props) {
    return () => {
      const cx = props.size / 2
      const cy = props.size / 2
      const r = props.size * 0.45
      const circumference = 2 * Math.PI * r
      let offset = 0
      const slices = props.data.map((item) => {
        const length = (item.percent / 100) * circumference
        const dasharray = `${length} ${circumference - length}`
        const dashoffset = -offset
        offset += length
        return { ...item, dasharray, dashoffset }
      })

      return h('svg', {
        width: props.size,
        height: props.size,
        viewBox: `0 0 ${props.size} ${props.size}`,
      }, [
        h('circle', { cx, cy, r, fill: 'none', stroke: '#f0f0f0', 'stroke-width': r * 0.28 }),
        ...slices.map((item) =>
          h('circle', {
            cx, cy, r, fill: 'none',
            stroke: item.color,
            'stroke-width': r * 0.28,
            'stroke-dasharray': item.dasharray,
            'stroke-dashoffset': item.dashoffset,
            transform: `rotate(-90 ${cx} ${cy})`,
            'stroke-linecap': 'round',
          })
        ),
        h('text', {
          x: cx, y: cy,
          'text-anchor': 'middle',
          'dominant-baseline': 'middle',
          'font-size': '16px',
          'font-weight': 'bold',
          fill: '#333',
        }, `${props.data.reduce((s, i) => s + i.percent, 0) || 0}%`),
      ])
    }
  },
})

// ========== 纯 SVG 环形图（甜甜圈）组件 ==========
const SvgDonutChart = defineComponent({
  name: 'SvgDonutChart',
  props: {
    data: { type: Array as PropType<PieItem[]>, default: () => [] },
    size: { type: Number, default: 180 },
    strokeWidth: { type: Number, default: 28 },
  },
  setup(props) {
    return () => {
      const cx = props.size / 2
      const cy = props.size / 2
      const r = (props.size - props.strokeWidth) / 2
      const circumference = 2 * Math.PI * r
      let offset = 0
      const slices = props.data.map((item) => {
        const length = (item.percent / 100) * circumference
        const dasharray = `${Math.max(length, 1)} ${circumference - length}`
        const dashoffset = -offset
        offset += length
        return { ...item, dasharray, dashoffset }
      })

      return h('svg', {
        width: props.size,
        height: props.size,
        viewBox: `0 0 ${props.size} ${props.size}`,
      }, [
        h('circle', { cx, cy, r, fill: 'none', stroke: '#f0f0f0', 'stroke-width': props.strokeWidth }),
        ...slices.map((item) =>
          h('circle', {
            cx, cy, r, fill: 'none',
            stroke: item.color,
            'stroke-width': props.strokeWidth,
            'stroke-dasharray': item.dasharray,
            'stroke-dashoffset': item.dashoffset,
            transform: `rotate(-90 ${cx} ${cy})`,
            'stroke-linecap': 'round',
          })
        ),
        h('text', {
          x: cx, y: cy - 6,
          'text-anchor': 'middle',
          'dominant-baseline': 'middle',
          'font-size': '22px',
          'font-weight': 'bold',
          fill: '#333',
        }, `${props.data.reduce((s, i) => s + i.value, 0)}`),
        h('text', {
          x: cx, y: cy + 14,
          'text-anchor': 'middle',
          'dominant-baseline': 'middle',
          'font-size': '12px',
          fill: '#999',
        }, 'SKU'),
      ])
    }
  },
})

// ========== 页面状态 ==========
const message = useMessage()
const loading = ref(false)
const timeRange = ref<[string, string] | null>(null)

// ========== 数据 ==========
const productStats = ref<PieItem[]>([])
const platformStats = ref<BarItem[]>([])
const inventoryHealth = ref<PieItem[]>([])
const summaryMetrics = ref<MetricItem[]>([])

// ========== 计算属性 ==========
const totalInventory = computed(() => {
  return inventoryHealth.value.reduce((sum, item) => sum + item.value, 0)
})

// ========== 时间筛选 ==========
function onTimeChange() {
  fetchAllData()
}

function resetTime() {
  timeRange.value = null
  fetchAllData()
}

// ========== 数据请求 ==========
async function fetchAllData() {
  loading.value = true
  const params: any = {}
  if (timeRange.value) {
    params.start_time = timeRange.value[0]
    params.end_time = timeRange.value[1]
  }

  try {
    const [productRes, platformRes, dashboardRes] = await Promise.all([
      http.get('/reports/product-stats', { params }),
      http.get('/reports/platform-stats', { params }),
      http.get('/dashboard/stats', { params }),
    ])

    // 商品分布饼图
    const pData = (productRes as any).data || []
    productStats.value = buildPieData(pData, [
      { key: 'on_shelf', label: '已上架', color: '#18a058' },
      { key: 'draft', label: '草稿', color: '#d03050' },
      { key: 'inactive', label: '未发布', color: '#909399' },
    ])

    // 平台发布状态柱状图
    const ppData = (platformRes as any).data || []
    const platformColors = ['#2080f0', '#18a058', '#d03050', '#f0a020', '#909399']
    platformStats.value = (Array.isArray(ppData) ? ppData : []).map((item: any, idx: number) => ({
      label: item.platform_name || item.name || item.code || `平台${idx + 1}`,
      value: item.published || item.count || 0,
      total: item.total || item.published || item.count || 0,
      percent: calcPercent(item.published || item.count || 0, item.total || item.published || item.count || 0),
      color: platformColors[idx % platformColors.length],
    }))

    // 库存健康度 & 综合指标
    const dData = (dashboardRes as any).data || {}
    const inv = dData.inventory || {}
    buildInventoryAndMetrics(inv, dData)
  } catch (e: any) {
    message.error('加载报表数据失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

function buildPieData(raw: any, mappings: { key: string; label: string; color: string }[]): PieItem[] {
  const items: PieItem[] = []
  let total = 0
  for (const m of mappings) {
    const val = typeof raw === 'object' && raw !== null ? (raw[m.key] ?? 0) : 0
    total += val
    items.push({ label: m.label, value: val, percent: 0, color: m.color })
  }
  for (const item of items) {
    item.percent = total > 0 ? Math.round((item.value / total) * 100) : 0
  }
  return items
}

function calcPercent(value: number, total: number): number {
  return total > 0 ? Math.round((value / total) * 100) : 0
}

function buildInventoryAndMetrics(inv: any, dData: any) {
  const healthy = inv.normal ?? inv.healthy ?? inv.normal_count ?? 0
  const warning = inv.low_stock ?? inv.warning ?? 0
  const danger = inv.out_of_stock ?? inv.danger ?? 0

  inventoryHealth.value = [
    { label: '健康', value: healthy, percent: 0, color: '#18a058' },
    { label: '告警', value: warning, percent: 0, color: '#f0a020' },
    { label: '缺货', value: danger, percent: 0, color: '#d03050' },
  ]
  const total = healthy + warning + danger
  for (const item of inventoryHealth.value) {
    item.percent = total > 0 ? Math.round((item.value / total) * 100) : 0
  }

  const products = dData.products || {}
  const suppliers = dData.suppliers || {}
  const brands = dData.brands || {}
  summaryMetrics.value = [
    { label: '商品总数', value: products.total ?? 0, unit: '个', sub: `上架 ${products.on_shelf ?? 0} / 草稿 ${products.draft ?? 0}` },
    { label: 'SKU 总数', value: dData.skus?.total ?? 0, unit: '个' },
    { label: '供应商', value: suppliers.total ?? 0, unit: '个' },
    { label: '品牌', value: brands.total ?? 0, unit: '个' },
  ]
}

onMounted(fetchAllData)
</script>

<style scoped>
.n-card {
  height: 100%;
}
</style>
