<template>
  <n-form>
    <n-form-item label="品类"><n-input v-model:value="form.category" placeholder="Baby Feeding" /></n-form-item>
    <n-form-item label="目标市场">
      <n-select v-model:value="form.marketplace" :options="[{label:'美国',value:'US'},{label:'欧洲',value:'EU'},{label:'日本',value:'JP'}]" />
    </n-form-item>
    <n-form-item label="候选产品 (JSON)">
      <n-input v-model:value="candidatesText" type="textarea" :rows="6" />
    </n-form-item>
    <n-button type="primary" @click="run" :loading="loading">扫描选品</n-button>
  </n-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { useMessage } from 'naive-ui'
const emit = defineEmits(['decision'])
const message = useMessage()
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
