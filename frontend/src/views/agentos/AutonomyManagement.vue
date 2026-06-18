<template>
  <div>
    <n-page-header subtitle="管理 Agent 自治等级与升级/降级操作">
      <template #title>自治管理</template>
      <template #extra>
        <n-button size="small" @click="fetchData" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <n-spin v-if="loading && !loaded" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchData">重试</n-button>
      </template>
    </n-result>

    <template v-else>
      <!-- 升级候选 -->
      <n-card title="候选升级" style="margin-top: 12px;">
        <n-empty v-if="upgradeCandidates.length === 0" description="暂无候选升级" />
        <n-table v-else :bordered="false" :single-line="false">
          <thead>
            <tr>
              <th>Agent</th>
              <th>小队</th>
              <th>当前等级</th>
              <th>建议</th>
              <th>置信度</th>
              <th>理由</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in upgradeCandidates" :key="c.agent_id">
              <td>{{ c.agent_name }}</td>
              <td>{{ c.squad_name }}</td>
              <td>{{ c.current_level }}</td>
              <td>
                <n-tag v-if="c.suggested && c.direction === 'upgrade'" type="success">升级 {{ c.target_level }}</n-tag>
                <n-tag v-else-if="c.suggested && c.direction === 'downgrade'" type="warning">降级 {{ c.target_level }}</n-tag>
                <span v-else>无</span>
              </td>
              <td>{{ (c.confidence * 100).toFixed(0) }}%</td>
              <td>{{ c.reason }}</td>
              <td>
                <n-button
                  v-if="c.suggested"
                  size="tiny"
                  :type="c.direction === 'upgrade' ? 'success' : 'warning'"
                  :loading="c.loading"
                  @click="executeLevelChange(c)"
                >
                  {{ c.direction === 'upgrade' ? '升级' : '降级' }}
                </n-button>
              </td>
            </tr>
          </tbody>
        </n-table>
      </n-card>

      <!-- 操作日志 -->
      <n-card title="操作记录" style="margin-top: 12px;">
        <n-empty v-if="operations.length === 0" description="暂无操作记录" />
        <n-table v-else :bordered="false" :single-line="false">
          <thead>
            <tr>
              <th>时间</th>
              <th>Agent</th>
              <th>动作</th>
              <th>来源</th>
              <th>说明</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="op in operations" :key="op.id">
              <td>{{ op.created_at ? new Date(op.created_at).toLocaleString() : '-' }}</td>
              <td>{{ op.item_id }}</td>
              <td>{{ op.action }}</td>
              <td>{{ op.source_type || '-' }}</td>
              <td>{{ op.comment || '-' }}</td>
            </tr>
          </tbody>
        </n-table>
      </n-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getAgentOSUpgradeCandidates,
  getAgentOSOperations,
  upgradeAgentLevel,
  downgradeAgentLevel,
} from '@/api/modules/agentos'
import type { AutonomyCandidate, AgentOSOperationLog } from '@/api/modules/agentos'
import { useMessage } from 'naive-ui'

const message = useMessage()

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)

const upgradeCandidates = ref<(AutonomyCandidate & { loading?: boolean })[]>([])
const operations = ref<AgentOSOperationLog[]>([])

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const [candRes, opRes] = await Promise.all([
      getAgentOSUpgradeCandidates(),
      getAgentOSOperations({ limit: 20 }),
    ])
    const candBody = candRes as any
    upgradeCandidates.value = (candBody.data?.candidates ?? candBody.data ?? []).map((c: AutonomyCandidate) => ({
      ...c,
      loading: false,
    }))
    const opBody = opRes as any
    operations.value = opBody.data?.records ?? opBody.data ?? []
    loaded.value = true
  } catch (e: any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
}

async function executeLevelChange(c: AutonomyCandidate & { loading?: boolean }) {
  c.loading = true
  try {
    if (c.direction === 'upgrade') {
      await upgradeAgentLevel(c.agent_id, c.target_level!)
    } else if (c.direction === 'downgrade') {
      await downgradeAgentLevel(c.agent_id, c.target_level!)
    }
    message.success(`${c.agent_name} 等级已变更`)
    await fetchData()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  } finally {
    c.loading = false
  }
}

onMounted(() => {
  fetchData()
})
</script>
