<template>
  <div>
    <h3 style="margin-bottom: 16px;">批量上架前经营决策</h3>

    <n-card title="批量测算">
      <n-space vertical :size="12">
        <n-alert type="info" :show-icon="false">
          最多一次测算 100 个 SKU。每行独立返回结果，单个 SKU 错误不会中断整批。
        </n-alert>

        <n-space>
          <n-button type="primary" @click="addRow">新增行</n-button>
          <n-button @click="removeSelectedRows" :disabled="checkedRowKeys.length === 0">删除选中</n-button>
          <n-button @click="handleDownloadTemplate">下载模板</n-button>
          <n-upload
            :show-file-list="false"
            accept=".xlsx"
            :custom-request="handlePreviewUpload"
          >
            <n-button>上传预览</n-button>
          </n-upload>
          <n-button type="primary" :loading="loading" @click="handleCalculate">批量计算</n-button>
          <n-button v-if="batchResult" @click="handleExportResult">导出结果</n-button>
          <n-button v-if="batchResult" type="primary" @click="handleCreateListingTasks">生成上架任务</n-button>
        </n-space>

        <n-data-table
          :columns="inputColumns"
          :data="rows"
          :row-key="rowKey"
          v-model:checked-row-keys="checkedRowKeys"
          :pagination="false"
          size="small"
        />
      </n-space>
    </n-card>

    <n-alert
      v-if="previewErrors.length > 0"
      type="warning"
      :show-icon="false"
      style="margin-top: 16px;"
    >
      <div v-for="err in previewErrors" :key="err">{{ err }}</div>
    </n-alert>

    <n-card v-if="batchResult" title="汇总结果" style="margin-top: 16px;">
      <n-descriptions :column="4" bordered>
        <n-descriptions-item label="总行数">{{ batchResult.summary.total_items }}</n-descriptions-item>
        <n-descriptions-item label="成功">{{ batchResult.summary.success_count }}</n-descriptions-item>
        <n-descriptions-item label="错误">{{ batchResult.summary.error_count }}</n-descriptions-item>
        <n-descriptions-item label="平均利润率">{{ batchResult.summary.average_profit_margin }}%</n-descriptions-item>
        <n-descriptions-item label="建议上架">{{ batchResult.summary.approve_count }}</n-descriptions-item>
        <n-descriptions-item label="不建议">{{ batchResult.summary.reject_count }}</n-descriptions-item>
        <n-descriptions-item label="数据不足">{{ batchResult.summary.needs_data_count }}</n-descriptions-item>
      </n-descriptions>
    </n-card>

    <n-card v-if="batchResult" title="明细结果" style="margin-top: 16px;">
      <n-data-table
        :columns="resultColumns"
        :data="batchResult.items"
        :pagination="{ pageSize: 20 }"
        size="small"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { NInput, NInputNumber, NSelect, NTag, NUpload, useMessage } from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import {
  calculateBatchPreListingDecision,
  downloadBatchPreListingDecisionTemplate,
  exportBatchPreListingDecisionResults,
  previewBatchPreListingDecisionExcel,
  type PreListingDecisionBatchItem,
  type PreListingDecisionBatchItemResult,
  type PreListingDecisionBatchResponse,
  type PreListingDecisionExcelPreviewResponse,
} from '@/api/modules/decision'
import { createListingTasksFromDecisions } from '@/api/modules/listing'

type BatchInputRow = PreListingDecisionBatchItem & {
  key: string
}

const message = useMessage()
const loading = ref(false)
const checkedRowKeys = ref<string[]>([])
const batchResult = ref<PreListingDecisionBatchResponse | null>(null)
const rows = reactive<BatchInputRow[]>([createRow()])

const cargoTypeOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感', value: 'sensitive' },
]

function createRow(): BatchInputRow {
  const key = `row-${Date.now()}-${Math.random().toString(16).slice(2)}`
  return {
    key,
    item_key: key,
    sku_id: null as unknown as number,
    destination_country: 'RU',
    target_sale_price: null as unknown as number,
    platform_id: null,
    category_id: null,
    platform_fee_pct: 10,
    payment_fee_pct: 3,
    other_fee: 0,
    minimum_margin_pct: 20,
    cargo_type: 'normal',
  }
}

function rowKey(row: BatchInputRow) {
  return row.key
}

function addRow() {
  if (rows.length >= 100) {
    message.warning('一次最多测算 100 行')
    return
  }
  rows.push(createRow())
}

function removeSelectedRows() {
  const selected = new Set(checkedRowKeys.value)
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (selected.has(rows[i].key)) {
      rows.splice(i, 1)
    }
  }
  checkedRowKeys.value = []
  if (rows.length === 0) {
    rows.push(createRow())
  }
}

function validateRows() {
  for (const [idx, row] of rows.entries()) {
    if (!row.sku_id || !row.destination_country || !row.target_sale_price) {
      message.warning(`第 ${idx + 1} 行缺少 SKU ID、目的国或目标售价`)
      return false
    }
  }
  return true
}

function setNumberField(row: BatchInputRow, field: keyof BatchInputRow, value: number | null) {
  ;(row as unknown as Record<string, number | null>)[field as string] = value
}

async function handleCalculate() {
  if (!validateRows()) return

  loading.value = true
  batchResult.value = null
  try {
    const payload = {
      items: rows.map(({ key: _key, ...row }) => row),
    }
    const resp = await calculateBatchPreListingDecision(payload)
    batchResult.value = resp.data as unknown as PreListingDecisionBatchResponse
  } catch (err: any) {
    message.error(err?.response?.data?.message || err?.message || '批量测算失败')
  } finally {
    loading.value = false
  }
}

const previewErrors = ref<string[]>([])

function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

function applyPreview(preview: PreListingDecisionExcelPreviewResponse) {
  previewErrors.value = preview.items
    .filter((item) => item.errors.length > 0)
    .flatMap((item) => item.errors.map((err) => `第 ${item.row_number} 行：${err}`))

  const validItems = preview.items
    .filter((item) => item.item)
    .map((item) => item.item as PreListingDecisionBatchItem)

  rows.splice(0, rows.length)
  for (const item of validItems) {
    const key = item.item_key || `row-${Date.now()}-${Math.random().toString(16).slice(2)}`
    rows.push({
      key,
      ...item,
    })
  }
  if (rows.length === 0) {
    rows.push(createRow())
  }
}

async function handleDownloadTemplate() {
  try {
    const resp = await downloadBatchPreListingDecisionTemplate()
    downloadBlob(resp as unknown as Blob, 'prelisting_decision_template.xlsx')
  } catch (err: any) {
    message.error(err?.message || '下载模板失败')
  }
}

async function handlePreviewUpload(options: { file: UploadFileInfo; onFinish: () => void; onError: () => void }) {
  const rawFile = options.file.file
  if (!rawFile) {
    message.error('未读取到上传文件')
    options.onError()
    return
  }
  try {
    const resp = await previewBatchPreListingDecisionExcel(rawFile)
    const preview = resp.data as unknown as PreListingDecisionExcelPreviewResponse
    applyPreview(preview)
    message.success(`解析成功：有效 ${preview.valid_rows} 行，错误 ${preview.error_rows} 行`)
    options.onFinish()
  } catch (err: any) {
    message.error(err?.message || '上传预览失败')
    options.onError()
  }
}

async function handleExportResult() {
  if (!batchResult.value) return
  try {
    const resp = await exportBatchPreListingDecisionResults(batchResult.value)
    downloadBlob(resp as unknown as Blob, 'prelisting_decision_results.xlsx')
  } catch (err: any) {
    message.error(err?.message || '导出结果失败')
  }
}

async function handleCreateListingTasks() {
  if (!batchResult.value) return
  const approved = batchResult.value.items.filter((item) => item.status === 'success' && item.result?.recommendation === 'approve')
  if (approved.length === 0) {
    message.warning('没有可生成上架任务的 approve 结果')
    return
  }

  const platformByKey = new Map(rows.map((row) => [row.item_key || row.key, row.platform_id]))
  const items = approved
    .map((item) => {
      const platformId = platformByKey.get(item.item_key || '')
      if (!item.result || !item.sku_id || !platformId) return null
      return {
        item_key: item.item_key,
        sku_id: item.sku_id,
        platform_id: platformId,
        decision_result: item.result,
      }
    })
    .filter(Boolean) as any[]

  if (items.length === 0) {
    message.warning('approve 结果缺少平台ID，无法生成上架任务')
    return
  }

  try {
    const resp = await createListingTasksFromDecisions(items)
    const data = resp.data
    message.success(`生成完成：新建 ${data.created_count}，复用 ${data.reused_count}，跳过 ${data.skipped_count}`)
  } catch (err: any) {
    message.error(err?.message || '生成上架任务失败')
  }
}

function renderNumberInput(row: BatchInputRow, field: keyof BatchInputRow, min = 0, precision = 2) {
  return h(NInputNumber, {
    value: row[field] as number | null,
    min,
    precision,
    style: 'width: 100%;',
    'onUpdate:value': (value: number | null) => {
      setNumberField(row, field, value)
    },
  })
}

const inputColumns = [
  { type: 'selection' as const },
  {
    title: 'SKU ID',
    key: 'sku_id',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'sku_id', 1, 0),
  },
  {
    title: '目的国',
    key: 'destination_country',
    width: 110,
    render: (row: BatchInputRow) =>
      h(NInput, {
        value: row.destination_country,
        maxlength: 10,
        'onUpdate:value': (value: string) => {
          row.destination_country = value
        },
      }),
  },
  {
    title: '目标售价',
    key: 'target_sale_price',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'target_sale_price', 0.01, 2),
  },
  {
    title: '平台ID',
    key: 'platform_id',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'platform_id', 1, 0),
  },
  {
    title: '平台费率%',
    key: 'platform_fee_pct',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'platform_fee_pct', 0, 1),
  },
  {
    title: '支付费率%',
    key: 'payment_fee_pct',
    width: 130,
    render: (row: BatchInputRow) => renderNumberInput(row, 'payment_fee_pct', 0, 1),
  },
  {
    title: '其他费用',
    key: 'other_fee',
    width: 120,
    render: (row: BatchInputRow) => renderNumberInput(row, 'other_fee', 0, 2),
  },
  {
    title: '最低利润率%',
    key: 'minimum_margin_pct',
    width: 140,
    render: (row: BatchInputRow) => renderNumberInput(row, 'minimum_margin_pct', 0, 1),
  },
  {
    title: '货品类型',
    key: 'cargo_type',
    width: 120,
    render: (row: BatchInputRow) =>
      h(NSelect, {
        value: row.cargo_type,
        options: cargoTypeOptions,
        'onUpdate:value': (value: string) => {
          row.cargo_type = value
        },
      }),
  },
]

const resultColumns = [
  { title: '行号', key: 'index', render: (row: PreListingDecisionBatchItemResult) => row.index + 1 },
  { title: 'SKU ID', key: 'sku_id' },
  {
    title: '状态',
    key: 'status',
    render: (row: PreListingDecisionBatchItemResult) =>
      h(
        NTag,
        { type: row.status === 'success' ? 'success' : 'error', size: 'small' },
        { default: () => (row.status === 'success' ? '成功' : '错误') },
      ),
  },
  {
    title: '建议',
    key: 'recommendation',
    render: (row: PreListingDecisionBatchItemResult) => row.result?.recommendation || '-',
  },
  {
    title: '利润率',
    key: 'profit_margin',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result ? `${row.result.profit_margin}%` : '-',
  },
  {
    title: '利润',
    key: 'profit_amount',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result ? `${row.result.profit_amount}` : '-',
  },
  {
    title: '费用来源',
    key: 'platform_fee_source',
    render: (row: PreListingDecisionBatchItemResult) =>
      row.result?.platform_fee_source === 'rule' ? '规则库' : row.result ? '手动输入' : '-',
  },
  {
    title: '原因/错误',
    key: 'message',
    render: (row: PreListingDecisionBatchItemResult) => {
      if (row.error_message) return row.error_message
      const reasons = row.result?.blocking_reasons || []
      const warnings = row.result?.warnings || []
      return [...reasons, ...warnings].join('；') || '-'
    },
  },
]
</script>
