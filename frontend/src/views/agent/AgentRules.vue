<template>
  <div>
    <n-page-header subtitle="个人规则库 — Agent 偏好过滤器">
      <template #title>个人规则</template>
    </n-page-header>

    <n-card style="margin-top: 12px;">
      <n-space style="margin-bottom: 12px;">
        <n-button type="primary" @click="showCreate = true">＋ 新建规则</n-button>
        <n-select v-model:value="filterAgent" :options="agentOptions" clearable placeholder="全部 Agent" style="width: 120px;" @update:value="fetchRules" />
      </n-space>
      <n-data-table :columns="columns" :data="rules" :loading="loading" />
    </n-card>

    <n-modal v-model:show="showCreate" title="新建规则" preset="card" style="width: 600px;">
      <n-form>
        <n-form-item label="Agent">
          <n-select v-model:value="form.agent_id" :options="agentOptions" />
        </n-form-item>
        <n-form-item label="决策点">
          <n-input v-model:value="form.decision_point" placeholder="例如: discount_check" />
        </n-form-item>
        <n-form-item label="规则类型">
          <n-select v-model:value="form.rule_type" :options="typeOptions" />
        </n-form-item>
        <n-form-item label="规则名称">
          <n-input v-model:value="form.rule_name" placeholder="例如: 竞品降价幅度减半" />
        </n-form-item>
        <n-form-item label="条件 (JSON)">
          <n-input v-model:value="form.rule_condition" type="textarea" :rows="3" placeholder='{"field": "discount_rate", "op": "gt", "value": 15}' />
        </n-form-item>
        <n-form-item label="动作 (JSON)">
          <n-input v-model:value="form.rule_action" type="textarea" :rows="3" placeholder='{"override": {"action": "block"}, "modifier": {"type": "absolute", "action": "block"}}' />
        </n-form-item>
        <n-space justify="end">
          <n-button @click="showCreate = false">取消</n-button>
          <n-button type="primary" @click="createRule">创建</n-button>
        </n-space>
      </n-form>
    </n-modal>

    <n-modal v-model:show="showEdit" title="编辑规则" preset="card" style="width: 600px;">
      <n-form>
        <n-form-item label="规则名称">
          <n-input v-model:value="editForm.rule_name" />
        </n-form-item>
        <n-form-item label="条件 (JSON)">
          <n-input v-model:value="editForm.rule_condition" type="textarea" :rows="3" />
        </n-form-item>
        <n-form-item label="动作 (JSON)">
          <n-input v-model:value="editForm.rule_action" type="textarea" :rows="3" />
        </n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="editForm.status" :options="statusOptions" />
        </n-form-item>
        <n-space justify="end">
          <n-button @click="showEdit = false">取消</n-button>
          <n-button type="primary" @click="updateRule">保存</n-button>
        </n-space>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, computed, onMounted } from 'vue'
import { NTag, useMessage, NSpace } from 'naive-ui'
import { agentApi } from '@/api/modules/agent'

const message = useMessage()
const loading = ref(false)
const rules = ref<any[]>([])
const agents = ref<any[]>([])
const filterAgent = ref(null)
const showCreate = ref(false)
const showEdit = ref(false)
const editingId = ref<number | null>(null)

const typeOptions = [
  { label: '阈值 (threshold)', value: 'threshold' },
  { label: '策略 (strategy)', value: 'strategy' },
  { label: '风格 (style)', value: 'style' },
  { label: '否决 (veto)', value: 'veto' },
]

const statusOptions = [
  { label: '启用', value: 'active' },
  { label: '暂停', value: 'paused' },
  { label: '退役', value: 'retired' },
]

const form = reactive({
  agent_id: 'G3',
  decision_point: '',
  rule_type: 'threshold',
  rule_name: '',
  rule_condition: '{}',
  rule_action: '{}',
  priority: 100,
})

const editForm = reactive({
  rule_name: '',
  rule_condition: '{}',
  rule_action: '{}',
  status: 'active',
})

const agentOptions = computed(() => agents.value.map((a: any) => ({ label: `${a.agent_id} - ${a.name}`, value: a.agent_id })))

const typeColors: Record<string, string> = {
  threshold: 'info',
  strategy: 'warning',
  style: 'success',
  veto: 'error',
}

const columns = [
  { title: 'Agent', key: 'agent_id', width: 70, render: (row: any) => h(NTag, { size: 'small' }, { default: () => row.agent_id }) },
  { title: '名称', key: 'rule_name', ellipsis: { tooltip: true } },
  { title: '类型', key: 'rule_type', width: 90, render: (row: any) => h(NTag, { type: typeColors[row.rule_type] || 'default', size: 'small' }, { default: () => row.rule_type }) },
  { title: '决策点', key: 'decision_point', width: 120 },
  { title: '来源', key: 'source', width: 80 },
  { title: '优先级', key: 'priority', width: 70 },
  { title: '状态', key: 'status', width: 80, render: (row: any) => h(NTag, { type: row.status === 'active' ? 'success' : 'default', size: 'small' }, { default: () => row.status }) },
  { title: '应用次数', key: 'times_applied', width: 80 },
  { title: '操作', key: 'actions', width: 150, render: (row: any) => h(NSpace, null, {
    default: () => [
      h('a', { style: 'cursor: pointer; color: #2080f0;', onClick: () => openEdit(row) }, '编辑'),
      h('a', { style: 'cursor: pointer; color: #e74c3c;', onClick: () => deleteRule(row.id) }, '删除'),
    ],
  })},
]

async function fetchAgents() {
  try {
    const res: any = await agentApi.list()
    agents.value = res?.data || []
  } catch { /* */ }
}

async function fetchRules() {
  loading.value = true
  try {
    const params: any = {}
    if (filterAgent.value) params.agent_id = filterAgent.value
    const res: any = await agentApi.listRules(params)
    rules.value = res?.data || []
  } catch { /* */ }
  loading.value = false
}

async function createRule() {
  try {
    let condition: any, action: any
    try { condition = JSON.parse(form.rule_condition) } catch { condition = {} }
    try { action = JSON.parse(form.rule_action) } catch { action = {} }
    await agentApi.createRule({
      agent_id: form.agent_id,
      decision_point: form.decision_point,
      rule_type: form.rule_type,
      rule_name: form.rule_name,
      rule_condition: condition,
      rule_action: action,
      priority: form.priority,
    })
    message.success('规则已创建')
    showCreate.value = false
    fetchRules()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '创建失败')
  }
}

function openEdit(row: any) {
  editingId.value = row.id
  editForm.rule_name = row.rule_name || ''
  editForm.rule_condition = JSON.stringify(row.rule_condition, null, 2)
  editForm.rule_action = JSON.stringify(row.rule_action, null, 2)
  editForm.status = row.status || 'active'
  showEdit.value = true
}

async function updateRule() {
  if (!editingId.value) return
  try {
    let condition: any, action: any
    try { condition = JSON.parse(editForm.rule_condition) } catch { condition = {} }
    try { action = JSON.parse(editForm.rule_action) } catch { action = {} }
    await agentApi.updateRule(editingId.value, {
      rule_name: editForm.rule_name,
      rule_condition: condition,
      rule_action: action,
      status: editForm.status,
    })
    message.success('规则已更新')
    showEdit.value = false
    fetchRules()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '更新失败')
  }
}

async function deleteRule(id: number) {
  try {
    await agentApi.deleteRule(id)
    message.success('规则已删除')
    fetchRules()
  } catch { message.error('删除失败') }
}

onMounted(() => { fetchAgents(); fetchRules() })
</script>
