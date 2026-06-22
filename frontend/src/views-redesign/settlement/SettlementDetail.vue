<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <a-button type="text" @click="router.push('/settlements')">&larr; 返回</a-button>
        <h2 style="margin: 0; display: inline-block; margin-left: 8px;">结算详情</h2>
        <span style="color: var(--ant-color-text-secondary, rgba(0,0,0,0.45)); font-size: 14px; margin-left: 8px;">结算明细与对账</span>
      </div>
      <a-space>
        <a-button :loading="reconciling" type="primary" @click="handleReconcile">执行对账</a-button>
        <a-button :type="settlement?.status === 'reconciled' ? 'primary' : 'default'"
                  :disabled="settlement?.status === 'reconciled'"
                  @click="handleChangeStatus('reconciled')">
          标记已对账
        </a-button>
      </a-space>
    </div>

    <!-- 结算单概览 -->
    <a-card style="margin-top: 12px;" :bordered="false" v-if="settlement">
      <a-row :gutter="12">
        <a-col :span="4">
          <a-statistic title="结算单号" :value="settlement.settlement_no" />
        </a-col>
        <a-col :span="4">
          <a-statistic title="平台" :value="settlement.platform_name || '-'" />
        </a-col>
        <a-col :span="4">
          <a-statistic title="状态">
            <template #formatter>
              <a-tag :color="statusTagColor">{{ statusLabel }}</a-tag>
            </template>
          </a-statistic>
        </a-col>
        <a-col :span="4">
          <a-statistic title="总收入" :value="settlement.total_revenue ?? 0" :precision="2" prefix="&yen;" />
        </a-col>
        <a-col :span="4">
          <a-statistic title="总费用" :value="settlement.total_fee ?? 0" :precision="2" prefix="&yen;" />
        </a-col>
        <a-col :span="4">
          <a-statistic title="净收入" :precision="2" prefix="&yen;">
            <template #formatter>
              <span :style="{ color: (settlement.total_net ?? 0) >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)', fontWeight: 700 }">
                {{ (settlement.total_net ?? 0).toFixed(2) }}
              </span>
            </template>
          </a-statistic>
        </a-col>
      </a-row>
    </a-card>

    <!-- 对账统计 -->
    <a-card style="margin-top: 12px;" :bordered="false" v-if="settlement">
      <template #title>
        <a-space align="center">
          <span>对账概览</span>
          <a-tag v-if="settlement.unmatched_count > 0" color="warning">
            {{ settlement.unmatched_count }} 条未匹配
          </a-tag>
          <a-tag v-if="settlement.discrepancy_count > 0" color="error">
            {{ settlement.discrepancy_count }} 条金额差异
          </a-tag>
        </a-space>
      </template>
      <a-row :gutter="12">
        <a-col :span="6">
          <a-statistic title="总明细" :value="settlement.item_count ?? 0" />
        </a-col>
        <a-col :span="6">
          <a-statistic title="已匹配">
            <template #formatter>
              <span style="color: var(--ant-color-success, #52c41a);">{{ settlement.matched_count ?? 0 }}</span>
            </template>
          </a-statistic>
        </a-col>
        <a-col :span="6">
          <a-statistic title="未匹配">
            <template #formatter>
              <span style="color: var(--ant-color-warning, #faad14);">{{ settlement.unmatched_count ?? 0 }}</span>
            </template>
          </a-statistic>
        </a-col>
        <a-col :span="6">
          <a-statistic title="金额差异">
            <template #formatter>
              <span style="color: var(--ant-color-error, #ff4d4f);">{{ settlement.discrepancy_count ?? 0 }}</span>
            </template>
          </a-statistic>
        </a-col>
      </a-row>
    </a-card>

    <!-- 筛选 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-space align="center">
        <a-select
          v-model:value="itemQuery.reconciliation_status"
          :options="reconStatusOptions"
          placeholder="对账状态"
          allow-clear
          style="width: 140px;"
          @change="fetchItems"
        />
        <a-select
          v-model:value="itemQuery.transaction_type"
          :options="txTypeOptions"
          placeholder="交易类型"
          allow-clear
          style="width: 140px;"
          @change="fetchItems"
        />
        <a-button @click="fetchItems" :loading="itemsLoading">查询</a-button>
      </a-space>
    </a-card>

    <!-- 明细表格 -->
    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="itemColumns"
        :data-source="items"
        :loading="itemsLoading"
        :pagination="{ current: itemQuery.page, pageSize: itemQuery.page_size, total: itemPagination.itemCount, showSizeChanger: true }"
        @change="onItemTableChange"
        row-key="id"
        :row-class-name="(record: any) => {
          if (record.reconciliation_status === 'discrepancy') return 'row-discrepancy'
          if (record.reconciliation_status === 'unmatched') return 'row-unmatched'
          return ''
        }"
      />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { message, Tag, Button, Space } from 'ant-design-vue'
import { useRouter, useRoute } from 'vue-router'
import { settlementApi } from '@/api/modules/settlement'

const router = useRouter()
const route = useRoute()

const loading = ref(false)
const itemsLoading = ref(false)
const reconciling = ref(false)
const settlement = ref<any>(null)
const items = ref<any[]>([])
const settlementId = ref(Number(route.params.id))

const reconStatusOptions = [
  { label: '待对账', value: 'pending' },
  { label: '已匹配', value: 'matched' },
  { label: '未匹配', value: 'unmatched' },
  { label: '金额差异', value: 'discrepancy' },
]

const txTypeOptions = [
  { label: '订单销售', value: 'order_sale' },
  { label: '退款', value: 'refund' },
  { label: '运费', value: 'shipping_fee' },
  { label: '平台费', value: 'platform_fee' },
  { label: '支付手续费', value: 'payment_fee' },
  { label: '其他', value: 'other' },
]

const itemQuery = reactive<{
  reconciliation_status: string | null
  transaction_type: string | null
  page: number
  page_size: number
}>({
  reconciliation_status: null,
  transaction_type: null,
  page: 1,
  page_size: 20,
})

const itemPagination = reactive({
  itemCount: 0,
})

let prevItemPageSize = itemQuery.page_size

function onItemTableChange(paginationInfo: any) {
  if (paginationInfo.pageSize !== prevItemPageSize) {
    itemQuery.page = 1
    prevItemPageSize = paginationInfo.pageSize
  } else {
    itemQuery.page = paginationInfo.current
  }
  itemQuery.page_size = paginationInfo.pageSize
  fetchItems()
}

const statusLabel = computed(() => {
  const map: Record<string, string> = {
    pending: '待对账',
    reconciling: '对账中',
    reconciled: '已对账',
    closed: '已关闭',
  }
  return map[settlement.value?.status] ?? settlement.value?.status ?? '-'
})

const statusTagColor = computed(() => {
  const map: Record<string, string> = {
    pending: 'warning',
    reconciling: 'processing',
    reconciled: 'success',
    closed: 'default',
  }
  return map[settlement.value?.status] ?? 'default'
})

async function fetchSettlement() {
  loading.value = true
  try {
    const res = await settlementApi.getById(settlementId.value)
    settlement.value = res.data?.data ?? null
  } catch {
    message.error('加载结算单失败')
  } finally {
    loading.value = false
  }
}

async function fetchItems() {
  itemsLoading.value = true
  try {
    const res = await settlementApi.listItems(settlementId.value, {
      reconciliation_status: itemQuery.reconciliation_status ?? undefined,
      transaction_type: itemQuery.transaction_type ?? undefined,
      page: itemQuery.page,
      page_size: itemQuery.page_size,
    })
    const body = res.data
    items.value = body?.data?.records ?? body?.records ?? []
    itemPagination.itemCount = body?.data?.total ?? body?.total ?? 0
  } catch {
    message.error('加载明细失败')
  } finally {
    itemsLoading.value = false
  }
}

async function handleReconcile() {
  reconciling.value = true
  try {
    const res = await settlementApi.reconcile(settlementId.value, {
      auto_match: true,
      strategy: 'by_order_no',
    })
    const result = res.data?.data ?? {}
    message.success(`对账完成: 匹配 ${result.matched ?? 0} / ${result.total ?? 0} 笔`)
    await fetchSettlement()
    await fetchItems()
  } catch (err: any) {
    message.error('对账失败: ' + (err.response?.data?.detail || err.message || ''))
  } finally {
    reconciling.value = false
  }
}

async function handleChangeStatus(status: string) {
  try {
    await settlementApi.update(settlementId.value, { status })
    message.success('状态已更新')
    await fetchSettlement()
  } catch {
    message.error('更新状态失败')
  }
}

async function handleUpdateReconciliation(itemId: number, status: string) {
  try {
    const note = status === 'matched' ? '手动匹配' : status === 'unmatched' ? '确认为未匹配' : undefined
    await settlementApi.updateReconciliation(itemId, { status, note })
    message.success('对账状态已更新')
    await fetchSettlement()
    await fetchItems()
  } catch {
    message.error('更新失败')
  }
}

const itemColumns = [
  {
    title: '交易类型', dataIndex: 'transaction_type', key: 'transaction_type', width: 130,
    customRender: ({ record }: any) => {
      const map: Record<string, string> = {
        order_sale: '订单销售',
        refund: '退款',
        shipping_fee: '运费',
        platform_fee: '平台费',
        payment_fee: '支付手续费',
        other: '其他',
      }
      return map[record.transaction_type] ?? record.transaction_type
    },
  },
  {
    title: '订单号', dataIndex: 'order_no', key: 'order_no', width: 160,
    customRender: ({ text }: any) => text || '-',
  },
  {
    title: '交易ID', dataIndex: 'transaction_id', key: 'transaction_id', width: 150,
    customRender: ({ text }: any) => text || '-',
  },
  {
    title: '金额', dataIndex: 'amount', key: 'amount', width: 110,
    customRender: ({ text }: any) => {
      const amt = text ?? 0
      return h('span', {
        style: `color: ${amt >= 0 ? 'var(--ant-color-success, #52c41a)' : 'var(--ant-color-error, #ff4d4f)'}`,
      }, `¥${amt.toFixed(2)}`)
    },
  },
  {
    title: '费用', dataIndex: 'fee', key: 'fee', width: 100,
    customRender: ({ text }: any) => `¥${(text ?? 0).toFixed(2)}`,
  },
  {
    title: '净额', dataIndex: 'net', key: 'net', width: 100,
    customRender: ({ text }: any) => `¥${(text ?? 0).toFixed(2)}`,
  },
  { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 60 },
  {
    title: '对账状态', dataIndex: 'reconciliation_status', key: 'reconciliation_status', width: 110,
    customRender: ({ record }: any) => {
      const map: Record<string, { label: string; color: string }> = {
        pending: { label: '待对账', color: 'warning' },
        matched: { label: '已匹配', color: 'success' },
        unmatched: { label: '未匹配', color: 'warning' },
        discrepancy: { label: '金额差异', color: 'error' },
      }
      const s = map[record.reconciliation_status] ?? { label: record.reconciliation_status, color: 'default' }
      return h(Tag, { color: s.color }, { default: () => s.label })
    },
  },
  {
    title: '备注', dataIndex: 'reconciliation_note', key: 'reconciliation_note', width: 200,
    customRender: ({ text }: any) => text || '-',
  },
  {
    title: '操作', key: 'actions', width: 200,
    customRender: ({ record }: any) => {
      if (record.reconciliation_status === 'matched') {
        return h(Button, {
          size: 'small',
          ghost: true,
          onClick: () => handleUpdateReconciliation(record.id, 'unmatched'),
        }, { default: () => '标记未匹配' })
      }
      return h(Space, null, {
        default: () => [
          h(Button, {
            size: 'small', type: 'primary', ghost: true,
            onClick: () => handleUpdateReconciliation(record.id, 'matched'),
          }, { default: () => '匹配' }),
          h(Button, {
            size: 'small', ghost: true, style: 'color: var(--ant-color-warning, #faad14); border-color: var(--ant-color-warning, #faad14);',
            onClick: () => handleUpdateReconciliation(record.id, 'discrepancy'),
          }, { default: () => '差异' }),
        ],
      })
    },
  },
]

onMounted(async () => {
  await fetchSettlement()
  await fetchItems()
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
:deep(.row-discrepancy) {
  background: #fff1f0;
}
:deep(.row-unmatched) {
  background: #fffbe6;
}
</style>
