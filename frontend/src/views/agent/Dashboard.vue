<template>
  <div>
    <n-page-header subtitle="汇总库存、利润、折扣风险与决策概览">
      <template #title>运营驾驶舱</template>
    </n-page-header>

    <!-- 概览卡片 -->
    <n-grid :cols="4" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card :bordered="false" style="background: #f0f9ff;">
          <n-statistic :value="summary.total_decisions_7d" title="近7天决策数" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: #f0fdf4;">
          <n-statistic :value="summary.acceptance_rate_7d * 100 + '%'" title="采纳率">
            <template #suffix>
              <span style="font-size: 14px;">%</span>
            </template>
          </n-statistic>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: #fff7ed;">
          <n-statistic :value="summary.pending_confirmations" title="待确认建议" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: #fef2f2;">
          <n-statistic :value="summary.active_risks" title="活跃风险" />
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- 待执行操作 -->
    <n-card title="待执行操作" style="margin-top: 12px;" :bordered="false">
      <n-empty v-if="actions.length === 0" description="无待执行操作" />
      <n-list v-else>
        <n-list-item v-for="act in actions" :key="act.id">
          <template #prefix>
            <n-tag :type="actionTypeTag(act.action_type)" size="small">{{ act.action_type }}</n-tag>
            <n-tag size="small" style="margin-left: 4px;">{{ act.agent_id }}</n-tag>
          </template>
          <span>{{ act.summary }}</span>
          <template #suffix>
            <n-space>
              <n-button size="tiny" type="primary" @click="doExecute(act.id)">执行</n-button>
              <n-button size="tiny" @click="doReject(act.id)">忽略</n-button>
            </n-space>
          </template>
        </n-list-item>
      </n-list>
    </n-card>

    <!-- 风险列表 -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="风险列表" :bordered="false">
          <n-empty v-if="recentRisks.length === 0" description="暂无风险" />
          <n-list v-else>
            <n-list-item v-for="risk in recentRisks" :key="risk.decision_id">
              <template #prefix>
                <n-tag :type="risk.severity === 'high' ? 'error' : 'warning'" size="small">{{ risk.agent_id }}</n-tag>
              </template>
              <n-space vertical size="small">
                <span style="font-weight: 500;">{{ risk.risk_type }}: {{ risk.sku }}</span>
                <span style="font-size: 12px; color: #666;">{{ risk.detail }}</span>
              </n-space>
              <template #suffix>
                <n-button size="tiny" quaternary @click="goToDecision(risk.decision_id)">详情</n-button>
              </template>
            </n-list-item>
          </n-list>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="待确认建议" :bordered="false">
          <n-empty v-if="pendingDecisions.length === 0" description="无待确认事项" />
          <n-list v-else>
            <n-list-item v-for="pd in pendingDecisions" :key="pd.id">
              <template #prefix>
                <n-tag size="small" type="info">{{ pd.agent_id }}</n-tag>
              </template>
              <n-space>
                <span>{{ pd.decision_point }}</span>
                <span style="font-size: 12px; color: #999;">{{ formatTime(pd.created_at) }}</span>
              </n-space>
              <template #suffix>
                <n-button size="tiny" quaternary @click="goToDecision(pd.id)">查看</n-button>
              </template>
            </n-list-item>
          </n-list>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- 决策分布 + 规则健康 -->
    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card title="Agent 决策分布 (近7天)" :bordered="false">
          <n-empty v-if="Object.keys(decisionsByAgent).length === 0" description="暂无数据" />
          <div v-for="(count, agent) in decisionsByAgent" :key="agent" style="margin-bottom: 8px; display: flex; align-items: center; gap: 8px;">
            <n-tag size="small" :color="{ color: tagColor(agent), textColor: '#fff' }">{{ agent }}</n-tag>
            <span>{{ count }} 次决策</span>
          </div>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card title="规则健康概览" :bordered="false">
          <n-empty v-if="ruleHealth.total === 0" description="暂无规则" />
          <n-space v-else vertical>
            <n-progress :percentage="ruleHealth.active / ruleHealth.total * 100" :indicator-placement="'inside'" type="line" color="#18a058">
              活跃 {{ ruleHealth.active }} / {{ ruleHealth.total }}
            </n-progress>
            <n-tag v-if="ruleHealth.shadow > 0" type="warning">Shadow: {{ ruleHealth.shadow }}</n-tag>
            <n-tag v-if="ruleHealth.retired_or_paused > 0" type="default">已停用: {{ ruleHealth.retired_or_paused }}</n-tag>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentApi } from '@/api/modules/agent'

const router = useRouter()
const message = useMessage()
const loading = ref(false)

const summary = reactive({
  total_decisions_7d: 0,
  acceptance_rate_7d: 0,
  pending_confirmations: 0,
  active_risks: 0,
})
const recentRisks = ref<any[]>([])
const pendingDecisions = ref<any[]>([])
const actions = ref<any[]>([])
const decisionsByAgent = reactive<Record<string, number>>({})
const ruleHealth = reactive({ total: 0, active: 0, shadow: 0, retired_or_paused: 0 })

function tagColor(agentId: string) {
  const colors: Record<string, string> = {
    A3: '#e67e22', A4: '#2ecc71', A5: '#e74c3c',
    A6: '#9b59b6', A7: '#3498db',
    G1: '#1abc9c', G2: '#f39c12', G3: '#e74c3c',
  }
  return colors[agentId] || '#95a5a6'
}

function actionTypeTag(type: string) {
  const map: Record<string, string> = {
    replenish: 'error',
    discount_review: 'warning',
    price_review: 'warning',
    ad_action: 'info',
  }
  return map[type] || 'default'
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

function goToDecision(id: number) {
  // 跳转到 Agent 详情页并打开决策抽屉
  // 目前跳转到对应 Agent 的详情页，用户可查看决策列表
  const targetAgent = recentRisks.value.find(r => r.decision_id === id)
  if (targetAgent) {
    router.push(`/agents/${targetAgent.agent_id}`)
  } else {
    router.push('/agents')
  }
}

async function fetchDashboard() {
  loading.value = true
  try {
    const res: any = await agentApi.getDashboard()
    const data = res?.data
    if (!data) return

    Object.assign(summary, data.summary || {})
    recentRisks.value = data.recent_risks || []
    pendingDecisions.value = data.pending_decisions || []
    Object.assign(decisionsByAgent, data.decisions_by_agent || {})
    Object.assign(ruleHealth, data.rule_health || {})
  } catch (e: any) {
    message.error(e?.response?.data?.message || '获取驾驶舱数据失败')
  }
  loading.value = false
}

async function fetchActions() {
  try {
    const res: any = await agentApi.listActions({ status: 'pending' })
    const records = res?.records || res?.data?.records || []
    actions.value = records
  } catch { /* ignore */ }
}

async function doExecute(id: number) {
  try {
    const res: any = await agentApi.executeAction(id)
    message.success('操作已执行')
    actions.value = actions.value.filter((a: any) => a.id !== id)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '执行失败')
  }
}

async function doReject(id: number) {
  try {
    await agentApi.rejectAction(id)
    message.info('操作已忽略')
    actions.value = actions.value.filter((a: any) => a.id !== id)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '操作失败')
  }
}

onMounted(() => { fetchDashboard(); fetchActions() })
</script>
