<template>
  <div>
    <n-page-header subtitle="管理角色及权限绑定">
      <template #title>🔒 角色管理</template>
      <template #extra>
        <n-button type="primary" @click="showAddRoleModal">＋ 新增角色</n-button>
        <n-button @click="showAddPermissionModal">＋ 新增权限</n-button>
      </template>
    </n-page-header>

    <n-tabs type="line" default-value="roles" style="margin-top: 12px;">
      <n-tab-pane name="roles" tab="角色列表">
        <n-card :bordered="false">
          <n-data-table :columns="roleColumns" :data="roleData" :loading="roleLoading"
            :pagination="rolePagination" @update:page="onRolePageChange" />
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="permissions" tab="权限列表">
        <n-card :bordered="false">
          <n-data-table :columns="permColumns" :data="permData" :loading="permLoading"
            :pagination="permPagination" @update:page="onPermPageChange" />
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <!-- 新增/编辑角色弹窗 -->
    <n-modal v-model:show="roleModalVisible" :title="editingRoleId ? '编辑角色' : '新增角色'" preset="card" style="width: 500px;">
      <n-form ref="roleFormRef" :model="roleForm" :rules="roleRules" label-placement="left" label-width="80px">
        <n-form-item label="角色名称" path="name">
          <n-input v-model:value="roleForm.name" placeholder="请输入角色名称" />
        </n-form-item>
        <n-form-item label="角色编码" path="code">
          <n-input v-model:value="roleForm.code" placeholder="角色代码（如 admin）" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="roleForm.description" type="textarea" :rows="2" placeholder="角色描述（选填）" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="roleModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingRole" @click="handleRoleSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 绑定权限弹窗 -->
    <n-modal v-model:show="permBindVisible" title="绑定权限" preset="card" style="width: 600px;">
      <n-form label-placement="left" label-width="80px">
        <n-form-item label="角色">
          <n-input :value="bindingRole?.name" disabled />
        </n-form-item>
        <n-form-item label="权限">
          <n-select multiple v-model:value="bindPermissionIds" :options="allPermissionOptions" placeholder="请选择权限" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="permBindVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingBind" @click="handleBindPermissions">保存</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 新增/编辑权限弹窗 -->
    <n-modal v-model:show="permModalVisible" :title="editingPermId ? '编辑权限' : '新增权限'" preset="card" style="width: 500px;">
      <n-form ref="permFormRef" :model="permForm" :rules="permRules" label-placement="left" label-width="80px">
        <n-form-item label="权限名称" path="name">
          <n-input v-model:value="permForm.name" placeholder="权限名称" />
        </n-form-item>
        <n-form-item label="权限编码" path="code">
          <n-input v-model:value="permForm.code" placeholder="权限代码（如 product:create）" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="permForm.description" type="textarea" :rows="2" placeholder="权限描述（选填）" />
        </n-form-item>
        <n-form-item label="所属模块">
          <n-input v-model:value="permForm.module" placeholder="模块名（如 product）" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="permModalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submittingPerm" @click="handlePermSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import { rbacApi } from '@/api/modules/rbac'

const message = useMessage()
const dialog = useDialog()

// ========== 角色 ==========
const roleLoading = ref(false)
const roleData = ref<any[]>([])
const roleQuery = reactive({ name: '', page: 1, page_size: 20 })
const rolePagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { roleQuery.page = page; fetchRoles() },
})

const roleColumns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '角色名称', key: 'name' },
  { title: '编码', key: 'code' },
  { title: '描述', key: 'description', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 70,
    render: (row: any) => h(NTag, { type: row.status === 1 ? 'success' : 'default', size: 'small' },
      { default: () => row.status === 1 ? '启用' : '禁用' }),
  },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 220,
    render: (row: any) => h(NSpace, null, { default: () => [
      h(NButton, { size: 'small', ghost: true, onClick: () => showBindPermModal(row) }, { default: () => '绑定权限' }),
      h(NButton, { size: 'small', onClick: () => showEditRoleModal(row) }, { default: () => '编辑' }),
      h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDeleteRole(row) }, { default: () => '删除' }),
    ]}),
  },
]

// ========== 权限 ==========
const permLoading = ref(false)
const permData = ref<any[]>([])
const permQuery = reactive({ name: '', page: 1, page_size: 20 })
const permPagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { permQuery.page = page; fetchPerms() },
})

const permColumns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '权限名称', key: 'name' },
  { title: '编码', key: 'code' },
  { title: '描述', key: 'description', ellipsis: { tooltip: true } },
  { title: '模块', key: 'module', width: 100 },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 150,
    render: (row: any) => h(NSpace, null, { default: () => [
      h(NButton, { size: 'small', onClick: () => showEditPermModal(row) }, { default: () => '编辑' }),
      h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDeletePerm(row) }, { default: () => '删除' }),
    ]}),
  },
]

// ========== 角色弹窗 ==========
const roleModalVisible = ref(false)
const submittingRole = ref(false)
const editingRoleId = ref<number | null>(null)
const roleFormRef = ref<any>(null)
const roleForm = ref({ name: '', code: '', description: '' })
const roleRules = {
  name: { required: true, message: '请输入角色名称', trigger: 'blur' },
  code: { required: true, message: '请输入角色编码', trigger: 'blur' },
}

function showAddRoleModal() {
  editingRoleId.value = null
  roleForm.value = { name: '', code: '', description: '' }
  roleModalVisible.value = true
}

function showEditRoleModal(row: any) {
  editingRoleId.value = row.id
  roleForm.value = { name: row.name, code: row.code, description: row.description || '' }
  roleModalVisible.value = true
}

async function handleRoleSubmit() {
  try { await roleFormRef.value?.validate() } catch { return }
  submittingRole.value = true
  try {
    if (editingRoleId.value) {
      await rbacApi.updateRole(editingRoleId.value, roleForm.value)
      message.success('更新成功')
    } else {
      await rbacApi.createRole(roleForm.value)
      message.success('创建成功')
    }
    roleModalVisible.value = false
    fetchRoles()
  } catch (e: any) { message.error(e.message) }
  finally { submittingRole.value = false }
}

function handleDeleteRole(row: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除角色"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try { await rbacApi.deleteRole(row.id); message.success('删除成功'); fetchRoles() }
      catch (e: any) { message.error(e.message) }
    },
  })
}

// ========== 权限弹窗 ==========
const permModalVisible = ref(false)
const submittingPerm = ref(false)
const editingPermId = ref<number | null>(null)
const permFormRef = ref<any>(null)
const permForm = ref({ name: '', code: '', description: '', module: '' })
const permRules = {
  name: { required: true, message: '请输入权限名称', trigger: 'blur' },
  code: { required: true, message: '请输入权限编码', trigger: 'blur' },
}

function showAddPermissionModal() {
  editingPermId.value = null
  permForm.value = { name: '', code: '', description: '', module: '' }
  permModalVisible.value = true
}

function showEditPermModal(row: any) {
  editingPermId.value = row.id
  permForm.value = { name: row.name, code: row.code, description: row.description || '', module: row.module || '' }
  permModalVisible.value = true
}

async function handlePermSubmit() {
  try { await permFormRef.value?.validate() } catch { return }
  submittingPerm.value = true
  try {
    if (editingPermId.value) {
      await rbacApi.updatePermission(editingPermId.value, permForm.value)
      message.success('更新成功')
    } else {
      await rbacApi.createPermission(permForm.value)
      message.success('创建成功')
    }
    permModalVisible.value = false
    fetchPerms()
  } catch (e: any) { message.error(e.message) }
  finally { submittingPerm.value = false }
}

function handleDeletePerm(row: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除权限"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try { await rbacApi.deletePermission(row.id); message.success('删除成功'); fetchPerms() }
      catch (e: any) { message.error(e.message) }
    },
  })
}

// ========== 绑定权限 ==========
const permBindVisible = ref(false)
const submittingBind = ref(false)
const bindingRole = ref<any>(null)
const bindPermissionIds = ref<number[]>([])
const allPermissionOptions = ref<{ label: string; value: number }[]>([])

async function showBindPermModal(row: any) {
  bindingRole.value = row
  bindPermissionIds.value = row.permission_ids || []
  permBindVisible.value = true
  try {
    const res: any = await rbacApi.listPermissions({ page_size: 200 })
    allPermissionOptions.value = (res?.records || []).map((p: any) => ({
      label: `${p.name} (${p.code})`,
      value: p.id,
    }))
  } catch { /* ignore */ }
}

async function handleBindPermissions() {
  submittingBind.value = true
  try {
    await rbacApi.assignRolePermissions(bindingRole.value.id, bindPermissionIds.value)
    message.success('权限绑定成功')
    permBindVisible.value = false
    fetchRoles()
  } catch (e: any) { message.error(e.message) }
  finally { submittingBind.value = false }
}

// ========== 数据加载 ==========
async function fetchRoles() {
  roleLoading.value = true
  try {
    const res: any = await rbacApi.listRoles(roleQuery)
    roleData.value = res?.records || []
    rolePagination.itemCount = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { roleLoading.value = false }
}

async function fetchPerms() {
  permLoading.value = true
  try {
    const res: any = await rbacApi.listPermissions(permQuery)
    permData.value = res?.records || []
    permPagination.itemCount = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { permLoading.value = false }
}

function onRolePageChange(page: number) { roleQuery.page = page; fetchRoles() }
function onPermPageChange(page: number) { permQuery.page = page; fetchPerms() }

onMounted(() => {
  fetchRoles()
  fetchPerms()
})
</script>
