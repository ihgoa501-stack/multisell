<template>
  <n-page-header @back="router.back()">
    <template #title>📦 库存管理</template>
    <template #subtitle>管理SKU库存</template>
  </n-page-header>

  <!-- 选择SKU -->
  <n-card style="margin-top: 12px;" :bordered="false">
    <n-form inline>
      <n-form-item label="选择SKU">
        <n-select v-model:value="selectedSkuId" :options="skuOptions" clearable filterable style="width: 300px;" @update:value="fetchInventory" />
      </n-form-item>
    </n-form>
  </n-card>

  <!-- 库存信息 -->
  <n-card v-if="selectedSkuId" title="库存信息" style="margin-top: 12px;" :bordered="false">
    <n-descriptions v-if="inventory" :column="3" bordered>
      <n-descriptions-item label="当前库存">{{ inventory.quantity }}</n-descriptions-item>
      <n-descriptions-item label="安全库存">{{ inventory.safety_stock }}</n-descriptions-item>
      <n-descriptions-item label="仓库">{{ inventory.warehouse }}</n-descriptions-item>
      <n-descriptions-item label="货位">{{ inventory.location || '-' }}</n-descriptions-item>
    </n-descriptions>

    <n-divider />

    <n-form inline :model="invForm">
      <n-form-item label="库存数量">
        <n-input-number v-model:value="invForm.quantity" :min="0" style="width: 120px;" />
      </n-form-item>
      <n-form-item label="安全库存">
        <n-input-number v-model:value="invForm.safety_stock" :min="0" style="width: 120px;" />
      </n-form-item>
      <n-form-item label="仓库">
        <n-input v-model:value="invForm.warehouse" style="width: 150px;" />
      </n-form-item>
      <n-form-item label="货位">
        <n-input v-model:value="invForm.location" style="width: 150px;" />
      </n-form-item>
      <n-form-item>
        <n-button type="primary" :loading="updating" @click="handleUpdate">更新库存</n-button>
      </n-form-item>
    </n-form>
  </n-card>

  <!-- 库存变动记录 -->
  <n-card v-if="selectedSkuId" title="变动记录" style="margin-top: 12px;" :bordered="false">
    <n-data-table :columns="logColumns" :data="logs" :loading="loadingLogs" :bordered="true" :max-height="300" />
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { skuApi, inventoryApi } from '@/api'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const productId = Number(route.params.id)

const selectedSkuId = ref<number | null>(null)
const skuOptions = ref<any[]>([])
const inventory = ref<any>(null)
const logs = ref<any[]>([])
const updating = ref(false)
const loadingLogs = ref(false)

const invForm = ref({ quantity: 0, safety_stock: 0, warehouse: '默认仓库', location: '' })

const logColumns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: '变动类型', key: 'change_type' },
  { title: '变动数量', key: 'change_qty' },
  { title: '变动前', key: 'before_qty' },
  { title: '变动后', key: 'after_qty' },
  { title: '备注', key: 'remark' },
  { title: '操作人', key: 'operator' },
]

async function fetchSkus() {
  try {
    const res: any = await skuApi.getSkus(productId)
    skuOptions.value = (res.data || []).map((s: any) => ({ label: `${s.code} - ${s.spec_desc || ''}`, value: s.id }))
  } catch { /* ignore */ }
}

async function fetchInventory() {
  if (!selectedSkuId.value) return
  try {
    const [invRes, logRes] = await Promise.all([
      inventoryApi.get(selectedSkuId.value),
      inventoryApi.getLogs(selectedSkuId.value),
    ])
    const inv = (invRes as any).data
    inventory.value = inv
    logs.value = (logRes as any).data || []
    if (inv && inv.id) {
      invForm.value = { quantity: inv.quantity || 0, safety_stock: inv.safety_stock || 0, warehouse: inv.warehouse || '默认仓库', location: inv.location || '' }
    }
  } catch (e: any) { message.error(e.message) }
}

async function handleUpdate() {
  if (!selectedSkuId.value) return
  updating.value = true
  try {
    await inventoryApi.update(selectedSkuId.value, invForm.value)
    message.success('库存更新成功')
    await fetchInventory()
  } catch (e: any) { message.error(e.message) }
  finally { updating.value = false }
}

onMounted(fetchSkus)
</script>
