<template>
  <div>
    <n-page-header subtitle="集中管理各模块的异常条目">
      <template #title>异常工作台</template>
    </n-page-header>

    <!-- Filters -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space>
        <n-select
          v-model:value="filterModule"
          :options="moduleOptions"
          clearable
          placeholder="模块"
          style="width: 140px;"
          @update:value="fetchItems"
        />
        <n-select
          v-model:value="filterSeverity"
          :options="severityOptions"
          clearable
          placeholder="严重程度"
          style="width: 140px;"
          @update:value="fetchItems"
        />
        <n-select
          v-model:value="filterStatus"
          :options="statusOptions"
          clearable
          placeholder="状态"
          style="width: 140px;"
          @update:value="fetchItems"
        />
      </n-space>
    </n-card>

    <!-- Exception list -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="items" :loading="loading" :pagination="{ pageSize: 20 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import {
  getExceptions,
  assignException,
  resolveException,
  ignoreException,
  type ExceptionItem,
} from '@/api/modules/exceptions'

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const items = ref<ExceptionItem[]>([])
const filterModule = ref<string | null>(null)
const filterSeverity = ref<string | null>(null)
const filterStatus = ref<string | null>(null)

const moduleOptions = [
  { label: '全部', value: '' },
  { label: '上架任务', value: 'listing' },
  { label: '物流账单', value: 'shipping' },
  { label: '平台结算', value: 'settlement' },
  { label: '财务', value: 'finance' },
]

const severityOptions = [
  { label: '全部', value: '' },
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '严重', value: 'critical' },
]

const statusOptions = [
  { label: '全部', value: '' },
  { label: '开放', value: 'open' },
  { label: '已分配', value: 'assigned' },
  { label: '已解决', value: 'resolved' },
  { label: '已忽略', value: 'ignored' },
]

const moduleLabel: Record<string, string> = {
  listing: '上架任务', shipping: '物流账单', settlement: '平台结算', finance: '财务',
}

const severityTag: Record<string, { type: any; text: string }> = {
  low: { type: 'default', text: '低' },
  medium: { type: 'info', text: '中' },
  high: { type: 'warning', text: '高' },
  critical: { type: 'error', text: '严重' },
}

const statusTag: Record<string, { type: any; text: string }> = {
  open: { type: 'error', text: '开放' },
  assigned: { type: 'warning', text: '已分配' },
  resolved: { type: 'success', text: '已解决' },
  ignored: { type: 'default', text: '已忽略' },
}

const columns = [
  { title: '模块', key: 'source_module', width: 90, render: (row: ExceptionItem) => moduleLabel[row.source_module] || row.source_module },
  {
    title: '严重程度',
    key: 'severity',
    width: 90,
    render: (row: ExceptionItem) => {
      const meta = severityTag[row.severity] || { type: 'default', text: row.severity }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: ExceptionItem) => {
      const meta = statusTag[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  { title: '描述', key: 'description', ellipsis: { tooltip: true } },
  { title: '建议操作', key: 'recommended_action', ellipsis: { tooltip: true } },
  { title: '分配给', key: 'assigned_to', width: 100 },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row: ExceptionItem) =>
      h(NSpace, null, {
        default: () => [
          h(NButton, {
            size: 'small',
            onClick: () => handleAssign(row),
            disabled: row.status === 'resolved' || row.status === 'ignored',
          }, { default: () => '分配' }),
          h(NButton, {
            size: 'small',
            type: 'primary',
            onClick: () => handleResolve(row),
            disabled: row.status === 'resolved' || row.status === 'ignored',
          }, { default: () => '解决' }),
          h(NButton, {
            size: 'small',
            onClick: () => handleIgnore(row),
            disabled: row.status === 'resolved' || row.status === 'ignored',
          }, { default: () => '忽略' }),
        ],
      }),
  },
]

async function fetchItems() {
  loading.value = true
  try {
    const params: any = {}
    if (filterModule.value) params.source_module = filterModule.value
    if (filterSeverity.value) params.severity = filterSeverity.value
    if (filterStatus.value) params.status = filterStatus.value
    const resp = await getExceptions(params)
    items.value = resp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function handleAssign(row: ExceptionItem) {
  const name = prompt('分配给（用户名）:', row.assigned_to || '')
  if (!name) return
  try {
    await assignException(row.id, name)
    message.success('已分配')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '分配失败')
  }
}

async function handleResolve(row: ExceptionItem) {
  try {
    await resolveException(row.id, '已处理')
    message.success('已解决')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '解决失败')
  }
}

async function handleIgnore(row: ExceptionItem) {
  try {
    await ignoreException(row.id, '已忽略')
    message.success('已忽略')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '忽略失败')
  }
}

onMounted(fetchItems)
</script>
