<template>
  <a-form>
    <a-form-item label="产品名称"><a-input v-model:value="form.product_name" /></a-form-item>
    <a-form-item label="目标市场">
      <a-select v-model:value="form.marketplace" :options="[{label:'美国',value:'US'},{label:'德国',value:'DE'},{label:'日本',value:'JP'}]" />
    </a-form-item>
    <a-form-item label="产品卖点（每行一个）">
      <a-textarea v-model:value="featuresText" :rows="4" placeholder="Suction base&#10;Food-grade silicone&#10;BPA free" />
    </a-form-item>
    <a-form-item label="关键词 (JSON)">
      <a-textarea v-model:value="keywordsText" :rows="3" />
    </a-form-item>
    <a-button type="primary" @click="run" :loading="loading">优化 Listing</a-button>
  </a-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'
const emit = defineEmits(['decision'])
const loading = ref(false)
const featuresText = ref('Suction base\nFood-grade silicone\nBPA free')
const keywordsText = ref('[{"word":"baby plate","volume":15000},{"word":"silicone plate","volume":12000}]')
const form = reactive({ product_name: 'Silicone Baby Plate', marketplace: 'US' })
async function run() {
  loading.value = true
  try {
    const features = featuresText.value.split('\n').filter(Boolean)
    let keywords = []
    try { keywords = JSON.parse(keywordsText.value) } catch { keywords = [] }
    const res: any = await agentApi.decide('A2', { decision_point: 'listing_optimize', context: { ...form, features, keywords } })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
