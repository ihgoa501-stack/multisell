<template>
  <div>
    <n-space justify="space-between" align="center" style="margin-bottom: 16px;">
      <h3 style="margin: 0;">🚚 物流管理</h3>
      <n-upload
        :show-file-list="false"
        accept=".xlsx,.csv"
        :custom-request="handleImportRules"
      >
        <n-button type="primary" :loading="importing">导入报价表</n-button>
      </n-upload>
    </n-space>

    <n-alert v-if="importResult" type="success" style="margin-bottom: 16px;" closable @close="importResult = null">
      导入完成：成功 {{ importResult.imported_rows }} 行，错误 {{ importResult.error_rows }} 行；
      新增供应商 {{ importResult.created_providers }} 个，渠道 {{ importResult.created_channels }} 个，
      区域 {{ importResult.created_zones }} 个，规则 {{ importResult.created_rules }} 条。
      <div v-if="importResult.errors?.length" style="margin-top: 8px;">
        <div v-for="err in importResult.errors" :key="err.row">
          第 {{ err.row }} 行：{{ err.message }}
        </div>
      </div>
    </n-alert>

    <!-- 供应商列表 -->
    <n-card title="物流供应商" style="margin-bottom: 16px;">
      <template #header-extra>
        <n-button size="small" type="primary" @click="showProviderModal = true">新增供应商</n-button>
      </template>
      <n-data-table :columns="providerColumns" :data="providers" :bordered="true" size="small" :loading="loadingProviders" />
    </n-card>

    <!-- 渠道列表 -->
    <n-card title="物流渠道" style="margin-bottom: 16px;">
      <template #header-extra>
        <n-button size="small" type="primary" @click="openChannelModal()">新增渠道</n-button>
      </template>
      <n-data-table :columns="channelColumns" :data="channels" :bordered="true" size="small" :loading="loadingChannels" />
    </n-card>

    <!-- 报价规则 -->
    <n-card title="报价规则" v-if="selectedChannelId">
      <template #header-extra>
        <n-button size="small" type="primary" @click="showRuleModal = true">新增规则</n-button>
      </template>
      <n-data-table :columns="ruleColumns" :data="rules" :bordered="true" size="small" :loading="loadingRules" />
    </n-card>

    <!-- Provider Modal -->
    <n-modal v-model:show="showProviderModal" title="物流供应商" preset="card" style="width: 500px;">
      <n-form :model="providerForm" label-placement="left" label-width="100">
        <n-form-item label="名称"><n-input v-model:value="providerForm.name" /></n-form-item>
        <n-form-item label="编码"><n-input v-model:value="providerForm.code" /></n-form-item>
        <n-form-item label="联系人"><n-input v-model:value="providerForm.contact" /></n-form-item>
        <n-form-item label="电话"><n-input v-model:value="providerForm.phone" /></n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showProviderModal = false">取消</n-button>
        <n-button type="primary" @click="saveProvider" style="margin-left: 8px;">保存</n-button>
      </template>
    </n-modal>

    <!-- Channel Modal -->
    <n-modal v-model:show="showChannelModal" title="物流渠道" preset="card" style="width: 600px;">
      <n-form :model="channelForm" label-placement="left" label-width="120">
        <n-form-item label="所属供应商">
          <n-select v-model:value="channelForm.provider_id" :options="providerOptions" filterable />
        </n-form-item>
        <n-form-item label="渠道名称"><n-input v-model:value="channelForm.name" /></n-form-item>
        <n-form-item label="编码"><n-input v-model:value="channelForm.code" /></n-form-item>
        <n-form-item label="抛重系数"><n-input-number v-model:value="channelForm.volumetric_divisor" :min="1" /></n-form-item>
        <n-form-item label="支持货品">
          <n-select v-model:value="channelForm.cargo_types" multiple :options="cargoOptions" />
        </n-form-item>
        <n-form-item label="最短时效"><n-input-number v-model:value="channelForm.estimated_delivery_min" :min="0" /></n-form-item>
        <n-form-item label="最长时效"><n-input-number v-model:value="channelForm.estimated_delivery_max" :min="0" /></n-form-item>
        <n-form-item label="币种"><n-input v-model:value="channelForm.currency" placeholder="CNY" /></n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showChannelModal = false">取消</n-button>
        <n-button type="primary" @click="saveChannel" style="margin-left: 8px;">保存</n-button>
      </template>
    </n-modal>

    <!-- Rule Modal -->
    <n-modal v-model:show="showRuleModal" title="报价规则" preset="card" style="width: 650px;">
      <n-form :model="ruleForm" label-placement="left" label-width="130">
        <n-form-item label="规则类型">
          <n-select v-model:value="ruleForm.rule_type" :options="ruleTypeOptions" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'fixed_plus_per_kg'" label="固定费">
          <n-input-number v-model:value="ruleForm.fixed_fee" :min="0" :step="0.1" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'fixed_plus_per_kg'" label="每公斤价格">
          <n-input-number v-model:value="ruleForm.per_kg_price" :min="0" :step="0.1" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="首重(kg)">
          <n-input-number v-model:value="ruleForm.first_kg" :min="0" :step="0.01" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="首重价格">
          <n-input-number v-model:value="ruleForm.first_price" :min="0" :step="0.1" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="续重单位(kg)">
          <n-input-number v-model:value="ruleForm.additional_kg" :min="0.01" :step="0.01" />
        </n-form-item>
        <n-form-item v-if="ruleForm.rule_type === 'first_weight_plus_increment'" label="续重单价">
          <n-input-number v-model:value="ruleForm.additional_price" :min="0" :step="0.1" />
        </n-form-item>
        <n-form-item label="最低收费">
          <n-input-number v-model:value="ruleForm.minimum_charge" :min="0" :step="0.1" placeholder="可选" clearable />
        </n-form-item>
        <n-form-item label="附加费(固定)">
          <n-input-number v-model:value="ruleForm.surcharge_fixed" :min="0" :step="0.1" />
        </n-form-item>
        <n-form-item label="燃油附加费%">
          <n-input-number v-model:value="ruleForm.fuel_surcharge_pct" :min="0" :step="0.5" />
        </n-form-item>
        <n-form-item label="取整增量(kg)">
          <n-input-number v-model:value="ruleForm.rounding_increment" :min="0.01" :step="0.01" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showRuleModal = false">取消</n-button>
        <n-button type="primary" @click="saveRule" style="margin-left: 8px;">保存</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, h } from 'vue'
import { useMessage } from 'naive-ui'
import { shippingApi } from '@/api/modules/shipping'

const message = useMessage()

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

// Provider columns
const providerColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '名称', key: 'name' },
  { title: '编码', key: 'code', width: 100 },
  { title: '联系人', key: 'contact', width: 100 },
  { title: '电话', key: 'phone', width: 120 },
  { title: '状态', key: 'status', width: 60, render: (r: any) => r.status === 1 ? '启用' : '禁用' },
  {
    title: '操作', width: 160,
    render: (row: any) => [
      h('span', { style: 'cursor:pointer;color:#2080f0;margin-right:8px;', onClick: () => editProvider(row) }, '编辑'),
      h('span', { style: 'cursor:pointer;color:#2080f0;', onClick: () => toggleProvider(row) }, row.status === 1 ? '禁用' : '启用'),
    ],
  },
]

// Channel columns
const channelColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '渠道名称', key: 'name' },
  { title: '供应商', key: 'provider_name', width: 120 },
  { title: '抛重系数', key: 'volumetric_divisor', width: 80 },
  { title: '货品类型', key: 'cargo_types', width: 120,
    render: (r: any) => (r.cargo_types || []).join(', ') },
  { title: '时效(天)', key: 'estimated_delivery_min', width: 80,
    render: (r: any) => r.estimated_delivery_min ? `${r.estimated_delivery_min}-${r.estimated_delivery_max}` : '-' },
  { title: '状态', key: 'status', width: 60, render: (r: any) => r.status === 1 ? '启用' : '禁用' },
  {
    title: '操作', width: 200,
    render: (row: any) => [
      h('span', { style: 'cursor:pointer;color:#2080f0;margin-right:8px;', onClick: () => selectChannel(row) }, '规则'),
      h('span', { style: 'cursor:pointer;color:#2080f0;margin-right:8px;', onClick: () => editChannel(row) }, '编辑'),
      h('span', { style: 'cursor:pointer;color:#2080f0;', onClick: () => toggleChannel(row) }, row.status === 1 ? '禁用' : '启用'),
    ],
  },
]

// Rule columns
const ruleColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '国家', key: 'country_code', width: 70, render: (r: any) => r.country_code || '全局' },
  { title: '规则类型', key: 'rule_type', width: 150 },
  { title: '固定费', key: 'fixed_fee', width: 70 },
  { title: '每公斤价', key: 'per_kg_price', width: 70 },
  { title: '首重/续重', key: 'first_kg', width: 120,
    render: (r: any) => r.rule_type === 'first_weight_plus_increment' ? `${r.first_kg}kg/${r.first_price} + ${r.additional_kg}kg/${r.additional_price}` : '-' },
  { title: '最低收费', key: 'minimum_charge', width: 80 },
  { title: '附加费', key: 'surcharge_fixed', width: 60 },
  { title: '燃油%', key: 'fuel_surcharge_pct', width: 60 },
  { title: '取整', key: 'rounding_increment', width: 60 },
  { title: '状态', key: 'status', width: 60, render: (r: any) => r.status === 1 ? '启用' : '禁用' },
  { title: '操作', width: 120,
    render: (row: any) => [
      h('span', { style: 'cursor:pointer;color:#2080f0;margin-right:8px;', onClick: () => deleteRule(row) }, '删除'),
    ],
  },
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
    const res: any = await shippingApi.importRules(options.file.file)
    importResult.value = res.data
    message.success('报价表导入完成')
    await loadProviders()
    await loadChannels()
    if (selectedChannelId.value) await loadRules(selectedChannelId.value)
    options.onFinish?.()
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
