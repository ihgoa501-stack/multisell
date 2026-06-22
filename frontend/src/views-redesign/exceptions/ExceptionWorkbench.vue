<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">异常工作台</h2>
        <div class="page-header-subtitle">集中管理各模块的异常条目</div>
      </div>
    </div>

    <!-- Filters -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-space>
        <a-select
          v-model:value="filterModule"
          :options="moduleOptions"
          allow-clear
          placeholder="模块"
          style="width: 140px;"
          @change="fetchItems"
        />
        <a-select
          v-model:value="filterSeverity"
          :options="severityOptions"
          allow-clear
          placeholder="严重程度"
          style="width: 140px;"
          @change="fetchItems"
        />
        <a-select
          v-model:value="filterStatus"
          :options="statusOptions"
          allow-clear
          placeholder="状态"
          style="width: 140px;"
          @change="fetchItems"
        />
      </a-space>
    </a-card>

    <!-- Exception list -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table :columns="columns" :data-source="items" :loading="loading" :pagination="{ pageSize: 20 }" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source_module'">
            {{ moduleLabel[record.source_module] || record.source_module }}
          </template>
          <template v-else-if="column.key === 'severity'">
            <a-tag :color="severityTag[record.severity]?.color || 'default'">{{ severityTag[record.severity]?.text || record.severity }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="statusTag[record.status]?.color || 'default'">{{ statusTag[record.status]?.text || record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button
                size="small"
                @click="handleAssign(record)"
                :disabled="record.status === 'resolved' || record.status === 'ignored'"
              >分配</a-button>
              <a-button
                size="small"
                type="primary"
                @click="handleResolve(record)"
                :disabled="record.status === 'resolved' || record.status === 'ignored'"
              >解决</a-button>
              <a-button
                size="small"
                @click="handleIgnore(record)"
                :disabled="record.status === 'resolved' || record.status === 'ignored'"
              >忽略</a-button>
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
  getExceptions,
  assignException,
  resolveException,
  ignoreException,
  type ExceptionItem,
} from '@/api/modules/exceptions'

const loading = ref(false)
const items = ref<ExceptionItem[]>([])
const filterModule = ref<string | undefined>(undefined)
const filterSeverity = ref<string | undefined>(undefined)
const filterStatus = ref<string | undefined>(undefined)

const moduleOptions = [
  { label: '全部', value: '' },
  { label: '上架任务', value: 'listing' },
  { label: '物流账单', value: 'shipping' },
  { label: '平台结算', value: 'settlement' },
  { label: '财务', value: 'finance' },
]

const severityOptions = [
  { label: '全部', value: '' },
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '严重', value: 'critical' },
]

const statusOptions = [
  { label: '全部', value: '' },
  { label: '开放', value: 'open' },
  { label: '已分配', value: 'assigned' },
  { label: '已解决', value: 'resolved' },
  { label: '已忽略', value: 'ignored' },
]

const moduleLabel: Record<string, string> = {
  listing: '上架任务', shipping: '物流账单', settlement: '平台结算', finance: '财务',
}

const severityTag: Record<string, { color: string; text: string }> = {
  low: { color: 'default', text: '低' },
  medium: { color: 'blue', text: '中' },
  high: { color: 'orange', text: '高' },
  critical: { color: 'red', text: '严重' },
}

const statusTag: Record<string, { color: string; text: string }> = {
  open: { color: 'red', text: '开放' },
  assigned: { color: 'orange', text: '已分配' },
  resolved: { color: 'success', text: '已解决' },
  ignored: { color: 'default', text: '已忽略' },
}

const columns = [
  { title: '模块', dataIndex: 'source_module', key: 'source_module', width: 90 },
  { title: '严重程度', dataIndex: 'severity', key: 'severity', width: 90 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '建议操作', dataIndex: 'recommended_action', key: 'recommended_action', ellipsis: true },
  { title: '分配给', dataIndex: 'assigned_to', key: 'assigned_to', width: 100 },
  { title: '操作', key: 'actions', width: 200 },
]

async function fetchItems() {
  loading.value = true
  try {
    const params: any = {}
    if (filterModule.value) params.source_module = filterModule.value
    if (filterSeverity.value) params.severity = filterSeverity.value
    if (filterStatus.value) params.status = filterStatus.value
    const resp = await getExceptions(params)
    items.value = resp.data || []
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

async function handleAssign(row: ExceptionItem) {
  const name = prompt('分配给（用户名）:', row.assigned_to || '')
  if (!name) return
  try {
    await assignException(row.id, name)
    message.success('已分配')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '分配失败')
  }
}

async function handleResolve(row: ExceptionItem) {
  try {
    await resolveException(row.id, '已处理')
    message.success('已解决')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '解决失败')
  }
}

async function handleIgnore(row: ExceptionItem) {
  try {
    await ignoreException(row.id, '已忽略')
    message.success('已忽略')
    await fetchItems()
  } catch (err: any) {
    message.error(err?.message || '忽略失败')
  }
}

onMounted(fetchItems)
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
</style>
