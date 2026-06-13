<template>
  <div>
    <n-page-header subtitle="管理用户角色分配">
      <template #title>👥 用户管理</template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading"
        :pagination="pagination" @update:page="onPageChange" />
    </n-card>

    <!-- 分配角色弹窗 -->
    <n-modal v-model:show="modalVisible" title="分配角色" preset="card" style="width: 600px;">
      <n-form label-placement="left" label-width="80px">
        <n-form-item label="用户名">
          <n-input :value="currentUser?.username" disabled />
        </n-form-item>
        <n-form-item label="当前角色">
          <n-tag v-for="r in currentUser?.role_names || []" :key="r" type="info" style="margin-right: 6px;">{{ r }}</n-tag>
          <span v-if="!(currentUser?.role_names?.length)" style="color: #999;">无角色</span>
        </n-form-item>
        <n-form-item label="分配角色">
          <n-select multiple v-model:value="selectedRoleIds" :options="roleOptions" placeholder="请选择角色" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleAssign">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, reactive, onMounted } from 'vue'
import { NButton, NSpace, NTag, NSelect, useMessage, useDialog } from 'naive-ui'
import { rbacApi } from '@/api/modules/rbac'

const message = useMessage()
const loading = ref(false)
const data = ref<any[]>([])

const query = reactive({ username: '', page: 1, page_size: 20 })
const pagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { query.page = page; fetchData() },
})

const columns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '用户名', key: 'username' },
  { title: '显示名称', key: 'display_name' },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true } },
  { title: '角色', key: 'role', width: 80,
    render: (row: any) => {
      const tags = (row.role_names || []).map((name: string) =>
        h(NTag, { type: 'info', size: 'small', style: 'margin-right: 4px;' }, { default: () => name })
      )
      return tags.length ? h('span', null, tags) : h('span', { style: 'color: #999;' }, '无')
    },
  },
  { title: '状态', key: 'status', width: 70,
    render: (row: any) => h(NTag, { type: row.status === 1 ? 'success' : 'default', size: 'small' },
      { default: () => row.status === 1 ? '启用' : '禁用' }),
  },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 120,
    render: (row: any) => h(NButton, { size: 'small', type: 'primary', ghost: true, onClick: () => showAssignModal(row) },
      { default: () => '分配角色' }),
  },
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
    pagination.itemCount = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function onPageChange(page: number) { query.page = page; fetchData() }

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
