<!-- ================================================
   FLOW: Product Management
   SCREEN 1 of 4: Product List
   ------------------------------------------------
   ENTRY:  Sidebar → 商品中心 → 商品管理
   EXIT:   Click row → Product Detail Sheet / Click + New → Create Form
   BRANCH: Multi-select → Batch Action Bar
   ================================================ -->
<template>
  <div class="product-page">
    <!-- ═══ Page Header ═══ -->
    <div class="page-header">
      <div>
        <h1 class="page-title">商品管理</h1>
        <p class="page-subtitle">管理所有商品、SKU 和多平台上架状态</p>
      </div>
      <a-button type="primary" @click="showCreateDrawer = true">
        <template #icon><PlusOutlined /></template>
        新增商品
      </a-button>
    </div>

    <!-- ═══ Search & Filters ═══ -->
    <a-card :bordered="false" class="filter-card">
      <div class="filter-row">
        <a-input-search
          v-model:value="searchQuery"
          placeholder="搜索商品名称、SKU..."
          style="width: 280px"
          allow-clear
          @search="handleSearch"
        />
        <a-space>
          <a-select
            v-model:value="statusFilter"
            placeholder="状态"
            allow-clear
            style="width: 120px"
            :options="statusOptions"
            @change="applyFilters"
          />
          <a-select
            v-model:value="categoryFilter"
            placeholder="分类"
            allow-clear
            style="width: 140px"
            :options="categoryOptions"
            @change="applyFilters"
          />
          <a-select
            v-model:value="platformFilter"
            placeholder="平台"
            allow-clear
            style="width: 120px"
            :options="platformOptions"
            @change="applyFilters"
          />
          <a-button @click="resetFilters">
            <template #icon><ReloadOutlined /></template>
            重置
          </a-button>
        </a-space>
      </div>
    </a-card>

    <!-- ═══ Product Table ═══ -->
    <a-card :bordered="false" class="table-card" style="margin-top: 12px">
      <a-table
        :columns="columns"
        :data-source="filteredProducts"
        :row-selection="{ selectedRowKeys, onChange: onSelectChange }"
        :pagination="{ pageSize: 10, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 个商品` }"
        row-key="id"
        :loading="loading"
        @row-click="openProductDetail"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div class="product-name-cell">
              <a-avatar
                :size="36"
                shape="square"
                :style="{ backgroundColor: '#f0f4ff', color: '#2962FF', fontSize: '14px' }"
              >
                {{ record.name.charAt(0) }}
              </a-avatar>
              <div class="product-name-info">
                <div class="product-name-text">{{ record.name }}</div>
                <div class="product-name-sub">{{ record.name_en || '-' }}</div>
              </div>
            </div>
          </template>

          <template v-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)" :bordered="false">
              {{ statusLabel(record.status) }}
            </a-tag>
          </template>

          <template v-if="column.key === 'sku_count'">
            <span>{{ record.sku_count }} SKU</span>
            <span v-if="record.on_shelf_count < record.sku_count" style="color: var(--ant-color-text-tertiary); font-size: 11px; margin-left: 4px">
              ({{ record.on_shelf_count }} 上架)
            </span>
          </template>

          <template v-if="column.key === 'revenue'">
            <span v-if="record.total_revenue > 0" style="font-weight: 500">
              ¥{{ record.total_revenue.toLocaleString() }}
            </span>
            <span v-else style="color: var(--ant-color-text-quaternary)">-</span>
          </template>

          <template v-if="column.key === 'platforms'">
            <a-space :size="4">
              <a-tag
                v-for="p in record.platforms"
                :key="p"
                :color="platformColor(p)"
                :bordered="false"
                size="small"
                style="font-size: 10px; text-transform: uppercase"
              >
                {{ p }}
              </a-tag>
              <span v-if="!record.platforms.length" style="color: var(--ant-color-text-quaternary); font-size: 12px">未上架</span>
            </a-space>
          </template>

          <template v-if="column.key === 'action'">
            <a-dropdown :trigger="['click']">
              <a-button type="text" size="small">
                <MoreOutlined />
              </a-button>
              <template #overlay>
                <a-menu @click="({ key }: any) => handleRowAction(key, record)">
                  <a-menu-item key="view"><EyeOutlined /> 查看详情</a-menu-item>
                  <a-menu-item key="edit"><EditOutlined /> 编辑</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="archive" danger><StopOutlined /> 归档</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- ═══ Batch Action Bar (Screen 4) ═══ -->
    <transition name="slide-up">
      <div v-if="selectedRowKeys.length > 0" class="batch-bar">
        <div class="batch-info">
          <a-checkbox
            :checked="isAllSelected"
            :indeterminate="isIndeterminate"
            @change="toggleSelectAll"
          />
          <span class="batch-count">已选 {{ selectedRowKeys.length }} 项</span>
        </div>
        <a-space>
          <a-button v-for="action in batchActions" :key="action.key"
            :danger="action.danger"
            @click="handleBatchAction(action.key)"
          >
            <template #icon><component :is="getBatchIcon(action.icon)" /></template>
            {{ action.label }}
          </a-button>
          <a-button @click="selectedRowKeys = []">取消选择</a-button>
        </a-space>
      </div>
    </transition>

    <!-- ═══ Product Detail Sheet (Screen 2) ═══ -->
    <a-drawer
      v-model:open="showDetailSheet"
      :title="selectedProduct?.name || '商品详情'"
      placement="right"
      width="560"
    >
      <template #extra>
        <a-space v-if="selectedProduct">
          <a-button @click="showCreateDrawer = true; showDetailSheet = false">
            <template #icon><EditOutlined /></template>
            编辑
          </a-button>
        </a-space>
      </template>

      <div v-if="selectedProduct">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="状态">
            <a-tag :color="statusColor(selectedProduct.status)">{{ statusLabel(selectedProduct.status) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="英文名">{{ selectedProduct.name_en || '-' }}</a-descriptions-item>
          <a-descriptions-item label="分类">{{ selectedProduct.category_name }}</a-descriptions-item>
          <a-descriptions-item label="品牌">{{ selectedProduct.brand_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="SKU 数量">{{ selectedProduct.sku_count }}</a-descriptions-item>
          <a-descriptions-item label="已上架">{{ selectedProduct.on_shelf_count }} / {{ selectedProduct.sku_count }}</a-descriptions-item>
          <a-descriptions-item label="总销量">{{ selectedProduct.total_sales }}</a-descriptions-item>
          <a-descriptions-item label="总收入">¥{{ selectedProduct.total_revenue.toLocaleString() }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ selectedProduct.created_at.slice(0, 10) }}</a-descriptions-item>
          <a-descriptions-item label="更新时间">{{ selectedProduct.updated_at.slice(0, 10) }}</a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <h4>上架平台</h4>
        <a-space v-if="selectedProduct.platforms.length" :size="8">
          <a-tag
            v-for="p in selectedProduct.platforms"
            :key="p"
            :color="platformColor(p)"
            :bordered="false"
          >
            {{ p.toUpperCase() }}
          </a-tag>
        </a-space>
        <a-empty v-else description="暂未上架任何平台" :image-style="{ height: '40px' }" />

        <a-divider />

        <a-space direction="vertical" style="width: 100%" :size="8">
          <a-tooltip title="即将上线，敬请期待">
            <a-button block disabled>
              <template #icon><DatabaseOutlined /></template>
              管理 SKU & 库存
            </a-button>
          </a-tooltip>
          <a-tooltip title="即将上线，敬请期待">
            <a-button block disabled>
              <template #icon><DollarOutlined /></template>
              价格管理
            </a-button>
          </a-tooltip>
          <a-tooltip title="即将上线，敬请期待">
            <a-button block disabled>
              <template #icon><UploadOutlined /></template>
              发布到平台
            </a-button>
          </a-tooltip>
        </a-space>
      </div>
    </a-drawer>

    <!-- ═══ Create/Edit Product Form (Screen 3) ═══ -->
    <a-drawer
      v-model:open="showCreateDrawer"
      :title="isEditing ? '编辑商品' : '新增商品'"
      placement="right"
      width="560"
      :body-style="{ paddingBottom: '80px' }"
    >
      <a-form
        :model="formState"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <!-- 基本信息 -->
        <a-card size="small" title="基本信息" :bordered="true" style="margin-bottom: 16px; border-radius: 8px">
          <a-form-item label="商品名称" name="name" :rules="[{ required: true, message: '请输入商品名称' }]">
            <a-input v-model:value="formState.name" placeholder="如：蓝牙音箱 Pro Max" />
          </a-form-item>
          <a-form-item label="英文名称" name="name_en">
            <a-input v-model:value="formState.name_en" placeholder="English name (optional)" />
          </a-form-item>
        </a-card>

        <!-- 分类 & 品牌 -->
        <a-card size="small" title="分类与品牌" :bordered="true" style="margin-bottom: 16px; border-radius: 8px">
          <a-form-item label="分类" name="category" :rules="[{ required: true, message: '请选择分类' }]">
            <a-select v-model:value="formState.category" placeholder="选择分类" :options="categoryOptions" />
          </a-form-item>
          <a-form-item label="品牌" name="brand">
            <a-input v-model:value="formState.brand" placeholder="品牌名称（可选）" />
          </a-form-item>
        </a-card>

        <!-- 图片 -->
        <a-card size="small" title="商品图片" :bordered="true" style="margin-bottom: 16px; border-radius: 8px">
          <a-form-item label="封面图" name="cover">
            <a-upload
              v-model:file-list="formState.fileList"
              list-type="picture-card"
              :max-count="1"
              :before-upload="() => false"
            >
              <div>
                <PlusOutlined />
                <div style="margin-top: 8px">上传</div>
              </div>
            </a-upload>
          </a-form-item>
        </a-card>
      </a-form>

      <div class="form-footer">
        <a-space>
          <a-button @click="showCreateDrawer = false">取消</a-button>
          <a-button type="primary" :loading="submitting" @click="handleFormSubmit">
            {{ isEditing ? '保存修改' : '创建商品' }}
          </a-button>
        </a-space>
      </div>
    </a-drawer>

    <!-- ═══ Batch Confirm Dialog ═══ -->
    <a-modal
      v-model:open="showBatchConfirm"
      :title="batchConfirmTitle"
      @ok="executeBatchAction"
      :ok-text="'确认'"
      :cancel-text="'取消'"
    >
      <p>确定要对选中的 {{ selectedRowKeys.length }} 个商品执行「{{ batchActionLabel }}」操作吗？</p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
/* ================================================
   FLOW: Product Management
   SCREEN 1-4 of 4: List + Detail + Create + Batch
   ================================================ */
import { ref, computed, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import type { UploadFile } from 'ant-design-vue'
import {
  PlusOutlined, ReloadOutlined, MoreOutlined, EyeOutlined, EditOutlined,
  StopOutlined, DatabaseOutlined, DollarOutlined, UploadOutlined,
  DownloadOutlined, InboxOutlined,
} from '@ant-design/icons-vue'
import type { Product } from '@/views-redesign/shared/types'
import { mockProducts, mockBatchActions } from '@/views-redesign/shared/mock-data'

const router = useRouter()
const loading = ref(false)

// ── Data ──
const products = ref<Product[]>([...mockProducts])
const searchQuery = ref('')
const statusFilter = ref<string | undefined>()
const categoryFilter = ref<string | undefined>()
const platformFilter = ref<string | undefined>()

// ── Selection ──
const selectedRowKeys = ref<string[]>([])
const batchActions = ref(mockBatchActions)

// ── Sheets ──
const showDetailSheet = ref(false)
const selectedProduct = ref<Product | null>(null)
const showCreateDrawer = ref(false)
const isEditing = ref(false)
const submitting = ref(false)

// ── Batch ──
const showBatchConfirm = ref(false)
const batchActionKey = ref('')
const batchActionLabel = ref('')
const batchConfirmTitle = ref('')

// ── Form ──
const formState = reactive({
  name: '',
  name_en: '',
  category: undefined as string | undefined,
  brand: '',
  fileList: [] as UploadFile[],
})

// ── Filter Options ──
const statusOptions = [
  { label: '在售', value: 'active' },
  { label: '草稿', value: 'draft' },
  { label: '下架', value: 'inactive' },
  { label: '归档', value: 'archived' },
]

const categoryOptions = computed(() => {
  const cats = [...new Set(products.value.map(p => p.category_name))]
  return cats.map(c => ({ label: c, value: c }))
})

const platformOptions = [
  { label: 'Ozon', value: 'ozon' },
  { label: 'Shopee', value: 'shopee' },
  { label: 'Wildberries', value: 'wildberries' },
]

// ── Table Columns ──
const columns = [
  { title: '商品名称', key: 'name', dataIndex: 'name', ellipsis: true },
  { title: '状态', key: 'status', dataIndex: 'status', width: 90 },
  { title: '分类', dataIndex: 'category_name', width: 100 },
  { title: 'SKU', key: 'sku_count', dataIndex: 'sku_count', width: 120 },
  { title: '销量', dataIndex: 'total_sales', width: 80, sorter: (a: Product, b: Product) => a.total_sales - b.total_sales },
  { title: '收入', key: 'revenue', dataIndex: 'total_revenue', width: 120, sorter: (a: Product, b: Product) => a.total_revenue - b.total_revenue },
  { title: '平台', key: 'platforms', dataIndex: 'platforms', width: 180 },
  { title: '', key: 'action', width: 50 },
]

// ── Computed ──
const filteredProducts = computed(() => {
  let result = products.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(p => p.name.toLowerCase().includes(q) || (p.name_en?.toLowerCase().includes(q)))
  }
  if (statusFilter.value) {
    result = result.filter(p => p.status === statusFilter.value)
  }
  if (categoryFilter.value) {
    result = result.filter(p => p.category_name === categoryFilter.value)
  }
  if (platformFilter.value) {
    result = result.filter(p => p.platforms.includes(platformFilter.value!))
  }
  return result
})

const isAllSelected = computed(() => selectedRowKeys.value.length === filteredProducts.value.length && filteredProducts.value.length > 0)
const isIndeterminate = computed(() => selectedRowKeys.value.length > 0 && selectedRowKeys.value.length < filteredProducts.value.length)

// ── Helpers ──
function statusColor(status: string): string {
  const map: Record<string, string> = { active: 'success', draft: 'default', inactive: 'warning', archived: 'error' }
  return map[status] || 'default'
}

function statusLabel(status: string): string {
  const map: Record<string, string> = { active: '在售', draft: '草稿', inactive: '下架', archived: '归档' }
  return map[status] || status
}

function platformColor(code: string): string {
  const map: Record<string, string> = { ozon: '#005bff', shopee: '#ee4d2d', wildberries: '#cb11ab' }
  return map[code.toLowerCase()] || '#666'
}

function getBatchIcon(icon: string) {
  const map: Record<string, any> = {
    UploadOutlined, DownloadOutlined, DollarOutlined, InboxOutlined,
    AppstoreOutlined: DatabaseOutlined,
  }
  return map[icon] || PlusOutlined
}

// ── Handlers ──
function handleSearch() { /* filters are reactive, auto-applied */ }

function applyFilters() { /* filters are reactive, auto-applied */ }

function resetFilters() {
  searchQuery.value = ''
  statusFilter.value = undefined
  categoryFilter.value = undefined
  platformFilter.value = undefined
}

function onSelectChange(keys: string[]) {
  selectedRowKeys.value = keys
}

function toggleSelectAll() {
  if (isAllSelected.value) {
    selectedRowKeys.value = []
  } else {
    selectedRowKeys.value = filteredProducts.value.map(p => p.id)
  }
}

function openProductDetail(record: Product) {
  selectedProduct.value = record
  showDetailSheet.value = true
}

function handleRowAction(key: string, record: Product) {
  if (key === 'view') {
    openProductDetail(record)
  } else if (key === 'edit') {
    isEditing.value = true
    formState.name = record.name
    formState.name_en = record.name_en || ''
    formState.category = record.category_name
    formState.brand = record.brand_name || ''
    showCreateDrawer.value = true
  } else if (key === 'archive') {
    Modal.confirm({
      title: '确认归档',
      content: `确定要归档「${record.name}」吗？归档后商品将不再展示在列表中。`,
      okText: '确认归档',
      okType: 'danger',
      cancelText: '取消',
      onOk() {
        record.status = 'archived'
        message.success(`${record.name} 已归档`)
      },
    })
  }
}

function handleBatchAction(actionKey: string) {
  const action = batchActions.value.find(a => a.key === actionKey)
  if (!action) return
  batchActionKey.value = actionKey
  batchActionLabel.value = action.label
  batchConfirmTitle.value = `批量${action.label}`
  showBatchConfirm.value = true
}

function executeBatchAction() {
  message.success(`已对 ${selectedRowKeys.value.length} 个商品执行「${batchActionLabel.value}」`)
  selectedRowKeys.value = []
  showBatchConfirm.value = false
}

async function handleFormSubmit() {
  if (!formState.name || !formState.category) {
    message.warning('请填写必填字段')
    return
  }
  submitting.value = true
  await new Promise(r => setTimeout(r, 600))

  if (isEditing.value) {
    message.success('商品已更新')
  } else {
    const newProduct: Product = {
      id: `p-${Date.now()}`,
      name: formState.name,
      name_en: formState.name_en,
      category_name: formState.category!,
      brand_name: formState.brand || undefined,
      status: 'draft',
      sku_count: 0,
      on_shelf_count: 0,
      total_sales: 0,
      total_revenue: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      platforms: [],
    }
    products.value.unshift(newProduct)
    message.success('商品已创建')
  }

  submitting.value = false
  showCreateDrawer.value = false
  isEditing.value = false
  formState.name = ''
  formState.name_en = ''
  formState.category = undefined
  formState.brand = ''
  formState.fileList = []
}
</script>

<style scoped>
.product-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
}
.page-subtitle {
  font-size: 14px;
  color: var(--ant-color-text-secondary);
  margin: 4px 0 0;
}

/* ═══ Filters ═══ */
.filter-card {
  border-radius: 12px;
}
.filter-card :deep(.ant-card-body) {
  padding: 12px 16px;
}
.filter-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

/* ═══ Table ═══ */
.table-card {
  border-radius: 12px;
}
.product-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.product-name-text {
  font-weight: 500;
  font-size: 13px;
}
.product-name-sub {
  font-size: 11px;
  color: var(--ant-color-text-tertiary);
}

/* ═══ Batch Action Bar ═══ */
.batch-bar {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 12px 24px;
  background: #1a1d23;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.24);
  z-index: 200;
}
.batch-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
}
.batch-count {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
}
.batch-bar :deep(.ant-btn-default) {
  color: rgba(255, 255, 255, 0.85);
  border-color: rgba(255, 255, 255, 0.2);
}
.batch-bar :deep(.ant-btn-default:hover) {
  color: #fff;
  border-color: rgba(255, 255, 255, 0.4);
}
.batch-bar :deep(.ant-checkbox-wrapper) {
  color: #fff;
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.3s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateX(-50%) translateY(20px);
  opacity: 0;
}

/* ═══ Form Footer ═══ */
.form-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px 24px;
  background: var(--ant-color-bg-container, #fff);
  border-top: 1px solid var(--ant-color-border-secondary, #f0f0f0);
  display: flex;
  justify-content: flex-end;
}
</style>
