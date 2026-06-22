<template>
  <div class="page-header">
    <div class="page-header-back" @click="router.back()">&larr; 返回</div>
    <div class="page-header-content">
      <h2 class="page-header-title">库存管理</h2>
      <span class="page-header-subtitle">管理SKU库存</span>
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
          @change="fetchInventory"
        />
      </a-form-item>
    </a-form>
  </a-card>

  <!-- 库存信息 -->
  <a-card v-if="selectedSkuId" title="库存信息" style="margin-top: 12px;" :bordered="false">
    <a-descriptions v-if="inventory" :column="3" bordered>
      <a-descriptions-item label="当前库存">{{ inventory.quantity }}</a-descriptions-item>
      <a-descriptions-item label="安全库存">{{ inventory.safety_stock }}</a-descriptions-item>
      <a-descriptions-item label="仓库">{{ inventory.warehouse }}</a-descriptions-item>
      <a-descriptions-item label="货位">{{ inventory.location || '-' }}</a-descriptions-item>
    </a-descriptions>

    <a-divider />

    <a-form layout="inline" :model="invForm">
      <a-form-item label="库存数量">
        <a-input-number v-model:value="invForm.quantity" :min="0" style="width: 120px;" />
      </a-form-item>
      <a-form-item label="安全库存">
        <a-input-number v-model:value="invForm.safety_stock" :min="0" style="width: 120px;" />
      </a-form-item>
      <a-form-item label="仓库">
        <a-input v-model:value="invForm.warehouse" style="width: 150px;" />
      </a-form-item>
      <a-form-item label="货位">
        <a-input v-model:value="invForm.location" style="width: 150px;" />
      </a-form-item>
      <a-form-item>
        <a-button type="primary" :loading="updating" @click="handleUpdate">更新库存</a-button>
      </a-form-item>
    </a-form>
  </a-card>

  <!-- 库存变动记录 -->
  <a-card v-if="selectedSkuId" title="变动记录" style="margin-top: 12px;" :bordered="false">
    <a-table
      :columns="logColumns"
      :data-source="logs"
      :loading="loadingLogs"
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
import { skuApi, inventoryApi } from '@/api'

const router = useRouter()
const route = useRoute()
const productId = Number(route.params.id)

const selectedSkuId = ref<number | null>(null)
const skuOptions = ref<any[]>([])
const inventory = ref<any>(null)
const logs = ref<any[]>([])
const updating = ref(false)
const loadingLogs = ref(false)

const invForm = ref({ quantity: 0, safety_stock: 0, warehouse: '默认仓库', location: '' })

const logColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '变动类型', dataIndex: 'change_type', key: 'change_type' },
  { title: '变动数量', dataIndex: 'change_qty', key: 'change_qty' },
  { title: '变动前', dataIndex: 'before_qty', key: 'before_qty' },
  { title: '变动后', dataIndex: 'after_qty', key: 'after_qty' },
  { title: '备注', dataIndex: 'remark', key: 'remark' },
  { title: '操作人', dataIndex: 'operator', key: 'operator' },
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
