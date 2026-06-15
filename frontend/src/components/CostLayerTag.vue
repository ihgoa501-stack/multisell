<template>
  <n-tag :type="tagType" size="small" :bordered="false">
    {{ label }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = defineProps<{
  layer: string
}>()

const config: Record<string, { type: 'success' | 'info' | 'warning' | 'primary' | 'default'; text: string }> = {
  estimated: { type: 'default', text: '预估' },
  snapshot: { type: 'info', text: '快照' },
  actual: { type: 'success', text: '实际' },
  allocated: { type: 'warning', text: '分摊' },
  mixed: { type: 'warning', text: '混合' },
}

const tagType = computed(() => config[props.layer]?.type || 'default')
const label = computed(() => config[props.layer]?.text || props.layer)
</script>
