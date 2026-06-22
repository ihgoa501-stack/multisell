<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h2 class="page-header-title">任务中心</h2>
        <p class="page-header-subtitle">统一处理 Agent 建议、异常、通知和待审批动作</p>
      </div>
      <a-space>
        <a-button size="small" @click="showCreateModal = true" type="primary">
          <template #icon><PlusOutlined /></template>
          新建提案
        </a-button>
        <a-button size="small" @click="fetchItems" :loading="loading">刷新</a-button>
      </a-space>
    </div>

    <!-- 筛选栏 -->
    <a-card size="small" style="margin-top: 12px;">
      <a-space wrap :size="[12, 8]">
        <div class="filter-group">
          <span class="filter-label">状态</span>
          <a-select
            v-model:value="filters.status"
            allow-clear
            :options="statusOptions"
            style="width: 120px;"
            placeholder="全部状态"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">优先级</span>
          <a-select
            v-model:value="filters.priority"
            allow-clear
            :options="priorityOptions"
            style="width: 120px;"
            placeholder="全部优先级"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">团队</span>
          <a-select
            v-model:value="filters.squad"
            allow-clear
            :options="squadOptions"
            style="width: 120px;"
            placeholder="全部团队"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">来源</span>
          <a-select
            v-model:value="filters.sourceType"
            allow-clear
            :options="sourceOptions"
            style="width: 130px;"
            placeholder="全部来源"
          />
        </div>
        <div class="filter-group">
          <span class="filter-label">需审批</span>
          <a-switch v-model:checked="filters.requiresApproval" />
        </div>
        <a-button type="primary" size="small" @click="applyFilters">筛选</a-button>
        <a-button size="small" @click="resetFilters">重置</a-button>
      </a-space>
    </a-card>

    <!-- 创建提案弹窗 -->
    <a-modal v-model:open="showCreateModal" title="新建动作提案" :width="640" :mask-closable="false" @ok="handleCreateProposal" :confirm-loading="submitting">
      <a-form :model="form" :rules="formRules" ref="formRef" layout="vertical">
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="动作类型" name="action_type">
              <a-select v-model:value="form.action_type" :options="actionTypeOptions" placeholder="选择动作类型" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="风险等级" name="risk_level">
              <a-select v-model:value="form.risk_level" :options="riskLevelOptions" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="小队" name="squad_id">
              <a-select v-model:value="form.squad_id" :options="squadOptions" placeholder="选择小队" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Agent" name="agent_id">
              <a-select v-model:value="form.agent_id" :options="agentOptions" placeholder="选择 Agent" allow-clear />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="业务对象类型" name="business_object_type">
              <a-input v-model:value="form.business_object_type" placeholder="如 sku / product" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="业务对象 ID" name="business_object_id">
              <a-input v-model:value="form.business_object_id" placeholder="如 SKU-100" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="标题" name="title">
          <a-input v-model:value="form.title" placeholder="提案标题" />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="form.description" :rows="2" placeholder="简要描述" />
        </a-form-item>
        <a-form-item label="执行参数 (JSON)" name="proposedPayloadJson">
          <a-textarea v-model:value="form.proposedPayloadJson" :rows="3" placeholder='{"sku_id": 1, "target_sale_price": 5000}' />
        </a-form-item>
        <a-space align="center">
          <a-form-item label="需审批">
            <a-switch v-model:checked="form.requires_approval" />
          </a-form-item>
          <a-form-item label="置信度">
            <div style="display: flex; align-items: center;">
              <a-slider v-model:value="form.confidence" :min="0" :max="1" :step="0.05" style="width: 160px;" />
              <span style="margin-left: 8px; min-width: 36px;">{{ (form.confidence * 100).toFixed(0) }}%</span>
            </div>
          </a-form-item>
        </a-space>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="showCreateModal = false">取消</a-button>
          <a-button type="primary" :loading="submitting" @click="handleCreateProposal">创建</a-button>
        </a-space>
      </template>
    </a-modal>

    <!-- 任务列表 -->
    <a-card size="small" style="margin-top: 12px;" :loading="loading">
      <template v-if="items.length === 0 && !loading">
        <a-empty description="暂无匹配的任务" style="padding: 40px 0;" />
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
      <div v-if="total > limit" style="margin-top: 16px; text-align: center;">
        <a-pagination
          :current="currentPage"
          :page-size="limit"
          :total="total"
          @change="onPageChange"
        />
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { getAgentOSWorkItems, createActionProposal } from '@/api/modules/agentos'
import type { AgentOSWorkItem, WorkItemQuery } from '@/api/modules/agentos'
import WorkItemCard from '@/components/agentos/WorkItemCard.vue'

const router = useRouter()

const loading = ref(false)
const items = ref<AgentOSWorkItem[]>([])
const total = ref(0)
const limit = ref(20)
const currentPage = ref(1)

// -- 创建提案弹窗 --
const showCreateModal = ref(false)
const submitting = ref(false)
const formRef = ref()

const defaultForm = {
  action_type: null as string | null,
  title: '',
  description: '',
  risk_level: 'medium',
  squad_id: null as string | null,
  agent_id: null as string | null,
  business_object_type: '',
  business_object_id: '',
  proposedPayloadJson: '{}',
  requires_approval: true,
  confidence: 0.8,
}
const form = reactive({ ...defaultForm })

const formRules = {
  action_type: { required: true, message: '请选择动作类型', trigger: 'change' },
  title: { required: true, message: '请输入标题', trigger: 'blur' },
}

const actionTypeOptions = [
  { label: '利润复核', value: 'profit_review' },
  { label: '库存分配', value: 'inventory_allocate' },
  { label: 'Listing 草稿', value: 'listing_draft' },
  { label: '经营日报', value: 'daily_report' },
  { label: '预警通知', value: 'notify' },
]

const riskLevelOptions = [
  { label: '低风险', value: 'low' },
  { label: '中风险', value: 'medium' },
  { label: '高风险', value: 'high' },
  { label: '严重', value: 'critical' },
]

const agentOptions = [
  { label: '选品助手 (A1)', value: 'A1' },
  { label: 'Listing 优化师 (A2)', value: 'A2' },
  { label: '广告顾问 (A3)', value: 'A3' },
  { label: '客服助手 (A4)', value: 'A4' },
  { label: '库存管家 (A5)', value: 'A5' },
  { label: '利润分析师 (A6)', value: 'A6' },
  { label: '合规检查员 (A7)', value: 'A7' },
  { label: '总控 (G1)', value: 'G1' },
  { label: '仓储专员 (G2)', value: 'G2' },
  { label: '折扣风控 (G3)', value: 'G3' },
]

function resetForm() {
  Object.assign(form, defaultForm)
}

async function handleCreateProposal() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    let payload: Record<string, any> = {}
    try {
      payload = JSON.parse(form.proposedPayloadJson || '{}')
    } catch {
      message.warning('执行参数 JSON 格式错误，使用空对象')
    }
    await createActionProposal({
      source_type: 'manual',
      source_id: null,
      agent_id: form.agent_id || undefined,
      squad_id: form.squad_id || undefined,
      action_type: form.action_type!,
      business_object_type: form.business_object_type || undefined,
      business_object_id: form.business_object_id || undefined,
      title: form.title,
      description: form.description || undefined,
      proposed_payload: payload,
      risk_level: form.risk_level as any,
      requires_approval: form.requires_approval,
      confidence: form.confidence,
    })
    message.success('提案已创建')
    showCreateModal.value = false
    resetForm()
    fetchItems()
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

const filters = reactive({
  status: null as string | null,
  priority: null as string | null,
  squad: null as string | null,
  sourceType: null as string | null,
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

const sourceOptions = [
  { label: '动作提案', value: 'action_proposal' },
  { label: 'Agent 动作', value: 'agent_action' },
  { label: '异常', value: 'exception' },
  { label: '通知', value: 'notification' },
  { label: '上架任务', value: 'listing_task' },
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
    if (filters.sourceType) query.source_type = filters.sourceType
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
  filters.sourceType = null
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
.filter-group {
  display: flex;
  align-items: center;
  gap: 6px;
}
.filter-label {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  white-space: nowrap;
}
</style>
