<template>
  <div>
    <a-button type="link" @click="router.back()" style="margin-bottom: 8px; padding-left: 0;">← 返回</a-button>
    <div style="margin-bottom: 16px;">
      <a-space align="center">
        <a-tag :color="tagColor(agentId)">{{ agentId }}</a-tag>
        <h2 style="margin: 0; font-size: 20px; font-weight: 600;">{{ agent?.name || agentId }}</h2>
      </a-space>
      <div style="color: rgba(0,0,0,0.45); font-size: 14px;">{{ agent?.description || '' }}</div>
    </div>

    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="8">
        <a-card title="决策模拟" :bordered="false">
          <!-- Agent 专属表单 -->
          <div v-if="agentFormComponent">
            <component :is="agentFormComponent" :result="result" @decision="onDecision" />
            <a-divider />
            <a-space justify="end" style="width: 100%; display: flex;">
              <a-button size="small" type="link" @click="showJsonInput = !showJsonInput">
                {{ showJsonInput ? '隐藏 JSON' : '高级：JSON 模式' }}
              </a-button>
            </a-space>
          </div>
          <!-- JSON 模式（回退） -->
          <div v-if="!agentFormComponent || showJsonInput">
            <a-form layout="vertical">
              <a-form-item label="决策点">
                <a-select v-model:value="simulatePoint" :options="decisionPointOptions" />
              </a-form-item>
              <a-form-item label="上下文 (JSON)">
                <a-textarea v-model:value="simulateContext" :rows="6" placeholder='{"sku_code": "SKU001", "sellable_stock": 50, "sales_7d": 70}' />
              </a-form-item>
              <a-space>
                <a-button type="primary" @click="runDecision(false)" :loading="loading">执行决策</a-button>
                <a-button @click="runDecision(true)" :loading="loading">模拟 (Dry Run)</a-button>
              </a-space>
            </a-form>
          </div>
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card title="决策结果" :bordered="false">
          <a-empty v-if="!result" description="尚未执行决策" />
          <a-space v-else direction="vertical" style="width: 100%;">
            <a-space wrap>
              <a-tag :color="result.confidence >= 0.9 ? 'success' : result.confidence >= 0.7 ? 'warning' : 'error'">
                置信度: {{ (result.confidence * 100).toFixed(0) }}%
              </a-tag>
              <a-tag>阶段: {{ result.stage }}</a-tag>
              <a-tag v-if="result.rules_applied?.length">已应用 {{ result.rules_applied.length }} 条规则</a-tag>
              <a-tag v-if="result.decision_id">决策ID: {{ result.decision_id }}</a-tag>
            </a-space>
            <!-- AI 解释 -->
            <a-alert v-if="result.decision?.ai_explanation" type="info" :message="'AI 解释'" :description="result.decision.ai_explanation" show-icon style="margin-top: 8px;" />
            <a-divider />
            <pre style="background: #f5f5f5; padding: 12px; border-radius: 6px; overflow: auto; font-size: 12px;"><code>{{ JSON.stringify(result.decision, null, 2) }}</code></pre>
          </a-space>
        </a-card>
      </a-col>
      <a-col :span="8">
        <a-card title="Honcho 配置" :bordered="false">
          <a-form layout="vertical">
            <a-form-item label="风险容忍度">
              <a-select v-model:value="profile.risk_tolerance" :options="riskOptions" @change="saveProfile" />
            </a-form-item>
            <a-form-item label="沟通风格">
              <a-select v-model:value="profile.communication_style" :options="styleOptions" @change="saveProfile" />
            </a-form-item>
          </a-form>
          <a-divider />
          <a-button size="small" @click="router.push('/agents/rules')">管理个人规则</a-button>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="决策日志" style="margin-top: 12px;">
      <a-table
        :columns="logColumns"
        :data-source="logs"
        :loading="loadingLogs"
        :pagination="{ current: logPagination.page, pageSize: logPagination.pageSize, total: logTotal }"
        :row-key="(row: any) => row.id"
        @change="onLogTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'confidence'">
            {{ record.confidence ? (record.confidence * 100).toFixed(0) + '%' : '-' }}
          </template>
          <template v-else-if="column.key === 'user_action'">
            <a-tag :color="feedbackTagColor(record.user_action)">{{ record.user_action }}</a-tag>
          </template>
          <template v-else-if="column.key === 'rules_applied'">
            <template v-if="!record.rules_applied?.length">-</template>
            <a-tag v-else color="processing">{{ record.rules_applied.length }}条</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" type="link" @click="openDetail(record)">详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 决策详情抽屉 -->
    <a-drawer v-model:open="showDetail" :width="600" :title="'决策 #' + (selectedLog?.id || '')" placement="right">
      <a-space direction="vertical" style="width: 100%;" v-if="selectedLog">
        <div><strong>决策点：</strong>{{ selectedLog.decision_point }}</div>
        <a-card title="上下文 (Context)" size="small">
          <pre style="background: #f5f5f5; padding: 8px; border-radius: 4px; overflow: auto; font-size: 12px; margin: 0;"><code>{{ JSON.stringify(selectedLog.context_json, null, 2) }}</code></pre>
        </a-card>
        <a-card title="Agent 输出" size="small">
          <pre style="background: #f5f5f5; padding: 8px; border-radius: 4px; overflow: auto; font-size: 12px; margin: 0;"><code>{{ JSON.stringify(selectedLog.agent_output, null, 2) }}</code></pre>
        </a-card>
        <a-card title="最终决策" size="small">
          <pre style="background: #f5f5f5; padding: 8px; border-radius: 4px; overflow: auto; font-size: 12px; margin: 0;"><code>{{ JSON.stringify(selectedLog.final_decision, null, 2) }}</code></pre>
        </a-card>
        <a-card v-if="selectedLog.rules_applied?.length" title="已应用规则" size="small">
          <a-tag v-for="rid in selectedLog.rules_applied" :key="rid" style="margin: 2px;">规则 #{{ rid }}</a-tag>
        </a-card>
        <a-card title="用户反馈" size="small">
          <a-empty v-if="selectedLog.user_action === 'ignored' && !selectedLog.user_feedback" description="暂无反馈" />
          <a-space v-else direction="vertical" style="width: 100%;">
            <a-tag :color="feedbackTagColor(selectedLog.user_action)">{{ selectedLog.user_action }}</a-tag>
            <div v-if="selectedLog.user_feedback">反馈: {{ selectedLog.user_feedback }}</div>
            <div v-if="selectedLog.user_overrides">覆盖: {{ JSON.stringify(selectedLog.user_overrides) }}</div>
          </a-space>
          <a-divider />
          <a-form layout="vertical">
            <a-form-item label="操作">
              <a-select v-model:value="feedbackAction" :options="feedbackOptions" />
            </a-form-item>
            <a-form-item label="修改内容 (JSON)">
              <a-textarea v-model:value="feedbackOverrides" :rows="4" placeholder='{"field": "value"}' />
            </a-form-item>
            <a-form-item label="反馈文字">
              <a-textarea v-model:value="feedbackText" :rows="2" />
            </a-form-item>
            <a-button type="primary" @click="submitFeedback" :loading="submittingFeedback">提交反馈</a-button>
          </a-form>
        </a-card>
      </a-space>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { agentApi } from '@/api/modules/agent'
import A5Form from './forms/A5Form.vue'
import G3Form from './forms/G3Form.vue'
import A6Form from './forms/A6Form.vue'
import A3Form from './forms/A3Form.vue'
import A1Form from './forms/A1Form.vue'
import A2Form from './forms/A2Form.vue'
import A4Form from './forms/A4Form.vue'
import A7Form from './forms/A7Form.vue'
import G2Form from './forms/G2Form.vue'

const router = useRouter()
const route = useRoute()
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

// Agent 专属表单映射
const formMap: Record<string, any> = {
  A5: A5Form, G3: G3Form, A6: A6Form, A3: A3Form,
  A1: A1Form, A2: A2Form, A4: A4Form, A7: A7Form, G2: G2Form,
}
const agentFormComponent = computed(() => formMap[agentId] || null)
const showJsonInput = ref(false)

function onDecision(data: any) {
  result.value = data
}

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
})

function tagColor(id: string) {
  const colors: Record<string, string> = {
    A3: 'orange', A4: 'green', A5: 'red',
    A6: 'purple', A7: 'blue',
    G1: 'cyan', G2: 'gold', G3: 'red',
  }
  return colors[id] || 'default'
}

function feedbackTagColor(action: string) {
  const map: Record<string, string> = { accepted: 'success', modified: 'warning', rejected: 'error', ignored: 'default' }
  return map[action] || 'default'
}

const logColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '决策点', dataIndex: 'decision_point', key: 'decision_point', width: 120 },
  { title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 80 },
  { title: '阶段', dataIndex: 'evolution_stage', key: 'evolution_stage', width: 100 },
  { title: '用户操作', dataIndex: 'user_action', key: 'user_action', width: 100 },
  { title: '规则', dataIndex: 'rules_applied', key: 'rules_applied', width: 70 },
  { title: '响应(ms)', dataIndex: 'response_time_ms', key: 'response_time_ms', width: 80 },
  { title: '操作', key: 'actions', width: 80 },
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

function onLogTableChange(pag: any) {
  logPagination.page = pag.current
  fetchLogs()
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
