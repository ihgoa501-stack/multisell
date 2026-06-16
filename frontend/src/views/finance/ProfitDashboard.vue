<template>
  <div>
    <n-page-header subtitle="基于 ledger 的真实利润与成本差异看板">
      <template #title>利润看板</template>
    </n-page-header>

    <!-- Date filter -->
    <n-card style="margin-top: 12px;" :bordered="false">
      <n-space>
        <n-date-picker v-model:value="dateRange" type="daterange" clearable style="width: 260px;" />
        <n-button type="primary" :loading="loading" @click="fetchAll">查询</n-button>
      </n-space>
    </n-card>

    <!-- Profit summary -->
    <n-card title="利润摘要" style="margin-top: 12px;">
      <n-grid :cols="5" :x-gap="12">
        <n-grid-item>
          <n-statistic label="收入" :value="summary.revenue_amount" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="商品成本" :value="summary.product_cost" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="运费" :value="summary.shipping_cost" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="平台费" :value="summary.platform_fee" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="支付费" :value="summary.payment_fee" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="退款" :value="summary.refund" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="分摊成本" :value="summary.allocated_cost" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="其他费用" :value="summary.other_fee" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="利润" :value="summary.profit_amount" :style="{ color: summary.profit_amount >= 0 ? 'green' : 'red' }" />
        </n-grid-item>
        <n-grid-item>
          <n-statistic label="利润率" :value="summary.profit_margin" precision="2">
            <template #suffix>%</template>
          </n-statistic>
        </n-grid-item>
      </n-grid>
    </n-card>

    <!-- Cost layer mix -->
    <n-card title="成本层分布" style="margin-top: 12px;">
      <n-data-table :columns="layerColumns" :data="layerMix.layers || []" size="small" :pagination="false" />
    </n-card>

    <!-- Cost variance -->
    <n-card title="运费差异" style="margin-top: 12px;">
      <n-data-table :columns="varianceColumns" :data="costVariance" :loading="loading" size="small" :pagination="{ pageSize: 10 }" />
    </n-card>

    <!-- Negative profit -->
    <n-card v-if="negativeProfit.length > 0" title="负利润订单" style="margin-top: 12px;">
      <n-data-table :columns="negColumns" :data="negativeProfit" size="small" :pagination="{ pageSize: 10 }" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, ref, reactive } from 'vue'
import { NTag, useMessage } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import {
  getProfitSummary,
  getCostVariance,
  getNegativeProfit,
  getCostLayerMix,
} from '@/api/modules/financeReports'

const message = useMessage()
const loading = ref(false)
const dateRange = ref<[number, number] | null>(null)

const summary = reactive({
  revenue_amount: 0, product_cost: 0, shipping_cost: 0, platform_fee: 0,
  payment_fee: 0, refund: 0, allocated_cost: 0, other_fee: 0,
  profit_amount: 0, profit_margin: 0,
})
const costVariance = ref<any[]>([])
const negativeProfit = ref<any[]>([])
const layerMix = ref<{ layers: any[] }>({ layers: [] })

const layerColumns = [
  { title: '成本层', key: 'cost_layer', width: 120, render: (row: any) => h(CostLayerTag, { layer: row.cost_layer }) },
  { title: '条目数', key: 'entry_count', width: 80 },
  { title: '总金额', key: 'total_amount', width: 120 },
]

const varianceColumns = [
  { title: '订单号', key: 'order_no', width: 160 },
  { title: '快照运费', key: 'snapshot_amount', width: 100 },
  { title: '账单运费', key: 'bill_amount', width: 100 },
  { title: '差异金额', key: 'variance_amount', width: 100 },
  { title: '差异%', key: 'variance_pct', width: 80, render: (row: any) => row.variance_pct == null ? '-' : `${row.variance_pct}%` },
  { title: '状态', key: 'status', width: 100, render: (row: any) => h(NTag, { size: 'small', type: row.status === 'matched' ? 'success' : 'warning' }, { default: () => row.status }) },
]

const negColumns = [
  { title: '订单号', key: 'order_no', width: 160 },
  { title: '利润', key: 'profit_amount', width: 100, render: (row: any) => h('span', { style: 'color:red' }, row.profit_amount.toFixed(2)) },
  { title: '利润率', key: 'profit_margin', width: 80, render: (row: any) => `${row.profit_margin}%` },
  { title: '运费来源', key: 'shipping_cost_layer', width: 100, render: (row: any) => h(CostLayerTag, { layer: row.shipping_cost_layer }) },
  { title: '平台费来源', key: 'platform_fee_cost_layer', width: 100, render: (row: any) => h(CostLayerTag, { layer: row.platform_fee_cost_layer }) },
  { title: '利润来源', key: 'profit_cost_layer', width: 100, render: (row: any) => h(CostLayerTag, { layer: row.profit_cost_layer }) },
]

function getParams() {
  const params: any = {}
  if (dateRange.value) {
    params.date_from = new Date(dateRange.value[0]).toISOString().split('T')[0]
    params.date_to = new Date(dateRange.value[1]).toISOString().split('T')[0]
  }
  return params
}

async function fetchAll() {
  loading.value = true
  const params = getParams()
  try {
    const [sumResp, varResp, negResp, layerResp] = await Promise.all([
      getProfitSummary(params),
      getCostVariance(params),
      getNegativeProfit(params),
      getCostLayerMix(params),
    ])
    Object.assign(summary, sumResp.data || {})
    costVariance.value = varResp.data || []
    negativeProfit.value = negResp.data || []
    layerMix.value = layerResp.data || { layers: [] }
  } catch (err: any) {
    message.error(err?.message || '查询失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchAll)
</script>
