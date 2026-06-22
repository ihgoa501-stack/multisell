<template>
  <!-- Custom page header (replacing n-page-header) -->
  <div class="page-header">
    <div class="page-header-left">
      <a-button type="text" @click="router.back()">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <div>
        <h2 class="page-header-title">SKU管理</h2>
        <span class="page-header-subtitle">管理商品规格与SKU</span>
      </div>
    </div>
  </div>

  <!-- 规格定义 -->
  <a-card title="规格定义" style="margin-top: 12px;" :bordered="false">
    <div v-for="(spec, idx) in specs" :key="idx" style="margin-bottom: 12px; display: flex; gap: 8px; align-items: flex-start;">
      <a-input v-model:value="spec.name" placeholder="规格名称（如：颜色）" style="width: 150px;" />
      <a-textarea v-model:value="spec.valuesInput" :rows="2" placeholder="规格值，用英文逗号分隔（如：红,蓝,绿）" style="flex: 1;" />
      <a-button type="text" danger shape="circle" @click="specs.splice(idx, 1)">
        <template #icon><CloseOutlined /></template>
      </a-button>
    </div>
    <a-space>
      <a-button size="small" @click="addSpec">+ 添加规格</a-button>
      <a-button type="primary" size="small" :loading="savingSpecs" @click="saveSpecs">保存规格</a-button>
      <a-button type="primary" size="small" :loading="generatingSku" danger @click="generateSkus">生成SKU</a-button>
    </a-space>
  </a-card>

  <!-- SKU列表 -->
  <a-card title="SKU列表" style="margin-top: 12px;" :bordered="false">
    <a-table
      :columns="skuColumns"
      :data-source="skus"
      :loading="loadingSkus"
      :bordered="true"
      :scroll="{ y: 500 }"
      row-key="id"
      :pagination="false"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'package_override'">
          <span v-if="record.sku_length_cm != null && record.sku_width_cm != null && record.sku_height_cm != null && record.sku_weight_kg != null">
            {{ record.sku_length_cm }} x {{ record.sku_width_cm }} x {{ record.sku_height_cm }} cm / {{ record.sku_weight_kg }} kg
          </span>
          <span v-else>使用商品默认包装</span>
        </template>
        <template v-else-if="column.dataIndex === 'status'">
          <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'action'">
          <a-button size="small" @click="openEditSku(record)">编辑</a-button>
        </template>
      </template>
    </a-table>
  </a-card>

  <!-- 编辑SKU弹窗 -->
  <a-modal v-model:open="editModalVisible" title="编辑SKU" :width="500">
    <a-form v-if="editingSku" :model="editingSku" :label-col="{ style: { width: '80px' } }" layout="horizontal">
      <a-form-item label="SKU编码">
        <a-input v-model:value="editingSku.code" />
      </a-form-item>
      <a-form-item label="条形码">
        <a-input v-model:value="editingSku.barcode" />
      </a-form-item>
      <a-form-item label="销售价">
        <a-input-number v-model:value="editingSku.price" :min="0" :precision="2" style="width: 150px;" />
      </a-form-item>
      <a-form-item label="库存">
        <a-input-number v-model:value="editingSku.stock" :min="0" style="width: 120px;" />
      </a-form-item>
      <a-form-item label="重量(旧字段)">
        <a-input-number v-model:value="editingSku.weight" :min="0" :precision="2" style="width: 150px;" />
      </a-form-item>
      <a-form-item label="包装长">
        <a-input-number v-model:value="editingSku.sku_length_cm" :min="0" :precision="2" style="width: 150px;" addon-after="cm" />
      </a-form-item>
      <a-form-item label="包装宽">
        <a-input-number v-model:value="editingSku.sku_width_cm" :min="0" :precision="2" style="width: 150px;" addon-after="cm" />
      </a-form-item>
      <a-form-item label="包装高">
        <a-input-number v-model:value="editingSku.sku_height_cm" :min="0" :precision="2" style="width: 150px;" addon-after="cm" />
      </a-form-item>
      <a-form-item label="包装重量">
        <a-input-number v-model:value="editingSku.sku_weight_kg" :min="0" :precision="2" style="width: 150px;" addon-after="kg" />
      </a-form-item>
      <a-form-item label="状态">
        <a-switch v-model:checked="editingSku.statusBool" checked-children="启用" un-checked-children="禁用" />
      </a-form-item>
    </a-form>
    <template #footer>
      <div style="display: flex; justify-content: flex-end; gap: 8px;">
        <a-button @click="editModalVisible = false">取消</a-button>
        <a-button type="primary" :loading="savingSku" @click="saveSku">保存</a-button>
      </div>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined, CloseOutlined } from '@ant-design/icons-vue'
import { skuApi } from '@/api'

const router = useRouter()
const route = useRoute()
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
  { title: 'SKU编码', dataIndex: 'code', width: 150 },
  { title: '规格描述', dataIndex: 'spec_desc', ellipsis: true },
  { title: '价格', dataIndex: 'price', width: 100 },
  { title: '库存', dataIndex: 'stock', width: 80 },
  { title: '包装覆盖', dataIndex: 'package_override', width: 180 },
  { title: '条形码', dataIndex: 'barcode', width: 130 },
  { title: '状态', dataIndex: 'status', width: 70 },
  { title: '操作', dataIndex: 'action', width: 100 },
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
      weight: editingSku.value.weight,
      sku_length_cm: editingSku.value.sku_length_cm,
      sku_width_cm: editingSku.value.sku_width_cm,
      sku_height_cm: editingSku.value.sku_height_cm,
      sku_weight_kg: editingSku.value.sku_weight_kg,
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

<style scoped>
/* Custom page header */
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--ant-color-border, #e5e5e5);
  margin-bottom: 16px;
}

.page-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-header-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ant-color-text);
}

.page-header-subtitle {
  font-size: 13px;
  color: var(--ant-color-text-secondary);
}
</style>
