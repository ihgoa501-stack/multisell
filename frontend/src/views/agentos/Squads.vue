<template>
  <div>
    <n-page-header subtitle="按增长、履约、风控管理 AI 运营团队">
      <template #title>Agent 团队</template>
      <template #extra>
        <n-button size="small" @click="fetchSquads" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <!-- 加载状态 -->
    <n-spin v-if="loading && !loaded" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <!-- 错误状态 -->
    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchSquads">重试</n-button>
      </template>
    </n-result>

    <template v-else>
      <!-- Squad 卡片网格 -->
      <n-grid :cols="3" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
        <n-grid-item v-for="squad in squads" :key="squad.id">
          <n-card :title="squad.name" size="small" hoverable>
            <template #header-extra>
              <n-space size="small" align="center">
                <n-tag :type="squadRiskType(squad.risk_level)" size="tiny" :bordered="false">
                  {{ squadRiskLabel(squad.risk_level) }}
                </n-tag>
                <AutonomyBadge :level="squad.autonomy_level" />
              </n-space>
            </template>

            <p style="color: #666; font-size: 13px; margin: 0 0 12px;">{{ squad.description }}</p>

            <!-- 统计 -->
            <n-grid :cols="3" :x-gap="8" :y-gap="8" style="margin-bottom: 12px;">
              <n-grid-item>
                <div class="stat-value">{{ squad.active_work_items }}</div>
                <div class="stat-label">活跃任务</div>
              </n-grid-item>
              <n-grid-item>
                <div class="stat-value" :style="{ color: squad.pending_approvals > 0 ? '#f0a020' : '#18a058' }">
                  {{ squad.pending_approvals }}
                </div>
                <div class="stat-label">待审批</div>
              </n-grid-item>
              <n-grid-item>
                <div class="stat-value">{{ squad.agents.length }}</div>
                <div class="stat-label">Agent</div>
              </n-grid-item>
            </n-grid>

            <!-- 健康分 -->
            <div style="margin-bottom: 12px;">
              <div style="display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 4px;">
                <span style="color: #888;">健康分</span>
                <span :style="{ color: healthColor(squad.health_score), fontWeight: 600 }">{{ squad.health_score }}</span>
              </div>
              <n-progress
                type="line"
                :percentage="squad.health_score"
                :height="4"
                :color="healthColor(squad.health_score)"
                :rail-color="'#e5e5e5'"
                :indicator-placement="'inside'"
              />
            </div>

            <!-- Agent 列表 -->
            <div style="font-size: 13px; font-weight: 600; margin-bottom: 8px; color: #555;">团队成员</div>
            <n-space vertical size="small">
              <AgentStatusCard
                v-for="agent in squad.agents"
                :key="agent.id"
                :agent="agent"
                style="cursor: pointer;"
                @click="router.push(`/agentos/agents/${agent.id}`)"
              />
            </n-space>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 升级建议 -->
      <template v-if="upgradeCandidates.length > 0">
        <n-card title="自治等级升级建议" size="small" style="margin-top: 12px;">
          <n-space>
            <n-tag
              v-for="c in upgradeCandidates"
              :key="c.agent_id"
              :type="c.direction === 'upgrade' ? 'success' : 'warning'"
              style="cursor: pointer;"
              @click="router.push('/agentos/autonomy')"
            >
              {{ c.agent_name }} → {{ c.target_level }}
            </n-tag>
          </n-space>
        </n-card>
      </template>

      <!-- 自治等级说明 -->
      <n-card title="自治等级说明" size="small" style="margin-top: 12px;">
        <n-grid :cols="4" :x-gap="12" :y-gap="8">
          <n-grid-item>
            <n-alert type="default" :bordered="false">
              <template #header>L0 观察</template>
              只读数据，生成报告，不产生建议
            </n-alert>
          </n-grid-item>
          <n-grid-item>
            <n-alert type="info" :bordered="false">
              <template #header>L1 建议</template>
              生成建议，用户手动执行
            </n-alert>
          </n-grid-item>
          <n-grid-item>
            <n-alert type="success" :bordered="false">
              <template #header>L2 半自主</template>
              低风险自动执行，高风险需审批
            </n-alert>
          </n-grid-item>
          <n-grid-item>
            <n-alert type="warning" :bordered="false">
              <template #header>L3 全自主</template>
              预算和权限边界内的自动执行
            </n-alert>
          </n-grid-item>
        </n-grid>
      </n-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAgentOSSquads, getAgentOSUpgradeCandidates } from '@/api/modules/agentos'
import type { AgentOSSquad, SquadsResponse, AutonomyCandidate } from '@/api/modules/agentos'
import AutonomyBadge from '@/components/agentos/AutonomyBadge.vue'
import AgentStatusCard from '@/components/agentos/AgentStatusCard.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)
const squads = ref<AgentOSSquad[]>([])
const upgradeCandidates = ref<AutonomyCandidate[]>([])

function squadRiskType(level: string): 'success' | 'warning' | 'error' | 'default' {
  const map: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
    low: 'success',
    medium: 'warning',
    high: 'error',
    critical: 'error',
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
  if (score >= 80) return '#18a058'
  if (score >= 60) return '#f0a020'
  return '#d03050'
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
.stat-value {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 11px;
  color: #888;
  margin-top: 2px;
}
</style>
