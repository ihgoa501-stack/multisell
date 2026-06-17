<template>
  <n-form>
    <n-form-item label="产品名称"><n-input v-model:value="form.product_name" /></n-form-item>
    <n-form-item label="目标市场">
      <n-select v-model:value="form.marketplace" :options="[{label:'美国',value:'US'},{label:'德国',value:'DE'},{label:'日本',value:'JP'}]" />
    </n-form-item>
    <n-form-item label="产品卖点（每行一个）">
      <n-input v-model:value="featuresText" type="textarea" :rows="4" placeholder="Suction base&#10;Food-grade silicone&#10;BPA free" />
    </n-form-item>
    <n-form-item label="关键词 (JSON)">
      <n-input v-model:value="keywordsText" type="textarea" :rows="3" />
    </n-form-item>
    <n-button type="primary" @click="run" :loading="loading">优化 Listing</n-button>
  </n-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'
const emit = defineEmits(['decision'])
const message = useMessage()
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
