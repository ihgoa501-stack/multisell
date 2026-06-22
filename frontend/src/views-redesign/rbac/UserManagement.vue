<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">用户管理</h2>
        <div class="page-header-subtitle">管理用户角色分配</div>
      </div>
    </div>

    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'role'">
            <a-tag v-for="name in (record.role_names || [])" :key="name" color="blue" style="margin-right: 4px;">{{ name }}</a-tag>
            <span v-if="!(record.role_names?.length)" style="color: var(--ant-color-text-tertiary);">无</span>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button size="small" type="primary" ghost @click="showAssignModal(record)">分配角色</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 分配角色弹窗 -->
    <a-modal v-model:open="modalVisible" title="分配角色" :width="600" @ok="handleAssign" :confirm-loading="submitting">
      <a-form :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="用户名">
          <a-input :value="currentUser?.username" disabled />
        </a-form-item>
        <a-form-item label="当前角色">
          <a-tag v-for="r in currentUser?.role_names || []" :key="r" color="blue" style="margin-right: 6px;">{{ r }}</a-tag>
          <span v-if="!(currentUser?.role_names?.length)" style="color: var(--ant-color-text-tertiary);">无角色</span>
        </a-form-item>
        <a-form-item label="分配角色">
          <a-select v-model:value="selectedRoleIds" mode="multiple" :options="roleOptions" placeholder="请选择角色" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { rbacApi } from '@/api/modules/rbac'

const loading = ref(false)
const data = ref<any[]>([])

const query = reactive({ username: '', page: 1, page_size: 20 })
const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '显示名称', dataIndex: 'display_name', key: 'display_name' },
  { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true },
  { title: '角色', dataIndex: 'role_names', key: 'role', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 120 },
]

// 分配角色弹窗状态
const modalVisible = ref(false)
const submitting = ref(false)
const currentUser = ref<any>(null)
const selectedRoleIds = ref<number[]>([])
const roleOptions = ref<{ label: string; value: number }[]>([])

async function fetchData() {
  loading.value = true
  try {
    const res: any = await rbacApi.listUsers(query)
    data.value = res?.records || []
    pagination.total = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function onTableChange(pag: any) {
  query.page = pag.current
  pagination.current = pag.current
  fetchData()
}

async function loadRoles() {
  try {
    const res: any = await rbacApi.listRoles({ page_size: 100 })
    roleOptions.value = (res?.records || []).map((r: any) => ({
      label: `${r.name} (${r.code})`,
      value: r.id,
    }))
  } catch { /* ignore */ }
}

async function showAssignModal(row: any) {
  currentUser.value = row
  selectedRoleIds.value = row.role_ids || []
  modalVisible.value = true
  await loadRoles()
}

async function handleAssign() {
  submitting.value = true
  try {
    await rbacApi.assignUserRoles(currentUser.value.id, selectedRoleIds.value)
    message.success('角色分配成功')
    modalVisible.value = false
    fetchData()
  } catch (e: any) { message.error(e.message) }
  finally { submitting.value = false }
}

onMounted(fetchData)
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
