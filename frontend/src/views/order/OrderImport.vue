<template>
  <div>
    <n-page-header subtitle="CSV 订单导入批次与明细">
      <template #title>📥 订单导入</template>
    </n-page-header>

    <n-space vertical size="large" style="margin-top: 12px;">
      <n-card :bordered="false">
        <template #header-extra>
          <n-button type="primary" @click="showUpload = true">上传 CSV</n-button>
        </template>
        <n-data-table :columns="batchColumns" :data="batchData" :loading="batchLoading" />
      </n-card>

      <n-card v-if="selectedBatchId" :bordered="false" title="批次明细">
        <n-descriptions label-placement="left" bordered :column="2">
          <n-descriptions-item label="批次 ID">{{ selectedBatchId }}</n-descriptions-item>
          <n-descriptions-item label="适配器">{{ selectedBatch?.adapter_code }}</n-descriptions-item>
        </n-descriptions>
        <n-data-table :columns="itemColumns" :data="itemData" :loading="itemLoading" style="margin-top: 12px;" />
      </n-card>
    </n-space>

    <n-modal v-model:show="showUpload" title="上传订单 CSV" preset="card" style="width: 560px;">
      <n-upload
        :show-file-list="false"
        :custom-request="handleUpload"
        accept=".csv"
      >
        <n-upload-dragger>
          <div style="padding: 24px 0;">
            <n-text>点击或拖拽上传 CSV 文件</n-text>
          </div>
        </n-upload-dragger>
      </n-upload>

      <n-divider>CSV 格式</n-divider>
      <n-code language="text" :code="csvExample" word-wrap />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { NButton, NCode, NDataTable, NDescriptions, NDescriptionsItem, NUpload, NUploadDragger } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useMessage, useDialog } from 'naive-ui'
import { listOrderImports, getOrderImport, listOrderImportItems, importOrders, processOrderImportChain } from '@/api/modules/orderImport'

const message = useMessage()
const dialog = useDialog()

const showUpload = ref(false)
const batchLoading = ref(false)
const itemLoading = ref(false)
const batchData = ref<any[]>([])
const itemData = ref<any[]>([])
const selectedBatchId = ref<number | null>(null)
const selectedBatch = ref<any | null>(null)

const statusRender = (row: any) => {
  const map: Record<string, any> = {
    imported: 'default',
    created_order: 'success',
    skipped_duplicate: 'warning',
    failed: 'error',
  }
  const labels: Record<string, string> = {
    imported: '已导入',
    created_order: '创建订单',
    skipped_duplicate: '跳过重复',
    failed: '失败',
  }
  return h('n-tag', { type: map[row.status] || 'default', size: 'small' }, { default: () => labels[row.status] || row.status })
}

const chainStatusRender = (row: any) => {
  const map: Record<string, any> = {
    chain_pending: 'default',
    chain_processed: 'success',
    chain_failed: 'error',
  }
  const labels: Record<string, string> = {
    chain_pending: '未处理',
    chain_processed: '已处理',
    chain_failed: '失败',
  }
  return h('n-tag', { type: map[row.chain_status] || 'default', size: 'small' }, { default: () => labels[row.chain_status] || row.chain_status })
}

const itemChainStatusRender = (row: any) => {
  const map: Record<string, any> = {
    chain_pending: 'default',
    ledger_rebuilt: 'info',
    exception_generated: 'success',
    chain_failed: 'error',
  }
  const labels: Record<string, string> = {
    chain_pending: '未处理',
    ledger_rebuilt: '账本已建',
    exception_generated: '异常已生成',
    chain_failed: '失败',
  }
  return h('n-tag', { type: map[row.chain_status] || 'default', size: 'small' }, { default: () => labels[row.chain_status] || row.chain_status })
}

const batchColumns: DataTableColumns<any> = [
  { title: 'ID', key: 'id', width: 80 },
  { title: '适配器', key: 'adapter_code' },
  { title: '平台', key: 'platform' },
  { title: '店铺', key: 'store_name' },
  { title: '文件', key: 'source_filename' },
  { title: '总行数', key: 'row_count', width: 100 },
  { title: '创建订单', key: 'created_order_count', width: 110 },
  { title: '跳过重复', key: 'skipped_duplicate_count', width: 110 },
  { title: '失败', key: 'failed_count', width: 90 },
  { title: '链路状态', key: 'chain_status', width: 110, render: (row: any) => chainStatusRender(row) },
  { title: '重建账本', key: 'ledger_rebuilt_count', width: 100 },
  { title: '生成异常', key: 'exception_generated_count', width: 100 },
  { title: '导入人', key: 'imported_by', width: 120 },
  { title: '导入时间', key: 'created_at', width: 180 },
  {
    title: '操作',
    width: 200,
    render: (row: any) => [
      h('n-button', { size: 'small', style: 'margin-right: 4px', onClick: () => openBatch(row) }, { default: () => '查看明细' }),
      row.chain_status === 'chain_pending' ? h('n-button', { size: 'small', type: 'warning', onClick: () => handleProcessChain(row) }, { default: () => '处理链路' }) : null,
    ],
  },
]

const itemColumns: DataTableColumns<any> = [
  { title: '行号', key: 'row_number', width: 80 },
  { title: '平台订单号', key: 'platform_order_no' },
  { title: 'SKU', key: 'sku_code' },
  { title: '数量', key: 'quantity', width: 90 },
  { title: '单价', key: 'unit_price', width: 100 },
  { title: '运费', key: 'shipping_fee', width: 100 },
  { title: '支付时间', key: 'paid_at', width: 170 },
  { title: '状态', key: 'status', width: 120, render: (row: any) => statusRender(row) },
  { title: '链路状态', key: 'chain_status', width: 130, render: (row: any) => itemChainStatusRender(row) },
  { title: '失败原因', key: 'failure_reason', ellipsis: { tooltip: true } },
]

async function fetchBatches() {
  batchLoading.value = true
  try {
    const res = await listOrderImports()
    batchData.value = (res as any).data?.data ?? []
  } catch (e: any) {
    message.error(e?.message ?? '加载失败')
  } finally {
    batchLoading.value = false
  }
}

async function openBatch(row: any) {
  selectedBatchId.value = row.id
  selectedBatch.value = row
  await fetchItems()
}

async function fetchItems() {
  if (!selectedBatchId.value) return
  itemLoading.value = true
  try {
    const res = await listOrderImportItems(selectedBatchId.value)
    itemData.value = (res as any).data ?? []
  } catch (e: any) {
    message.error(e?.message ?? '加载明细失败')
  } finally {
    itemLoading.value = false
  }
}

async function handleProcessChain(row: any) {
  try {
    const res = await processOrderImportChain(row.id)
    message.success('链路处理完成')
    await fetchBatches()
    if (selectedBatchId.value === row.id) {
      await fetchItems()
    }
  } catch (e: any) {
    message.error(e?.message ?? '链路处理失败')
  }
}

async function handleUpload({ file }: any) {
  const rawFile = file.file as File
  if (!rawFile) return false
  try {
    await importOrders(rawFile)
    message.success('上传成功')
    showUpload.value = false
    await fetchBatches()
  } catch (e: any) {
    message.error(e?.message ?? '上传失败')
  }
  return false
}

const csvExample =
  'platform,store_name,platform_order_no,order_no,sku_code,quantity,unit_price,currency,recipient_name,recipient_phone,country_code,shipping_address,shipping_fee,paid_at\n' +
  'Ozon,Ozon主店,EX-001,,SKU1001,1,12.5,USD,张三,+8613800000000,US,addr1,5.0,2026-06-01 12:00:00\n' +
  'Shopee,店铺A,EX-002,,SKU1002,2,9.9,CNY,李四,+8613700000000,CN,addr2,2.0,\n'

onMounted(() => {
  fetchBatches()
})
</script>
