<template>
  <div>
    <n-page-header subtitle="头程 / FBA / 海外仓费用分摊">
      <template #title>费用分摊</template>
    </n-page-header>

    <!-- Import -->
    <n-card title="导入分摊" style="margin-top: 12px;">
      <n-space vertical>
        <n-space>
          <n-select v-model:value="form.allocation_type" :options="typeOptions" placeholder="分摊类型" style="width: 160px;" />
          <n-select v-model:value="form.allocation_method" :options="methodOptions" placeholder="分摊方法" style="width: 160px;" />
          <n-input-number v-model:value="form.total_amount" placeholder="总金额" :min="0.01" style="width: 140px;" />
          <n-input v-model:value="form.currency" placeholder="CNY" maxlength="10" style="width: 80px;" />
        </n-space>
        <n-upload :show-file-list="false" accept=".csv" :custom-request="handleImport">
          <n-button :disabled="!form.allocation_type || !form.allocation_method || !form.total_amount">上传 CSV</n-button>
        </n-upload>
      </n-space>
      <n-alert v-if="importResult" type="success" :show-icon="false" style="margin-top: 8px;">
        导入完成：共 {{ importResult.total_rows }} 行
      </n-alert>
    </n-card>

    <!-- Batch list -->
    <n-card title="分摊批次" style="margin-top: 12px;">
      <n-data-table :columns="batchColumns" :data="batches" :loading="loading" :pagination="{ pageSize: 10 }" />
    </n-card>

    <!-- Batch detail -->
    <n-card v-if="selectedBatch" :title="`批次详情 #${selectedBatch.id}`" style="margin-top: 12px;">
      <n-space style="margin-bottom: 8px;">
        <n-tag>{{ selectedBatch.allocation_type }}</n-tag>
        <n-tag>{{ selectedBatch.allocation_method }}</n-tag>
        <n-tag>总金额 {{ selectedBatch.total_amount }} {{ selectedBatch.currency }}</n-tag>
        <n-tag>状态: {{ selectedBatch.status }}</n-tag>
        <n-button v-if="selectedBatch.status === 'imported'" type="primary" :loading="calculating" @click="handleCalculate(selectedBatch.id)">计算分摊</n-button>
        <n-button v-if="selectedBatch.status === 'calculated'" type="success" :loading="posting" @click="handlePost(selectedBatch.id)">入账</n-button>
      </n-space>
      <n-data-table :columns="itemColumns" :data="items" :loading="loadingItems" :pagination="{ pageSize: 20 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref, reactive } from 'vue'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import {
  importAllocation,
  getAllocationBatches,
  getAllocationBatch,
  getAllocationItems,
  calculateAllocation,
  postAllocationToLedger,
  type AllocationBatch,
  type AllocationItem,
} from '@/api/modules/allocation'

const message = useMessage()
const loading = ref(false)
const loadingItems = ref(false)
const calculating = ref(false)
const posting = ref(false)
const batches = ref<any[]>([])
const selectedBatch = ref<any | null>(null)
const items = ref<any[]>([])
const importResult = ref<any | null>(null)

const form = reactive({
  allocation_type: null as string | null,
  allocation_method: null as string | null,
  total_amount: null as number | null,
  currency: 'CNY',
})

const typeOptions = [
  { label: '头程', value: 'first_leg' },
  { label: 'FBA', value: 'fba' },
  { label: '海外仓', value: 'overseas_warehouse' },
  { label: '其他', value: 'other' },
]

const methodOptions = [
  { label: '按数量', value: 'quantity' },
  { label: '按重量', value: 'weight' },
  { label: '按体积', value: 'volume' },
  { label: '按货值', value: 'value' },
]

const batchColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '类型', key: 'allocation_type', width: 100 },
  { title: '方法', key: 'allocation_method', width: 90 },
  { title: '总金额', key: 'total_amount', width: 100 },
  { title: '行数', key: 'row_count', width: 60 },
  { title: '状态', key: 'status', width: 100 },
  { title: '入账行', key: 'posted_count', width: 70 },
  {
    title: '操作',
    width: 80,
    render: (row: any) => h(NButton, { size: 'small', type: 'primary', onClick: () => loadBatchDetail(row.id) }, { default: () => '详情' }),
  },
]

const itemColumns = [
  { title: '行号', key: 'row_number', width: 60 },
  { title: 'SKU', key: 'sku_code', width: 100 },
  { title: '数量', key: 'quantity', width: 60 },
  { title: '权重', key: 'allocation_factor', width: 80 },
  { title: '分摊金额', key: 'allocated_amount', width: 100 },
  {
    title: '来源',
    key: 'cost_layer',
    width: 80,
    render: (row: any) => h(CostLayerTag, { layer: row.cost_layer }),
  },
  { title: '已入账', key: 'posted_to_ledger', width: 80, render: (row: any) => row.posted_to_ledger ? '✅' : '—' },
]

async function fetchBatches() {
  loading.value = true
  try { const r = await getAllocationBatches(); batches.value = r.data || [] }
  catch (err: any) { message.error(err?.message || '查询失败') }
  finally { loading.value = false }
}

async function handleImport(options: { file: UploadFileInfo; onFinish: () => void; onError: () => void }) {
  const rawFile = options.file.file
  if (!rawFile || !form.allocation_type || !form.allocation_method || !form.total_amount) {
    message.error('请填写完整参数'); options.onError(); return
  }
  try {
    const r = await importAllocation(rawFile, {
      allocation_type: form.allocation_type,
      allocation_method: form.allocation_method,
      total_amount: form.total_amount,
      currency: form.currency || 'CNY',
    })
    importResult.value = r.data
    message.success('导入完成')
    options.onFinish()
    await fetchBatches()
  } catch (err: any) { message.error(err?.message || '导入失败'); options.onError() }
}

async function loadBatchDetail(batchId: number) {
  try {
    const [batchResp, itemsResp] = await Promise.all([
      getAllocationBatch(batchId),
      getAllocationItems(batchId),
    ])
    selectedBatch.value = batchResp.data
    items.value = itemsResp.data || []
  } catch (err: any) { message.error(err?.message || '查询失败') }
}

async function handleCalculate(batchId: number) {
  calculating.value = true
  try {
    const r = await calculateAllocation(batchId)
    selectedBatch.value = { ...selectedBatch.value, status: 'calculated' }
    items.value = r.data?.items || []
    message.success('计算完成')
  } catch (err: any) { message.error(err?.message || '计算失败') }
  finally { calculating.value = false }
}

async function handlePost(batchId: number) {
  posting.value = true
  try {
    await postAllocationToLedger(batchId)
    message.success('入账完成')
    await loadBatchDetail(batchId)
    await fetchBatches()
  } catch (err: any) { message.error(err?.message || '入账失败') }
  finally { posting.value = false }
}

onMounted(fetchBatches)
</script>
