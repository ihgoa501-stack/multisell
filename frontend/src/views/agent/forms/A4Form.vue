<template>
  <n-form>
    <n-form-item label="客户消息"><n-input v-model:value="form.message" type="textarea" :rows="4" /></n-form-item>
    <n-form-item label="语言">
      <n-select v-model:value="form.language" :options="[{label:'英语',value:'en'},{label:'德语',value:'de'},{label:'法语',value:'fr'},{label:'日语',value:'jp'}]" />
    </n-form-item>
    <n-form-item label="预计送达（天）"><n-input v-model:value="form.eta" placeholder="5-7" /></n-form-item>
    <n-space>
      <n-button type="primary" @click="run('auto_reply')" :loading="loading">自动回复</n-button>
      <n-button @click="run('intent_classify')" :loading="loading">意图分类</n-button>
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
const form = reactive({ message: 'Where is my order?', language: 'en', eta: '5-7' })
async function run(dp: string) {
  loading.value = true
  try {
    const ctx: any = { message: form.message, language: form.language }
    if (form.eta) ctx.order_context = { estimated_delivery_days: form.eta }
    const res: any = await agentApi.decide('A4', { decision_point: dp, context: ctx })
    emit('decision', res?.data)
  } catch (e: any) { message.error(e?.response?.data?.message || '失败') }
  loading.value = false
}
</script>
