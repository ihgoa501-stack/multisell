<template>
  <n-form>
    <n-form-item label="SKU">
      <n-input v-model:value="form.sku_code" placeholder="SKU001" />
    </n-form-item>
    <template v-if="!autoMode">
      <n-form-item label="售价">
        <n-input-number v-model:value="form.selling_price" :min="0" style="width:100%" />
      </n-form-item>
      <n-form-item label="采购成本">
        <n-input-number v-model:value="form.cost_price" :min="0" style="width:100%" />
      </n-form-item>
    </template>
    <n-form-item label="平台佣金率 %">
      <n-input-number v-model:value="form.platform_fee_rate" :min="0" :max="100" style="width:100%" />
    </n-form-item>
    <n-form-item label="物流费用">
      <n-input-number v-model:value="form.shipping_fee" :min="0" style="width:100%" />
    </n-form-item>
    <n-form-item label="广告成本（单件摊）">
      <n-input-number v-model:value="form.ad_cost_per_unit" :min="0" style="width:100%" />
    </n-form-item>
    <n-form-item label="退款率 %">
      <n-input-number v-model:value="form.refund_rate" :min="0" style="width:100%" />
    </n-form-item>
    <n-form-item label="最低毛利率阈值 %">
      <n-input-number v-model:value="form.min_margin_threshold" :min="0" style="width:100%" />
    </n-form-item>
    <n-space>
      <n-button type="primary" @click="run" :loading="loading">检查利润</n-button>
      <n-checkbox v-model:value="autoMode">自动读取数据库</n-checkbox>
    </n-space>
  </n-form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'

const emit = defineEmits(['decision'])
const message = useMessage()
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
