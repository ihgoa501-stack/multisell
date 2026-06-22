<template>
  <n-card size="small">
    <template #header>
      <n-space justify="space-between" align="center">
        <strong>{{ squad.name }}</strong>
        <n-tag :type="autonomyType" size="small">{{ autonomyLabel }}</n-tag>
      </n-space>
    </template>
    <n-space vertical size="small">
      <div class="muted">{{ squad.description }}</div>
      <n-space>
        <n-tag v-for="agent in squad.agents" :key="agent" size="small">{{ agent }}</n-tag>
      </n-space>
      <n-grid :cols="3" :x-gap="8">
        <n-grid-item><div class="metric">{{ squad.decision_count_7d }}</div><div class="label">7天决策</div></n-grid-item>
        <n-grid-item><div class="metric">{{ Math.round(squad.adoption_rate * 100) }}%</div><div class="label">采纳率</div></n-grid-item>
        <n-grid-item><div class="metric">{{ squad.pending_approvals }}</div><div class="label">待审批</div></n-grid-item>
      </n-grid>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSSquad } from '@/api/modules/agentos'

const props = defineProps<{ squad: AgentOSSquad }>()

const autonomyType = computed(() => props.squad.autonomy_level === 'semi_autonomous' ? 'success' : 'info')
const autonomyLabel = computed(() => {
  const map: Record<string, string> = {
    observation: '观察',
    suggestion: '建议',
    semi_autonomous: '半自主',
    full_autonomous: '全自主',
  }
  return map[props.squad.autonomy_level] || props.squad.autonomy_level
})
</script>

<style scoped>
.muted { color: #666; font-size: 13px; }
.metric { font-weight: 700; font-size: 18px; }
.label { color: #999; font-size: 12px; }
</style>
