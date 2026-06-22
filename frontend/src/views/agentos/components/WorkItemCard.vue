<template>
  <n-card size="small" class="work-item-card">
    <template #header>
      <n-space justify="space-between" align="center">
        <n-space align="center">
          <n-tag :type="riskType" size="small">{{ riskLabel }}</n-tag>
          <strong>{{ item.title }}</strong>
        </n-space>
        <n-tag size="small" :type="statusType">{{ statusLabel }}</n-tag>
      </n-space>
    </template>

    <n-space vertical size="small">
      <div class="muted">{{ item.summary || item.recommendation || '暂无说明' }}</div>
      <n-space size="small">
        <n-tag size="small" :bordered="false">{{ squadLabel }}</n-tag>
        <n-tag v-if="item.agent_id" size="small" :bordered="false">{{ item.agent_id }}</n-tag>
        <n-tag v-if="item.approval_required" size="small" type="warning">需审批</n-tag>
      </n-space>
      <n-space>
        <n-button size="tiny" type="primary" ghost @click="$emit('inspect', item)">查看</n-button>
        <n-button v-if="item.approval_required" size="tiny" type="success" ghost @click="$emit('approve', item)">批准</n-button>
        <n-button v-if="item.approval_required" size="tiny" type="warning" ghost @click="$emit('reject', item)">拒绝</n-button>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AgentOSWorkItem } from '@/api/modules/agentos'

const props = defineProps<{ item: AgentOSWorkItem }>()
defineEmits<{
  inspect: [item: AgentOSWorkItem]
  approve: [item: AgentOSWorkItem]
  reject: [item: AgentOSWorkItem]
}>()

const riskType = computed(() => {
  const map: Record<string, string> = { critical: 'error', high: 'error', medium: 'warning', low: 'info' }
  return map[props.item.risk_level] || 'default'
})
const riskLabel = computed(() => {
  const map: Record<string, string> = { critical: '严重', high: '高风险', medium: '中风险', low: '低风险' }
  return map[props.item.risk_level] || props.item.risk_level
})
const statusType = computed(() => {
  const map: Record<string, string> = { pending: 'warning', unread: 'warning', executed: 'success', resolved: 'success', rejected: 'default' }
  return map[props.item.status] || 'default'
})
const statusLabel = computed(() => {
  const map: Record<string, string> = { pending: '待处理', unread: '未读', read: '已读', executed: '已执行', resolved: '已解决', rejected: '已拒绝' }
  return map[props.item.status] || props.item.status
})
const squadLabel = computed(() => {
  const map: Record<string, string> = { growth: '增长小队', fulfillment: '履约小队', risk: '风控小队' }
  return map[props.item.squad] || props.item.squad
})
</script>

<style scoped>
.work-item-card { margin-bottom: 10px; }
.muted { color: #666; font-size: 13px; line-height: 1.5; }
</style>
