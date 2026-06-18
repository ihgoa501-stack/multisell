<template>
  <div class="agent-container">
    <!-- ════ 页面标题区 ════ -->
    <div class="page-header">
      <div class="page-header-left">
        <h1 class="page-title">🤖 AI Agent 操作面板</h1>
        <p class="page-subtitle">智能助手状态总览 · 决策管理 · 风险控制</p>
      </div>
      <div class="page-header-right">
        <n-tag :bordered="false" type="success" size="small" v-if="systemStatus === 'healthy'">
          <template #icon><n-icon :component="CheckmarkCircleOutline" /></template>
          系统正常
        </n-tag>
        <n-tag :bordered="false" type="warning" size="small" v-else>
          <template #icon><n-icon :component="WarningOutline" /></template>
          {{ systemStatusText }}
        </n-tag>
        <n-button size="small" @click="fetchDashboard">
          <template #icon><n-icon :component="RefreshOutline" /></template>
          刷新
        </n-button>
      </div>
    </div>

    <n-spin :show="loading">
      <!-- ════ Agent 状态总览 ════ -->
      <div class="agent-overview-grid">
        <div class="agent-card" v-for="agent in agentList" :key="agent.id" @click="router.push(`/agents/${agent.id}`)">
          <div class="agent-card-header">
            <div class="agent-avatar" :class="'stage-' + agent.stage">
              {{ agent.name.charAt(0) }}
            </div>
            <div class="agent-info">
              <div class="agent-name">{{ agent.name }}</div>
              <div class="agent-role">{{ agent.role }}</div>
            </div>
            <n-tag size="small" :type="agent.stageType" round>{{ agent.stageLabel }}</n-tag>
          </div>
          <div class="agent-card-stats">
            <div class="agent-stat">
              <div class="agent-stat-value">{{ agent.decisions_7d || 0 }}</div>
              <div class="agent-stat-label">近7天决策</div>
            </div>
            <div class="agent-stat">
              <div class="agent-stat-value text-primary">{{ (agent.acceptance_rate * 100).toFixed(1) }}%</div>
              <div class="agent-stat-label">采纳率</div>
            </div>
            <div class="agent-stat">
              <div class="agent-stat-value" :class="agent.active_risks > 0 ? 'text-danger' : 'text-success'">{{ agent.active_risks || 0 }}</div>
              <div class="agent-stat-label">活跃风险</div>
            </div>
          </div>
          <div class="agent-card-footer">
            <n-progress
              :percentage="agent.health_score || 100"
              :height="4"
              :color="agent.health_score > 70 ? '#10b981' : '#ef4444'"
              :show-indicator="false"
            />
            <span class="health-label">健康度 {{ agent.health_score || 100 }}%</span>
          </div>
        </div>
      </div>

      <!-- ════ 决策概览 KPI 卡片 ════ -->
      <div class="kpi-grid">
        <div class="kpi-card kpi-primary">
          <div class="kpi-icon"><n-icon :component="BrainOutline" size="22" /></div>
          <div class="kpi-content">
            <div class="kpi-label">近7天决策数</div>
            <div class="kpi-value">{{ summary.total_decisions_7d || 0 }}</div>
          </div>
        </div>
        <div class="kpi-card kpi-success">
          <div class="kpi-icon"><n-icon :component="ThumbsUpOutline" size="22" /></div>
          <div class="kpi-content">
            <div class="kpi-label">采纳率</div>
            <div class="kpi-value">{{ (summary.acceptance_rate_7d * 100).toFixed(1) }}%</div>
          </div>
        </div>
        <div class="kpi-card kpi-warning">
          <div class="kpi-icon"><n-icon :component="TimeOutline" size="22" /></div>
          <div class="kpi-content">
            <div class="kpi-label">待确认建议</div>
            <div class="kpi-value">{{ summary.pending_confirmations || 0 }}</div>
          </div>
        </div>
        <div class="kpi-card kpi-danger">
          <div class="kpi-icon"><n-icon :component="WarningOutline" size="22" /></div>
          <div class="kpi-content">
            <div class="kpi-label">活跃风险</div>
            <div class="kpi-value">{{ summary.active_risks || 0 }}</div>
          </div>
        </div>
      </div>

      <!-- ════ 待执行操作 + 风险列表 ════ -->
      <div class="content-grid-2col">
        <!-- 待执行操作 -->
        <div class="action-panel">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="NotificationsOutline" size="16" />
              待执行操作
              <n-tag size="small" :bordered="false" type="warning">{{ actions.length }}</n-tag>
            </h3>
          </div>
          <div v-if="actions.length" class="action-list">
            <div v-for="act in actions" :key="act.id" class="action-item">
              <div class="action-item-header">
                <n-tag size="small" :type="actionTypeTag(act.action_type)" round>{{ act.action_type_label || act.action_type }}</n-tag>
                <n-tag size="small" :color="{ color: agentColor(act.agent_id), textColor: '#fff' }" round>{{ act.agent_id }}</n-tag>
                <span class="action-time">{{ formatTime(act.created_at) }}</span>
              </div>
              <div class="action-summary">{{ act.summary }}</div>
              <div class="action-footer">
                <n-button size="tiny" type="primary" @click="doExecute(act.id)">
                  <template #icon><n-icon :component="PlayOutline" /></template>
                  执行
                </n-button>
                <n-button size="tiny" @click="doReject(act.id)">
                  <template #icon><n-icon :component="CloseOutline" /></template>
                  忽略
                </n-button>
                <n-button size="tiny" quaternary @click="showActionDetail(act)">详情</n-button>
              </div>
            </div>
          </div>
          <n-empty v-else description="无待执行操作" />
        </div>

        <!-- 风险列表 + 待确认建议 -->
        <div class="right-panels">
          <div class="risk-panel">
            <div class="card-header">
              <h3 class="card-title">
                <n-icon :component="AlertCircleOutline" size="16" />
                活跃风险
                <n-tag size="small" :bordered="false" type="error">{{ recentRisks.length }}</n-tag>
              </h3>
            </div>
            <div v-if="recentRisks.length" class="risk-list">
              <div v-for="risk in recentRisks" :key="risk.decision_id" class="risk-item">
                <div class="risk-header">
                  <n-tag size="small" :type="risk.severity === 'high' ? 'error' : 'warning'" round>{{ risk.severity === 'high' ? '高危' : '中危' }}</n-tag>
                  <n-tag size="small" :color="{ color: agentColor(risk.agent_id), textColor: '#fff' }" round>{{ risk.agent_id }}</n-tag>
                </div>
                <div class="risk-content">
                  <div class="risk-type">{{ risk.risk_type }}</div>
                  <div class="risk-sku">{{ risk.sku }}</div>
                  <div class="risk-detail">{{ risk.detail }}</div>
                </div>
                <n-button size="tiny" quaternary @click="goToDecision(risk.decision_id)">查看决策 →</n-button>
              </div>
            </div>
            <n-empty v-else description="暂无风险" />
          </div>

          <div class="decision-panel">
            <div class="card-header">
              <h3 class="card-title">
                <n-icon :component="DocumentTextOutline" size="16" />
                待确认建议
                <n-tag size="small" :bordered="false" type="info">{{ pendingDecisions.length }}</n-tag>
              </h3>
            </div>
            <div v-if="pendingDecisions.length" class="decision-list">
              <div v-for="pd in pendingDecisions" :key="pd.id" class="decision-item">
                <n-tag size="small" :color="{ color: agentColor(pd.agent_id), textColor: '#fff' }" round>{{ pd.agent_id }}</n-tag>
                <span class="decision-point">{{ pd.decision_point }}</span>
                <span class="decision-time">{{ formatTime(pd.created_at) }}</span>
                <n-button size="tiny" quaternary @click="goToDecision(pd.id)">查看</n-button>
              </div>
            </div>
            <n-empty v-else description="无待确认事项" />
          </div>
        </div>
      </div>

      <!-- ════ 决策分布 + 规则健康 ════ -->
      <div class="content-grid-2col">
        <div class="distribution-panel">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="BarChartOutline" size="16" />
              Agent 决策分布（近7天）
            </h3>
          </div>
          <div v-if="Object.keys(decisionsByAgent).length" class="agent-distribution-list">
            <div v-for="(count, agent) in decisionsByAgent" :key="agent" class="agent-distribution-item">
              <n-tag size="small" :color="{ color: agentColor(agent), textColor: '#fff' }" round>{{ agent }}</n-tag>
              <div class="distribution-bar-container">
                <div class="distribution-bar" :style="{ width: (count / maxDecisionCount * 100) + '%', background: agentColor(agent) }"></div>
              </div>
              <span class="distribution-count">{{ count }} 次</span>
            </div>
          </div>
          <n-empty v-else description="暂无数据" />
        </div>

        <div class="rule-panel">
          <div class="card-header">
            <h3 class="card-title">
              <n-icon :component="ConstructOutline" size="16" />
              规则健康概览
            </h3>
            <n-button size="small" @click="router.push('/agents/rules')">管理规则 →</n-button>
          </div>
          <div v-if="ruleHealth.total > 0" class="rule-health">
            <div class="rule-progress">
              <n-progress
                :percentage="ruleHealth.active / ruleHealth.total * 100"
                :indicator-placement="'inside'"
                type="line"
                color="#10b981"
                :show-indicator="true"
              >
                {{ ruleHealth.active }} / {{ ruleHealth.total }} 活跃
              </n-progress>
            </div>
            <div class="rule-tags">
              <n-tag v-if="ruleHealth.shadow > 0" type="warning" size="small">
                Shadow 规则: {{ ruleHealth.shadow }}
              </n-tag>
              <n-tag v-if="ruleHealth.retired_or_paused > 0" type="default" size="small">
                已停用: {{ ruleHealth.retired_or_paused }}
              </n-tag>
            </div>
          </div>
          <n-empty v-else description="暂无规则" />
        </div>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { Component } from 'vue'
import { agentApi } from '@/api/modules/agent'

// Icons
import {
  CheckmarkCircleOutline,
  WarningOutline,
  RefreshOutline,
  BrainOutline,
  ThumbsUpOutline,
  TimeOutline,
  NotificationsOutline,
  AlertCircleOutline,
  DocumentTextOutline,
  BarChartOutline,
  ConstructOutline,
  PlayOutline,
  CloseOutline,
} from '@vicons/ionicons5'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const systemStatus = ref('healthy')
const systemStatusText = ref('')

// ── 数据 ──
const summary = reactive({
  total_decisions_7d: 0,
  acceptance_rate_7d: 0,
  pending_confirmations: 0,
  active_risks: 0,
})

const agentList = ref<any[]>([
  { id: 'G1', name: 'G1 总经理', role: '全局决策与资源调配', stage: 3, stageLabel: '半自主', stageType: 'warning', decisions_7d: 0, acceptance_rate: 0, active_risks: 0, health_score: 100 },
  { id: 'A5', name: 'A5 市场分析师', role: '市场趋势与选品建议', stage: 2, stageLabel: '建议', stageType: 'info', decisions_7d: 0, acceptance_rate: 0, active_risks: 0, health_score: 100 },
  { id: 'A1', name: 'A1 选品专家', role: '选品策略与优化', stage: 2, stageLabel: '建议', stageType: 'info', decisions_7d: 0, acceptance_rate: 0, active_risks: 0, health_score: 100 },
  { id: 'A3', name: 'A3 定价专家', role: '动态定价与利润优化', stage: 3, stageLabel: '半自主', stageType: 'warning', decisions_7d: 0, acceptance_rate: 0, active_risks: 0, health_score: 100 },
])

const actions = ref<any[]>([])
const recentRisks = ref<any[]>([])
const pendingDecisions = ref<any[]>([])
const decisionsByAgent = reactive<Record<string, number>>({})
const ruleHealth = reactive({ total: 0, active: 0, shadow: 0, retired_or_paused: 0 })

const maxDecisionCount = computed(() => {
  const counts = Object.values(decisionsByAgent)
  return Math.max(...counts, 1)
})

// ── 辅助函数 ──
function agentColor(agentId: string): string {
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
  const targetAgent = recentRisks.value.find(r => r.decision_id === id)
  if (targetAgent) {
    router.push(`/agents/${targetAgent.agent_id}`)
  } else {
    router.push('/agents')
  }
}

function showActionDetail(act: any) {
  // 显示操作详情，可以打开一个抽屉或模态框
  message.info(`查看操作详情: ${act.summary}`)
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

// ── 数据获取 ──
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

    // 更新 Agent 列表数据
    if (data.agent_stats) {
      agentList.value = agentList.value.map(agent => ({
        ...agent,
        ...data.agent_stats[agent.id] || {},
      }))
    }
  } catch (e: any) {
    message.error(e?.response?.data?.message || '获取驾驶舱数据失败')
  } finally {
    loading.value = false
  }
}

async function fetchActions() {
  try {
    const res: any = await agentApi.listActions({ status: 'pending' })
    const records = res?.records || res?.data?.records || []
    actions.value = records
  } catch { /* ignore */ }
}

onMounted(() => { fetchDashboard(); fetchActions() })
</script>

<style scoped>
/* ════ 设计系统 Token 应用 ════ */
.agent-container {
  padding: 24px;
  max-width: 1440px;
  margin: 0 auto;
  background: #f8fafc;
  min-height: 100vh;
}

/* ════ 页面标题 ════ */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #e2e8f0;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px 0;
}
.page-subtitle {
  font-size: 13px;
  color: #94a3b8;
  margin: 0;
}
.page-header-right {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* ════ Agent 状态总览网格 ════ */
.agent-overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}
.agent-card {
  background: white;
  border-radius: 12px;
  padding: 16px;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  transition: all 0.2s ease;
}
.agent-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
.agent-card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.agent-avatar {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  color: white;
  flex-shrink: 0;
}
.stage-1 { background: linear-gradient(135deg, #0ea5e9, #38bdf8); }  /* 观察 */
.stage-2 { background: linear-gradient(135deg, #6366f1, #818cf8); }  /* 建议 */
.stage-3 { background: linear-gradient(135deg, #f59e0b, #fbbf24); }  /* 半自主 */
.stage-4 { background: linear-gradient(135deg, #10b981, #34d399); }  /* 全自主 */
.agent-info { flex: 1; }
.agent-name { font-size: 13px; font-weight: 600; color: #1e293b; }
.agent-role { font-size: 11px; color: #94a3b8; }

.agent-card-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f1f5f9;
}
.agent-stat { text-align: center; }
.agent-stat-value { font-size: 18px; font-weight: 700; color: #1e293b; }
.agent-stat-label { font-size: 10px; color: #94a3b8; margin-top: 2px; }

.agent-card-footer {
  display: flex;
  align-items: center;
  gap: 8px;
}
.health-label {
  font-size: 11px;
  color: #94a3b8;
  white-space: nowrap;
}

/* ════ KPI 卡片 ════ */
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.kpi-card {
  background: white;
  border-radius: 12px;
  padding: 18px 16px;
  border: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  gap: 14px;
  transition: all 0.2s ease;
}
.kpi-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}
.kpi-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}
.kpi-primary .kpi-icon { background: linear-gradient(135deg, #0ea5e9, #38bdf8); }
.kpi-success .kpi-icon { background: linear-gradient(135deg, #10b981, #34d399); }
.kpi-warning .kpi-icon { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
.kpi-danger .kpi-icon { background: linear-gradient(135deg, #ef4444, #f87171); }
.kpi-label { font-size: 12px; color: #94a3b8; margin-bottom: 4px; }
.kpi-value { font-size: 24px; font-weight: 700; color: #1e293b; }
.text-primary { color: #0ea5e9; }
.text-success { color: #10b981; }
.text-danger { color: #ef4444; }

/* ════ 内容网格 ════ */
.content-grid-2col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 20px;
}

/* ════ 卡片通用样式 ════ */
.action-panel,
.risk-panel,
.decision-panel,
.distribution-panel,
.rule-panel {
  background: white;
  border-radius: 12px;
  padding: 18px;
  border: 1px solid #e2e8f0;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.card-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

/* ════ 待执行操作列表 ════ */
.action-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 500px;
  overflow-y: auto;
}
.action-item {
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
  transition: all 0.15s ease;
}
.action-item:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}
.action-item-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.action-time {
  margin-left: auto;
  font-size: 11px;
  color: #94a3b8;
}
.action-summary {
  font-size: 13px;
  color: #334155;
  margin-bottom: 10px;
  line-height: 1.5;
}
.action-footer {
  display: flex;
  gap: 6px;
}

/* ════ 右侧面板 ════ */
.right-panels {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ════ 风险列表 ════ */
.risk-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 280px;
  overflow-y: auto;
}
.risk-item {
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid #fef2f2;
  background: #fffcfc;
  transition: all 0.15s ease;
}
.risk-item:hover {
  background: #fef2f2;
}
.risk-header {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.risk-content {
  margin-bottom: 8px;
}
.risk-type {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}
.risk-sku {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 2px;
}
.risk-detail {
  font-size: 12px;
  color: #94a3b8;
}

/* ════ 待确认建议列表 ════ */
.decision-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 200px;
  overflow-y: auto;
}
.decision-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  transition: background 0.15s;
}
.decision-item:hover {
  background: #f8fafc;
}
.decision-point {
  flex: 1;
  font-size: 13px;
  color: #334155;
}
.decision-time {
  font-size: 11px;
  color: #94a3b8;
  white-space: nowrap;
}

/* ════ Agent 决策分布 ════ */
.agent-distribution-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.agent-distribution-item {
  display: flex;
  align-items: center;
  gap: 10px;
}
.distribution-bar-container {
  flex: 1;
  height: 16px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}
.distribution-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.5s ease;
  min-width: 4px;
}
.distribution-count {
  width: 60px;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

/* ════ 规则健康 ════ */
.rule-health {
  padding: 8px 0;
}
.rule-progress {
  margin-bottom: 12px;
}
.rule-tags {
  display: flex;
  gap: 6px;
}

/* ════ 响应式 ════ */
@media (max-width: 1280px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .content-grid-2col { grid-template-columns: 1fr; }
}
@media (max-width: 960px) {
  .kpi-grid { grid-template-columns: 1fr; }
  .agent-overview-grid { grid-template-columns: 1fr; }
}
</style>
