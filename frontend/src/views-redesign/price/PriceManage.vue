<template>
  <div class="page-header">
    <div class="page-header-back" @click="router.back()">&larr; 返回</div>
    <div class="page-header-content">
      <h2 class="page-header-title">价格管理</h2>
      <span class="page-header-subtitle">管理SKU价格与调价记录</span>
    </div>
  </div>

  <!-- 选择SKU -->
  <a-card style="margin-top: 12px;" :bordered="false">
    <a-form layout="inline">
      <a-form-item label="选择SKU">
        <a-select
          v-model:value="selectedSkuId"
          :options="skuOptions"
          allowClear
          showSearch
          style="width: 300px;"
          placeholder="请选择SKU"
          @change="fetchPrices"
        />
      </a-form-item>
    </a-form>
  </a-card>

  <!-- 价格列表 -->
  <a-card v-if="selectedSkuId" title="当前价格" style="margin-top: 12px;" :bordered="false">
    <a-table
      :columns="priceColumns"
      :data-source="prices"
      :loading="loadingPrices"
      :pagination="false"
      bordered
      row-key="id"
      size="middle"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          {{ record.status === 1 ? '启用' : '禁用' }}
        </template>
      </template>
    </a-table>
  </a-card>

  <!-- 设置价格 -->
  <a-card v-if="selectedSkuId" title="设置价格" style="margin-top: 12px;" :bordered="false">
    <a-form layout="inline" :model="priceForm">
      <a-form-item label="价格类型">
        <a-select v-model:value="priceForm.price_type" :options="priceTypeOptions" style="width: 180px;" />
      </a-form-item>
      <a-form-item label="价格">
        <a-input-number v-model:value="priceForm.price" :min="0" :precision="2" style="width: 150px;" />
      </a-form-item>
      <a-form-item>
        <a-button type="primary" :loading="settingPrice" @click="handleSetPrice">设置</a-button>
      </a-form-item>
    </a-form>
  </a-card>

  <!-- 调价历史 -->
  <a-card v-if="selectedSkuId" title="调价历史" style="margin-top: 12px;" :bordered="false">
    <a-table
      :columns="historyColumns"
      :data-source="priceHistory"
      :loading="loadingHistory"
      :pagination="false"
      bordered
      row-key="id"
      size="middle"
      :scroll="{ y: 300 }"
    />
  </a-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { skuApi, priceApi } from '@/api'

const router = useRouter()
const route = useRoute()
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
  { title: '价格类型', dataIndex: 'price_type', key: 'price_type' },
  { title: '价格', dataIndex: 'price', key: 'price' },
  { title: '生效时间', dataIndex: 'start_time', key: 'start_time' },
  { title: '失效时间', dataIndex: 'end_time', key: 'end_time' },
  { title: '状态', dataIndex: 'status', key: 'status' },
]

const historyColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '价格类型', dataIndex: 'price_type', key: 'price_type' },
  { title: '旧价格', dataIndex: 'old_price', key: 'old_price' },
  { title: '新价格', dataIndex: 'new_price', key: 'new_price' },
  { title: '操作人', dataIndex: 'operator', key: 'operator' },
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

<style scoped>
.page-header {
  padding: 12px 0;
}
.page-header-back {
  cursor: pointer;
  color: var(--ant-color-primary);
  margin-bottom: 8px;
  font-size: 14px;
}
.page-header-back:hover {
  opacity: 0.8;
}
.page-header-content {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.page-header-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  color: var(--ant-color-text-secondary);
  font-size: 14px;
}
</style>
