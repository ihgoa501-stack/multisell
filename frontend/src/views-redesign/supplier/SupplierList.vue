<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 style="margin: 0">供应商管理</h2>
        <span style="color: var(--ant-color-text-secondary)">管理供应商信息</span>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showAddModal">+ 新增供应商</a-button>
      </div>
    </div>

    <!-- 搜索 -->
    <a-card style="margin-top: 12px; margin-bottom: 12px" :bordered="false">
      <a-form layout="inline">
        <a-form-item label="供应商名称">
          <a-input
            v-model:value="query.name"
            placeholder="搜索"
            allow-clear
            @keyup.enter="search"
          />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="search">搜索</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 表格 -->
    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="tablePagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'action'">
            <a-space>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="editingId ? '编辑供应商' : '新增供应商'"
      :footer="null"
      style="width: 550px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
        style="margin-top: 16px"
      >
        <a-form-item label="供应商名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入供应商名称" />
        </a-form-item>
        <a-form-item label="联系人">
          <a-input v-model:value="form.contact_person" placeholder="联系人" />
        </a-form-item>
        <a-form-item label="联系电话">
          <a-input v-model:value="form.contact_phone" placeholder="电话" />
        </a-form-item>
        <a-form-item label="邮箱">
          <a-input v-model:value="form.email" placeholder="邮箱" />
        </a-form-item>
        <a-form-item label="地址">
          <a-textarea v-model:value="form.address" :rows="2" placeholder="地址" />
        </a-form-item>
        <a-form-item label="备注">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
      <template #footer>
        <a-space>
          <a-button @click="modalVisible = false">取消</a-button>
          <a-button type="primary" :loading="submitting" @click="handleSubmit">保存</a-button>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { supplierApi } from '@/api'

const loading = ref(false)
const data = ref<any[]>([])
const total = ref(0)

const query = reactive({ name: '', page: 1, page_size: 20 })

const tablePagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: false,
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '供应商名称', dataIndex: 'name', key: 'name' },
  { title: '联系人', dataIndex: 'contact_person', key: 'contact_person' },
  { title: '联系电话', dataIndex: 'contact_phone', key: 'contact_phone' },
  { title: '邮箱', dataIndex: 'email', key: 'email', ellipsis: true },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 150 },
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
    tablePagination.total = total.value
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

function search() {
  query.page = 1
  tablePagination.current = 1
  fetchData()
}

function onTableChange(pag: any) {
  query.page = pag.current
  tablePagination.current = pag.current
  fetchData()
}

function showAddModal() {
  editingId.value = null
  form.value = { name: '', contact_person: '', contact_phone: '', email: '', address: '', remark: '' }
  modalVisible.value = true
}

function showEditModal(row: any) {
  editingId.value = row.id
  form.value = {
    name: row.name,
    contact_person: row.contact_person || '',
    contact_phone: row.contact_phone || '',
    email: row.email || '',
    address: row.address || '',
    remark: row.remark || '',
  }
  modalVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
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
  } catch (e: any) {
    message.error(e.message)
  } finally {
    submitting.value = false
  }
}

function handleDelete(row: any) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除供应商"${row.name}"吗？`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await supplierApi.delete(row.id)
        message.success('删除成功')
        fetchData()
      } catch (e: any) {
        message.error(e.message)
      }
    },
  })
}

onMounted(fetchData)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 16px 0;
}
.page-header-content h2 {
  font-size: 20px;
  margin-bottom: 4px;
}
.page-header-extra {
  display: flex;
  align-items: center;
}
</style>
