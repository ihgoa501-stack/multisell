<template>
  <div>
    <n-page-header subtitle="导入物流商账单并与订单运费快照对账">
      <template #title>运费账单对账</template>
    </n-page-header>

    <!-- 导入 -->
    <n-card title="导入账单" style="margin-top: 12px;">
      <n-space>
        <n-upload :show-file-list="false" accept=".csv" :custom-request="handleImport">
          <n-button>上传 CSV 账单</n-button>
        </n-upload>
      </n-space>
      <n-alert v-if="importResult" type="success" :show-icon="false" style="margin-top: 8px;">
        导入完成：共 {{ importResult.total_rows }} 行，成功 {{ importResult.imported_rows }} 行，错误 {{ importResult.error_rows }} 行
      </n-alert>
    </n-card>

    <!-- 批次列表 -->
    <n-card title="账单批次" style="margin-top: 12px;">
      <n-data-table :columns="batchColumns" :data="batches" :loading="loading" :pagination="{ pageSize: 10 }" />
    </n-card>

    <!-- 批次详情 -->
    <n-card v-if="selectedBatch" :title="`批次详情: ${selectedBatch.source_filename}`" style="margin-top: 12px;">
      <n-space style="margin-bottom: 8px;">
        <n-button type="primary" @click="handleReconcile(selectedBatch.id)">对账</n-button>
        <n-tag>匹配 {{ selectedBatch.matched_count }}</n-tag>
        <n-tag type="warning">不匹配 {{ selectedBatch.mismatch_count }}</n-tag>
        <n-tag type="error">无匹配 {{ selectedBatch.unmatched_count }}</n-tag>
      </n-space>
      <n-data-table :columns="itemColumns" :data="items" :loading="loadingItems" :pagination="{ pageSize: 20 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NTag, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import { shippingApi } from '@/api/modules/shipping'

const message = useMessage()
const loading = ref(false)
const loadingItems = ref(false)
const batches = ref<any[]>([])
const selectedBatch = ref<any | null>(null)
const items = ref<any[]>([])
const importResult = ref<any | null>(null)

const statusTag: Record<string, { type: string; text: string }> = {
  imported: { type: 'default', text: '已导入' },
  reconciled: { type: 'success', text: '已对账' },
}

const batchColumns = [
  { title: '文件名', key: 'source_filename', ellipsis: { tooltip: true } },
  { title: '总行数', key: 'row_count', width: 80 },
  { title: '匹配', key: 'matched_count', width: 70 },
  { title: '差异', key: 'mismatch_count', width: 70 },
  { title: '无匹配', key: 'unmatched_count', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: any) => {
      const meta = statusTag[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: meta.type as any, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '操作',
    width: 100,
    render: (row: any) =>
      h(NButton, { size: 'small', type: 'primary', onClick: () => loadBatchDetail(row.id) }, { default: () => '详情' }),
  },
]

const itemStatusTag: Record<string, { type: string; text: string }> = {
  matched: { type: 'success', text: '已匹配' },
  unmatched_bill: { type: 'error', text: '无匹配' },
  amount_mismatch: { type: 'warning', text: '金额差异' },
  missing_snapshot: { type: 'warning', text: '缺快照' },
  currency_mismatch: { type: 'warning', text: '币种差异' },
  manual_resolved: { type: 'info', text: '已手动解决' },
}

const itemColumns = [
  { title: '行号', key: 'row_number', width: 60 },
  { title: '运单号', key: 'tracking_number', width: 130 },
  { title: '物流商', key: 'provider_name', width: 120 },
  { title: '目的国', key: 'destination_country', width: 80 },
  { title: '账单运费', key: 'actual_shipping_fee', width: 100 },
  { title: '快照运费', key: 'snapshot_shipping_fee', width: 100 },
  { title: '差异', key: 'variance_amount', width: 90 },
  {
    title: '成本来源',
    key: 'cost_layer',
    width: 80,
    render: (row: any) => h(CostLayerTag, { layer: row.cost_layer || 'estimated' }),
  },
  {
    title: '状态',
    key: 'reconciliation_status',
    width: 100,
    render: (row: any) => {
      const meta = itemStatusTag[row.reconciliation_status] || { type: 'default', text: row.reconciliation_status }
      return h(NTag, { type: meta.type as any, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '操作',
    width: 160,
    render: (row: any) => {
      if (row.reconciliation_status === 'manual_resolved') return h('span', '已解决')
      return h(NButton, {
        size: 'small',
        onClick: () => handleResolve(row.id),
      }, { default: () => '手动解决' })
    },
  },
]

async function fetchBatches() {
  loading.value = true
  try {
    const resp = await shippingApi.listBillBatches()
    batches.value = resp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function handleImport(options: { file: UploadFileInfo; onFinish: () => void; onError: () => void }) {
  const rawFile = options.file.file
  if (!rawFile) { message.error('未读取到文件'); options.onError(); return }
  try {
    const resp = await shippingApi.importBills(rawFile)
    importResult.value = resp.data
    message.success('导入完成')
    options.onFinish()
    await fetchBatches()
  } catch (err: any) {
    message.error(err?.message || '导入失败')
    options.onError()
  }
}

async function loadBatchDetail(batchId: number) {
  try {
    const [batchResp, itemsResp] = await Promise.all([
      shippingApi.getBillBatch(batchId),
      shippingApi.listBillItems(batchId),
    ])
    selectedBatch.value = batchResp.data
    items.value = itemsResp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  }
}

async function handleReconcile(batchId: number) {
  try {
    await shippingApi.reconcileBatch(batchId)
    message.success('对账完成')
    await loadBatchDetail(batchId)
    await fetchBatches()
  } catch (err: any) {
    message.error(err?.message || '对账失败')
  }
}

async function handleResolve(itemId: number) {
  try {
    await shippingApi.resolveBillItem(itemId, '已人工确认')
    message.success('已解决')
    if (selectedBatch.value) await loadBatchDetail(selectedBatch.value.id)
  } catch (err: any) {
    message.error(err?.message || '解决失败')
  }
}

onMounted(fetchBatches)
</script>
