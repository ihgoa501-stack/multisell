<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h3 style="margin: 0;">运费账单对账</h3>
      <span style="color: var(--ant-color-text-secondary);">导入物流商账单并与订单运费快照对账</span>
    </div>

    <!-- 导入 -->
    <a-card title="导入账单" style="margin-top: 12px;">
      <a-space>
        <a-upload :show-upload-list="false" accept=".csv" :custom-request="handleImport">
          <a-button>上传 CSV 账单</a-button>
        </a-upload>
      </a-space>
      <a-alert
        v-if="importResult"
        type="success"
        :show-icon="false"
        style="margin-top: 8px;"
        :message="`导入完成：共 ${importResult.total_rows} 行，成功 ${importResult.imported_rows} 行，错误 ${importResult.error_rows} 行`"
      />
    </a-card>

    <!-- 批次列表 -->
    <a-card title="账单批次" style="margin-top: 12px;">
      <a-table
        :columns="batchColumns"
        :data-source="batches"
        :loading="loading"
        :pagination="{ pageSize: 10 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'status'">
            <a-tag :color="statusTag[record.status]?.color || 'default'">
              {{ statusTag[record.status]?.text || record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <a-button size="small" type="primary" @click="loadBatchDetail(record.id)">详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 批次详情 -->
    <a-card v-if="selectedBatch" :title="`批次详情: ${selectedBatch.source_filename}`" style="margin-top: 12px;">
      <a-space style="margin-bottom: 8px;">
        <a-button type="primary" @click="handleReconcile(selectedBatch.id)">对账</a-button>
        <a-tag color="success">匹配 {{ selectedBatch.matched_count }}</a-tag>
        <a-tag color="warning">不匹配 {{ selectedBatch.mismatch_count }}</a-tag>
        <a-tag color="error">无匹配 {{ selectedBatch.unmatched_count }}</a-tag>
      </a-space>
      <a-table
        :columns="itemColumns"
        :data-source="items"
        :loading="loadingItems"
        :pagination="{ pageSize: 20 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'cost_layer'">
            <CostLayerTag :layer="record.actual_shipping_cost_layer || 'actual'" />
          </template>
          <template v-else-if="column.dataIndex === 'reconciliation_status'">
            <a-tag :color="itemStatusTag[record.reconciliation_status]?.color || 'default'">
              {{ itemStatusTag[record.reconciliation_status]?.text || record.reconciliation_status }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <span v-if="record.reconciliation_status === 'manual_resolved'" style="color: var(--ant-color-text-tertiary);">已解决</span>
            <a-button v-else size="small" @click="handleResolve(record.id)">手动解决</a-button>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import CostLayerTag from '@/components/CostLayerTag.vue'
import { shippingApi } from '@/api/modules/shipping'

const loading = ref(false)
const loadingItems = ref(false)
const batches = ref<any[]>([])
const selectedBatch = ref<any | null>(null)
const items = ref<any[]>([])
const importResult = ref<any | null>(null)

const statusTag: Record<string, { color: string; text: string }> = {
  imported: { color: 'default', text: '已导入' },
  reconciled: { color: 'success', text: '已对账' },
}

const batchColumns = [
  { title: '文件名', dataIndex: 'source_filename', key: 'source_filename', ellipsis: true },
  { title: '总行数', dataIndex: 'row_count', key: 'row_count', width: 80 },
  { title: '匹配', dataIndex: 'matched_count', key: 'matched_count', width: 70 },
  { title: '差异', dataIndex: 'mismatch_count', key: 'mismatch_count', width: 70 },
  { title: '无匹配', dataIndex: 'unmatched_count', key: 'unmatched_count', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
]

const itemStatusTag: Record<string, { color: string; text: string }> = {
  matched: { color: 'success', text: '已匹配' },
  unmatched_bill: { color: 'error', text: '无匹配' },
  amount_mismatch: { color: 'warning', text: '金额差异' },
  missing_snapshot: { color: 'warning', text: '缺快照' },
  currency_mismatch: { color: 'warning', text: '币种差异' },
  manual_resolved: { color: 'processing', text: '已手动解决' },
}

const itemColumns = [
  { title: '行号', dataIndex: 'row_number', key: 'row_number', width: 60 },
  { title: '运单号', dataIndex: 'tracking_number', key: 'tracking_number', width: 130 },
  { title: '物流商', dataIndex: 'provider_name', key: 'provider_name', width: 120 },
  { title: '目的国', dataIndex: 'destination_country', key: 'destination_country', width: 80 },
  { title: '账单运费', dataIndex: 'actual_shipping_fee', key: 'actual_shipping_fee', width: 100 },
  { title: '快照运费', dataIndex: 'snapshot_shipping_fee', key: 'snapshot_shipping_fee', width: 100 },
  { title: '差异', dataIndex: 'variance_amount', key: 'variance_amount', width: 90 },
  { title: '成本来源', dataIndex: 'cost_layer', key: 'cost_layer', width: 80 },
  { title: '状态', dataIndex: 'reconciliation_status', key: 'reconciliation_status', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
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

async function handleImport(options: any) {
  const rawFile = options.file
  if (!rawFile) { message.error('未读取到文件'); options.onError(); return }
  try {
    const resp = await shippingApi.importBills(rawFile)
    importResult.value = resp.data
    message.success('导入完成')
    options.onSuccess()
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
