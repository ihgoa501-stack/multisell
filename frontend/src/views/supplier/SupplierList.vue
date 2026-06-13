<template>
  <div>
    <n-page-header subtitle="管理供应商信息">
      <template #title>🤝 供应商管理</template>
      <template #extra>
        <n-button type="primary" @click="showAddModal">＋ 新增供应商</n-button>
      </template>
    </n-page-header>

    <!-- 搜索 -->
    <n-card style="margin-top: 12px; margin-bottom: 12px;" :bordered="false">
      <n-form inline>
        <n-form-item label="供应商名称">
          <n-input v-model:value="query.name" placeholder="搜索" clearable @keyup.enter="search" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="search">搜索</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 表格 -->
    <n-card :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination" @update:page="onPageChange" />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="modalVisible" :title="editingId ? '编辑供应商' : '新增供应商'" preset="card" style="width: 550px;">
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100px">
        <n-form-item label="供应商名称" path="name">
          <n-input v-model:value="form.name" placeholder="请输入供应商名称" />
        </n-form-item>
        <n-form-item label="联系人">
          <n-input v-model:value="form.contact_person" placeholder="联系人" />
        </n-form-item>
        <n-form-item label="联系电话">
          <n-input v-model:value="form.contact_phone" placeholder="电话" />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input v-model:value="form.email" placeholder="邮箱" />
        </n-form-item>
        <n-form-item label="地址">
          <n-input v-model:value="form.address" type="textarea" :rows="2" placeholder="地址" />
        </n-form-item>
        <n-form-item label="备注">
          <n-input v-model:value="form.remark" type="textarea" :rows="2" />
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
import { h, ref, reactive, onMounted } from 'vue'
import { NButton, NSpace, useMessage, useDialog } from 'naive-ui'
import { supplierApi } from '@/api'

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)

const query = reactive({ name: '', page: 1, page_size: 20 })

const pagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { query.page = page; fetchData() },
})

const columns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '供应商名称', key: 'name' },
  { title: '联系人', key: 'contact_person' },
  { title: '联系电话', key: 'contact_phone' },
  { title: '邮箱', key: 'email', ellipsis: { tooltip: true } },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', width: 150, render: (row: any) => h(NSpace, null, { default: () => [
    h(NButton, { size: 'small', onClick: () => showEditModal(row) }, { default: () => '编辑' }),
    h(NButton, { size: 'small', type: 'error', ghost: true, onClick: () => handleDelete(row) }, { default: () => '删除' }),
  ]})},
]

const modalVisible = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<any>(null)
const form = ref({ name: '', contact_person: '', contact_phone: '', email: '', address: '', remark: '' })
const rules = { name: { required: true, message: '请输入供应商名称', trigger: 'blur' } }

async function fetchData() {
  loading.value = true
  try {
    const res: any = await supplierApi.list(query)
    data.value = res?.records || []
    total.value = res?.total || 0
    pagination.itemCount = total.value
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function search() { query.page = 1; fetchData() }
function onPageChange(page: number) { query.page = page; fetchData() }

function showAddModal() {
  editingId.value = null
  form.value = { name: '', contact_person: '', contact_phone: '', email: '', address: '', remark: '' }
  modalVisible.value = true
}

function showEditModal(row: any) {
  editingId.value = row.id
  form.value = { name: row.name, contact_person: row.contact_person || '', contact_phone: row.contact_phone || '', email: row.email || '', address: row.address || '', remark: row.remark || '' }
  modalVisible.value = true
}

async function handleSubmit() {
  try { await formRef.value?.validate() } catch { return }
  submitting.value = true
  try {
    if (editingId.value) {
      await supplierApi.update(editingId.value, form.value)
      message.success('更新成功')
    } else {
      await supplierApi.create(form.value)
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
    content: `确定删除供应商"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try { await supplierApi.delete(row.id); message.success('删除成功'); fetchData() }
      catch (e: any) { message.error(e.message) }
    },
  })
}

onMounted(fetchData)
</script>
