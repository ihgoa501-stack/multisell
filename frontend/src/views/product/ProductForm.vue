<template>
  <n-page-header :subtitle="isEdit ? '编辑商品信息' : '创建新商品'" @back="router.back()">
    <template #title>{{ isEdit ? '✏️ 编辑商品' : '➕ 新增商品' }}</template>
  </n-page-header>

  <n-card style="margin-top: 12px; max-width: 800px;" :bordered="false">
    <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100px">
      <n-form-item label="商品名称" path="name">
        <n-input v-model:value="form.name" placeholder="请输入商品名称" maxlength="200" />
      </n-form-item>
      <n-form-item label="副标题" path="subtitle">
        <n-input v-model:value="form.subtitle" placeholder="选填" maxlength="500" />
      </n-form-item>
      <n-form-item label="分类" path="category_id">
        <n-tree-select :options="categoryTree" v-model:value="form.category_id" placeholder="选择分类" clearable filterable />
      </n-form-item>
      <n-form-item label="品牌">
        <n-select v-model:value="form.brand_id" :options="brandOptions" placeholder="选择品牌" clearable filterable />
      </n-form-item>
      <n-form-item label="单位">
        <n-input v-model:value="form.unit" placeholder="件" style="width: 100px;" />
      </n-form-item>
      <n-form-item label="描述" path="description">
        <n-input v-model:value="form.description" type="textarea" :rows="4" placeholder="商品描述" />
      </n-form-item>
      <n-divider title-placement="left">商品尺寸</n-divider>
      <n-grid :cols="2" :x-gap="12">
        <n-form-item-gi label="商品长">
          <n-input-number v-model:value="form.product_length_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="商品宽">
          <n-input-number v-model:value="form.product_width_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="商品高">
          <n-input-number v-model:value="form.product_height_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="商品重量">
          <n-input-number v-model:value="form.product_weight_kg" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>kg</template>
          </n-input-number>
        </n-form-item-gi>
      </n-grid>
      <n-divider title-placement="left">包装信息</n-divider>
      <n-grid :cols="2" :x-gap="12">
        <n-form-item-gi label="包装长">
          <n-input-number v-model:value="form.package_length_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="包装宽">
          <n-input-number v-model:value="form.package_width_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="包装高">
          <n-input-number v-model:value="form.package_height_cm" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>cm</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="包装重量">
          <n-input-number v-model:value="form.package_weight_kg" :min="0" :precision="2" style="width: 100%;">
            <template #suffix>kg</template>
          </n-input-number>
        </n-form-item-gi>
        <n-form-item-gi label="货品类型">
          <n-select v-model:value="form.cargo_type" :options="cargoTypeOptions" />
        </n-form-item-gi>
      </n-grid>
      <n-form-item v-if="isEdit" label="AI优化">
        <n-space>
          <n-button :loading="aiLoading" type="warning" @click="handleAiEnhance">✨ AI优化</n-button>
          <span v-if="form.ai_status" style="font-size: 12px; color: #999;">
            AI状态: {{ form.ai_status === 'completed' ? '✅ 已优化' : form.ai_status === 'failed' ? '❌ 失败' : '⏳ 待处理' }}
          </span>
        </n-space>
      </n-form-item>
      <n-form-item label="主图">
        <n-upload
          :show-file-list="false"
          accept="image/*"
          @change="handleImageUpload"
          :disabled="uploading"
        >
          <n-button :loading="uploading">上传图片</n-button>
        </n-upload>
        <n-image v-if="form.main_image" :src="form.main_image" width="80" style="margin-left: 12px; border-radius: 4px;" />
      </n-form-item>
      <n-form-item label="状态">
        <n-radio-group v-model:value="form.status">
          <n-radio :value="0">草稿</n-radio>
          <n-radio :value="1">上架</n-radio>
          <n-radio :value="2">下架</n-radio>
        </n-radio-group>
      </n-form-item>
      <n-form-item>
        <n-space>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
          <n-button @click="router.back()">取消</n-button>
        </n-space>
      </n-form-item>
    </n-form>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage } from 'naive-ui'
import { productApi, categoryApi, brandApi } from '@/api'
import http from '@/api/http'

const router = useRouter()
const route = useRoute()
const message = useMessage()
const formRef = ref<any>(null)
const submitting = ref(false)
const uploading = ref(false)
const aiLoading = ref(false)
const isEdit = computed(() => !!route.params.id)
const categoryTree = ref<any[]>([])
const brandOptions = ref<any[]>([])
const cargoTypeOptions = [
  { label: '普通货品', value: 'normal' },
  { label: '带电', value: 'battery' },
  { label: '液体', value: 'liquid' },
  { label: '敏感货', value: 'sensitive' },
]

const createEmptyForm = () => ({
  name: '',
  subtitle: '',
  category_id: null,
  brand_id: null,
  unit: '件',
  description: '',
  main_image: '',
  status: 0,
  ai_status: '',
  product_length_cm: null,
  product_width_cm: null,
  product_height_cm: null,
  product_weight_kg: null,
  package_length_cm: null,
  package_width_cm: null,
  package_height_cm: null,
  package_weight_kg: null,
  cargo_type: 'normal',
})

const form = ref<any>(createEmptyForm())

const rules = {
  name: { required: true, message: '请输入商品名称', trigger: 'blur' },
}

onMounted(async () => {
  // 加载分类树
  try {
    const res: any = await categoryApi.getTree()
    categoryTree.value = res.data || []
  } catch { /* ignore */ }

  // 加载品牌列表
  try {
    const res: any = await brandApi.getAll()
    brandOptions.value = (res.data || res || []).map((b: any) => ({ label: b.name, value: b.id }))
  } catch { /* ignore */ }

  // 编辑模式加载数据
  if (isEdit.value) {
    try {
      const res: any = await productApi.getById(Number(route.params.id))
      if (res.data) {
        form.value = {
          ...createEmptyForm(),
          name: res.data.name,
          subtitle: res.data.subtitle || '',
          category_id: res.data.category_id,
          brand_id: res.data.brand_id || null,
          unit: res.data.unit || '件',
          description: res.data.description || '',
          main_image: res.data.main_image || '',
          status: res.data.status ?? 0,
          ai_status: res.data.ai_status || '',
          product_length_cm: res.data.product_length_cm ?? null,
          product_width_cm: res.data.product_width_cm ?? null,
          product_height_cm: res.data.product_height_cm ?? null,
          product_weight_kg: res.data.product_weight_kg ?? null,
          package_length_cm: res.data.package_length_cm ?? null,
          package_width_cm: res.data.package_width_cm ?? null,
          package_height_cm: res.data.package_height_cm ?? null,
          package_weight_kg: res.data.package_weight_kg ?? null,
          cargo_type: res.data.cargo_type || 'normal',
        }
      }
    } catch (e: any) {
      message.error('加载商品信息失败')
    }
  }
})

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch { return }

  submitting.value = true
  try {
    if (isEdit.value) {
      await productApi.update(Number(route.params.id), form.value)
      message.success('更新成功')
    } else {
      await productApi.create(form.value)
      message.success('创建成功')
    }
    router.push('/products')
  } catch (e: any) {
    message.error(e.message)
  } finally {
    submitting.value = false
  }
}

async function handleImageUpload({ file }: any) {
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', file.file)
    const res: any = await http.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    if (res.code === 200 && res.data?.url) {
      form.value.main_image = res.data.url
      message.success('图片上传成功')
    }
  } catch (e: any) {
    message.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

async function handleAiEnhance() {
  if (!isEdit.value) {
    message.warning('请先保存商品，再使用AI优化')
    return
  }
  const productId = Number(route.params.id)
  if (!productId) return
  
  aiLoading.value = true
  try {
    const res: any = await http.post(`/products/${productId}/ai-enhance`)
    if (res.code === 200 && res.data) {
      form.value.description = res.data.enhanced_description || form.value.description
      form.value.ai_status = 'completed'
      message.success('AI优化完成，请检查并确认')
    }
  } catch (e: any) {
    message.error('AI优化失败')
    form.value.ai_status = 'failed'
  } finally {
    aiLoading.value = false
  }
}
</script>
