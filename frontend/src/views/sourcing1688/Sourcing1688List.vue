<template>
  <div class="sourcing-1688-list-page">
    <!-- 页面头部 -->
    <n-page-header subtitle="管理 1688 采集的货源，审核后导入为正式商品">
      <template #title>
        <span class="page-title">📦 1688 货源池</span>
      </template>
    </n-page-header>

    <!-- 筛选区 -->
    <n-card class="filter-card" :bordered="false" size="small" style="margin-top: 16px;">
      <n-form inline :label-width="80">
        <n-form-item label="状态">
          <n-select
            v-model:value="query.status"
            :options="statusOptions"
            placeholder="全部状态"
            clearable
            style="width: 140px;"
            @update:value="handleSearch"
          />
        </n-form-item>
        <n-form-item label="搜索">
          <n-input
            v-model:value="query.keyword"
            placeholder="标题 / 供应商名称"
            clearable
            style="width: 220px;"
            @keydown.enter="handleSearch"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">搜索</n-button>
          <n-button style="margin-left: 8px;" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 列表 -->
    <n-card class="table-card" :bordered="false" style="margin-top: 16px;">
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        :row-key="(row: Sourcing1688ProductVO) => row.id"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        stripe
        size="small"
      />
    </n-card>

    <!-- 导入确认弹窗 -->
    <n-modal
      v-model:show="showImportModal"
      :mask-closable="false"
      preset="dialog"
      title="确认导入"
      :positive-text="'确认导入'"
      :negative-text="'取消'"
      :loading="importLoading"
      @positive-click="handleImportConfirm"
      @negative-click="showImportModal = false"
    >
      <n-form label-placement="left" :label-width="100">
        <n-form-item label="商品标题">
          <n-input :value="currentCandidate?.title" disabled />
        </n-form-item>
        <n-form-item label="分类 ID">
          <n-input-number v-model:value="importForm.category_id" placeholder="选填" clearable min="0" style="width: 100%;" />
        </n-form-item>
        <n-form-item label="品牌 ID">
          <n-input-number v-model:value="importForm.brand_id" placeholder="选填" clearable min="0" style="width: 100%;" />
        </n-form-item>
        <n-form-item label="货品类型">
          <n-select
            v-model:value="importForm.cargo_type"
            :options="cargoTypeOptions"
          />
        </n-form-item>
        <n-form-item label="单位">
          <n-input v-model:value="importForm.unit" placeholder="件" />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 详情抽屉 -->
    <n-drawer v-model:show="showDetail" :width="600" placement="right">
      <n-drawer-content title="候选商品详情" closable>
        <template v-if="detailData">
          <n-descriptions :column="1" bordered size="small" label-placement="left">
            <n-descriptions-item label="ID">{{ detailData.id }}</n-descriptions-item>
            <n-descriptions-item label="标题">{{ detailData.title }}</n-descriptions-item>
            <n-descriptions-item label="1688 链接">
              <a :href="detailData.source_url" target="_blank" rel="noopener">{{ detailData.source_url }}</a>
            </n-descriptions-item>
            <n-descriptions-item label="供货价">{{ detailData.price != null ? `¥${detailData.price}` : '-' }}</n-descriptions-item>
            <n-descriptions-item label="最小起订量">{{ detailData.moq ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="供应商">{{ detailData.supplier_name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="店铺链接">
              <a v-if="detailData.shop_url" :href="detailData.shop_url" target="_blank">{{ detailData.shop_url }}</a>
              <span v-else>-</span>
            </n-descriptions-item>
            <n-descriptions-item label="店铺地区">{{ detailData.shop_location || '-' }}</n-descriptions-item>
            <n-descriptions-item label="描述">
              <div style="white-space: pre-wrap;">{{ detailData.description || '-' }}</div>
            </n-descriptions-item>
            <n-descriptions-item label="包装长(cm)">{{ detailData.package_length_cm ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="包装宽(cm)">{{ detailData.package_width_cm ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="包装高(cm)">{{ detailData.package_height_cm ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="包装重(kg)">{{ detailData.package_weight_kg ?? '-' }}</n-descriptions-item>
            <n-descriptions-item label="状态">
              <n-tag :type="statusType(detailData.status)" size="small">{{ statusLabel(detailData.status) }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item v-if="detailData.product_id" label="关联商品 ID">{{ detailData.product_id }}</n-descriptions-item>
            <n-descriptions-item v-if="detailData.supplier_id" label="关联供应商 ID">{{ detailData.supplier_id }}</n-descriptions-item>
            <n-descriptions-item label="采集人">{{ detailData.collected_by || '-' }}</n-descriptions-item>
            <n-descriptions-item label="采集时间">{{ detailData.created_at || '-' }}</n-descriptions-item>
            <n-descriptions-item v-if="detailData.imported_by" label="导入人">{{ detailData.imported_by }}</n-descriptions-item>
            <n-descriptions-item v-if="detailData.imported_at" label="导入时间">{{ detailData.imported_at }}</n-descriptions-item>
          </n-descriptions>

          <template v-if="detailData.images && detailData.images.length > 0">
            <h4 style="margin-top: 16px; margin-bottom: 8px;">图片</h4>
            <n-space>
              <n-image
                v-for="(img, idx) in detailData.images"
                :key="idx"
                :src="img"
                :width="100"
                :height="100"
                object-fit="cover"
                style="border-radius: 4px; border: 1px solid #eee;"
              />
            </n-space>
          </template>

          <template v-if="detailData.attributes && detailData.attributes.length > 0">
            <h4 style="margin-top: 16px; margin-bottom: 8px;">属性</h4>
            <n-table size="small" :bordered="true">
              <thead>
                <tr><th>名称</th><th>值</th></tr>
              </thead>
              <tbody>
                <tr v-for="(attr, idx) in detailData.attributes" :key="idx">
                  <td>{{ attr.name || attr.key || idx }}</td>
                  <td>{{ attr.value || JSON.stringify(attr) }}</td>
                </tr>
              </tbody>
            </n-table>
          </template>

          <template v-if="detailData.sku_variants && detailData.sku_variants.length > 0">
            <h4 style="margin-top: 16px; margin-bottom: 8px;">SKU 变体</h4>
            <n-table size="small" :bordered="true">
              <thead>
                <tr><th>规格</th><th>价格</th><th>库存</th></tr>
              </thead>
              <tbody>
                <tr v-for="(sk, idx) in detailData.sku_variants" :key="idx">
                  <td>{{ sk.spec || '-' }}</td>
                  <td>{{ sk.price != null ? `¥${sk.price}` : '-' }}</td>
                  <td>{{ sk.stock ?? '-' }}</td>
                </tr>
              </tbody>
            </n-table>
          </template>
        </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { NButton, NPopconfirm, NTag, NSpace, NDrawer, NDrawerContent, NDescriptions, NDescriptionsItem, NImage, useMessage, type DataTableColumn } from 'naive-ui'
import type { Sourcing1688ProductVO, ImportPayload } from '@/api/modules/sourcing1688'
import { listProducts, importProduct, rejectProduct, getProduct } from '@/api/modules/sourcing1688'

const message = useMessage()

// ===== 状态选项 =====
const statusOptions = [
  { label: '已采集', value: 'collected' },
  { label: '已导入', value: 'imported' },
  { label: '已驳回', value: 'rejected' },
]

const cargoTypeOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感', value: 'sensitive' },
]

// ===== 查询参数 =====
const query = reactive({
  status: null as string | null,
  keyword: '',
  page: 1,
  page_size: 20,
})

const loading = ref(false)
const data = ref<Sourcing1688ProductVO[]>([])
const total = ref(0)

const pagination = reactive({
  page: 1,
  pageSize: 20,
  pageCount: 1,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  onChange: (page: number) => { query.page = page; fetchData() },
  onUpdatePageSize: (size: number) => { query.page_size = size; query.page = 1; fetchData() },
})

// ===== 详情抽屉 =====
const showDetail = ref(false)
const detailData = ref<Sourcing1688ProductVO | null>(null)

const statusType = (s: string) => ({ collected: 'info', imported: 'success', rejected: 'error' } as Record<string, string>)[s] || 'default'
const statusLabel = (s: string) => ({ collected: '已采集', imported: '已导入', rejected: '已驳回' } as Record<string, string>)[s] || s

// ===== 弹窗状态 =====
const showImportModal = ref(false)
const importLoading = ref(false)
const currentCandidate = ref<Sourcing1688ProductVO | null>(null)
const importForm = reactive<ImportPayload>({
  category_id: undefined,
  brand_id: undefined,
  cargo_type: 'normal',
  unit: '件',
})

// ===== 列定义 =====
const statusRender = (status: string) => {
  const map: Record<string, { label: string; type: 'success' | 'warning' | 'error' | 'info' }> = {
    collected: { label: '已采集', type: 'info' },
    imported: { label: '已导入', type: 'success' },
    rejected: { label: '已驳回', type: 'error' },
  }
  const s = map[status] || { label: status, type: 'warning' }
  return h(NTag, { type: s.type, size: 'small' }, { default: () => s.label })
}

const columns: DataTableColumn[] = [
  { title: 'ID', key: 'id', width: 70, align: 'center' },
  { title: '标题', key: 'title', ellipsis: { tooltip: true }, minWidth: 200 },
  { title: '供货价', key: 'price', width: 100, render: (r: Sourcing1688ProductVO) => r.price != null ? `¥${r.price}` : '-' },
  { title: '供应商', key: 'supplier_name', width: 140, ellipsis: { tooltip: true } },
  { title: '最小起订', key: 'moq', width: 80, align: 'center' },
  {
    title: '状态', key: 'status', width: 90, align: 'center',
    render: (r: Sourcing1688ProductVO) => statusRender(r.status),
  },
  {
    title: '操作', key: 'actions', width: 200, align: 'center', fixed: 'right',
    render: (r: Sourcing1688ProductVO) => {
      if (r.status === 'collected') {
        return h(NSpace, { justify: 'center' }, {
          default: () => [
            h(NButton, {
              size: 'small',
              onClick: () => openDetail(r),
            }, { default: () => '详情' }),
            h(NButton, {
              size: 'small', type: 'primary',
              onClick: () => openImportModal(r),
            }, { default: () => '导入' }),
            h(NPopconfirm, {
              onPositiveClick: () => handleReject(r),
            }, {
              default: () => '确定驳回该商品？',
              trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => '驳回' }),
            }),
          ],
        })
      }
      if (r.status === 'imported') {
        return h(NSpace, { justify: 'center' }, {
          default: () => [
            h(NButton, { size: 'small', onClick: () => openDetail(r) }, { default: () => '详情' }),
            h(NTag, { type: 'success', size: 'small' }, { default: () => `已导入 (ID: ${r.product_id})` }),
          ],
        })
      }
      return null
    },
  },
]

// ===== 方法 =====
function handleSearch() {
  query.page = 1
  pagination.page = 1
  fetchData()
}

function handleReset() {
  query.status = null
  query.keyword = ''
  query.page = 1
  pagination.page = 1
  fetchData()
}

async function fetchData() {
  loading.value = true
  try {
    const res = await listProducts({
      status: query.status ?? undefined,
      keyword: query.keyword || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    data.value = (res as any).items ?? []
    total.value = (res as any).total ?? 0
    pagination.page = query.page
    pagination.pageCount = Math.ceil(total.value / query.page_size) || 1
  } catch (e: any) {
    message.error('加载失败: ' + (e?.response?.data?.message || e.message))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  query.page = page
  fetchData()
}

function handlePageSizeChange(size: number) {
  query.page_size = size
  query.page = 1
  fetchData()
}

async function openDetail(candidate: Sourcing1688ProductVO) {
  showDetail.value = true
  detailData.value = candidate
  try {
    const res = await getProduct(candidate.id)
    detailData.value = (res as any).data ?? candidate
  } catch {
    // fallback to list data
  }
}

function openImportModal(candidate: Sourcing1688ProductVO) {
  currentCandidate.value = candidate
  importForm.category_id = undefined
  importForm.brand_id = undefined
  importForm.cargo_type = 'normal'
  importForm.unit = '件'
  showImportModal.value = true
}

async function handleImportConfirm() {
  if (!currentCandidate.value) return
  importLoading.value = true
  try {
    await importProduct(currentCandidate.value.id, { ...importForm })
    message.success('导入成功')
    showImportModal.value = false
    fetchData()
  } catch (e: any) {
    message.error('导入失败: ' + (e?.response?.data?.message || e.message))
  } finally {
    importLoading.value = false
  }
}

async function handleReject(candidate: Sourcing1688ProductVO) {
  try {
    await rejectProduct(candidate.id)
    message.success('已驳回')
    fetchData()
  } catch (e: any) {
    message.error('驳回失败: ' + (e?.response?.data?.message || e.message))
  }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.sourcing-1688-list-page {
  padding: 16px;
}
.page-title {
  font-size: 22px;
  font-weight: 700;
}
.filter-card, .table-card {
  border-radius: 8px;
}
</style>
