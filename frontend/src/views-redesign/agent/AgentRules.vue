<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h2 style="margin: 0; font-size: 20px; font-weight: 600;">个人规则</h2>
      <span style="color: rgba(0,0,0,0.45); font-size: 14px;">个人规则库 — Agent 偏好过滤器</span>
    </div>

    <a-card style="margin-top: 12px;">
      <a-space style="margin-bottom: 12px;">
        <a-button type="primary" @click="showCreate = true">+ 新建规则</a-button>
        <a-select v-model:value="filterAgent" :options="agentOptions" allow-clear placeholder="全部 Agent" style="width: 160px;" @change="fetchRules" />
      </a-space>
      <a-table :columns="columns" :data-source="rules" :loading="loading" row-key="id" :pagination="false">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'agent_id'">
            <a-tag>{{ record.agent_id }}</a-tag>
          </template>
          <template v-else-if="column.key === 'rule_type'">
            <a-tag :color="typeColors[record.rule_type] || 'default'">{{ record.rule_type }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 'active' ? 'success' : 'default'">{{ record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a type="link" style="cursor: pointer;" @click="openEdit(record)">编辑</a>
              <a-popconfirm title="确定删除该规则？" @confirm="deleteRule(record.id)" ok-text="确定" cancel-text="取消">
                <a style="cursor: pointer; color: var(--ant-color-error);">删除</a>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="showCreate" title="新建规则" :width="600" @ok="createRule" ok-text="创建" cancel-text="取消">
      <a-form layout="vertical">
        <a-form-item label="Agent">
          <a-select v-model:value="form.agent_id" :options="agentOptions" />
        </a-form-item>
        <a-form-item label="决策点">
          <a-input v-model:value="form.decision_point" placeholder="例如: discount_check" />
        </a-form-item>
        <a-form-item label="规则类型">
          <a-select v-model:value="form.rule_type" :options="typeOptions" />
        </a-form-item>
        <a-form-item label="规则名称">
          <a-input v-model:value="form.rule_name" placeholder="例如: 竞品降价幅度减半" />
        </a-form-item>
        <a-form-item label="条件 (JSON)">
          <a-textarea v-model:value="form.rule_condition" :rows="3" placeholder='{"field": "discount_rate", "op": "gt", "value": 15}' />
        </a-form-item>
        <a-form-item label="动作 (JSON)">
          <a-textarea v-model:value="form.rule_action" :rows="3" placeholder='{"override": {"action": "block"}, "modifier": {"type": "absolute", "action": "block"}}' />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="showEdit" title="编辑规则" :width="600" @ok="updateRule" ok-text="保存" cancel-text="取消">
      <a-form layout="vertical">
        <a-form-item label="规则名称">
          <a-input v-model:value="editForm.rule_name" />
        </a-form-item>
        <a-form-item label="条件 (JSON)">
          <a-textarea v-model:value="editForm.rule_condition" :rows="3" />
        </a-form-item>
        <a-form-item label="动作 (JSON)">
          <a-textarea v-model:value="editForm.rule_action" :rows="3" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="editForm.status" :options="statusOptions" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { agentApi } from '@/api/modules/agent'

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
  threshold: 'processing',
  strategy: 'warning',
  style: 'success',
  veto: 'error',
}

const columns = [
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 70 },
  { title: '名称', dataIndex: 'rule_name', key: 'rule_name', ellipsis: true },
  { title: '类型', dataIndex: 'rule_type', key: 'rule_type', width: 90 },
  { title: '决策点', dataIndex: 'decision_point', key: 'decision_point', width: 120 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 80 },
  { title: '优先级', dataIndex: 'priority', key: 'priority', width: 70 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '应用次数', dataIndex: 'times_applied', key: 'times_applied', width: 80 },
  { title: '操作', key: 'actions', width: 150 },
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
