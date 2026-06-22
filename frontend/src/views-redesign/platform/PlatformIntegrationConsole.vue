<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">平台集成</h2>
        <div class="page-header-subtitle">管理平台连接、类目映射与属性映射</div>
      </div>
    </div>

    <a-tabs v-model:activeKey="activeTab" style="margin-top: 12px;">
      <!-- ── 账号管理 ── -->
      <a-tab-pane key="accounts" tab="平台账号">
        <a-card :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showAddAccountModal">新增账号</a-button>
          </template>
          <a-table :columns="accountColumns" :data-source="accountData" :loading="accountLoading" row-key="id">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'adapter_code'">
                {{ adapterCodeMap[record.adapter_code] || record.adapter_code }}
              </template>
              <template v-else-if="column.key === 'status'">
                <a-tag :color="statusColorMap[record.status] ?? 'default'">{{ statusLabelMap[record.status] ?? record.status }}</a-tag>
              </template>
              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button size="small" ghost @click="handleTestAccount(record)">测试连接</a-button>
                  <a-button size="small" @click="showEditAccountModal(record)">编辑</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- ── 类目映射 ── -->
      <a-tab-pane key="categories" tab="类目映射">
        <a-card :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showAddCategoryModal">新增映射</a-button>
          </template>
          <a-table :columns="categoryColumns" :data-source="categoryData" :loading="categoryLoading" row-key="id" />
        </a-card>
      </a-tab-pane>

      <!-- ── 属性映射 ── -->
      <a-tab-pane key="attributes" tab="属性映射">
        <a-card :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showAddAttributeModal">新增映射</a-button>
          </template>
          <a-table :columns="attributeColumns" :data-source="attributeData" :loading="attributeLoading" row-key="id" />
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- ════════════════ 账号弹窗 ════════════════ -->
    <a-modal v-model:open="accountModalVisible" :title="editingAccountId ? '编辑账号' : '新增账号'" :width="550" @ok="handleAccountSubmit" :confirm-loading="submittingAccount">
      <a-form ref="accountFormRef" :model="accountForm" :rules="accountRules" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="平台" name="platform_id">
          <a-select v-model:value="accountForm.platform_id" :options="platformOptions" placeholder="选择平台" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="适配器" name="adapter_code">
          <a-select v-model:value="accountForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="账号名称" name="account_name">
          <a-input v-model:value="accountForm.account_name" placeholder="如: Ozon 主店铺" />
        </a-form-item>
        <a-form-item label="凭据">
          <div v-for="(cred, index) in accountForm.credentials" :key="index" style="display: flex; gap: 8px; margin-bottom: 8px;">
            <a-input v-model:value="cred.key" placeholder="key" style="flex: 1;" />
            <a-input v-model:value="cred.value" placeholder="value" style="flex: 1;" />
            <a-button danger @click="removeCredential(index)">删除</a-button>
          </div>
          <a-button type="dashed" block @click="addCredential">+ 添加凭据</a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ════════════════ 类目映射弹窗 ════════════════ -->
    <a-modal v-model:open="categoryModalVisible" title="新增类目映射" :width="550" @ok="handleCategorySubmit" :confirm-loading="submittingCategory">
      <a-form ref="categoryFormRef" :model="categoryForm" :rules="categoryRules" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="平台" name="platform_id">
          <a-select v-model:value="categoryForm.platform_id" :options="platformOptions" placeholder="选择平台" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="适配器" name="adapter_code">
          <a-select v-model:value="categoryForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="本地类目" name="local_category_id">
          <a-select v-model:value="categoryForm.local_category_id" :options="localCategoryOptions" placeholder="选择本地类目" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="平台类目ID" name="platform_category_id">
          <a-input v-model:value="categoryForm.platform_category_id" placeholder="如: ozon_cat_100" />
        </a-form-item>
        <a-form-item label="平台类目名">
          <a-input v-model:value="categoryForm.platform_category_name" placeholder="选填" />
        </a-form-item>
        <a-form-item label="类目路径">
          <a-input v-model:value="categoryForm.platform_category_path" placeholder="选填，如 电子产品 > 手机" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ════════════════ 属性映射弹窗 ════════════════ -->
    <a-modal v-model:open="attributeModalVisible" title="新增属性映射" :width="550" @ok="handleAttributeSubmit" :confirm-loading="submittingAttribute">
      <a-form ref="attributeFormRef" :model="attributeForm" :rules="attributeRules" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="平台" name="platform_id">
          <a-select v-model:value="attributeForm.platform_id" :options="platformOptions" placeholder="选择平台" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="适配器" name="adapter_code">
          <a-select v-model:value="attributeForm.adapter_code" :options="adapterOptions" placeholder="选择适配器" show-search :filter-option="filterOption" />
        </a-form-item>
        <a-form-item label="本地属性" name="local_attribute">
          <a-input v-model:value="attributeForm.local_attribute" placeholder="如: 品牌" />
        </a-form-item>
        <a-form-item label="平台属性" name="platform_attribute">
          <a-input v-model:value="attributeForm.platform_attribute" placeholder="如: Brand" />
        </a-form-item>
        <a-form-item label="默认值">
          <a-input v-model:value="attributeForm.default_value" placeholder="选填" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
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

const activeTab = ref('accounts')

// ── Shared options ──────────────────────────────────────────────────────

const adapterOptions = ref<{ label: string; value: string }[]>([])
const platformOptions = ref<{ label: string; value: number }[]>([])
const localCategoryOptions = ref<{ label: string; value: number }[]>([])

const statusColorMap: Record<string, string> = { draft: 'default', active: 'success', disabled: 'warning' }
const statusLabelMap: Record<string, string> = { draft: '草稿', active: '启用', disabled: '禁用' }

function filterOption(input: string, option: any) {
  return (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
}

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
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '账号名称', dataIndex: 'account_name', key: 'account_name' },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name' },
  { title: '适配器', dataIndex: 'adapter_code', key: 'adapter_code' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'actions', width: 220 },
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
  platform_id: [{ required: true, message: '请选择平台', trigger: 'change' }],
  adapter_code: [{ required: true, message: '请选择适配器', trigger: 'change' }],
  account_name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
}

function addCredential() {
  accountForm.value.credentials.push({ key: '', value: '' })
}

function removeCredential(index: number) {
  accountForm.value.credentials.splice(index, 1)
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
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name' },
  { title: '适配器', dataIndex: 'adapter_code', key: 'adapter_code' },
  { title: '本地类目', dataIndex: 'local_category_name', key: 'local_category_name' },
  { title: '平台类目ID', dataIndex: 'platform_category_id', key: 'platform_category_id' },
  { title: '平台类目名', dataIndex: 'platform_category_name', key: 'platform_category_name' },
  { title: '类目路径', dataIndex: 'platform_category_path', key: 'platform_category_path', ellipsis: true },
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
  platform_id: [{ required: true, message: '请选择平台', trigger: 'change' }],
  adapter_code: [{ required: true, message: '请选择适配器', trigger: 'change' }],
  local_category_id: [{ required: true, message: '请选择本地类目', trigger: 'change' }],
  platform_category_id: [{ required: true, message: '请输入平台类目ID', trigger: 'blur' }],
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
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '平台', dataIndex: 'platform_name', key: 'platform_name' },
  { title: '适配器', dataIndex: 'adapter_code', key: 'adapter_code' },
  { title: '本地属性', dataIndex: 'local_attribute', key: 'local_attribute' },
  { title: '平台属性', dataIndex: 'platform_attribute', key: 'platform_attribute' },
  { title: '默认值', dataIndex: 'default_value', key: 'default_value' },
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
  platform_id: [{ required: true, message: '请选择平台', trigger: 'change' }],
  adapter_code: [{ required: true, message: '请选择适配器', trigger: 'change' }],
  local_attribute: [{ required: true, message: '请输入本地属性', trigger: 'blur' }],
  platform_attribute: [{ required: true, message: '请输入平台属性', trigger: 'blur' }],
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

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 4px;
}
.page-header-content {
  flex: 1;
}
.page-header-title {
  margin: 0 0 4px 0;
  font-size: 20px;
  font-weight: 600;
}
.page-header-subtitle {
  color: var(--ant-color-text-secondary);
  font-size: 14px;
}
</style>
