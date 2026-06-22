<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">通知中心</h2>
        <div class="page-header-subtitle">查看和管理系统通知与预警</div>
      </div>
      <div class="page-header-extra">
        <a-space>
          <a-button @click="handleCheck">触发预警检查</a-button>
          <a-button @click="handleMarkAllRead" :disabled="!hasUnread">全部已读</a-button>
        </a-space>
      </div>
    </div>

    <a-tabs v-model:activeKey="activeTab" style="margin-top: 12px;">
      <a-tab-pane key="notifications" tab="通知列表">
        <a-card :bordered="false">
          <a-space style="margin-bottom: 12px;">
            <a-select v-model:value="filterType" :options="typeOptions" allow-clear placeholder="通知类型" style="width: 160px;" @change="fetchData" />
            <a-select v-model:value="filterRead" :options="readOptions" allow-clear placeholder="阅读状态" style="width: 120px;" @change="fetchData" />
          </a-space>
          <a-table
            :columns="columns"
            :data-source="data"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="onTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'alert_type'">
                <a-tag :color="alertTypeColorMap[record.alert_type] || 'blue'">{{ alertTypeLabelMap[record.alert_type] || record.alert_type }}</a-tag>
              </template>
              <template v-else-if="column.key === 'is_read'">
                <a-tag :color="record.is_read ? 'default' : 'blue'">{{ record.is_read ? '已读' : '未读' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'severity'">
                <a-tag :color="severityColorMap[record.severity] || 'default'">{{ record.severity }}</a-tag>
              </template>
              <template v-else-if="column.key === 'created_at'">
                {{ record.created_at ? record.created_at.slice(0, 19).replace('T', ' ') : '-' }}
              </template>
              <template v-else-if="column.key === 'content'">
                {{ record.content || '-' }}
              </template>
              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button v-if="!record.is_read" size="small" type="primary" ghost @click="handleMarkRead(record)">标已读</a-button>
                  <a-button v-if="record.link_url" size="small" ghost @click="router.push(record.link_url)">查看</a-button>
                  <a-button size="small" danger ghost @click="handleDelete(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="rules" tab="预警规则">
        <a-card :bordered="false">
          <a-space style="margin-bottom: 12px;">
            <a-button @click="handleInitRules">初始化默认规则</a-button>
          </a-space>
          <a-table :columns="ruleColumns" :data-source="rules" :loading="rulesLoading" row-key="id">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'enabled'">
                <a-switch :checked="record.enabled" @change="(v: boolean) => toggleRule(record.id, v)" />
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import { notificationApi } from '@/api/modules/notification'

const router = useRouter()

const activeTab = ref('notifications')
const loading = ref(false)
const rulesLoading = ref(false)
const data = ref<any[]>([])
const rules = ref<any[]>([])
const filterType = ref<string | undefined>(undefined)
const filterRead = ref<string | undefined>(undefined)

const typeOptions = [
  { label: '库存预警', value: 'inventory_low_stock' },
  { label: '缺货', value: 'inventory_out_of_stock' },
  { label: '结算待对账', value: 'settlement_pending' },
  { label: '对账差异', value: 'settlement_discrepancy' },
  { label: '发布失败', value: 'listing_failed' },
  { label: '订单超时', value: 'order_pending' },
]
const readOptions = [
  { label: '未读', value: 'unread' },
  { label: '已读', value: 'read' },
]

const pagination = ref({ current: 1, pageSize: 20, total: 0 })

const alertTypeLabelMap: Record<string, string> = {
  inventory_low_stock: '库存预警',
  inventory_out_of_stock: '缺货',
  settlement_pending: '待对账',
  settlement_discrepancy: '对账差异',
  listing_failed: '发布失败',
  order_pending: '订单超时',
  system: '系统',
}
const alertTypeColorMap: Record<string, string> = {
  inventory_low_stock: 'orange',
  inventory_out_of_stock: 'red',
  settlement_pending: 'blue',
  settlement_discrepancy: 'red',
  listing_failed: 'red',
  order_pending: 'orange',
  system: 'default',
}
const severityColorMap: Record<string, string> = {
  info: 'blue',
  warning: 'orange',
  error: 'red',
  critical: 'red',
}

const hasUnread = computed(() => data.value.some((n: any) => !n.is_read))

const columns = [
  { title: '类型', dataIndex: 'alert_type', key: 'alert_type', width: 120 },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: '状态', dataIndex: 'is_read', key: 'is_read', width: 70 },
  { title: '严重', dataIndex: 'severity', key: 'severity', width: 70 },
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160 },
  { title: '操作', key: 'actions', width: 160 },
]

const ruleColumns = [
  { title: '规则名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'alert_type', key: 'alert_type' },
  { title: '说明', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 80 },
]

function onTableChange(pag: any) {
  pagination.value.current = pag.current
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const params: any = { page: pagination.value.current, page_size: pagination.value.pageSize }
    if (filterType.value) params.alert_type = filterType.value
    if (filterRead.value === 'unread') params.unread_only = true
    const res = await notificationApi.list(params)
    const body = res.data
    data.value = body?.data?.records ?? body?.records ?? []
    pagination.value.total = body?.data?.total ?? body?.total ?? 0
  } catch { message.error('加载失败') }
  finally { loading.value = false }
}

async function fetchRules() {
  rulesLoading.value = true
  try {
    const res = await notificationApi.listRules()
    rules.value = res.data?.data ?? []
  } catch {}
  finally { rulesLoading.value = false }
}

async function handleMarkRead(row: any) {
  try { await notificationApi.markRead(row.id); row.is_read = true; message.success('已标记') }
  catch { message.error('操作失败') }
}

async function handleMarkAllRead() {
  try { await notificationApi.markAllRead(); message.success('已全部标记已读'); fetchData() }
  catch { message.error('操作失败') }
}

async function handleDelete(row: any) {
  Modal.confirm({
    title: '确认删除',
    content: '确定删除此通知？',
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try { await notificationApi.delete(row.id); message.success('已删除'); fetchData() }
      catch { message.error('删除失败') }
    },
  })
}

async function handleCheck() {
  try {
    const res = await notificationApi.checkAlerts()
    const created = res.data?.data?.created || {}
    const keys = Object.keys(created)
    message.success(keys.length ? `已生成 ${keys.length} 类型预警: ${keys.join(', ')}` : '检查完成，无新预警')
    fetchData()
  } catch { message.error('检查失败') }
}

async function handleInitRules() {
  try {
    const res = await notificationApi.initializeRules()
    message.success(`已初始化 ${res.data?.data?.created || 0} 条规则`)
    fetchRules()
  } catch { message.error('初始化失败') }
}

async function toggleRule(id: number, enabled: boolean) {
  try { await notificationApi.updateRule(id, { enabled }); message.success(enabled ? '已启用' : '已禁用') }
  catch { message.error('操作失败') }
}

onMounted(async () => { await fetchData(); await fetchRules() })
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 4px;
}
.page-header-content {
  flex: 1;
}
.page-header-title {
  margin: 0 0 4px 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  color: var(--ant-color-text-secondary);
  font-size: 14px;
}
.page-header-extra {
  display: flex;
  align-items: center;
}
</style>
