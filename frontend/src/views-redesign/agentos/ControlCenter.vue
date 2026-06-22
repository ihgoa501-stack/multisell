<!-- ================================================
   FLOW: AgentOS Control Center
   SCREEN 1 of 4: Main Console
   ------------------------------------------------
   ENTRY:  Sidebar → AgentOS 总控
   EXIT:   Click agent → Agent Detail Sheet / Click work item → Approval Sheet
   BRANCH: Autonomy management → Autonomy Drawer
   ================================================ -->
<template>
  <div class="agentos-page">
    <!-- ═══ Page Header ═══ -->
    <div class="page-header">
      <div>
        <h1 class="page-title">AgentOS 总控台</h1>
        <p class="page-subtitle">跨境电商 AI Agent 运营管理中枢</p>
      </div>
      <a-space>
        <a-button @click="refreshData" :loading="loading">
          <template #icon><ReloadOutlined /></template>
        </a-button>
        <a-button type="primary" @click="showAutonomyDrawer = true">
          <template #icon><SettingOutlined /></template>
          自治管理
        </a-button>
      </a-space>
    </div>

    <!-- ═══ Row 1: Overview Metrics ═══ -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card :bordered="false" class="metric-card">
          <div class="metric-center">
            <a-progress
              type="dashboard"
              :percent="overview.health_score"
              :size="80"
              :stroke-color="healthScoreColor"
            />
            <div class="metric-label">系统健康分</div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card :bordered="false" class="metric-card">
          <div class="metric-inline">
            <div class="metric-big" style="color: #2962FF">{{ overview.active_agents }}</div>
            <div class="metric-label">活跃 Agent</div>
            <div class="metric-sub">共 {{ agents.length }} 个已注册</div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card :bordered="false" class="metric-card" @click="scrollToQueue" style="cursor: pointer">
          <div class="metric-inline">
            <div class="metric-big" :style="{ color: overview.pending_approvals > 0 ? '#D97706' : '#059669' }">
              {{ overview.pending_approvals }}
            </div>
            <div class="metric-label">待审批</div>
            <div class="metric-sub">{{ overview.pending_approvals > 0 ? '需立即处理' : '无待审批项' }}</div>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :lg="6">
        <a-card :bordered="false" class="metric-card">
          <div class="metric-inline">
            <div class="metric-big" :style="{ color: overview.critical_items > 0 ? '#DC2626' : '#059669' }">
              {{ overview.critical_items }}
            </div>
            <div class="metric-label">高风险任务</div>
            <div class="metric-sub">{{ overview.critical_items > 0 ? '需紧急处理' : '一切正常' }}</div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ═══ Row 2: Agent Grid + Priority Queue ═══ -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <!-- Agent Grid -->
      <a-col :xs="24" :lg="16">
        <a-card :bordered="false" class="section-card">
          <template #title>
            <span>
              <RobotOutlined style="margin-right: 6px" />
              Agent 团队
            </span>
          </template>
          <template #extra>
            <a-radio-group v-model:value="agentView" size="small" button-style="solid">
              <a-radio-button value="grid">网格</a-radio-button>
              <a-radio-button value="squad">团队</a-radio-button>
            </a-radio-group>
          </template>

          <!-- Grid View -->
          <div v-if="agentView === 'grid'" class="agent-grid">
            <div
              v-for="agent in agents"
              :key="agent.id"
              class="agent-card"
              @click="openAgentDetail(agent)"
            >
              <div class="agent-card-header">
                <a-badge :status="agentBadge(agent.status)" :offset="[-2, 32]">
                  <a-avatar :size="44" :style="{ backgroundColor: agent.avatar_color, fontSize: '16px', fontWeight: 700 }">
                    {{ agent.name.charAt(0) }}
                  </a-avatar>
                </a-badge>
                <div class="agent-card-meta">
                  <div class="agent-card-name">{{ agent.name }}</div>
                  <div class="agent-card-code">{{ agent.code }}</div>
                </div>
              </div>
              <div class="agent-card-stats">
                <div class="agent-stat">
                  <span class="stat-value">{{ agent.trust_score }}</span>
                  <span class="stat-label">信任分</span>
                </div>
                <div class="agent-stat">
                  <span class="stat-value">{{ agent.success_rate }}%</span>
                  <span class="stat-label">成功率</span>
                </div>
                <div class="agent-stat">
                  <span class="stat-value">{{ agent.pending_approvals }}</span>
                  <span class="stat-label">待审批</span>
                </div>
              </div>
              <a-progress
                :percent="agent.health_score"
                :stroke-color="agent.health_score >= 80 ? '#059669' : '#D97706'"
                :show-info="false"
                size="small"
              />
            </div>
          </div>

          <!-- Squad View -->
          <div v-else class="squad-list">
            <a-card
              v-for="squad in squads"
              :key="squad.id"
              size="small"
              :bordered="true"
              class="squad-card"
            >
              <div class="squad-header">
                <div>
                  <span class="squad-name">{{ squad.name }}</span>
                  <a-tag :bordered="false" size="small">{{ squad.agents.length }} Agent</a-tag>
                </div>
                <div class="squad-badges">
                  <a-badge v-if="squad.pending_approvals" :count="squad.pending_approvals" type="warning" />
                  <a-progress
                    type="circle"
                    :percent="squad.health_score"
                    :size="32"
                    :stroke-color="squad.health_score >= 80 ? '#059669' : '#D97706'"
                  />
                </div>
              </div>
              <div class="squad-agents">
                <a-avatar-group :max-count="5" :size="28">
                  <a-avatar
                    v-for="a in squad.agents"
                    :key="a.id"
                    :style="{ backgroundColor: a.avatar_color, fontSize: '11px' }"
                  >
                    {{ a.name.charAt(0) }}
                  </a-avatar>
                </a-avatar-group>
              </div>
            </a-card>
          </div>
        </a-card>
      </a-col>

      <!-- Priority Queue -->
      <a-col :xs="24" :lg="8">
        <a-card :bordered="false" class="section-card" ref="queueRef">
          <template #title>
            <span>
              <FireOutlined style="color: #DC2626; margin-right: 6px" />
              优先处理
            </span>
          </template>
          <template #extra>
            <a-tag :bordered="false">{{ pendingWorkItems.length }} 项</a-tag>
          </template>
          <a-empty v-if="pendingWorkItems.length === 0" description="暂无待处理事项" />
          <div v-else class="workitem-list">
            <div
              v-for="item in pendingWorkItems"
              :key="item.id"
              class="workitem-card"
              @click="openWorkItemDetail(item)"
            >
              <div class="wi-header">
                <a-tag :color="riskColor(item.risk_level)" :bordered="false" size="small">
                  {{ riskLabel(item.risk_level) }}
                </a-tag>
                <span class="wi-agent">{{ item.agent_name }}</span>
              </div>
              <div class="wi-title">{{ item.title }}</div>
              <div class="wi-meta">
                <span>置信度 {{ item.confidence_score }}%</span>
                <span>{{ timeAgo(item.created_at) }}</span>
              </div>
              <div v-if="item.requires_approval" class="wi-approval-hint">
                <ExclamationCircleOutlined /> 需要人工审批
              </div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- ═══ Row 3: Recent Activity ═══ -->
    <a-card title="最近活动" :bordered="false" class="section-card" style="margin-top: 16px">
      <a-table
        :columns="activityColumns"
        :data-source="recentActivities"
        :pagination="false"
        size="middle"
        :show-header="false"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'agent'">
            <div style="display: flex; align-items: center; gap: 8px">
              <a-avatar :size="28" :style="{ backgroundColor: getAgentColor(record.agent_id), fontSize: '11px' }">
                {{ record.agent_name.charAt(0) }}
              </a-avatar>
              <div>
                <div style="font-weight: 500; font-size: 13px">{{ record.agent_name }}</div>
                <div style="font-size: 11px; color: var(--ant-color-text-tertiary)">{{ record.description }}</div>
              </div>
            </div>
          </template>
          <template v-if="column.key === 'status'">
            <a-tag :color="actionTagColor(record.status)" :bordered="false" size="small">
              {{ actionLabel(record.status) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'confidence'">
            <a-progress :percent="record.confidence" :size="'small'" :stroke-color="record.confidence >= 90 ? '#059669' : '#D97706'" style="width: 80px" />
          </template>
          <template v-if="column.key === 'time'">
            <span style="font-size: 12px; color: var(--ant-color-text-tertiary)">{{ timeAgo(record.created_at) }}</span>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- ═══ WorkItem Approval Sheet (Screen 2) ═══ -->
    <a-drawer
      v-model:open="showWorkItemSheet"
      :title="selectedWorkItem?.title || '任务详情'"
      placement="right"
      width="520"
    >
      <template v-if="selectedWorkItem">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="状态">
            <a-tag :color="wiStatusColor(selectedWorkItem.status)">{{ wiStatusLabel(selectedWorkItem.status) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="风险等级">
            <a-tag :color="riskColor(selectedWorkItem.risk_level)">{{ riskLabel(selectedWorkItem.risk_level) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="Agent">{{ selectedWorkItem.agent_name }}</a-descriptions-item>
          <a-descriptions-item label="团队">{{ selectedWorkItem.squad_name }}</a-descriptions-item>
          <a-descriptions-item label="置信度">
            <a-progress :percent="selectedWorkItem.confidence_score" size="small" :stroke-color="selectedWorkItem.confidence_score >= 90 ? '#059669' : '#D97706'" />
          </a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <h4>提议操作</h4>
        <a-alert
          :message="selectedWorkItem.proposed_action"
          :type="selectedWorkItem.risk_level === 'critical' ? 'warning' : 'info'"
          show-icon
          style="margin-bottom: 12px"
        />

        <h4>预期影响</h4>
        <p style="color: var(--ant-color-text-secondary)">{{ selectedWorkItem.expected_impact }}</p>

        <a-divider />

        <h4>详细描述</h4>
        <p style="color: var(--ant-color-text-secondary); font-size: 13px">{{ selectedWorkItem.description }}</p>

        <template v-if="selectedWorkItem.status === 'pending' && selectedWorkItem.requires_approval">
          <a-divider />
          <a-space style="width: 100%; justify-content: flex-end">
            <a-button danger @click="rejectWorkItem">
              <template #icon><CloseOutlined /></template>
              驳回
            </a-button>
            <a-button type="primary" @click="approveWorkItem">
              <template #icon><CheckOutlined /></template>
              批准执行
            </a-button>
          </a-space>
        </template>
      </template>
    </a-drawer>

    <!-- ═══ Agent Detail Sheet (Screen 3) ═══ -->
    <a-drawer
      v-model:open="showAgentSheet"
      :title="selectedAgent?.name || 'Agent 详情'"
      placement="right"
      width="480"
    >
      <template v-if="selectedAgent">
        <div class="agent-profile">
          <a-avatar :size="64" :style="{ backgroundColor: selectedAgent.avatar_color, fontSize: '24px', fontWeight: 700 }">
            {{ selectedAgent.name.charAt(0) }}
          </a-avatar>
          <div class="agent-profile-info">
            <h3 style="margin: 0">{{ selectedAgent.name }}</h3>
            <p style="color: var(--ant-color-text-secondary); margin: 4px 0 0; font-size: 13px">
              {{ selectedAgent.description }}
            </p>
            <a-space style="margin-top: 8px">
              <a-badge :status="agentBadge(selectedAgent.status)" :text="agentStatusLabel(selectedAgent.status)" />
              <a-tag :bordered="false">{{ selectedAgent.squad_name }}</a-tag>
            </a-space>
          </div>
        </div>

        <a-row :gutter="16" style="margin-top: 20px">
          <a-col :span="8">
            <a-statistic title="信任分" :value="selectedAgent.trust_score" suffix="/ 100" />
          </a-col>
          <a-col :span="8">
            <a-statistic title="成功率" :value="selectedAgent.success_rate" suffix="%" :precision="1" />
          </a-col>
          <a-col :span="8">
            <a-statistic title="累计动作" :value="selectedAgent.total_actions" />
          </a-col>
        </a-row>

        <a-divider />

        <h4>自治等级</h4>
        <a-steps :current="autonomyStepIndex(selectedAgent.autonomy_level)" size="small" style="margin: 16px 0">
          <a-step title="观察" description="只看不做" />
          <a-step title="建议" description="提出方案" />
          <a-step title="半自治" description="执行+审批" />
          <a-step title="全自治" description="完全自主" />
        </a-steps>

        <a-divider />

        <h4>最近决策</h4>
        <a-timeline>
          <a-timeline-item
            v-for="act in selectedAgentActions"
            :key="act.id"
            :color="act.status === 'executed' ? 'green' : act.status === 'rejected' ? 'red' : 'blue'"
          >
            <div style="font-size: 13px">{{ act.description }}</div>
            <div style="font-size: 11px; color: var(--ant-color-text-tertiary)">{{ timeAgo(act.created_at) }} · 置信度 {{ act.confidence }}%</div>
          </a-timeline-item>
          <a-timeline-item v-if="selectedAgentActions.length === 0" color="gray">
            <span style="color: var(--ant-color-text-tertiary)">暂无决策记录</span>
          </a-timeline-item>
        </a-timeline>
      </template>
    </a-drawer>

    <!-- ═══ Autonomy Management Drawer (Screen 4) ═══ -->
    <a-drawer
      v-model:open="showAutonomyDrawer"
      title="自治等级管理"
      placement="right"
      width="560"
    >
      <p style="color: var(--ant-color-text-secondary); margin-bottom: 20px">
        管理每个 Agent 的自治权限等级。更高的自治等级意味着 Agent 可以自主执行更多操作，无需人工审批。
      </p>

      <div v-for="agent in agents" :key="agent.id" class="autonomy-item">
        <div class="autonomy-agent">
          <a-avatar :size="32" :style="{ backgroundColor: agent.avatar_color, fontSize: '13px' }">
            {{ agent.name.charAt(0) }}
          </a-avatar>
          <div>
            <div style="font-weight: 500; font-size: 13px">{{ agent.name }}</div>
            <div style="font-size: 11px; color: var(--ant-color-text-tertiary)">信任分 {{ agent.trust_score }}</div>
          </div>
        </div>
        <a-steps
          :current="autonomyStepIndex(agent.autonomy_level)"
          size="small"
          style="flex: 1; margin-left: 16px"
          @change="(step: number) => handleAutonomyChange(agent, step)"
        >
          <a-step title="观察" />
          <a-step title="建议" />
          <a-step title="半自治" />
          <a-step title="全自治" />
        </a-steps>
      </div>

      <a-divider />

      <a-alert
        message="自治等级调整说明"
        description="提升自治等级需要信任分达到对应阈值：建议模式 ≥ 70，半自治 ≥ 80，全自治 ≥ 95。降低等级立即生效。"
        type="info"
        show-icon
      />
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
/* ================================================
   FLOW: AgentOS Control Center
   SCREEN 1-4 of 4: Console + Approval + Agent Detail + Autonomy
   ================================================ */
import { ref, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  ReloadOutlined, SettingOutlined, RobotOutlined, FireOutlined,
  ExclamationCircleOutlined, CheckOutlined, CloseOutlined,
} from '@ant-design/icons-vue'
import type { Agent, AgentSquad, WorkItem, AgentAction } from '@/views-redesign/shared/types'
import {
  mockAgents, mockSquads, mockWorkItems, mockAgentActions,
} from '@/views-redesign/shared/mock-data'

// ── State ──
const loading = ref(false)
const agentView = ref<'grid' | 'squad'>('grid')
const queueRef = ref()

const agents = ref<Agent[]>(mockAgents)
const squads = ref<AgentSquad[]>(mockSquads)
const workItems = ref<WorkItem[]>([...mockWorkItems])
const recentActivities = ref<AgentAction[]>(mockAgentActions)

const overview = ref({
  health_score: 89,
  active_agents: mockAgents.filter(a => a.status === 'online' || a.status === 'thinking').length,
  pending_approvals: mockWorkItems.filter(w => w.status === 'pending' && w.requires_approval).length,
  critical_items: mockWorkItems.filter(w => w.risk_level === 'critical').length,
})

const pendingWorkItems = computed(() => workItems.value.filter(w => w.status === 'pending'))

// Sheets
const showWorkItemSheet = ref(false)
const selectedWorkItem = ref<WorkItem | null>(null)
const showAgentSheet = ref(false)
const selectedAgent = ref<Agent | null>(null)
const showAutonomyDrawer = ref(false)

const selectedAgentActions = computed(() => {
  if (!selectedAgent.value) return []
  return mockAgentActions.filter(a => a.agent_id === selectedAgent.value!.id)
})

// Activity table columns
const activityColumns = [
  { key: 'agent', dataIndex: 'agent_name' },
  { key: 'status', dataIndex: 'status', width: 100 },
  { key: 'confidence', dataIndex: 'confidence', width: 120 },
  { key: 'time', dataIndex: 'created_at', width: 100 },
]

// ── Computed ──
const healthScoreColor = computed(() => {
  if (overview.value.health_score >= 80) return '#059669'
  if (overview.value.health_score >= 60) return '#D97706'
  return '#DC2626'
})

// ── Helpers ──
function agentBadge(status: string): 'success' | 'warning' | 'default' | 'processing' {
  const map: Record<string, 'success' | 'warning' | 'default' | 'processing'> = {
    online: 'success', idle: 'warning', offline: 'default', thinking: 'processing',
  }
  return map[status] || 'default'
}

function agentStatusLabel(status: string): string {
  const map: Record<string, string> = { online: '在线', idle: '空闲', offline: '离线', thinking: '思考中' }
  return map[status] || status
}

function riskColor(level: string): string {
  const map: Record<string, string> = { critical: 'red', high: 'orange', medium: 'blue', low: 'default' }
  return map[level] || 'default'
}

function riskLabel(level: string): string {
  const map: Record<string, string> = { critical: '紧急', high: '重要', medium: '一般', low: '低' }
  return map[level] || level
}

function wiStatusColor(status: string): string {
  const map: Record<string, string> = { pending: 'orange', approved: 'blue', executed: 'green', rejected: 'red', failed: 'red' }
  return map[status] || 'default'
}

function wiStatusLabel(status: string): string {
  const map: Record<string, string> = { pending: '待审批', approved: '已批准', executed: '已执行', rejected: '已驳回', failed: '失败' }
  return map[status] || status
}

function actionTagColor(status: string): string {
  const map: Record<string, string> = { proposed: 'processing', approved: 'blue', executed: 'success', rejected: 'error' }
  return map[status] || 'default'
}

function actionLabel(status: string): string {
  const map: Record<string, string> = { proposed: '提议中', approved: '已批准', executed: '已执行', rejected: '已驳回' }
  return map[status] || status
}

function getAgentColor(agentId: string): string {
  return agents.value.find(a => a.id === agentId)?.avatar_color || '#999'
}

function timeAgo(t: string): string {
  if (!t) return ''
  const s = Math.floor((Date.now() - new Date(t).getTime()) / 1000)
  if (s < 60) return '刚刚'
  if (s < 3600) return `${Math.floor(s / 60)}分钟前`
  if (s < 86400) return `${Math.floor(s / 3600)}小时前`
  return `${Math.floor(s / 86400)}天前`
}

function autonomyStepIndex(level: string): number {
  const map: Record<string, number> = { observation: 0, suggestion: 1, semi_autonomous: 2, full_autonomous: 3 }
  return map[level] || 0
}

function autonomyLabel(level: string): string {
  const map: Record<string, string> = {
    observation: '观察模式', suggestion: '建议模式',
    semi_autonomous: '半自治模式', full_autonomous: '全自治模式',
  }
  return map[level] || level
}

const autonomyLevels = ['observation', 'suggestion', 'semi_autonomous', 'full_autonomous'] as const

function handleAutonomyChange(agent: Agent, step: number) {
  const minTrust = [0, 70, 80, 95]
  if (agent.trust_score < minTrust[step]) {
    message.warning(`${agent.name} 信任分 ${agent.trust_score} 不满足${autonomyLabel(autonomyLevels[step])}的最低要求 (${minTrust[step]})`)
    return
  }
  Modal.confirm({
    title: '确认调整自治等级',
    content: `将「${agent.name}」的自治等级调整为「${autonomyLabel(autonomyLevels[step])}」，确定吗？`,
    okText: '确认调整',
    cancelText: '取消',
    onOk() {
      agent.autonomy_level = autonomyLevels[step]
      message.success(`${agent.name} 自治等级已更新为${autonomyLabel(autonomyLevels[step])}`)
    },
  })
}

// ── Actions ──
function openAgentDetail(agent: Agent) {
  selectedAgent.value = agent
  showAgentSheet.value = true
}

function openWorkItemDetail(item: WorkItem) {
  selectedWorkItem.value = item
  showWorkItemSheet.value = true
}

function approveWorkItem() {
  if (!selectedWorkItem.value) return
  Modal.confirm({
    title: '确认批准执行',
    content: `批准「${selectedWorkItem.value.title}」后，Agent 将自动执行该操作。确定批准吗？`,
    okText: '确认批准',
    okType: 'primary',
    cancelText: '取消',
    onOk() {
      selectedWorkItem.value!.status = 'approved'
      overview.value.pending_approvals = Math.max(0, overview.value.pending_approvals - 1)
      message.success(`已批准: ${selectedWorkItem.value!.title}`)
      showWorkItemSheet.value = false
    },
  })
}

function rejectWorkItem() {
  if (!selectedWorkItem.value) return
  Modal.confirm({
    title: '确认驳回',
    content: `驳回「${selectedWorkItem.value.title}」后，Agent 将收到拒绝反馈。确定驳回吗？`,
    okText: '确认驳回',
    okType: 'danger' as any,
    cancelText: '取消',
    onOk() {
      selectedWorkItem.value!.status = 'rejected'
      overview.value.pending_approvals = Math.max(0, overview.value.pending_approvals - 1)
      message.info(`已驳回: ${selectedWorkItem.value!.title}`)
      showWorkItemSheet.value = false
    },
  })
}

function scrollToQueue() {
  queueRef.value?.$el?.scrollIntoView({ behavior: 'smooth' })
}

async function refreshData() {
  loading.value = true
  await new Promise(r => setTimeout(r, 800))
  loading.value = false
  message.success('数据已刷新')
}
</script>

<style scoped>
.agentos-page {
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
}
.page-subtitle {
  font-size: 14px;
  color: var(--ant-color-text-secondary);
  margin: 4px 0 0;
}

/* ═══ Metric Cards ═══ */
.metric-card {
  border-radius: 12px;
  text-align: center;
}
.metric-card :deep(.ant-card-body) {
  padding: 20px;
}
.metric-center {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.metric-inline {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}
.metric-big {
  font-size: 32px;
  font-weight: 700;
  line-height: 1.2;
}
.metric-label {
  font-size: 13px;
  color: var(--ant-color-text-secondary);
  margin-top: 8px;
}
.metric-sub {
  font-size: 12px;
  color: var(--ant-color-text-tertiary);
}

.section-card {
  border-radius: 12px;
  height: 100%;
}

/* ═══ Agent Grid ═══ */
.agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}
.agent-card {
  padding: 16px;
  border: 1px solid var(--ant-color-border-secondary, #f3f4f6);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}
.agent-card:hover {
  border-color: var(--ant-color-primary-border, #91b5ff);
  box-shadow: 0 2px 8px rgba(41, 98, 255, 0.08);
}
.agent-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.agent-card-name {
  font-weight: 600;
  font-size: 14px;
}
.agent-card-code {
  font-size: 11px;
  color: var(--ant-color-text-tertiary);
}
.agent-card-stats {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}
.agent-stat {
  text-align: center;
}
.stat-value {
  font-size: 14px;
  font-weight: 600;
  display: block;
}
.stat-label {
  font-size: 10px;
  color: var(--ant-color-text-tertiary);
}

/* ═══ Squad View ═══ */
.squad-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.squad-card {
  border-radius: 10px;
}
.squad-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.squad-name {
  font-weight: 600;
  font-size: 14px;
  margin-right: 8px;
}
.squad-badges {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ═══ WorkItem List ═══ */
.workitem-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.workitem-card {
  padding: 14px;
  border: 1px solid var(--ant-color-border-secondary, #f3f4f6);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}
.workitem-card:hover {
  border-color: var(--ant-color-primary-border, #91b5ff);
  background: var(--ant-color-bg-text-hover, #fafafa);
}
.wi-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.wi-agent {
  font-size: 11px;
  color: var(--ant-color-text-tertiary);
}
.wi-title {
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 6px;
  line-height: 1.4;
}
.wi-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--ant-color-text-tertiary);
}
.wi-approval-hint {
  margin-top: 8px;
  font-size: 11px;
  color: #D97706;
  display: flex;
  align-items: center;
  gap: 4px;
}

/* ═══ Autonomy Management ═══ */
.autonomy-item {
  display: flex;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--ant-color-border-secondary, #f3f4f6);
}
.autonomy-agent {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 140px;
}

/* ═══ Agent Profile ═══ */
.agent-profile {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
</style>
