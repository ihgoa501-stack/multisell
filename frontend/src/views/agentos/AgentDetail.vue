<template>
  <div>
    <!-- 加载状态 -->
    <n-spin v-if="loading" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <!-- 错误状态 -->
    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchDetail">重试</n-button>
      </template>
    </n-result>

    <template v-else-if="detail">
      <!-- 页面标题 + Agent 信息 -->
      <n-page-header :subtitle="detail.squad_name" @back="router.back()">
        <template #title>
          <n-space align="center" size="small">
            <n-avatar :size="32" round>{{ detail.agent.name.charAt(0) }}</n-avatar>
            <span>{{ detail.agent.name }}</span>
          </n-space>
        </template>
        <template #extra>
          <n-space>
            <AutonomyBadge :level="detail.agent.autonomy_level" />
            <n-tag v-if="detail.agent.status === 'active'" type="success" size="small" :bordered="false">工作中</n-tag>
            <n-tag v-else type="default" size="small" :bordered="false">{{ detail.agent.status }}</n-tag>
          </n-space>
        </template>
      </n-page-header>

      <!-- 统计行 -->
      <n-grid :cols="4" :x-gap="12" style="margin-top: 12px;">
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.agent.role || '-' }}</div>
            <div class="stat-label">角色</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.current_work_items.length }}</div>
            <div class="stat-label">当前任务</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ detail.decision_count_7d }}</div>
            <div class="stat-label">7天决策</div>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card size="small">
            <div class="stat-value">{{ (detail.adoption_rate_7d * 100).toFixed(0) }}%</div>
            <div class="stat-label">7天采纳率</div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <!-- 两列布局 -->
      <n-grid :cols="2" :x-gap="12" style="margin-top: 12px;">
        <!-- 左：当前任务 -->
        <n-grid-item>
          <n-card title="当前任务" size="small">
            <template v-if="detail.current_work_items.length === 0">
              <n-empty description="暂无任务" style="padding: 20px 0;" />
            </template>
            <WorkItemCard
              v-for="item in detail.current_work_items"
              :key="item.id"
              :item="item"
              @status-updated="fetchDetail"
            />
          </n-card>
        </n-grid-item>

        <!-- 右：近期操作记录 -->
        <n-grid-item>
          <n-card title="近期操作记录" size="small">
            <template v-if="detail.recent_operations.length === 0">
              <n-empty description="暂无操作记录" style="padding: 20px 0;" />
            </template>
            <n-list v-else>
              <n-list-item v-for="op in detail.recent_operations" :key="op.id">
                <n-space align="center" justify="space-between">
                  <n-space align="center" size="small">
                    <n-tag :type="opActionType(op.action)" size="tiny" :bordered="false">
                      {{ opActionLabel(op.action) }}
                    </n-tag>
                    <span style="font-size: 13px;">{{ op.comment || op.new_status || '-' }}</span>
                  </n-space>
                  <span style="color: #999; font-size: 11px;">{{ formatTime(op.created_at) }}</span>
                </n-space>
              </n-list-item>
            </n-list>
          </n-card>
        </n-grid-item>
      </n-grid>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAgentOSAgentDetail } from '@/api/modules/agentos'
import type { AgentDetailResponse } from '@/api/modules/agentos'
import AutonomyBadge from '@/components/agentos/AutonomyBadge.vue'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()

const loading = ref(false)
const error = ref<string | null>(null)
const detail = ref<AgentDetailResponse | null>(null)

async function fetchDetail() {
  const agentId = route.params.agentId as string
  if (!agentId) return

  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSAgentDetail(agentId)
    detail.value = res?.data || res
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载 Agent 详情失败')
  } finally {
    loading.value = false
  }
}

function opActionType(action: string): 'success' | 'info' | 'warning' | 'error' {
  const map: Record<string, 'success' | 'info' | 'warning' | 'error'> = {
    approve: 'success',
    reject: 'warning',
    autonomy_upgrade: 'success',
    autonomy_downgrade: 'warning',
    status_update: 'info',
  }
  return map[action] || 'info'
}

function opActionLabel(action: string): string {
  const map: Record<string, string> = {
    approve: '审批通过',
    reject: '拒绝',
    autonomy_upgrade: '升级',
    autonomy_downgrade: '降级',
    status_update: '状态变更',
  }
  return map[action] || action
}

function formatTime(val: string | null): string {
  if (!val) return ''
  return new Date(val).toLocaleString('zh-CN')
}

onMounted(fetchDetail)
</script>

<style scoped>
.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: #888;
  margin-top: 2px;
}
</style>
