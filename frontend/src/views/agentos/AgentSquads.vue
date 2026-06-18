<template>
  <div>
    <n-page-header subtitle="按增长、履约、风控管理 AI 运营团队">
      <template #title>Agent 小队</template>
      <template #extra><n-button size="small" @click="fetchSquads">刷新</n-button></template>
    </n-page-header>

    <n-grid :cols="3" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
      <n-grid-item v-for="squad in squads" :key="squad.id">
        <squad-card :squad="squad" />
        <n-card size="small" style="margin-top: 8px;">
          <n-space>
            <n-button size="small" ghost @click="router.push('/agents')">查看 Agent</n-button>
            <n-button size="small" ghost @click="router.push('/agents/rules')">规则</n-button>
            <n-button size="small" ghost @click="router.push('/agents/entropy')">熵管理</n-button>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card title="自治等级说明" size="small" style="margin-top: 12px;">
      <n-grid :cols="4" :x-gap="12">
        <n-grid-item><n-alert type="default" title="观察">只读数据，生成报告。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="info" title="建议">生成建议，人执行。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="success" title="半自主">低风险自动，高风险审批。</n-alert></n-grid-item>
        <n-grid-item><n-alert type="warning" title="全自主">仅限高信任低风险链路。</n-alert></n-grid-item>
      </n-grid>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSSquad } from '@/api/modules/agentos'
import SquadCard from './components/SquadCard.vue'

const router = useRouter()
const message = useMessage()
const squads = ref<AgentOSSquad[]>([])

async function fetchSquads() {
  try {
    const res: any = await agentosApi.getSquads()
    squads.value = res?.data || []
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载 Agent 小队失败')
  }
}

onMounted(fetchSquads)
</script>
