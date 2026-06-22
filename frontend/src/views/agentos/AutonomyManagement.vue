<template>
  <div>
    <n-page-header subtitle="管理 Agent 自治等级升级与降级建议">
      <template #title>自治管理</template>
      <template #extra>
        <n-button size="small" @click="fetchCandidates" :loading="loading">刷新</n-button>
      </template>
    </n-page-header>

    <n-spin v-if="loading && !loaded" :show="true" style="margin-top: 40px;">
      <div style="text-align: center; padding: 40px;">加载中...</div>
    </n-spin>

    <n-result v-else-if="error" status="error" title="加载失败" :description="error">
      <template #footer>
        <n-button @click="fetchCandidates">重试</n-button>
      </template>
    </n-result>

    <template v-else>
      <!-- 建议升级卡片 -->
      <n-card title="升级建议" size="small" style="margin-top: 12px;">
        <template v-if="upgradeCandidates.length === 0">
          <n-empty description="暂无升级建议" style="padding: 20px 0;" />
        </template>
        <n-grid :cols="3" :x-gap="12" :y-gap="12">
          <n-grid-item v-for="c in upgradeCandidates" :key="c.agent_id">
            <AutonomyUpgradeCard
              :candidate="c"
              :actioning="actioning === c.agent_id"
              @upgrade="handleUpgrade"
              @downgrade="handleDowngrade"
            />
          </n-grid-item>
        </n-grid>
      </n-card>

      <!-- 自治等级说明 -->
      <n-card title="自治等级体系" size="small" style="margin-top: 12px;">
        <n-grid :cols="4" :x-gap="12" :y-gap="8">
          <n-grid-item><n-alert type="default" :bordered="false"><template #header>L0 观察</template>只读数据，无建议</n-alert></n-grid-item>
          <n-grid-item><n-alert type="info" :bordered="false"><template #header>L1 建议</template>生成建议，人执行</n-alert></n-grid-item>
          <n-grid-item><n-alert type="success" :bordered="false"><template #header>L2 半自主</template>低风险自动，高风险审批</n-alert></n-grid-item>
          <n-grid-item><n-alert type="warning" :bordered="false"><template #header>L3 全自主</template>边界内自动执行</n-alert></n-grid-item>
        </n-grid>
      </n-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { getAgentOSUpgradeCandidates, upgradeAgentLevel, downgradeAgentLevel } from '@/api/modules/agentos'
import type { AutonomyCandidate } from '@/api/modules/agentos'
import AutonomyUpgradeCard from '@/components/agentos/AutonomyUpgradeCard.vue'

const message = useMessage()

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
