<template>
  <n-card size="small" hoverable @click="$emit('open', template)">
    <template #header>
      <n-space justify="space-between">
        <strong>{{ template.title }}</strong>
        <n-tag size="small" :type="modeType">{{ template.mode }}</n-tag>
      </n-space>
    </template>
    <p class="desc">{{ template.description }}</p>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSTemplate } from '@/api/modules/agentos'

const props = defineProps<{ template: AgentOSTemplate }>()
defineEmits<{ open: [template: AgentOSTemplate] }>()

const modeType = computed(() => props.template.mode === 'Agent' ? 'success' : 'info')
</script>

<style scoped>
.desc { color: #666; font-size: 13px; margin: 0; line-height: 1.5; }
</style>
