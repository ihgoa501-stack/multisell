<template>
  <div>
    <n-page-header subtitle="统一处理 Agent 建议、异常、通知和待审批动作">
      <template #title>任务中心</template>
    </n-page-header>

    <n-card size="small" style="margin-top: 12px;">
      <n-space>
        <span class="filter-label">来源</span>
        <n-select v-model:value="query.source_type" clearable style="width: 160px" :options="sourceOptions" />
        <span class="filter-label">小队</span>
        <n-select v-model:value="query.squad" clearable style="width: 160px" :options="squadOptions" />
        <span class="filter-label">状态</span>
        <n-select v-model:value="query.status" clearable style="width: 160px" :options="statusOptions" />
        <n-button type="primary" @click="fetchItems">筛选</n-button>
      </n-space>
    </n-card>

    <n-card size="small" style="margin-top: 12px;" :loading="loading">
      <n-empty v-if="items.length === 0" description="暂无任务" />
      <work-item-card
        v-for="item in items"
        :key="item.id"
        :item="item"
        @inspect="inspectItem"
        @approve="inspectItem"
        @reject="inspectItem"
      />
      <n-pagination
        v-if="total > query.page_size"
        v-model:page="query.page"
        :page-size="query.page_size"
        :item-count="total"
        @update:page="fetchItems"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { agentosApi } from '@/api/modules/agentos'
import type { AgentOSWorkItem } from '@/api/modules/agentos'
import WorkItemCard from './components/WorkItemCard.vue'

const router = useRouter()
const message = useMessage()
const loading = ref(false)
const items = ref<AgentOSWorkItem[]>([])
const total = ref(0)
const query = reactive({
  source_type: null as string | null,
  squad: null as string | null,
  status: null as string | null,
  page: 1,
  page_size: 20,
})

const sourceOptions = [
  { label: 'Agent 动作', value: 'agent_action' },
  { label: '异常', value: 'exception' },
  { label: '通知', value: 'notification' },
]
const squadOptions = [
  { label: '增长小队', value: 'growth' },
  { label: '履约小队', value: 'fulfillment' },
  { label: '风控小队', value: 'risk' },
]
const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '未读', value: 'unread' },
  { label: '已执行', value: 'executed' },
  { label: '已解决', value: 'resolved' },
]

async function fetchItems() {
  loading.value = true
  try {
    const res: any = await agentosApi.getWorkItems({
      source_type: query.source_type || undefined,
      squad: query.squad || undefined,
      status: query.status || undefined,
      page: query.page,
      page_size: query.page_size,
    })
    items.value = res?.records || res?.data?.records || []
    total.value = res?.total || res?.data?.total || 0
  } catch (error: any) {
    message.error(error?.response?.data?.message || '加载任务失败')
  } finally {
    loading.value = false
  }
}

function inspectItem(item: AgentOSWorkItem) {
  if (item.audit_link) router.push(item.audit_link)
}

onMounted(fetchItems)
</script>

<style scoped>
.filter-label { color: #666; font-size: 13px; line-height: 32px; }
</style>
