<template>
  <div>
    <n-button quaternary @click="router.back()" style="margin-bottom: 8px;">← 返回</n-button>
    <n-page-header :subtitle="agent?.description || ''">
      <template #title>
        <n-space align="center">
          <n-tag :bordered="false" :color="{ color: tagColor(agentId), textColor: '#fff' }">{{ agentId }}</n-tag>
          {{ agent?.name || agentId }}
        </n-space>
      </template>
    </n-page-header>

    <n-grid :cols="3" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="决策模拟" :bordered="false">
          <n-form>
            <n-form-item label="决策点">
              <n-select v-model:value="simulatePoint" :options="decisionPointOptions" />
            </n-form-item>
            <n-form-item label="上下文 (JSON)">
              <n-input v-model:value="simulateContext" type="textarea" :rows="6" placeholder='{"sku_code": "SKU001", "quantity": 50, "daily_sales": 10}' />
            </n-form-item>
            <n-space>
              <n-button type="primary" @click="runDecision(false)" :loading="loading">执行决策</n-button>
              <n-button @click="runDecision(true)" :loading="loading">模拟 (Dry Run)</n-button>
            </n-space>
          </n-form>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="决策结果" :bordered="false">
          <n-empty v-if="!result" description="尚未执行决策" />
          <n-space v-else vertical>
            <n-tag :type="result.confidence >= 0.9 ? 'success' : result.confidence >= 0.7 ? 'warning' : 'error'">
              置信度: {{ (result.confidence * 100).toFixed(0) }}%
            </n-tag>
            <n-tag>阶段: {{ result.stage }}</n-tag>
            <n-tag v-if="result.rules_applied?.length">已应用 {{ result.rules_applied.length }} 条规则</n-tag>
            <n-divider />
            <n-code :code="JSON.stringify(result.decision, null, 2)" language="json" />
          </n-space>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="Honcho 配置" :bordered="false">
          <n-form>
            <n-form-item label="风险容忍度">
              <n-select v-model:value="profile.risk_tolerance" :options="riskOptions" @update:value="saveProfile" />
            </n-form-item>
            <n-form-item label="沟通风格">
              <n-select v-model:value="profile.communication_style" :options="styleOptions" @update:value="saveProfile" />
            </n-form-item>
          </n-form>
          <n-divider />
          <n-button size="small" @click="router.push('/agents/rules')">管理个人规则</n-button>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card title="决策日志" style="margin-top: 12px;">
      <n-data-table :columns="logColumns" :data="logs" :loading="loadingLogs" :pagination="logPagination" @update:page="onLogPageChange" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted, computed } from 'vue'
import { useRouter, useRoute, useMessage, NTag, NSpace, NCode, NEmpty, NDivider } from 'naive-ui'
import { agentApi } from '@/api/modules/agent'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const agentId = route.params.agentId as string
const agent = ref<any>(null)
const loading = ref(false)
const loadingLogs = ref(false)
const logs = ref<any[]>([])
const logTotal = ref(0)
const result = ref<any>(null)
const simulatePoint = ref('')
const simulateContext = ref('{}')

const profile = reactive({ risk_tolerance: 'moderate', communication_style: 'balanced' })

const riskOptions = [
  { label: '保守', value: 'conservative' },
  { label: '适中', value: 'moderate' },
  { label: '激进', value: 'aggressive' },
]

const styleOptions = [
  { label: '简洁', value: 'concise' },
  { label: '均衡', value: 'balanced' },
  { label: '详细', value: 'detailed' },
]

const decisionPointOptions = computed(() => {
  return (agent.value?.decision_points || []).map((dp: string) => ({ label: dp, value: dp }))
})

const logPagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
  onChange: (page: number) => { logPagination.page = page; fetchLogs() },
})

function tagColor(id: string) {
  const colors: Record<string, string> = {
    A3: '#e67e22', A4: '#2ecc71', A5: '#e74c3c',
    A6: '#9b59b6', A7: '#3498db',
    G1: '#1abc9c', G2: '#f39c12', G3: '#e74c3c',
  }
  return colors[id] || '#95a5a6'
}

const logColumns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: '决策点', key: 'decision_point', width: 120 },
  { title: '置信度', key: 'confidence', width: 80, render: (row: any) => row.confidence ? (row.confidence * 100).toFixed(0) + '%' : '-' },
  { title: '阶段', key: 'evolution_stage', width: 100 },
  { title: '用户操作', key: 'user_action', width: 100, render: (row: any) => {
    const colors: Record<string, string> = { accepted: 'success', modified: 'warning', rejected: 'error', ignored: 'default' }
    return h(NTag, { type: colors[row.user_action] || 'default', size: 'small' }, { default: () => row.user_action })
  }},
  { title: '响应(ms)', key: 'response_time_ms', width: 80 },
]

async function fetchAgent() {
  try {
    const res: any = await agentApi.get(agentId)
    agent.value = res?.data
    if (agent.value?.decision_points?.length) simulatePoint.value = agent.value.decision_points[0]
  } catch { /* handle */ }
}

async function fetchLogs() {
  loadingLogs.value = true
  try {
    const res: any = await agentApi.getDecisions({
      agent_id: agentId, page: logPagination.page, page_size: logPagination.pageSize,
    })
    logs.value = res?.records || []
    logTotal.value = res?.total || 0
    logPagination.itemCount = logTotal.value
  } catch { /* no logs */ }
  loadingLogs.value = false
}

async function runDecision(dryRun: boolean) {
  loading.value = true
  try {
    let context: any
    try { context = JSON.parse(simulateContext.value) } catch { context = {} }
    const res: any = await agentApi.decide(agentId, { decision_point: simulatePoint.value, context, dry_run: dryRun })
    result.value = res?.data
  } catch (e: any) {
    message.error(e?.response?.data?.message || '决策执行失败')
  }
  loading.value = false
}

async function saveProfile() {
  try {
    await agentApi.updateProfile({ risk_tolerance: profile.risk_tolerance, communication_style: profile.communication_style })
    message.success('配置已保存')
  } catch { /* ignore */ }
}

function onLogPageChange(page: number) {
  logPagination.page = page
}

onMounted(async () => {
  await fetchAgent()
  await fetchLogs()
  try {
    const res: any = await agentApi.getProfile()
    if (res?.data) {
      profile.risk_tolerance = res.data.risk_tolerance || 'moderate'
      profile.communication_style = res.data.communication_style || 'balanced'
    }
  } catch { /* no profile yet */ }
})
</script>
