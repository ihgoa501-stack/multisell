<template>
  <div>
    <a-space style="display: flex; margin-bottom: 16px;" align="center">
      <h3 style="margin: 0;">物流管理</h3>
      <template #split><span /></template>
      <a-upload
        :show-upload-list="false"
        accept=".xlsx,.csv"
        :custom-request="handleImportRules"
      >
        <a-button type="primary" :loading="importing">导入报价表</a-button>
      </a-upload>
    </a-space>

    <a-alert
      v-if="importResult"
      type="success"
      style="margin-bottom: 16px;"
      closable
      @close="importResult = null"
    >
      <template #message>
        导入完成：成功 {{ importResult.imported_rows }} 行，错误 {{ importResult.error_rows }} 行；
        新增供应商 {{ importResult.created_providers }} 个，渠道 {{ importResult.created_channels }} 个，
        区域 {{ importResult.created_zones }} 个，规则 {{ importResult.created_rules }} 条。
      </template>
      <template #description v-if="importResult.errors?.length">
        <div v-for="err in importResult.errors" :key="err.row">
          第 {{ err.row }} 行：{{ err.message }}
        </div>
      </template>
    </a-alert>

    <!-- 供应商列表 -->
    <a-card title="物流供应商" style="margin-bottom: 16px;">
      <template #extra>
        <a-button size="small" type="primary" @click="showProviderModal = true">新增供应商</a-button>
      </template>
      <a-table
        :columns="providerColumns"
        :data-source="providers"
        :pagination="false"
        :loading="loadingProviders"
        size="small"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'status'">
            {{ record.status === 1 ? '启用' : '禁用' }}
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <a @click="editProvider(record)" style="margin-right: 8px;">编辑</a>
            <a @click="toggleProvider(record)">{{ record.status === 1 ? '禁用' : '启用' }}</a>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 渠道列表 -->
    <a-card title="物流渠道" style="margin-bottom: 16px;">
      <template #extra>
        <a-button size="small" type="primary" @click="openChannelModal()">新增渠道</a-button>
      </template>
      <a-table
        :columns="channelColumns"
        :data-source="channels"
        :pagination="false"
        :loading="loadingChannels"
        size="small"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'cargo_types'">
            {{ (record.cargo_types || []).join(', ') }}
          </template>
          <template v-else-if="column.dataIndex === 'estimated_delivery_min'">
            {{ record.estimated_delivery_min ? `${record.estimated_delivery_min}-${record.estimated_delivery_max}` : '-' }}
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            {{ record.status === 1 ? '启用' : '禁用' }}
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <a @click="selectChannel(record)" style="margin-right: 8px;">规则</a>
            <a @click="editChannel(record)" style="margin-right: 8px;">编辑</a>
            <a @click="toggleChannel(record)">{{ record.status === 1 ? '禁用' : '启用' }}</a>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 报价规则 -->
    <a-card title="报价规则" v-if="selectedChannelId">
      <template #extra>
        <a-button size="small" type="primary" @click="showRuleModal = true">新增规则</a-button>
      </template>
      <a-table
        :columns="ruleColumns"
        :data-source="rules"
        :pagination="false"
        :loading="loadingRules"
        size="small"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'country_code'">
            {{ record.country_code || '全局' }}
          </template>
          <template v-else-if="column.dataIndex === 'first_kg'">
            {{ record.rule_type === 'first_weight_plus_increment' ? `${record.first_kg}kg/${record.first_price} + ${record.additional_kg}kg/${record.additional_price}` : '-' }}
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            {{ record.status === 1 ? '启用' : '禁用' }}
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <a-popconfirm title="确认删除该规则？" @confirm="deleteRule(record)">
              <a style="color: var(--ant-color-error);">删除</a>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Provider Modal -->
    <a-modal v-model:open="showProviderModal" title="物流供应商" :footer="null" style="width: 500px;">
      <a-form :model="providerForm" layout="horizontal" :label-col="{ style: { width: '100px' } }">
        <a-form-item label="名称"><a-input v-model:value="providerForm.name" /></a-form-item>
        <a-form-item label="编码"><a-input v-model:value="providerForm.code" /></a-form-item>
        <a-form-item label="联系人"><a-input v-model:value="providerForm.contact" /></a-form-item>
        <a-form-item label="电话"><a-input v-model:value="providerForm.phone" /></a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="showProviderModal = false">取消</a-button>
        <a-button type="primary" @click="saveProvider">保存</a-button>
      </template>
    </a-modal>

    <!-- Channel Modal -->
    <a-modal v-model:open="showChannelModal" title="物流渠道" :footer="null" style="width: 600px;">
      <a-form :model="channelForm" layout="horizontal" :label-col="{ style: { width: '120px' } }">
        <a-form-item label="所属供应商">
          <a-select
            v-model:value="channelForm.provider_id"
            :options="providerOptions"
            show-search
            :filter-option="filterOption"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="渠道名称"><a-input v-model:value="channelForm.name" /></a-form-item>
        <a-form-item label="编码"><a-input v-model:value="channelForm.code" /></a-form-item>
        <a-form-item label="抛重系数"><a-input-number v-model:value="channelForm.volumetric_divisor" :min="1" style="width: 100%;" /></a-form-item>
        <a-form-item label="支持货品">
          <a-select v-model:value="channelForm.cargo_types" mode="multiple" :options="cargoOptions" />
        </a-form-item>
        <a-form-item label="最短时效"><a-input-number v-model:value="channelForm.estimated_delivery_min" :min="0" style="width: 100%;" /></a-form-item>
        <a-form-item label="最长时效"><a-input-number v-model:value="channelForm.estimated_delivery_max" :min="0" style="width: 100%;" /></a-form-item>
        <a-form-item label="币种"><a-input v-model:value="channelForm.currency" placeholder="CNY" /></a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="showChannelModal = false">取消</a-button>
        <a-button type="primary" @click="saveChannel">保存</a-button>
      </template>
    </a-modal>

    <!-- Rule Modal -->
    <a-modal v-model:open="showRuleModal" title="报价规则" :footer="null" style="width: 650px;">
      <a-form :model="ruleForm" layout="horizontal" :label-col="{ style: { width: '130px' } }">
        <a-form-item label="规则类型">
          <a-select v-model:value="ruleForm.rule_type" :options="ruleTypeOptions" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'fixed_plus_per_kg'" label="固定费">
          <a-input-number v-model:value="ruleForm.fixed_fee" :min="0" :step="0.1" style="width: 100%;" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'fixed_plus_per_kg'" label="每公斤价格">
          <a-input-number v-model:value="ruleForm.per_kg_price" :min="0" :step="0.1" style="width: 100%;" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="首重(kg)">
          <a-input-number v-model:value="ruleForm.first_kg" :min="0" :step="0.01" style="width: 100%;" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="首重价格">
          <a-input-number v-model:value="ruleForm.first_price" :min="0" :step="0.1" style="width: 100%;" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="续重单位(kg)">
          <a-input-number v-model:value="ruleForm.additional_kg" :min="0.01" :step="0.01" style="width: 100%;" />
        </a-form-item>
        <a-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="续重单价">
          <a-input-number v-model:value="ruleForm.additional_price" :min="0" :step="0.1" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="最低收费">
          <a-input-number v-model:value="ruleForm.minimum_charge" :min="0" :step="0.1" placeholder="可选" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="附加费(固定)">
          <a-input-number v-model:value="ruleForm.surcharge_fixed" :min="0" :step="0.1" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="燃油附加费%">
          <a-input-number v-model:value="ruleForm.fuel_surcharge_pct" :min="0" :step="0.5" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="取整增量(kg)">
          <a-input-number v-model:value="ruleForm.rounding_increment" :min="0.01" :step="0.01" style="width: 100%;" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-button @click="showRuleModal = false">取消</a-button>
        <a-button type="primary" @click="saveRule">保存</a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { shippingApi } from '@/api/modules/shipping'

const loadingProviders = ref(false)
const loadingChannels = ref(false)
const loadingRules = ref(false)
const importing = ref(false)
const providers = ref<any[]>([])
const channels = ref<any[]>([])
const rules = ref<any[]>([])
const importResult = ref<any | null>(null)
const selectedChannelId = ref<number | null>(null)

const showProviderModal = ref(false)
const showChannelModal = ref(false)
const showRuleModal = ref(false)

const providerForm = reactive({ name: '', code: '', contact: '', phone: '' })
const channelForm = reactive({
  provider_id: null as number | null, name: '', code: '', volumetric_divisor: 6000,
  cargo_types: ['normal'] as string[], estimated_delivery_min: null as number | null,
  estimated_delivery_max: null as number | null, currency: 'CNY',
})
const ruleForm = reactive({
  rule_type: 'fixed_plus_per_kg', fixed_fee: 0, per_kg_price: 0,
  first_kg: 0.1, first_price: 0, additional_kg: 0.1, additional_price: 0,
  minimum_charge: null as number | null, surcharge_fixed: 0,
  fuel_surcharge_pct: 0, rounding_increment: 0.1,
})

const cargoOptions = [
  { label: '普通', value: 'normal' }, { label: '带电池', value: 'battery' },
  { label: '液体', value: 'liquid' }, { label: '敏感品', value: 'sensitive' },
]
const ruleTypeOptions = [
  { label: '固定费+每公斤', value: 'fixed_plus_per_kg' },
  { label: '首重+续重', value: 'first_weight_plus_increment' },
]
const providerOptions = computed(() => providers.value.map((p: any) => ({
  label: p.name, value: p.id,
})))

function filterOption(input: string, option: any) {
  return (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
}

// Provider columns
const providerColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '编码', dataIndex: 'code', key: 'code', width: 100 },
  { title: '联系人', dataIndex: 'contact', key: 'contact', width: 100 },
  { title: '电话', dataIndex: 'phone', key: 'phone', width: 120 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 60 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
]

// Channel columns
const channelColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '渠道名称', dataIndex: 'name', key: 'name' },
  { title: '供应商', dataIndex: 'provider_name', key: 'provider_name', width: 120 },
  { title: '抛重系数', dataIndex: 'volumetric_divisor', key: 'volumetric_divisor', width: 80 },
  { title: '货品类型', dataIndex: 'cargo_types', key: 'cargo_types', width: 120 },
  { title: '时效(天)', dataIndex: 'estimated_delivery_min', key: 'estimated_delivery_min', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 60 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
]

// Rule columns
const ruleColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '国家', dataIndex: 'country_code', key: 'country_code', width: 70 },
  { title: '规则类型', dataIndex: 'rule_type', key: 'rule_type', width: 150 },
  { title: '固定费', dataIndex: 'fixed_fee', key: 'fixed_fee', width: 70 },
  { title: '每公斤价', dataIndex: 'per_kg_price', key: 'per_kg_price', width: 70 },
  { title: '首重/续重', dataIndex: 'first_kg', key: 'first_kg', width: 120 },
  { title: '最低收费', dataIndex: 'minimum_charge', key: 'minimum_charge', width: 80 },
  { title: '附加费', dataIndex: 'surcharge_fixed', key: 'surcharge_fixed', width: 60 },
  { title: '燃油%', dataIndex: 'fuel_surcharge_pct', key: 'fuel_surcharge_pct', width: 60 },
  { title: '取整', dataIndex: 'rounding_increment', key: 'rounding_increment', width: 60 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 60 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 120 },
]

async function loadProviders() {
  loadingProviders.value = true
  try {
    const res: any = await shippingApi.listProviders()
    if (res.code === 200) providers.value = res.data
  } finally { loadingProviders.value = false }
}

async function loadChannels() {
  loadingChannels.value = true
  try {
    const res: any = await shippingApi.listChannels()
    if (res.code === 200) channels.value = res.data
  } finally { loadingChannels.value = false }
}

async function loadRules(channelId: number) {
  loadingRules.value = true
  try {
    const res: any = await shippingApi.listRules(channelId)
    if (res.code === 200) rules.value = res.data
  } finally { loadingRules.value = false }
}

async function saveProvider() {
  const res: any = await shippingApi.createProvider({ ...providerForm })
  if (res.code === 200) {
    message.success('创建成功')
    showProviderModal.value = false
    Object.assign(providerForm, { name: '', code: '', contact: '', phone: '' })
    await loadProviders()
  } else { message.error(res.message) }
}

async function saveChannel() {
  const res: any = await shippingApi.createChannel({ ...channelForm })
  if (res.code === 200) {
    message.success('创建成功')
    showChannelModal.value = false
    await loadChannels()
  } else { message.error(res.message) }
}

async function saveRule() {
  if (!selectedChannelId.value) return
  const res: any = await shippingApi.createRule(selectedChannelId.value, { ...ruleForm })
  if (res.code === 200) {
    message.success('创建成功')
    showRuleModal.value = false
    await loadRules(selectedChannelId.value)
  } else { message.error(res.message) }
}

function editProvider(row: any) {
  Object.assign(providerForm, { name: row.name, code: row.code, contact: row.contact || '', phone: row.phone || '' })
  showProviderModal.value = true
}

async function toggleProvider(row: any) {
  const newStatus = row.status === 1 ? 0 : 1
  const res: any = await shippingApi.updateProvider(row.id, { status: newStatus })
  if (res.code === 200) {
    message.success(newStatus === 1 ? '已启用' : '已禁用')
    await loadProviders()
  }
}

function editChannel(row: any) {
  Object.assign(channelForm, {
    provider_id: row.provider_id, name: row.name, code: row.code || '',
    volumetric_divisor: row.volumetric_divisor, cargo_types: row.cargo_types || ['normal'],
    estimated_delivery_min: row.estimated_delivery_min, estimated_delivery_max: row.estimated_delivery_max,
    currency: row.currency || 'CNY',
  })
  showChannelModal.value = true
}

async function toggleChannel(row: any) {
  const newStatus = row.status === 1 ? 0 : 1
  const res: any = await shippingApi.updateChannel(row.id, { status: newStatus })
  if (res.code === 200) {
    message.success(newStatus === 1 ? '已启用' : '已禁用')
    await loadChannels()
  }
}

function selectChannel(row: any) {
  selectedChannelId.value = row.id
  loadRules(row.id)
}

async function deleteRule(row: any) {
  const res: any = await shippingApi.deleteRule(row.id)
  if (res.code === 200) {
    message.success('已删除')
    if (selectedChannelId.value) loadRules(selectedChannelId.value)
  }
}

async function handleImportRules(options: any) {
  importing.value = true
  try {
    const res: any = await shippingApi.importRules(options.file)
    importResult.value = res.data
    message.success('报价表导入完成')
    await loadProviders()
    await loadChannels()
    if (selectedChannelId.value) await loadRules(selectedChannelId.value)
    options.onSuccess?.()
  } catch (e: any) {
    message.error(e.message || '报价表导入失败')
    options.onError?.()
  } finally {
    importing.value = false
  }
}

function openChannelModal() {
  Object.assign(channelForm, {
    provider_id: null, name: '', code: '', volumetric_divisor: 6000,
    cargo_types: ['normal'], estimated_delivery_min: null, estimated_delivery_max: null, currency: 'CNY',
  })
  showChannelModal.value = true
}

onMounted(() => {
  loadProviders()
  loadChannels()
})
</script>
