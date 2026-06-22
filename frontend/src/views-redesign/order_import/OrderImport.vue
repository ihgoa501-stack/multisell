<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">订单导入</h2>
        <div class="page-header-subtitle">从平台文件或API导入订单到系统</div>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showMockModal = true">生成模拟订单</a-button>
      </div>
    </div>

    <!-- 导入记录 -->
    <a-card title="导入记录" style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'file_name'">
            {{ record.file_name || '-' }}
          </template>
          <template v-else-if="column.key === 'success_count'">
            <span style="color: var(--ant-color-success);">{{ record.success_count }}</span>
          </template>
          <template v-else-if="column.key === 'error_count'">
            <span style="color: var(--ant-color-error);">{{ record.error_count }}</span>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="statusMap[record.status]?.color || 'default'">{{ statusMap[record.status]?.label || record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ record.created_at ? record.created_at.slice(0, 19).replace('T', ' ') : '-' }}
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 模拟弹窗 -->
    <a-modal v-model:open="showMockModal" title="生成模拟订单" :width="380" @ok="handleGenerate" :confirm-loading="generating">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="订单数量">
          <a-input-number v-model:value="mockCount" :min="1" :max="50" style="width: 120px;" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { orderImportApi } from '@/api/modules/orderImport'

const loading = ref(false)
const generating = ref(false)
const data = ref<any[]>([])
const showMockModal = ref(false)
const mockCount = ref(5)

const pagination = ref({ current: 1, pageSize: 20, total: 0 })

const statusMap: Record<string, { label: string; color: string }> = {
  pending: { label: '处理中', color: 'orange' },
  processing: { label: '处理中', color: 'blue' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'red' },
}

const columns = [
  { title: '来源', dataIndex: 'source_type', key: 'source_type', width: 100 },
  { title: '文件名', dataIndex: 'file_name', key: 'file_name', width: 180 },
  { title: '总行数', dataIndex: 'total_rows', key: 'total_rows', width: 80 },
  { title: '成功', dataIndex: 'success_count', key: 'success_count', width: 70 },
  { title: '失败', dataIndex: 'error_count', key: 'error_count', width: 70 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '导入人', dataIndex: 'created_by', key: 'created_by', width: 100 },
  { title: '导入时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
]

function onTableChange(pag: any) {
  pagination.value.current = pag.current
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const res = await orderImportApi.list({ page: pagination.value.current, page_size: pagination.value.pageSize })
    const body = res.data
    data.value = body?.data?.records ?? body?.records ?? []
    pagination.value.total = body?.data?.total ?? body?.total ?? 0
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
