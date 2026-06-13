<template>
  <div>
    <n-page-header subtitle="查看和管理所有订单">
      <template #title>📋 订单管理</template>
    </n-page-header>

    <!-- 筛选栏 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space align="center">
        <n-select
          v-model:value="query.status"
          :options="statusOptions"
          placeholder="订单状态"
          clearable
          style="width: 160px;"
          @update:value="onStatusChange"
        />
        <n-button @click="fetchData" :loading="loading">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <!-- 表格 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        :row-key="(row: any) => row.id"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import { apiModules } from '@/api'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])

const statusOptions = [
  { label: '待付款', value: 'pending' },
  { label: '待发货', value: 'paid' },
  { label: '已发货', value: 'shipped' },
  { label: '已签收', value: 'delivered' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
]

const query = reactive<{ status: string | null; page: number; page_size: number }>({
  status: null,
  page: 1,
  page_size: 20,
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  itemCount: 0,
  pageSizes: [10, 20, 50, 100],
  showSizePicker: true,
  onChange: (page: number) => {
    query.page = page
    fetchData()
  },
  onUpdatePageSize: (pageSize: number) => {
    query.page_size = pageSize
    query.page = 1
    fetchData()
  },
})

const statusTagMap: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default'; label: string }> = {
  pending: { type: 'warning', label: '待付款' },
  paid: { type: 'info', label: '待发货' },
  shipped: { type: 'info', label: '已发货' },
  delivered: { type: 'success', label: '已签收' },
  completed: { type: 'success', label: '已完成' },
  cancelled: { type: 'error', label: '已取消' },
}

const columns = [
  { title: '订单号', key: 'order_no', width: 200, ellipsis: { tooltip: true } },
  { title: '商品', key: 'product_name', ellipsis: { tooltip: true } },
  { title: '数量', key: 'quantity', width: 80 },
  {
    title: '金额',
    key: 'total_amount',
    width: 100,
    render: (row: any) => `¥${(row.total_amount || 0).toFixed(2)}`,
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const cfg = statusTagMap[row.status] || { type: 'default' as const, label: row.status || '-' }
      return h(NTag, { type: cfg.type, size: 'small' }, { default: () => cfg.label })
    },
  },
  { title: '下单时间', key: 'created_at', width: 170 },
  {
    title: '操作',
    width: 100,
    render: (row: any) =>
      h(NButton, { size: 'small', type: 'primary', ghost: true, onClick: () => router.push(`/orders/${row.id}`) }, {
        default: () => '查看',
      }),
  },
]

function onPageChange(page: number) {
  query.page = page
  fetchData()
}

function onPageSizeChange(pageSize: number) {
  query.page_size = pageSize
  query.page = 1
  fetchData()
}

function onStatusChange() {
  query.page = 1
  fetchData()
}

function resetFilter() {
  query.status = null
  query.page = 1
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = { page: query.page, page_size: query.page_size }
    if (query.status) params.status = query.status
    const res: any = await apiModules.orderApi.list(params)
    data.value = res?.records || res?.data || []
    pagination.itemCount = res?.total || res?.count || 0
  } catch (e: any) {
    message.error(e.message || '加载订单列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>
