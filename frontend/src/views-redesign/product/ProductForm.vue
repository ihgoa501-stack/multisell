<template>
  <!-- Custom page header (replacing n-page-header) -->
  <div class="page-header">
    <div class="page-header-left">
      <a-button type="text" @click="router.back()">
        <template #icon><ArrowLeftOutlined /></template>
      </a-button>
      <div>
        <h2 class="page-header-title">{{ isEdit ? '编辑商品' : '新增商品' }}</h2>
        <span class="page-header-subtitle">{{ isEdit ? '编辑商品信息' : '创建新商品' }}</span>
      </div>
    </div>
  </div>

  <a-card style="margin-top: 12px; max-width: 800px;" :bordered="false">
    <a-form
      ref="formRef"
      :model="form"
      :rules="rules"
      :label-col="{ style: { width: '100px' } }"
      layout="horizontal"
    >
      <a-form-item label="商品名称" name="name">
        <a-input v-model:value="form.name" placeholder="请输入商品名称" :maxlength="200" />
      </a-form-item>
      <a-form-item label="副标题" name="subtitle">
        <a-input v-model:value="form.subtitle" placeholder="选填" :maxlength="500" />
      </a-form-item>
      <a-form-item label="分类" name="category_id">
        <a-tree-select
          v-model:value="form.category_id"
          :tree-data="categoryTree"
          placeholder="选择分类"
          allow-clear
          show-search
          tree-node-filter-prop="title"
          :field-names="{ label: 'label', value: 'key', children: 'children' }"
        />
      </a-form-item>
      <a-form-item label="品牌">
        <a-select v-model:value="form.brand_id" :options="brandOptions" placeholder="选择品牌" allow-clear show-search />
      </a-form-item>
      <a-form-item label="单位">
        <a-input v-model:value="form.unit" placeholder="件" style="width: 100px;" />
      </a-form-item>
      <a-form-item label="描述" name="description">
        <a-textarea v-model:value="form.description" :rows="4" placeholder="商品描述" />
      </a-form-item>

      <a-divider orientation="left">商品尺寸</a-divider>
      <a-row :gutter="12">
        <a-col :span="12">
          <a-form-item label="商品长">
            <a-input-number v-model:value="form.product_length_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="商品宽">
            <a-input-number v-model:value="form.product_width_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="商品高">
            <a-input-number v-model:value="form.product_height_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="商品重量">
            <a-input-number v-model:value="form.product_weight_kg" :min="0" :precision="2" style="width: 100%;" addon-after="kg" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-divider orientation="left">包装信息</a-divider>
      <a-row :gutter="12">
        <a-col :span="12">
          <a-form-item label="包装长">
            <a-input-number v-model:value="form.package_length_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="包装宽">
            <a-input-number v-model:value="form.package_width_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="包装高">
            <a-input-number v-model:value="form.package_height_cm" :min="0" :precision="2" style="width: 100%;" addon-after="cm" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="包装重量">
            <a-input-number v-model:value="form.package_weight_kg" :min="0" :precision="2" style="width: 100%;" addon-after="kg" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="货品类型">
            <a-select v-model:value="form.cargo_type" :options="cargoTypeOptions" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-form-item v-if="isEdit" label="AI优化">
        <a-space>
          <a-button :loading="aiLoading" type="primary" danger @click="handleAiEnhance">AI优化</a-button>
          <span v-if="form.ai_status" style="font-size: 12px; color: var(--ant-color-text-tertiary);">
            AI状态: {{ form.ai_status === 'completed' ? '已优化' : form.ai_status === 'failed' ? '失败' : '待处理' }}
          </span>
        </a-space>
      </a-form-item>

      <a-form-item label="主图">
        <a-upload
          :show-upload-list="false"
          accept="image/*"
          :custom-request="handleImageUpload"
          :disabled="uploading"
        >
          <a-button :loading="uploading">上传图片</a-button>
        </a-upload>
        <img
          v-if="form.main_image"
          :src="form.main_image"
          style="width: 80px; margin-left: 12px; border-radius: 4px; vertical-align: middle;"
        />
      </a-form-item>

      <a-form-item label="状态">
        <a-radio-group v-model:value="form.status">
          <a-radio :value="0">草稿</a-radio>
          <a-radio :value="1">上架</a-radio>
          <a-radio :value="2">下架</a-radio>
        </a-radio-group>
      </a-form-item>

      <a-form-item :wrapper-col="{ offset: 0 }">
        <a-space>
          <a-button type="primary" :loading="submitting" @click="handleSubmit">保存</a-button>
          <a-button @click="router.back()">取消</a-button>
        </a-space>
      </a-form-item>
    </a-form>
  </a-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import { productApi, categoryApi, brandApi } from '@/api'
import http from '@/api/http'

const router = useRouter()
const route = useRoute()
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
  name: [{ required: true, message: '请输入商品名称', trigger: 'blur' }],
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

async function handleImageUpload(options: any) {
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', options.file)
    const res: any = await http.post('/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    if (res.code === 200 && res.data?.url) {
      form.value.main_image = res.data.url
      message.success('图片上传成功')
      options.onSuccess?.(res)
    }
  } catch (e: any) {
    message.error('图片上传失败')
    options.onError?.(e)
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
