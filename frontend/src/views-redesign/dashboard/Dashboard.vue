<!-- ================================================
   FLOW: Dashboard Workbench
   SCREEN 1 of 2: Decision Cockpit
   ------------------------------------------------
   ENTRY:  Sidebar → 指挥中心 → 运营驾驶舱
   EXIT:   Click any card/widget → navigate to detail page
   BRANCH: Click agent card → Agent Activity Sheet (Screen 2)
   ================================================ -->
<template>
  <div class="dashboard-page">
    <!-- ═══ Page Header ═══ -->
    <div class="page-header">
      <div>
        <h1 class="page-title">运营驾驶舱</h1>
        <p class="page-subtitle">决策系统总览 · Agent 智能运营</p>
      </div>
      <a-button @click="refreshData" :loading="loading">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <!-- ═══ Agent Decision Summary Bar ═══ -->
    <a-card class="agent-summary-bar" :bordered="false">
      <div class="summary-content">
        <div class="summary-icon">
          <a-avatar :size="42" style="background: linear-gradient(135deg, #7C3AED, #2962FF)">
            <template #icon><RobotOutlined /></template>
          </a-avatar>
        </div>
        <div class="summary-stats">
          <div class="summary-title">Agent 今日决策摘要</div>
          <div class="summary-metrics">
            <span class="summary-metric">
              <strong>{{ decisionSummary.total_decisions_today }}</strong> 次决策
            </span>
            <a-divider type="vertical" />
            <span class="summary-metric summary-metric-warn">
              <strong>{{ decisionSummary.pending_approvals }}</strong> 待审批
            </span>
            <a-divider type="vertical" />
            <span class="summary-metric summary-metric-good">
              <strong>{{ decisionSummary.auto_executed }}</strong> 自动执行
            </span>
            <a-divider type="vertical" />
            <span class="summary-metric">
              信任分 <strong>{{ decisionSummary.avg_trust_score }}</strong>
            </span>
          </div>
        </div>
        <a-button type="primary" ghost @click="router.push('/redesign/agentos')">
          进入 AgentOS
          <template #icon><ArrowRightOutlined /></template>
        </a-button>
      </div>
    </a-card>

    <!-- ═══ KPI Cards (4 columns) ═══ -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :sm="12" :lg="6" v-for="kpi in kpis" :key="kpi.key">
        <a-card class="kpi-card" :bordered="false" hoverable @click="handleKpiClick(kpi)">
          <div class="kpi-header">
            <span class="kpi-label">{{ kpi.label }}</span>
            <span class="kpi-trend" :class="kpi.trend >= 0 ? 'trend-up' : 'trend-down'">
              <component :is="kpi.trend >= 0 ? CaretUpOutlined : CaretDownOutlined" />
              {{ Math.abs(kpi.trend) }}%
            </span>
          </div>
          <div class="kpi-value">
            <span class="kpi-number" :style="{ color: kpi.color }">{{ formatKpiValue(kpi) }}</span>
            <span class="kpi-unit">{{ kpi.unit }}</span>
          </div>
          <div class="kpi-trend-label">{{ kpi.trend_label }}</div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ═══ Row 2: Order Trend + Priority Queue ═══ -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="16">
        <a-card title="订单趋势（近 30 天）" :bordered="false" class="section-card">
          <template #extra>
            <a-radio-group v-model:value="trendRange" size="small" button-style="solid">
              <a-radio-button value="7d">7天</a-radio-button>
              <a-radio-button value="30d">30天</a-radio-button>
            </a-radio-group>
          </template>
          <div class="trend-chart">
            <div
              v-for="d in displayTrend"
              :key="d.date"
              class="trend-bar-wrapper"
            >
              <a-tooltip :title="`${d.date}: ${d.orders}单 / ¥${d.revenue.toLocaleString()}`">
                <div class="trend-bar-col">
                  <span class="trend-bar-value">{{ d.orders }}</span>
                  <div
                    class="trend-bar"
                    :style="{
                      height: Math.max((d.revenue / maxRevenue) * 140, 3) + 'px',
                      background: d.date === displayTrend[displayTrend.length - 1]?.date
                        ? 'var(--ant-color-primary)' : 'var(--ant-color-primary-bg)',
                    }"
                  />
                  <span class="trend-bar-label">{{ d.date.slice(5) }}</span>
                </div>
              </a-tooltip>
            </div>
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="8">
        <a-card :bordered="false" class="section-card">
          <template #title>
            <span>
              <FireOutlined style="color: #DC2626; margin-right: 6px" />
              待处理队列
            </span>
          </template>
          <template #extra>
            <a-badge :count="pendingItems.length" :number-style="{ backgroundColor: pendingItems.length > 0 ? '#D97706' : '#059669' }" />
          </template>
          <a-empty v-if="pendingItems.length === 0" description="暂无待处理事项" />
          <div v-else class="queue-list">
            <div
              v-for="item in pendingItems"
              :key="item.id"
              class="queue-item"
              @click="router.push(item.action_url || '/redesign/agentos')"
            >
              <div class="queue-item-left">
                <a-tag
                  :color="riskColor(item.risk_level)"
                  :bordered="false"
                  style="font-size: 11px"
                >
                  {{ riskLabel(item.risk_level) }}
                </a-tag>
                <div class="queue-item-info">
                  <div class="queue-item-title">{{ item.title }}</div>
                  <div class="queue-item-meta">{{ item.agent_name }} · {{ timeAgo(item.created_at) }}</div>
                </div>
              </div>
              <ArrowRightOutlined class="queue-item-arrow" />
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ═══ Row 3: Agent Activity Timeline ═══ -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="Agent 活动流" :bordered="false" class="section-card">
          <template #extra>
            <a-button type="link" size="small" @click="router.push('/redesign/agentos')">
              查看全部 <ArrowRightOutlined />
            </a-button>
          </template>
          <a-timeline>
            <a-timeline-item
              v-for="action in recentActions"
              :key="action.id"
              :color="actionColor(action.status)"
            >
              <div class="activity-item">
                <div class="activity-header">
                  <a-avatar :size="24" :style="{ backgroundColor: getAgentColor(action.agent_id), fontSize: '11px' }">
                    {{ action.agent_name.charAt(0) }}
                  </a-avatar>
                  <span class="activity-agent">{{ action.agent_name }}</span>
                  <a-tag :bordered="false" size="small">{{ actionTypeLabel(action.action_type) }}</a-tag>
                  <span class="activity-time">{{ timeAgo(action.created_at) }}</span>
                </div>
                <div class="activity-desc">{{ action.description }}</div>
                <div class="activity-meta">
                  <span v-if="action.confidence">置信度 {{ action.confidence }}%</span>
                  <span v-if="action.impact"> · {{ action.impact }}</span>
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>
        </a-card>
      </a-col>
    </a-row>

    <!-- ═══ Agent Activity Sheet (Screen 2) ═══ -->
    <a-drawer
      v-model:open="showAgentSheet"
      :title="selectedAgent?.name || 'Agent 详情'"
      placement="right"
      width="480"
    >
      <template v-if="selectedAgent">
        <div class="agent-detail-header">
          <a-avatar :size="56" :style="{ backgroundColor: selectedAgent.avatar_color, fontSize: '20px' }">
            {{ selectedAgent.name.charAt(0) }}
          </a-avatar>
          <div class="agent-detail-info">
            <h3 style="margin: 0">{{ selectedAgent.name }}</h3>
            <p style="color: var(--ant-color-text-secondary); margin: 4px 0 0; font-size: 13px">
              {{ selectedAgent.description }}
            </p>
          </div>
        </div>

        <a-descriptions :column="2" size="small" style="margin-top: 20px" bordered>
          <a-descriptions-item label="状态">
            <a-badge :status="agentBadgeStatus(selectedAgent.status)" :text="agentStatusLabel(selectedAgent.status)" />
          </a-descriptions-item>
          <a-descriptions-item label="信任分">
            <a-progress
              type="dashboard"
              :percent="selectedAgent.trust_score"
              :size="40"
              :stroke-color="selectedAgent.trust_score >= 80 ? '#059669' : '#D97706'"
            />
          </a-descriptions-item>
          <a-descriptions-item label="自治等级">{{ autonomyLabel(selectedAgent.autonomy_level) }}</a-descriptions-item>
          <a-descriptions-item label="进化阶段">{{ selectedAgent.evolution_stage }}</a-descriptions-item>
          <a-descriptions-item label="成功率">{{ selectedAgent.success_rate }}%</a-descriptions-item>
          <a-descriptions-item label="累计动作">{{ selectedAgent.total_actions }}</a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <h4>最近决策</h4>
        <a-timeline>
          <a-timeline-item
            v-for="act in agentActions"
            :key="act.id"
            :color="actionColor(act.status)"
          >
            <div class="activity-desc">{{ act.description }}</div>
            <div class="activity-meta">{{ timeAgo(act.created_at) }}</div>
          </a-timeline-item>
        </a-timeline>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
/* ================================================
   FLOW: Dashboard Workbench
   SCREEN 1-2 of 2: Decision Cockpit + Agent Sheet
   ================================================ */
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  RobotOutlined, ReloadOutlined, ArrowRightOutlined, FireOutlined,
  CaretUpOutlined, CaretDownOutlined,
} from '@ant-design/icons-vue'
import type { DashboardKPI, Agent, WorkItem, AgentAction } from '@/views-redesign/shared/types'
import {
  mockDashboardKPIs,
  mockOrderTrend,
  mockAgentDecisionSummary,
  mockWorkItems,
  mockAgentActions,
  mockAgents,
} from '@/views-redesign/shared/mock-data'

const router = useRouter()
const loading = ref(false)
const trendRange = ref('30d')

// Data
const kpis = ref<DashboardKPI[]>(mockDashboardKPIs)
const decisionSummary = ref(mockAgentDecisionSummary)
const orderTrend = ref(mockOrderTrend)
const pendingItems = ref<WorkItem[]>(mockWorkItems.filter(w => w.status === 'pending'))
const recentActions = ref<AgentAction[]>(mockAgentActions.slice(0, 6))

// Agent Sheet
const showAgentSheet = ref(false)
const selectedAgent = ref<Agent | null>(null)

const agentActions = computed(() => {
  if (!selectedAgent.value) return []
  return mockAgentActions.filter(a => a.agent_id === selectedAgent.value!.id)
})

// Trend
const displayTrend = computed(() => {
  if (trendRange.value === '7d') return orderTrend.value.slice(-7)
  return orderTrend.value
})

const maxRevenue = computed(() => Math.max(...displayTrend.value.map(d => d.revenue), 1))

// ── Helpers ──
function formatKpiValue(kpi: DashboardKPI): string {
  if (kpi.unit === '元') return (kpi.value / 10000).toFixed(1) + '万'
  if (kpi.unit === '%') return kpi.value.toFixed(1)
  if (kpi.value >= 1000) return (kpi.value / 1000).toFixed(1) + 'K'
  return kpi.value.toLocaleString()
}

function handleKpiClick(kpi: DashboardKPI) {
  const routes: Record<string, string> = {
    revenue: '/finance',
    orders: '/orders',
    profit_margin: '/finance',
    inventory_health: '/products/u-001/inventory',
  }
  router.push(routes[kpi.key] || '/redesign/dashboard')
}

function riskColor(level: string): string {
  const map: Record<string, string> = { critical: 'red', high: 'orange', medium: 'blue', low: 'default' }
  return map[level] || 'default'
}

function riskLabel(level: string): string {
  const map: Record<string, string> = { critical: '紧急', high: '重要', medium: '一般', low: '低' }
  return map[level] || level
}

function timeAgo(t: string): string {
  if (!t) return ''
  const s = Math.floor((Date.now() - new Date(t).getTime()) / 1000)
  if (s < 60) return '刚刚'
  if (s < 3600) return `${Math.floor(s / 60)}分钟前`
  if (s < 86400) return `${Math.floor(s / 3600)}小时前`
  return `${Math.floor(s / 86400)}天前`
}

function actionColor(status: string): string {
  const map: Record<string, string> = { proposed: 'blue', approved: 'green', executed: 'green', rejected: 'red' }
  return map[status] || 'gray'
}

function actionTypeLabel(type: string): string {
  const map: Record<string, string> = {
    price_adjust: '调价', listing_publish: '上架', compliance_check: '合规',
    analysis: '分析', reorder: '补货', optimize: '优化',
  }
  return map[type] || type
}

function getAgentColor(agentId: string): string {
  const agent = mockAgents.find(a => a.id === agentId)
  return agent?.avatar_color || '#999'
}

function agentBadgeStatus(status: string): 'success' | 'warning' | 'default' | 'processing' {
  const map: Record<string, 'success' | 'warning' | 'default' | 'processing'> = {
    online: 'success', idle: 'warning', offline: 'default', thinking: 'processing',
  }
  return map[status] || 'default'
}

function agentStatusLabel(status: string): string {
  const map: Record<string, string> = { online: '在线', idle: '空闲', offline: '离线', thinking: '思考中' }
  return map[status] || status
}

function autonomyLabel(level: string): string {
  const map: Record<string, string> = {
    observation: '观察模式', suggestion: '建议模式',
    semi_autonomous: '半自治模式', full_autonomous: '全自治模式',
  }
  return map[level] || level
}

async function refreshData() {
  loading.value = true
  await new Promise(r => setTimeout(r, 800))
  loading.value = false
}

onMounted(() => {
  // Data would be fetched from API in production
})
</script>

<style scoped>
.dashboard-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0;
  color: var(--ant-color-text, #111827);
}
.page-subtitle {
  font-size: 14px;
  color: var(--ant-color-text-secondary, #6b7280);
  margin: 4px 0 0;
}

/* ═══ Agent Summary Bar ═══ */
.agent-summary-bar {
  background: linear-gradient(135deg, #f0f4ff 0%, #f5f0ff 100%);
  border-radius: 12px;
}
.agent-summary-bar :deep(.ant-card-body) {
  padding: 16px 24px;
}
.summary-content {
  display: flex;
  align-items: center;
  gap: 16px;
}
.summary-stats {
  flex: 1;
}
.summary-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--ant-color-text, #111827);
  margin-bottom: 4px;
}
.summary-metrics {
  font-size: 13px;
  color: var(--ant-color-text-secondary, #6b7280);
}
.summary-metric strong {
  color: var(--ant-color-text, #111827);
  font-weight: 700;
}
.summary-metric-warn strong {
  color: #D97706;
}
.summary-metric-good strong {
  color: #059669;
}

/* ═══ KPI Cards ═══ */
.kpi-card {
  border-radius: 12px;
  transition: all 0.2s;
  cursor: pointer;
}
.kpi-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}
.kpi-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.kpi-label {
  font-size: 13px;
  color: var(--ant-color-text-secondary, #6b7280);
}
.kpi-trend {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 12px;
}
.trend-up {
  color: #059669;
  background: #ecfdf5;
}
.trend-down {
  color: #DC2626;
  background: #fef2f2;
}
.kpi-value {
  margin: 8px 0 4px;
}
.kpi-number {
  font-size: 28px;
  font-weight: 700;
  line-height: 1;
}
.kpi-unit {
  font-size: 14px;
  color: var(--ant-color-text-tertiary, #9ca3af);
  margin-left: 4px;
}
.kpi-trend-label {
  font-size: 12px;
  color: var(--ant-color-text-tertiary, #9ca3af);
}

/* ═══ Section Cards ═══ */
.section-card {
  border-radius: 12px;
  height: 100%;
}

/* ═══ Trend Chart ═══ */
.trend-chart {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 180px;
  padding: 8px 0;
}
.trend-bar-wrapper {
  flex: 1;
}
.trend-bar-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
}
.trend-bar-value {
  font-size: 9px;
  color: var(--ant-color-text-tertiary, #9ca3af);
  margin-bottom: 2px;
}
.trend-bar {
  width: 100%;
  border-radius: 3px 3px 0 0;
  transition: all 0.3s;
  min-height: 3px;
}
.trend-bar-label {
  font-size: 9px;
  color: var(--ant-color-text-quaternary, #d1d5db);
  margin-top: 4px;
  writing-mode: vertical-lr;
  max-height: 24px;
  overflow: hidden;
}

/* ═══ Queue ═══ */
.queue-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.queue-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}
.queue-item:hover {
  background: var(--ant-color-bg-text-hover, #fafafa);
}
.queue-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
.queue-item-info {
  min-width: 0;
}
.queue-item-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--ant-color-text, #111827);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.queue-item-meta {
  font-size: 11px;
  color: var(--ant-color-text-tertiary, #9ca3af);
  margin-top: 2px;
}
.queue-item-arrow {
  font-size: 12px;
  color: var(--ant-color-text-quaternary, #d1d5db);
  flex-shrink: 0;
}

/* ═══ Activity ═══ */
.activity-item {
  padding: 4px 0;
}
.activity-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.activity-agent {
  font-weight: 600;
  font-size: 13px;
}
.activity-time {
  font-size: 11px;
  color: var(--ant-color-text-tertiary, #9ca3af);
  margin-left: auto;
}
.activity-desc {
  font-size: 13px;
  color: var(--ant-color-text, #111827);
  margin: 4px 0;
}
.activity-meta {
  font-size: 12px;
  color: var(--ant-color-text-tertiary, #9ca3af);
}

/* ═══ Agent Detail Sheet ═══ */
.agent-detail-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
</style>
