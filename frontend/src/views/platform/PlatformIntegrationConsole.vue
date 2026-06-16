<template>
  <div>
    <n-page-header subtitle="管理平台连接、类目映射与属性映射">
      <template #title>🔌 平台集成</template>
    </n-page-header>

    <n-tabs type="line" default-value="accounts" style="margin-top: 12px;">
      <!-- ── 账号管理 ── -->
      <n-tab-pane name="accounts" tab="平台账号">
        <n-card :bordered="false">
          <template #header-extra>
            <n-button type="primary" @click="showAddAccountModal">＋ 新增账号</n-button>
          </template>
          <n-data-table :columns="accountColumns" :data="accountData" :loading="accountLoading" />
        </n-card>
      </n-tab-pane>

      <!-- ── 类目映射 ── -->
      <n-tab-pane name="categories" tab="类目映射">
        <n-card :bordered="false">
          <template #header-extra>
            <n-button type="primary" @click="showAddCategoryModal">＋ 新增映射</n-button>
          </template>
          <n-data-table :columns="categoryColumns" :data="categoryData" :loading="categoryLoading" />
        </n-card>
      </n-tab-pane>

      <!-- ── 属性映射 ── -->
      <n-tab-pane name="attributes" tab="属性映射">
        <n-card :bordered="false">
          <template #header-extra>
            <n-button type="primary" @click="showAddAttributeModal">＋ 新增映射</n-button>
          </template>
          <n-data-table :columns="attributeColumns" :data="attributeData" :loading="attributeLoading" />
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <!-- ════════════════ 账号弹窗 ════════════════ -->
    <n-modal v-model:show="accountModalVisible" :title="editingAccountId ? '编辑账号' : '新增账号'" preset="card" style="width: 550px;">
      <n-form ref="accountFormRef" :model="accountForm" :rules="accountRules" label-placement="left" label-width="100px">
        <n-form-item label="平台" path="platform_id">
          <n-select v-model:value="accountForm.platform_id" :options="platformOptions" placeholder="选择平台" filterable />
        </n-form-item>
        <n-form-item label="适配器" path="adapter_code">
          <n-select v-model:value="accountForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" filterable />
        </n-form-item>
        <n-form-item label="账号名称" path="account_name">
          <n-input v-model:value="accountForm.account_name" placeholder="如: Ozon 主店铺" />
        </n-form-item>
        <n-form-item label="凭据">
          <n-dynamic-input v-model:value="accountForm.credentials" :on-create="() => ({ key: '', value: '' })" preset="pair" key-placeholder="key" value-placeholder="value" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="accountModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingAccount" @click="handleAccountSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- ════════════════ 类目映射弹窗 ════════════════ -->
    <n-modal v-model:show="categoryModalVisible" title="新增类目映射" preset="card" style="width: 550px;">
      <n-form ref="categoryFormRef" :model="categoryForm" :rules="categoryRules" label-placement="left" label-width="100px">
        <n-form-item label="平台" path="platform_id">
          <n-select v-model:value="categoryForm.platform_id" :options="platformOptions" placeholder="选择平台" filterable />
        </n-form-item>
        <n-form-item label="适配器" path="adapter_code">
          <n-select v-model:value="categoryForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" filterable />
        </n-form-item>
        <n-form-item label="本地类目" path="local_category_id">
          <n-select v-model:value="categoryForm.local_category_id" :options="localCategoryOptions" placeholder="选择本地类目" filterable />
        </n-form-item>
        <n-form-item label="平台类目ID" path="platform_category_id">
          <n-input v-model:value="categoryForm.platform_category_id" placeholder="如: ozon_cat_100" />
        </n-form-item>
        <n-form-item label="平台类目名">
          <n-input v-model:value="categoryForm.platform_category_name" placeholder="选填" />
        </n-form-item>
        <n-form-item label="类目路径">
          <n-input v-model:value="categoryForm.platform_category_path" placeholder="选填，如 电子产品 > 手机" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="categoryModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingCategory" @click="handleCategorySubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- ════════════════ 属性映射弹窗 ════════════════ -->
    <n-modal v-model:show="attributeModalVisible" title="新增属性映射" preset="card" style="width: 550px;">
      <n-form ref="attributeFormRef" :model="attributeForm" :rules="attributeRules" label-placement="left" label-width="100px">
        <n-form-item label="平台" path="platform_id">
          <n-select v-model:value="attributeForm.platform_id" :options="platformOptions" placeholder="选择平台" filterable />
        </n-form-item>
        <n-form-item label="适配器" path="adapter_code">
          <n-select v-model:value="attributeForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" filterable />
        </n-form-item>
        <n-form-item label="本地属性" path="local_attribute">
          <n-input v-model:value="attributeForm.local_attribute" placeholder="如: 品牌" />
        </n-form-item>
        <n-form-item label="平台属性" path="platform_attribute">
          <n-input v-model:value="attributeForm.platform_attribute" placeholder="如: Brand" />
        </n-form-item>
        <n-form-item label="默认值">
          <n-input v-model:value="attributeForm.default_value" placeholder="选填" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="attributeModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingAttribute" @click="handleAttributeSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import {
  listAdapters,
  listAccounts,
  createAccount,
  updateAccount,
  testAccount,
  listCategoryMappings,
  createCategoryMapping,
  listAttributeMappings,
  createAttributeMapping,
} from '@/api/modules/platformIntegration'

const message = useMessage()
const dialog = useDialog()

// ── Shared options ──────────────────────────────────────────────────────

const adapterOptions = ref<{ label: string; value: string }[]>([])
const platformOptions = ref<{ label: string; value: number }[]>([])
const localCategoryOptions = ref<{ label: string; value: number }[]>([])

async function loadOptions() {
  try {
    const [adapterRes, platformRes, catRes] = await Promise.all([
      listAdapters(),
      (await import('@/api')).platformApi.list(),
      (await import('@/api')).categoryApi?.list?.() ?? Promise.resolve({ data: [] }),
    ])
    adapterOptions.value = (adapterRes as any).data?.map((a: any) => ({ label: a.display_name, value: a.adapter_code })) ?? []
    platformOptions.value = (platformRes as any).data?.map((p: any) => ({ label: p.name, value: p.id })) ?? []
    localCategoryOptions.value = (catRes as any).data?.map((c: any) => ({ label: c.name, value: c.id })) ?? []
  } catch { /* ignore */ }
}

// ── Accounts ────────────────────────────────────────────────────────────

const accountLoading = ref(false)
const accountData = ref<any[]>([])

const adapterCodeMap = ref<Record<string, string>>({})

const accountColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '账号名称', key: 'account_name' },
  { title: '平台', key: 'platform_name' },
  {
    title: '适配器',
    key: 'adapter_code',
    render: (row: any) => adapterCodeMap.value[row.adapter_code] || row.adapter_code,
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row: any) => {
      const m: Record<string, any> = { draft: 'default', active: 'success', disabled: 'warning' }
      const labels: Record<string, string> = { draft: '草稿', active: '启用', disabled: '禁用' }
      return h(NTag, { type: m[row.status] ?? 'default', size: 'small' }, { default: () => labels[row.status] ?? row.status })
    },
  },
  {
    title: '操作',
    width: 220,
    render: (row: any) => h(NSpace, null, { default: () => [
      h(NButton, { size: 'small', ghost: true, onClick: () => handleTestAccount(row) }, { default: (): any => '测试连接' }),
      h(NButton, { size: 'small', onClick: () => showEditAccountModal(row) }, { default: () => '编辑' }),
    ]}),
  },
]

async function fetchAccounts() {
  accountLoading.value = true
  try {
    const res: any = await listAccounts()
    accountData.value = res.data ?? []
  } catch (e: any) { message.error(e.message) }
  finally { accountLoading.value = false }
}

const accountModalVisible = ref(false)
const submittingAccount = ref(false)
const editingAccountId = ref<number | null>(null)
const accountFormRef = ref<any>(null)
const accountForm = ref<any>({ platform_id: null, adapter_code: '', account_name: '', credentials: [] })
const accountRules = {
  platform_id: { required: true, message: '请选择平台', trigger: 'blur' },
  adapter_code: { required: true, message: '请选择适配器', trigger: 'blur' },
  account_name: { required: true, message: '请输入账号名称', trigger: 'blur' },
}

function showAddAccountModal() {
  editingAccountId.value = null
  accountForm.value = { platform_id: null, adapter_code: '', account_name: '', credentials: [] }
  accountModalVisible.value = true
}

function showEditAccountModal(row: any) {
  editingAccountId.value = row.id
  accountForm.value = {
    platform_id: row.platform_id,
    adapter_code: row.adapter_code,
    account_name: row.account_name,
    credentials: [],
  }
  accountModalVisible.value = true
}

async function handleAccountSubmit() {
  try { await accountFormRef.value?.validate() } catch { return }
  submittingAccount.value = true
  try {
    const creds: Record<string, string> = {}
    for (const pair of accountForm.value.credentials) {
      if (pair.key) creds[pair.key] = pair.value
    }
    const payload: any = {
      platform_id: accountForm.value.platform_id,
      adapter_code: accountForm.value.adapter_code,
      account_name: accountForm.value.account_name,
    }
    if (Object.keys(creds).length) payload.credentials = creds

    if (editingAccountId.value) {
      const upd: any = { account_name: accountForm.value.account_name }
      if (Object.keys(creds).length) upd.credentials = creds
      await updateAccount(editingAccountId.value, upd)
      message.success('更新成功')
    } else {
      await createAccount(payload)
      message.success('创建成功')
    }
    accountModalVisible.value = false
    fetchAccounts()
  } catch (e: any) { message.error(e.message) }
  finally { submittingAccount.value = false }
}

async function handleTestAccount(row: any) {
  try {
    const res: any = await testAccount(row.id)
    const r = res.data ?? res
    message.success(`连接${r.success ? '成功' : '失败'}: ${r.message}`)
  } catch (e: any) { message.error(e.message) }
}

// ── Category Mappings ───────────────────────────────────────────────────

const categoryLoading = ref(false)
const categoryData = ref<any[]>([])

const categoryColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '平台', key: 'platform_name' },
  { title: '适配器', key: 'adapter_code' },
  { title: '本地类目', key: 'local_category_name' },
  { title: '平台类目ID', key: 'platform_category_id' },
  { title: '平台类目名', key: 'platform_category_name' },
  { title: '类目路径', key: 'platform_category_path', ellipsis: { tooltip: true } },
]

async function fetchCategories() {
  categoryLoading.value = true
  try {
    const res: any = await listCategoryMappings()
    categoryData.value = res.data ?? []
  } catch (e: any) { message.error(e.message) }
  finally { categoryLoading.value = false }
}

const categoryModalVisible = ref(false)
const submittingCategory = ref(false)
const categoryFormRef = ref<any>(null)
const categoryForm = ref<any>({
  platform_id: null, adapter_code: '', local_category_id: null,
  platform_category_id: '', platform_category_name: '', platform_category_path: '',
})
const categoryRules = {
  platform_id: { required: true, message: '请选择平台', trigger: 'blur' },
  adapter_code: { required: true, message: '请选择适配器', trigger: 'blur' },
  local_category_id: { required: true, message: '请选择本地类目', trigger: 'blur' },
  platform_category_id: { required: true, message: '请输入平台类目ID', trigger: 'blur' },
}

function showAddCategoryModal() {
  categoryForm.value = {
    platform_id: null, adapter_code: '', local_category_id: null,
    platform_category_id: '', platform_category_name: '', platform_category_path: '',
  }
  categoryModalVisible.value = true
}

async function handleCategorySubmit() {
  try { await categoryFormRef.value?.validate() } catch { return }
  submittingCategory.value = true
  try {
    await createCategoryMapping(categoryForm.value)
    message.success('创建成功')
    categoryModalVisible.value = false
    fetchCategories()
  } catch (e: any) { message.error(e.message) }
  finally { submittingCategory.value = false }
}

// ── Attribute Mappings ──────────────────────────────────────────────────

const attributeLoading = ref(false)
const attributeData = ref<any[]>([])

const attributeColumns = [
  { title: 'ID', key: 'id', width: 60 },
  { title: '平台', key: 'platform_name' },
  { title: '适配器', key: 'adapter_code' },
  { title: '本地属性', key: 'local_attribute' },
  { title: '平台属性', key: 'platform_attribute' },
  { title: '默认值', key: 'default_value' },
]

async function fetchAttributes() {
  attributeLoading.value = true
  try {
    const res: any = await listAttributeMappings()
    attributeData.value = res.data ?? []
  } catch (e: any) { message.error(e.message) }
  finally { attributeLoading.value = false }
}

const attributeModalVisible = ref(false)
const submittingAttribute = ref(false)
const attributeFormRef = ref<any>(null)
const attributeForm = ref<any>({
  platform_id: null, adapter_code: '', local_attribute: '',
  platform_attribute: '', default_value: '',
})
const attributeRules = {
  platform_id: { required: true, message: '请选择平台', trigger: 'blur' },
  adapter_code: { required: true, message: '请选择适配器', trigger: 'blur' },
  local_attribute: { required: true, message: '请输入本地属性', trigger: 'blur' },
  platform_attribute: { required: true, message: '请输入平台属性', trigger: 'blur' },
}

function showAddAttributeModal() {
  attributeForm.value = {
    platform_id: null, adapter_code: '', local_attribute: '',
    platform_attribute: '', default_value: '',
  }
  attributeModalVisible.value = true
}

async function handleAttributeSubmit() {
  try { await attributeFormRef.value?.validate() } catch { return }
  submittingAttribute.value = true
  try {
    await createAttributeMapping(attributeForm.value)
    message.success('创建成功')
    attributeModalVisible.value = false
    fetchAttributes()
  } catch (e: any) { message.error(e.message) }
  finally { submittingAttribute.value = false }
}

// ── Init ────────────────────────────────────────────────────────────────

onMounted(async () => {
  await loadOptions()
  adapterCodeMap.value = Object.fromEntries(adapterOptions.value.map((o) => [o.value, o.label]))
  fetchAccounts()
  fetchCategories()
  fetchAttributes()
})
</script>
