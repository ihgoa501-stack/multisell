<template>
  <div>
    <h3 style="margin-bottom: 16px;">批量上架前经营决策</h3>

    <a-card title="批量测算">
      <a-space direction="vertical" :size="12" style="width: 100%;">
        <a-alert type="info" :show-icon="false" message="最多一次测算 100 个 SKU。每行独立返回结果，单个 SKU 错误不会中断整批。" />

        <a-space>
          <a-button type="primary" @click="addRow">新增行</a-button>
          <a-button @click="removeSelectedRows" :disabled="checkedRowKeys.length === 0">删除选中</a-button>
          <a-button @click="handleDownloadTemplate">下载模板</a-button>
          <a-upload
            :show-upload-list="false"
            accept=".xlsx"
            :custom-request="handlePreviewUpload"
          >
            <a-button>上传预览</a-button>
          </a-upload>
          <a-button type="primary" :loading="loading" @click="handleCalculate">批量计算</a-button>
          <a-button v-if="batchResult" @click="handleExportResult">导出结果</a-button>
          <a-button v-if="batchResult" type="primary" @click="handleCreateListingTasks">生成上架任务</a-button>
        </a-space>

        <a-table
          :columns="inputColumns"
          :data-source="rows"
          :row-key="rowKey"
          :row-selection="{ selectedRowKeys: checkedRowKeys, onChange: onSelectChange }"
          :pagination="false"
          size="small"
        />
      </a-space>
    </a-card>

    <a-alert
      v-if="previewErrors.length > 0"
      type="warning"
      :show-icon="false"
      style="margin-top: 16px;"
    >
      <template #message>
        <div v-for="err in previewErrors" :key="err">{{ err }}</div>
      </template>
    </a-alert>

    <a-card v-if="batchResult" title="汇总结果" style="margin-top: 16px;">
      <a-descriptions :column="4" bordered>
        <a-descriptions-item label="总行数">{{ batchResult.summary.total_items }}</a-descriptions-item>
        <a-descriptions-item label="成功">{{ batchResult.summary.success_count }}</a-descriptions-item>
        <a-descriptions-item label="错误">{{ batchResult.summary.error_count }}</a-descriptions-item>
        <a-descriptions-item label="平均利润率">{{ batchResult.summary.average_profit_margin }}%</a-descriptions-item>
        <a-descriptions-item label="建议上架">{{ batchResult.summary.approve_count }}</a-descriptions-item>
        <a-descriptions-item label="不建议">{{ batchResult.summary.reject_count }}</a-descriptions-item>
        <a-descriptions-item label="数据不足">{{ batchResult.summary.needs_data_count }}</a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-card v-if="batchResult" title="明细结果" style="margin-top: 16px;">
      <a-table
        :columns="resultColumns"
        :data-source="batchResult.items"
        :pagination="{ pageSize: 20 }"
        size="small"
        row-key="index"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'index'">{{ (record as any).index + 1 }}</template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="(record as any).status === 'success' ? 'success' : 'error'">
              {{ (record as any).status === 'success' ? '成功' : '错误' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'recommendation'">
            {{ (record as any).result?.recommendation || '-' }}
          </template>
          <template v-else-if="column.key === 'profit_margin'">
            {{ (record as any).result ? `${(record as any).result.profit_margin}%` : '-' }}
          </template>
          <template v-else-if="column.key === 'profit_amount'">
            {{ (record as any).result ? `${(record as any).result.profit_amount}` : '-' }}
          </template>
          <template v-else-if="column.key === 'shipping_cost_layer'">
            <CostLayerTag v-if="(record as any).result" :layer="(record as any).result.shipping_cost_layer || 'estimated'" />
            <template v-else>-</template>
          </template>
          <template v-else-if="column.key === 'platform_fee_cost_layer'">
            <CostLayerTag v-if="(record as any).result" :layer="(record as any).result.platform_fee_cost_layer || 'estimated'" />
            <template v-else>-</template>
          </template>
          <template v-else-if="column.key === 'platform_fee_source'">
            {{ (record as any).result?.platform_fee_source === 'rule' ? '规则库' : (record as any).result ? '手动输入' : '-' }}
          </template>
          <template v-else-if="column.key === 'message'">
            <template v-if="(record as any).error_message">{{ (record as any).error_message }}</template>
            <template v-else>
              {{ [...((record as any).result?.blocking_reasons || []), ...((record as any).result?.warnings || [])].join('；') || '-' }}
            </template>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { message, InputNumber, Input, Select } from 'ant-design-vue'
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
import CostLayerTag from '@/components/CostLayerTag.vue'
import { createListingTasksFromDecisions } from '@/api/modules/listing'

type BatchInputRow = PreListingDecisionBatchItem & {
  key: string
}

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

function onSelectChange(keys: string[]) {
  checkedRowKeys.value = keys
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

async function handlePreviewUpload(options: any) {
  const rawFile = options.file
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
    options.onSuccess()
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
    message.success('导出成功')
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
      const pId = platformByKey.get(item.item_key || '')
      if (!item.result || !item.sku_id || !pId) return null
      return {
        item_key: item.item_key,
        sku_id: item.sku_id,
        platform_id: pId,
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
  return h(InputNumber, {
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
  {
    title: 'SKU ID',
    dataIndex: 'sku_id',
    key: 'sku_id',
    width: 120,
    customRender: ({ record }: any) => renderNumberInput(record, 'sku_id', 1, 0),
  },
  {
    title: '目的国',
    dataIndex: 'destination_country',
    key: 'destination_country',
    width: 110,
    customRender: ({ record }: any) =>
      h(Input, {
        value: record.destination_country,
        maxlength: 10,
        'onUpdate:value': (value: string) => {
          record.destination_country = value
        },
      }),
  },
  {
    title: '目标售价',
    dataIndex: 'target_sale_price',
    key: 'target_sale_price',
    width: 130,
    customRender: ({ record }: any) => renderNumberInput(record, 'target_sale_price', 0.01, 2),
  },
  {
    title: '平台ID',
    dataIndex: 'platform_id',
    key: 'platform_id',
    width: 120,
    customRender: ({ record }: any) => renderNumberInput(record, 'platform_id', 1, 0),
  },
  {
    title: '平台费率%',
    dataIndex: 'platform_fee_pct',
    key: 'platform_fee_pct',
    width: 130,
    customRender: ({ record }: any) => renderNumberInput(record, 'platform_fee_pct', 0, 1),
  },
  {
    title: '支付费率%',
    dataIndex: 'payment_fee_pct',
    key: 'payment_fee_pct',
    width: 130,
    customRender: ({ record }: any) => renderNumberInput(record, 'payment_fee_pct', 0, 1),
  },
  {
    title: '其他费用',
    dataIndex: 'other_fee',
    key: 'other_fee',
    width: 120,
    customRender: ({ record }: any) => renderNumberInput(record, 'other_fee', 0, 2),
  },
  {
    title: '最低利润率%',
    dataIndex: 'minimum_margin_pct',
    key: 'minimum_margin_pct',
    width: 140,
    customRender: ({ record }: any) => renderNumberInput(record, 'minimum_margin_pct', 0, 1),
  },
  {
    title: '货品类型',
    dataIndex: 'cargo_type',
    key: 'cargo_type',
    width: 120,
    customRender: ({ record }: any) =>
      h(Select, {
        value: record.cargo_type,
        options: cargoTypeOptions,
        'onUpdate:value': (value: string) => {
          record.cargo_type = value
        },
      }),
  },
]

const resultColumns = [
  { title: '行号', dataIndex: 'index', key: 'index' },
  { title: 'SKU ID', dataIndex: 'sku_id', key: 'sku_id' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '建议', dataIndex: 'recommendation', key: 'recommendation' },
  { title: '利润率', dataIndex: 'profit_margin', key: 'profit_margin' },
  { title: '利润', dataIndex: 'profit_amount', key: 'profit_amount' },
  { title: '运费来源', dataIndex: 'shipping_cost_layer', key: 'shipping_cost_layer', width: 100 },
  { title: '平台费来源', dataIndex: 'platform_fee_cost_layer', key: 'platform_fee_cost_layer', width: 100 },
  { title: '费用来源', dataIndex: 'platform_fee_source', key: 'platform_fee_source' },
  { title: '原因/错误', dataIndex: 'message', key: 'message' },
]
</script>
