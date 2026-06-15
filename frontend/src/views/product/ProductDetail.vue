<template>
  <n-page-header @back="router.back()">
    <template #title>📄 商品详情</template>
    <template #extra>
      <n-space>
        <n-button @click="handleDuplicate">📋 复制</n-button>
        <n-button type="primary" @click="router.push(`/products/${productId}/edit`)">编辑</n-button>
      </n-space>
    </template>
  </n-page-header>

  <n-spin :show="loading">
    <n-grid :cols="3" :x-gap="12" style="margin-top: 12px;">
      <!-- 基本信息 -->
      <n-grid-item :span="1">
        <n-card title="基本信息" :bordered="true">
          <n-descriptions label-placement="left" :column="1">
            <n-descriptions-item label="名称">{{ detail.product?.name }}</n-descriptions-item>
            <n-descriptions-item label="副标题">{{ detail.product?.subtitle || '-' }}</n-descriptions-item>
            <n-descriptions-item label="分类">{{ detail.product?.category_name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="品牌">{{ detail.product?.brand_name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="单位">{{ detail.product?.unit }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="detail.product?.status === 1 ? 'success' : detail.product?.status === 2 ? 'warning' : 'default'" size="small">
                {{ detail.product?.status_name }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="创建时间">{{ detail.product?.created_at }}</n-descriptions-item>
            <n-descriptions-item label="AI状态">
              <n-tag v-if="detail.product?.ai_status" :type="detail.product?.ai_status === 'completed' ? 'success' : 'warning'" size="small">
                {{ detail.product?.ai_status === 'completed' ? '已优化' : detail.product?.ai_status === 'failed' ? '失败' : '待处理' }}
              </n-tag>
              <span v-else style="color:#999;">-</span>
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-grid-item>

      <!-- 物流信息 -->
      <n-grid-item :span="1">
        <n-card title="物流信息" :bordered="true">
          <n-descriptions label-placement="left" :column="1">
            <n-descriptions-item label="货品类型">
              <n-tag :color="cargoTypeColor(detail.product?.cargo_type)" size="small">
                {{ cargoTypeLabel(detail.product?.cargo_type) }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="商品尺寸">
              {{ formatDimensions(detail.product?.product_length_cm, detail.product?.product_width_cm, detail.product?.product_height_cm) }}
            </n-descriptions-item>
            <n-descriptions-item label="商品重量">
              {{ formatWeight(detail.product?.product_weight_kg) }}
            </n-descriptions-item>
            <n-descriptions-item label="包装尺寸">
              {{ formatDimensions(detail.product?.package_length_cm, detail.product?.package_width_cm, detail.product?.package_height_cm) }}
            </n-descriptions-item>
            <n-descriptions-item label="包装重量">
              {{ formatWeight(detail.product?.package_weight_kg) }}
            </n-descriptions-item>
            <n-descriptions-item label="体积重预览">
              {{ detail.product?.package_volume_weight_kg != null ? `${detail.product.package_volume_weight_kg} kg` : '-' }}
            </n-descriptions-item>
            <n-descriptions-item label="物流状态">
              <n-tag v-if="detail.product?.logistics_status === 'complete'" type="success" size="small">可计算运费</n-tag>
              <div v-else>
                <n-tag type="warning" size="small">缺物流数据</n-tag>
                <div v-if="missingFields.length > 0" style="margin-top:4px;color:#d03050;font-size:12px;">
                  缺失: {{ missingFields.join('、') }}
                </div>
              </div>
            </n-descriptions-item>
          </n-descriptions>
          <template #action>
            <n-space>
              <n-button
                v-if="detail.product?.logistics_status === 'complete'"
                size="small"
                type="info"
                @click="goShippingCalculator"
              >
                试算运费
              </n-button>
              <n-button
                v-else
                size="small"
                type="warning"
                @click="router.push(`/products/${productId}/edit`)"
              >
                补齐物流数据
              </n-button>
            </n-space>
          </template>
        </n-card>
      </n-grid-item>

      <!-- SKU列表 -->
      <n-grid-item :span="1">
        <n-card title="SKU列表" :bordered="true">
          <n-empty v-if="!detail.skus?.length" description="暂无SKU" />
          <n-list v-else>
            <n-list-item v-for="sku in detail.skus" :key="sku.id">
              <template #header>
                <span style="font-size: 13px;">{{ sku.spec_desc || sku.code }}</span>
              </template>
              <div style="font-size: 12px; color: #666;">
                售价: <b>¥{{ sku.sale_price || sku.price || '-' }}</b>
                &nbsp;|&nbsp; 库存: <b>{{ sku.stock }}</b>
                &nbsp;|&nbsp; 条码: {{ sku.barcode || '-' }}
              </div>
            </n-list-item>
          </n-list>
        </n-card>
      </n-grid-item>

      <!-- 右侧：库存+供应商 -->
      <n-grid-item :span="1">
        <n-card title="库存信息" :bordered="true" style="margin-bottom: 12px;">
          <n-empty v-if="!detail.inventory?.length" description="暂无库存记录" />
          <n-list v-else>
            <n-list-item v-for="inv in detail.inventory" :key="inv.id">
              仓库: {{ inv.warehouse }} — 库存: <b :style="inv.quantity <= inv.safety_stock ? 'color:#d03050' : ''">{{ inv.quantity }}</b>
              &nbsp;(安全库存: {{ inv.safety_stock }})
            </n-list-item>
          </n-list>
        </n-card>

        <n-card title="发布状态" :bordered="true" style="margin-bottom: 12px;">
          <n-empty v-if="!detail.listings?.length" description="暂未发布到任何平台">
            <template #extra>
              <n-button size="small" type="primary" @click="router.push(`/listings`)">去发布</n-button>
            </template>
          </n-empty>
          <n-list v-else>
            <n-list-item v-for="l in detail.listings" :key="l.id">
              <template #prefix>
                <n-tag size="tiny" :color="{ color: l.platform_code === 'ozon' ? '#005bff' : '#ee4d2d' }">{{ l.platform_name }}</n-tag>
              </template>
              <span style="font-size: 13px;">
                <n-tag :type="l.status === 'synced' ? 'success' : 'warning'" size="tiny">{{ l.status === 'synced' ? '已发布' : l.status }}</n-tag>
              </span>
              <template #suffix>
                <span style="font-size: 12px; color: #999;">{{ l.platform_product_id }}</span>
              </template>
            </n-list-item>
          </n-list>
        </n-card>

        <n-card title="供应商" :bordered="true">
          <n-empty v-if="!detail.suppliers?.length" description="暂无供应商" />
          <n-list v-else>
            <n-list-item v-for="sup in detail.suppliers" :key="sup.id">
              {{ sup.supplier_name }}
              <span v-if="sup.supply_price" style="color:#999;"> — 供货价 ¥{{ sup.supply_price }}</span>
            </n-list-item>
          </n-list>
        </n-card>
      </n-grid-item>
    </n-grid>
  </n-spin>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import http from '@/api/http'

const router = useRouter()
const route = useRoute()
const message = useMessage()
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
    normal: '#18a058',
    battery: '#d03050',
    liquid: '#2080f0',
    sensitive: '#f0a020',
  }
  return { color: map[value || 'normal'] || '#808080', textColor: '#fff' }
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
