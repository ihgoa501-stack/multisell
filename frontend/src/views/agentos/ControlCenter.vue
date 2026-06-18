<template>
  <div>
    <!-- 页面标题 -->
    <n-page-header subtitle="跨境电商 AI AgentOS 运营总控台">
      <template #title>AgentOS 总控台</template>
      <template #extra>
        <n-button size="small" @click="fetchData" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <!-- 加载状态 -->
    <n-spin v-if="loading && !loaded" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <!-- 错误状态 -->
    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchData">重试</n-button>
      </template>
    </n-result>

    <!-- 主内容 -->
    <template v-else>
      <!-- Row 1: 概览指标 -->
      <n-grid :cols="4" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
        <n-grid-item>
          <n-card size="small">
            <div class="metric">{{ overview.health_score }}</div>
            <div class="label">系统健康分</div>
            <n-progress
              type="line"
                  :percentage="overview.health_score"
              :height="4"
              :color="healthColor"
              :rail-color="'#e5e5e5'"
              :indicator-placement="'inside'"
            />
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="metric">{{ overview.active_agents }}</div>
            <div class="label">活跃 Agent</div>
            <div style="font-size: 12px; color: #888;">共 {{ overview.active_agents }} 个 Agent 在运行</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="metric" :style="{ color: overview.pending_approvals > 0 ? '#f0a020' : '#18a058' }">
              {{ overview.pending_approvals }}
            </div>
            <div class="label">待审批</div>
            <div style="font-size: 12px; color: #888;">
              {{ overview.pending_approvals > 0 ? '需立即处理' : '无待审批项' }}
            </div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="metric" :style="{ color: overview.critical_items > 0 ? '#d03050' : '#18a058' }">
              {{ overview.critical_items }}
            </div>
            <div class="label">高风险任务</div>
            <div style="font-size: 12px; color: #888;">
              {{ overview.critical_items > 0 ? '需紧急处理' : '一切正常' }}
            </div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 第二行：优先任务 + 团队状态 + 指标 -->
      <n-grid :cols="3" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
        <!-- 左栏：优先处理队列 -->
        <n-grid-item :span="1">
          <n-card title="优先处理" size="small" :segmented="{ content: true }">
            <template v-if="priority_work_items.length === 0">
              <n-empty description="暂无优先任务" style="padding: 20px 0;" />
            </template>
            <WorkItemCard
              v-for="item in priority_work_items"
              :key="item.id"
              :item="item"
              @inspect="navigateToItem"
              @approve="navigateToItem"
              @status-updated="fetchData"
            />
          </n-card>
        </n-grid-item>

        <!-- 中栏：Agent 团队状态 -->
        <n-grid-item :span="1">
          <n-card title="Agent 团队" size="small" :segmented="{ content: true }">
            <template v-if="squads.length === 0">
              <n-empty description="暂无团队数据" style="padding: 20px 0;" />
            </template>
            <n-space vertical size="small">
              <n-card
                v-for="squad in squads"
                :key="squad.id"
                size="small"
                :bordered="true"
                hoverable
                @click="router.push('/agentos/squads')"
                style="cursor: pointer;"
              >
                <n-space align="center" justify="space-between">
                  <div>
                    <div style="font-weight: 600; font-size: 14px;">{{ squad.name }}</div>
                    <div style="color: #888; font-size: 12px;">{{ squad.agents.length }} 个 Agent</div>
                  </div>
                  <n-space size="small">
                    <n-badge :value="squad.active_work_items" :max="99">
                      <n-tag size="small" :bordered="false">任务 {{ squad.active_work_items }}</n-tag>
                    </n-badge>
                    <n-badge v-if="squad.pending_approvals > 0" :value="squad.pending_approvals" type="warning" :max="99">
                      <n-tag size="small" type="warning" :bordered="false">审批</n-tag>
                    </n-badge>
                  </n-space>
                </n-space>
                <div style="margin-top: 6px;">
                  <n-progress
                    type="line"
                    :percentage="squad.health_score"
                    :height="4"
                    :color="squadHealthColor(squad.health_score)"
                    :rail-color="'#e5e5e5'"
                    :indicator-placement="'inside'"
                  />
                  <div style="font-size: 11px; color: #999; margin-top: 2px;">健康分 {{ squad.health_score }}</div>
                </div>
              </n-card>
            </n-space>
          </n-card>
        </n-grid-item>

        <!-- 右栏：指标和风险摘要 -->
        <n-grid-item :span="1">
          <n-card title="业务指标" size="small" :segmented="{ content: true }">
            <n-grid :cols="1" :y-gap="8">
              <n-grid-item v-for="metric in metrics" :key="metric.key">
                <n-space align="center" justify="space-between">
                  <span style="font-size: 13px; color: #555;">{{ metric.label }}</span>
                  <span style="font-weight: 600; font-size: 14px;">
                    {{ metric.value }}{{ metric.unit }}
                  </span>
                </n-space>
              </n-grid-item>
            </n-grid>
          </n-card>

          <!-- 风险摘要 -->
          <n-card title="风险摘要" size="small" style="margin-top: 12px;" :segmented="{ content: true }">
            <n-empty v-if="priority_work_items.length === 0" description="当前无风险" style="padding: 20px 0;" />
            <n-space vertical size="small">
              <n-alert
                v-for="item in priorityRiskItems"
                :key="item.id"
                :type="riskAlertType(item.risk_level)"
                :title="item.title"
                closable
              >
                <template #default>
                  <div style="font-size: 12px;">
                    {{ item.squad_name || item.squad_id }} · {{ item.agent_name || item.agent_id || '-' }}
                  </div>
                </template>
              </n-alert>
            </n-space>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 第三行：最近活动 -->
      <n-card title="最近活动" size="small" style="margin-top: 12px;">
        <template v-if="recent_activity.length === 0">
          <n-empty description="暂无最近活动" style="padding: 20px 0;" />
        </template>
        <n-list v-else>
          <n-list-item v-for="item in recent_activity" :key="item.id">
            <n-space align="center" justify="space-between">
              <n-space align="center" size="small">
                <n-tag :type="activityType(item.source_type)" size="tiny" :bordered="false">
                  {{ activityLabel(item.source_type) }}
                </n-tag>
                <span style="font-size: 13px;">{{ item.title }}</span>
              </n-space>
              <n-space align="center" size="small">
                <n-tag v-if="item.requires_approval" size="tiny" type="warning" :bordered="false">待审批</n-tag>
                <span style="color: #999; font-size: 11px;">{{ formatTime(item.created_at) }}</span>
              </n-space>
            </n-space>
          </n-list-item>
        </n-list>
      </n-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAgentOSControlCenter } from '@/api/modules/agentos'
import type { AgentOSOverview, AgentOSSquad, AgentOSWorkItem, AgentOSMetric, ControlCenterResponse } from '@/api/modules/agentos'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)

const overview = ref<AgentOSOverview>({
  health_score: 0,
  active_agents: 0,
  pending_approvals: 0,
  critical_items: 0,
})
const squads = ref<AgentOSSquad[]>([])
const priority_work_items = ref<AgentOSWorkItem[]>([])
const metrics = ref<AgentOSMetric[]>([])
const recent_activity = ref<AgentOSWorkItem[]>([])

const priorityRiskItems = computed(() =>
  priority_work_items.value.filter(i => i.risk_level === 'critical' || i.risk_level === 'high').slice(0, 5)
)

const healthColor = computed(() => {
  if (overview.value.health_score >= 80) return '#18a058'
  if (overview.value.health_score >= 60) return '#f0a020'
  return '#d03050'
})

function squadHealthColor(score: number): string {
  if (score >= 80) return '#18a058'
  if (score >= 60) return '#f0a020'
  return '#d03050'
}

function riskAlertType(level: string): 'error' | 'warning' | 'info' {
  if (level === 'critical') return 'error'
  if (level === 'high') return 'warning'
  return 'info'
}

function activityType(sourceType: string): 'success' | 'info' | 'warning' | 'error' {
  const map: Record<string, 'success' | 'info' | 'warning' | 'error'> = {
    agent_action: 'info',
    exception: 'error',
    notification: 'warning',
    listing_task: 'success',
  }
  return map[sourceType] || 'default' as any
}

function activityLabel(sourceType: string): string {
  const map: Record<string, string> = {
    agent_action: 'Agent',
    exception: '异常',
    notification: '通知',
    listing_task: '上架',
  }
  return map[sourceType] || sourceType
}

function formatTime(val: string | null): string {
  if (!val) return ''
  return new Date(val).toLocaleString('zh-CN')
}

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSControlCenter()
    const data: ControlCenterResponse = res.data || res
    overview.value = data.overview || overview.value
    squads.value = data.squads || []
    priority_work_items.value = data.priority_work_items || []
    metrics.value = data.metrics || []
    recent_activity.value = data.recent_activity || []
    loaded.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载 AgentOS 总控台失败')
  } finally {
    loading.value = false
  }
}

function navigateToItem(item: AgentOSWorkItem) {
  if (item.action_url) {
    router.push(item.action_url)
  }
}

onMounted(fetchData)
</script>

<style scoped>
.metric {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
}
.label {
  color: #888;
  font-size: 12px;
  margin-top: 2px;
  margin-bottom: 6px;
}
</style>
