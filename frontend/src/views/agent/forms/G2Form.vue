<template>
  <n-form>
    <n-form-item label="产品名称"><n-input v-model:value="form.product_name" /></n-form-item>
    <n-form-item label="目的地国家">
      <n-select v-model:value="form.destination_country" :options="[{label:'美国',value:'US'},{label:'德国',value:'DE'},{label:'英国',value:'UK'},{label:'日本',value:'JP'}]" />
    </n-form-item>
    <n-form-item label="货品类型">
      <n-select v-model:value="form.cargo_type" :options="[{label:'电子产品',value:'electronics'},{label:'服装',value:'clothing'},{label:'食品',value:'food'},{label:'化妆品',value:'cosmetics'}]" />
    </n-form-item>
    <n-form-item label="申报价值 ¥"><n-input-number v-model:value="form.declared_value" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="重量 kg"><n-input-number v-model:value="form.weight_kg" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="月销量预估"><n-input-number v-model:value="form.monthly_sales_volume" :min="0" style="width:100%" /></n-form-item>
    <n-space>
      <n-button type="primary" @click="run('customs_clearance')" :loading="loading">清关查询</n-button>
      <n-button @click="run('warehouse_advice')" :loading="loading">仓储建议</n-button>
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
