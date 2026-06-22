<template>
  <div>
    <n-page-header subtitle="上架任务条目详情" @back="goBack">
      <template #title>{{ task?.name || '上架任务详情' }}</template>
      <template #extra>
        <n-tag v-if="task" :type="taskStatusTag(task.status).type" size="small">
          {{ taskStatusTag(task.status).text }}
        </n-tag>
      </template>
    </n-page-header>

    <n-card v-if="task" style="margin-top: 12px;" :bordered="false" size="small">
      <n-descriptions :column="4" size="small" bordered>
        <n-descriptions-item label="总数">{{ task.total_count }}</n-descriptions-item>
        <n-descriptions-item label="成功">{{ task.success_count }}</n-descriptions-item>
        <n-descriptions-item label="失败">{{ task.failed_count }}</n-descriptions-item>
        <n-descriptions-item label="创建时间">{{ task.created_at || '-' }}</n-descriptions-item>
      </n-descriptions>
    </n-card>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space style="margin-bottom: 12px;">
        <n-select
          v-model:value="statusFilter"
          :options="itemStatusOptions"
          clearable
          placeholder="条目状态"
          style="width: 160px;"
          @update:value="load"
        />
        <n-button :loading="loading" @click="load">刷新</n-button>
        <n-button
          v-if="hasFailedItems"
          type="warning"
          secondary
          :loading="retryingAll"
          @click="handleRetryAllFailed"
        >
          {{ retryingAll ? '重置中...' : '重试所有失败项' }}
        </n-button>
      </n-space>

      <n-data-table
        :columns="columns"
        :data="items"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NPageHeader,
  NSelect,
  NSpace,
  NTag,
  useMessage,
} from 'naive-ui'
import { fetchListingTaskDetail, retryListingItem, retryAllFailed } from '@/api/modules/listingTask'

const route = useRoute()
const router = useRouter()
const msg = useMessage()

const loading = ref(false)
const task = ref<any>(null)
const items = ref<any[]>([])
const statusFilter = ref<string | null>(null)
const retryingIds = ref<Set<number>>(new Set())
const retryingAll = ref(false)

const hasFailedItems = computed(() => items.value.some((i: any) => i.status === 'failed'))

const itemStatusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '进行中', value: 'in_progress' },
  { label: '成功', value: 'success' },
  { label: '失败', value: 'failed' },
]

function taskStatusTag(status: string): { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string } {
  const map: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string }> = {
    pending: { type: 'default', text: '待处理' },
    in_progress: { type: 'info', text: '进行中' },
    completed: { type: 'success', text: '已完成' },
    partial_failed: { type: 'warning', text: '部分失败' },
    all_failed: { type: 'error', text: '全部失败' },
  }
  return map[status] || { type: 'default', text: status }
}

function itemStatusTag(status: string): { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string } {
  const map: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string }> = {
    pending: { type: 'default', text: '待处理' },
    in_progress: { type: 'info', text: '进行中' },
    success: { type: 'success', text: '成功' },
    failed: { type: 'error', text: '失败' },
  }
  return map[status] || { type: 'default', text: status }
}

const columns = [
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '平台', key: 'platform_name', width: 120 },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const meta = itemStatusTag(row.status)
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '重试次数',
    key: 'retry_count',
    width: 90,
  },
  {
    title: '错误',
    key: 'error_message',
    ellipsis: { tooltip: true },
    render: (row: any) => row.error_message || '-',
  },
  {
    title: '操作',
    key: 'actions',
    width: 120,
    render: (row: any) => {
      if (row.status !== 'failed') return null
      return h(NButton, {
        size: 'small',
        type: 'primary',
        loading: retryingIds.value.has(row.id),
        onClick: () => handleRetry(row.id),
      }, { default: () => '重试' })
    },
  },
]

async function load() {
  const taskId = Number(route.params.id)
  if (!taskId) return
  loading.value = true
  try {
    const res = await fetchListingTaskDetail(taskId)
    task.value = res.data
    if (res.data.items) {
      items.value = statusFilter.value
        ? res.data.items.filter((i: any) => i.status === statusFilter.value)
        : res.data.items
    } else {
      items.value = []
    }
  } catch (err: any) {
    msg.error(err?.message || '加载任务详情失败')
  } finally {
    loading.value = false
  }
}

async function handleRetry(itemId: number) {
  const taskId = Number(route.params.id)
  retryingIds.value.add(itemId)
  try {
    await retryListingItem(taskId, itemId)
    msg.success('已加入重试队列')
    await load()
  } catch (err: any) {
    msg.error(err?.message || '重试失败')
  } finally {
    retryingIds.value.delete(itemId)
  }
}

async function handleRetryAllFailed() {
  const taskId = Number(route.params.id)
  retryingAll.value = true
  try {
    const res = await retryAllFailed(taskId)
    msg.success(`已重置 ${res.data?.reset_count || 0} 个失败条目`)
    await load()
  } catch (err: any) {
    msg.error(err?.message || '重试失败')
  } finally {
    retryingAll.value = false
  }
}

function goBack() {
  router.push({ name: 'ListingTaskQueue' })
}

onMounted(load)
</script>

<style scoped></style>
