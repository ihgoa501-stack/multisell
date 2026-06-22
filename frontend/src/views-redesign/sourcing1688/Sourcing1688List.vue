<template>
  <div class="sourcing-1688-list-page">
    <!-- 页面头部 -->
    <div style="margin-bottom: 16px;">
      <h3 class="page-title" style="margin: 0;">1688 货源池</h3>
      <span style="color: var(--ant-color-text-secondary);">管理 1688 采集的货源，审核后导入为正式商品</span>
    </div>

    <!-- 筛选区 -->
    <a-card class="filter-card" :bordered="false" size="small" style="margin-top: 16px;">
      <a-form layout="inline">
        <a-form-item label="状态">
          <a-select
            v-model:value="query.status"
            :options="statusOptions"
            placeholder="全部状态"
            allow-clear
            style="width: 140px;"
            @change="handleSearch"
          />
        </a-form-item>
        <a-form-item label="搜索">
          <a-input
            v-model:value="query.keyword"
            placeholder="标题 / 供应商名称"
            allow-clear
            style="width: 220px;"
            @pressEnter="handleSearch"
          />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">搜索</a-button>
          <a-button style="margin-left: 8px;" @click="handleReset">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 列表 -->
    <a-card class="table-card" :bordered="false" style="margin-top: 16px;">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="antPagination"
        :row-key="(row: Sourcing1688ProductVO) => row.id"
        @change="handleTableChange"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'price'">
            {{ record.price != null ? `¥${record.price}` : '-' }}
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <a-tag :color="statusColorMap[record.status] || 'default'">
              {{ statusLabel(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'actions'">
            <template v-if="record.status === 'collected'">
              <a-space>
                <a-button size="small" @click="openDetail(record)">详情</a-button>
                <a-button size="small" type="primary" @click="openImportModal(record)">导入</a-button>
                <a-popconfirm title="确定驳回该商品？" @confirm="handleReject(record)">
                  <a-button size="small" danger>驳回</a-button>
                </a-popconfirm>
              </a-space>
            </template>
            <template v-else-if="record.status === 'imported'">
              <a-space>
                <a-button size="small" @click="openDetail(record)">详情</a-button>
                <a-tag color="success">已导入 (ID: {{ record.product_id }})</a-tag>
              </a-space>
            </template>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 导入确认弹窗 -->
    <a-modal
      v-model:open="showImportModal"
      title="确认导入"
      :confirm-loading="importLoading"
      @ok="handleImportConfirm"
    >
      <a-form layout="horizontal" :label-col="{ style: { width: '100px' } }">
        <a-form-item label="商品标题">
          <a-input :value="currentCandidate?.title" disabled />
        </a-form-item>
        <a-form-item label="分类 ID">
          <a-input-number v-model:value="importForm.category_id" placeholder="选填" :min="0" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="品牌 ID">
          <a-input-number v-model:value="importForm.brand_id" placeholder="选填" :min="0" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="货品类型">
          <a-select
            v-model:value="importForm.cargo_type"
            :options="cargoTypeOptions"
          />
        </a-form-item>
        <a-form-item label="单位">
          <a-input v-model:value="importForm.unit" placeholder="件" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="showDetail" title="候选商品详情" :width="600" placement="right">
      <template v-if="detailData">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="ID">{{ detailData.id }}</a-descriptions-item>
          <a-descriptions-item label="标题">{{ detailData.title }}</a-descriptions-item>
          <a-descriptions-item label="1688 链接">
            <a :href="detailData.source_url" target="_blank" rel="noopener">{{ detailData.source_url }}</a>
          </a-descriptions-item>
          <a-descriptions-item label="供货价">{{ detailData.price != null ? `¥${detailData.price}` : '-' }}</a-descriptions-item>
          <a-descriptions-item label="最小起订量">{{ detailData.moq ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="供应商">{{ detailData.supplier_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="店铺链接">
            <a v-if="detailData.shop_url" :href="detailData.shop_url" target="_blank">{{ detailData.shop_url }}</a>
            <span v-else>-</span>
          </a-descriptions-item>
          <a-descriptions-item label="店铺地区">{{ detailData.shop_location || '-' }}</a-descriptions-item>
          <a-descriptions-item label="描述">
            <div style="white-space: pre-wrap;">{{ detailData.description || '-' }}</div>
          </a-descriptions-item>
          <a-descriptions-item label="包装长(cm)">{{ detailData.package_length_cm ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="包装宽(cm)">{{ detailData.package_width_cm ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="包装高(cm)">{{ detailData.package_height_cm ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="包装重(kg)">{{ detailData.package_weight_kg ?? '-' }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="statusColorMap[detailData.status] || 'default'" size="small">{{ statusLabel(detailData.status) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item v-if="detailData.product_id" label="关联商品 ID">{{ detailData.product_id }}</a-descriptions-item>
          <a-descriptions-item v-if="detailData.supplier_id" label="关联供应商 ID">{{ detailData.supplier_id }}</a-descriptions-item>
          <a-descriptions-item label="采集人">{{ detailData.collected_by || '-' }}</a-descriptions-item>
          <a-descriptions-item label="采集时间">{{ detailData.created_at || '-' }}</a-descriptions-item>
          <a-descriptions-item v-if="detailData.imported_by" label="导入人">{{ detailData.imported_by }}</a-descriptions-item>
          <a-descriptions-item v-if="detailData.imported_at" label="导入时间">{{ detailData.imported_at }}</a-descriptions-item>
        </a-descriptions>

        <template v-if="detailData.images && detailData.images.length > 0">
          <h4 style="margin-top: 16px; margin-bottom: 8px;">图片</h4>
          <a-space>
            <img
              v-for="(img, idx) in detailData.images"
              :key="idx"
              :src="img"
              style="width: 100px; height: 100px; object-fit: cover; border-radius: 4px; border: 1px solid #eee;"
            />
          </a-space>
        </template>

        <template v-if="detailData.attributes && detailData.attributes.length > 0">
          <h4 style="margin-top: 16px; margin-bottom: 8px;">属性</h4>
          <a-table size="small" :pagination="false" :data-source="detailData.attributes" row-key="name">
            <a-table-column title="名称" data-index="name" key="name">
              <template #default="{ record: attr, index: idx }">
                {{ attr.name || attr.key || idx }}
              </template>
            </a-table-column>
            <a-table-column title="值" data-index="value" key="value">
              <template #default="{ record: attr }">
                {{ attr.value || JSON.stringify(attr) }}
              </template>
            </a-table-column>
          </a-table>
        </template>

        <template v-if="detailData.sku_variants && detailData.sku_variants.length > 0">
          <h4 style="margin-top: 16px; margin-bottom: 8px;">SKU 变体</h4>
          <a-table size="small" :pagination="false" :data-source="detailData.sku_variants" row-key="spec">
            <a-table-column title="规格" data-index="spec" key="spec">
              <template #default="{ record: sk }">
                {{ sk.spec || '-' }}
              </template>
            </a-table-column>
            <a-table-column title="价格" data-index="price" key="price">
              <template #default="{ record: sk }">
                {{ sk.price != null ? `¥${sk.price}` : '-' }}
              </template>
            </a-table-column>
            <a-table-column title="库存" data-index="stock" key="stock">
              <template #default="{ record: sk }">
                {{ sk.stock ?? '-' }}
              </template>
            </a-table-column>
          </a-table>
        </template>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import type { Sourcing1688ProductVO, ImportPayload } from '@/api/modules/sourcing1688'
import { listProducts, importProduct, rejectProduct, getProduct } from '@/api/modules/sourcing1688'

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

const statusColorMap: Record<string, string> = {
  collected: 'processing',
  imported: 'success',
  rejected: 'error',
}

const statusLabel = (s: string) => ({ collected: '已采集', imported: '已导入', rejected: '已驳回' } as Record<string, string>)[s] || s

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

const antPagination = computed(() => ({
  current: query.page,
  pageSize: query.page_size,
  total: total.value,
  showSizeChanger: true,
  pageSizeOptions: ['10', '20', '50', '100'],
}))

function handleTableChange(pag: any) {
  if (pag.current !== query.page) {
    query.page = pag.current
    fetchData()
  }
  if (pag.pageSize !== query.page_size) {
    query.page_size = pag.pageSize
    query.page = 1
    fetchData()
  }
}

// ===== 详情抽屉 =====
const showDetail = ref(false)
const detailData = ref<Sourcing1688ProductVO | null>(null)

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
const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70, align: 'center' },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true, width: 200 },
  { title: '供货价', dataIndex: 'price', key: 'price', width: 100 },
  { title: '供应商', dataIndex: 'supplier_name', key: 'supplier_name', width: 140, ellipsis: true },
  { title: '最小起订', dataIndex: 'moq', key: 'moq', width: 80, align: 'center' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90, align: 'center' },
  { title: '操作', dataIndex: 'actions', key: 'actions', width: 200, align: 'center', fixed: 'right' },
]

// ===== 方法 =====
function handleSearch() {
  query.page = 1
  fetchData()
}

function handleReset() {
  query.status = null
  query.keyword = ''
  query.page = 1
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
  } catch (e: any) {
    message.error('加载失败: ' + (e?.response?.data?.message || e.message))
  } finally {
    loading.value = false
  }
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
