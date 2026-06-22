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

    <n-space v-if="item.metadata?.action_type" size="small" style="margin-top: 6px;">
      <n-tag size="small" :bordered="false">{{ item.metadata.action_type }}</n-tag>
      <n-tag v-if="item.metadata?.business_object_type" size="small" :bordered="false">
        {{ item.metadata.business_object_type }}:{{ item.metadata.business_object_id || '-' }}
      </n-tag>
      <n-tag v-if="item.metadata?.confidence !== undefined" size="small" :bordered="false">
        置信度 {{ Math.round(Number(item.metadata.confidence || 0) * 100) }}%
      </n-tag>
    </n-space>

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

    <n-alert v-if="execError" type="error" closable style="margin-top: 8px;" @close="execError = ''">
      {{ execError }}
    </n-alert>

    <template #footer>
      <n-space justify="space-between" align="center">
        <div style="color: #999; font-size: 11px;">
          {{ timeLabel }}
        </div>
        <n-space size="small">
          <n-button v-if="item.action_url" size="tiny" @click="handleAction">查看详情</n-button>

          <!-- 执行按钮：已审批或无需审批且待决的 action_proposal -->
          <n-button
            v-if="showExecute"
            size="tiny"
            type="primary"
            :loading="mutating"
            @click="handleExecuteProposal"
          >执行</n-button>

          <!-- 撤销按钮：已执行的 action_proposal（显示在复盘按钮之前） -->
          <n-button
            v-if="showUndo"
            size="tiny"
            type="warning"
            ghost
            :loading="undoing"
            @click="showUndoConfirm = true"
          >↩ 撤销</n-button>

          <!-- 复盘按钮：已执行的 action_proposal -->
          <n-button
            v-if="showReview"
            size="tiny"
            secondary
            @click="showReviewModal = true"
          >复盘</n-button>

          <!-- 低风险非提案 → 直接标记完成 -->
          <n-button
            v-if="!item.requires_approval && item.status === 'pending' && !isActionProposal"
            size="tiny"
            secondary
            :loading="mutating"
            @click="handleComplete"
          >标记完成</n-button>

          <!-- 审批按钮 -->
          <n-button
            v-if="showApprove"
            size="tiny"
            type="primary"
            ghost
            :loading="mutating"
            @click="handleApprove"
          >审批通过</n-button>
          <n-button
            v-if="showReject"
            size="tiny"
            type="warning"
            ghost
            :loading="mutating"
            @click="handleReject"
          >拒绝</n-button>
        </n-space>
      </n-space>
    </template>

    <!-- 复盘弹窗 -->
    <n-modal v-model:show="showReviewModal" title="提交复盘" preset="card" style="width: 420px;">
      <n-form label-placement="top">
        <n-form-item label="执行结果">
          <n-radio-group v-model:value="reviewForm.outcome">
            <n-radio value="positive">正面</n-radio>
            <n-radio value="neutral">中性</n-radio>
            <n-radio value="negative">负面</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="业务指标">
          <n-input v-model:value="reviewForm.business_metric" placeholder="如: 销售额变化" />
        </n-form-item>
        <n-form-item label="指标变化">
          <n-input-number v-model:value="reviewForm.metric_delta" placeholder="数值" style="width: 100%;" />
        </n-form-item>
        <n-form-item label="备注">
          <n-input v-model:value="reviewForm.notes" type="textarea" :rows="3" placeholder="复盘说明" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showReviewModal = false">取消</n-button>
          <n-button type="primary" :loading="reviewing" @click="handleSubmitReview">提交复盘</n-button>
        </n-space>
      </template>
    </n-modal>

      <!-- 撤销确认弹窗 -->
      <n-modal v-model:show="showUndoConfirm" title="确认撤销" preset="card" style="width: 400px;" :mask-closable="false">
        <p style="margin-bottom: 12px;">确定要撤销操作 "{{ item.title }}" 吗？</p>
        <p style="color: #888; font-size: 13px;">撤销操作会创建一个新的补偿提案，需经过审批和重新执行。</p>
        <template #footer>
          <n-space justify="end">
            <n-button @click="showUndoConfirm = false" :disabled="undoing">取消</n-button>
            <n-button type="warning" :loading="undoing" @click="handleUndo">确认撤销</n-button>
          </n-space>
        </template>
      </n-modal>
  </n-card>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import type { AgentOSWorkItem } from '@/api/modules/agentos'
import {
  updateWorkItemStatus,
  approveWorkItem,
  rejectWorkItem,
  executeActionProposal,
  reviewActionProposal,
  undoActionProposal,
} from '@/api/modules/agentos'
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
const execError = ref('')
const showReviewModal = ref(false)
const reviewing = ref(false)
const showUndoConfirm = ref(false)
const undoing = ref(false)
const reviewForm = ref({
  outcome: 'positive' as 'positive' | 'neutral' | 'negative',
  business_metric: '',
  metric_delta: null as number | null,
  notes: '',
})

const proposalId = computed(() => {
  if (props.item.source_type === 'action_proposal' && props.item.id.startsWith('action_proposal:')) {
    return parseInt(props.item.id.split(':')[1], 10)
  }
  return null
})

const isActionProposal = computed(() => proposalId.value !== null)

const status = computed(() => props.item.status)

const showExecute = computed(() =>
  !props.hideActions &&
  isActionProposal.value &&
  status.value === 'in_progress'
)

const showReview = computed(() =>
  !props.hideActions &&
  isActionProposal.value &&
  status.value === 'completed'
)

const showUndo = computed(() =>
  !props.hideActions &&
  isActionProposal.value &&
  status.value === 'completed'
)

const showApprove = computed(() =>
  !props.hideActions &&
  isActionProposal.value &&
  status.value === 'pending' &&
  props.item.requires_approval
)

const showReject = computed(() =>
  !props.hideActions &&
  isActionProposal.value &&
  status.value === 'pending' &&
  props.item.requires_approval
)

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
    action_proposal: '动作提案',
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
  execError.value = ''
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
  execError.value = ''
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

async function handleExecuteProposal() {
  if (!proposalId.value) return
  mutating.value = true
  execError.value = ''
  try {
    await executeActionProposal(proposalId.value, { executor: 'operator' })
    message.success('执行成功')
    emit('statusUpdated', props.item)
  } catch (e: any) {
    const msg = e?.response?.data?.message || e?.message || '执行失败'
    execError.value = msg
    message.error(msg)
  } finally {
    mutating.value = false
  }
}

async function handleSubmitReview() {
  if (!proposalId.value) return
  reviewing.value = true
  try {
    await reviewActionProposal(proposalId.value, {
      outcome: reviewForm.value.outcome,
      business_metric: reviewForm.value.business_metric || null,
      metric_delta: reviewForm.value.metric_delta,
      notes: reviewForm.value.notes || null,
    })
    message.success('复盘已提交')
    showReviewModal.value = false
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || '复盘提交失败')
  } finally {
    reviewing.value = false
  }
}

async function handleUndo() {
  if (!proposalId.value) return
  undoing.value = true
  try {
    const res: any = await undoActionProposal(proposalId.value)
    const proposalId = res?.data?.compensation_proposal?.source_id
    message.success('撤销提案已创建，请审批后执行')
    showUndoConfirm.value = false
    emit('statusUpdated', props.item)
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.message || '撤销失败')
    showUndoConfirm.value = false
  } finally {
    undoing.value = false
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
