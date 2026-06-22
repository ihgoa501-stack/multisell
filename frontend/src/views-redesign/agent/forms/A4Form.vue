<template>
  <a-form>
    <a-form-item label="客户消息"><a-textarea v-model:value="form.message" :rows="4" /></a-form-item>
    <a-form-item label="语言">
      <a-select v-model:value="form.language" :options="[{label:'英语',value:'en'},{label:'德语',value:'de'},{label:'法语',value:'fr'},{label:'日语',value:'jp'}]" />
    </a-form-item>
    <a-form-item label="预计送达（天）"><a-input v-model:value="form.eta" placeholder="5-7" /></a-form-item>
    <a-space>
      <a-button type="primary" @click="run('auto_reply')" :loading="loading">自动回复</a-button>
      <a-button @click="run('intent_classify')" :loading="loading">意图分类</a-button>
    </a-space>
  </a-form>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { agentApi } from '@/api/modules/agent'
import { message } from 'ant-design-vue'
const emit = defineEmits(['decision'])
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
