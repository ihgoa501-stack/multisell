<template>
  <div>
    <!-- 加载状态 -->
    <a-spin v-if="loading" style="margin-top: 40px; display: block;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </a-spin>

    <!-- 错误状态 -->
    <a-result v-else-if="error" status="error" title="加载失败" :sub-title="error">
      <template #extra>
        <a-button @click="fetchDetail">重试</a-button>
      </template>
    </a-result>

    <template v-else-if="detail">
      <!-- 页面标题 + Agent 信息 -->
      <div class="page-header">
        <div style="display: flex; align-items: center; gap: 12px;">
          <a-button type="text" size="small" @click="router.back()">
            <template #icon><ArrowLeftOutlined /></template>
          </a-button>
          <div>
            <div style="display: flex; align-items: center; gap: 8px;">
              <a-avatar :size="32">{{ detail.agent.name.charAt(0) }}</a-avatar>
              <h2 class="page-header-title">{{ detail.agent.name }}</h2>
            </div>
            <p class="page-header-subtitle">{{ detail.squad_name }}</p>
          </div>
        </div>
        <a-space>
          <AutonomyBadge :level="detail.agent.autonomy_level" />
          <a-tag v-if="detail.agent.status === 'active'" color="var(--ant-color-success)">工作中</a-tag>
          <a-tag v-else>{{ detail.agent.status }}</a-tag>
        </a-space>
      </div>

      <!-- 统计行 -->
      <a-row :gutter="12" style="margin-top: 12px;">
        <a-col :span="6">
          <a-card size="small">
            <div class="stat-value">{{ detail.agent.role || '-' }}</div>
            <div class="stat-label">角色</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card size="small">
            <div class="stat-value">{{ detail.current_work_items.length }}</div>
            <div class="stat-label">当前任务</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card size="small">
            <div class="stat-value">{{ detail.decision_count_7d }}</div>
            <div class="stat-label">7天决策</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card size="small">
            <div class="stat-value">{{ (detail.adoption_rate_7d * 100).toFixed(0) }}%</div>
            <div class="stat-label">7天采纳率</div>
          </a-card>
        </a-col>
      </a-row>

      <!-- 两列布局 -->
      <a-row :gutter="12" style="margin-top: 12px;">
        <!-- 左：当前任务 -->
        <a-col :span="12">
          <a-card title="当前任务" size="small">
            <template v-if="detail.current_work_items.length === 0">
              <a-empty description="暂无任务" style="padding: 20px 0;" />
            </template>
            <WorkItemCard
              v-for="item in detail.current_work_items"
              :key="item.id"
              :item="item"
              @status-updated="fetchDetail"
            />
          </a-card>
        </a-col>

        <!-- 右：近期操作记录 -->
        <a-col :span="12">
          <a-card title="近期操作记录" size="small">
            <template v-if="detail.recent_operations.length === 0">
              <a-empty description="暂无操作记录" style="padding: 20px 0;" />
            </template>
            <a-list v-else :data-source="detail.recent_operations" size="small">
              <template #renderItem="{ item: op }">
                <a-list-item :key="op.id">
                  <div style="display: flex; align-items: center; justify-content: space-between; width: 100%;">
                    <a-space align="center" size="small">
                      <a-tag :color="opActionColor(op.action)">
                        {{ opActionLabel(op.action) }}
                      </a-tag>
                      <span style="font-size: 13px;">{{ op.comment || op.new_status || '-' }}</span>
                    </a-space>
                    <span style="color: rgba(0,0,0,0.45); font-size: 11px;">{{ formatTime(op.created_at) }}</span>
                  </div>
                </a-list-item>
              </template>
            </a-list>
          </a-card>
        </a-col>
      </a-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import { getAgentOSAgentDetail } from '@/api/modules/agentos'
import type { AgentDetailResponse } from '@/api/modules/agentos'
import AutonomyBadge from '@/components/agentos/AutonomyBadge.vue'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const route = useRoute()
const router = useRouter()

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

function opActionColor(action: string): string {
  const map: Record<string, string> = {
    approve: 'var(--ant-color-success)',
    reject: 'var(--ant-color-warning)',
    autonomy_upgrade: 'var(--ant-color-success)',
    autonomy_downgrade: 'var(--ant-color-warning)',
    status_update: 'var(--ant-color-primary)',
  }
  return map[action] || 'var(--ant-color-primary)'
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
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.page-header-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  margin: 4px 0 0;
  color: rgba(0, 0, 0, 0.45);
  font-size: 14px;
}
.stat-value {
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 2px;
}
</style>
