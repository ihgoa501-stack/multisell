<template>
  <div>
    <n-page-header subtitle="管理商品分类树">
      <template #title>📂 分类管理</template>
      <template #extra>
        <n-button type="primary" @click="showAddModal(null)">＋ 添加根分类</n-button>
      </template>
    </n-page-header>

    <n-card style="margin-top: 12px;" :bordered="false">
      <n-tree :data="treeData" :default-expand-all="true" :render-label="renderLabel" />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="modalVisible" :title="editingId ? '编辑分类' : '新增分类'" preset="card" style="width: 450px;">
      <n-form ref="formRef" :model="form" :rules="rules">
        <n-form-item label="分类名称" path="name">
          <n-input v-model:value="form.name" placeholder="请输入分类名称" />
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
import { NSpace, NButton, NTag, useMessage, useDialog } from 'naive-ui'
import { categoryApi } from '@/api'

const message = useMessage()
const dialog = useDialog()
const treeData = ref<any[]>([])
const modalVisible = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const parentId = ref(0)
const formRef = ref<any>(null)
const form = ref({ name: '', sort_order: 0 })
const rules = { name: { required: true, message: '请输入分类名称', trigger: 'blur' } }

async function fetchTree() {
  try {
    const res: any = await categoryApi.getTree()
    treeData.value = res.data || []
  } catch (e: any) {
    message.error('加载分类失败')
  }
}

function showAddModal(parent: any) {
  editingId.value = null
  parentId.value = parent?.id || 0
  form.value = { name: '', sort_order: 0 }
  modalVisible.value = true
}

function showEditModal(node: any) {
  editingId.value = node.id
  parentId.value = node.parent_id || 0
  form.value = { name: node.name, sort_order: node.sort_order || 0 }
  modalVisible.value = true
}

async function handleDelete(node: any) {
  dialog.warning({
    title: '确认删除',
    content: `确定删除分类"${node.name}"吗？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await categoryApi.delete(node.id)
        message.success('删除成功')
        fetchTree()
      } catch (e: any) {
        message.error(e.message)
      }
    },
  })
}

async function handleSubmit() {
  try { await formRef.value?.validate() } catch { return }
  submitting.value = true
  try {
    const data = { ...form.value, parent_id: parentId.value }
    if (editingId.value) {
      await categoryApi.update(editingId.value, data)
      message.success('更新成功')
    } else {
      await categoryApi.create(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    fetchTree()
  } catch (e: any) {
    message.error(e.message)
  } finally {
    submitting.value = false
  }
}

function renderLabel({ option }: any) {
  return h('span', { style: 'display: flex; align-items: center; justify-content: space-between; width: 100%;' }, [
    h('span', option.name),
    h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'tiny', quaternary: true, onClick: (e: any) => { e.stopPropagation(); showAddModal(option) } }, { default: () => '添加子分类' }),
        h(NButton, { size: 'tiny', quaternary: true, onClick: (e: any) => { e.stopPropagation(); showEditModal(option) } }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: (e: any) => { e.stopPropagation(); handleDelete(option) } }, { default: () => '删除' }),
      ]
    }),
  ])
}

onMounted(fetchTree)
</script>
