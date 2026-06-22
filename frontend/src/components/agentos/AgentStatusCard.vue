<template>
  <n-card size="small" :bordered="true" hoverable>
    <template #header>
      <n-space align="center" justify="space-between">
        <n-space align="center" size="small">
          <n-avatar :size="28" round>{{ agent.name.charAt(0) }}</n-avatar>
          <div>
            <div style="font-weight: 600; font-size: 14px;">{{ agent.name }}</div>
            <div style="color: #888; font-size: 12px;">{{ agent.role }}</div>
          </div>
        </n-space>
        <AutonomyBadge :level="agent.autonomy_level" />
      </n-space>
    </template>

    <n-grid :cols="2" :x-gap="8" :y-gap="8">
      <n-grid-item>
        <div class="stat-label">状态</div>
        <n-tag :type="statusType" size="small" :bordered="false">{{ statusText }}</n-tag>
      </n-grid-item>
      <n-grid-item>
        <div class="stat-label">负载</div>
        <n-progress
          type="line"
          :percentage="Math.min(agent.current_workload * 10, 100)"
          :height="6"
          :indicator-placement="'inside'"
          :processing="agent.current_workload > 5"
          :color="workloadColor"
        />
      </n-grid-item>
      <n-grid-item>
        <div class="stat-label">成功率</div>
        <span :style="{ color: successColor, fontWeight: 600 }">{{ (agent.success_rate * 100).toFixed(0) }}%</span>
      </n-grid-item>
      <n-grid-item>
        <div class="stat-label">风险</div>
        <n-tag :type="riskType" size="small" :bordered="false">{{ riskLabel }}</n-tag>
      </n-grid-item>
    </n-grid>

    <template #footer>
      <div style="color: #999; font-size: 11px;">
        最后活动: {{ lastActive }}
      </div>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSAgent, AutonomyLevel, RiskLevel } from '@/api/modules/agentos'
import AutonomyBadge from './AutonomyBadge.vue'

const props = defineProps<{
  agent: AgentOSAgent
}>()

const statusType = computed(() => {
  const map: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
    active: 'success',
    idle: 'default',
    busy: 'warning',
    error: 'error',
  }
  return map[props.agent.status] || 'default'
})

const statusText = computed(() => {
  const map: Record<string, string> = {
    active: '工作中',
    idle: '空闲',
    busy: '忙碌',
    error: '异常',
  }
  return map[props.agent.status] || props.agent.status
})

const workloadColor = computed(() => {
  if (props.agent.current_workload > 8) return '#d03050'
  if (props.agent.current_workload > 5) return '#f0a020'
  return '#18a058'
})

const successColor = computed(() => {
  if (props.agent.success_rate >= 0.9) return '#18a058'
  if (props.agent.success_rate >= 0.7) return '#f0a020'
  return '#d03050'
})

const riskLabel = computed(() => {
  const map: Record<string, string> = {
    low: '低风险',
    medium: '中风险',
    high: '高风险',
    critical: '严重',
  }
  return map[props.agent.risk_level] || props.agent.risk_level
})

const riskType = computed(() => {
  const map: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
    low: 'success',
    medium: 'warning',
    high: 'error',
    critical: 'error',
  }
  return map[props.agent.risk_level] || 'default'
})

const lastActive = computed(() => {
  if (!props.agent.last_activity_at) return '未知'
  return new Date(props.agent.last_activity_at).toLocaleString('zh-CN')
})
</script>

<style scoped>
.stat-label {
  font-size: 11px;
  color: #888;
  margin-bottom: 2px;
}
</style>
