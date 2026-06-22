<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h2 style="margin: 0; font-size: 20px; font-weight: 600;">LLM 配置</h2>
      <span style="color: rgba(0,0,0,0.45); font-size: 14px;">管理 LLM API Key、提供商、各 Agent 模型</span>
    </div>

    <a-card style="margin-top: 12px;" :bordered="false">
      <a-form v-if="definitions" layout="vertical">
        <a-form-item v-for="(def, key) in definitions" :key="key" :label="def.label">
          <template v-if="def.type === 'secret'">
            <a-input-password v-model:value="form[key]" :placeholder="def.description" />
          </template>
          <template v-else-if="def.type === 'select'">
            <a-select v-model:value="form[key]" :options="def.options.map((o: string) => ({label: o, value: o}))" />
          </template>
          <template v-else-if="def.type === 'json'">
            <a-textarea v-model:value="form[key]" :rows="3" :placeholder="def.description" />
          </template>
          <template v-else>
            <a-input v-model:value="form[key]" :placeholder="def.description" />
          </template>
          <template #extra>{{ def.description }}</template>
        </a-form-item>

        <a-space style="margin-top: 16px;">
          <a-button type="primary" @click="save" :loading="saving">保存配置</a-button>
          <a-button @click="testConnection">测试连接</a-button>
        </a-space>
        <a-alert v-if="testResult" :type="testResult.ok ? 'success' : 'error'" :message="testResult.message" show-icon style="margin-top: 12px;" />
      </a-form>
      <a-spin v-else tip="加载中..." />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import http from '@/api/http'

const saving = ref(false)
const definitions = ref<any>(null)
const testResult = ref<any>(null)
const form = reactive<Record<string, any>>({})

async function fetchConfig() {
  try {
    const res: any = await http.get('/settings/llm')
    const data = res?.data || {}
    definitions.value = data.definitions || {}
    const configs = data.configs || {}
    // 初始化表单
    for (const [key, def] of Object.entries(data.definitions || {})) {
      const val = configs[key]
      const defaultVal = (def as any).default
      if ((def as any).type === 'json') {
        form[key] = val ? JSON.stringify(val, null, 2) : JSON.stringify(defaultVal, null, 2)
      } else {
        form[key] = val !== undefined ? val : (defaultVal ?? '')
      }
    }
  } catch (e: any) {
    message.error('获取配置失败')
  }
}

async function save() {
  saving.value = true
  try {
    const payload: Record<string, any> = {}
    for (const key of Object.keys(definitions.value || {})) {
      const def = definitions.value[key]
      if (def.type === 'json') {
        try { payload[key] = JSON.parse(form[key] || '{}') } catch { payload[key] = {} }
      } else {
        payload[key] = form[key]
      }
    }
    await http.put('/settings/llm', payload)
    message.success('配置已保存')
  } catch (e: any) {
    message.error(e?.response?.data?.message || '保存失败')
  }
  saving.value = false
}

async function testConnection() {
  testResult.value = null
  try {
    // 调一个 A5 决策来验证 LLM 是否工作
    const dr = (await http.post('/agents/A5/decide', {
      decision_point: 'stock_alert',
      context: { sku_code: 'DEMO-RED', sellable_stock: 5, sales_7d: 21, lead_time_days: 20, safety_stock_days: 14 },
      dry_run: true,
    } as any)).data
    const decision = dr?.decision || {}
    if (decision.ai_explanation && !decision.ai_explanation.startsWith('\u26a0\ufe0f') && !decision.ai_explanation.startsWith('\u2705') && !decision.ai_explanation.startsWith('\ud83d\udd34')) {
      testResult.value = { ok: true, message: `LLM 连接成功！返回解释: ${decision.ai_explanation.slice(0, 80)}...` }
    } else {
      testResult.value = { ok: false, message: 'LLM 未连接（使用了模板回退），请检查 API Key 是否正确' }
    }
  } catch (e: any) {
    testResult.value = { ok: false, message: `连接失败: ${e?.response?.data?.message || e.message}` }
  }
}

onMounted(() => { fetchConfig() })
</script>
