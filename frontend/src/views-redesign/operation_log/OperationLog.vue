<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">操作日志</h2>
        <div class="page-header-subtitle">查看系统操作记录</div>
      </div>
    </div>

    <!-- 筛选 -->
    <a-card style="margin-top: 12px; margin-bottom: 12px;" :bordered="false">
      <a-form layout="inline">
        <a-form-item label="模块">
          <a-select v-model:value="query.module" :options="moduleOptions" allow-clear placeholder="全部模块" style="width: 150px;" @change="search" />
        </a-form-item>
        <a-form-item label="操作">
          <a-input v-model:value="query.action" placeholder="操作关键词" allow-clear style="width: 150px;" @press-enter="search" />
        </a-form-item>
        <a-form-item label="操作人">
          <a-input v-model:value="query.operator" placeholder="操作人" allow-clear style="width: 120px;" @press-enter="search" />
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="search">搜索</a-button>
            <a-button @click="reset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 表格 -->
    <a-card :bordered="false">
      <a-table :columns="columns" :data-source="data" :loading="loading" :pagination="pagination" row-key="id" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'module'">
            <a-tag>{{ record.module || '-' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-tag :color="actionTagColor[record.action] || 'default'">{{ record.action || '-' }}</a-tag>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { operationLogApi } from '@/api'

const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)
const moduleOptions = ref<any[]>([])

const query = reactive({ module: null as string | null, action: '', operator: '', page: 1, page_size: 20 })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
})

const actionTagColor: Record<string, string> = {
  '创建': 'success',
  '新增': 'success',
  '更新': 'blue',
  '修改': 'blue',
  '删除': 'red',
  '导入': 'orange',
  '导出': 'blue',
}

const columns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '模块', dataIndex: 'module', key: 'module', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 80 },
  { title: '操作内容', dataIndex: 'content', key: 'content', ellipsis: true },
  { title: '资源ID', dataIndex: 'resource_id', key: 'resource_id', width: 100 },
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 100 },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 130 },
  { title: '耗时(ms)', dataIndex: 'duration', key: 'duration', width: 90 },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await operationLogApi.list(query)
    data.value = res?.records || []
    total.value = res?.total || 0
    pagination.total = total.value
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

async function fetchModules() {
  try {
    const res: any = await operationLogApi.getModules()
    moduleOptions.value = (res.data || []).map((m: string) => ({ label: m, value: m }))
  } catch { /* ignore */ }
}

function search() { query.page = 1; fetchData() }
function reset() { query.module = null; query.action = ''; query.operator = ''; query.page = 1; fetchData() }

function onTableChange(pag: any) {
  query.page = pag.current
  pagination.current = pag.current
  fetchData()
}

onMounted(() => { fetchModules(); fetchData() })
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
