<template>
  <div>
    <div class="page-header">
      <div class="page-header-content">
        <h2 style="margin: 0">分类管理</h2>
        <span style="color: var(--ant-color-text-secondary)">管理商品分类树</span>
      </div>
      <div class="page-header-extra">
        <a-button type="primary" @click="showAddModal(null)">+ 添加根分类</a-button>
      </div>
    </div>

    <a-card style="margin-top: 12px" :bordered="false">
      <a-tree
        :tree-data="treeData"
        :default-expand-all="true"
        block-node
        :field-names="{ key: 'key', title: 'title', children: 'children' }"
      >
        <template #title="{ name, id, parent_id, sort_order }">
          <div style="display: flex; align-items: center; justify-content: space-between; width: 100%">
            <span>{{ name }}</span>
            <a-space :size="4" @click.stop>
              <a-button size="small" type="link" @click.stop="showAddModal({ id })">添加子分类</a-button>
              <a-button size="small" type="link" @click.stop="showEditModal({ id, name, parent_id, sort_order })">编辑</a-button>
              <a-button size="small" type="link" danger @click.stop="handleDelete({ id, name })">删除</a-button>
            </a-space>
          </div>
        </template>
      </a-tree>
    </a-card>

    <!-- 新增/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="editingId ? '编辑分类' : '新增分类'"
      :footer="null"
      style="width: 450px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
        style="margin-top: 16px"
      >
        <a-form-item label="分类名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入分类名称" />
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
import { ref, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { categoryApi } from '@/api'

const treeData = ref<any[]>([])
const modalVisible = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const parentId = ref(0)
const formRef = ref<any>(null)
const form = ref({ name: '', sort_order: 0 })
const rules = { name: { required: true, message: '请输入分类名称', trigger: 'blur' } }

function transformTree(nodes: any[]): any[] {
  return (nodes || []).map((n) => ({
    key: n.id,
    title: n.name,
    name: n.name,
    id: n.id,
    parent_id: n.parent_id,
    sort_order: n.sort_order,
    children: n.children ? transformTree(n.children) : [],
  }))
}

async function fetchTree() {
  try {
    const res: any = await categoryApi.getTree()
    treeData.value = transformTree(res.data || [])
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

function handleDelete(node: any) {
  Modal.confirm({
    title: '确认删除',
    content: `确定删除分类"${node.name}"吗？`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { danger: true },
    onOk: async () => {
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
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
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

onMounted(fetchTree)
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
