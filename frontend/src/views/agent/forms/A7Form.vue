<template>
  <n-form>
    <n-form-item label="产品名称"><n-input v-model:value="form.product_name" /></n-form-item>
    <n-form-item label="品类">
      <n-select v-model:value="form.category" :options="[{label:'电子产品',value:'electronics'},{label:'玩具',value:'toys'},{label:'婴儿用品',value:'baby'},{label:'化妆品',value:'cosmetics'},{label:'食品',value:'food'}]" />
    </n-form-item>
    <n-form-item label="目标国家">
      <n-select v-model:value="form.target_country" :options="[{label:'美国',value:'US'},{label:'欧盟',value:'EU'},{label:'英国',value:'UK'},{label:'日本',value:'JP'}]" />
    </n-form-item>
    <n-form-item label="平台"><n-input v-model:value="form.target_platform" placeholder="amazon" /></n-form-item>
    <n-space>
      <n-button type="primary" @click="run" :loading="loading">合规检查</n-button>
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
const form = reactive({ product_name: 'Wireless Earbuds', category: 'electronics', target_country: 'US', target_platform: 'amazon' })
async function run() {
  loading.value = true
  try {
    const res: any = await agentApi.decide('A7', { decision_point: 'compliance_check', context: { ...form } })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
