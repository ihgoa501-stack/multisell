<template>
  <div class="max-w-7xl mx-auto">
    <!-- ====== 页面标题 ====== -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-xl font-bold text-[var(--text-primary)]">AI 生图</h1>
        <p class="text-sm text-[var(--text-tertiary)] mt-0.5">为商品生成产品图片，支持白底 / 场景 / 模特 / 3D 渲染</p>
      </div>
      <div class="flex items-center gap-2">
        <n-tag size="small" type="info" round>Replicate · FLUX.2 Pro</n-tag>
      </div>
    </div>

    <!-- ====== 双栏主体 ====== -->
    <div class="grid grid-cols-[340px_1fr] gap-4 items-start mb-6">
      <!-- 左栏：商品搜索（多选） -->
      <div class="bg-white border border-[var(--border-light)] rounded-lg">
        <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
          <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">选择商品</span>
          <span class="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-[var(--accent-bg)] text-[var(--accent)]">
            {{ selectedProductIds.length }} 个已选
          </span>
        </div>
        <div class="p-4">
          <div class="flex items-center gap-1.5 px-3 py-1.5 border border-[var(--border)] rounded-[6px] mb-3 transition-colors focus-within:border-[var(--accent)]">
            <svg class="w-3.5 h-3.5 text-[var(--text-tertiary)] shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><path d="M21 21l-4.3-4.3"/></svg>
            <input v-model="searchQuery" type="text" placeholder="搜索商品名称..."
              class="w-full border-none bg-transparent outline-none text-[13px] text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] font-sans">
          </div>
          <!-- 快捷操作 -->
          <div class="flex items-center gap-1 mb-2">
            <n-button size="tiny" quaternary @click="selectAllFiltered">全选当前</n-button>
            <n-button size="tiny" quaternary @click="selectedProductIds = []">清空</n-button>
          </div>
          <!-- 商品列表（多选） -->
          <div class="flex flex-col gap-0.5 max-h-[360px] overflow-y-auto">
            <div v-for="p in filteredProducts" :key="p.id"
              class="flex items-center gap-2.5 px-2.5 py-2 rounded-[6px] cursor-pointer transition-colors"
              :class="selectedProductIds.includes(p.id) ? 'bg-[var(--accent-bg)]' : 'hover:bg-[var(--bg-hover)]'"
              @click="toggleProduct(p.id)"
            >
              <n-checkbox :checked="selectedProductIds.includes(p.id)" @click.stop />
              <div class="w-8 h-8 rounded-[4px] bg-[var(--bg-subtle)] flex items-center justify-center overflow-hidden shrink-0">
                <img v-if="p.main_image" :src="p.main_image" class="w-full h-full object-cover" />
                <span v-else class="text-base">📦</span>
              </div>
              <div class="flex-1 min-w-0">
                <div class="text-[13px] font-medium text-[var(--text-primary)] truncate">{{ p.name || '商品' }}</div>
                <div class="text-[11px] text-[var(--text-tertiary)] mt-px">{{ p.category_name || '' }}</div>
              </div>
            </div>
            <div v-if="filteredProducts.length === 0" class="text-center py-8 text-sm text-[var(--text-tertiary)]">暂无匹配商品</div>
          </div>
        </div>
      </div>

      <!-- 右栏：生图工作台 -->
      <div class="flex flex-col gap-3.5">
        <!-- ↑ 提示词 + 配置 + 模板 （一行内紧凑排列）-->
        <div class="bg-white border border-[var(--border-light)] rounded-lg">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
            <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">生成配置</span>
            <div class="flex items-center gap-1">
              <n-button size="tiny" quaternary @click="showTemplateManager = true">
                <template #icon><svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/></svg></template>
                管理模板
              </n-button>
            </div>
          </div>
          <div class="px-4 py-3 space-y-3">
            <!-- 模板选择 -->
            <div v-if="templates.length > 0">
              <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">快速模板</label>
              <div class="flex gap-1.5 flex-wrap">
                <n-button v-for="t in templates.slice(0, 8)" :key="t.id" size="tiny" ghost
                  :class="activeTemplateId === t.id ? '!border-[var(--accent)] !text-[var(--accent)]' : ''"
                  @click="applyTemplate(t)">
                  {{ t.name }}
                </n-button>
              </div>
            </div>
            <!-- 正向提示词 -->
            <div>
              <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">正向提示词</label>
              <n-input v-model:value="formData.prompt" type="textarea" :rows="2"
                placeholder="描述你想要的图片内容，例如：白色耳机在木质桌面上，自然光..."
                :disabled="selectedProductIds.length === 0" />
            </div>
            <!-- 反向提示词 -->
            <div>
              <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">反向提示词（可选）</label>
              <n-input v-model:value="formData.negative_prompt" type="textarea" :rows="1"
                placeholder="模糊、变形、多余的手、文字..."
                :disabled="selectedProductIds.length === 0" />
            </div>
            <!-- 风格 + 尺寸 + 数量 -->
            <div class="grid grid-cols-3 gap-3">
              <div>
                <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">风格</label>
                <n-select v-model:value="formData.style" :options="styleOptions" :disabled="selectedProductIds.length === 0" />
              </div>
              <div>
                <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">尺寸</label>
                <n-select v-model:value="formData.size" :options="sizeOptions" :disabled="selectedProductIds.length === 0" />
              </div>
              <div>
                <label class="text-[12px] font-medium text-[var(--text-secondary)] block mb-1">每商品数量</label>
                <n-input-number v-model:value="formData.count" :min="1" :max="4" :disabled="selectedProductIds.length === 0" class="w-full" />
              </div>
            </div>
            <!-- 操作按钮 -->
            <div class="flex items-center gap-2 pt-1">
              <n-button type="primary" size="small" :loading="generating"
                :disabled="selectedProductIds.length === 0 || !formData.prompt" @click="handleGenerate">
                <template #icon>
                  <svg v-if="!generating" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 3v18M3 12h18"/>
                  </svg>
                </template>
                {{ selectedProductIds.length > 1 ? `批量生成 (${selectedProductIds.length} 个商品)` : '开始生成' }}
              </n-button>
              <n-button size="small" quaternary :disabled="selectedProductIds.length === 0" @click="loadHistory">
                刷新历史
              </n-button>
              <n-button size="small" quaternary :disabled="!formData.prompt" @click="showSaveTemplate = true">
                存为模板
              </n-button>
            </div>
            <!-- 批量进度 -->
            <div v-if="batchResult" class="px-3 py-2 rounded-[6px] text-[12px] border"
              :class="batchResult.failed > 0 ? 'border-[var(--amber)] bg-[var(--amber-bg)]' : 'border-[var(--green)] bg-[var(--green-bg)]'">
              <span class="font-semibold">批量完成：</span>
              共 {{ batchResult.total }} 个商品，成功 {{ batchResult.success }}，失败 {{ batchResult.failed }}
              <n-button size="tiny" quaternary class="ml-2" @click="batchResult = null">关闭</n-button>
            </div>
          </div>
        </div>

        <!-- 生成结果画廊 -->
        <div class="bg-white border border-[var(--border-light)] rounded-lg">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
            <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">生成结果</span>
            <span v-if="latestGrouped.length > 0" class="text-[11px] text-[var(--text-tertiary)]">
              {{ totalLatestImages }} 张图片
            </span>
          </div>
          <div class="p-4">
            <!-- 空状态 -->
            <div v-if="latestGrouped.length === 0 && selectedProductIds.length === 0" class="text-center py-12">
              <div class="text-3xl mb-2">🎨</div>
              <div class="text-[13px] text-[var(--text-tertiary)]">请先在左侧选择商品，然后输入提示词开始生成</div>
            </div>
            <div v-else-if="latestGrouped.length === 0 && generating" class="text-center py-8">
              <n-spin size="medium" />
              <div class="text-sm text-[var(--text-secondary)] mt-3">正在生成图片，请稍候...</div>
            </div>
            <div v-else-if="latestGrouped.length === 0" class="text-center py-8 text-sm text-[var(--text-tertiary)]">
              暂无生成结果，点击"开始生成"创建图片
            </div>
            <!-- 按商品分组的图片 -->
            <div v-for="(group, gi) in latestGrouped" :key="gi" class="mb-4 last:mb-0">
              <div class="text-[12px] font-medium text-[var(--text-secondary)] mb-2 flex items-center gap-2">
                <span>{{ group.product_name || `商品 #${group.product_id}` }}</span>
                <span class="text-[10px] text-[var(--text-tertiary)]">{{ group.images.length }} 张</span>
              </div>
              <div class="grid grid-cols-4 gap-2">
                <div v-for="(img, ii) in group.images" :key="ii"
                  class="group relative rounded-lg overflow-hidden border border-[var(--border-light)] bg-[var(--bg-subtle)]">
                  <img :src="img" class="w-full aspect-square object-cover" alt="生成图片" />
                  <div class="absolute inset-x-0 bottom-0 p-1.5 bg-gradient-to-t from-black/60 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex gap-1 flex-wrap justify-end">
                    <n-button size="tiny" type="primary" @click="handleSetMain(group.product_id, img)">主图</n-button>
                    <n-button size="tiny" ghost style="color:#fff;border-color:rgba(255,255,255,0.5);" @click="handleAddToGallery(group.product_id, img)">图库</n-button>
                    <n-button size="tiny" ghost style="color:#fff;border-color:rgba(255,255,255,0.5);" @click="handleRemoveBg(img)">去背景</n-button>
                    <n-button size="tiny" ghost style="color:#fff;border-color:rgba(255,255,255,0.5);" @click="handleDownload(img)">下载</n-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ====== 历史记录 ====== -->
    <div class="bg-white border border-[var(--border-light)] rounded-lg mb-6">
      <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--border-light)]">
        <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">生成历史</span>
        <span class="text-[11px] text-[var(--text-tertiary)]">共 {{ historyTotal }} 条</span>
      </div>
      <div class="p-4">
        <div v-if="historyItems.length === 0" class="text-center py-6 text-sm text-[var(--text-tertiary)]">暂无历史记录</div>
        <div v-else class="flex flex-col gap-2 max-h-[300px] overflow-y-auto">
          <div v-for="h in historyItems" :key="h.id"
            class="flex items-start gap-3 px-3 py-2.5 rounded-[6px] border border-[var(--border-light)] hover:bg-[var(--bg-hover)] transition-colors">
            <div class="flex gap-1 shrink-0">
              <div v-for="(url, ui) in (h.image_urls || []).slice(0, 3)" :key="ui"
                class="w-10 h-10 rounded-[4px] overflow-hidden bg-[var(--bg-subtle)]">
                <img :src="url" class="w-full h-full object-cover" />
              </div>
              <div v-if="(h.image_urls || []).length > 3" class="w-10 h-10 rounded-[4px] bg-[var(--bg-subtle)] flex items-center justify-center text-[11px] text-[var(--text-tertiary)]">
                +{{ h.image_urls.length - 3 }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-[12px] text-[var(--text-primary)] truncate">{{ h.prompt }}</div>
              <div class="flex items-center gap-2 mt-0.5">
                <n-tag size="tiny" :type="statusType(h.status)" round>{{ statusLabel(h.status) }}</n-tag>
                <span class="text-[10px] text-[var(--text-tertiary)]">{{ h.product_name || '' }}</span>
                <span class="text-[10px] text-[var(--text-tertiary)]">{{ h.created_at ? formatTime(h.created_at) : '' }}</span>
              </div>
            </div>
            <n-button size="tiny" quaternary @click="showPromptInForm(h)">复用</n-button>
          </div>
        </div>
      </div>
    </div>

    <!-- ====== 模板管理弹窗 ====== -->
    <n-modal v-model:show="showTemplateManager" title="Prompt 模板管理" preset="card" style="width:700px;max-width:90vw;">
      <template #header>
        <div class="text-base font-semibold">Prompt 模板管理</div>
      </template>
      <div class="space-y-3">
        <div v-if="templates.length === 0" class="text-center py-8 text-sm text-[var(--text-tertiary)]">
          暂无模板，可以在生成时"存为模板"
        </div>
        <div v-for="t in templates" :key="t.id"
          class="flex items-start gap-3 px-3 py-2.5 rounded-[6px] border border-[var(--border-light)]">
          <div class="flex-1 min-w-0">
            <div class="text-[13px] font-medium text-[var(--text-primary)]">{{ t.name }}</div>
            <div class="text-[11px] text-[var(--text-tertiary)] mt-0.5 truncate">{{ t.prompt }}</div>
            <div class="flex items-center gap-2 mt-1">
              <n-tag size="tiny">{{ t.style }}</n-tag>
              <span v-if="t.platform_code" class="text-[10px] text-[var(--accent)]">{{ t.platform_code }}</span>
              <span class="text-[10px] text-[var(--text-tertiary)]">已用 {{ t.usage_count }} 次</span>
            </div>
          </div>
          <div class="flex gap-1 shrink-0">
            <n-button size="tiny" quaternary @click="applyTemplate(t); showTemplateManager = false">使用</n-button>
            <n-popconfirm @positive-click="handleDeleteTemplate(t.id)">
              <template #trigger>
                <n-button size="tiny" quaternary type="error">删除</n-button>
              </template>
              确定删除模板「{{ t.name }}」？
            </n-popconfirm>
          </div>
        </div>
      </div>
    </n-modal>

    <!-- ====== 存为模板弹窗 ====== -->
    <n-modal v-model:show="showSaveTemplate" title="保存为模板" preset="card" style="width:450px;max-width:90vw;"
      @positive-click="handleSaveTemplate" positive-text="保存">
      <div class="space-y-3">
        <div>
          <label class="text-[12px] font-medium block mb-1">模板名称 *</label>
          <n-input v-model:value="newTemplateName" placeholder="例如：白底主图" />
        </div>
        <div>
          <label class="text-[12px] font-medium block mb-1">描述（可选）</label>
          <n-input v-model:value="newTemplateDesc" placeholder="简短描述模板用途" />
        </div>
        <div>
          <label class="text-[12px] font-medium block mb-1">关联平台（可选）</label>
          <n-select v-model:value="newTemplatePlatform" :options="platformOptions" placeholder="通用模板" clearable />
        </div>
        <div class="flex items-center gap-2">
          <n-checkbox v-model:checked="newTemplateShared">团队共享</n-checkbox>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { productApi } from '@/api'
import { imageGenApi } from '@/api/modules/imageGen'
import type { GenerateImageResponse, BatchGenerateResponse, BatchGenerateItem, GenHistoryItem, PromptTemplateItem } from '@/api/modules/imageGen'

const message = useMessage()

// ====== 数据 ======
const searchQuery = ref('')
const products = ref<any[]>([])
const selectedProductIds = ref<number[]>([])
const generating = ref(false)
const latestResults = ref<BatchGenerateItem[]>([])
const batchResult = ref<{ total: number; success: number; failed: number } | null>(null)
const historyItems = ref<GenHistoryItem[]>([])
const historyTotal = ref(0)
const templates = ref<PromptTemplateItem[]>([])
const activeTemplateId = ref<number | null>(null)
const showTemplateManager = ref(false)
const showSaveTemplate = ref(false)
const newTemplateName = ref('')
const newTemplateDesc = ref('')
const newTemplatePlatform = ref<string | null>(null)
const newTemplateShared = ref(true)

const formData = reactive({
  prompt: '',
  negative_prompt: '',
  style: 'product_white',
  size: '1024x1024',
  count: 1,
})

const styleOptions = [
  { label: '白底产品图', value: 'product_white' },
  { label: '场景图', value: 'scene' },
  { label: '模特展示', value: 'model' },
  { label: '3D 渲染', value: '3d_render' },
]

const sizeOptions = [
  { label: '1024×1024', value: '1024x1024' },
  { label: '768×1024', value: '768x1024' },
  { label: '1024×768', value: '1024x768' },
  { label: '1536×1024', value: '1536x1024' },
  { label: '1024×1536', value: '1024x1536' },
]

const platformOptions = [
  { label: '通用', value: null },
  { label: 'Ozon', value: 'ozon' },
  { label: 'Shopee', value: 'shopee' },
  { label: 'Wildberries', value: 'wildberries' },
]

// ====== 计算属性 ======
const filteredProducts = computed(() => {
  if (!searchQuery.value) return products.value
  const q = searchQuery.value.toLowerCase()
  return products.value.filter(p => (p.name || '').toLowerCase().includes(q))
})

/** 将最新生成结果按商品分组 */
const latestGrouped = computed(() => {
  const map = new Map<number, { product_id: number; product_name: string; images: string[] }>()
  for (const r of latestResults.value) {
    if (r.status !== 'done' || !r.images.length) continue
    if (!map.has(r.product_id)) {
      map.set(r.product_id, { product_id: r.product_id, product_name: r.product_name || `商品 #${r.product_id}`, images: [] })
    }
    map.get(r.product_id)!.images.push(...r.images)
  }
  return Array.from(map.values())
})

const totalLatestImages = computed(() => {
  return latestGrouped.value.reduce((s, g) => s + g.images.length, 0)
})

// ====== 工具函数 ======
function statusType(status: string) {
  return { pending: 'warning' as const, done: 'success' as const, failed: 'error' as const }[status] || 'default'
}
function statusLabel(status: string) {
  return { pending: '生成中', done: '已完成', failed: '失败' }[status] || status
}
function formatTime(ts: string) {
  try { return ts.slice(0, 16).replace('T', ' ') } catch { return ts }
}

// ====== 商品选择 ======
function toggleProduct(id: number) {
  const idx = selectedProductIds.value.indexOf(id)
  if (idx >= 0) selectedProductIds.value.splice(idx, 1)
  else selectedProductIds.value.push(id)
}
function selectAllFiltered() {
  for (const p of filteredProducts.value) {
    if (!selectedProductIds.value.includes(p.id)) {
      selectedProductIds.value.push(p.id)
    }
  }
}

// ====== 加载数据 ======
async function loadProducts() {
  try {
    const resp = await productApi.list({ page: 1, page_size: 100 })
    const data = resp.data as any
    const list = data?.records || (Array.isArray(data) ? data : data?.items || [])
    products.value = list
  } catch { message.warning('加载商品列表失败') }
}

async function loadTemplates() {
  try {
    const resp = await imageGenApi.listTemplates({ page_size: 50 })
    const data = resp.data as any
    templates.value = data?.items || []
  } catch { /* 静默 */ }
}

async function loadHistory() {
  // 带第一个选中商品的 history
  const pid = selectedProductIds.value[0]
  try {
    const resp = await imageGenApi.history({ product_id: pid, page: 1, page_size: 20 })
    const data = resp.data as any
    if (data?.items) { historyItems.value = data.items; historyTotal.value = data.total }
  } catch { /* 静默 */ }
}

// ====== 模板操作 ======
function applyTemplate(t: PromptTemplateItem) {
  formData.prompt = t.prompt
  formData.negative_prompt = t.negative_prompt || ''
  formData.style = t.style
  formData.size = t.size
  activeTemplateId.value = t.id
  message.success(`已应用模板「${t.name}」`)
}

async function handleSaveTemplate() {
  if (!newTemplateName.value.trim() || !formData.prompt) {
    message.warning('请输入模板名称')
    return
  }
  try {
    await imageGenApi.createTemplate({
      name: newTemplateName.value.trim(),
      description: newTemplateDesc.value || undefined,
      prompt: formData.prompt,
      negative_prompt: formData.negative_prompt || undefined,
      style: formData.style,
      size: formData.size,
      platform_code: newTemplatePlatform.value || undefined,
      is_shared: newTemplateShared.value,
    })
    message.success('模板已保存')
    showSaveTemplate.value = false
    newTemplateName.value = ''
    newTemplateDesc.value = ''
    newTemplatePlatform.value = null
    loadTemplates()
  } catch (e: any) {
    message.error(e?.response?.data?.message || '保存失败')
  }
}

async function handleDeleteTemplate(id: number) {
  try {
    await imageGenApi.deleteTemplate(id)
    message.success('模板已删除')
    loadTemplates()
  } catch { message.error('删除失败') }
}

// ====== 生图 ======
async function handleGenerate() {
  if (selectedProductIds.value.length === 0 || !formData.prompt) return

  generating.value = true
  latestResults.value = []
  batchResult.value = null

  try {
    if (selectedProductIds.value.length === 1) {
      // 单商品
      const resp = await imageGenApi.generate({
        product_id: selectedProductIds.value[0],
        prompt: formData.prompt,
        negative_prompt: formData.negative_prompt,
        style: formData.style,
        size: formData.size,
        count: formData.count,
      })
      const data = resp.data as unknown as GenerateImageResponse
      const item: BatchGenerateItem = {
        product_id: selectedProductIds.value[0],
        product_name: products.value.find(p => p.id === selectedProductIds.value[0])?.name,
        job_id: data.job_id,
        status: data.status,
        images: data.images || [],
        error: data.error,
      }
      latestResults.value = [item]
      if (data.status === 'done') {
        message.success(`生成完成，共 ${data.images.length} 张`)
      } else if (data.status === 'failed') {
        message.error(data.error || '生成失败')
      }
    } else {
      // 批量
      const resp = await imageGenApi.batchGenerate({
        product_ids: selectedProductIds.value,
        prompt: formData.prompt,
        negative_prompt: formData.negative_prompt,
        style: formData.style,
        size: formData.size,
        count: formData.count,
      })
      const data = resp.data as unknown as BatchGenerateResponse
      latestResults.value = data.results || []
      batchResult.value = { total: data.total, success: data.success, failed: data.failed }
      if (data.success > 0) message.success(`批量完成：${data.success}/${data.total} 个商品成功`)
      if (data.failed > 0) message.warning(`${data.failed} 个商品生成失败`)
    }

    // 使用模板后增加计数
    if (activeTemplateId.value) {
      try {
        await imageGenApi.listTemplates({ page_size: 0 })
      } catch { /* 忽略 */ }
    }

    loadHistory()
  } catch (e: any) {
    message.error(e?.response?.data?.message || e?.message || '生成失败')
  } finally {
    generating.value = false
  }
}

// ====== 图片操作 ======
async function handleSetMain(productId: number, imageUrl: string) {
  try {
    await imageGenApi.save({ product_id: productId, image_url: imageUrl, set_as_main: true })
    message.success('已设为主图')
  } catch { message.error('保存失败') }
}

async function handleAddToGallery(productId: number, imageUrl: string) {
  try {
    await imageGenApi.save({ product_id: productId, image_url: imageUrl })
    message.success('已加入图库')
  } catch { message.error('保存失败') }
}

async function handleRemoveBg(imageUrl: string) {
  try {
    const resp = await imageGenApi.removeBg(imageUrl)
    const data = resp.data as any
    if (data?.url) {
      window.open(data.url, '_blank')
      message.success('去背景完成，已在新标签页打开')
    } else {
      message.warning('去背景返回为空')
    }
  } catch (e: any) {
    message.error(e?.response?.data?.message || '去背景失败')
  }
}

function handleDownload(imageUrl: string) {
  const link = document.createElement('a')
  link.href = imageUrl
  link.download = imageUrl.split('/').pop() || 'image.jpg'
  link.target = '_blank'
  link.click()
}

function showPromptInForm(h: GenHistoryItem) {
  formData.prompt = h.prompt
  formData.style = h.style || 'product_white'
  activeTemplateId.value = null
  message.success('已复用该提示词')
}

// ====== 初始化 ======
onMounted(() => {
  loadProducts()
  loadTemplates()
  loadHistory()
})
</script>
