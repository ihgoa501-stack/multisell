<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h2 style="margin: 0; font-size: 20px; font-weight: 600;">运营驾驶舱</h2>
      <span style="color: rgba(0,0,0,0.45); font-size: 14px;">汇总库存、利润、折扣风险与决策概览</span>
    </div>

    <!-- 概览卡片 -->
    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="6">
        <a-card :bordered="false" style="background: #f0f9ff;">
          <a-statistic :value="summary.total_decisions_7d" title="近7天决策数" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: #f0fdf4;">
          <a-statistic :value="summary.acceptance_rate_7d * 100" title="采纳率" suffix="%" :precision="0" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: #fff7ed;">
          <a-statistic :value="summary.pending_confirmations" title="待确认建议" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: #fef2f2;">
          <a-statistic :value="summary.active_risks" title="活跃风险" />
        </a-card>
      </a-col>
    </a-row>

    <!-- 待执行操作 -->
    <a-card title="待执行操作" style="margin-top: 12px;" :bordered="false">
      <a-empty v-if="actions.length === 0" description="无待执行操作" />
      <a-list v-else :data-source="actions" item-layout="horizontal">
        <template #renderItem="{ item }">
          <a-list-item>
            <template #actions>
              <a-button size="small" type="primary" @click="doExecute(item.id)">执行</a-button>
              <a-button size="small" @click="doReject(item.id)">忽略</a-button>
            </template>
            <a-list-item-meta>
              <template #avatar>
                <a-space>
                  <a-tag :color="actionTypeTag(item.action_type)">{{ item.action_type }}</a-tag>
                  <a-tag>{{ item.agent_id }}</a-tag>
                </a-space>
              </template>
              <template #title>{{ item.summary }}</template>
            </a-list-item-meta>
          </a-list-item>
        </template>
      </a-list>
    </a-card>

    <!-- 风险列表 -->
    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="12">
        <a-card title="风险列表" :bordered="false">
          <a-empty v-if="recentRisks.length === 0" description="暂无风险" />
          <a-list v-else :data-source="recentRisks" item-layout="horizontal">
            <template #renderItem="{ item }">
              <a-list-item>
                <template #actions>
                  <a-button size="small" type="link" @click="goToDecision(item.decision_id)">详情</a-button>
                </template>
                <a-list-item-meta>
                  <template #avatar>
                    <a-tag :color="item.severity === 'high' ? 'error' : 'warning'">{{ item.agent_id }}</a-tag>
                  </template>
                  <template #title>{{ item.risk_type }}: {{ item.sku }}</template>
                  <template #description>{{ item.detail }}</template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="待确认建议" :bordered="false">
          <a-empty v-if="pendingDecisions.length === 0" description="无待确认事项" />
          <a-list v-else :data-source="pendingDecisions" item-layout="horizontal">
            <template #renderItem="{ item }">
              <a-list-item>
                <template #actions>
                  <a-button size="small" type="link" @click="goToDecision(item.id)">查看</a-button>
                </template>
                <a-list-item-meta>
                  <template #avatar>
                    <a-tag color="processing">{{ item.agent_id }}</a-tag>
                  </template>
                  <template #title>{{ item.decision_point }}</template>
                  <template #description>{{ formatTime(item.created_at) }}</template>
                </a-list-item-meta>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>
    </a-row>

    <!-- 决策分布 + 规则健康 -->
    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="12">
        <a-card title="Agent 决策分布 (近7天)" :bordered="false">
          <a-empty v-if="Object.keys(decisionsByAgent).length === 0" description="暂无数据" />
          <div v-for="(count, agent) in decisionsByAgent" :key="agent" style="margin-bottom: 8px; display: flex; align-items: center; gap: 8px;">
            <a-tag :color="tagColor(agent as string)">{{ agent }}</a-tag>
            <span>{{ count }} 次决策</span>
          </div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="规则健康概览" :bordered="false">
          <a-empty v-if="ruleHealth.total === 0" description="暂无规则" />
          <a-space v-else direction="vertical" style="width: 100%;">
            <a-progress :percent="Math.round(ruleHealth.active / ruleHealth.total * 100)" stroke-color="var(--ant-color-success)">
              <template #format>活跃 {{ ruleHealth.active }} / {{ ruleHealth.total }}</template>
            </a-progress>
            <a-tag v-if="ruleHealth.shadow > 0" color="warning">Shadow: {{ ruleHealth.shadow }}</a-tag>
            <a-tag v-if="ruleHealth.retired_or_paused > 0">已停用: {{ ruleHealth.retired_or_paused }}</a-tag>
          </a-space>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { agentApi } from '@/api/modules/agent'

const router = useRouter()
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
    A3: 'orange', A4: 'green', A5: 'red',
    A6: 'purple', A7: 'blue',
    G1: 'cyan', G2: 'gold', G3: 'red',
  }
  return colors[agentId] || 'default'
}

function actionTypeTag(type: string) {
  const map: Record<string, string> = {
    replenish: 'error',
    discount_review: 'warning',
    price_review: 'warning',
    ad_action: 'processing',
  }
  return map[type] || 'default'
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getMonth() + 1}/${d.getDate()} ${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
}

function goToDecision(id: number) {
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

<style scoped>
:deep(.ant-card) {
  border-radius: 8px;
  transition: all 0.2s ease;
}

:deep(.ant-card:hover) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

:deep(.ant-statistic-content-value) {
  font-weight: 700;
}

:deep(.ant-card-head-title) {
  font-weight: 600;
  font-size: 15px;
}

:deep(.ant-list-item) {
  padding: 12px 16px;
  border-radius: 6px;
  margin-bottom: 4px;
  transition: background 0.15s ease;
}

:deep(.ant-list-item:hover) {
  background: var(--color-neutral-50, #f9fafb);
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    max-width: 100% !important;
  }

  :deep(.ant-statistic-content-value) {
    font-size: 20px;
  }
}
</style>
