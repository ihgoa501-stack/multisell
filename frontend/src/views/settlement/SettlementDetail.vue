<template>
  <div>
    <n-page-header subtitle="结算明细与对账" @back="router.push('/settlements')">
      <template #title>💰 结算详情</template>
      <template #extra>
        <n-space>
          <n-button :loading="reconciling" type="primary" @click="handleReconcile">🔄 执行对账</n-button>
          <n-button :type="settlement?.status === 'reconciled' ? 'success' : 'default'"
                    @click="handleChangeStatus('reconciled')" :disabled="settlement?.status === 'reconciled'">
            ✅ 标记已对账
          </n-button>
        </n-space>
      </template>
    </n-page-header>

    <!-- 结算单概览 -->
    <n-card style="margin-top: 12px;" :bordered="false" v-if="settlement">
      <n-grid :cols="6" :x-gap="12">
        <n-grid-item>
          <n-statistic label="结算单号" :value="settlement.settlement_no" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="平台" :value="settlement.platform_name || '-'" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="状态">
            <n-tag :type="statusTagType" size="small">{{ statusLabel }}</n-tag>
          </n-statistic>
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="总收入" :value="`¥${(settlement.total_revenue ?? 0).toFixed(2)}`" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="总费用" :value="`¥${(settlement.total_fee ?? 0).toFixed(2)}`" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="净收入">
            <span :style="{ color: (settlement.total_net ?? 0) >= 0 ? '#18a058' : '#d03050', fontWeight: 700 }">
              ¥{{ (settlement.total_net ?? 0).toFixed(2) }}
            </span>
          </n-statistic>
        </n-grid-item>
      </n-grid>
    </n-card>

    <!-- 对账统计 -->
    <n-card style="margin-top: 12px;" :bordered="false" v-if="settlement">
      <template #header>
        <n-space align="center">
          <span>📊 对账概览</span>
          <n-tag v-if="settlement.unmatched_count > 0" type="warning" size="small">
            {{ settlement.unmatched_count }} 条未匹配
          </n-tag>
          <n-tag v-if="settlement.discrepancy_count > 0" type="error" size="small">
            {{ settlement.discrepancy_count }} 条金额差异
          </n-tag>
        </n-space>
      </template>
      <n-grid :cols="4" :x-gap="12">
        <n-grid-item>
          <n-statistic label="总明细" :value="settlement.item_count ?? 0" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="已匹配">
            <span style="color: #18a058;">{{ settlement.matched_count ?? 0 }}</span>
          </n-statistic>
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="未匹配">
            <span style="color: #f0a020;">{{ settlement.unmatched_count ?? 0 }}</span>
          </n-statistic>
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="金额差异">
            <span style="color: #d03050;">{{ settlement.discrepancy_count ?? 0 }}</span>
          </n-statistic>
        </n-grid-item>
      </n-grid>
    </n-card>

    <!-- 筛选 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space align="center">
        <n-select
          v-model:value="itemQuery.reconciliation_status"
          :options="reconStatusOptions"
          placeholder="对账状态"
          clearable
          style="width: 140px;"
          @update:value="fetchItems"
        />
        <n-select
          v-model:value="itemQuery.transaction_type"
          :options="txTypeOptions"
          placeholder="交易类型"
          clearable
          style="width: 140px;"
          @update:value="fetchItems"
        />
        <n-button @click="fetchItems" :loading="itemsLoading">查询</n-button>
      </n-space>
    </n-card>

    <!-- 明细表格 -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table
        :columns="itemColumns"
        :data="items"
        :loading="itemsLoading"
        :pagination="itemPagination"
        @update:page="onItemPageChange"
        @update:page-size="onItemPageSizeChange"
        :row-key="(row: any) => row.id"
        :row-props="(row: any) => ({
          style: row.reconciliation_status === 'discrepancy'
            ? 'background: #fff1f0;'
            : row.reconciliation_status === 'unmatched'
              ? 'background: #fffbe6;'
              : undefined
        })"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { NButton, NTag, NSpace, NSelect, useMessage, useDialog } from 'naive-ui'
import { useRouter, useRoute } from 'vue-router'
import { settlementApi } from '@/api/modules/settlement'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const dialog = useDialog()

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
  page: 1,
  pageSize: 20,
  itemCount: 0,
  onChange: (page: number) => { itemQuery.page = page; fetchItems() },
  onUpdatePageSize: (size: number) => { itemQuery.page_size = size; itemQuery.page = 1; fetchItems() },
})

function onItemPageChange(page: number) {
  itemQuery.page = page
  fetchItems()
}

function onItemPageSizeChange(size: number) {
  itemQuery.page_size = size
  itemQuery.page = 1
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

const statusTagType = computed(() => {
  const map: Record<string, string> = {
    pending: 'warning',
    reconciling: 'info',
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
  { title: '交易类型', key: 'transaction_type', width: 130,
    render: (row: any) => {
      const map: Record<string, string> = {
        order_sale: '订单销售',
        refund: '退款',
        shipping_fee: '运费',
        platform_fee: '平台费',
        payment_fee: '支付手续费',
        other: '其他',
      }
      return map[row.transaction_type] ?? row.transaction_type
    },
  },
  { title: '订单号', key: 'order_no', width: 160,
    render: (row: any) => row.order_no || '-',
  },
  { title: '交易ID', key: 'transaction_id', width: 150,
    render: (row: any) => row.transaction_id || '-',
  },
  { title: '金额', key: 'amount', width: 110,
    render: (row: any) => {
      const amt = row.amount ?? 0
      const color = amt >= 0 ? '#18a058' : '#d03050'
      return h('span', { style: `color: ${color}` }, `¥${amt.toFixed(2)}`)
    },
  },
  { title: '费用', key: 'fee', width: 100,
    render: (row: any) => `¥${(row.fee ?? 0).toFixed(2)}`,
  },
  { title: '净额', key: 'net', width: 100,
    render: (row: any) => `¥${(row.net ?? 0).toFixed(2)}`,
  },
  { title: '数量', key: 'quantity', width: 60 },
  { title: '对账状态', key: 'reconciliation_status', width: 110,
    render: (row: any) => {
      const map: Record<string, { label: string; type: string }> = {
        pending: { label: '待对账', type: 'warning' },
        matched: { label: '已匹配', type: 'success' },
        unmatched: { label: '未匹配', type: 'warning' },
        discrepancy: { label: '金额差异', type: 'error' },
      }
      const s = map[row.reconciliation_status] ?? { label: row.reconciliation_status, type: 'default' }
      return h(NTag, { type: s.type as any, size: 'small' }, { default: () => s.label })
    },
  },
  { title: '备注', key: 'reconciliation_note', width: 200,
    render: (row: any) => row.reconciliation_note || '-',
  },
  { title: '操作', key: 'actions', width: 200,
    render: (row: any) => {
      if (row.reconciliation_status === 'matched') {
        return h(NButton, {
          size: 'small',
          ghost: true,
          onClick: () => handleUpdateReconciliation(row.id, 'unmatched'),
        }, { default: () => '标记未匹配' })
      }
      return h(NSpace, null, {
        default: () => [
          h(NButton, {
            size: 'small', type: 'success', ghost: true,
            onClick: () => handleUpdateReconciliation(row.id, 'matched'),
          }, { default: () => '匹配' }),
          h(NButton, {
            size: 'small', type: 'warning', ghost: true,
            onClick: () => handleUpdateReconciliation(row.id, 'discrepancy'),
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
