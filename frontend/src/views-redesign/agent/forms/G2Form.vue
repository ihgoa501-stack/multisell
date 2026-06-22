<template>
  <a-form>
    <a-form-item label="产品名称"><a-input v-model:value="form.product_name" /></a-form-item>
    <a-form-item label="目的地国家">
      <a-select v-model:value="form.destination_country" :options="[{label:'美国',value:'US'},{label:'德国',value:'DE'},{label:'英国',value:'UK'},{label:'日本',value:'JP'}]" />
    </a-form-item>
    <a-form-item label="货品类型">
      <a-select v-model:value="form.cargo_type" :options="[{label:'电子产品',value:'electronics'},{label:'服装',value:'clothing'},{label:'食品',value:'food'},{label:'化妆品',value:'cosmetics'}]" />
    </a-form-item>
    <a-form-item label="申报价值 ¥"><a-input-number v-model:value="form.declared_value" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="重量 kg"><a-input-number v-model:value="form.weight_kg" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="月销量预估"><a-input-number v-model:value="form.monthly_sales_volume" :min="0" style="width:100%" /></a-form-item>
    <a-space>
      <a-button type="primary" @click="run('customs_clearance')" :loading="loading">清关查询</a-button>
      <a-button @click="run('warehouse_advice')" :loading="loading">仓储建议</a-button>
    </a-space>
  </a-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'
const emit = defineEmits(['decision'])
const loading = ref(false)
const form = reactive({ product_name: 'Bluetooth Speaker', destination_country: 'US', cargo_type: 'electronics', declared_value: 1500, weight_kg: 0.8, monthly_sales_volume: 200 })
async function run(dp: string) {
  loading.value = true
  try {
    const res: any = await agentApi.decide('G2', { decision_point: dp, context: { ...form } })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
