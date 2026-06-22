<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h2 class="page-header-title">自治管理</h2>
        <p class="page-header-subtitle">管理 Agent 自治等级升级与降级建议</p>
      </div>
      <a-space>
        <a-button size="small" @click="fetchCandidates" :loading="loading">刷新</a-button>
      </a-space>
    </div>

    <a-spin v-if="loading && !loaded" style="margin-top: 40px; display: block;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </a-spin>

    <a-result v-else-if="error" status="error" title="加载失败" :sub-title="error">
      <template #extra>
        <a-button @click="fetchCandidates">重试</a-button>
      </template>
    </a-result>

    <template v-else>
      <!-- 建议升级卡片 -->
      <a-card title="升级建议" size="small" style="margin-top: 12px;">
        <template v-if="upgradeCandidates.length === 0">
          <a-empty description="暂无升级建议" style="padding: 20px 0;" />
        </template>
        <a-row :gutter="[12, 12]">
          <a-col :span="8" v-for="c in upgradeCandidates" :key="c.agent_id">
            <AutonomyUpgradeCard
              :candidate="c"
              :actioning="actioning === c.agent_id"
              @upgrade="handleUpgrade"
              @downgrade="handleDowngrade"
            />
          </a-col>
        </a-row>
      </a-card>

      <!-- 自治等级说明 -->
      <a-card title="自治等级体系" size="small" style="margin-top: 12px;">
        <a-row :gutter="[12, 8]">
          <a-col :span="6">
            <a-alert type="info" :bordered="false">
              <template #message>L0 观察</template>
              <template #description>只读数据，无建议</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="info" :bordered="false">
              <template #message>L1 建议</template>
              <template #description>生成建议，人执行</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="success" :bordered="false">
              <template #message>L2 半自主</template>
              <template #description>低风险自动，高风险审批</template>
            </a-alert>
          </a-col>
          <a-col :span="6">
            <a-alert type="warning" :bordered="false">
              <template #message>L3 全自主</template>
              <template #description>边界内自动执行</template>
            </a-alert>
          </a-col>
        </a-row>
      </a-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { getAgentOSUpgradeCandidates, upgradeAgentLevel, downgradeAgentLevel } from '@/api/modules/agentos'
import type { AutonomyCandidate } from '@/api/modules/agentos'
import AutonomyUpgradeCard from '@/components/agentos/AutonomyUpgradeCard.vue'

const loading = ref(false)
const loaded = ref(false)
const error = ref<string | null>(null)
const actioning = ref<string | null>(null)
const candidates = ref<AutonomyCandidate[]>([])

const upgradeCandidates = computed(() =>
  candidates.value.filter(c => c.suggested)
)

async function fetchCandidates() {
  loading.value = true
  error.value = null
  try {
    const res: any = await getAgentOSUpgradeCandidates()
    candidates.value = res?.data || []
    loaded.value = true
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '加载失败'
    message.error('加载升级候选失败')
  } finally {
    loading.value = false
  }
}

async function handleUpgrade(candidate: AutonomyCandidate) {
  if (!candidate.target_level) return
  actioning.value = candidate.agent_id
  try {
    await upgradeAgentLevel(candidate.agent_id, candidate.target_level)
    message.success(`${candidate.agent_name} 已升级至 ${candidate.target_level}`)
    await fetchCandidates()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '升级失败')
  } finally {
    actioning.value = null
  }
}

async function handleDowngrade(candidate: AutonomyCandidate) {
  if (!candidate.target_level) return
  actioning.value = candidate.agent_id
  try {
    await downgradeAgentLevel(candidate.agent_id, candidate.target_level)
    message.success(`${candidate.agent_name} 已降级至 ${candidate.target_level}`)
    await fetchCandidates()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '降级失败')
  } finally {
    actioning.value = null
  }
}

onMounted(fetchCandidates)
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
</style>
