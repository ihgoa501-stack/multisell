<template>
  <div>
    <!-- Page Header -->
    <div style="margin-bottom: 16px;">
      <a-typography-title :level="4" style="margin-bottom: 4px;">上架任务队列</a-typography-title>
      <a-typography-text type="secondary">从上架决策生成的待发布任务</a-typography-text>
    </div>

    <a-card style="margin-top: 12px;" :bordered="false">
      <a-space style="margin-bottom: 12px;">
        <a-select
          v-model:value="statusFilter"
          :options="statusOptions"
          allow-clear
          placeholder="任务状态"
          style="width: 180px;"
        />
        <a-button type="primary" :loading="loading" @click="fetchTasks">查询</a-button>
      </a-space>

      <a-table
        :columns="columns"
        :data-source="tasks"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'target_profit_margin'">
            {{ record.target_profit_margin == null ? '-' : `${record.target_profit_margin}%` }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="taskStatusColor[record.status] || 'default'">
              {{ taskStatusText[record.status] || record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'missing'">
            {{ record.last_error || record.missing_requirements.join('；') || '-' }}
          </template>
          <template v-else-if="column.key === 'task_actions'">
            <a-space>
              <a-button
                size="small"
                @click="handleRecheck(record)"
                :disabled="record.status === 'published' || record.status === 'cancelled'"
              >
                重检
              </a-button>
              <a-button
                size="small"
                type="primary"
                @click="handlePublish(record)"
                :disabled="record.status !== 'ready'"
              >
                发布
              </a-button>
              <a-button
                size="small"
                @click="handleCancel(record)"
                :disabled="record.status === 'published' || record.status === 'cancelled'"
              >
                取消
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  cancelListingTask,
  getListingTasks,
  publishListingTask,
  recheckListingTask,
  type ListingTask,
} from '@/api/modules/listing'

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

const taskStatusColor: Record<string, string> = {
  ready: 'green',
  blocked: 'orange',
  published: 'blue',
  failed: 'red',
  cancelled: 'default',
}

const taskStatusText: Record<string, string> = {
  ready: '可发布',
  blocked: '阻塞',
  published: '已发布',
  failed: '失败',
  cancelled: '已取消',
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
  { title: '商品', dataIndex: 'product_name', key: 'product_name', ellipsis: true },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name', width: 120 },
  { title: '目的国', dataIndex: 'destination_country', key: 'destination_country', width: 90 },
  { title: '目标售价', dataIndex: 'target_sale_price', key: 'target_sale_price', width: 110 },
  { title: '利润率', dataIndex: 'target_profit_margin', key: 'target_profit_margin', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '缺失项/错误', dataIndex: 'missing', key: 'missing', ellipsis: true },
  { title: '操作', key: 'task_actions', width: 260 },
]

onMounted(fetchTasks)
</script>
