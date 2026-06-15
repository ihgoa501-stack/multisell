<template>
  <n-card title="Agent 动作提案" :bordered="false" class="mt-3">
    <template #header-extra>
      <n-button size="small" type="primary" @click="showCreateModal = true">
        新建提案
      </n-button>
    </template>

    <n-data-table
      :columns="columns"
      :data="actions"
      :loading="loading"
      :pagination="{ pageSize: 5 }"
      size="small"
    />

    <!-- Create modal -->
    <n-modal v-model:show="showCreateModal" title="新建动作提案" preset="card" style="width: 600px;">
      <n-form>
        <n-form-item label="动作类型">
          <n-input v-model:value="form.action_type" placeholder="如 resolve_bill" />
        </n-form-item>
        <n-form-item label="标题">
          <n-input v-model:value="form.title" placeholder="动作标题" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="form.description" type="textarea" :rows="3" />
        </n-form-item>
        <n-form-item label="提议参数 (JSON)">
          <n-input v-model:value="form.proposedPayloadStr" type="textarea" :rows="3" placeholder='{"action": "resolve"}' />
        </n-form-item>
        <n-form-item label="当前状态快照 (JSON)">
          <n-input v-model:value="form.beforeSnapshotStr" type="textarea" :rows="3" placeholder='{"status": "amount_mismatch"}' />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false">取消</n-button>
          <n-button type="primary" :loading="creating" @click="handleCreate">提交</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-card>
</template>

<script setup lang="ts">
import { h, ref, onMounted, reactive } from 'vue'
import { NButton, NSpace, NTag, useMessage } from 'naive-ui'
import {
  createAgentAction,
  getAgentActions,
  approveAgentAction,
  rejectAgentAction,
  markExecutedAgentAction,
  type AgentAction,
} from '@/api/modules/agentActions'

const props = defineProps<{
  exceptionId?: number
}>()

const message = useMessage()
const loading = ref(false)
const creating = ref(false)
const actions = ref<AgentAction[]>([])
const showCreateModal = ref(false)
const form = reactive({
  action_type: '',
  title: '',
  description: '',
  proposedPayloadStr: '',
  beforeSnapshotStr: '',
})

const statusTag: Record<string, { type: any; text: string }> = {
  proposed: { type: 'info', text: '待审批' },
  approved: { type: 'success', text: '已通过' },
  rejected: { type: 'error', text: '已驳回' },
  executed: { type: 'success', text: '已执行' },
}

const columns = [
  { title: '动作类型', key: 'action_type', width: 120 },
  { title: '标题', key: 'title', ellipsis: { tooltip: true } },
  { title: '提议人', key: 'proposed_by', width: 100 },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: AgentAction) => {
      const meta = statusTag[row.status] || { type: 'default', text: row.status }
      return h(NTag, { type: meta.type, size: 'small' }, { default: () => meta.text })
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 260,
    render: (row: AgentAction) =>
      h(NSpace, null, {
        default: () => {
          const btns: any[] = []
          if (row.status === 'proposed') {
            btns.push(
              h(NButton, { size: 'small', type: 'success', onClick: () => handleApprove(row) }, { default: () => '审批通过' }),
              h(NButton, { size: 'small', type: 'warning', onClick: () => handleReject(row) }, { default: () => '驳回' }),
            )
          }
          if (row.status === 'approved') {
            btns.push(
              h(NButton, { size: 'small', type: 'primary', onClick: () => handleExecute(row) }, { default: () => '标记执行' }),
            )
          }
          return btns
        },
      }),
  },
]

async function fetchActions() {
  loading.value = true
  try {
    const params: any = {}
    if (props.exceptionId) params.exception_id = props.exceptionId
    const resp = await getAgentActions(params)
    actions.value = resp.data || []
  } catch { /* ignore */ }
  finally { loading.value = false }
}

async function handleCreate() {
  creating.value = true
  try {
    let proposed_payload: any = undefined
    let before_snapshot: any = undefined
    try { if (form.proposedPayloadStr) proposed_payload = JSON.parse(form.proposedPayloadStr) } catch { /* ignore */ }
    try { if (form.beforeSnapshotStr) before_snapshot = JSON.parse(form.beforeSnapshotStr) } catch { /* ignore */ }

    await createAgentAction({
      exception_id: props.exceptionId,
      action_type: form.action_type,
      title: form.title,
      description: form.description || undefined,
      proposed_payload,
      before_snapshot,
    })
    message.success('提案已创建')
    showCreateModal.value = false
    form.action_type = ''
    form.title = ''
    form.description = ''
    form.proposedPayloadStr = ''
    form.beforeSnapshotStr = ''
    await fetchActions()
  } catch (err: any) {
    message.error(err?.response?.data?.message || err?.message || '创建失败')
  } finally { creating.value = false }
}

async function handleApprove(row: AgentAction) {
  try { await approveAgentAction(row.id); message.success('已审批通过'); await fetchActions() }
  catch (err: any) { message.error(err?.response?.data?.message || '审批失败') }
}

async function handleReject(row: AgentAction) {
  const reason = prompt('驳回原因:', '')
  if (reason === null) return
  try { await rejectAgentAction(row.id, reason || undefined); message.success('已驳回'); await fetchActions() }
  catch (err: any) { message.error(err?.response?.data?.message || '驳回失败') }
}

async function handleExecute(row: AgentAction) {
  try { await markExecutedAgentAction(row.id, {}); message.success('已标记执行'); await fetchActions() }
  catch (err: any) { message.error(err?.response?.data?.message || '执行失败') }
}

onMounted(fetchActions)
</script>

<style scoped>
.mt-3 { margin-top: 12px; }
</style>
