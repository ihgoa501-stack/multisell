<template>
  <!-- Custom page header (replacing n-page-header) -->
  <div class="page-header">
    <div class="page-header-left">
      <a-button type="text" @click="router.back()">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <h2 class="page-header-title">商品详情</h2>
    </div>
    <div class="page-header-extra">
      <a-space>
        <a-button @click="handleDuplicate">复制</a-button>
        <a-button type="primary" @click="router.push(`/products/${productId}/edit`)">编辑</a-button>
      </a-space>
    </div>
  </div>

  <a-spin :spinning="loading">
    <a-row :gutter="12" style="margin-top: 12px;">
      <!-- 基本信息 -->
      <a-col :span="8">
        <a-card title="基本信息" :bordered="true">
          <a-descriptions layout="horizontal" :column="1">
            <a-descriptions-item label="名称">{{ detail.product?.name }}</a-descriptions-item>
            <a-descriptions-item label="副标题">{{ detail.product?.subtitle || '-' }}</a-descriptions-item>
            <a-descriptions-item label="分类">{{ detail.product?.category_name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="品牌">{{ detail.product?.brand_name || '-' }}</a-descriptions-item>
            <a-descriptions-item label="单位">{{ detail.product?.unit }}</a-descriptions-item>
            <a-descriptions-item label="状态">
              <a-tag :color="detail.product?.status === 1 ? 'success' : detail.product?.status === 2 ? 'warning' : 'default'">
                {{ detail.product?.status_name }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ detail.product?.created_at }}</a-descriptions-item>
            <a-descriptions-item label="AI状态">
              <a-tag v-if="detail.product?.ai_status" :color="detail.product?.ai_status === 'completed' ? 'success' : 'warning'">
                {{ detail.product?.ai_status === 'completed' ? '已优化' : detail.product?.ai_status === 'failed' ? '失败' : '待处理' }}
              </a-tag>
              <span v-else style="color: var(--ant-color-text-tertiary);">-</span>
            </a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-col>

      <!-- 物流信息 -->
      <a-col :span="8">
        <a-card title="物流信息" :bordered="true">
          <a-descriptions layout="horizontal" :column="1">
            <a-descriptions-item label="货品类型">
              <a-tag :color="cargoTypeColor(detail.product?.cargo_type)">
                {{ cargoTypeLabel(detail.product?.cargo_type) }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="商品尺寸">
              {{ formatDimensions(detail.product?.product_length_cm, detail.product?.product_width_cm, detail.product?.product_height_cm) }}
            </a-descriptions-item>
            <a-descriptions-item label="商品重量">
              {{ formatWeight(detail.product?.product_weight_kg) }}
            </a-descriptions-item>
            <a-descriptions-item label="包装尺寸">
              {{ formatDimensions(detail.product?.package_length_cm, detail.product?.package_width_cm, detail.product?.package_height_cm) }}
            </a-descriptions-item>
            <a-descriptions-item label="包装重量">
              {{ formatWeight(detail.product?.package_weight_kg) }}
            </a-descriptions-item>
            <a-descriptions-item label="体积重预览">
              {{ detail.product?.package_volume_weight_kg != null ? `${detail.product.package_volume_weight_kg} kg` : '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="物流状态">
              <a-tag v-if="detail.product?.logistics_status === 'complete'" color="success">可计算运费</a-tag>
              <div v-else>
                <a-tag color="warning">缺物流数据</a-tag>
                <div v-if="missingFields.length > 0" style="margin-top:4px;color:var(--ant-color-error);font-size:12px;">
                  缺失: {{ missingFields.join('、') }}
                </div>
              </div>
            </a-descriptions-item>
          </a-descriptions>
          <template #extra>
            <a-space>
              <a-button
                v-if="detail.product?.logistics_status === 'complete'"
                size="small"
                type="primary"
                @click="goShippingCalculator"
              >
                试算运费
              </a-button>
              <a-button
                v-else
                size="small"
                type="default"
                danger
                @click="router.push(`/products/${productId}/edit`)"
              >
                补齐物流数据
              </a-button>
            </a-space>
          </template>
        </a-card>
      </a-col>

      <!-- SKU列表 -->
      <a-col :span="8">
        <a-card title="SKU列表" :bordered="true">
          <a-empty v-if="!detail.skus?.length" description="暂无SKU" />
          <a-list v-else :data-source="detail.skus" size="small">
            <template #renderItem="{ item: sku }">
              <a-list-item :key="sku.id">
                <a-list-item-meta>
                  <template #title>
                    <span style="font-size: 13px;">{{ sku.spec_desc || sku.code }}</span>
                  </template>
                  <template #description>
                    <div style="font-size: 12px; color: var(--ant-color-text-secondary);">
                      售价: <b>¥{{ sku.sale_price || sku.price || '-' }}</b>
                      &nbsp;|&nbsp; 库存: <b>{{ sku.stock }}</b>
                      &nbsp;|&nbsp; 条码: {{ sku.barcode || '-' }}
                    </div>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>

      <!-- 右侧：库存+供应商 -->
      <a-col :span="8" style="margin-top: 12px;">
        <a-card title="库存信息" :bordered="true" style="margin-bottom: 12px;">
          <a-empty v-if="!detail.inventory?.length" description="暂无库存记录" />
          <a-list v-else :data-source="detail.inventory" size="small">
            <template #renderItem="{ item: inv }">
              <a-list-item :key="inv.id">
                仓库: {{ inv.warehouse }} — 库存: <b :style="inv.quantity <= inv.safety_stock ? 'color:var(--ant-color-error)' : ''">{{ inv.quantity }}</b>
                &nbsp;(安全库存: {{ inv.safety_stock }})
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>

      <a-col :span="8" style="margin-top: 12px;">
        <a-card title="发布状态" :bordered="true" style="margin-bottom: 12px;">
          <template #extra>
            <a-button v-if="!detail.listings?.length" size="small" type="primary" @click="router.push(`/listings`)">去发布</a-button>
          </template>
          <a-empty v-if="!detail.listings?.length" description="暂未发布到任何平台" />
          <a-list v-else :data-source="detail.listings" size="small">
            <template #renderItem="{ item: l }">
              <a-list-item :key="l.id">
                <a-list-item-meta>
                  <template #title>
                    <a-space>
                      <a-tag :color="l.platform_code === 'ozon' ? '#005bff' : '#ee4d2d'">{{ l.platform_name }}</a-tag>
                      <a-tag :color="l.status === 'synced' ? 'success' : 'warning'">{{ l.status === 'synced' ? '已发布' : l.status }}</a-tag>
                    </a-space>
                  </template>
                  <template #description>
                    <span style="font-size: 12px; color: var(--ant-color-text-tertiary);">{{ l.platform_product_id }}</span>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>

      <a-col :span="8" style="margin-top: 12px;">
        <a-card title="供应商" :bordered="true">
          <a-empty v-if="!detail.suppliers?.length" description="暂无供应商" />
          <a-list v-else :data-source="detail.suppliers" size="small">
            <template #renderItem="{ item: sup }">
              <a-list-item :key="sup.id">
                {{ sup.supplier_name }}
                <span v-if="sup.supply_price" style="color:var(--ant-color-text-tertiary);"> — 供货价 ¥{{ sup.supply_price }}</span>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>
    </a-row>
  </a-spin>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import http from '@/api/http'

const router = useRouter()
const route = useRoute()
const productId = Number(route.params.id)
const loading = ref(false)
const detail = ref<any>({})

// 物流辅助函数
const missingFields = computed(() => {
  return detail.value.product?.missing_logistics_fields || []
})

function formatDimensions(length?: number, width?: number, height?: number) {
  if (!length || !width || !height) return '-'
  return `${length} x ${width} x ${height} cm`
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
    normal: 'green',
    battery: 'red',
    liquid: 'blue',
    sensitive: 'orange',
  }
  return map[value || 'normal'] || 'default'
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
/* Custom page header */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--ant-color-border, #e5e5e5);
  margin-bottom: 16px;
}

.page-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-header-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ant-color-text);
}

/* 卡片样式优化 */
:deep(.ant-card) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.ant-card:hover) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

/* 卡片标题 */
:deep(.ant-card-head-title) {
  font-weight: 600;
  font-size: 15px;
  color: var(--ant-color-text);
}

/* 描述列表标签 */
:deep(.ant-descriptions-item-label) {
  font-weight: 500;
  color: var(--ant-color-text-secondary);
  background: var(--ant-color-fill-quaternary, #f9fafb);
}

/* 标签样式 */
:deep(.ant-tag) {
  font-weight: 500;
  border-radius: 4px;
}

/* 加载状态 */
:deep(.ant-spin-nested-loading) {
  min-height: 400px;
}

/* AI 状态提示 */
.ai-status-hint {
  background: linear-gradient(135deg, #eff6ff 0%, #f0f9ff 100%);
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  padding: 12px 16px;
  margin-top: 16px;
}

/* 响应式调整 */
@media (max-width: 768px) {
  :deep(.ant-col) {
    max-width: 100% !important;
    flex: 0 0 100% !important;
  }
}
</style>
