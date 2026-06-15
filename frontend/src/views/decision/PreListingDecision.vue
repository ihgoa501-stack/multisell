<template>
  <div>
    <h3 style="margin-bottom: 16px;">📊 上架前经营决策</h3>
    <n-card title="利润测算">
      <n-form :model="form" label-width="140">
        <n-grid :cols="24" :x-gap="12">
          <n-form-item-gi :span="8" label="SKU ID">
            <n-input-number v-model:value="form.sku_id" placeholder="输入SKU ID" :min="1" style="width: 100%;" />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="平台ID">
            <n-input-number v-model:value="form.platform_id" placeholder="选填，填写后自动匹配规则" :min="1" style="width: 100%;" clearable />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="目的国代码">
            <n-input v-model:value="form.destination_country" placeholder="如 RU" maxlength="10" />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="目标售价">
            <n-input-number v-model:value="form.target_sale_price" :min="0.01" :precision="2" style="width: 100%;">
              <template #suffix>元</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="平台费率">
            <n-input-number v-model:value="form.platform_fee_pct" :min="0" :max="100" :precision="1" style="width: 100%;">
              <template #suffix>%</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="支付费率">
            <n-input-number v-model:value="form.payment_fee_pct" :min="0" :max="100" :precision="1" style="width: 100%;">
              <template #suffix>%</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="其他费用">
            <n-input-number v-model:value="form.other_fee" :min="0" :precision="2" style="width: 100%;">
              <template #suffix>元</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="类目ID">
            <n-input-number v-model:value="form.category_id" placeholder="选填，用于匹配类目级规则" :min="1" style="width: 100%;" clearable />
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="最低利润率">
            <n-input-number v-model:value="form.minimum_margin_pct" :min="0" :max="100" :precision="1" style="width: 100%;">
              <template #suffix>%</template>
            </n-input-number>
          </n-form-item-gi>
          <n-form-item-gi :span="8" label="货品类型">
            <n-select
              v-model:value="form.cargo_type"
              :options="cargoTypeOptions"
              style="width: 100%;"
            />
          </n-form-item-gi>
        </n-grid>
        <div style="margin-top: 16px;">
          <n-button type="primary" @click="handleCalculate" :loading="loading">
            计算决策
          </n-button>
        </div>
      </n-form>
    </n-card>

    <n-card v-if="result" title="决策结果" style="margin-top: 16px;">
      <n-alert
        :type="result.recommendation === 'approve' ? 'success' : result.recommendation === 'reject' ? 'error' : 'warning'"
        :show-icon="false"
        style="margin-bottom: 16px;"
      >
        <template #header>
          {{ result.recommendation === 'approve' ? '✅ 建议上架' : result.recommendation === 'reject' ? '❌ 不建议上架' : '⚠️ 数据不足' }}
        </template>
        <div v-for="(reason, idx) in result.blocking_reasons" :key="'br-' + idx">
          ⛔ {{ reason }}
        </div>
        <div v-for="(warn, idx) in result.warnings" :key="'w-' + idx">
          ⚠️ {{ warn }}
        </div>
      </n-alert>

      <n-descriptions :column="3" bordered>
        <n-descriptions-item label="商品成本">{{ result.product_cost }} 元</n-descriptions-item>
        <n-descriptions-item label="运费">
          {{ result.shipping_fee }} 元
          <CostLayerTag :layer="result.shipping_cost_layer || 'estimated'" />
        </n-descriptions-item>
        <n-descriptions-item label="平台费">
          {{ result.platform_fee }} 元
          <CostLayerTag :layer="result.platform_fee_cost_layer || 'estimated'" />
        </n-descriptions-item>
        <n-descriptions-item label="支付费">{{ result.payment_fee }} 元</n-descriptions-item>
        <n-descriptions-item label="固定交易费">{{ result.fixed_fee }} 元</n-descriptions-item>
        <n-descriptions-item label="广告预留">{{ result.advertising_fee }} 元</n-descriptions-item>
        <n-descriptions-item label="其他费用">{{ result.other_fee }} 元</n-descriptions-item>
        <n-descriptions-item label="费用规则来源">
          {{ result.platform_fee_source === 'rule' ? '规则库' : '手动输入' }}
          <span v-if="result.applied_platform_fee_rule_id">
            #{{ result.applied_platform_fee_rule_id }} {{ result.platform_fee_rule_summary || '' }}
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="总成本">
          {{ (result.product_cost + result.shipping_fee + result.platform_fee + result.payment_fee + result.fixed_fee + result.advertising_fee + result.other_fee).toFixed(2) }} 元
        </n-descriptions-item>
        <n-descriptions-item label="利润金额">
          <span :style="{ color: result.profit_amount >= 0 ? 'green' : 'red', fontWeight: 'bold' }">
            {{ result.profit_amount }} 元
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="利润率">
          <span :style="{ color: result.profit_margin >= result.minimum_margin_pct ? 'green' : 'red', fontWeight: 'bold' }">
            {{ result.profit_margin }}%
          </span>
        </n-descriptions-item>
        <n-descriptions-item label="目标售价">{{ result.target_sale_price }} 元</n-descriptions-item>
      </n-descriptions>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useMessage } from 'naive-ui'
import CostLayerTag from '@/components/CostLayerTag.vue'
import {
  calculatePreListingDecision,
  type PreListingDecisionRequest,
  type PreListingDecisionResponse,
} from '@/api/modules/decision'

const message = useMessage()

const form = reactive<PreListingDecisionRequest>({
  sku_id: null as unknown as number,
  destination_country: '',
  target_sale_price: null as unknown as number,
  platform_id: null as unknown as number | null,
  category_id: null as unknown as number | null,
  platform_fee_pct: 10,
  payment_fee_pct: 3,
  other_fee: 0,
  minimum_margin_pct: 20,
  cargo_type: 'normal',
})

const cargoTypeOptions = [
  { label: '普通', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感', value: 'sensitive' },
]

const loading = ref(false)
const result = ref<PreListingDecisionResponse | null>(null)

async function handleCalculate() {
  if (!form.sku_id || !form.destination_country || !form.target_sale_price) {
    message.warning('请填写 SKU ID、目的国代码和目标售价')
    return
  }

  loading.value = true
  result.value = null
  try {
    const resp = await calculatePreListingDecision({ ...form })
    result.value = resp.data as unknown as PreListingDecisionResponse
  } catch (err: any) {
    message.error(err?.response?.data?.message || err?.message || '请求失败')
  } finally {
    loading.value = false
  }
}
</script>
