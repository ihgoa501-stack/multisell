<template>
  <div class="detail-container">
    <!-- ═════ 页面标题区 ═════ -->
    <div class="page-header">
      <div class="page-header-left">
        <n-button text @click="router.back()" class="back-button">
          <template #icon><n-icon :component="ArrowBackOutline" /></template>
          返回
        </n-button>
        <div class="product-title-row">
          <div class="product-avatar">
            <n-icon :component="CubeOutline" size="24" />
          </div>
          <div>
            <h1 class="product-name">{{ detail.product?.name || '加载中...' }}</h1>
            <div class="product-meta">
              <n-tag size="small" :type="statusType" round>{{ detail.product?.status_name || '-' }}</n-tag>
              <span class="meta-item">ID: {{ productId }}</span>
              <span class="meta-item">分类: {{ detail.product?.category_name || '-' }}</span>
              <span class="meta-item">品牌: {{ detail.product?.brand_name || '-' }}</span>
            </div>
          </div>
        </div>
      </div>
      <div class="page-header-right">
        <n-button @click="handleDuplicate">
          <template #icon><n-icon :component="CopyOutline" /></template>
          复制商品
        </n-button>
        <n-button type="primary" @click="router.push(`/products/${productId}/edit`)">
          <template #icon><n-icon :component="CreateOutline" /></template>
          编辑
        </n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <!-- ═════ AI 洞察栏 ═════ -->
      <div class="ai-insight-bar" v-if="aiSuggestion">
        <n-icon :component="BulbOutline" size="18" />
        <div class="insight-content">
          <div class="insight-title">AI 优化建议</div>
          <div class="insight-text">{{ aiSuggestion }}</div>
        </div>
        <n-button size="small" type="primary" ghost>查看详情</n-button>
      </div>

      <!-- ═════ 核心信息卡片 ═════ -->
      <div class="info-grid">
        <!-- 基本信息 -->
        <div class="info-card">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="InformationCircleOutline" size="16" />
              基本信息
            </h3>
          </div>
          <div class="info-list">
            <div class="info-item">
              <span class="info-label">商品名称</span>
              <span class="info-value">{{ detail.product?.name || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">副标题</span>
              <span class="info-value text-secondary">{{ detail.product?.subtitle || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">分类</span>
              <span class="info-value">{{ detail.product?.category_name || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">品牌</span>
              <span class="info-value">{{ detail.product?.brand_name || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">单位</span>
              <span class="info-value">{{ detail.product?.unit || '-' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">创建时间</span>
              <span class="info-value text-secondary">{{ detail.product?.created_at || '-' }}</span>
            </div>
          </div>
        </div>

        <!-- 物流信息 -->
        <div class="info-card">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="CarOutline" size="16" />
              物流信息
            </h3>
            <n-tag v-if="detail.product?.logistics_status === 'complete'" type="success" size="small" round>可计算运费</n-tag>
            <n-tag v-else type="warning" size="small" round>缺物流数据</n-tag>
          </div>
          <div class="info-list">
            <div class="info-item">
              <span class="info-label">货品类型</span>
              <n-tag :color="cargoTypeColor(detail.product?.cargo_type)" size="small" round>
                {{ cargoTypeLabel(detail.product?.cargo_type) }}
              </n-tag>
            </div>
            <div class="info-item">
              <span class="info-label">商品尺寸</span>
              <span class="info-value">{{ formatDimensions(detail.product?.product_length_cm, detail.product?.product_width_cm, detail.product?.product_height_cm) }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">商品重量</span>
              <span class="info-value">{{ formatWeight(detail.product?.product_weight_kg) }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">包装尺寸</span>
              <span class="info-value">{{ formatDimensions(detail.product?.package_length_cm, detail.product?.package_width_cm, detail.product?.package_height_cm) }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">包装重量</span>
              <span class="info-value">{{ formatWeight(detail.product?.package_weight_kg) }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">体积重</span>
              <span class="info-value" :class="detail.product?.package_volume_weight_kg > 2 ? 'text-warning' : ''">
                {{ detail.product?.package_volume_weight_kg != null ? `${detail.product.package_volume_weight_kg} kg` : '-' }}
              </span>
            </div>
          </div>
          <div class="card-actions">
            <n-button
              v-if="detail.product?.logistics_status === 'complete'"
              size="small"
              type="info"
              @click="goShippingCalculator"
            >
              <template #icon><n-icon :component="CalculatorOutline" /></template>
              试算运费
            </n-button>
            <n-button
              v-else
              size="small"
              type="warning"
              @click="router.push(`/products/${productId}/edit`)"
            >
              <template #icon><n-icon :component="WarningOutline" /></template>
              补齐物流数据
            </n-button>
          </div>
          <div v-if="missingFields.length > 0" class="missing-fields">
            <n-icon :component="AlertCircleOutline" size="14" />
            <span>缺失字段: {{ missingFields.join('、') }}</span>
          </div>
        </div>

        <!-- 库存概况 -->
        <div class="info-card">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="ArchiveOutline" size="16" />
              库存概况
            </h3>
          </div>
          <div v-if="detail.inventory?.length" class="inventory-summary">
            <div class="inventory-stat">
              <div class="inventory-stat-value">{{ totalStock }}</div>
              <div class="inventory-stat-label">总库存</div>
            </div>
            <div class="inventory-stat">
              <div class="inventory-stat-value text-warning">{{ lowStockCount }}</div>
              <div class="inventory-stat-label">预警</div>
            </div>
            <div class="inventory-stat">
              <div class="inventory-stat-value text-danger">{{ outOfStockCount }}</div>
              <div class="inventory-stat-label">缺货</div>
            </div>
          </div>
          <n-empty v-else description="暂无库存记录" />
          <div v-if="detail.inventory?.length" class="inventory-list">
            <div v-for="inv in detail.inventory" :key="inv.id" class="inventory-item">
              <div class="inventory-warehouse">{{ inv.warehouse }}</div>
              <div class="inventory-quantity" :class="inv.quantity <= inv.safety_stock ? 'text-danger' : ''">
                {{ inv.quantity }}
              </div>
              <div class="inventory-safety">安全库存: {{ inv.safety_stock }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- ═════ SKU 列表 ═════ -->
      <div class="section-card">
        <div class="card-header">
          <h3 class="card-title">
            <n-icon :component="GridOutline" size="16" />
            SKU 列表
            <n-tag size="small" :bordered="false" type="info">{{ detail.skus?.length || 0 }} 个</n-tag>
          </h3>
        </div>
        <n-data-table
          v-if="detail.skus?.length"
          :columns="skuColumns"
          :data="detail.skus"
          :bordered="false"
          :single-line="false"
          size="small"
        />
        <n-empty v-else description="暂无 SKU" />
      </div>

      <!-- ═════ 平台发布状态 ═════ -->
      <div class="section-card">
        <div class="card-header">
          <h3 class="card-title">
            <n-icon :component="GlobeOutline" size="16" />
            平台发布状态
            <n-tag size="small" :bordered="false" type="success">{{ publishedCount }} 个已发布</n-tag>
          </h3>
          <n-button size="small" type="primary" @click="router.push(`/listing-tasks?product_id=${productId}`)">
            <template #icon><n-icon :component="SendOutline" /></template>
            去发布
          </n-button>
        </div>
        <div v-if="detail.listings?.length" class="platform-list">
          <div v-for="l in detail.listings" :key="l.id" class="platform-item">
            <div class="platform-badge" :style="{ background: platformColor(l.platform_code) }">
              {{ l.platform_code.toUpperCase() }}
            </div>
            <div class="platform-info">
              <div class="platform-name">{{ l.platform_name }}</div>
              <div class="platform-id">{{ l.platform_product_id }}</div>
            </div>
            <n-tag :type="l.status === 'synced' ? 'success' : 'warning'" size="small" round>
              {{ l.status === 'synced' ? '已发布' : l.status }}
            </n-tag>
          </div>
        </div>
        <n-empty v-else description="暂未发布到任何平台">
          <template #extra>
            <n-button size="small" type="primary" @click="router.push(`/listing-tasks?product_id=${productId}`)">去发布</n-button>
          </template>
        </n-empty>
      </div>

      <!-- ═════ 供应商信息 ═════ -->
      <div class="section-card">
        <div class="card-header">
          <h3 class="card-title">
            <n-icon :component="BusinessOutline" size="16" />
            供应商信息
            <n-tag size="small" :bordered="false">{{ detail.suppliers?.length || 0 }} 个</n-tag>
          </h3>
        </div>
        <div v-if="detail.suppliers?.length" class="supplier-list">
          <div v-for="sup in detail.suppliers" :key="sup.id" class="supplier-item">
            <div class="supplier-avatar">
              <n-icon :component="BusinessOutline" />
            </div>
            <div class="supplier-info">
              <div class="supplier-name">{{ sup.supplier_name }}</div>
              <div v-if="sup.supply_price" class="supplier-price">供货价: ¥{{ sup.supply_price }}</div>
            </div>
            <n-button text size="small">查看详情 →</n-button>
          </div>
        </div>
        <n-empty v-else description="暂无供应商" />
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { Component } from 'vue'
import http from '@/api/http'

// Icons
import {
  ArrowBackOutline,
  CubeOutline,
  CopyOutline,
  CreateOutline,
  InformationCircleOutline,
  CarOutline,
  CalculatorOutline,
  WarningOutline,
  AlertCircleOutline,
  ArchiveOutline,
  GridOutline,
  GlobeOutline,
  SendOutline,
  BusinessOutline,
  BulbOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const productId = Number(route.params.id)
const loading = ref(false)
const detail = ref<any>({})
const aiSuggestion = ref('建议优化商品包装尺寸，当前体积重偏高，可能影响 OZON 平台运费成本。')

// ── 计算属性 ──
const statusType = computed(() => {
  const status = detail.value.product?.status
  if (status === 1) return 'success' as const
  if (status === 2) return 'warning' as const
  return 'default' as const
})

const totalStock = computed(() => {
  return (detail.value.inventory || []).reduce((sum: number, inv: any) => sum + (inv.quantity || 0), 0)
})

const lowStockCount = computed(() => {
  return (detail.value.inventory || []).filter((inv: any) => inv.quantity <= inv.safety_stock && inv.quantity > 0).length
})

const outOfStockCount = computed(() => {
  return (detail.value.inventory || []).filter((inv: any) => inv.quantity === 0).length
})

const publishedCount = computed(() => {
  return (detail.value.listings || []).filter((l: any) => l.status === 'synced').length
})

const missingFields = computed(() => {
  return detail.value.product?.missing_logistics_fields || []
})

// ── SKU 表格列 ──
const skuColumns = [
  { title: '规格', key: 'spec_desc', width: 150 },
  { title: 'SKU 编码', key: 'code', width: 120 },
  { title: '售价', key: 'sale_price', width: 100, render: (r: any) => `¥${r.sale_price || r.price || '-'}` },
  { title: '库存', key: 'stock', width: 80 },
  { title: '条码', key: 'barcode', width: 140 },
  { title: '状态', key: 'status', width: 80, render: (r: any) => (
    h(NTag, { size: 'tiny', type: r.status === 1 ? 'success' : 'default', round: true }, () => r.status === 1 ? '启用' : '停用')
  )},
]

// ── 辅助函数 ──
function formatDimensions(length?: number, width?: number, height?: number) {
  if (!length || !width || !height) return '-'
  return `${length} × ${width} × ${height} cm`
}

function formatWeight(weight?: number) {
  if (!weight) return '-'
  return `${weight} kg`
}

function cargoTypeLabel(value?: string) {
  const map: Record<string, string> = {
    normal: '普通货品',
    battery: '带电',
    liquid: '液体',
    sensitive: '敏感货',
  }
  return map[value || 'normal'] || value || '-'
}

function cargoTypeColor(value?: string) {
  const map: Record<string, string> = {
    normal: '#18a058',
    battery: '#d03050',
    liquid: '#2080f0',
    sensitive: '#f0a020',
  }
  return { color: map[value || 'normal'] || '#808080', textColor: '#fff' }
}

function platformColor(code: string): string {
  const m: Record<string, string> = {
    ozon: '#005bff', shopee: '#ee4d2d', wb: '#cb11ab',
    wildberries: '#cb11ab', aliexpress: '#e62e04', temu: '#e0120c',
  }
  return m[code.toLowerCase()] || '#2080f0'
}

function hasCompletePackage(product: any) {
  return !!product?.package_length_cm
    && !!product?.package_width_cm
    && !!product?.package_height_cm
    && !!product?.package_weight_kg
}

function goShippingCalculator() {
  const product = detail.value.product
  if (!hasCompletePackage(product)) {
    router.push(`/products/${productId}/edit`)
    return
  }
  router.push({
    path: '/shipping/calculator',
    query: {
      length_cm: String(product.package_length_cm),
      width_cm: String(product.package_width_cm),
      height_cm: String(product.package_height_cm),
      weight_kg: String(product.package_weight_kg),
      cargo_type: product.cargo_type || 'normal',
      quantity: '1',
      source_product_id: String(product.id),
      source_product_name: product.name || '',
    },
  })
}

onMounted(async () => {
  loading.value = true
  try {
    const res: any = await http.get(`/products/${productId}/detail`)
    detail.value = res.data || {}
  } catch (e: any) {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
})

async function handleDuplicate() {
  try {
    const res: any = await http.post(`/products/${productId}/duplicate`)
    if (res.code === 200) {
      message.success(`已复制为"${res.data.name}"`)
    }
  } catch (e: any) {
    message.error('复制失败')
  }
}
</script>

<style scoped>
/* ═════ 设计系统 Token 应用 ═════ */
.detail-container {
  padding: 24px;
  max-width: 1440px;
  margin: 0 auto;
  background: #f8fafc;
  min-height: 100vh;
}

/* ═════ 页面标题 ═════ */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e2e8f0;
}
.back-button {
  margin-bottom: 12px;
  color: #64748b;
}
.product-title-row {
  display: flex;
  align-items: center;
  gap: 14px;
}
.product-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #0ea5e9, #38bdf8);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}
.product-name {
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 6px 0;
}
.product-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: #94a3b8;
}
.meta-item {
  padding-left: 12px;
  border-left: 1px solid #e2e8f0;
}

/* ═════ AI 洞察栏 ═════ */
.ai-insight-bar {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 18px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 1px solid #0ea5e9;
  border-radius: 10px;
  margin-bottom: 20px;
  color: #0c4a6e;
}
.insight-content {
  flex: 1;
}
.insight-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 4px;
}
.insight-text {
  font-size: 13px;
  line-height: 1.5;
}

/* ═════ 信息卡片网格 ═════ */
.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.info-card {
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
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* ═════ 信息列表 ═════ */
.info-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  border-radius: 6px;
  transition: background 0.15s;
}
.info-item:hover {
  background: #f8fafc;
}
.info-label {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 500;
}
.info-value {
  font-size: 13px;
  color: #1e293b;
  font-weight: 500;
}
.text-secondary {
  color: #64748b;
}
.text-warning {
  color: #d97706;
  font-weight: 600;
}
.text-danger {
  color: #dc2626;
  font-weight: 600;
}

/* ═════ 卡片操作区 ═════ */
.card-actions {
  margin-top: 14px;
  display: flex;
  gap: 8px;
}
.missing-fields {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 8px 12px;
  background: #fef2f2;
  border-radius: 6px;
  font-size: 12px;
  color: #dc2626;
}

/* ═════ 库存概况 ═════ */
.inventory-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 16px;
}
.inventory-stat {
  text-align: center;
  padding: 12px 8px;
  background: #f8fafc;
  border-radius: 8px;
}
.inventory-stat-value {
  font-size: 22px;
  font-weight: 700;
  color: #1e293b;
}
.inventory-stat-label {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 4px;
}
.inventory-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.inventory-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  background: #f8fafc;
  border-radius: 6px;
  font-size: 13px;
}
.inventory-warehouse {
  flex: 1;
  font-weight: 500;
  color: #1e293b;
}
.inventory-quantity {
  font-weight: 700;
  color: #1e293b;
}
.inventory-safety {
  font-size: 11px;
  color: #94a3b8;
}

/* ═════ 区块卡片 ═════ */
.section-card {
  background: white;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
  margin-bottom: 20px;
}

/* ═════ SKU 表格 ═════ */
.section-card :deep(.n-data-table) {
  --n-border-color: transparent;
}

/* ═════ 平台列表 ═════ */
.platform-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.platform-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
  transition: all 0.15s;
}
.platform-item:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}
.platform-badge {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 9px;
  font-weight: 800;
  flex-shrink: 0;
}
.platform-info {
  flex: 1;
}
.platform-name {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}
.platform-id {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

/* ═════ 供应商列表 ═════ */
.supplier-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.supplier-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  transition: all 0.15s;
}
.supplier-item:hover {
  background: #f8fafc;
}
.supplier-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #f0f9ff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #0ea5e9;
}
.supplier-info {
  flex: 1;
}
.supplier-name {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}
.supplier-price {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

/* ═════ 响应式 ═════ */
@media (max-width: 1280px) {
  .info-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 960px) {
  .info-grid { grid-template-columns: 1fr; }
  .inventory-summary { grid-template-columns: repeat(3, 1fr); }
}
</style>
