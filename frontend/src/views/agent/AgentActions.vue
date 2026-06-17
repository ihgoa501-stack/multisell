<template>
  <div>
    <n-page-header subtitle="Agent 决策产生的可执行操作，确认后系统自动执行">
      <template #title>
        <n-space align="center">
          <span>待执行操作</span>
          <n-badge :value="pendingCount" :max="99" />
        </n-space>
      </template>
      <template #extra>
        <n-button @click="fetchActions" :loading="loading" quaternary>
          <template #icon><n-icon><ReloadIcon /></n-icon></template>
          刷新
        </n-button>
      </template>
    </n-page-header>

    <!-- 待处理标签页 -->
    <n-tabs default-value="pending" type="line" @update:value="onTabChange" style="margin-top: 8px;">
      <n-tab-pane name="pending" tab="待处理">
        <n-data-table
          :columns="pendingColumns"
          :data="pendingActions"
          :loading="loading"
          :pagination="pagination"
          @update:page="onPageChange"
        />
      </n-tab-pane>
      <n-tab-pane name="history" tab="历史记录">
        <n-data-table
          :columns="historyColumns"
          :data="historyActions"
          :loading="historyLoading"
          :pagination="historyPagination"
          @update:page="onHistoryPageChange"
        />
      </n-tab-pane>
    </n-tabs>

    <!-- 执行确认对话框 -->
    <n-modal v-model:show="showConfirm" preset="dialog" title="确认执行操作"
      positive-text="确认执行" negative-text="取消"
      @positive-click="confirmExecute"
      @negative-click="showConfirm = false"
    >
      <p style="margin-bottom: 8px; font-weight: 600;">{{ selectedAction?.summary }}</p>
      <n-code :code="actionPayloadPreview" language="json" />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted, computed } from 'vue'
import { useMessage, NTag, NSpace, NButton, NBadge, NIcon, NCode } from 'naive-ui'
import { agentApi } from '@/api/modules/agent'
import { Reload as ReloadIcon } from '@vicons/ionicons5'

const message = useMessage()
const loading = ref(false)
const historyLoading = ref(false)
const showConfirm = ref(false)
const selectedAction = ref<any>(null)
const currentTab = ref('pending')

const pendingActions = ref<any[]>([])
const historyActions = ref<any[]>([])
const pendingCount = ref(0)
const total = ref(0)
const historyTotal = ref(0)

const pagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
  onChange: (page: number) => { pagination.page = page; fetchActions() },
})

const historyPagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
  onChange: (page: number) => { historyPagination.page = page; fetchHistory() },
})

const actionPayloadPreview = computed(() => {
  if (!selectedAction.value?.action_payload) return '{}'
  return JSON.stringify(selectedAction.value.action_payload, null, 2)
})

function tagColor(agentId: string) {
  const colors: Record<string, string> = {
    A3: '#e67e22', A4: '#2ecc71', A5: '#e74c3c',
    A6: '#9b59b6', A7: '#3498db',
    G1: '#1abc9c', G2: '#f39c12', G3: '#e74c3c',
  }
  return colors[agentId] || '#95a5a6'
}

function statusType(status: string) {
  const map: Record<string, string> = {
    pending: 'warning', executed: 'success', rejected: 'default',
    failed: 'error',
  }
  return map[status] || 'default'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    pending: '待执行', executed: '已执行', rejected: '已拒绝',
    failed: '执行失败',
  }
  return map[status] || status
}

function actionTypeLabel(type: string) {
  const map: Record<string, string> = {
    replenish: '补货', adjust_price: '调价', optimize_listing: '优化Listing',
    contact_customer: '联系客户', check_compliance: '合规检查',
    review_discount: '审查折扣', monitor_profit: '利润监控',
    scout_product: '产品侦查',
  }
  return map[type] || type
}

// ── 待处理列定义 ──

const pendingColumns = [
  { title: 'Agent', key: 'agent_id', width: 80,
    render: (row: any) => h(NTag, { size: 'small', color: { color: tagColor(row.agent_id), textColor: '#fff' } }, () => row.agent_id),
  },
  { title: '操作类型', key: 'action_type', width: 110,
    render: (row: any) => h('span', { style: 'font-size: 13px;' }, actionTypeLabel(row.action_type)),
  },
  { title: '说明', key: 'summary', ellipsis: { tooltip: true } },
  { title: '时间', key: 'created_at', width: 170,
    render: (row: any) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-',
  },
  {
    title: '操作', key: 'actions', width: 180,
    render: (row: any) => h(NSpace, {}, {
      default: () => [
        h(NButton, {
          size: 'small', type: 'primary',
          onClick: () => { selectedAction.value = row; showConfirm.value = true },
        }, { default: () => '执行' }),
        h(NButton, {
          size: 'small', type: 'warning',
          onClick: () => rejectAction(row.id),
        }, { default: () => '拒绝' }),
      ],
    }),
  },
]

// ── 历史列定义 ──

const historyColumns = [
  { title: 'Agent', key: 'agent_id', width: 80,
    render: (row: any) => h(NTag, { size: 'small', color: { color: tagColor(row.agent_id), textColor: '#fff' } }, () => row.agent_id),
  },
  { title: '类型', key: 'action_type', width: 110,
    render: (row: any) => h('span', { style: 'font-size: 13px;' }, actionTypeLabel(row.action_type)),
  },
  { title: '说明', key: 'summary', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 100,
    render: (row: any) => h(NTag, { size: 'small', type: statusType(row.status) as any }, () => statusLabel(row.status)),
  },
  { title: '执行结果', key: 'execution_result', ellipsis: { tooltip: true },
    render: (row: any) => row.execution_result ? String(row.execution_result) : '-',
  },
  { title: '时间', key: 'created_at', width: 170,
    render: (row: any) => row.created_at ? new Date(row.created_at).toLocaleString('zh-CN') : '-',
  },
]

// ── 数据加载 ──

function onTabChange(value: string) {
  currentTab.value = value
  if (value === 'pending') fetchActions()
  else fetchHistory()
}

function onPageChange(page: number) {
  pagination.page = page
  fetchActions()
}

function onHistoryPageChange(page: number) {
  historyPagination.page = page
  fetchHistory()
}

async function fetchActions() {
  loading.value = true
  try {
    const res = await agentApi.listActions({ status: 'pending', page: pagination.page, page_size: pagination.pageSize })
    if (res.data) {
      pendingActions.value = res.data.records || []
      pendingCount.value = res.data.total || 0
      pagination.itemCount = res.data.total || 0
    }
  } catch (e: any) {
    message.error('加载待执行操作失败: ' + (e.message || ''))
  } finally {
    loading.value = false
  }
}

async function fetchHistory() {
  historyLoading.value = true
  try {
    const res = await agentApi.listActions({ status: 'all', page: historyPagination.page, page_size: historyPagination.pageSize })
    if (res.data) {
      historyActions.value = res.data.records?.filter((a: any) => a.status !== 'pending') || []
      historyTotal.value = res.data.total || 0
      historyPagination.itemCount = res.data.total || 0
    }
  } catch (e: any) {
    message.error('加载历史记录失败: ' + (e.message || ''))
  } finally {
    historyLoading.value = false
  }
}

async function confirmExecute() {
  if (!selectedAction.value) return
  try {
    await agentApi.executeAction(selectedAction.value.id)
    message.success('操作已执行')
    showConfirm.value = false
    selectedAction.value = null
    await fetchActions()
    await fetchHistory()
  } catch (e: any) {
    message.error('执行失败: ' + (e.message || ''))
  }
}

async function rejectAction(actionId: number) {
  try {
    await agentApi.rejectAction(actionId)
    message.success('操作已拒绝')
    await fetchActions()
    await fetchHistory()
  } catch (e: any) {
    message.error('拒绝失败: ' + (e.message || ''))
  }
}

onMounted(() => {
  fetchActions()
})
</script>
