<template>
  <div>
    <n-page-header subtitle="AI 原生跨境电商经营工作台">
      <template #title>AgentOS 总控台</template>
      <template #extra>
        <n-space>
          <n-button size="small" secondary @click="fetchData">刷新</n-button>
          <n-button size="small" type="error" ghost>暂停全部 Agent</n-button>
        </n-space>
      </template>
    </n-page-header>

    <n-grid :cols="4" :x-gap="12" :y-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card size="small"><div class="metric">¥{{ fmt(summary.sales_today) }}</div><div class="label">今日销售</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">¥{{ fmt(summary.profit_today) }}</div><div class="label">今日利润</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">{{ summary.pending_approvals }}</div><div class="label">待审批动作</div></n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card size="small"><div class="metric">{{ Math.round(summary.agent_automation_rate * 100) }}%</div><div class="label">自动化率</div></n-card>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="24" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item :span="5">
        <n-card title="任务筛选" size="small">
          <n-space vertical>
            <n-button block :type="selectedSquad === '' ? 'primary' : 'default'" @click="selectedSquad = ''">全部任务</n-button>
            <n-button block :type="selectedSquad === 'growth' ? 'primary' : 'default'" @click="selectedSquad = 'growth'">增长小队</n-button>
            <n-button block :type="selectedSquad === 'fulfillment' ? 'primary' : 'default'" @click="selectedSquad = 'fulfillment'">履约小队</n-button>
            <n-button block :type="selectedSquad === 'risk' ? 'primary' : 'default'" @click="selectedSquad = 'risk'">风控小队</n-button>
          </n-space>
        </n-card>
      </n-grid-item>

      <n-grid-item :span="12">
        <n-card title="AI 任务工作台" size="small" :loading="loading">
          <n-empty v-if="filteredItems.length === 0" description="暂无待处理任务" />
          <work-item-card
            v-for="item in filteredItems"
            :key="item.id"
            :item="item"
            @inspect="inspectItem"
            @approve="inspectItem"
            @reject="inspectItem"
          />
        </n-card>
      </n-grid-item>

      <n-grid-item :span="7">
        <n-space vertical>
          <squad-card v-for="squad in squads" :key="squad.id" :squad="squad" />
        </n-space>
      </n-grid-item>
    </n-grid>

    <n-card title="内置电商模板" size="small" style="margin-top: 12px;">
      <n-grid :cols="3" :x-gap="12" :y-gap="12">
        <n-grid-item v-for="template in templates" :key="template.id">
          <template-card :template="template" @open="openTemplate" />
        </n-grid-item>
      </n-grid>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSSummary, AgentOSSquad, AgentOSTemplate, AgentOSWorkItem } from '@/api/modules/agentos'
import WorkItemCard from './components/WorkItemCard.vue'
import SquadCard from './components/SquadCard.vue'
import TemplateCard from './components/TemplateCard.vue'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const selectedSquad = ref('')
const summary = reactive<AgentOSSummary>({
  sales_today: 0,
  profit_today: 0,
  inventory_risks: 0,
  pending_approvals: 0,
  active_work_items: 0,
  agent_automation_rate: 0,
})
const workItems = ref<AgentOSWorkItem[]>([])
const squads = ref<AgentOSSquad[]>([])
const templates = ref<AgentOSTemplate[]>([])

const filteredItems = computed(() => {
  if (!selectedSquad.value) return workItems.value
  return workItems.value.filter(item => item.squad === selectedSquad.value)
})

function fmt(value: number) {
  return (value || 0).toLocaleString('zh-CN', { maximumFractionDigits: 2 })
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await agentosApi.getControlCenter()
    const data = res?.data || {}
    Object.assign(summary, data.summary || {})
    workItems.value = data.work_items || []
    squads.value = data.squads || []
    templates.value = data.templates || []
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载 AgentOS 总控台失败')
  } finally {
    loading.value = false
  }
}

function inspectItem(item: AgentOSWorkItem) {
  if (item.audit_link) router.push(item.audit_link)
}

function openTemplate(template: AgentOSTemplate) {
  router.push(template.route)
}

onMounted(fetchData)
</script>

<style scoped>
.metric { font-size: 22px; font-weight: 700; }
.label { color: #888; font-size: 12px; margin-top: 4px; }
</style>
