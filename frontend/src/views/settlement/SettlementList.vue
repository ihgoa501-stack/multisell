<template>
  <div>
    <n-page-header subtitle="管理各平台结算单与对账">
      <template #title>💰 结算管理</template>
      <template #extra>
        <n-space>
          <n-button @click="showGenerateMock = true" :disabled="platforms.length === 0">🎲 生成模拟数据</n-button>
          <n-button type="primary" @click="showImportModal = true">＋ 导入结算单</n-button>
        </n-space>
      </template>
    </n-page-header>

    <!-- 筛选栏 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space align="center">
        <n-select
          v-model:value="query.platform_id"
          :options="platformOptions"
          placeholder="选择平台"
          clearable
          style="width: 160px;"
          @update:value="fetchData"
        />
        <n-select
          v-model:value="query.status"
          :options="statusOptions"
          placeholder="结算状态"
          clearable
          style="width: 140px;"
          @update:value="fetchData"
        />
        <n-input
          v-model:value="query.keyword"
          placeholder="搜索结算单号"
          clearable
          style="width: 200px;"
          @keyup.enter="fetchData"
        />
        <n-button @click="fetchData" :loading="loading">查询</n-button>
        <n-button quaternary @click="resetFilter">重置</n-button>
      </n-space>
    </n-card>

    <!-- 表格 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :pagination="pagination"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
        :row-key="(row: any) => row.id"
      />
    </n-card>

    <!-- 导入弹窗 -->
    <n-modal v-model:show="showImportModal" title="导入结算单" preset="card" style="width: 520px;">
      <n-form ref="importFormRef" :model="importForm" :rules="importRules" label-placement="left" label-width="120px">
        <n-form-item label="平台" path="platform_id">
          <n-select v-model:value="importForm.platform_id" :options="platformOptions" placeholder="选择平台" />
        </n-form-item>
        <n-form-item label="结算单号" path="settlement_no">
          <n-input v-model:value="importForm.settlement_no" placeholder="如: OZN-202501-001" />
        </n-form-item>
        <n-form-item label="结算周期">
          <n-date-picker v-model:value="periodRange" type="daterange" clearable style="width: 100%;" />
        </n-form-item>
        <n-form-item label="币种">
          <n-select v-model:value="importForm.currency" :options="currencyOptions" style="width: 120px;" />
        </n-form-item>
        <n-space>
          <n-form-item label="总收入">
            <n-input-number v-model:value="importForm.total_revenue" :min="0" :step="0.01" style="width: 140px;" />
          </n-form-item>
          <n-form-item label="总费用">
            <n-input-number v-model:value="importForm.total_fee" :min="0" :step="0.01" style="width: 140px;" />
          </n-form-item>
        </n-space>
        <n-space>
          <n-form-item label="总退款">
            <n-input-number v-model:value="importForm.total_refund" :min="0" :step="0.01" style="width: 140px;" />
          </n-form-item>
          <n-form-item label="净收入">
            <n-input-number v-model:value="importForm.total_net" :min="0" :step="0.01" style="width: 140px;" />
          </n-form-item>
        </n-space>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showImportModal = false">取消</n-button>
          <n-button type="primary" :loading="importing" @click="handleImport">导入</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 生成模拟数据弹窗 -->
    <n-modal v-model:show="showGenerateMock" title="生成模拟结算数据" preset="card" style="width: 400px;">
      <n-form label-placement="left" label-width="100px">
        <n-form-item label="平台">
          <n-select v-model:value="mockPlatformId" :options="platformOptions" placeholder="选择平台" />
        </n-form-item>
        <n-form-item label="订单数量">
          <n-input-number v-model:value="mockCount" :min="1" :max="50" style="width: 120px;" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showGenerateMock = false">取消</n-button>
          <n-button type="primary" :loading="generating" @click="handleGenerateMock">生成</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { NButton, NTag, NSpace, useMessage, useDialog } from 'naive-ui'
import { useRouter } from 'vue-router'
import { apiModules } from '@/api'
import { platformApi } from '@/api'
import { settlementApi } from '@/api/modules/settlement'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()
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
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (page: number) => { query.page = page; fetchData() },
  onUpdatePageSize: (size: number) => { query.page_size = size; query.page = 1; fetchData() },
})

function resetFilter() {
  query.platform_id = null
  query.status = null
  query.keyword = ''
  query.page = 1
  fetchData()
}

function onPageChange(page: number) {
  query.page = page
  fetchData()
}

function onPageSizeChange(size: number) {
  query.page_size = size
  query.page = 1
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
  dialog.warning({
    title: '确认删除',
    content: `确定删除结算单 "${row.settlement_no}"？此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
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
  { title: '结算单号', key: 'settlement_no', width: 180,
    render: (row: any) => h('a', {
      style: 'cursor: pointer; color: #18a058;',
      onClick: () => viewDetail(row.id),
    }, row.settlement_no),
  },
  { title: '平台', key: 'platform_name', width: 120 },
  { title: '周期', key: 'period',
    width: 180,
    render: (row: any) => {
      if (!row.period_start && !row.period_end) return '-'
      const s = row.period_start ? row.period_start.slice(0, 10) : '?'
      const e = row.period_end ? row.period_end.slice(0, 10) : '?'
      return `${s} ~ ${e}`
    },
  },
  { title: '总收入', key: 'total_revenue', width: 110,
    render: (row: any) => `¥${(row.total_revenue ?? 0).toFixed(2)}`,
  },
  { title: '总费用', key: 'total_fee', width: 110,
    render: (row: any) => `¥${(row.total_fee ?? 0).toFixed(2)}`,
  },
  { title: '净收入', key: 'total_net', width: 110,
    render: (row: any) => {
      const net = row.total_net ?? 0
      const color = net >= 0 ? '#18a058' : '#d03050'
      return h('span', { style: `color: ${color}; font-weight: 600;` }, `¥${net.toFixed(2)}`)
    },
  },
  { title: '币种', key: 'currency', width: 70 },
  { title: '对账状态', key: 'status', width: 110,
    render: (row: any) => {
      const map: Record<string, { label: string; type: string }> = {
        pending: { label: '待对账', type: 'warning' },
        reconciling: { label: '对账中', type: 'info' },
        reconciled: { label: '已对账', type: 'success' },
        closed: { label: '已关闭', type: 'default' },
      }
      const s = map[row.status] ?? { label: row.status, type: 'default' }
      return h(NTag, { type: s.type as any, size: 'small' }, { default: () => s.label })
    },
  },
  { title: '明细', key: 'item_count', width: 70,
    render: (row: any) => {
      const extra = row.unmatched_count > 0 ? ` ⚠${row.unmatched_count}` : ''
      return `${row.item_count ?? 0}${extra}`
    },
  },
  { title: '操作', key: 'actions', width: 160,
    render: (row: any) => h(NSpace, null, {
      default: () => [
        h(NButton, { size: 'small', type: 'primary', ghost: true, onClick: () => viewDetail(row.id) },
          { default: () => '对账' }),
        h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDelete(row) },
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
