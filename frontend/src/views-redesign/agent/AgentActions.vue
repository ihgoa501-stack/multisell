<template>
  <div>
    <div style="margin-bottom: 16px; display: flex; align-items: center; justify-content: space-between;">
      <div>
        <a-space align="center">
          <h2 style="margin: 0; font-size: 20px; font-weight: 600;">待执行操作</h2>
          <a-badge :count="pendingCount" :overflow-count="99" />
        </a-space>
        <div style="color: rgba(0,0,0,0.45); font-size: 14px;">Agent 决策产生的可执行操作，确认后系统自动执行</div>
      </div>
      <a-button @click="fetchActions" :loading="loading">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <!-- 待处理标签页 -->
    <a-tabs v-model:activeKey="currentTab" @change="onTabChange" style="margin-top: 8px;">
      <a-tab-pane key="pending" tab="待处理">
        <a-table
          :columns="pendingColumns"
          :data-source="pendingActions"
          :loading="loading"
          :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: total }"
          @change="onTableChange"
          row-key="id"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'agent_id'">
              <a-tag :color="tagColor(record.agent_id)">{{ record.agent_id }}</a-tag>
            </template>
            <template v-else-if="column.key === 'action_type'">
              <span style="font-size: 13px;">{{ actionTypeLabel(record.action_type) }}</span>
            </template>
            <template v-else-if="column.key === 'created_at'">
              {{ record.created_at ? new Date(record.created_at).toLocaleString('zh-CN') : '-' }}
            </template>
            <template v-else-if="column.key === 'actions'">
              <a-space>
                <a-button size="small" type="primary" @click="selectedAction = record; showConfirm = true">执行</a-button>
                <a-button size="small" danger @click="rejectAction(record.id)">拒绝</a-button>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
      <a-tab-pane key="history" tab="历史记录">
        <a-table
          :columns="historyColumns"
          :data-source="historyActions"
          :loading="historyLoading"
          :pagination="{ current: historyPagination.page, pageSize: historyPagination.pageSize, total: historyTotal }"
          @change="onHistoryTableChange"
          row-key="id"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'agent_id'">
              <a-tag :color="tagColor(record.agent_id)">{{ record.agent_id }}</a-tag>
            </template>
            <template v-else-if="column.key === 'action_type'">
              <span style="font-size: 13px;">{{ actionTypeLabel(record.action_type) }}</span>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
            </template>
            <template v-else-if="column.key === 'execution_result'">
              {{ record.execution_result ? String(record.execution_result) : '-' }}
            </template>
            <template v-else-if="column.key === 'created_at'">
              {{ record.created_at ? new Date(record.created_at).toLocaleString('zh-CN') : '-' }}
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <!-- 执行确认对话框 -->
    <a-modal v-model:open="showConfirm" title="确认执行操作" @ok="confirmExecute" ok-text="确认执行" cancel-text="取消">
      <p style="margin-bottom: 8px; font-weight: 600;">{{ selectedAction?.summary }}</p>
      <pre style="background: #f5f5f5; padding: 12px; border-radius: 6px; overflow: auto; font-size: 12px;"><code>{{ actionPayloadPreview }}</code></pre>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { agentApi } from '@/api/modules/agent'

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
})

const historyPagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
})

const actionPayloadPreview = computed(() => {
  if (!selectedAction.value?.action_payload) return '{}'
  return JSON.stringify(selectedAction.value.action_payload, null, 2)
})

function tagColor(agentId: string) {
  const colors: Record<string, string> = {
    A3: 'orange', A4: 'green', A5: 'red',
    A6: 'purple', A7: 'blue',
    G1: 'cyan', G2: 'gold', G3: 'red',
  }
  return colors[agentId] || 'default'
}

function statusColor(status: string) {
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
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 80 },
  { title: '操作类型', dataIndex: 'action_type', key: 'action_type', width: 110 },
  { title: '说明', dataIndex: 'summary', key: 'summary', ellipsis: true },
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 180 },
]

// ── 历史列定义 ──

const historyColumns = [
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 80 },
  { title: '类型', dataIndex: 'action_type', key: 'action_type', width: 110 },
  { title: '说明', dataIndex: 'summary', key: 'summary', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '执行结果', dataIndex: 'execution_result', key: 'execution_result', ellipsis: true },
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
]

// ── 数据加载 ──

function onTabChange(key: string) {
  currentTab.value = key
  if (key === 'pending') fetchActions()
  else fetchHistory()
}

function onTableChange(pag: any) {
  pagination.page = pag.current
  fetchActions()
}

function onHistoryTableChange(pag: any) {
  historyPagination.page = pag.current
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
