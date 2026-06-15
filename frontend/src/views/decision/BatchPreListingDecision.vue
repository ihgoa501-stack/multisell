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
          <n-button type="primary" :loading="loading" @click="handleCalculate">批量计算</n-button>
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
import { NInput, NInputNumber, NSelect, NTag, useMessage } from 'naive-ui'
import {
  calculateBatchPreListingDecision,
  type PreListingDecisionBatchItem,
  type PreListingDecisionBatchItemResult,
  type PreListingDecisionBatchResponse,
} from '@/api/modules/decision'

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
