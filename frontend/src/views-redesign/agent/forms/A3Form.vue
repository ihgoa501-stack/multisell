<template>
  <a-form>
    <a-form-item label="Campaign ID"><a-input v-model:value="form.campaign_id" /></a-form-item>
    <a-form-item label="广告花费"><a-input-number v-model:value="form.spend" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="销售额"><a-input-number v-model:value="form.sales" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="点击量"><a-input-number v-model:value="form.clicks" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="展示量"><a-input-number v-model:value="form.impressions" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="转化数"><a-input-number v-model:value="form.conversions" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="毛利率 %"><a-input-number v-model:value="form.gross_margin" :min="0" style="width:100%" /></a-form-item>
    <a-form-item label="目标 ACoS %"><a-input-number v-model:value="form.target_acos" :min="0" style="width:100%" /></a-form-item>
    <a-button type="primary" @click="run" :loading="loading">分析 ACoS</a-button>
  </a-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'
const emit = defineEmits(['decision'])
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
