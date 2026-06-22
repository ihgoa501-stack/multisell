<template>
  <a-form>
    <a-form-item label="品类"><a-input v-model:value="form.category" placeholder="Baby Feeding" /></a-form-item>
    <a-form-item label="目标市场">
      <a-select v-model:value="form.marketplace" :options="[{label:'美国',value:'US'},{label:'欧洲',value:'EU'},{label:'日本',value:'JP'}]" />
    </a-form-item>
    <a-form-item label="候选产品 (JSON)">
      <a-textarea v-model:value="candidatesText" :rows="6" />
    </a-form-item>
    <a-button type="primary" @click="run" :loading="loading">扫描选品</a-button>
  </a-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'
const emit = defineEmits(['decision'])
const loading = ref(false)
const candidatesText = ref('[{"name":"Silicone Plate","price":19.99,"cost":4.5,"search_volume":15000,"trend_growth":120,"review_count":200}]')
const form = reactive({ category: 'Baby Feeding', marketplace: 'US' })
async function run() {
  loading.value = true
  try {
    let candidates = []
    try { candidates = JSON.parse(candidatesText.value) } catch { candidates = [] }
    const res: any = await agentApi.decide('A1', { decision_point: 'product_scout', context: { ...form, candidates } })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
