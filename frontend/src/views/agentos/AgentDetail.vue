<template>
  <n-spin :show="loading">
    <n-page-header @back="handleBack">
      <template #title>Agent 详情</template>
    </n-page-header>
    <n-empty v-if="!loading && !agent" description="Agent not found" />
  </n-spin>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { AgentDetailResponse } from '@/api/modules/agentos'
import { getAgentOSAgentDetail } from '@/api/modules/agentos'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const agent = ref<AgentDetailResponse | null>(null)

function handleBack() {
  router.push('/agentos/control-center')
}

onMounted(async () => {
  loading.value = true
  try {
    const agentId = route.params.agentId as string
    const res = await getAgentOSAgentDetail(agentId)
    agent.value = res.data as AgentDetailResponse
  } catch (e) {
    console.error('Failed to load agent detail', e)
  } finally {
    loading.value = false
  }
})
</script>
