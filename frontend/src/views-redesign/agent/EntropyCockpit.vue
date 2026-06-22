<template>
  <div>
    <div style="margin-bottom: 16px;">
      <h2 style="margin: 0; font-size: 20px; font-weight: 600;">熵管理驾驶舱</h2>
      <span style="color: rgba(0,0,0,0.45); font-size: 14px;">控制 Agent 系统退化, 保持规则健康</span>
    </div>

    <a-row :gutter="[12, 12]" style="margin-top: 12px;">
      <a-col :span="6">
        <a-card :bordered="false" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: #fff;">
          <a-statistic :value="dashboard.system_entropy_index" :precision="3">
            <template #title>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">系统熵指数</span>
            </template>
          </a-statistic>
          <a-progress :percent="entropyPercent" :stroke-color="entropyColor" :show-info="false" size="small" style="margin-top: 8px;" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); color: #fff;">
          <a-statistic :value="dashboard.avg_health_score" :precision="3">
            <template #title>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">平均健康分</span>
            </template>
          </a-statistic>
          <a-space style="margin-top: 8px;">
            <a-tag color="#27ae60">健康 {{ dashboard.healthy_rule_count || 0 }}</a-tag>
            <a-tag color="#f39c12">警告 {{ dashboard.warning_rule_count || 0 }}</a-tag>
            <a-tag color="#e74c3c">不健康 {{ dashboard.unhealthy_rule_count || 0 }}</a-tag>
          </a-space>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); color: #fff;">
          <a-statistic :value="dashboard.active_rules">
            <template #title>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">活跃规则 / 总数</span>
            </template>
          </a-statistic>
          <a-space style="margin-top: 8px;">
            <span style="font-size: 12px; opacity: 0.8;">影子: {{ dashboard.shadow_rules }} | 总数: {{ dashboard.total_rules }}</span>
          </a-space>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :bordered="false" style="background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); color: #fff;">
          <a-statistic :value="dashboard.pending_merge_count">
            <template #title>
              <span style="color: rgba(255,255,255,0.8); font-size: 13px;">待合并 / 24h变更</span>
            </template>
          </a-statistic>
          <a-space style="margin-top: 8px;">
            <span style="font-size: 12px; opacity: 0.8;">24h变更: {{ dashboard.recent_changes_count }}</span>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <a-space style="margin-top: 16px; margin-bottom: 12px;">
      <a-button type="primary" @click="runDefenses" :loading="defending">
        执行防守 (TTL+Budget+Decay+Merge+Regret)
      </a-button>
      <a-button @click="refreshAll" :loading="loading">刷新</a-button>
    </a-space>

    <a-card v-if="defenseResult" title="防守执行结果" style="margin-bottom: 12px;" :bordered="false">
      <a-descriptions :column="5" size="small" bordered>
        <a-descriptions-item label="过期规则">{{ defenseResult.actions.expired_rules }}</a-descriptions-item>
        <a-descriptions-item label="Budget超限">{{ defenseResult.actions.budget_exceeded }}</a-descriptions-item>
        <a-descriptions-item label="置信度衰减">{{ defenseResult.actions.decay_applied }}</a-descriptions-item>
        <a-descriptions-item label="合并规则">{{ defenseResult.actions.merged_pairs }}</a-descriptions-item>
        <a-descriptions-item label="遗憾回滚">{{ defenseResult.actions.regret_rollbacks }}</a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="health" tab="规则健康评分">
        <a-table :columns="healthColumns" :data-source="healthScores" :loading="loading" :scroll="{ y: 400 }" row-key="rule_name" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'score'">
              <span :style="{ color: record.score >= 0.6 ? '#27ae60' : record.score >= 0.4 ? '#f39c12' : '#e74c3c', fontWeight: 600 }">
                {{ record.score }}
              </span>
            </template>
            <template v-else-if="column.key === 'risk_level'">
              <a-tag :color="riskColor(record.risk_level)">{{ record.risk_level }}</a-tag>
            </template>
            <template v-else-if="column.key === 'override_rate'">
              {{ ((1 - record.override_rate) * 100).toFixed(0) }}%
            </template>
            <template v-else-if="column.key === 'confidence'">
              {{ (record.confidence * 100).toFixed(0) }}%
            </template>
            <template v-else-if="column.key === 'days_since_last_applied'">
              {{ record.days_since_last_applied != null ? record.days_since_last_applied + 'd' : '-' }}
            </template>
          </template>
        </a-table>
      </a-tab-pane>
      <a-tab-pane key="spc" tab="SPC 控制状态">
        <a-table :columns="spcColumns" :data-source="spcStatus" :loading="loadingSpc" :scroll="{ y: 400 }" row-key="decision_point" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'is_out_of_control'">
              <a-tag v-if="record.is_out_of_control" color="error">失控</a-tag>
              <a-tag v-else-if="record.is_warning" color="warning">警告</a-tag>
              <span v-else>-</span>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
      <a-tab-pane key="changes" tab="变更日志">
        <a-space style="margin-bottom: 8px;">
          <a-select v-model:value="changeFilter" :options="sourceTypeOptions" allow-clear placeholder="筛选来源" style="width: 200px;" @change="fetchChanges" />
        </a-space>
        <a-table :columns="changeColumns" :data-source="changes" :loading="loadingChanges" :pagination="{ current: changePagination.page, pageSize: changePagination.pageSize, total: changeTotal }" @change="onChangeTableChange" row-key="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'source_type'">
              <a-tag :color="sourceTypeColor(record.source_type)">{{ record.source_type }}</a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { entropyApi } from '@/api/modules/agent'

const loading = ref(false)
const loadingSpc = ref(false)
const loadingChanges = ref(false)
const defending = ref(false)
const activeTab = ref('health')

const dashboard = ref<any>({})
const healthScores = ref<any[]>([])
const spcStatus = ref<any[]>([])
const changes = ref<any[]>([])
const changeTotal = ref(0)
const changeFilter = ref<string | null>(null)
const defenseResult = ref<any>(null)

const changePagination = reactive({
  page: 1, pageSize: 10, itemCount: 0,
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

function sourceTypeColor(type: string) {
  const colors: Record<string, string> = {
    gds: 'blue', gds_proxy: 'purple', human: 'green', nudge: 'gold', auto_extract: 'orange',
  }
  return colors[type] || 'default'
}

function riskColor(level: string) {
  return level === 'healthy' ? 'success' : level === 'warning' ? 'warning' : 'error'
}

const healthColumns = [
  { title: '规则名', dataIndex: 'rule_name', key: 'rule_name', width: 150, ellipsis: true },
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 60 },
  { title: '类型', dataIndex: 'rule_type', key: 'rule_type', width: 80 },
  { title: '评分', dataIndex: 'score', key: 'score', width: 80 },
  { title: '风险', dataIndex: 'risk_level', key: 'risk_level', width: 80 },
  { title: '采纳率', dataIndex: 'override_rate', key: 'override_rate', width: 80 },
  { title: '频次', dataIndex: 'times_applied', key: 'times_applied', width: 60 },
  { title: '覆盖', dataIndex: 'times_overridden', key: 'times_overridden', width: 60 },
  { title: '置信度', dataIndex: 'confidence', key: 'confidence', width: 80 },
  { title: '距上次(天)', dataIndex: 'days_since_last_applied', key: 'days_since_last_applied', width: 90 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
]

const spcColumns = [
  { title: 'Agent', dataIndex: 'agent_id', key: 'agent_id', width: 60 },
  { title: '决策点', dataIndex: 'decision_point', key: 'decision_point', width: 120 },
  { title: '指标', dataIndex: 'metric_name', key: 'metric_name', width: 100 },
  { title: '当前值', dataIndex: 'current_value', key: 'current_value', width: 80 },
  { title: '均值(μ)', dataIndex: 'baseline_mean', key: 'baseline_mean', width: 80 },
  { title: 'UCL(μ+3σ)', dataIndex: 'ucl', key: 'ucl', width: 90 },
  { title: 'LCL(μ-3σ)', dataIndex: 'lcl', key: 'lcl', width: 90 },
  { title: '连续同侧', dataIndex: 'consecutive_same_side', key: 'consecutive_same_side', width: 80 },
  { title: '异常', key: 'is_out_of_control', width: 60 },
  { title: '下次重算', dataIndex: 'next_recalc_at', key: 'next_recalc_at', width: 160 },
]

const changeColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 160 },
  { title: '类型', dataIndex: 'source_type', key: 'source_type', width: 80 },
  { title: '变更内容', dataIndex: 'change_summary', key: 'change_summary', ellipsis: true },
  { title: '目标', dataIndex: 'target_type', key: 'target_type', width: 80 },
  { title: '字段', dataIndex: 'field_path', key: 'field_path', width: 80 },
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

function onChangeTableChange(pag: any) {
  changePagination.page = pag.current
  fetchChanges()
}

onMounted(() => { refreshAll() })
</script>
