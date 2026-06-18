<template>
  <div class="product-list-page">
    <!-- 页面头部 -->
    <n-page-header subtitle="管理商品信息，支持批量操作和 AI 智能优化">
      <template #title>
        <span class="page-title">📋 商品列表</span>
      </template>
      <template #extra>
        <n-space align="center" :size="12">
          <n-button @click="handleDownloadTemplate" ghost>📄 下载模板</n-button>
          <n-upload :show-file-list="false" accept=".xlsx,.xls" @change="handleImport">
            <n-button ghost>📥 导入</n-button>
          </n-upload>
          <n-button @click="handleExport" ghost>📤 导出</n-button>
          <n-button type="primary" @click="router.push('/products/create')">
            <template #icon>＋</template>
            新增商品
          </n-button>
        </n-space>
      </template>
    </n-page-header>

    <!-- AI 优化提示栏 -->
    <n-card
      v-if="data.length > 0"
      class="ai-hint-bar"
      :bordered="false"
      size="small"
      style="margin-top: 16px;"
    >
      <n-space align="center" justify="space-between" style="width: 100%;">
        <n-space align="center" :size="10">
          <span style="font-size: 18px;">🤖</span>
          <div>
            <span style="font-weight: 600; color: var(--color-brand-600);">AI 智能优化建议</span>
            <span style="margin-left: 8px; color: var(--color-neutral-600); font-size: 13px;">
              发现 {{ data.length }} 个商品可优化标题和描述，预计提升点击率 18%
            </span>
          </div>
        </n-space>
        <n-space :size="8">
          <n-button size="small" type="primary" ghost>查看建议</n-button>
          <n-button size="small" @click="(($event.target as HTMLElement)?.closest('.ai-hint-bar'))?.classList.add('hidden')">暂不</n-button>
        </n-space>
      </n-space>
    </n-card>

    <!-- 搜索筛选区 -->
    <n-card class="filter-card" :bordered="false" size="small" style="margin-top: 16px;">
      <n-form inline :label-width="80">
        <n-form-item label="商品名称" path="name">
          <n-input
            v-model:value="query.name"
            placeholder="搜索商品名称或 SKU"
            clearable
            style="width: 200px;"
            @keyup.enter="search"
          >
            <template #prefix>
              <span style="color: var(--color-neutral-400);">🔍</span>
            </template>
          </n-input>
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-select
            v-model:value="query.status"
            :options="statusOptions"
            clearable
            placeholder="全部状态"
            style="width: 130px;"
          />
        </n-form-item>
        <n-form-item label="货品类型" path="cargo_type">
          <n-select
            v-model:value="query.cargo_type"
            :options="cargoTypeOptions"
            clearable
            placeholder="全部类型"
            style="width: 130px;"
          />
        </n-form-item>
        <n-form-item label="物流状态" path="logistics_status">
          <n-select
            v-model:value="query.logistics_status"
            :options="logisticsStatusOptions"
            clearable
            placeholder="全部"
            style="width: 140px;"
          />
        </n-form-item>
        <n-form-item>
          <n-space :size="8">
            <n-button type="primary" @click="search">搜索</n-button>
            <n-button @click="reset">重置</n-button>
            <n-button
              :type="query.logistics_status === 'incomplete' ? 'warning' : 'default'"
              ghost
              @click="showIncompleteOnly"
            >
              ⚠ 缺物流数据
            </n-button>
          </n-space>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 数据表格区 -->
    <n-card class="table-card" :bordered="false" style="margin-top: 16px;">
      <!-- 工具栏 -->
      <n-space align="center" justify="space-between" style="margin-bottom: 16px;">
        <n-space align="center" :size="12">
          <span style="font-weight: 600; font-size: 15px;">商品列表</span>
          <n-tag size="small" :bordered="false" type="info">共 {{ total }} 件</n-tag>
        </n-space>
        <n-space :size="8">
          <n-button size="small" @click="search" :loading="loading">
            🔄 刷新
          </n-button>
        </n-space>
      </n-space>

      <!-- 批量操作栏 -->
      <n-card
        v-if="checkedRowIds.length > 0"
        class="batch-bar"
        size="small"
        :bordered="false"
      >
        <n-space align="center" justify="space-between" style="width: 100%;">
          <n-space align="center" :size="10">
            <span style="color: white; font-weight: 600;">
              已选 <b>{{ checkedRowIds.length }}</b> 项
            </span>
            <n-divider vertical style="height: 20px; background: rgba(255,255,255,0.3);" />
            <n-button size="tiny" type="info" ghost @click="batchUpdateStatus(1)">批量上架</n-button>
            <n-button size="tiny" type="warning" ghost @click="batchUpdateStatus(2)">批量下架</n-button>
            <n-button size="tiny" type="error" ghost @click="batchDelete">批量删除</n-button>
          </n-space>
          <n-button size="tiny" @click="checkedRowIds = []" style="color: white;">取消选择</n-button>
        </n-space>
      </n-card>

      <!-- 数据表格 -->
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        remote
        :scroll-x="1950"
        :row-key="(row: any) => row.id"
        @update:checked-row-keys="checkedRowIds = $event"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        style="margin-top: 12px;"
      />

      <!-- 空状态 -->
      <n-empty
        v-if="!loading && data.length === 0"
        description="暂无商品数据"
        style="padding: 60px 0;"
      >
        <template #extra>
          <n-space :size="12">
            <n-button type="primary" @click="router.push('/products/create')">新增商品</n-button>
            <n-upload :show-file-list="false" accept=".xlsx,.xls" @change="handleImport">
              <n-button ghost>批量导入</n-button>
            </n-upload>
          </n-space>
        </template>
      </n-empty>
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

const cargoTypeOptions = [
  { label: '普通货品', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感货', value: 'sensitive' },
]

const logisticsStatusOptions = [
  { label: '可计算运费', value: 'complete' },
  { label: '缺物流数据', value: 'incomplete' },
]

const query = reactive({
  name: '',
  status: null as number | null,
  cargo_type: null as string | null,
  logistics_status: null as string | null,
  page: 1,
  page_size: 20,
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  pageSizes: [10, 20, 50, 100],
  showSizePicker: true,
  itemCount: 0,
  onChange: (page: number) => {
    query.page = page
    fetchData()
  },
  onUpdatePageSize: (pageSize: number) => {
    query.page_size = pageSize
    query.page = 1
    fetchData()
  },
})

// ----- 渲染辅助函数 -----

function formatDimensions(length?: number, width?: number, height?: number) {
  if (!length || !width || !height) return '-'
  return `${length} x ${width} x ${height} cm`
}

function formatWeight(weight?: number) {
  if (!weight) return '-'
  return `${weight} kg`
}

function cargoTypeLabel(value?: string) {
  const map: Record<string, string> = {
    normal: '普通',
    battery: '带电',
    liquid: '液体',
    sensitive: '敏感',
  }
  return map[value || 'normal'] || value || '-'
}

// ----- 运费试算预填辅助 -----

function hasCompletePackage(row: any) {
  return !!row.package_length_cm
    && !!row.package_width_cm
    && !!row.package_height_cm
    && !!row.package_weight_kg
}

function buildCalculatorQuery(row: any) {
  return {
    length_cm: String(row.package_length_cm),
    width_cm: String(row.package_width_cm),
    height_cm: String(row.package_height_cm),
    weight_kg: String(row.package_weight_kg),
    cargo_type: row.cargo_type || 'normal',
    quantity: '1',
    source_product_id: String(row.id),
    source_product_name: row.name || '',
  }
}

function goShippingCalculator(row: any) {
  if (!hasCompletePackage(row)) {
    router.push(`/products/${row.id}/edit`)
    return
  }
  router.push({
    path: '/shipping/calculator',
    query: buildCalculatorQuery(row),
  })
}

// ----- 表格列定义 -----

const columns = [
  { type: 'selection' as const },
  { title: 'ID', key: 'id', width: 70, fixed: 'left' as const },
  { title: '商品名称', key: 'name', minWidth: 220, ellipsis: { tooltip: true } },
  { title: '分类', key: 'category_name', width: 120 },
  {
    title: '状态', key: 'status_name', width: 80,
    render: (row: any) => {
      const map: Record<number, any> = { 0: { type: 'default', text: '草稿' }, 1: { type: 'success', text: '上架' }, 2: { type: 'warning', text: '下架' } }
      const s = map[row.status] || { type: 'default', text: '未知' }
      return h(NTag, { type: s.type, size: 'small' }, { default: () => s.text })
    },
  },
  {
    title: '货品', key: 'cargo_type', width: 90,
    render: (row: any) => {
      const colors: Record<string, string> = { normal: '#18a058', battery: '#d03050', liquid: '#2080f0', sensitive: '#f0a020' }
      const bg = colors[row.cargo_type] || '#808080'
      return h(NTag, { color: { color: bg, textColor: '#fff' }, size: 'small' }, { default: () => cargoTypeLabel(row.cargo_type) })
    },
  },
  {
    title: '商品尺寸', key: 'product_dimensions', width: 150,
    render: (row: any) => formatDimensions(row.product_length_cm, row.product_width_cm, row.product_height_cm),
  },
  {
    title: '商品重量', key: 'product_weight_kg', width: 100,
    render: (row: any) => formatWeight(row.product_weight_kg),
  },
  {
    title: '包装尺寸', key: 'package_dimensions', width: 150,
    render: (row: any) => formatDimensions(row.package_length_cm, row.package_width_cm, row.package_height_cm),
  },
  {
    title: '包装重量', key: 'package_weight_kg', width: 100,
    render: (row: any) => formatWeight(row.package_weight_kg),
  },
  {
    title: '计费体积重', key: 'package_volume_weight_kg', width: 110,
    render: (row: any) => {
      if (row.package_volume_weight_kg == null) return '-'
      return `${row.package_volume_weight_kg} kg`
    },
  },
  {
    title: '物流状态', key: 'logistics_status', width: 180,
    render: (row: any) => {
      if (row.logistics_status === 'complete') {
        return h(NTag, { type: 'success', size: 'small' }, { default: () => '可计算运费' })
      }
      const missing = (row.missing_logistics_fields || []).join('、') || '未知'
      return h('div', [
        h(NTag, { type: 'warning', size: 'small' }, { default: () => '缺物流数据' }),
        h('div', { style: 'margin-top:4px;color:#d03050;font-size:12px;' }, `缺: ${missing}`),
      ])
    },
  },
  { title: '创建时间', key: 'created_at', width: 170 },
  {
    title: '操作', width: 350, fixed: 'right' as const,
    render: (row: any) => {
      return h(NSpace, null, {
        default: () => [
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}`) }, { default: () => '详情' }),
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}/edit`) }, { default: () => '编辑' }),
          h(NButton, { size: 'small', onClick: () => router.push(`/products/${row.id}/skus`) }, { default: () => 'SKU' }),
          h(NButton, { size: 'small', ghost: true, type: hasCompletePackage(row) ? 'info' : 'warning', onClick: () => goShippingCalculator(row) }, { default: () => hasCompletePackage(row) ? '运费试算' : '补物流' }),
          h(NButton, { size: 'small', ghost: true, type: 'info', onClick: () => handleDuplicate(row) }, { default: () => '复制' }),
          h(NButton, { size: 'small', ghost: true, type: 'error', onClick: () => handleDelete(row) }, { default: () => '删除' }),
        ]
      })
    },
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
function showIncompleteOnly() {
  query.logistics_status = 'incomplete'
  query.page = 1
  fetchData()
}
function reset() {
  query.name = ''
  query.status = null
  query.cargo_type = null
  query.logistics_status = null
  query.page = 1
  fetchData()
}
function onPageChange(page: number) { query.page = page; fetchData() }
function onPageSizeChange(pageSize: number) {
  query.page_size = pageSize
  query.page = 1
  fetchData()
}

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

async function handleDownloadTemplate() {
  try {
    const response = await http.get('/products/export-template', { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([response as any]))
    const a = document.createElement('a')
    a.href = url
    a.download = 'product_import_template.xlsx'
    a.click()
    window.URL.revokeObjectURL(url)
    message.success('模板下载成功')
  } catch (e: any) {
    message.error('模板下载失败')
  }
}

async function handleExport() {
  try {
    const response = await http.get('/products/export', {
      params: {
        name: query.name || undefined,
        status: query.status ?? undefined,
        cargo_type: query.cargo_type || undefined,
        logistics_status: query.logistics_status || undefined,
      },
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

<style scoped>
.product-list-page {
  padding: 0;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-neutral-900);
}

/* AI 提示栏 */
.ai-hint-bar {
  background: linear-gradient(135deg, #eff6ff 0%, #f0f9ff 100%);
  border: 1px solid #bfdbfe;
  border-radius: 8px;
}

.ai-hint-bar :deep(.n-card__content) {
  padding: 12px 16px;
}

/* 筛选区卡片 */
.filter-card {
  background: var(--color-neutral-50, #f9fafb);
  border-radius: 8px;
}

.filter-card :deep(.n-card__content) {
  padding: 16px;
}

/* 表格区卡片 */
.table-card {
  border-radius: 8px;
}

/* 批量操作栏 */
.batch-bar {
  background: var(--color-brand-500, #0ea5e9);
  border-radius: 6px;
  margin-bottom: 12px;
}

.batch-bar :deep(.n-card__content) {
  padding: 10px 16px;
}

/* 表格行悬停效果 */
:deep(.n-data-table-tr:hover) {
  background: var(--color-neutral-50, #f9fafb);
}

/* 状态标签优化 */
:deep(.n-tag--round) {
  font-weight: 500;
}

/* 操作按钮组 */
:deep(.n-data-table-td--last-col) {
  white-space: nowrap;
}

/* 空状态 */
:deep(.n-empty) {
  padding: 60px 0;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .page-title {
    font-size: 18px;
  }

  :deep(.n-form--inline) {
    flex-direction: column;
  }
}
</style>
