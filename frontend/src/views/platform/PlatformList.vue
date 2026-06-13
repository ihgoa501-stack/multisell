<template>
  <div>
    <n-page-header subtitle="配置多电商平台API">
      <template #title>🌐 平台管理</template>
      <template #extra>
        <n-button type="primary" @click="showAddModal">＋ 添加平台</n-button>
      </template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="modalVisible" :title="editingId ? '编辑平台' : '添加平台'" preset="card" style="width: 550px;">
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="120px">
        <n-form-item label="平台名称" path="name">
          <n-input v-model:value="form.name" placeholder="如: Ozon, Shopee, Wildberries" />
        </n-form-item>
        <n-form-item label="平台代码" path="code">
          <n-input v-model:value="form.code" placeholder="如: ozon, shopee, wb" />
        </n-form-item>
        <n-form-item label="API地址">
          <n-input v-model:value="form.api_base_url" placeholder="API基础地址" />
        </n-form-item>
        <n-form-item label="Client ID">
          <n-input v-model:value="form.client_id" placeholder="平台分配的Client ID" />
        </n-form-item>
        <n-form-item label="API密钥">
          <n-input v-model:value="form.api_key" type="password" show-password-on="click" placeholder="平台API密钥" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="form.sort_order" :min="0" style="width: 100px;" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted } from 'vue'
import { NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import { platformApi } from '@/api'

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])

const columns = [
  { title: '排序', key: 'sort_order', width: 60 },
  { title: '平台名称', key: 'name' },
  { title: '代码', key: 'code', width: 80 },
  { title: 'API地址', key: 'api_base_url', ellipsis: { tooltip: true } },
  { title: 'Client ID', key: 'client_id', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 70, render: (row: any) => h(NTag, { type: row.status === 1 ? 'success' : 'default', size: 'small' }, { default: () => row.status === 1 ? '启用' : '禁用' }) },
  { title: '操作', width: 150, render: (row: any) => h(NSpace, null, { default: () => [
    h(NButton, { size: 'small', onClick: () => showEditModal(row) }, { default: () => '编辑' }),
    h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDelete(row) }, { default: () => '删除' }),
  ]})},
]

const modalVisible = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<any>(null)
const form = ref({ name: '', code: '', api_base_url: '', client_id: '', api_key: '', sort_order: 0 })
const rules = {
  name: { required: true, message: '请输入平台名称', trigger: 'blur' },
  code: { required: true, message: '请输入平台代码', trigger: 'blur' },
}

async function fetchData() {
  loading.value = true
  try {
    const res: any = await platformApi.list()
    data.value = res.data || []
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function showAddModal() {
  editingId.value = null
  form.value = { name: '', code: '', api_base_url: '', client_id: '', api_key: '', sort_order: 0 }
  modalVisible.value = true
}

function showEditModal(row: any) {
  editingId.value = row.id
  form.value = {
    name: row.name,
    code: row.code,
    api_base_url: row.api_base_url || '',
    client_id: row.client_id || '',
    api_key: '',
    sort_order: row.sort_order || 0,
  }
  modalVisible.value = true
}

async function handleSubmit() {
  try { await formRef.value?.validate() } catch { return }
  submitting.value = true
  try {
    const data: any = { ...form.value }
    if (!data.api_key) delete data.api_key
    if (editingId.value) {
      await platformApi.update(editingId.value, data)
      message.success('更新成功')
    } else {
      await platformApi.create(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    fetchData()
  } catch (e: any) { message.error(e.message) }
  finally { submitting.value = false }
}

function handleDelete(row: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除平台"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try { await platformApi.delete(row.id); message.success('删除成功'); fetchData() }
      catch (e: any) { message.error(e.message) }
    },
  })
}

onMounted(fetchData)
</script>
