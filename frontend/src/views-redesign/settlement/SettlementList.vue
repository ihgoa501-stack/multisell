<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h2 style="margin: 0;">结算管理</h2>
        <span style="color: var(--ant-color-text-secondary, rgba(0,0,0,0.45)); font-size: 14px;">管理各平台结算单与对账</span>
      </div>
      <a-space>
        <a-button @click="showGenerateMock = true" :disabled="platforms.length === 0">生成模拟数据</a-button>
        <a-button type="primary" @click="showImportModal = true">＋ 导入结算单</a-button>
      </a-space>
    </div>

    <!-- 筛选栏 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-space align="center">
        <a-select
          v-model:value="query.platform_id"
          :options="platformOptions"
          placeholder="选择平台"
          allow-clear
          style="width: 160px;"
          @change="fetchData"
        />
        <a-select
          v-model:value="query.status"
          :options="statusOptions"
          placeholder="结算状态"
          allow-clear
          style="width: 140px;"
          @change="fetchData"
        />
        <a-input
          v-model:value="query.keyword"
          placeholder="搜索结算单号"
          allow-clear
          style="width: 200px;"
          @pressEnter="fetchData"
        />
        <a-button @click="fetchData" :loading="loading">查询</a-button>
        <a-button type="text" @click="resetFilter">重置</a-button>
      </a-space>
    </a-card>

    <!-- 表格 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="{ current: query.page, pageSize: query.page_size, total: pagination.itemCount, showSizeChanger: true }"
        @change="onTableChange"
        row-key="id"
      />
    </a-card>

    <!-- 导入弹窗 -->
    <a-modal v-model:open="showImportModal" title="导入结算单" :footer="null" style="width: 520px;">
      <a-form ref="importFormRef" :model="importForm" :rules="importRules" layout="vertical">
        <a-form-item label="平台" name="platform_id">
          <a-select v-model:value="importForm.platform_id" :options="platformOptions" placeholder="选择平台" />
        </a-form-item>
        <a-form-item label="结算单号" name="settlement_no">
          <a-input v-model:value="importForm.settlement_no" placeholder="如: OZN-202501-001" />
        </a-form-item>
        <a-form-item label="结算周期">
          <a-range-picker v-model:value="periodRange" valueFormat="x" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="币种">
          <a-select v-model:value="importForm.currency" :options="currencyOptions" style="width: 120px;" />
        </a-form-item>
        <a-space>
          <a-form-item label="总收入">
            <a-input-number v-model:value="importForm.total_revenue" :min="0" :step="0.01" style="width: 140px;" />
          </a-form-item>
          <a-form-item label="总费用">
            <a-input-number v-model:value="importForm.total_fee" :min="0" :step="0.01" style="width: 140px;" />
          </a-form-item>
        </a-space>
        <a-space>
          <a-form-item label="总退款">
            <a-input-number v-model:value="importForm.total_refund" :min="0" :step="0.01" style="width: 140px;" />
          </a-form-item>
          <a-form-item label="净收入">
            <a-input-number v-model:value="importForm.total_net" :min="0" :step="0.01" style="width: 140px;" />
          </a-form-item>
        </a-space>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showImportModal = false">取消</a-button>
          <a-button type="primary" :loading="importing" @click="handleImport">导入</a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 生成模拟数据弹窗 -->
    <a-modal v-model:open="showGenerateMock" title="生成模拟结算数据" :footer="null" style="width: 400px;">
      <a-form layout="vertical">
        <a-form-item label="平台">
          <a-select v-model:value="mockPlatformId" :options="platformOptions" placeholder="选择平台" />
        </a-form-item>
        <a-form-item label="订单数量">
          <a-input-number v-model:value="mockCount" :min="1" :max="50" style="width: 120px;" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showGenerateMock = false">取消</a-button>
          <a-button type="primary" :loading="generating" @click="handleGenerateMock">生成</a-button>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { message, Modal, Tag, Button, Space } from 'ant-design-vue'
import { useRouter } from 'vue-router'
import { apiModules } from '@/api'
import { platformApi } from '@/api'
import { settlementApi } from '@/api/modules/settlement'

const router = useRouter()
const loading = ref(false)
const importing = ref(false)
const generating = ref(false)
const data = ref<any[]>([])
const platforms = ref<any[]>([])
const showImportModal = ref(false)
const showGenerateMock = ref(false)
const mockPlatformId = ref<number | null>(null)
const mockCount = ref(5)

const periodRange = ref<[number, number] | null>(null)

const platformOptions = computed(() =>
  platforms.value.map((p: any) => ({ label: p.name, value: p.id }))
)

const currencyOptions = [
  { label: 'CNY - 人民币', value: 'CNY' },
  { label: 'USD - 美元', value: 'USD' },
  { label: 'RUB - 卢布', value: 'RUB' },
  { label: 'VND - 越南盾', value: 'VND' },
]

const statusOptions = [
  { label: '待对账', value: 'pending' },
  { label: '对账中', value: 'reconciling' },
  { label: '已对账', value: 'reconciled' },
  { label: '已关闭', value: 'closed' },
]

const query = reactive<{
  platform_id: number | null
  status: string | null
  keyword: string
  page: number
  page_size: number
}>({
  platform_id: null,
  status: null,
  keyword: '',
  page: 1,
  page_size: 20,
})

const importForm = reactive({
  platform_id: null as number | null,
  settlement_no: '',
  currency: 'CNY',
  total_revenue: 0,
  total_fee: 0,
  total_refund: 0,
  total_net: 0,
})

const importRules = {
  platform_id: { required: true, message: '请选择平台', trigger: 'change' },
  settlement_no: { required: true, message: '请填写结算单号', trigger: 'blur' },
}

const pagination = reactive({
  itemCount: 0,
})

let prevPageSize = query.page_size

function resetFilter() {
  query.platform_id = null
  query.status = null
  query.keyword = ''
  query.page = 1
  fetchData()
}

function onTableChange(paginationInfo: any) {
  if (paginationInfo.pageSize !== prevPageSize) {
    query.page = 1
    prevPageSize = paginationInfo.pageSize
  } else {
    query.page = paginationInfo.current
  }
  query.page_size = paginationInfo.pageSize
  fetchData()
}

async function fetchPlatforms() {
  try {
    const res = await platformApi.list()
    platforms.value = res.data?.data ?? []
  } catch {
    // ignore
  }
}

async function fetchData() {
  loading.value = true
  try {
    const res = await settlementApi.list({
      platform_id: query.platform_id ?? undefined,
      status: query.status ?? undefined,
      keyword: query.keyword || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    const body = res.data
    data.value = body?.data?.records ?? body?.records ?? []
    pagination.itemCount = body?.data?.total ?? body?.total ?? 0
  } catch (err: any) {
    message.error('加载失败: ' + (err.message || ''))
  } finally {
    loading.value = false
  }
}

async function handleImport() {
  if (!importForm.platform_id || !importForm.settlement_no) {
    message.warning('请填写必要信息')
    return
  }
  importing.value = true
  try {
    const payload: any = {
      platform_id: importForm.platform_id,
      settlement_no: importForm.settlement_no,
      currency: importForm.currency,
      total_revenue: importForm.total_revenue,
      total_fee: importForm.total_fee,
      total_refund: importForm.total_refund,
      total_net: importForm.total_net,
    }
    if (periodRange.value) {
      payload.period_start = new Date(periodRange.value[0]).toISOString()
      payload.period_end = new Date(periodRange.value[1]).toISOString()
    }
    await settlementApi.import(payload)
    message.success('导入成功')
    showImportModal.value = false
    fetchData()
  } catch (err: any) {
    message.error('导入失败: ' + (err.response?.data?.detail || err.message || ''))
  } finally {
    importing.value = false
  }
}

async function handleGenerateMock() {
  if (!mockPlatformId.value) {
    message.warning('请选择平台')
    return
  }
  generating.value = true
  try {
    await settlementApi.generateMock({ platform_id: mockPlatformId.value, count: mockCount.value })
    message.success('模拟数据生成成功')
    showGenerateMock.value = false
    fetchData()
  } catch (err: any) {
    message.error('生成失败: ' + (err.response?.data?.detail || err.message || ''))
  } finally {
    generating.value = false
  }
}

function viewDetail(id: number) {
  router.push(`/settlements/${id}`)
}

function handleDelete(row: any) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除结算单 "${row.settlement_no}"？此操作不可恢复。`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        await settlementApi.delete(row.id)
        message.success('已删除')
        fetchData()
      } catch {
        message.error('删除失败')
      }
    },
  })
}

const columns = [
  {
    title: '结算单号', dataIndex: 'settlement_no', key: 'settlement_no', width: 180,
    customRender: ({ record }: any) => h('a', {
      style: 'cursor: pointer; color: var(--ant-color-primary, #1677ff);',
      onClick: () => viewDetail(record.id),
    }, record.settlement_no),
  },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name', width: 120 },
  {
    title: '周期', dataIndex: 'period', key: 'period', width: 180,
    customRender: ({ record }: any) => {
      if (!record.period_start && !record.period_end) return '-'
      const s = record.period_start ? record.period_start.slice(0, 10) : '?'
      const e = record.period_end ? record.period_end.slice(0, 10) : '?'
      return `${s} ~ ${e}`
    },
  },
  {
    title: '总收入', dataIndex: 'total_revenue', key: 'total_revenue', width: 110,
    customRender: ({ text }: any) => `¥${(text ?? 0).toFixed(2)}`,
  },
  {
    title: '总费用', dataIndex: 'total_fee', key: 'total_fee', width: 110,
    customRender: ({ text }: any) => `¥${(text ?? 0).toFixed(2)}`,
  },
  {
    title: '净收入', dataIndex: 'total_net', key: 'total_net', width: 110,
    customRender: ({ text }: any) => {
      const net = text ?? 0
      return h('span', {
        style: `color: ${net >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)'}; font-weight: 600;`,
      }, `¥${net.toFixed(2)}`)
    },
  },
  { title: '币种', dataIndex: 'currency', key: 'currency', width: 70 },
  {
    title: '对账状态', dataIndex: 'status', key: 'status', width: 110,
    customRender: ({ record }: any) => {
      const map: Record<string, { label: string; color: string }> = {
        pending: { label: '待对账', color: 'warning' },
        reconciling: { label: '对账中', color: 'processing' },
        reconciled: { label: '已对账', color: 'success' },
        closed: { label: '已关闭', color: 'default' },
      }
      const s = map[record.status] ?? { label: record.status, color: 'default' }
      return h(Tag, { color: s.color }, { default: () => s.label })
    },
  },
  {
    title: '明细', dataIndex: 'item_count', key: 'item_count', width: 70,
    customRender: ({ record }: any) => {
      const extra = record.unmatched_count > 0 ? ` ⚠${record.unmatched_count}` : ''
      return `${record.item_count ?? 0}${extra}`
    },
  },
  {
    title: '操作', key: 'actions', width: 160,
    customRender: ({ record }: any) => h(Space, null, {
      default: () => [
        h(Button, { size: 'small', type: 'primary', ghost: true, onClick: () => viewDetail(record.id) },
          { default: () => '对账' }),
        h(Button, { size: 'small', danger: true, ghost: true, onClick: () => handleDelete(record) },
          { default: () => '删除' }),
      ],
    }),
  },
]

onMounted(async () => {
  await fetchPlatforms()
  await fetchData()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
</style>
