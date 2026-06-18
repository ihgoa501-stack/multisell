<template>
  <n-card size="small" hoverable>
    <template #header>
      <n-space align="center" justify="space-between">
        <n-space align="center" size="small">
          <n-avatar :size="24" round>{{ candidate.agent_name.charAt(0) }}</n-avatar>
          <div>
            <div style="font-weight: 600; font-size: 14px;">{{ candidate.agent_name }}</div>
            <div style="color: #888; font-size: 12px;">{{ candidate.squad_name }}</div>
          </div>
        </n-space>
        <n-tag :type="directionType" size="small" :bordered="false">
          {{ directionLabel }}
        </n-tag>
      </n-space>
    </template>

    <n-space vertical size="small">
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">当前等级</span>
        <n-tag size="small">{{ currentLevelLabel }}</n-tag>
      </n-space>
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">目标等级</span>
        <n-tag size="small" :type="directionType">{{ targetLevelLabel }}</n-tag>
      </n-space>
      <n-space justify="space-between">
        <span style="font-size: 13px; color: #666;">置信度</span>
        <span style="font-weight: 600;">{{ (candidate.confidence * 100).toFixed(0) }}%</span>
      </n-space>
      <div style="font-size: 12px; color: #999; padding: 4px 0;">
        {{ candidate.reason || '-' }}
      </div>
    </n-space>

    <template #footer>
      <n-space justify="space-between">
        <n-button
          v-if="candidate.direction === 'upgrade'"
          size="tiny"
          type="primary"
          :loading="actioning && candidate.agent_id === actioning"
          @click="$emit('upgrade', candidate)"
        >执行升级</n-button>
        <n-button
          v-if="candidate.direction === 'downgrade'"
          size="tiny"
          type="warning"
          :loading="actioning && candidate.agent_id === actioning"
          @click="$emit('downgrade', candidate)"
        >执行降级</n-button>
        <span v-else style="color: #ccc; font-size: 12px;">无需变更</span>
      </n-space>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AutonomyCandidate } from '@/api/modules/agentos'

const props = defineProps<{
  candidate: AutonomyCandidate
  actioning?: string | boolean
}>()

defineEmits<{
  upgrade: [candidate: AutonomyCandidate]
  downgrade: [candidate: AutonomyCandidate]
}>()

const levelLabels: Record<string, string> = {
  OBSERVATION: 'L0 观察',
  SUGGESTION: 'L1 建议',
  SEMI_AUTONOMOUS: 'L2 半自主',
  FULL_AUTONOMOUS: 'L3 全自主',
}

const currentLevelLabel = computed(() => levelLabels[props.candidate.current_level] || props.candidate.current_level)
const targetLevelLabel = computed(() => levelLabels[props.candidate.target_level || ''] || props.candidate.target_level || '-')

const directionType = computed(() => {
  if (props.candidate.direction === 'upgrade') return 'success'
  if (props.candidate.direction === 'downgrade') return 'warning'
  return 'default'
})

const directionLabel = computed(() => {
  if (props.candidate.direction === 'upgrade') return '建议升级'
  if (props.candidate.direction === 'downgrade') return '建议降级'
  return '稳定'
})
</script>
