<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 style="margin: 0">品牌管理</h2>
        <span style="color: var(--ant-color-text-secondary)">管理商品品牌</span>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showAddModal">+ 新增品牌</a-button>
      </div>
    </div>

    <a-card style="margin-top: 12px" :bordered="false">
      <a-table
        :columns="columns"
        :data-source="data"
        :loading="loading"
        :pagination="tablePagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'status'">
            <a-tag :color="record.status === 1 ? 'success' : 'default'">
              {{ record.status === 1 ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
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
      :title="editingId ? '编辑品牌' : '新增品牌'"
      :footer="null"
      style="width: 500px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 4 }"
        :wrapper-col="{ span: 19 }"
        style="margin-top: 16px"
      >
        <a-form-item label="品牌名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入品牌名称" />
        </a-form-item>
        <a-form-item label="Logo URL">
          <a-input v-model:value="form.logo" placeholder="品牌Logo链接（选填）" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" :rows="3" placeholder="品牌描述（选填）" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sort_order" :min="0" style="width: 100px" />
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
import { brandApi } from '@/api'

const loading = ref(false)
const data = ref<any[]>([])

const query = reactive({ name: '', page: 1, page_size: 20 })

const tablePagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: false,
})

const columns = [
  { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 60 },
  { title: '品牌名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 70 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 150 },
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
    tablePagination.total = res?.total || 0
  } catch (e: any) {
    message.error(e.message)
  } finally {
    loading.value = false
  }
}

function onTableChange(pag: any) {
  query.page = pag.current
  tablePagination.current = pag.current
  fetchData()
}

function showAddModal() {
  editingId.value = null
  form.value = { name: '', logo: '', description: '', sort_order: 0 }
  modalVisible.value = true
}

function showEditModal(row: any) {
  editingId.value = row.id
  form.value = {
    name: row.name,
    logo: row.logo || '',
    description: row.description || '',
    sort_order: row.sort_order || 0,
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
      await brandApi.update(editingId.value, form.value)
      message.success('更新成功')
    } else {
      await brandApi.create(form.value)
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
    content: `确定删除品牌"${row.name}"吗？`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await brandApi.delete(row.id)
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
