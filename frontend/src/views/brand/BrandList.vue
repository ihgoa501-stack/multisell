<template>
  <div>
    <n-page-header subtitle="管理商品品牌">
      <template #title>🏷️ 品牌管理</template>
      <template #extra>
        <n-button type="primary" @click="showAddModal">＋ 新增品牌</n-button>
      </template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-data-table :columns="columns" :data="data" :loading="loading" :pagination="pagination" @update:page="onPageChange" />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="modalVisible" :title="editingId ? '编辑品牌' : '新增品牌'" preset="card" style="width: 500px;">
      <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="80px">
        <n-form-item label="品牌名称" path="name">
          <n-input v-model:value="form.name" placeholder="请输入品牌名称" />
        </n-form-item>
        <n-form-item label="Logo URL">
          <n-input v-model:value="form.logo" placeholder="品牌Logo链接（选填）" />
        </n-form-item>
        <n-form-item label="描述">
          <n-input v-model:value="form.description" type="textarea" :rows="3" placeholder="品牌描述（选填）" />
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
import { h, ref, reactive, onMounted } from 'vue'
import { NButton, NSpace, NTag, useMessage, useDialog } from 'naive-ui'
import { brandApi } from '@/api'

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const data = ref<any[]>([])

const query = reactive({ name: '', page: 1, page_size: 20 })

const pagination = reactive({
  page: 1, pageSize: 20, itemCount: 0,
  onChange: (page: number) => { query.page = page; fetchData() },
})

const columns = [
  { title: '排序', key: 'sort_order', width: 60 },
  { title: '品牌名称', key: 'name' },
  { title: '描述', key: 'description', ellipsis: { tooltip: true } },
  { title: '状态', key: 'status', width: 70, render: (row: any) => h(NTag, { type: row.status === 1 ? 'success' : 'default', size: 'small' }, { default: () => row.status === 1 ? '启用' : '禁用' }) },
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
const form = ref({ name: '', logo: '', description: '', sort_order: 0 })
const rules = { name: { required: true, message: '请输入品牌名称', trigger: 'blur' } }

async function fetchData() {
  loading.value = true
  try {
    const res: any = await brandApi.list(query)
    data.value = res?.records || []
    pagination.itemCount = res?.total || 0
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function onPageChange(page: number) { query.page = page; fetchData() }

function showAddModal() {
  editingId.value = null
  form.value = { name: '', logo: '', description: '', sort_order: 0 }
  modalVisible.value = true
}

function showEditModal(row: any) {
  editingId.value = row.id
  form.value = { name: row.name, logo: row.logo || '', description: row.description || '', sort_order: row.sort_order || 0 }
  modalVisible.value = true
}

async function handleSubmit() {
  try { await formRef.value?.validate() } catch { return }
  submitting.value = true
  try {
    if (editingId.value) {
      await brandApi.update(editingId.value, form.value)
      message.success('更新成功')
    } else {
      await brandApi.create(form.value)
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
    content: `确定删除品牌"${row.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try { await brandApi.delete(row.id); message.success('删除成功'); fetchData() }
      catch (e: any) { message.error(e.message) }
    },
  })
}

onMounted(fetchData)
</script>
