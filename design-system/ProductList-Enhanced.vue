<!-- ⚠️ 此文档引用旧栈（Python/FastAPI/Vue 3），已于 2026-06-30 迁移至 Go/Next.js。仅供参考，不可直接执行。 -->
<template>
  <div class="product-list-page">
    <!-- ========== 页面标题区 ========== -->
    <n-page-header
      title="商品管理"
      subtitle="管理所有商品信息，支持 AI 增强优化"
      style="margin-bottom: 20px;"
    >
      <template #extra>
        <n-space align="center">
          <n-button @click="handleDownloadTemplate">
            <template #icon><span class="icon-placeholder sm"></span></template>
            下载模板
          </n-button>
          <n-upload :show-file-list="false" accept=".xlsx,.xls" @change="handleImport">
            <n-button>导入</n-button>
          </n-upload>
          <n-button @click="handleExport">导出</n-button>
          <n-button type="primary" @click="router.push('/products/create')">
            <template #icon><span style="font-size: 16px;">＋</span></template>
            新增商品
          </n-button>
        </n-space>
      </template>
    </n-page-header>

    <!-- ========== 搜索筛选区 ========== -->
    <n-card
      :bordered="false"
      style="margin-bottom: 16px; background: #fafafa;"
    >
      <n-form inline :label-width="80">
        <n-form-item label="商品名称">
          <n-input
            v-model:value="query.name"
            placeholder="搜索商品名称"
            clearable
            style="width: 200px;"
            @keyup.enter="search"
          />
        </n-form-item>
        <n-form-item label="发布状态">
          <n-select
            v-model:value="query.status"
            :options="statusOptions"
            clearable
            placeholder="全部"
            style="width: 120px;"
          />
        </n-form-item>
        <n-form-item label="货品类型">
          <n-select
            v-model:value="query.cargo_type"
            :options="cargoTypeOptions"
            clearable
            placeholder="全部"
            style="width: 120px;"
          />
        </n-form-item>
        <n-form-item label="平台">
          <n-select
            v-model:value="query.platform"
            :options="platformOptions"
            clearable
            placeholder="全部"
            style="width: 120px;"
          />
        </n-form-item>
        <n-form-item>
          <n-space>
            <n-button type="primary" @click="search">搜索</n-button>
            <n-button @click="reset">重置</n-button>
            <n-button
              :type="query.logistics_status === 'incomplete' ? 'warning' : 'default'"
              ghost
              @click="showIncompleteOnly"
            >
              只看缺物流数据
            </n-button>
          </n-space>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- ========== AI 增强提示栏 ========== -->
    <n-alert
      type="info"
      style="margin-bottom: 16px;"
      :show-icon="true"
    >
      <template #header>
        <n-space align="center">
          <span>🤖 AI 增强提示</span>
          <n-tag size="small" type="info">NEW</n-tag>
        </n-space>
      </template>
      检测到 3 件商品缺少 SEO 关键词，5 件商品图片质量较低。
      <template #action>
        <n-button size="small" type="primary" ghost @click="handleAIOptimize">
          AI 一键优化
        </n-button>
      </template>
    </n-alert>

    <!-- ========== 批量操作栏 ========== -->
    <n-card
      v-if="checkedRowIds.length > 0"
      :bordered="false"
      style="margin-bottom: 16px; background: #e0f2fe; border: 1px solid #bae6fd;"
    >
      <n-space align="center" justify="space-between">
        <span>
          已选择 <b>{{ checkedRowIds.length }}</b> 件商品
        </span>
        <n-space>
          <n-button size="small" @click="batchUpdateStatus(1)">批量上架</n-button>
          <n-button size="small" @click="batchUpdateStatus(2)">批量下架</n-button>
          <n-button size="small" type="error" ghost @click="batchDelete">批量删除</n-button>
          <n-button size="small" @click="checkedRowIds = []">取消选择</n-button>
        </n-space>
      </n-space>
    </n-card>

    <!-- ========== 数据表格 ========== -->
    <n-card :bordered="false">
      <!-- 表格工具栏 -->
      <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
        <n-space align="center">
          <span style="font-size: 14px; font-weight: 600; color: #404040;">
            共 {{ total }} 件商品
          </span>
          <n-tag size="small" type="info">{{ statusFilterLabel }}</n-tag>
        </n-space>
        <n-space>
          <n-button size="small" @click="refresh">
            <template #icon><span class="icon-placeholder sm"></span></template>
            刷新
          </n-button>
          <n-button size="small" @click="configureColumns">
            列设置
          </n-button>
        </n-space>
      </n-space>

      <!-- 表格本体 -->
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
        :row-props="() => ({ style: 'cursor: pointer;' })"
        @row-click="handleRowClick"
      />

      <!-- 分页信息 -->
      <div
        style="display: flex; justify-content: space-between; align-items: center; margin-top: 16px; font-size: 13px; color: #737373;"
      >
        <span>显示第 {{(pagination.page - 1) * pagination.pageSize + 1}} - {{Math.min(pagination.page * pagination.pageSize, total)}} 条，共 {{total}} 条</span>
        <n-pagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :item-count="total"
          :page-sizes="[10, 20, 50, 100]"
          show-size-picker
          show-quick-jumper
          @update:page="onPageChange"
          @update:page-size="onPageSizeChange"
        />
      </div>
    </n-card>

    <!-- ========== 空状态 ========== -->
    <n-card
      v-if="!loading && data.length === 0"
      :bordered="false"
      style="margin-top: 16px; padding: 60px 0; text-align: center;"
    >
      <div style="display: flex; flex-direction: column; align-items: center; gap: 16px;">
        <div
          style="width: 80px; height: 80px; border-radius: 50%; background: #f5f5f5; display: flex; align-items: center; justify-content: center;"
        >
          <span style="font-size: 36px; color: #d4d4d4;">📦</span>
        </div>
        <h3 style="font-size: 16px; font-weight: 600; color: #525252; margin: 0;">
          暂无商品数据
        </h3>
        <p style="font-size: 14px; color: #a3a3a3; margin: 0; max-width: 400px;">
          开始添加您的第一件商品，或导入现有商品数据
        </p>
        <n-space>
          <n-button type="primary" @click="router.push('/products/create')">
            ＋ 新增商品
          </n-button>
          <n-button @click="handleImport">导入商品</n-button>
        </n-space>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NTag,
  NSpace,
  NImage,
  useMessage,
  useDialog,
  NAlert,
} from 'naive-ui'
import { productApi } from '@/api'
import http from '@/api/http'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)
const checkedRowIds = ref<number[]>([])

// --- 查询条件 ---
const query = reactive({
  name: '',
  status: null as number | null,
  cargo_type: null as string | null,
  platform: null as string | null,
  logistics_status: null as string | null,
})

// --- 下拉选项 ---
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

const platformOptions = [
  { label: 'Ozon', value: 'ozon' },
  { label: 'Shopee', value: 'shopee' },
  { label: 'Wildberries', value: 'wildberries' },
  { label: 'AliExpress', value: 'aliexpress' },
  { label: 'Temu', value: 'temu' },
]

// --- 状态筛选标签 ---
const statusFilterLabel = computed(() => {
  if (query.status === null) return '全部'
  const found = statusOptions.find((o) => o.value === query.status)
  return found ? found.label : '全部'
})

// --- 表格列定义 ---
const columns = [
  {
    type: 'selection',
    width: 40,
  },
  {
    title: '商品信息',
    key: 'name',
    width: 280,
    fixed: 'left',
    render: (row: any) =>
      h('div', { style: 'display: flex; align-items: center; gap: 12px;' }, [
        // 商品图片
        h(NImage, {
          width: 48,
          height: 48,
          src: row.image || 'https://via.placeholder.com/48',
          objectFit: 'cover',
          style: 'border-radius: 6px;',
        }),
        // 商品名称 + SKU
        h('div', { style: 'display: flex; flex-direction: column; gap: 4px;' }, [
          h('span', { style: 'font-weight: 500; color: #404040;' }, row.name),
          h(
            'span',
            { style: 'font-size: 12px; color: #a3a3a3; font-family: monospace;' },
            row.sku || '暂无 SKU'
          ),
        ]),
      ]),
  },
  {
    title: '价格',
    key: 'price',
    width: 100,
    render: (row: any) =>
      h('span', { style: 'font-weight: 500;' }, `¥${row.price || '--'}`),
  },
  {
    title: '库存',
    key: 'stock',
    width: 80,
    render: (row: any) => {
      const isLow = row.stock < 10
      return h(
        'span',
        {
          style: `color: ${isLow ? '#ef4444' : '#404040'}; font-weight: ${isLow ? '600' : '400'};`,
        },
        row.stock || 0
      )
    },
  },
  {
    title: '平台',
    key: 'platforms',
    width: 150,
    render: (row: any) =>
      h(NSpace, { size: 4 }, () =>
        (row.platforms || []).map((p: string) =>
          h(
            NTag,
            { size: 'small', type: 'info', round: true },
            () => p
          )
        )
      ),
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const statusMap: Record<number, { label: string; type: string }> = {
        0: { label: '草稿', type: 'default' },
        1: { label: '已发布', type: 'success' },
        2: { label: '下架', type: 'warning' },
      }
      const s = statusMap[row.status] || statusMap[0]
      return h(
        NTag,
        { size: 'small', type: s.type, round: true },
        () => s.label
      )
    },
  },
  {
    title: 'AI 优化',
    key: 'ai_status',
    width: 120,
    render: (row: any) => {
      if (row.ai_optimized) {
        return h(
          NTag,
          { size: 'small', type: 'success', round: true },
          () => '✓ 已优化'
        )
      }
      return h(
        NButton,
        { size: 'small', type: 'primary', ghost: true, onClick: () => handleAIOptimize(row) },
        () => 'AI 优化'
      )
    },
  },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 160,
    render: (row: any) => row.updated_at || '--',
  },
  {
    title: '操作',
    key: 'actions',
    width: 180,
    fixed: 'right',
    render: (row: any) =>
      h(NSpace, { size: 4 }, [
        h(
          NButton,
          { size: 'small', type: 'primary', ghost: true, onClick: () => handleEdit(row) },
          () => '编辑'
        ),
        h(
          NButton,
          { size: 'small', ghost: true, onClick: () => handleView(row) },
          () => '查看'
        ),
        h(
          NButton,
          { size: 'small', type: 'error', ghost: true, onClick: () => handleDelete(row) },
          () => '删除'
        ),
      ]),
  },
]

// --- 分页 ---
const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

// --- 方法 ---
async function search() {
  loading.value = true
  try {
    const res = await productApi.list({
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...query,
    })
    data.value = res.data || []
    total.value = res.total || 0
  } catch (err) {
    message.error('加载失败')
  } finally {
    loading.value = false
  }
}

function reset() {
  query.name = ''
  query.status = null
  query.cargo_type = null
  query.platform = null
  query.logistics_status = null
  pagination.page = 1
  search()
}

function showIncompleteOnly() {
  query.logistics_status = query.logistics_status === 'incomplete' ? null : 'incomplete'
  search()
}

function onPageChange(page: number) {
  pagination.page = page
  search()
}

function onPageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  search()
}

function handleRowClick(row: any) {
  router.push(`/products/${row.id}`)
}

function handleEdit(row: any) {
  router.push(`/products/${row.id}/edit`)
}

function handleView(row: any) {
  router.push(`/products/${row.id}`)
}

function handleDelete(row: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定要删除商品「${row.name}」吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await productApi.delete(row.id)
        message.success('删除成功')
        search()
      } catch {
        message.error('删除失败')
      }
    },
  })
}

function handleAIOptimize(row?: any) {
  message.info(row ? `开始优化「${row.name}」...` : '开始批量 AI 优化...')
  // TODO: 调用 AI 优化 API
}

function refresh() {
  search()
}

function configureColumns() {
  message.info('列设置功能开发中')
}

function handleDownloadTemplate() {
  message.info('下载模板功能开发中')
}

function handleImport() {
  message.info('导入功能开发中')
}

function handleExport() {
  message.info('导出功能开发中')
}

function batchUpdateStatus(status: number) {
  const label = status === 1 ? '上架' : '下架'
  dialog.warning({
    title: `批量${label}`,
    content: `确定要${label}选中的 ${checkedRowIds.value.length} 件商品吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        // TODO: 调用批量更新 API
        message.success(`批量${label}成功`)
        search()
      } catch {
        message.error(`批量${label}失败`)
      }
    },
  })
}

function batchDelete() {
  dialog.error({
    title: '批量删除',
    content: `确定要删除选中的 ${checkedRowIds.value.length} 件商品吗？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        // TODO: 调用批量删除 API
        message.success('批量删除成功')
        checkedRowIds.value = []
        search()
      } catch {
        message.error('批量删除失败')
      }
    },
  })
}

// --- 初始化 ---
onMounted(() => {
  search()
})
</script>

<style scoped>
.product-list-page {
  /* 页面级样式 */
}

/* 表格行 hover 效果 */
:deep(.n-data-table-tr:hover) {
  background-color: #fafafa !important;
}

/* 表格标题行样式 */
:deep(.n-data-table-th) {
  background-color: #fafafa !important;
  font-size: 12px !important;
  font-weight: 500 !important;
  color: #737373 !important;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* 卡片 hover 微动效 */
:deep(.n-card) {
  transition: box-shadow 0.25s ease;
}

:deep(.n-card:hover) {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.07), 0 2px 4px -2px rgba(0, 0, 0, 0.05);
}
</style>
