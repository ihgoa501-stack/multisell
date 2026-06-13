<template>
  <n-page-header @back="router.back()">
    <template #title>🏷️ SKU管理</template>
    <template #subtitle>管理商品规格与SKU</template>
  </n-page-header>

  <!-- 规格定义 -->
  <n-card title="规格定义" style="margin-top: 12px;" :bordered="false">
    <div v-for="(spec, idx) in specs" :key="idx" style="margin-bottom: 12px; display: flex; gap: 8px; align-items: flex-start;">
      <n-input v-model:value="spec.name" placeholder="规格名称（如：颜色）" style="width: 150px;" />
      <n-input v-model:value="spec.valuesInput" type="textarea" :rows="2" placeholder="规格值，用英文逗号分隔（如：红,蓝,绿）" style="flex: 1;" />
      <n-button quaternary circle type="error" @click="specs.splice(idx, 1)">✕</n-button>
    </div>
    <n-space>
      <n-button size="small" @click="addSpec">＋ 添加规格</n-button>
      <n-button type="primary" size="small" :loading="savingSpecs" @click="saveSpecs">保存规格</n-button>
      <n-button type="warning" size="small" :loading="generatingSku" @click="generateSkus">⚡ 生成SKU</n-button>
    </n-space>
  </n-card>

  <!-- SKU列表 -->
  <n-card title="SKU列表" style="margin-top: 12px;" :bordered="false">
    <n-data-table :columns="skuColumns" :data="skus" :loading="loadingSkus" :bordered="true" :max-height="500" />
  </n-card>

  <!-- 编辑SKU弹窗 -->
  <n-modal v-model:show="editModalVisible" title="编辑SKU" preset="card" style="width: 500px;">
    <n-form v-if="editingSku" :model="editingSku" label-placement="left" label-width="80px">
      <n-form-item label="SKU编码"><n-input v-model:value="editingSku.code" /></n-form-item>
      <n-form-item label="条形码"><n-input v-model:value="editingSku.barcode" /></n-form-item>
      <n-form-item label="销售价"><n-input-number v-model:value="editingSku.price" :min="0" :precision="2" style="width: 150px;" /></n-form-item>
      <n-form-item label="库存"><n-input-number v-model:value="editingSku.stock" :min="0" style="width: 120px;" /></n-form-item>
      <n-form-item label="状态">
        <n-switch v-model:value="editingSku.statusBool" :checked-value="true" :unchecked-value="false">
          <template #checked>启用</template>
          <template #unchecked>禁用</template>
        </n-switch>
      </n-form-item>
    </n-form>
    <template #footer>
      <n-space justify="end">
        <n-button @click="editModalVisible = false">取消</n-button>
        <n-button type="primary" :loading="savingSku" @click="saveSku">保存</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NButton, NTag, NSpace, useMessage } from 'naive-ui'
import { skuApi } from '@/api'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const productId = Number(route.params.id)

const specs = ref<any[]>([{ name: '', valuesInput: '' }])
const savingSpecs = ref(false)
const generatingSku = ref(false)
const loadingSkus = ref(false)
const skus = ref<any[]>([])
const editModalVisible = ref(false)
const editingSku = ref<any>(null)
const savingSku = ref(false)

function addSpec() { specs.value.push({ name: '', valuesInput: '' }) }

async function saveSpecs() {
  const specData = specs.value
    .filter(s => s.name && s.valuesInput)
    .map(s => ({ name: s.name, values: s.valuesInput.split(',').map((v: string) => v.trim()).filter(Boolean) }))
  if (!specData.length) { message.warning('请填写规格'); return }
  savingSpecs.value = true
  try {
    await skuApi.defineSpecs(productId, { specs: specData })
    message.success('规格保存成功')
  } catch (e: any) { message.error(e.message) }
  finally { savingSpecs.value = false }
}

async function generateSkus() {
  generatingSku.value = true
  try {
    const res: any = await skuApi.generateSkus(productId)
    message.success(`成功生成 ${res.data?.total || 0} 个SKU`)
    await fetchSkus()
  } catch (e: any) { message.error(e.message) }
  finally { generatingSku.value = false }
}

async function fetchSkus() {
  loadingSkus.value = true
  try {
    const res: any = await skuApi.getSkus(productId)
    skus.value = res.data || []
  } catch (e: any) { message.error(e.message) }
  finally { loadingSkus.value = false }
}

const skuColumns = [
  { title: 'SKU编码', key: 'code', width: 150 },
  { title: '规格描述', key: 'spec_desc', ellipsis: { tooltip: true } },
  { title: '价格', key: 'price', width: 100 },
  { title: '库存', key: 'stock', width: 80 },
  { title: '条形码', key: 'barcode', width: 130 },
  { title: '状态', key: 'status', width: 70, render: (row: any) => h(NTag, { type: row.status === 1 ? 'success' : 'default', size: 'small' }, { default: () => row.status === 1 ? '启用' : '禁用' }) },
  { title: '操作', width: 100, render: (row: any) => h(NButton, { size: 'small', onClick: () => openEditSku(row) }, { default: () => '编辑' }) },
]

function openEditSku(sku: any) {
  editingSku.value = { ...sku, statusBool: sku.status === 1 }
  editModalVisible.value = true
}

async function saveSku() {
  if (!editingSku.value) return
  savingSku.value = true
  try {
    const data = {
      code: editingSku.value.code,
      barcode: editingSku.value.barcode,
      price: editingSku.value.price,
      stock: editingSku.value.stock,
      status: editingSku.value.statusBool ? 1 : 0,
    }
    await skuApi.updateSku(editingSku.value.id, data)
    message.success('更新成功')
    editModalVisible.value = false
    await fetchSkus()
  } catch (e: any) { message.error(e.message) }
  finally { savingSku.value = false }
}

onMounted(async () => {
  // 加载现有规格
  try {
    const res: any = await skuApi.getSpecs(productId)
    if (res.data && res.data.length) {
      specs.value = res.data.map((s: any) => ({ name: s.name, valuesInput: s.values?.map((v: any) => v.value).join(',') || '' }))
    }
  } catch { /* ignore */ }
  await fetchSkus()
})
</script>
