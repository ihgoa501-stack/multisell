<template>
  <div>
    <!-- Page Header -->
    <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; padding-bottom: 16px; border-bottom: 1px solid #f0f0f0;">
      <div style="display: flex; align-items: center; gap: 12px;">
        <a-button type="text" @click="router.back()">
          <template #icon><ArrowLeftOutlined /></template>
        </a-button>
        <a-typography-title :level="4" style="margin: 0;">订单详情</a-typography-title>
      </div>
      <a-button v-if="detail.status === 'paid'" type="primary" :loading="shippingState" @click="handleShip">
        确认发货
      </a-button>
    </div>

    <a-spin :spinning="loading">
      <!-- 基本信息 -->
      <a-card title="基本信息" style="margin-top: 12px;" :bordered="false">
        <a-descriptions :column="2" layout="horizontal">
          <a-descriptions-item label="订单号">{{ detail.order_no || '-' }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="statusTagColor">{{ statusLabel }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="收件人">{{ detail.recipient_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="联系电话">{{ detail.recipient_phone || '-' }}</a-descriptions-item>
          <a-descriptions-item label="收货地址" :span="2">{{ detail.shipping_address || '-' }}</a-descriptions-item>
          <a-descriptions-item label="商品总额">¥{{ ((detail.total_amount ?? 0)).toFixed(2) }}</a-descriptions-item>
          <a-descriptions-item label="运费">¥{{ ((detail.shipping_fee ?? 0)).toFixed(2) }}</a-descriptions-item>
          <a-descriptions-item label="实付金额">
            <span style="color: #ff4d4f; font-weight: bold;">¥{{ ((detail.pay_amount ?? detail.total_amount ?? 0)).toFixed(2) }}</span>
          </a-descriptions-item>
          <a-descriptions-item label="支付方式">{{ detail.payment_method || '-' }}</a-descriptions-item>
          <a-descriptions-item label="下单时间">{{ detail.created_at || '-' }}</a-descriptions-item>
          <a-descriptions-item label="支付时间">{{ detail.paid_at || '-' }}</a-descriptions-item>
          <a-descriptions-item label="发货时间">{{ detail.shipped_at || '-' }}</a-descriptions-item>
          <a-descriptions-item label="备注" :span="2">{{ detail.remark || '-' }}</a-descriptions-item>
        </a-descriptions>
      </a-card>

      <!-- 运费快照 -->
      <a-card title="运费快照" style="margin-top: 12px;" :bordered="false">
        <a-space style="margin-bottom: 12px;" align="center">
          <a-input v-model:value="shippingForm.destination_country" placeholder="目的国家，如 US" style="width: 140px;" />
          <a-input v-model:value="shippingForm.postal_code" placeholder="邮编，可选" style="width: 140px;" />
          <a-input v-model:value="shippingForm.cargo_type" placeholder="货品类型" style="width: 120px;" />
          <a-button type="primary" :loading="bindingShipping" @click="handleBindShippingQuote">
            计算并保存最低运费
          </a-button>
        </a-space>
        <a-empty v-if="!detail.shipping_snapshot" description="暂无运费快照" />
        <a-descriptions v-else :column="2" layout="horizontal">
          <a-descriptions-item label="物流商">{{ detail.shipping_snapshot.provider_name }}</a-descriptions-item>
          <a-descriptions-item label="渠道">{{ detail.shipping_snapshot.channel_name }}</a-descriptions-item>
          <a-descriptions-item label="目的地">{{ detail.shipping_snapshot.destination_country }}</a-descriptions-item>
          <a-descriptions-item label="币种">{{ detail.shipping_snapshot.currency }}</a-descriptions-item>
          <a-descriptions-item label="实际重">{{ detail.shipping_snapshot.actual_weight_kg }} kg</a-descriptions-item>
          <a-descriptions-item label="体积重">{{ detail.shipping_snapshot.volumetric_weight_kg }} kg</a-descriptions-item>
          <a-descriptions-item label="计费重">{{ detail.shipping_snapshot.chargeable_weight_kg }} kg</a-descriptions-item>
          <a-descriptions-item label="总运费">¥{{ money(detail.shipping_snapshot.total_shipping_fee) }}</a-descriptions-item>
          <a-descriptions-item label="计算说明" :span="2">{{ detail.shipping_snapshot.calculation_detail || '-' }}</a-descriptions-item>
        </a-descriptions>
      </a-card>

      <!-- 利润测算 -->
      <a-card title="利润测算" style="margin-top: 12px;" :bordered="false">
        <a-descriptions :column="3" layout="horizontal">
          <a-descriptions-item label="销售额">¥{{ money(detail.profit?.revenue_amount) }}</a-descriptions-item>
          <a-descriptions-item label="商品成本">¥{{ money(detail.profit?.product_cost) }}</a-descriptions-item>
          <a-descriptions-item label="运费">
            ¥{{ money(detail.profit?.shipping_fee) }}
            <CostLayerTag v-if="detail.profit?.shipping_cost_layer" :layer="detail.profit.shipping_cost_layer" />
          </a-descriptions-item>
          <a-descriptions-item label="平台费">
            ¥{{ money(detail.profit?.platform_fee) }}
            <CostLayerTag v-if="detail.profit?.platform_fee_cost_layer" :layer="detail.profit.platform_fee_cost_layer" />
          </a-descriptions-item>
          <a-descriptions-item label="支付费">¥{{ money(detail.profit?.payment_fee) }}</a-descriptions-item>
          <a-descriptions-item label="其他费用">¥{{ money(detail.profit?.other_fee) }}</a-descriptions-item>
          <a-descriptions-item label="利润">¥{{ money(detail.profit?.profit_amount) }}</a-descriptions-item>
          <a-descriptions-item label="利润率">{{ money(detail.profit?.profit_margin) }}%</a-descriptions-item>
          <a-descriptions-item label="利润来源">
            <CostLayerTag v-if="detail.profit?.profit_cost_layer" :layer="detail.profit.profit_cost_layer" />
            <span v-else>-</span>
          </a-descriptions-item>
        </a-descriptions>
        <a-space style="margin-top: 12px;" align="center">
          <a-input-number v-model:value="profitForm.product_cost" placeholder="商品成本" :min="0" />
          <a-input-number v-model:value="profitForm.platform_fee" placeholder="平台费" :min="0" />
          <a-input-number v-model:value="profitForm.payment_fee" placeholder="支付费" :min="0" />
          <a-input-number v-model:value="profitForm.other_fee" placeholder="其他费用" :min="0" />
          <a-button :loading="savingProfit" @click="handleSaveProfitInputs">保存利润输入</a-button>
        </a-space>
      </a-card>

      <!-- 商品明细 -->
      <a-card title="商品明细" style="margin-top: 12px;" :bordered="false">
        <a-table
          :columns="itemColumns"
          :data-source="detail.items || []"
          :bordered="false"
          :pagination="false"
          :scroll="{ y: 360 }"
          row-key="id"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'unit_price'">
              {{ `¥${(record.unit_price || 0).toFixed(2)}` }}
            </template>
            <template v-else-if="column.key === 'subtotal'">
              {{ `¥${(record.subtotal || 0).toFixed(2)}` }}
            </template>
          </template>
        </a-table>
      </a-card>

      <!-- 状态流转记录 -->
      <a-card title="状态流转" style="margin-top: 12px;" :bordered="false">
        <a-empty v-if="!(detail.status_logs?.length)" description="暂无流转记录" />
        <a-timeline v-else>
          <a-timeline-item
            v-for="log in detail.status_logs"
            :key="log.id"
            :color="log.is_current ? 'blue' : 'gray'"
          >
            <div>
              <a-tag v-if="log.from_status" size="small" :color="getStatusColor(log.from_status)">
                {{ getStatusLabel(log.from_status) }}
              </a-tag>
              <span style="margin: 0 6px;">→</span>
              <a-tag size="small" :color="getStatusColor(log.to_status)">
                {{ getStatusLabel(log.to_status) }}
              </a-tag>
            </div>
            <div style="font-size: 13px; color: #666; margin-top: 4px;">
              {{ log.operator || '系统' }}
              <span v-if="log.remark" style="margin-left: 8px; color: #999;">{{ log.remark }}</span>
              <span style="margin-left: 8px; color: #bbb;">{{ log.created_at }}</span>
            </div>
          </a-timeline-item>
        </a-timeline>
      </a-card>

      <!-- 利润账本 -->
      <OrderProfitLedger v-if="detail.id" :order-id="detail.id" />
    </a-spin>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import CostLayerTag from '@/components/CostLayerTag.vue'
import OrderProfitLedger from '@/components/OrderProfitLedger.vue'
import { apiModules } from '@/api'

const router = useRouter()
const route = useRoute()

const orderId = route.params.id as string
const loading = ref(false)
const shippingState = ref(false)
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

const statusTagMap: Record<string, { color: string; label: string }> = {
  pending: { color: 'orange', label: '待付款' },
  paid: { color: 'blue', label: '待发货' },
  shipped: { color: 'blue', label: '已发货' },
  delivered: { color: 'green', label: '已签收' },
  completed: { color: 'green', label: '已完成' },
  cancelled: { color: 'red', label: '已取消' },
}

const statusTagColor = computed(() => statusTagMap[detail.value?.status]?.color || 'default')
const statusLabel = computed(() => statusTagMap[detail.value?.status]?.label || detail.value?.status || '-')

function getStatusColor(s: string) {
  return statusTagMap[s]?.color || 'default'
}

function getStatusLabel(s: string) {
  return statusTagMap[s]?.label || s
}

function money(value: any) {
  return Number(value || 0).toFixed(2)
}

const itemColumns = [
  { title: '商品名称', dataIndex: 'product_name', key: 'product_name', ellipsis: true },
  { title: 'SKU', dataIndex: 'sku_code', key: 'sku_code', width: 140 },
  { title: '规格', dataIndex: 'spec_desc', key: 'spec_desc', width: 140 },
  { title: '单价', dataIndex: 'unit_price', key: 'unit_price', width: 100 },
  { title: '数量', dataIndex: 'quantity', key: 'quantity', width: 80 },
  { title: '小计', dataIndex: 'subtotal', key: 'subtotal', width: 100 },
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
  Modal.confirm({
    title: '确认发货',
    content: '确定要将此订单标记为已发货吗？',
    okText: '确认发货',
    cancelText: '取消',
    onOk: async () => {
      shippingState.value = true
      try {
        await apiModules.orderApi.updateStatus(orderId, 'shipped')
        message.success('发货成功')
        fetchDetail()
      } catch (e: any) {
        message.error(e.message || '发货失败')
      } finally {
        shippingState.value = false
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
