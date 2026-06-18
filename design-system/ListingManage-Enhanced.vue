<template>
  <div class="listing-container">
    <!-- ═══ 页面标题区 ═══ -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">🌐 多平台刊登管理</h1>
        <p class="page-subtitle">管理商品在各平台的发布状态 · 一键批量刊登</p>
      </div>
      <div class="page-header-right">
        <n-button @click="router.push('/ai-listing')" type="primary">
          <template #icon><n-icon :component="SparklesOutline" /></template>
          AI 智能刊登
        </n-button>
        <n-button @click="fetchData" :loading="loading">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <!-- ═══ 平台发布概览卡片 ═══ -->
      <div class="platform-overview-grid">
        <div class="platform-card" v-for="p in platformStats" :key="p.code" @click="filterByPlatform(p.code)">
          <div class="platform-card-header">
            <div class="platform-badge" :style="{ background: platformColor(p.code) }">
              {{ p.code.toUpperCase() }}
            </div>
            <div class="platform-info">
              <div class="platform-name">{{ p.name }}</div>
              <div class="platform-count">{{ p.total }} 个商品</div>
            </div>
            <n-tag size="small" :type="p.syncRate >= 80 ? 'success' : p.syncRate >= 50 ? 'warning' : 'error'" round>
              {{ p.syncRate }}% 同步
            </n-tag>
          </div>
          <div class="platform-card-bar">
            <div class="platform-bar-fill" :style="{ width: p.syncRate + '%', background: platformColor(p.code) }"></div>
          </div>
        </div>
      </div>

      <!-- ═══ 筛选区 ═══ -->
      <div class="filter-bar">
        <div class="filter-left">
          <n-input v-model:value="searchText" placeholder="搜索商品名称..." clearable style="width: 240px;" size="small">
            <template #prefix><n-icon :component="SearchOutline" /></template>
          </n-input>
          <n-select v-model:value="filterPlatform" :options="platformOptions" placeholder="所有平台" clearable style="width: 160px;" size="small" />
          <n-select v-model:value="filterStatus" :options="statusOptions" placeholder="所有状态" clearable style="width: 140px;" size="small" />
        </div>
        <div class="filter-right">
          <n-button size="small" @click="handleBatchPublish" :disabled="!selectedRowKeys.length">
            <template #icon><n-icon :component="SendOutline" /></template>
            批量刊登 ({{ selectedRowKeys.length }})
          </n-button>
        </div>
      </div>

      <!-- ═══ 刊登列表表格 ═══ -->
      <div class="table-card">
        <div class="table-header">
          <h3 class="card-title">
            刊登列表
            <n-tag size="small" :bordered="false" type="info">{{ filteredData.length }} 条</n-tag>
          </h3>
          <div class="table-actions">
            <n-button size="tiny" quaternary @click="clearSelection">清除选择</n-button>
          </div>
        </div>
        <n-data-table
          :columns="columns"
          :data="filteredData"
          :bordered="false"
          :single-line="false"
          size="small"
          :max-height="520"
          v-model:checked-row-keys="selectedRowKeys"
          :row-key="(row: any) => row.id"
        />
      </div>

      <!-- ═══ 刊登任务队列 ═══ -->
      <div class="task-panel">
        <div class="card-header">
          <h3 class="card-title">
            <n-icon :component="TimeOutline" size="16" />
            刊登任务队列
            <n-tag size="small" :bordered="false" type="warning">{{ pendingTasks.length }} 个待处理</n-tag>
          </h3>
          <n-button size="small" @click="router.push('/listing-tasks')">查看全部 →</n-button>
        </div>
        <div v-if="pendingTasks.length" class="task-list">
          <div v-for="task in pendingTasks.slice(0, 5)" :key="task.id" class="task-item">
            <div class="task-icon" :class="'task-' + task.status">
              <n-icon :component="task.status === 'pending' ? TimeOutline : task.status === 'running' ? SyncOutline : CheckmarkDoneOutline" size="16" />
            </div>
            <div class="task-info">
              <div class="task-title">{{ task.product_name }} → {{ task.platform_name }}</div>
              <div class="task-meta">{{ task.status_label }} · {{ formatTime(task.created_at) }}</div>
            </div>
            <n-tag size="tiny" :type="task.status === 'pending' ? 'warning' : task.status === 'running' ? 'info' : 'success'" round>
              {{ task.status === 'pending' ? '等待中' : task.status === 'running' ? '执行中' : '完成' }}
            </n-tag>
          </div>
        </div>
        <n-empty v-else description="暂无待处理任务" />
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { Component } from 'vue'
import http from '@/api/http'

// Icons
import {
  SparklesOutline,
  RefreshOutline,
  SearchOutline,
  SendOutline,
  TimeOutline,
  SyncOutline,
  CheckmarkDoneOutline,
  CheckmarkCircleOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const searchText = ref('')
const filterPlatform = ref<number | null>(null)
const filterStatus = ref<string | null>(null)
const selectedRowKeys = ref<number[]>([])

// ── 模拟数据 ──
const data = ref<any[]>([
  { id: 1, product_name: '无线蓝牙耳机 Pro', platform_name: 'OZON', platform_code: 'ozon', platform_product_id: 'OZ-12345', status: 'synced', platform_url: 'https://ozon.ru/product/12345', last_sync_at: '2024-01-15 10:30:00' },
  { id: 2, product_name: '无线蓝牙耳机 Pro', platform_name: 'Shopee', platform_code: 'shopee', platform_product_id: 'SP-67890', status: 'synced', platform_url: 'https://shopee.com/product/67890', last_sync_at: '2024-01-15 09:20:00' },
  { id: 3, product_name: '智能手表 SE', platform_name: 'OZON', platform_code: 'ozon', platform_product_id: '', status: 'pending', platform_url: '', last_sync_at: '' },
  { id: 4, product_name: '智能手表 SE', platform_name: 'Wildberries', platform_code: 'wb', platform_product_id: 'WB-11111', status: 'failed', platform_url: '', last_sync_at: '2024-01-14 15:45:00' },
  { id: 5, product_name: '便携式充电宝 10000mAh', platform_name: 'OZON', platform_code: 'ozon', platform_product_id: 'OZ-22222', status: 'synced', platform_url: 'https://ozon.ru/product/22222', last_sync_at: '2024-01-15 08:10:00' },
])

const pendingTasks = ref<any[]>([
  { id: 1, product_name: '智能手表 SE', platform_name: 'OZON', status: 'pending', status_label: '等待发布', created_at: '2024-01-15T11:00:00' },
  { id: 2, product_name: '智能手表 SE', platform_name: 'Wildberries', status: 'running', status_label: '发布中', created_at: '2024-01-15T10:30:00' },
  { id: 3, product_name: '便携式充电宝', platform_name: 'Shopee', status: 'pending', status_label: '等待发布', created_at: '2024-01-15T10:00:00' },
])

// ── 平台统计 ──
const platformStats = computed(() => {
  const platforms = [
    { code: 'ozon', name: 'OZON', total: 0, synced: 0 },
    { code: 'shopee', name: 'Shopee', total: 0, synced: 0 },
    { code: 'wb', name: 'Wildberries', total: 0, synced: 0 },
  ]
  
  data.value.forEach(item => {
    const p = platforms.find(p => p.code === item.platform_code)
    if (p) {
      p.total++
      if (item.status === 'synced') p.synced++
    }
  })
  
  return platforms.map(p => ({
    ...p,
    syncRate: p.total > 0 ? Math.round(p.synced / p.total * 100) : 0,
  }))
})

// ── 筛选选项 ──
const platformOptions = [
  { label: 'OZON', value: 'ozon' },
  { label: 'Shopee', value: 'shopee' },
  { label: 'Wildberries', value: 'wb' },
]

const statusOptions = [
  { label: '已发布', value: 'synced' },
  { label: '发布中', value: 'pending' },
  { label: '失败', value: 'failed' },
  { label: '草稿', value: 'draft' },
]

// ── 筛选后的数据 ──
const filteredData = computed(() => {
  let result = data.value
  
  if (searchText.value) {
    result = result.filter(item => item.product_name?.toLowerCase().includes(searchText.value.toLowerCase()))
  }
  
  if (filterPlatform.value) {
    result = result.filter(item => item.platform_code === filterPlatform.value)
  }
  
  if (filterStatus.value) {
    result = result.filter(item => item.status === filterStatus.value)
  }
  
  return result
})

// ── 表格列 ──
const columns = [
  { type: 'selection' as const },
  { title: '商品名称', key: 'product_name', ellipsis: { tooltip: true }, width: 200 },
  { title: '平台', key: 'platform_name', width: 120, render: (row: any) => h(NTag, { size: 'small', color: { color: platformColor(row.platform_code), textColor: '#fff' }, { default: () => row.platform_name }) },
  { title: '平台商品ID', key: 'platform_product_id', ellipsis: { tooltip: true }, width: 150 },
  { title: '状态', key: 'status', width: 100, render: (row: any) => {
    const statusMap: Record<string, { type: string; text: string }> = {
      'synced': { type: 'success', text: '已发布' },
      'pending': { type: 'warning', text: '发布中' },
      'failed': { type: 'error', text: '失败' },
      'draft': { type: 'default', text: '草稿' },
    }
    const s = statusMap[row.status] || { type: 'default', text: row.status }
    return h(NTag, { type: s.type as any, size: 'small' }, { default: () => s.text })
  }},
  { title: '平台链接', key: 'platform_url', ellipsis: { tooltip: true }, width: 200, render: (row: any) => row.platform_url ? h('a', { href: row.platform_url, target: '_blank', style: 'color: #0ea5e9; font-size: 12px;' }, '查看商品') : '-' },
  { title: '上次同步', key: 'last_sync_at', width: 160 },
  { title: '操作', width: 180, render: (row: any) => h(NSpace, null, { default: () => [
    h(NButton, { size: 'tiny', type: 'primary', onClick: () => handleRepublish(row) }, { default: () => '重新发布' }),
    h(NButton, { size: 'tiny', onClick: () => router.push(`/products/${row.product_id}`) }, { default: () => '商品' }),
  ]})},
]

// ── 辅助函数 ──
function platformColor(code: string): string {
  const m: Record<string, string> = {
    ozon: '#005bff', shopee: '#ee4d2d', wb: '#cb11ab',
    wildberries: '#cb11ab', aliexpress: '#e62e04', temu: '#e0120c',
  }
  return m[code.toLowerCase()] || '#2080f0'
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

function filterByPlatform(code: string) {
  filterPlatform.value = filterPlatform.value === code ? null : code as any
}

function clearSelection() {
  selectedRowKeys.value = []
}

async function handleBatchPublish() {
  if (!selectedRowKeys.value.length) {
    message.warning('请先选择要刊登的商品')
    return
  }
  message.success(`已提交 ${selectedRowKeys.value.length} 个商品的刊登任务`)
  selectedRowKeys.value = []
}

async function handleRepublish(row: any) {
  try {
    await http.post(`/products/${row.product_id}/publish/${row.platform_id}`)
    message.success('重新发布成功')
    fetchData()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '发布失败')
  }
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await http.get('/listings')
    data.value = res.data || []
  } catch (e: any) {
    message.error(e?.response?.data?.message || '获取数据失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
/* ═══ 设计系统 Token 应用 ═══ */
.listing-container {
  padding: 24px;
  max-width: 1440px;
  margin: 0 auto;
  background: #f8fafc;
  min-height: 100vh;
}

/* ═══ 页面标题 ═══ */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e2e8f0;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px 0;
}
.page-subtitle {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
}

/* ═══ 平台概览网格 ═══ */
.platform-overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}
.platform-card {
  background: white;
  border-radius: 12px;
  padding: 16px;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  transition: all 0.2s ease;
}
.platform-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
.platform-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.platform-badge {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 11px;
  font-weight: 800;
  flex-shrink: 0;
}
.platform-info { flex: 1; }
.platform-name { font-size: 14px; font-weight: 600; color: #1e293b; }
.platform-count { font-size: 12px; color: #94a3b8; margin-top: 2px; }
.platform-card-bar {
  height: 4px;
  background: #f1f5f9;
  border-radius: 2px;
  overflow: hidden;
}
.platform-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.6s ease;
}

/* ═══ 筛选区 ═══ */
.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 14px 18px;
  background: white;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
  margin-bottom: 20px;
}
.filter-left {
  display: flex;
  gap: 10px;
  align-items: center;
}
.filter-right {
  display: flex;
  gap: 8px;
}

/* ═══ 表格卡片 ═══ */
.table-card {
  background: white;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
  margin-bottom: 20px;
}
.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.card-title {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* ═══ 任务面板 ═══ */
.task-panel {
  background: white;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.task-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.task-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
  transition: all 0.15s;
}
.task-item:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}
.task-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.task-pending { background: #fff7ed; color: #f59e0b; }
.task-running { background: #eff6ff; color: #0ea5e9; }
.task-done { background: #f0fdf4; color: #10b981; }
.task-info { flex: 1; }
.task-title { font-size: 13px; font-weight: 600; color: #1e293b; }
.task-meta { font-size: 11px; color: #94a3b8; margin-top: 2px; }

/* ═══ 响应式 ═══ */
@media (max-width: 1280px) {
  .platform-overview-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 960px) {
  .platform-overview-grid { grid-template-columns: 1fr; }
  .filter-bar { flex-direction: column; }
}
</style>
