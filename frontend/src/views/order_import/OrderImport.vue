<template>
  <div>
    <n-page-header subtitle="从平台文件或API导入订单到系统">
      <template #title>📥 订单导入</template>
      <template #extra>
        <n-button type="primary" @click="showMockModal = true">🎲 生成模拟订单</n-button>
      </template>
    </n-page-header>

    <!-- 导入记录 -->
    <n-card title="导入记录" style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination"
        @update:page="onPageChange" />
    </n-card>

    <!-- 模拟弹窗 -->
    <n-modal v-model:show="showMockModal" title="生成模拟订单" preset="card" style="width: 380px;">
      <n-form label-placement="left" label-width="100px">
        <n-form-item label="订单数量">
          <n-input-number v-model:value="mockCount" :min="1" :max="50" style="width: 120px;" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showMockModal = false">取消</n-button>
          <n-button type="primary" :loading="generating" @click="handleGenerate">生成并导入</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import { orderImportApi } from '@/api/modules/orderImport'

const message = useMessage()
const loading = ref(false)
const generating = ref(false)
const data = ref<any[]>([])
const showMockModal = ref(false)
const mockCount = ref(5)

const pagination = ref({ page: 1, pageSize: 20, itemCount: 0,
  onChange: (p: number) => { pagination.value.page = p; fetchData() } })

function onPageChange(p: number) {
  pagination.value.page = p
  fetchData()
}

const columns = [
  { title: '来源', key: 'source_type', width: 100 },
  { title: '文件名', key: 'file_name', width: 180, render: (r: any) => r.file_name || '-' },
  { title: '总行数', key: 'total_rows', width: 80 },
  { title: '成功', key: 'success_count', width: 70,
    render: (r: any) => h('span', { style: 'color: #18a058' }, r.success_count) },
  { title: '失败', key: 'error_count', width: 70,
    render: (r: any) => h('span', { style: 'color: #d03050' }, r.error_count) },
  { title: '状态', key: 'status', width: 100,
    render: (r: any) => {
      const m: Record<string, any> = { pending: { label: '处理中', type: 'warning' }, processing: { label: '处理中', type: 'info' }, completed: { label: '已完成', type: 'success' }, failed: { label: '失败', type: 'error' } }
      const s = m[r.status] ?? { label: r.status, type: 'default' }
      return h(NTag, { type: s.type, size: 'small' }, { default: () => s.label })
    },
  },
  { title: '导入人', key: 'created_by', width: 100 },
  { title: '导入时间', key: 'created_at', width: 170,
    render: (r: any) => r.created_at ? r.created_at.slice(0, 19).replace('T', ' ') : '-' },
]

async function fetchData() {
  loading.value = true
  try {
    const res = await orderImportApi.list({ page: pagination.value.page, page_size: pagination.value.pageSize })
    const body = res.data
    data.value = body?.data?.records ?? body?.records ?? []
    pagination.value.itemCount = body?.data?.total ?? body?.total ?? 0
  } catch { message.error('加载失败') }
  finally { loading.value = false }
}

async function handleGenerate() {
  generating.value = true
  try {
    const res = await orderImportApi.generateMock({ platform_id: 1, count: mockCount.value })
    message.success(`导入成功: ${res.data?.data?.success ?? 0} 条`)
    showMockModal.value = false
    fetchData()
  } catch (err: any) { message.error('生成失败: ' + (err.message || '')) }
  finally { generating.value = false }
}

onMounted(fetchData)
</script>
