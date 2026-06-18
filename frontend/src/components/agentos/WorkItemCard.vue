<template>
  <n-card size="small" :bordered="true" hoverable style="margin-bottom: 8px;">
    <template #header>
      <n-space align="center" justify="space-between">
        <n-space align="center" size="small">
          <n-tag :type="priorityType" size="small" :bordered="false">{{ priorityLabel }}</n-tag>
          <span style="font-weight: 600; font-size: 14px;">{{ item.title }}</span>
        </n-space>
        <n-space align="center" size="small">
          <n-tag v-if="item.requires_approval" type="warning" size="small" :bordered="false">需审批</n-tag>
          <AutonomyBadge :level="item.autonomy_level" />
        </n-space>
      </n-space>
    </template>

    <p v-if="item.description" style="color: #666; font-size: 13px; margin: 4px 0;">{{ item.description }}</p>

    <n-grid :cols="4" :x-gap="12" :y-gap="8" style="margin-top: 8px;">
      <n-grid-item>
        <div class="meta-label">来源</div>
        <div class="meta-value">{{ sourceLabel }}</div>
      </n-grid-item>
      <n-grid-item>
        <div class="meta-label">风险</div>
        <n-tag :type="riskType" size="small" :bordered="false">{{ riskLabel }}</n-tag>
      </n-grid-item>
      <n-grid-item>
        <div class="meta-label">负责 Agent</div>
        <div class="meta-value">{{ item.agent_name || item.agent_id || '-' }}</div>
      </n-grid-item>
      <n-grid-item>
        <div class="meta-label">团队</div>
        <div class="meta-value">{{ item.squad_name || item.squad_id || '-' }}</div>
      </n-grid-item>
    </n-grid>

    <template #footer>
      <n-space justify="space-between" align="center">
        <div style="color: #999; font-size: 11px;">
          {{ timeLabel }}
        </div>
        <n-space size="small">
          <n-button v-if="item.action_url" size="tiny" @click="handleAction">查看详情</n-button>
          <n-button
            v-if="!item.requires_approval && item.status === 'pending'"
            size="tiny"
            secondary
            :loading="mutating"
            @click="handleComplete"
          >标记完成</n-button>
          <n-button
            v-if="item.requires_approval"
            size="tiny"
            type="primary"
            ghost
            :loading="mutating"
            @click="handleApprove"
          >审批通过</n-button>
          <n-button
            v-if="item.requires_approval"
            size="tiny"
            type="warning"
            ghost
            :loading="mutating"
            @click="handleReject"
          >拒绝</n-button>
        </n-space>
      </n-space>
    </template>
  </n-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { AgentOSWorkItem } from '@/api/modules/agentos'
import { updateWorkItemStatus, approveWorkItem, rejectWorkItem } from '@/api/modules/agentos'
import AutonomyBadge from './AutonomyBadge.vue'

const props = defineProps<{
  item: AgentOSWorkItem
  hideActions?: boolean
}>()

const emit = defineEmits<{
  inspect: [item: AgentOSWorkItem]
  approve: [item: AgentOSWorkItem]
  reject: [item: AgentOSWorkItem]
  statusUpdated: [item: AgentOSWorkItem]
}>()

const router = useRouter()
const message = useMessage()
const mutating = ref(false)

const priorityLabel = computed(() => {
  const map: Record<string, string> = { low: '低', medium: '中', high: '高', critical: '紧急' }
  return map[props.item.priority] || props.item.priority
})

const priorityType = computed(() => {
  const map: Record<string, 'default' | 'info' | 'warning' | 'error'> = {
    low: 'default',
    medium: 'info',
    high: 'warning',
    critical: 'error',
  }
  return map[props.item.priority] || 'default'
})

const riskLabel = computed(() => {
  const map: Record<string, string> = { low: '低风险', medium: '中风险', high: '高风险', critical: '严重' }
  return map[props.item.risk_level] || props.item.risk_level
})

const riskType = computed(() => {
  const map: Record<string, 'success' | 'warning' | 'error'> = {
    low: 'success',
    medium: 'warning',
    high: 'error',
    critical: 'error',
  }
  return map[props.item.risk_level] || 'default'
})

const sourceLabel = computed(() => {
  const map: Record<string, string> = {
    agent_action: 'Agent 动作',
    exception: '异常',
    notification: '通知',
    listing_task: '上架任务',
  }
  return map[props.item.source_type] || props.item.source_type
})

const timeLabel = computed(() => {
  if (!props.item.created_at) return ''
  return new Date(props.item.created_at).toLocaleString('zh-CN')
})

function handleAction() {
  if (props.item.action_url) {
    router.push(props.item.action_url)
  }
  emit('inspect', props.item)
}

async function handleComplete() {
  mutating.value = true
  try {
    await updateWorkItemStatus(props.item.id, { status: 'completed' })
    message.success('已标记完成')
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '操作失败')
  } finally {
    mutating.value = false
  }
}

async function handleApprove() {
  mutating.value = true
  try {
    await approveWorkItem(props.item.id, { action: 'approve', comment: '批准执行' })
    message.success('已审批通过')
    emit('approve', props.item)
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '审批失败')
  } finally {
    mutating.value = false
  }
}

async function handleReject() {
  mutating.value = true
  try {
    await rejectWorkItem(props.item.id, { action: 'reject', comment: '已拒绝' })
    message.success('已拒绝')
    emit('reject', props.item)
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '操作失败')
  } finally {
    mutating.value = false
  }
}
</script>

<style scoped>
.meta-label {
  font-size: 11px;
  color: #888;
  margin-bottom: 2px;
}
.meta-value {
  font-size: 13px;
  color: #333;
}
</style>
