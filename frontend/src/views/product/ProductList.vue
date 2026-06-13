<template>
  <div>
    <n-page-header subtitle="管理商品信息">
      <template #title>📋 商品列表</template>
      <template #extra>
        <n-space>
          <n-upload :show-file-list="false" accept=".xlsx,.xls" @change="handleImport">
            <n-button>📥 导入</n-button>
          </n-upload>
          <n-button @click="handleExport">📤 导出</n-button>
          <n-button type="primary" @click="router.push('/products/create')">＋ 新增商品</n-button>
        </n-space>
      </template>
    </n-page-header>

    <!-- 搜索栏 -->
    <n-card style="margin-top: 12px; margin-bottom: 12px;" :bordered="false">
      <n-form inline>
        <n-form-item label="商品名称">
          <n-input v-model:value="query.name" placeholder="搜索商品名称" clearable @keyup.enter="search" />
        </n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="query.status" :options="statusOptions" clearable style="width: 120px;" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="search">搜索</n-button>
          <n-button style="margin-left: 8px;" @click="reset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 表格 -->
    <n-card :bordered="false">
      <n-space v-if="checkedRowIds.length > 0" style="margin-bottom: 12px; align-items: center;">
        <span>已选 <b>{{ checkedRowIds.length }}</b> 项</span>
        <n-button size="small" @click="batchUpdateStatus(1)">批量上架</n-button>
        <n-button size="small" @click="batchUpdateStatus(2)">批量下架</n-button>
        <n-button size="small" type="error" ghost @click="batchDelete">批量删除</n-button>
        <n-button size="small" @click="checkedRowIds = []">取消选择</n-button>
      </n-space>
      <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination"
        :row-key="(row: any) => row.id"
        @update:checked-row-keys="checkedRowIds = $event"
        @update:page="onPageChange" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { productApi } from '@/api'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)
const checkedRowIds = ref<number[]>([])

const statusOptions = [
  { label: '草稿', value: 0 },
  { label: '上架', value: 1 },
  { label: '下架', value: 2 },
]

const query = reactive({ name: '', status: null as number | null, page: 1, page_size: 20 })

const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: false,
  itemCount: 0,
  onChange: (page: number) => {
    query.page = page
    fetchData()
  },
})

const columns = [
  { type: 'selection' as const },
  { title: 'ID', key: 'id', width: 70 },
  { title: '商品名称', key: 'name', ellipsis: { tooltip: true } },
  { title: '分类', key: 'category_name' },
  { title: '单位', key: 'unit', width: 60 },
  { title: '状态', key: 'status_name', width: 80, render: (row: any) => {
      const map: Record<number, any> = { 0: { type: 'default', text: '草稿' }, 1: { type: 'success', text: '上架' }, 2: { type: 'warning', text: '下架' } }
      const s = map[row.status] || { type: 'default', text: '未知' }
      return h(NTag, { type: s.type, size: 'small' }, { default: () => s.text })
    }
  },
  { title: 'AI', key: 'ai_status', width: 70, render: (row: any) => {
      if (!row.ai_status) return null
      return h(NTag, { type: row.ai_status === 'completed' ? 'success' : row.ai_status === 'failed' ? 'error' : 'warning', size: 'small' }, { default: () => row.ai_status === 'completed' ? '已优化' : row.ai_status === 'failed' ? '失败' : '待处理' })
    }
  },
  { title: '平台', key: 'platform_statuses', width: 120, render: (row: any) => {
      const ps = row.platform_statuses
      if (!ps || !Object.keys(ps).length) return h('span', { style: 'color:#999;font-size:12px;' }, '未发布')
      return h(NSpace, { size: 'small' }, {
        default: () => Object.entries(ps).map(([platformId, status]) => {
          const platformNames: Record<string, string> = { '1': 'Ozon', '2': 'Shopee' }
          const name = platformNames[platformId] || `平台${platformId}`
          return h(NTag, { size: 'tiny', type: status === 'synced' ? 'success' : 'warning' }, { default: () => name })
        })
      })
    }
  },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 320, render: (row: any) => {
      return h(NSpace, null, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}`) }, { default: () => '详情' }),
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}/edit`) }, { default: () => '编辑' }),
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}/skus`) }, { default: () => 'SKU' }),
          h(NButton, { size: 'small', type: 'info', ghost: true, onClick: () => handleDuplicate(row) }, { default: () => '复制' }),
          h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDelete(row) }, { default: () => '删除' }),
        ]
      })
    }
  },
]

async function fetchData() {
  loading.value = true
  try {
    const res: any = await productApi.list(query)
    data.value = res?.records || []
    total.value = res?.total || 0
    pagination.itemCount = total.value
    pagination.page = query.page
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

function search() { query.page = 1; fetchData() }
function reset() { query.name = ''; query.status = null; query.page = 1; fetchData() }
function onPageChange(page: number) { query.page = page; fetchData() }

function handleDelete(row: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除商品"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await productApi.delete(row.id)
        message.success('删除成功')
        fetchData()
      } catch (e: any) {
        message.error(e.message)
      }
    },
  })
}

async function handleDuplicate(row: any) {
  try {
    const res: any = await http.post(`/products/${row.id}/duplicate`)
    if (res.code === 200) {
      message.success(`已复制为"${res.data.name}"`)
      fetchData()
    }
  } catch (e: any) {
    message.error('复制失败')
  }
}

async function handleExport() {
  try {
    const response = await http.get('/products/export', {
      params: { name: query.name || undefined, status: query.status ?? undefined },
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(new Blob([response as any]))
    const a = document.createElement('a')
    a.href = url
    a.download = `products_${new Date().toISOString().slice(0, 10)}.xlsx`
    a.click()
    window.URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (e: any) {
    message.error('导出失败')
  }
}

async function handleImport({ file }: any) {
  const formData = new FormData()
  formData.append('file', file.file)
  try {
    const res: any = await http.post('/products/import', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    if (res.code === 200) {
      message.success(`成功导入 ${res.data.imported} 个商品`)
      fetchData()
    } else {
      message.error(res.message || '导入失败')
    }
  } catch (e: any) {
    message.error('导入失败: ' + (e.message || ''))
  }
}

async function batchUpdateStatus(status: number) {
  if (!checkedRowIds.value.length) return
  try {
    await http.post('/products/batch/status', { ids: checkedRowIds.value, status })
    message.success(`已更新 ${checkedRowIds.value.length} 个商品状态`)
    checkedRowIds.value = []
    fetchData()
  } catch (e: any) {
    message.error(e.message)
  }
}

function batchDelete() {
  if (!checkedRowIds.value.length) return
  dialog.warning({
    title: '批量删除',
    content: `确定删除选中的 ${checkedRowIds.value.length} 个商品吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await http.post('/products/batch/delete', { ids: checkedRowIds.value })
        message.success(`已删除 ${checkedRowIds.value.length} 个商品`)
        checkedRowIds.value = []
        fetchData()
      } catch (e: any) {
        message.error(e.message)
      }
    },
  })
}

onMounted(fetchData)
</script>
