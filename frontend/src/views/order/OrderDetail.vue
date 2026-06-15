<template>
  <div>
    <n-page-header @back="router.back()">
      <template #title>📄 订单详情</template>
      <template #extra>
        <n-button v-if="detail.status === 'paid'" type="primary" :loading="shipping" @click="handleShip">
          🚚 确认发货
        </n-button>
      </template>
    </n-page-header>

    <n-spin :show="loading">
      <!-- 基本信息 -->
      <n-card title="基本信息" style="margin-top: 12px;" :bordered="false">
        <n-descriptions label-placement="left" :column="2">
          <n-descriptions-item label="订单号">{{ detail.order_no || '-' }}</n-descriptions-item>
          <n-descriptions-item label="状态">
            <n-tag :type="statusTagType" size="small">{{ statusLabel }}</n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="收件人">{{ detail.recipient_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="联系电话">{{ detail.recipient_phone || '-' }}</n-descriptions-item>
          <n-descriptions-item label="收货地址" :span="2">{{ detail.shipping_address || '-' }}</n-descriptions-item>
          <n-descriptions-item label="商品总额">¥{{ ((detail.total_amount ?? 0)).toFixed(2) }}</n-descriptions-item>
          <n-descriptions-item label="运费">¥{{ ((detail.shipping_fee ?? 0)).toFixed(2) }}</n-descriptions-item>
          <n-descriptions-item label="实付金额">
            <span style="color: #d03050; font-weight: bold;">¥{{ ((detail.pay_amount ?? detail.total_amount ?? 0)).toFixed(2) }}</span>
          </n-descriptions-item>
          <n-descriptions-item label="支付方式">{{ detail.payment_method || '-' }}</n-descriptions-item>
          <n-descriptions-item label="下单时间">{{ detail.created_at || '-' }}</n-descriptions-item>
          <n-descriptions-item label="支付时间">{{ detail.paid_at || '-' }}</n-descriptions-item>
          <n-descriptions-item label="发货时间">{{ detail.shipped_at || '-' }}</n-descriptions-item>
          <n-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</n-descriptions-item>
        </n-descriptions>
      </n-card>

      <!-- 运费快照 -->
      <n-card title="运费快照" style="margin-top: 12px;" :bordered="false">
        <n-space style="margin-bottom: 12px;" align="center">
          <n-input v-model:value="shippingForm.destination_country" placeholder="目的国家，如 US" style="width: 140px;" />
          <n-input v-model:value="shippingForm.postal_code" placeholder="邮编，可选" style="width: 140px;" />
          <n-input v-model:value="shippingForm.cargo_type" placeholder="货品类型" style="width: 120px;" />
          <n-button type="primary" :loading="bindingShipping" @click="handleBindShippingQuote">
            计算并保存最低运费
          </n-button>
        </n-space>
        <n-empty v-if="!detail.shipping_snapshot" description="暂无运费快照" />
        <n-descriptions v-else :column="2" label-placement="left">
          <n-descriptions-item label="物流商">{{ detail.shipping_snapshot.provider_name }}</n-descriptions-item>
          <n-descriptions-item label="渠道">{{ detail.shipping_snapshot.channel_name }}</n-descriptions-item>
          <n-descriptions-item label="目的地">{{ detail.shipping_snapshot.destination_country }}</n-descriptions-item>
          <n-descriptions-item label="币种">{{ detail.shipping_snapshot.currency }}</n-descriptions-item>
          <n-descriptions-item label="实际重">{{ detail.shipping_snapshot.actual_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="体积重">{{ detail.shipping_snapshot.volumetric_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="计费重">{{ detail.shipping_snapshot.chargeable_weight_kg }} kg</n-descriptions-item>
          <n-descriptions-item label="总运费">¥{{ money(detail.shipping_snapshot.total_shipping_fee) }}</n-descriptions-item>
          <n-descriptions-item label="计算说明" :span="2">{{ detail.shipping_snapshot.calculation_detail || '-' }}</n-descriptions-item>
        </n-descriptions>
      </n-card>

      <!-- 利润测算 -->
      <n-card title="利润测算" style="margin-top: 12px;" :bordered="false">
        <n-descriptions :column="3" label-placement="left">
          <n-descriptions-item label="销售额">¥{{ money(detail.profit?.revenue_amount) }}</n-descriptions-item>
          <n-descriptions-item label="商品成本">¥{{ money(detail.profit?.product_cost) }}</n-descriptions-item>
          <n-descriptions-item label="运费">¥{{ money(detail.profit?.shipping_fee) }}
            <CostLayerTag v-if="detail.profit?.shipping_cost_layer" :layer="detail.profit.shipping_cost_layer" />
          </n-descriptions-item>
          <n-descriptions-item label="平台费">¥{{ money(detail.profit?.platform_fee) }}
            <CostLayerTag v-if="detail.profit?.platform_fee_cost_layer" :layer="detail.profit.platform_fee_cost_layer" />
          </n-descriptions-item>
          <n-descriptions-item label="支付费">¥{{ money(detail.profit?.payment_fee) }}</n-descriptions-item>
          <n-descriptions-item label="其他费用">¥{{ money(detail.profit?.other_fee) }}</n-descriptions-item>
          <n-descriptions-item label="利润">¥{{ money(detail.profit?.profit_amount) }}</n-descriptions-item>
          <n-descriptions-item label="利润率">{{ money(detail.profit?.profit_margin) }}%</n-descriptions-item>
          <n-descriptions-item label="利润来源">
            <CostLayerTag v-if="detail.profit?.profit_cost_layer" :layer="detail.profit.profit_cost_layer" />
            <span v-else>-</span>
          </n-descriptions-item>
        </n-descriptions>
        <n-space style="margin-top: 12px;" align="center">
          <n-input-number v-model:value="profitForm.product_cost" placeholder="商品成本" :min="0" />
          <n-input-number v-model:value="profitForm.platform_fee" placeholder="平台费" :min="0" />
          <n-input-number v-model:value="profitForm.payment_fee" placeholder="支付费" :min="0" />
          <n-input-number v-model:value="profitForm.other_fee" placeholder="其他费用" :min="0" />
          <n-button :loading="savingProfit" @click="handleSaveProfitInputs">保存利润输入</n-button>
        </n-space>
      </n-card>

      <!-- 商品明细 -->
      <n-card title="商品明细" style="margin-top: 12px;" :bordered="false">
        <n-data-table :columns="itemColumns" :data="detail.items || []" :bordered="false" :max-height="360" />
      </n-card>

      <!-- 状态流转记录 -->
      <n-card title="状态流转" style="margin-top: 12px;" :bordered="false">
        <n-empty v-if="!(detail.status_logs?.length)" description="暂无流转记录" />
        <n-timeline v-else>
          <n-timeline-item
            v-for="log in detail.status_logs"
            :key="log.id"
            :type="log.is_current ? 'info' : 'default'"
            :time="log.created_at"
          >
            <template #header>
              <n-tag size="tiny" :type="getStatusTagType(log.from_status)" v-if="log.from_status">
                {{ getStatusLabel(log.from_status) }}
              </n-tag>
              <span style="margin: 0 6px;">→</span>
              <n-tag size="tiny" :type="getStatusTagType(log.to_status)">
                {{ getStatusLabel(log.to_status) }}
              </n-tag>
            </template>
            <template #default>
              <span style="font-size: 13px; color: #666;">{{ log.operator || '系统' }}</span>
              <span v-if="log.remark" style="margin-left: 8px; color: #999;">{{ log.remark }}</span>
            </template>
          </n-timeline-item>
        </n-timeline>
      </n-card>

      <!-- 利润账本 -->
      <OrderProfitLedger v-if="detail.id" :order-id="detail.id" />
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NTag, useMessage, useDialog } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import OrderProfitLedger from '@/components/OrderProfitLedger.vue'
import { apiModules } from '@/api'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const dialog = useDialog()

const orderId = route.params.id as string
const loading = ref(false)
const shipping = ref(false)
const bindingShipping = ref(false)
const savingProfit = ref(false)
const detail = ref<any>({})
const shippingForm = ref({
  destination_country: 'US',
  postal_code: '',
  cargo_type: 'normal',
})
const profitForm = ref({
  product_cost: 0,
  platform_fee: 0,
  payment_fee: 0,
  other_fee: 0,
})

const statusTagMap: Record<string, { type: 'success' | 'warning' | 'error' | 'info' | 'default'; label: string }> = {
  pending: { type: 'warning', label: '待付款' },
  paid: { type: 'info', label: '待发货' },
  shipped: { type: 'info', label: '已发货' },
  delivered: { type: 'success', label: '已签收' },
  completed: { type: 'success', label: '已完成' },
  cancelled: { type: 'error', label: '已取消' },
}

const statusTagType = computed(() => statusTagMap[detail.value?.status]?.type || 'default')
const statusLabel = computed(() => statusTagMap[detail.value?.status]?.label || detail.value?.status || '-')

function getStatusTagType(s: string) {
  return statusTagMap[s]?.type || 'default'
}

function getStatusLabel(s: string) {
  return statusTagMap[s]?.label || s
}

function money(value: any) {
  return Number(value || 0).toFixed(2)
}

const itemColumns = [
  { title: '商品名称', key: 'product_name', ellipsis: { tooltip: true } },
  { title: 'SKU', key: 'sku_code', width: 140 },
  { title: '规格', key: 'spec_desc', width: 140 },
  { title: '单价', key: 'unit_price', width: 100, render: (row: any) => `¥${(row.unit_price || 0).toFixed(2)}` },
  { title: '数量', key: 'quantity', width: 80 },
  { title: '小计', key: 'subtotal', width: 100, render: (row: any) => `¥${(row.subtotal || 0).toFixed(2)}` },
]

async function fetchDetail() {
  loading.value = true
  try {
    const res: any = await apiModules.orderApi.getById(orderId)
    detail.value = res?.data || res || {}
    profitForm.value = {
      product_cost: Number(detail.value.product_cost || 0),
      platform_fee: Number(detail.value.platform_fee || 0),
      payment_fee: Number(detail.value.payment_fee || 0),
      other_fee: Number(detail.value.other_fee || 0),
    }
  } catch (e: any) {
    message.error(e.message || '加载订单详情失败')
  } finally {
    loading.value = false
  }
}

async function handleShip() {
  dialog.warning({
    title: '确认发货',
    content: '确定要将此订单标记为已发货吗？',
    positiveText: '确认发货',
    negativeText: '取消',
    onPositiveClick: async () => {
      shipping.value = true
      try {
        await apiModules.orderApi.updateStatus(orderId, 'shipped')
        message.success('发货成功')
        fetchDetail()
      } catch (e: any) {
        message.error(e.message || '发货失败')
      } finally {
        shipping.value = false
      }
    },
  })
}

async function handleBindShippingQuote() {
  const firstItem = detail.value?.items?.[0]
  if (!firstItem?.sku_id) {
    message.error('订单缺少 SKU，无法计算运费')
    return
  }
  bindingShipping.value = true
  try {
    const res: any = await apiModules.orderApi.bindShippingQuote(orderId, {
      sku_id: firstItem.sku_id,
      quantity: firstItem.quantity || 1,
      destination_country: shippingForm.value.destination_country,
      postal_code: shippingForm.value.postal_code || undefined,
      cargo_type: shippingForm.value.cargo_type || 'normal',
      channel_id: null,
    })
    detail.value = res?.data || res || {}
    message.success('已保存运费快照')
  } catch (e: any) {
    message.error(e.message || '保存运费快照失败')
  } finally {
    bindingShipping.value = false
  }
}

async function handleSaveProfitInputs() {
  savingProfit.value = true
  try {
    const res: any = await apiModules.orderApi.updateProfitInputs(orderId, profitForm.value)
    detail.value = res?.data || res || {}
    message.success('利润输入已保存')
  } catch (e: any) {
    message.error(e.message || '保存利润输入失败')
  } finally {
    savingProfit.value = false
  }
}

onMounted(fetchDetail)
</script>
