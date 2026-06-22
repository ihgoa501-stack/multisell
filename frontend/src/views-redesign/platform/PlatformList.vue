<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 class="page-header-title">平台管理</h2>
        <div class="page-header-subtitle">配置多电商平台API</div>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showAddModal">添加平台</a-button>
      </div>
    </div>

    <a-card style="margin-top: 12px;" :bordered="false">
      <a-table :columns="columns" :data-source="data" :loading="loading" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button size="small" type="primary" danger ghost @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑平台' : '添加平台'" :width="550" @ok="handleSubmit" :confirm-loading="submitting">
      <a-form ref="formRef" :model="form" :rules="rules" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="平台名称" name="name">
          <a-input v-model:value="form.name" placeholder="如: Ozon, Shopee, Wildberries" />
        </a-form-item>
        <a-form-item label="平台代码" name="code">
          <a-input v-model:value="form.code" placeholder="如: ozon, shopee, wb" />
        </a-form-item>
        <a-form-item label="API地址">
          <a-input v-model:value="form.api_base_url" placeholder="API基础地址" />
        </a-form-item>
        <a-form-item label="Client ID">
          <a-input v-model:value="form.client_id" placeholder="平台分配的Client ID" />
        </a-form-item>
        <a-form-item label="API密钥">
          <a-input-password v-model:value="form.api_key" placeholder="平台API密钥" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sort_order" :min="0" style="width: 100px;" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { platformApi } from '@/api'

const loading = ref(false)
const data = ref<any[]>([])

const columns = [
  { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
  { title: '平台名称', dataIndex: 'name', key: 'name' },
  { title: '代码', dataIndex: 'code', key: 'code', width: 80 },
  { title: 'API地址', dataIndex: 'api_base_url', key: 'api_base_url', ellipsis: true },
  { title: 'Client ID', dataIndex: 'client_id', key: 'client_id', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
  { title: '操作', key: 'actions', width: 150 },
]

const modalVisible = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<any>(null)
const form = ref({ name: '', code: '', api_base_url: '', client_id: '', api_key: '', sort_order: 0 })
const rules = {
  name: [{ required: true, message: '请输入平台名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入平台代码', trigger: 'blur' }],
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
    const payload: any = { ...form.value }
    if (!payload.api_key) delete payload.api_key
    if (editingId.value) {
      await platformApi.update(editingId.value, payload)
      message.success('更新成功')
    } else {
      await platformApi.create(payload)
      message.success('创建成功')
    }
    modalVisible.value = false
    fetchData()
  } catch (e: any) { message.error(e.message) }
  finally { submitting.value = false }
}

function handleDelete(row: any) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除平台"${row.name}"吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try { await platformApi.delete(row.id); message.success('删除成功'); fetchData() }
      catch (e: any) { message.error(e.message) }
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
