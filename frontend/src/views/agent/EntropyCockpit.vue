<template>
  <div>
    <n-page-header subtitle="控制 Agent 系统退化, 保持规则健康">
      <template #title>
        熵管理驾驶舱
      </template>
    </n-page-header>

    <n-grid :cols="4" :x-gap="12" style="margin-top: 12px;">
      <n-grid-item>
        <n-card :bordered="false" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: #fff;">
          <n-statistic :value="dashboard.system_entropy_index" :precision="3">
            <template #label>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">系统熵指数</span>
            </template>
          </n-statistic>
          <n-progress :percentage="entropyPercent" :color="entropyColor" :height="6" style="margin-top: 8px;" />
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); color: #fff;">
          <n-statistic :value="dashboard.avg_health_score" :precision="3">
            <template #label>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">平均健康分</span>
            </template>
          </n-statistic>
          <n-space style="margin-top: 8px;">
            <n-tag size="small" :color="{ color: '#27ae60', textColor: '#fff' }">健康 {{ dashboard.healthy_rule_count || 0 }}</n-tag>
            <n-tag size="small" :color="{ color: '#f39c12', textColor: '#fff' }">警告 {{ dashboard.warning_rule_count || 0 }}</n-tag>
            <n-tag size="small" :color="{ color: '#e74c3c', textColor: '#fff' }">不健康 {{ dashboard.unhealthy_rule_count || 0 }}</n-tag>
          </n-space>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); color: #fff;">
          <n-statistic :value="dashboard.active_rules">
            <template #label>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">活跃规则 / 总数</span>
            </template>
          </n-statistic>
          <n-space style="margin-top: 8px;">
            <span style="font-size: 12px; opacity: 0.8;">影子: {{ dashboard.shadow_rules }} | 总数: {{ dashboard.total_rules }}</span>
          </n-space>
        </n-card>
      </n-grid-item>
      <n-grid-item>
        <n-card :bordered="false" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); color: #fff;">
          <n-statistic :value="dashboard.pending_merge_count">
            <template #label>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">待合并 / 24h变更</span>
            </template>
          </n-statistic>
          <n-space style="margin-top: 8px;">
            <span style="font-size: 12px; opacity: 0.8;">24h变更: {{ dashboard.recent_changes_count }}</span>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-space style="margin-top: 16px; margin-bottom: 12px;">
      <n-button type="primary" @click="runDefenses" :loading="defending" icon-placement="left">
        执行防守 (TTL+Budget+Decay+Merge+Regret)
      </n-button>
      <n-button @click="refreshAll" :loading="loading">刷新</n-button>
    </n-space>

    <n-card v-if="defenseResult" title="防守执行结果" style="margin-bottom: 12px;" type="success">
      <n-descriptions :column="5" size="small">
        <n-descriptions-item label="过期规则">{{ defenseResult.actions.expired_rules }}</n-descriptions-item>
        <n-descriptions-item label="Budget超限">{{ defenseResult.actions.budget_exceeded }}</n-descriptions-item>
        <n-descriptions-item label="置信度衰减">{{ defenseResult.actions.decay_applied }}</n-descriptions-item>
        <n-descriptions-item label="合并规则">{{ defenseResult.actions.merged_pairs }}</n-descriptions-item>
        <n-descriptions-item label="遗憾回滚">{{ defenseResult.actions.regret_rollbacks }}</n-descriptions-item>
      </n-descriptions>
    </n-card>

    <n-tabs type="line" animated>
      <n-tab-pane name="health" tab="规则健康评分">
        <n-data-table :columns="healthColumns" :data="healthScores" :loading="loading" :max-height="400" />
      </n-tab-pane>
      <n-tab-pane name="spc" tab="SPC 控制状态">
        <n-data-table :columns="spcColumns" :data="spcStatus" :loading="loadingSpc" :max-height="400" />
      </n-tab-pane>
      <n-tab-pane name="changes" tab="变更日志">
        <n-space style="margin-bottom: 8px;">
          <n-select v-model:value="changeFilter" :options="sourceTypeOptions" clearable placeholder="筛选来源" style="width: 200px;" @update:value="fetchChanges" />
        </n-space>
        <n-data-table :columns="changeColumns" :data="changes" :loading="loadingChanges" :pagination="changePagination" @update:page="onChangePage" />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted, computed } from 'vue'
import { useMessage, NTag, NSpace, NCode } from 'naive-ui'
import { entropyApi } from '@/api/modules/agent'

const message = useMessage()
const loading = ref(false)
const loadingSpc = ref(false)
const loadingChanges = ref(false)
const defending = ref(false)

const dashboard = ref<any>({})
const healthScores = ref<any[]>([])
const spcStatus = ref<any[]>([])
const changes = ref<any[]>([])
const changeTotal = ref(0)
const changeFilter = ref<string | null>(null)
const defenseResult = ref<any>(null)

const changePagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
  onChange: (page: number) => { changePagination.page = page; fetchChanges() },
})

const entropyPercent = computed(() => Math.round((dashboard.value.system_entropy_index || 0) * 100))
const entropyColor = computed(() => {
  const idx = dashboard.value.system_entropy_index || 0
  if (idx < 0.3) return '#27ae60'
  if (idx < 0.6) return '#f39c12'
  return '#e74c3c'
})

const sourceTypeOptions = [
  { label: 'GDS 决策', value: 'gds' },
  { label: 'GDS 代理', value: 'gds_proxy' },
  { label: '人工', value: 'human' },
  { label: 'Nudge', value: 'nudge' },
  { label: '自动提取', value: 'auto_extract' },
]

function riskColor(level: string) {
  return level === 'healthy' ? 'success' : level === 'warning' ? 'warning' : 'error'
}

const healthColumns = [
  { title: '规则名', key: 'rule_name', width: 150, ellipsis: true },
  { title: 'Agent', key: 'agent_id', width: 60 },
  { title: '类型', key: 'rule_type', width: 80 },
  { title: '评分', key: 'score', width: 80, render: (row: any) => h('span', {
    style: { color: row.score >= 0.6 ? '#27ae60' : row.score >= 0.4 ? '#f39c12' : '#e74c3c', fontWeight: 600 },
  }, row.score) },
  { title: '风险', key: 'risk_level', width: 80, render: (row: any) => h(NTag, { type: riskColor(row.risk_level), size: 'small' }, { default: () => row.risk_level }) },
  { title: '采纳率', key: 'override_rate', width: 80, render: (row: any) => `${((1 - row.override_rate) * 100).toFixed(0)}%` },
  { title: '频次', key: 'times_applied', width: 60 },
  { title: '覆盖', key: 'times_overridden', width: 60 },
  { title: '置信度', key: 'confidence', width: 80, render: (row: any) => `${(row.confidence * 100).toFixed(0)}%` },
  { title: '距上次(天)', key: 'days_since_last_applied', width: 90, render: (row: any) => row.days_since_last_applied != null ? `${row.days_since_last_applied}d` : '-' },
  { title: '状态', key: 'status', width: 70 },
]

const spcColumns = [
  { title: 'Agent', key: 'agent_id', width: 60 },
  { title: '决策点', key: 'decision_point', width: 120 },
  { title: '指标', key: 'metric_name', width: 100 },
  { title: '当前值', key: 'current_value', width: 80 },
  { title: '均值(μ)', key: 'baseline_mean', width: 80 },
  { title: 'UCL(μ+3σ)', key: 'ucl', width: 90 },
  { title: 'LCL(μ-3σ)', key: 'lcl', width: 90 },
  { title: '连续同侧', key: 'consecutive_same_side', width: 80 },
  { title: '异常', key: 'is_out_of_control', width: 60, render: (row: any) => row.is_out_of_control ? h(NTag, { type: 'error', size: 'small' }, { default: () => '失控' }) : row.is_warning ? h(NTag, { type: 'warning', size: 'small' }, { default: () => '警告' }) : '-' },
  { title: '下次重算', key: 'next_recalc_at', width: 160 },
]

const changeColumns = [
  { title: '时间', key: 'created_at', width: 160 },
  { title: '类型', key: 'source_type', width: 80, render: (row: any) => {
    const colors: Record<string, string> = { gds: '#3498db', gds_proxy: '#9b59b6', human: '#2ecc71', nudge: '#f39c12', auto_extract: '#e67e22' }
    return h(NTag, { size: 'small', color: { color: colors[row.source_type] || '#95a5a6', textColor: '#fff' } }, { default: () => row.source_type })
  }},
  { title: '变更内容', key: 'change_summary', ellipsis: true },
  { title: '目标', key: 'target_type', width: 80 },
  { title: '字段', key: 'field_path', width: 80 },
]

async function fetchDashboard() {
  try {
    const res: any = await entropyApi.getDashboard()
    dashboard.value = res?.data || {}
  } catch { /* not ready */ }
}

async function fetchHealthScores() {
  try {
    const res: any = await entropyApi.getHealthScores()
    healthScores.value = res?.data || []
  } catch { /* not ready */ }
}

async function fetchSpc() {
  loadingSpc.value = true
  try {
    const res: any = await entropyApi.getSpc()
    spcStatus.value = res?.data || []
  } catch { /* not ready */ }
  loadingSpc.value = false
}

async function fetchChanges() {
  loadingChanges.value = true
  try {
    const res: any = await entropyApi.getChanges({
      source_type: changeFilter.value || undefined,
      page: changePagination.page,
      page_size: changePagination.pageSize,
    })
    changes.value = res?.records || []
    changeTotal.value = res?.total || 0
    changePagination.itemCount = changeTotal.value
  } catch { /* not ready */ }
  loadingChanges.value = false
}

async function runDefenses() {
  defending.value = true
  try {
    const res: any = await entropyApi.runDefenses()
    defenseResult.value = res?.data
    message.success(`防守完成, 影响 ${res?.data?.total_affected || 0} 条规则`)
    await Promise.all([fetchDashboard(), fetchHealthScores(), fetchChanges()])
  } catch (e: any) {
    message.error(e?.response?.data?.message || '防守执行失败')
  }
  defending.value = false
}

async function refreshAll() {
  loading.value = true
  await Promise.all([fetchDashboard(), fetchHealthScores(), fetchSpc(), fetchChanges()])
  loading.value = false
}

function onChangePage(page: number) {
  changePagination.page = page
}

onMounted(() => { refreshAll() })
</script>
