<template>
  <n-card title="利润账本" :bordered="false" class="mt-3">
    <template #header-extra>
      <n-button
        size="small"
        type="primary"
        :loading="rebuilding"
        @click="handleRebuild"
      >
        重建账本
      </n-button>
    </template>

    <n-spin :show="loading">

      <!-- 汇总行 -->
      <n-descriptions v-if="profit" :column="3" label-placement="left" bordered size="small">
        <n-descriptions-item label="商品收入">¥{{ money(profit.revenue_amount) }}</n-descriptions-item>
        <n-descriptions-item label="商品成本">¥{{ money(profit.product_cost) }}</n-descriptions-item>
        <n-descriptions-item label="运费">
          ¥{{ money(profit.shipping_cost) }}
          <CostLayerTag :layer="profit.shipping_cost_layer" />
        </n-descriptions-item>
        <n-descriptions-item label="平台费">
          ¥{{ money(profit.platform_fee) }}
          <CostLayerTag :layer="profit.platform_fee_cost_layer" />
        </n-descriptions-item>
        <n-descriptions-item label="支付费">¥{{ money(profit.payment_fee) }}</n-descriptions-item>
        <n-descriptions-item label="退款">¥{{ money(profit.refund) }}</n-descriptions-item>
        <n-descriptions-item label="调整">¥{{ money(profit.adjustment) }}</n-descriptions-item>
        <n-descriptions-item label="其他费用">¥{{ money(profit.other_fee) }}</n-descriptions-item>
        <n-descriptions-item label="利润金额">
          <span :style="{ color: profit.profit_amount >= 0 ? 'green' : 'red', fontWeight: 'bold' }">
            ¥{{ money(profit.profit_amount) }}
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="利润率">
          <span :style="{ color: profit.profit_margin >= 0 ? 'green' : 'red' }">
            {{ profit.profit_margin }}%
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="利润来源">
          <CostLayerTag :layer="profit.profit_cost_layer" />
        </n-descriptions-item>
      </n-descriptions>

      <n-empty v-else-if="!loading" description="点击「重建账本」生成利润账本" />

      <!-- 明细条目 -->
      <n-data-table
        v-if="ledgerEntries.length > 0"
        :columns="entryColumns"
        :data="ledgerEntries"
        :pagination="{ pageSize: 20 }"
        striped
        size="small"
        style="margin-top: 12px;"
      />
    </n-spin>
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useMessage } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import { rebuildOrderLedger, getOrderLedger, getOrderProfit } from '@/api/modules/finance'

const props = defineProps<{
  orderId: number
}>()

const message = useMessage()

const loading = ref(false)
const rebuilding = ref(false)
const profit = ref<any>(null)
const ledgerEntries = ref<any[]>([])

const entryTypeMap: Record<string, string> = {
  revenue: '商品收入',
  product_cost: '商品成本',
  shipping_cost: '运费',
  platform_fee: '平台费',
  payment_fee: '支付费',
  refund: '退款',
  adjustment: '调整',
  other_fee: '其他费用',
}

const entryColumns = [
  { title: '类型', key: 'entry_type', width: 120,
    render: (row: any) => entryTypeMap[row.entry_type] || row.entry_type
  },
  { title: '金额', key: 'amount', width: 100,
    render: (row: any) => {
      const color = row.amount >= 0 ? 'green' : 'red'
      return h('span', { style: `color:${color}` }, row.amount.toFixed(2))
    }
  },
  { title: '币种', key: 'currency', width: 60 },
  { title: '来源', key: 'cost_layer', width: 80,
    render: (row: any) => h(CostLayerTag, { layer: row.cost_layer })
  },
  { title: '来源类型', key: 'source_type', width: 120 },
  { title: '说明', key: 'description', ellipsis: { tooltip: true } },
]

function money(val: number | undefined | null): string {
  return (val ?? 0).toFixed(2)
}

async function handleRebuild() {
  rebuilding.value = true
  try {
    const resp = await rebuildOrderLedger(props.orderId)
    profit.value = resp.data
    message.success('账本重建完成')
    await fetchLedger()
  } catch (err: any) {
    message.error(err?.response?.data?.message || err?.message || '重建失败')
  } finally {
    rebuilding.value = false
  }
}

async function fetchLedger() {
  loading.value = true
  try {
    const [profitResp, ledgerResp] = await Promise.all([
      getOrderProfit(props.orderId),
      getOrderLedger(props.orderId),
    ])
    profit.value = profitResp.data
    ledgerEntries.value = ledgerResp.data?.entries || []
  } catch {
    // ignore — ledger not yet built
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchLedger()
})
</script>

<style scoped>
.mt-3 { margin-top: 12px; }
</style>
