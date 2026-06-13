<template>
  <div>
    <n-page-header subtitle="查看系统操作记录">
      <template #title>📋 操作日志</template>
    </n-page-header>

    <!-- 筛选 -->
    <n-card style="margin-top: 12px; margin-bottom: 12px;" :bordered="false">
      <n-form inline>
        <n-form-item label="模块">
          <n-select v-model:value="query.module" :options="moduleOptions" clearable placeholder="全部模块" style="width: 150px;" @update:value="search" />
        </n-form-item>
        <n-form-item label="操作">
          <n-input v-model:value="query.action" placeholder="操作关键词" clearable style="width: 150px;" @keyup.enter="search" />
        </n-form-item>
        <n-form-item label="操作人">
          <n-input v-model:value="query.operator" placeholder="操作人" clearable style="width: 120px;" @keyup.enter="search" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="search">搜索</n-button>
          <n-button style="margin-left: 8px;" @click="reset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 表格 -->
    <n-card :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination" @update:page="onPageChange" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { NTag, useMessage } from 'naive-ui'
import { operationLogApi } from '@/api'

const message = useMessage()
const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)
const moduleOptions = ref<any[]>([])

const query = reactive({ module: null as string | null, action: '', operator: '', page: 1, page_size: 20 })

const pagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { query.page = page; fetchData() },
})

const actionTagType: Record<string, string> = {
  '创建': 'success',
  '新增': 'success',
  '更新': 'info',
  '修改': 'info',
  '删除': 'error',
  '导入': 'warning',
  '导出': 'primary',
}

const columns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: '模块', key: 'module', width: 100, render: (row: any) => h(NTag, { size: 'small' }, { default: () => row.module || '-' }) },
  { title: '操作', key: 'action', width: 80, render: (row: any) => {
    const type = actionTagType[row.action] || 'default'
    return h(NTag, { type, size: 'small' }, { default: () => row.action || '-' })
  }},
  { title: '操作内容', key: 'content', ellipsis: { tooltip: true } },
  { title: '资源ID', key: 'resource_id', width: 100 },
  { title: '操作人', key: 'operator', width: 100 },
  { title: 'IP', key: 'ip', width: 130 },
  { title: '耗时(ms)', key: 'duration', width: 90 },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await operationLogApi.list(query)
    data.value = res?.records || []
    total.value = res?.total || 0
    pagination.itemCount = total.value
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
function onPageChange(page: number) { query.page = page; fetchData() }

onMounted(() => { fetchModules(); fetchData() })
</script>
