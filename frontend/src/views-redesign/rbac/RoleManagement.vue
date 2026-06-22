<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">角色管理</h2>
        <div class="page-header-subtitle">管理角色及权限绑定</div>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showAddRoleModal">新增角色</a-button>
        <a-button style="margin-left: 8px;" @click="showAddPermissionModal">新增权限</a-button>
      </div>
    </div>

    <a-tabs v-model:activeKey="activeTab" style="margin-top: 12px;">
      <a-tab-pane key="roles" tab="角色列表">
        <a-card :bordered="false">
          <a-table
            :columns="roleColumns"
            :data-source="roleData"
            :loading="roleLoading"
            :pagination="rolePagination"
            row-key="id"
            @change="onRoleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
              </template>
              <template v-else-if="column.key === 'actions'">
                <a-space>
                  <a-button size="small" ghost @click="showBindPermModal(record)">绑定权限</a-button>
                  <a-button size="small" @click="showEditRoleModal(record)">编辑</a-button>
                  <a-button size="small" type="primary" danger ghost @click="handleDeleteRole(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="permissions" tab="权限列表">
        <a-card :bordered="false">
          <a-table
            :columns="permColumns"
            :data-source="permData"
            :loading="permLoading"
            :pagination="permPagination"
            row-key="id"
            @change="onPermTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'actions'">
                <a-space>
                  <a-button size="small" @click="showEditPermModal(record)">编辑</a-button>
                  <a-button size="small" type="primary" danger ghost @click="handleDeletePerm(record)">删除</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 新增/编辑角色弹窗 -->
    <a-modal v-model:open="roleModalVisible" :title="editingRoleId ? '编辑角色' : '新增角色'" :width="500" @ok="handleRoleSubmit" :confirm-loading="submittingRole">
      <a-form ref="roleFormRef" :model="roleForm" :rules="roleRules" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="角色名称" name="name">
          <a-input v-model:value="roleForm.name" placeholder="请输入角色名称" />
        </a-form-item>
        <a-form-item label="角色编码" name="code">
          <a-input v-model:value="roleForm.code" placeholder="角色代码（如 admin）" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="roleForm.description" :rows="2" placeholder="角色描述（选填）" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 绑定权限弹窗 -->
    <a-modal v-model:open="permBindVisible" title="绑定权限" :width="600" @ok="handleBindPermissions" :confirm-loading="submittingBind">
      <a-form :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="角色">
          <a-input :value="bindingRole?.name" disabled />
        </a-form-item>
        <a-form-item label="权限">
          <a-select v-model:value="bindPermissionIds" mode="multiple" :options="allPermissionOptions" placeholder="请选择权限" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 新增/编辑权限弹窗 -->
    <a-modal v-model:open="permModalVisible" :title="editingPermId ? '编辑权限' : '新增权限'" :width="500" @ok="handlePermSubmit" :confirm-loading="submittingPerm">
      <a-form ref="permFormRef" :model="permForm" :rules="permRules" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="权限名称" name="name">
          <a-input v-model:value="permForm.name" placeholder="权限名称" />
        </a-form-item>
        <a-form-item label="权限编码" name="code">
          <a-input v-model:value="permForm.code" placeholder="权限代码（如 product:create）" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="permForm.description" :rows="2" placeholder="权限描述（选填）" />
        </a-form-item>
        <a-form-item label="所属模块">
          <a-input v-model:value="permForm.module" placeholder="模块名（如 product）" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { rbacApi } from '@/api/modules/rbac'

const activeTab = ref('roles')

// ========== 角色 ==========
const roleLoading = ref(false)
const roleData = ref<any[]>([])
const roleQuery = reactive({ name: '', page: 1, page_size: 20 })
const rolePagination = reactive({
  current: 1, pageSize: 20, total: 0,
})

const roleColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '角色名称', dataIndex: 'name', key: 'name' },
  { title: '编码', dataIndex: 'code', key: 'code' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 220 },
]

// ========== 权限 ==========
const permLoading = ref(false)
const permData = ref<any[]>([])
const permQuery = reactive({ name: '', page: 1, page_size: 20 })
const permPagination = reactive({
  current: 1, pageSize: 20, total: 0,
})

const permColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '权限名称', dataIndex: 'name', key: 'name' },
  { title: '编码', dataIndex: 'code', key: 'code' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '模块', dataIndex: 'module', key: 'module', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 150 },
]

// ========== 角色弹窗 ==========
const roleModalVisible = ref(false)
const submittingRole = ref(false)
const editingRoleId = ref<number | null>(null)
const roleFormRef = ref<any>(null)
const roleForm = ref({ name: '', code: '', description: '' })
const roleRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色编码', trigger: 'blur' }],
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
  Modal.confirm({
    title: '确认删除',
    content: `确定删除角色"${row.name}"吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
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
  name: [{ required: true, message: '请输入权限名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入权限编码', trigger: 'blur' }],
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
  Modal.confirm({
    title: '确认删除',
    content: `确定删除权限"${row.name}"吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
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
    rolePagination.total = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { roleLoading.value = false }
}

async function fetchPerms() {
  permLoading.value = true
  try {
    const res: any = await rbacApi.listPermissions(permQuery)
    permData.value = res?.records || []
    permPagination.total = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { permLoading.value = false }
}

function onRoleTableChange(pag: any) {
  roleQuery.page = pag.current
  rolePagination.current = pag.current
  fetchRoles()
}

function onPermTableChange(pag: any) {
  permQuery.page = pag.current
  permPagination.current = pag.current
  fetchPerms()
}

onMounted(() => {
  fetchRoles()
  fetchPerms()
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
.page-header-extra {
  display: flex;
  align-items: center;
}
</style>
