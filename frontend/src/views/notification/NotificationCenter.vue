<template>
  <div>
    <n-page-header subtitle="查看和管理系统通知与预警">
      <template #title>🔔 通知中心</template>
      <template #extra>
        <n-space>
          <n-button @click="handleCheck">🔄 触发预警检查</n-button>
          <n-button @click="handleMarkAllRead" :disabled="!hasUnread">全部已读</n-button>
        </n-space>
      </template>
    </n-page-header>

    <n-tabs type="line" default-value="notifications" style="margin-top:12px;">
      <n-tab-pane name="notifications" tab="通知列表">
        <n-card :bordered="false">
          <n-space style="margin-bottom:12px;">
            <n-select v-model:value="filterType" :options="typeOptions" clearable placeholder="通知类型" style="width:160px;" @update:value="fetchData" />
            <n-select v-model:value="filterRead" :options="readOptions" clearable placeholder="阅读状态" style="width:120px;" @update:value="fetchData" />
          </n-space>
          <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination"
            @update:page="onPageChange" />
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="rules" tab="预警规则">
        <n-card :bordered="false">
          <n-space style="margin-bottom:12px;">
            <n-button @click="handleInitRules">初始化默认规则</n-button>
          </n-space>
          <n-data-table :columns="ruleColumns" :data="rules" :loading="rulesLoading" />
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { h, ref, computed, onMounted } from 'vue'
import { NButton, NTag, NSpace, NSwitch, useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import { notificationApi } from '@/api/modules/notification'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const rulesLoading = ref(false)
const data = ref<any[]>([])
const rules = ref<any[]>([])
const filterType = ref<string | null>(null)
const filterRead = ref<string | null>(null)

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

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0,
  onChange: (p: number) => { pagination.value.page = p; fetchData() } })

function onPageChange(p: number) { pagination.value.page = p; fetchData() }

const hasUnread = computed(() => data.value.some((n: any) => !n.is_read))

const columns = [
  { title: '类型', key: 'alert_type', width: 120,
    render: (row: any) => {
      const m: Record<string, string> = { inventory_low_stock: '库存预警', inventory_out_of_stock: '缺货', settlement_pending: '待对账', settlement_discrepancy: '对账差异', listing_failed: '发布失败', order_pending: '订单超时', system: '系统' }
      const typeMap: Record<string, string> = { inventory_low_stock: 'warning', inventory_out_of_stock: 'error', settlement_pending: 'info', settlement_discrepancy: 'error', listing_failed: 'error', order_pending: 'warning', system: 'default' }
      return h(NTag, { type: (typeMap[row.alert_type] || 'info') as any, size: 'small' }, { default: () => m[row.alert_type] || row.alert_type })
    },
  },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  { title: '内容', key: 'content', ellipsis: { tooltip: true }, render: (r: any) => r.content || '-' },
  { title: '状态', key: 'is_read', width: 70,
    render: (row: any) => h(NTag, { type: row.is_read ? 'default' : 'primary', size: 'small' }, { default: () => row.is_read ? '已读' : '未读' }),
  },
  { title: '严重', key: 'severity', width: 70,
    render: (row: any) => {
      const m: Record<string, string> = { info: 'info', warning: 'warning', error: 'error', critical: 'error' }
      return h(NTag, { type: (m[row.severity] || 'default') as any, size: 'small' }, { default: () => row.severity })
    },
  },
  { title: '时间', key: 'created_at', width: 160,
    render: (r: any) => r.created_at ? r.created_at.slice(0, 19).replace('T', ' ') : '-',
  },
  { title: '操作', key: 'actions', width: 120,
    render: (row: any) => h(NSpace, null, {
      default: () => [
        !row.is_read ? h(NButton, { size: 'small', ghost: true, type: 'primary', onClick: () => handleMarkRead(row) }, { default: () => '标已读' }) : null,
        row.link_url ? h(NButton, { size: 'small', ghost: true, onClick: () => router.push(row.link_url) }, { default: () => '查看' }) : null,
        h(NButton, { size: 'small', ghost: true, type: 'error', onClick: () => handleDelete(row) }, { default: () => '删除' }),
      ],
    }),
  },
]

const ruleColumns = [
  { title: '规则名称', key: 'name' },
  { title: '类型', key: 'alert_type' },
  { title: '说明', key: 'description', ellipsis: { tooltip: true } },
  { title: '启用', key: 'enabled', width: 80,
    render: (row: any) => h(NSwitch, {
      value: row.enabled,
      onUpdateValue: (v: boolean) => toggleRule(row.id, v),
    }),
  },
]

async function fetchData() {
  loading.value = true
  try {
    const params: any = { page: pagination.value.page, page_size: pagination.value.pageSize }
    if (filterType.value) params.alert_type = filterType.value
    if (filterRead.value === 'unread') params.unread_only = true
    const res = await notificationApi.list(params)
    const body = res.data
    data.value = body?.data?.records ?? body?.records ?? []
    pagination.value.itemCount = body?.data?.total ?? body?.total ?? 0
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
  dialog.warning({ title: '确认删除', content: '确定删除此通知？', positiveText: '删除', negativeText: '取消',
    onPositiveClick: async () => { try { await notificationApi.delete(row.id); message.success('已删除'); fetchData() } catch { message.error('删除失败') } },
  })
}

async function handleCheck() {
  try { const res = await notificationApi.checkAlerts(); const created = res.data?.data?.created || {}; const keys = Object.keys(created); message.success(keys.length ? `已生成 ${keys.length} 类型预警: ${keys.join(', ')}` : '检查完成，无新预警'); fetchData() }
  catch { message.error('检查失败') }
}

async function handleInitRules() {
  try { const res = await notificationApi.initializeRules(); message.success(`已初始化 ${res.data?.data?.created || 0} 条规则`); fetchRules() }
  catch { message.error('初始化失败') }
}

async function toggleRule(id: number, enabled: boolean) {
  try { await notificationApi.updateRule(id, { enabled }); message.success(enabled ? '已启用' : '已禁用') }
  catch { message.error('操作失败') }
}

onMounted(async () => { await fetchData(); await fetchRules() })
</script>
