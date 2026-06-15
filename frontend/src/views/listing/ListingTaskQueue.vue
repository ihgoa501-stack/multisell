<template>
  <div>
    <n-page-header subtitle="从上架决策生成的待发布任务">
      <template #title>上架任务队列</template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space style="margin-bottom: 12px;">
        <n-select
          v-model:value="statusFilter"
          :options="statusOptions"
          clearable
          placeholder="任务状态"
          style="width: 180px;"
        />
        <n-button type="primary" :loading="loading" @click="fetchTasks">查询</n-button>
      </n-space>

      <n-data-table :columns="columns" :data="tasks" :loading="loading" :pagination="{ pageSize: 20 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import {
  cancelListingTask,
  getListingTasks,
  publishListingTask,
  recheckListingTask,
  type ListingTask,
} from '@/api/modules/listing'

const message = useMessage()
const loading = ref(false)
const tasks = ref<ListingTask[]>([])
const statusFilter = ref<string | null>(null)

const statusOptions = [
  { label: '可发布', value: 'ready' },
  { label: '阻塞', value: 'blocked' },
  { label: '已发布', value: 'published' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
]

const statusTag: Record<string, { type: 'success' | 'warning' | 'error' | 'default' | 'info'; text: string }> = {
  ready: { type: 'success', text: '可发布' },
  blocked: { type: 'warning', text: '阻塞' },
  published: { type: 'info', text: '已发布' },
  failed: { type: 'error', text: '失败' },
  cancelled: { type: 'default', text: '已取消' },
}

async function fetchTasks() {
  loading.value = true
  try {
    const resp = await getListingTasks({ status: statusFilter.value || undefined })
    tasks.value = resp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询上架任务失败')
  } finally {
    loading.value = false
  }
}

async function handleRecheck(row: ListingTask) {
  try {
    await recheckListingTask(row.id)
    message.success('检查完成')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '检查失败')
  }
}

async function handlePublish(row: ListingTask) {
  try {
    await publishListingTask(row.id)
    message.success('发布完成')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '发布失败')
  }
}

async function handleCancel(row: ListingTask) {
  try {
    await cancelListingTask(row.id)
    message.success('已取消')
    await fetchTasks()
  } catch (err: any) {
    message.error(err?.message || '取消失败')
  }
}

const columns = [
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '平台', key: 'platform_name', width: 120 },
  { title: '目的国', key: 'destination_country', width: 90 },
  { title: '目标售价', key: 'target_sale_price', width: 110 },
  {
    title: '利润率',
    key: 'target_profit_margin',
    width: 100,
    render: (row: ListingTask) => row.target_profit_margin == null ? '-' : `${row.target_profit_margin}%`,
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: ListingTask) => {
      const meta = statusTag[row.status] || { type: 'default' as const, text: row.status }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '缺失项/错误',
    key: 'missing',
    ellipsis: { tooltip: true },
    render: (row: ListingTask) => row.last_error || row.missing_requirements.join('；') || '-',
  },
  {
    title: '操作',
    key: 'actions',
    width: 260,
    render: (row: ListingTask) =>
      h(NSpace, null, {
        default: () => [
          h(NButton, {
            size: 'small',
            onClick: () => handleRecheck(row),
            disabled: row.status === 'published' || row.status === 'cancelled',
          }, { default: () => '重检' }),
          h(NButton, {
            size: 'small',
            type: 'primary',
            onClick: () => handlePublish(row),
            disabled: row.status !== 'ready',
          }, { default: () => '发布' }),
          h(NButton, {
            size: 'small',
            onClick: () => handleCancel(row),
            disabled: row.status === 'published' || row.status === 'cancelled',
          }, { default: () => '取消' }),
        ],
      }),
  },
]

onMounted(fetchTasks)
</script>
