<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h2 style="margin: 0; font-size: 20px; font-weight: 600;">AI 助手</h2>
      <span style="color: rgba(0,0,0,0.45); font-size: 14px;">基于 Hermes 架构的跨境电商 AI Agent 系统</span>
    </div>

    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="12" v-for="agent in agents" :key="agent.agent_id">
        <a-card hoverable @click="goToAgent(agent.agent_id)" style="cursor: pointer;">
          <template #title>
            <a-space align="center">
              <a-tag :color="tagColor(agent.agent_id)">{{ agent.agent_id }}</a-tag>
              <span style="font-weight: 600;">{{ agent.name }}</span>
            </a-space>
          </template>
          <p style="color: #666; font-size: 13px; margin: 0;">{{ agent.description }}</p>
          <a-space style="margin-top: 12px;" wrap>
            <a-tag v-for="dp in agent.decision_points" :key="dp" color="processing">{{ dp }}</a-tag>
          </a-space>
          <a-space style="margin-top: 8px;" align="center">
            <span style="font-size: 12px; color: #999;">v{{ agent.version }}</span>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="近期决策" style="margin-top: 12px;">
      <a-table
        :columns="decisionColumns"
        :data-source="recentDecisions"
        :loading="loading"
        :pagination="{ current: pagination.page, pageSize: pagination.pageSize, total: total }"
        @change="onTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'agent_id'">
            <a-tag :color="tagColor(record.agent_id)">{{ record.agent_id }}</a-tag>
          </template>
          <template v-else-if="column.key === 'user_action'">
            <a-tag :color="actionColor(record.user_action)">{{ record.user_action }}</a-tag>
          </template>
          <template v-else-if="column.key === 'evolution_stage'">
            <a-tag>{{ record.evolution_stage }}</a-tag>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { agentApi } from '@/api/modules/agent'

const router = useRouter()
const loading = ref(false)
const agents = ref<any[]>([])
const recentDecisions = ref<any[]>([])
const total = ref(0)

const pagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
})

function tagColor(agentId: string) {
  const colors: Record<string, string> = {
    A3: 'orange', A4: 'green', A5: 'red',
    A6: 'purple', A7: 'blue',
    G1: 'cyan', G2: 'gold', G3: 'red',
  }
  return colors[agentId] || 'default'
}

function actionColor(action: string) {
  const colors: Record<string, string> = {
    accepted: 'success', modified: 'warning', rejected: 'error', ignored: 'default',
  }
  return colors[action] || 'default'
}

const decisionColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 70 },
  { title: '决策点', dataIndex: 'decision_point', key: 'decision_point', width: 120 },
  { title: '用户操作', dataIndex: 'user_action', key: 'user_action', width: 100 },
  { title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 80 },
  { title: '阶段', dataIndex: 'evolution_stage', key: 'evolution_stage', width: 100 },
]

async function fetchAgents() {
  try {
    const res: any = await agentApi.list()
    agents.value = res?.data || []
  } catch { /* placeholder */ }
}

async function fetchDecisions() {
  loading.value = true
  try {
    const res: any = await agentApi.getDecisions({ page: pagination.page, page_size: pagination.pageSize })
    recentDecisions.value = res?.records || []
    total.value = res?.total || 0
    pagination.itemCount = total.value
  } catch { /* no decisions yet */ }
  loading.value = false
}

function goToAgent(agentId: string) {
  router.push(`/agents/${agentId}`)
}

function onTableChange(pag: any) {
  pagination.page = pag.current
  fetchDecisions()
}

onMounted(() => { fetchAgents(); fetchDecisions() })
</script>
