<template>
  <div>
    <!-- Page Header -->
    <div style="margin-bottom: 16px;">
      <a-typography-title :level="4" style="margin-bottom: 4px;">订单管理</a-typography-title>
      <a-typography-text type="secondary">查看和管理所有订单</a-typography-text>
    </div>

    <!-- 筛选栏 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-space align="center">
        <a-select
          v-model:value="query.status"
          :options="statusOptions"
          placeholder="订单状态"
          allow-clear
          style="width: 160px;"
          @change="onStatusChange"
        />
        <a-button @click="fetchData" :loading="loading">查询</a-button>
        <a-button type="link" @click="resetFilter">重置</a-button>
      </a-space>
    </a-card>

    <!-- 表格 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="tablePagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'total_amount'">
            {{ `¥${(record.total_amount || 0).toFixed(2)}` }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="statusTagMap[record.status]?.color || 'default'">
              {{ statusTagMap[record.status]?.label || record.status || '-' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" type="primary" ghost @click="router.push(`/orders/${record.id}`)">
              查看
            </a-button>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import { apiModules } from '@/api'

const router = useRouter()
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

const statusTagMap: Record<string, { color: string; label: string }> = {
  pending: { color: 'orange', label: '待付款' },
  paid: { color: 'blue', label: '待发货' },
  shipped: { color: 'blue', label: '已发货' },
  delivered: { color: 'green', label: '已签收' },
  completed: { color: 'green', label: '已完成' },
  cancelled: { color: 'red', label: '已取消' },
}

const columns = [
  { title: '订单号', dataIndex: 'order_no', key: 'order_no', width: 200, ellipsis: true },
  { title: '商品', dataIndex: 'product_name', key: 'product_name', ellipsis: true },
  { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 80 },
  { title: '金额', dataIndex: 'total_amount', key: 'total_amount', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '下单时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 100 },
]

const tablePagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50', '100'],
})

function onTableChange(pag: any) {
  query.page = pag.current
  query.page_size = pag.pageSize
  fetchData()
}

function onStatusChange() {
  query.page = 1
  tablePagination.current = 1
  fetchData()
}

function resetFilter() {
  query.status = null
  query.page = 1
  tablePagination.current = 1
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = { page: query.page, page_size: query.page_size }
    if (query.status) params.status = query.status
    const res: any = await apiModules.orderApi.list(params)
    data.value = res?.records || res?.data || []
    tablePagination.total = res?.total || res?.count || 0
  } catch (e: any) {
    message.error(e.message || '加载订单列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>
