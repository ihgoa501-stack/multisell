<template>
  <n-page-header @back="router.back()">
    <template #title>💰 价格管理</template>
    <template #subtitle>管理SKU价格与调价记录</template>
  </n-page-header>

  <!-- 选择SKU -->
  <n-card style="margin-top: 12px;" :bordered="false">
    <n-form inline>
      <n-form-item label="选择SKU">
        <n-select v-model:value="selectedSkuId" :options="skuOptions" clearable filterable style="width: 300px;" @update:value="fetchPrices" />
      </n-form-item>
    </n-form>
  </n-card>

  <!-- 价格列表 -->
  <n-card v-if="selectedSkuId" title="当前价格" style="margin-top: 12px;" :bordered="false">
    <n-data-table :columns="priceColumns" :data="prices" :loading="loadingPrices" :bordered="true" />
  </n-card>

  <!-- 设置价格 -->
  <n-card v-if="selectedSkuId" title="设置价格" style="margin-top: 12px;" :bordered="false">
    <n-form inline :model="priceForm">
      <n-form-item label="价格类型">
        <n-select v-model:value="priceForm.price_type" :options="priceTypeOptions" style="width: 180px;" />
      </n-form-item>
      <n-form-item label="价格">
        <n-input-number v-model:value="priceForm.price" :min="0" :precision="2" style="width: 150px;" />
      </n-form-item>
      <n-form-item>
        <n-button type="primary" :loading="settingPrice" @click="handleSetPrice">设置</n-button>
      </n-form-item>
    </n-form>
  </n-card>

  <!-- 调价历史 -->
  <n-card v-if="selectedSkuId" title="调价历史" style="margin-top: 12px;" :bordered="false">
    <n-data-table :columns="historyColumns" :data="priceHistory" :loading="loadingHistory" :bordered="true" :max-height="300" />
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { skuApi, priceApi } from '@/api'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const productId = Number(route.params.id)

const selectedSkuId = ref<number | null>(null)
const skuOptions = ref<any[]>([])
const prices = ref<any[]>([])
const priceHistory = ref<any[]>([])
const loadingPrices = ref(false)
const loadingHistory = ref(false)
const settingPrice = ref(false)

const priceForm = ref({ price_type: 'sale_price', price: 0 })

const priceTypeOptions = [
  { label: '销售价', value: 'sale_price' },
  { label: '市场价', value: 'market_price' },
  { label: '成本价', value: 'cost_price' },
]

const priceColumns = [
  { title: '价格类型', key: 'price_type' },
  { title: '价格', key: 'price' },
  { title: '生效时间', key: 'start_time' },
  { title: '失效时间', key: 'end_time' },
  { title: '状态', key: 'status', render: (r: any) => r.status === 1 ? '启用' : '禁用' },
]

const historyColumns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: '价格类型', key: 'price_type' },
  { title: '旧价格', key: 'old_price' },
  { title: '新价格', key: 'new_price' },
  { title: '操作人', key: 'operator' },
]

async function fetchSkus() {
  try {
    const res: any = await skuApi.getSkus(productId)
    skuOptions.value = (res.data || []).map((s: any) => ({ label: `${s.code} - ${s.spec_desc || ''}`, value: s.id }))
  } catch { /* ignore */ }
}

async function fetchPrices() {
  if (!selectedSkuId.value) return
  loadingPrices.value = true
  try {
    const [pRes, hRes] = await Promise.all([
      priceApi.getPricesBySku(selectedSkuId.value),
      priceApi.getPriceHistory(selectedSkuId.value),
    ])
    prices.value = (pRes as any).data || []
    priceHistory.value = (hRes as any).data || []
  } catch (e: any) { message.error(e.message) }
  finally { loadingPrices.value = false; loadingHistory.value = false }
}

async function handleSetPrice() {
  settingPrice.value = true
  try {
    await priceApi.setPrice({
      sku_id: selectedSkuId.value,
      price_type: priceForm.value.price_type,
      price: priceForm.value.price,
    })
    message.success('价格设置成功')
    await fetchPrices()
  } catch (e: any) { message.error(e.message) }
  finally { settingPrice.value = false }
}

onMounted(fetchSkus)
</script>
