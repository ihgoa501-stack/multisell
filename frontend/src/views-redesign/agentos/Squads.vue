<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h2 class="page-header-title">Agent 团队</h2>
        <p class="page-header-subtitle">按增长、履约、风控管理 AI 运营团队</p>
      </div>
      <a-space>
        <a-button size="small" @click="fetchSquads" :loading="loading">刷新</a-button>
      </a-space>
    </div>

    <!-- 加载状态 -->
    <a-spin v-if="loading && !loaded" style="margin-top: 40px; display: block;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </a-spin>

    <!-- 错误状态 -->
    <a-result v-else-if="error" status="error" title="加载失败" :sub-title="error">
      <template #extra>
        <a-button @click="fetchSquads">重试</a-button>
      </template>
    </a-result>

    <template v-else>
      <!-- Squad 卡片网格 -->
      <a-row :gutter="[12, 12]" style="margin-top: 12px;">
        <a-col :span="8" v-for="squad in squads" :key="squad.id">
          <a-card :title="squad.name" size="small" hoverable>
            <template #extra>
              <a-space size="small" align="center">
                <a-tag :color="squadRiskColor(squad.risk_level)">
                  {{ squadRiskLabel(squad.risk_level) }}
                </a-tag>
                <AutonomyBadge :level="squad.autonomy_level" />
              </a-space>
            </template>

            <p style="color: rgba(0,0,0,0.45); font-size: 13px; margin: 0 0 12px;">{{ squad.description }}</p>

            <!-- 统计 -->
            <a-row :gutter="8" style="margin-bottom: 12px;">
              <a-col :span="8">
                <div class="stat-value">{{ squad.active_work_items }}</div>
                <div class="stat-label">活跃任务</div>
              </a-col>
              <a-col :span="8">
                <div class="stat-value" :style="{ color: squad.pending_approvals > 0 ? 'var(--ant-color-warning)' : 'var(--ant-color-success)' }">
                  {{ squad.pending_approvals }}
                </div>
                <div class="stat-label">待审批</div>
              </a-col>
              <a-col :span="8">
                <div class="stat-value">{{ squad.agents.length }}</div>
                <div class="stat-label">Agent</div>
              </a-col>
            </a-row>

            <!-- 健康分 -->
            <div style="margin-bottom: 12px;">
              <div style="display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 4px;">
                <span style="color: rgba(0,0,0,0.45);">健康分</span>
                <span :style="{ color: healthColor(squad.health_score), fontWeight: 600 }">{{ squad.health_score }}</span>
              </div>
              <a-progress
                :percent="squad.health_score"
                :stroke-width="4"
                :stroke-color="healthColor(squad.health_score)"
                trail-color="#e5e5e5"
                :show-info="false"
              />
            </div>

            <!-- Agent 列表 -->
            <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px; color: rgba(0,0,0,0.65);">团队成员</div>
            <a-space direction="vertical" size="small" style="width: 100%;">
              <AgentStatusCard
                v-for="agent in squad.agents"
                :key="agent.id"
                :agent="agent"
                style="cursor: pointer;"
                @click="router.push(`/agentos/agents/${agent.id}`)"
              />
            </a-space>
          </a-card>
        </a-col>
      </a-row>

      <!-- 升级建议 -->
      <template v-if="upgradeCandidates.length > 0">
        <a-card title="自治等级升级建议" size="small" style="margin-top: 12px;">
          <a-space>
            <a-tag
              v-for="c in upgradeCandidates"
              :key="c.agent_id"
              :color="c.direction === 'upgrade' ? 'var(--ant-color-success)' : 'var(--ant-color-warning)'"
              style="cursor: pointer;"
              @click="router.push('/agentos/autonomy')"
            >
              {{ c.agent_name }} → {{ c.target_level }}
            </a-tag>
          </a-space>
        </a-card>
      </template>

      <!-- 自治等级说明 -->
      <a-card title="自治等级说明" size="small" style="margin-top: 12px;">
        <a-row :gutter="[12, 8]">
          <a-col :span="6">
            <a-alert type="info" :bordered="false">
              <template #message>L0 观察</template>
              <template #description>只读数据，生成报告，不产生建议</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="info" :bordered="false">
              <template #message>L1 建议</template>
              <template #description>生成建议，用户手动执行</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="success" :bordered="false">
              <template #message>L2 半自主</template>
              <template #description>低风险自动执行，高风险需审批</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="warning" :bordered="false">
              <template #message>L3 全自主</template>
              <template #description>预算和权限边界内的自动执行</template>
            </a-alert>
          </a-col>
        </a-row>
      </a-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { getAgentOSSquads, getAgentOSUpgradeCandidates } from '@/api/modules/agentos'
import type { AgentOSSquad, SquadsResponse, AutonomyCandidate } from '@/api/modules/agentos'
import AutonomyBadge from '@/components/agentos/AutonomyBadge.vue'
import AgentStatusCard from '@/components/agentos/AgentStatusCard.vue'

const router = useRouter()

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)
const squads = ref<AgentOSSquad[]>([])
const upgradeCandidates = ref<AutonomyCandidate[]>([])

function squadRiskColor(level: string): string {
  const map: Record<string, string> = {
    low: 'var(--ant-color-success)',
    medium: 'var(--ant-color-warning)',
    high: 'var(--ant-color-error)',
    critical: 'var(--ant-color-error)',
  }
  return map[level] || 'default'
}

function squadRiskLabel(level: string): string {
  const map: Record<string, string> = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '严重',
  }
  return map[level] || level
}

function healthColor(score: number): string {
  if (score >= 80) return 'var(--ant-color-success)'
  if (score >= 60) return 'var(--ant-color-warning)'
  return 'var(--ant-color-error)'
}

async function fetchSquads() {
  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSSquads()
    const data: SquadsResponse = res.data || res
    squads.value = data.squads || []
    loaded.value = true
    try {
      const candRes: any = await getAgentOSUpgradeCandidates()
      upgradeCandidates.value = (candRes?.data || []).filter((c: AutonomyCandidate) => c.suggested)
    } catch (_e) {
      // 静默降级
    }
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载 Agent 团队失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchSquads)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.page-header-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  margin: 4px 0 0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 14px;
}
.stat-value {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 11px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 2px;
}
</style>
