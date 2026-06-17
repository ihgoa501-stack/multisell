<template>
  <div>
    <n-page-header subtitle="基于 Hermes 架构的跨境电商 AI Agent 系统">
      <template #title>AI 助手</template>
    </n-page-header>

    <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item v-for="agent in agents" :key="agent.agent_id">
        <n-card hoverable @click="goToAgent(agent.agent_id)" style="cursor: pointer;">
          <template #header>
            <n-space align="center">
              <n-tag :bordered="false" :color="{ color: tagColor(agent.agent_id), textColor: '#fff' }">{{ agent.agent_id }}</n-tag>
              <span style="font-weight: 600;">{{ agent.name }}</span>
            </n-space>
          </template>
          <p style="color: #666; font-size: 13px; margin: 0;">{{ agent.description }}</p>
          <n-space style="margin-top: 12px;">
            <n-tag v-for="dp in agent.decision_points" :key="dp" size="small" type="info">{{ dp }}</n-tag>
          </n-space>
          <n-space style="margin-top: 8px;" align="center">
            <span style="font-size: 12px; color: #999;">v{{ agent.version }}</span>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card title="近期决策" style="margin-top: 12px;">
      <n-data-table :columns="decisionColumns" :data="recentDecisions" :loading="loading" :pagination="pagination" @update:page="onPageChange" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { useRouter, useMessage, NTag, NSpace } from 'naive-ui'
import { agentApi } from '@/api/modules/agent'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const agents = ref<any[]>([])
const recentDecisions = ref<any[]>([])
const total = ref(0)

const pagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
  onChange: (page: number) => { pagination.page = page; fetchDecisions() },
})

function tagColor(agentId: string) {
  const colors: Record<string, string> = {
    A3: '#e67e22', A4: '#2ecc71', A5: '#e74c3c',
    A6: '#9b59b6', A7: '#3498db',
    G1: '#1abc9c', G2: '#f39c12', G3: '#e74c3c',
  }
  return colors[agentId] || '#95a5a6'
}

const decisionColumns = [
  { title: '时间', key: 'created_at', width: 170 },
  { title: 'Agent', key: 'agent_id', width: 70, render: (row: any) => h(NTag, { size: 'small', color: { color: tagColor(row.agent_id), textColor: '#fff' } }, { default: () => row.agent_id }) },
  { title: '决策点', key: 'decision_point', width: 120 },
  { title: '用户操作', key: 'user_action', width: 100, render: (row: any) => {
    const colors: Record<string, string> = { accepted: 'success', modified: 'warning', rejected: 'error', ignored: 'default' }
    return h(NTag, { type: colors[row.user_action] || 'default', size: 'small' }, { default: () => row.user_action })
  }},
  { title: '置信度', key: 'confidence', width: 80 },
  { title: '阶段', key: 'evolution_stage', width: 100, render: (row: any) => h(NTag, { size: 'small' }, { default: () => row.evolution_stage }) },
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

function onPageChange(page: number) {
  pagination.page = page
}

onMounted(() => { fetchAgents(); fetchDecisions() })
</script>
