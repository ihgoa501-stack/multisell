<template>
  <n-form>
    <n-form-item label="Campaign ID"><n-input v-model:value="form.campaign_id" /></n-form-item>
    <n-form-item label="广告花费"><n-input-number v-model:value="form.spend" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="销售额"><n-input-number v-model:value="form.sales" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="点击量"><n-input-number v-model:value="form.clicks" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="展示量"><n-input-number v-model:value="form.impressions" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="转化数"><n-input-number v-model:value="form.conversions" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="毛利率 %"><n-input-number v-model:value="form.gross_margin" :min="0" style="width:100%" /></n-form-item>
    <n-form-item label="目标 ACoS %"><n-input-number v-model:value="form.target_acos" :min="0" style="width:100%" /></n-form-item>
    <n-button type="primary" @click="run" :loading="loading">分析 ACoS</n-button>
  </n-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'
const emit = defineEmits(['decision'])
const message = useMessage()
const loading = ref(false)
const form = reactive({ campaign_id: 'CAM-001', spend: 500, sales: 2000, clicks: 200, impressions: 8000, conversions: 15, gross_margin: 35, target_acos: 30 })
async function run() {
  loading.value = true
  try {
    const res: any = await agentApi.decide('A3', { decision_point: 'acos_analysis', context: { ...form } })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
