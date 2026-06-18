<template>
  <div>
    <n-page-header subtitle="统一处理 Agent 建议、异常、通知和待审批动作">
      <template #title>任务中心</template>
      <template #extra>
        <n-button size="small" @click="fetchItems" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <!-- 筛选栏 -->
    <n-card size="small" style="margin-top: 12px;">
      <n-space wrap :size="[12, 8]">
        <div class="filter-group">
          <span class="filter-label">状态</span>
          <n-select
            v-model:value="filters.status"
            clearable
            :options="statusOptions"
            style="width: 120px;"
            placeholder="全部状态"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">优先级</span>
          <n-select
            v-model:value="filters.priority"
            clearable
            :options="priorityOptions"
            style="width: 120px;"
            placeholder="全部优先级"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">团队</span>
          <n-select
            v-model:value="filters.squad"
            clearable
            :options="squadOptions"
            style="width: 120px;"
            placeholder="全部团队"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">需审批</span>
          <n-switch v-model:value="filters.requiresApproval" />
        </div>
        <n-button type="primary" size="small" @click="applyFilters">筛选</n-button>
        <n-button size="small" @click="resetFilters">重置</n-button>
      </n-space>
    </n-card>

    <!-- 任务列表 -->
    <n-card size="small" style="margin-top: 12px;" :loading="loading">
      <template v-if="items.length === 0 && !loading">
        <n-empty description="暂无匹配的任务" style="padding: 40px 0;" />
      </template>

      <WorkItemCard
        v-for="item in items"
        :key="item.id"
        :item="item"
        @inspect="navigateToItem"
        @approve="navigateToItem"
        @status-updated="fetchItems"
      />

      <!-- 分页 -->
      <n-space v-if="total > limit" justify="center" style="margin-top: 16px;">
        <n-pagination
          :page="currentPage"
          :page-size="limit"
          :item-count="total"
          @update:page="onPageChange"
        />
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAgentOSWorkItems } from '@/api/modules/agentos'
import type { AgentOSWorkItem, WorkItemQuery } from '@/api/modules/agentos'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const items = ref<AgentOSWorkItem[]>([])
const total = ref(0)
const limit = ref(20)
const currentPage = ref(1)

const filters = reactive({
  status: null as string | null,
  priority: null as string | null,
  squad: null as string | null,
  requiresApproval: false,
})

const statusOptions = [
  { label: '待处理', value: 'pending' },
  { label: '处理中', value: 'in_progress' },
  { label: '已完成', value: 'completed' },
  { label: '失败', value: 'failed' },
  { label: '已阻塞', value: 'blocked' },
  { label: '已取消', value: 'cancelled' },
]

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '中', value: 'medium' },
  { label: '高', value: 'high' },
  { label: '紧急', value: 'critical' },
]

const squadOptions = [
  { label: '增长小队', value: 'growth' },
  { label: '履约小队', value: 'fulfillment' },
  { label: '风控小队', value: 'risk' },
]

async function fetchItems() {
  loading.value = true
  try {
    const query: WorkItemQuery = {
      limit: limit.value,
      offset: (currentPage.value - 1) * limit.value,
    }
    if (filters.status) query.status = filters.status
    if (filters.priority) query.priority = filters.priority
    if (filters.squad) query.squad = filters.squad
    if (filters.requiresApproval) query.requires_approval = true

    const res: any = await getAgentOSWorkItems(query)
    // PageResult format: { code, records, total, page, page_size }
    items.value = res?.records || res?.data?.records || []
    total.value = res?.total || res?.data?.total || 0
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.message || '加载任务失败')
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  currentPage.value = 1
  fetchItems()
}

function resetFilters() {
  filters.status = null
  filters.priority = null
  filters.squad = null
  filters.requiresApproval = false
  currentPage.value = 1
  fetchItems()
}

function onPageChange(page: number) {
  currentPage.value = page
  fetchItems()
}

function navigateToItem(item: AgentOSWorkItem) {
  if (item.action_url) {
    router.push(item.action_url)
  }
}
</script>

<style scoped>
.filter-group {
  display: flex;
  align-items: center;
  gap: 6px;
}
.filter-label {
  color: #666;
  font-size: 13px;
  white-space: nowrap;
}
</style>
