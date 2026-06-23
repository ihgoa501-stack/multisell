<template>
  <div>
    <n-page-header subtitle="平台导入的退货记录">
      <template #title>退货管理</template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space style="margin-bottom: 12px;">
        <n-select
          v-model:value="statusFilter"
          :options="statusOptions"
          clearable
          placeholder="退货状态"
          style="width: 160px;"
          @update:value="load"
        />
        <n-button :loading="loading" @click="load">刷新</n-button>
      </n-space>

      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NCard, NPageHeader, NSelect, NSpace, NTag, useMessage } from 'naive-ui'
import { fetchReturns } from '@/api/modules/aftersales'

const msg = useMessage()
const loading = ref(false)
const data = ref<any[]>([])
const statusFilter = ref<string | null>(null)

const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '已审批', value: 'approved' },
  { label: '已驳回', value: 'rejected' },
  { label: '已入库', value: 'received' },
  { label: '已退款', value: 'refunded' },
]

const statusTagMap: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default'; text: string }> = {
  pending: { type: 'warning', text: '待处理' },
  approved: { type: 'info', text: '已审批' },
  rejected: { type: 'error', text: '已驳回' },
  received: { type: 'success', text: '已入库' },
  refunded: { type: 'success', text: '已退款' },
}

const columns = [
  { title: '订单号', key: 'order_id', width: 100 },
  { title: 'SKU', key: 'sku_id', width: 90 },
  { title: '数量', key: 'return_quantity', width: 70 },
  { title: '原因', key: 'reason', ellipsis: { tooltip: true } },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const meta = statusTagMap[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 170,
    render: (row: any) => row.created_at || '-',
  },
]

async function load() {
  loading.value = true
  try {
    const res: any = await fetchReturns({ status: statusFilter.value || undefined })
    data.value = res.data || []
  } catch (err: any) {
    msg.error(err?.message || '加载退货列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
:deep(.n-page-header) {
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-neutral-200, #e5e5e5);
  margin-bottom: 16px;
}

:deep(.n-card) {
  border-radius: 8px;
}
</style>
