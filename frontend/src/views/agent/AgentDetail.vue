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
              <n-input v-model:value="simulateContext" type="textarea" :rows="6" placeholder='{"sku_code": "SKU001", "sellable_stock": 50, "sales_7d": 70}' />
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
            <n-tag v-if="result.decision_id">决策ID: {{ result.decision_id }}</n-tag>
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
      <n-data-table
        :columns="logColumns"
        :data="logs"
        :loading="loadingLogs"
        :pagination="logPagination"
        :row-key="(row: any) => row.id"
        @update:page="onLogPageChange"
      />
    </n-card>

    <!-- 决策详情抽屉 -->
    <n-drawer v-model:show="showDetail" :width="600" placement="right">
      <n-drawer-content :title="'决策 #' + (selectedLog?.id || '')" closable>
        <n-space vertical v-if="selectedLog">
          <n-description label="决策点" :content="selectedLog.decision_point" />
          <n-card title="上下文 (Context)" size="small">
            <n-code :code="JSON.stringify(selectedLog.context_json, null, 2)" language="json" />
          </n-card>
          <n-card title="Agent 输出" size="small">
            <n-code :code="JSON.stringify(selectedLog.agent_output, null, 2)" language="json" />
          </n-card>
          <n-card title="最终决策" size="small">
            <n-code :code="JSON.stringify(selectedLog.final_decision, null, 2)" language="json" />
          </n-card>
          <n-card v-if="selectedLog.rules_applied?.length" title="已应用规则" size="small">
            <n-tag v-for="rid in selectedLog.rules_applied" :key="rid" style="margin: 2px;">规则 #{{ rid }}</n-tag>
          </n-card>
          <n-card title="用户反馈" size="small">
            <n-empty v-if="selectedLog.user_action === 'ignored' && !selectedLog.user_feedback" description="暂无反馈" />
            <n-space v-else vertical>
              <n-tag :type="feedbackTagType(selectedLog.user_action)">{{ selectedLog.user_action }}</n-tag>
              <div v-if="selectedLog.user_feedback">反馈: {{ selectedLog.user_feedback }}</div>
              <div v-if="selectedLog.user_overrides">覆盖: {{ JSON.stringify(selectedLog.user_overrides) }}</div>
            </n-space>
            <n-divider />
            <n-form>
              <n-form-item label="操作">
                <n-select v-model:value="feedbackAction" :options="feedbackOptions" />
              </n-form-item>
              <n-form-item label="修改内容 (JSON)">
                <n-input v-model:value="feedbackOverrides" type="textarea" :rows="4" placeholder='{"field": "value"}' />
              </n-form-item>
              <n-form-item label="反馈文字">
                <n-input v-model:value="feedbackText" type="textarea" :rows="2" />
              </n-form-item>
              <n-button type="primary" @click="submitFeedback" :loading="submittingFeedback">提交反馈</n-button>
            </n-form>
          </n-card>
        </n-space>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute, useMessage, NTag, NSpace, NCode, NEmpty, NDivider, NButton } from 'naive-ui'
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

// 决策详情抽屉
const showDetail = ref(false)
const selectedLog = ref<any>(null)
const feedbackAction = ref('accepted')
const feedbackOverrides = ref('')
const feedbackText = ref('')
const submittingFeedback = ref(false)

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

const feedbackOptions = [
  { label: '采纳', value: 'accepted' },
  { label: '修改', value: 'modified' },
  { label: '拒绝', value: 'rejected' },
  { label: '忽略', value: 'ignored' },
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

function feedbackTagType(action: string) {
  const map: Record<string, string> = { accepted: 'success', modified: 'warning', rejected: 'error', ignored: 'default' }
  return map[action] || 'default'
}

const logColumns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: '决策点', key: 'decision_point', width: 120 },
  { title: '置信度', key: 'confidence', width: 80, render: (row: any) => row.confidence ? (row.confidence * 100).toFixed(0) + '%' : '-' },
  { title: '阶段', key: 'evolution_stage', width: 100 },
  { title: '用户操作', key: 'user_action', width: 100, render: (row: any) => {
    return h(NTag, { type: feedbackTagType(row.user_action), size: 'small' }, { default: () => row.user_action })
  }},
  { title: '规则', key: 'rules_applied', width: 70, render: (row: any) => {
    if (!row.rules_applied?.length) return '-'
    return h(NTag, { size: 'small', type: 'info' }, { default: () => `${row.rules_applied.length}条` })
  }},
  { title: '响应(ms)', key: 'response_time_ms', width: 80 },
  { title: '操作', key: 'actions', width: 80, render: (row: any) => {
    return h(NButton, {
      size: 'tiny', quaternary: true,
      onClick: () => openDetail(row),
    }, { default: () => '详情' })
  }},
]

function openDetail(row: any) {
  selectedLog.value = { ...row }
  feedbackAction.value = row.user_action || 'ignored'
  feedbackOverrides.value = row.user_overrides ? JSON.stringify(row.user_overrides, null, 2) : ''
  feedbackText.value = row.user_feedback || ''
  showDetail.value = true
}

async function submitFeedback() {
  if (!selectedLog.value?.id) return
  submittingFeedback.value = true
  try {
    let overrides: any = null
    try {
      if (feedbackOverrides.value.trim()) overrides = JSON.parse(feedbackOverrides.value)
    } catch { /* invalid json, skip */ }

    await agentApi.submitFeedback(selectedLog.value.id, {
      user_action: feedbackAction.value,
      user_overrides: overrides,
      user_feedback: feedbackText.value || undefined,
    })
    message.success('反馈已提交')
    selectedLog.value.user_action = feedbackAction.value
    selectedLog.value.user_overrides = overrides
    selectedLog.value.user_feedback = feedbackText.value
    await fetchLogs()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '提交失败')
  }
  submittingFeedback.value = false
}

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
    if (res?.data?.decision?.status === 'insufficient_data') {
      message.warning(res.data.decision.message || '数据不足')
    } else if (!dryRun && res?.data?.decision_id) {
      message.success('决策已记录')
      await fetchLogs()
    }
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
