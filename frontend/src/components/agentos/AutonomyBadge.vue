<template>
  <n-tag
    :type="tagType"
    :bordered="false"
    size="small"
    :round="true"
  >
    {{ label }}
    <template #icon>
      <span v-html="iconSvg" />
    </template>
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AutonomyLevel } from '@/api/modules/agentos'

const props = withDefaults(defineProps<{
  level: AutonomyLevel
  showLevel?: boolean
}>(), {
  showLevel: false,
})

const autonomyConfig: Record<AutonomyLevel, { label: string; type: 'default' | 'info' | 'success' | 'warning'; icon: string; level: number }> = {
  OBSERVATION: { label: '观察', type: 'default', icon: '👁', level: 0 },
  SUGGESTION: { label: '建议', type: 'info', icon: '💡', level: 1 },
  SEMI_AUTONOMOUS: { label: '半自主', type: 'success', icon: '⚡', level: 2 },
  FULL_AUTONOMOUS: { label: '全自主', type: 'warning', icon: '🤖', level: 3 },
}

const config = computed(() => autonomyConfig[props.level] || autonomyConfig.SUGGESTION)
const tagType = computed(() => config.value.type)
const label = computed(() => props.showLevel ? `L${config.value.level} ${config.value.label}` : config.value.label)
const iconSvg = computed(() => config.value.icon)
</script>
