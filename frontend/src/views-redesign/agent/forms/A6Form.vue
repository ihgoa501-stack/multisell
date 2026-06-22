<template>
  <a-form>
    <a-form-item label="SKU">
      <a-input v-model:value="form.sku_code" placeholder="SKU001" />
    </a-form-item>
    <template v-if="!autoMode">
      <a-form-item label="售价">
        <a-input-number v-model:value="form.selling_price" :min="0" style="width:100%" />
      </a-form-item>
      <a-form-item label="采购成本">
        <a-input-number v-model:value="form.cost_price" :min="0" style="width:100%" />
      </a-form-item>
    </template>
    <a-form-item label="平台佣金率 %">
      <a-input-number v-model:value="form.platform_fee_rate" :min="0" :max="100" style="width:100%" />
    </a-form-item>
    <a-form-item label="物流费用">
      <a-input-number v-model:value="form.shipping_fee" :min="0" style="width:100%" />
    </a-form-item>
    <a-form-item label="广告成本（单件摊）">
      <a-input-number v-model:value="form.ad_cost_per_unit" :min="0" style="width:100%" />
    </a-form-item>
    <a-form-item label="退款率 %">
      <a-input-number v-model:value="form.refund_rate" :min="0" style="width:100%" />
    </a-form-item>
    <a-form-item label="最低毛利率阈值 %">
      <a-input-number v-model:value="form.min_margin_threshold" :min="0" style="width:100%" />
    </a-form-item>
    <a-space>
      <a-button type="primary" @click="run" :loading="loading">检查利润</a-button>
      <a-checkbox v-model:checked="autoMode">自动读取数据库</a-checkbox>
    </a-space>
  </a-form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'

const emit = defineEmits(['decision'])
const loading = ref(false)
const autoMode = ref(false)

const form = reactive({
  sku_code: '',
  selling_price: 100,
  cost_price: 50,
  platform_fee_rate: 10,
  shipping_fee: 12,
  fixed_fee: 2,
  ad_cost_per_unit: 5,
  refund_rate: 3,
  min_margin_threshold: 15,
})

async function run() {
  loading.value = true
  try {
    const ctx: any = { ...form }
    const res: any = await agentApi.decide('A6', { decision_point: 'profit_check', context: ctx })
    emit('decision', res?.data)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '执行失败')
  }
  loading.value = false
}
</script>
