<template>
  <div class="canvas-editor h-full flex flex-col">
    <!-- 顶栏 -->
    <div class="flex items-center gap-2 px-4 py-2 bg-white border-b border-[var(--ant-color-border)] shrink-0 z-10">
      <a-select
        v-model:value="selectedProductId"
        :options="productOptions"
        placeholder="选择商品"
        show-search
        style="width:200px"
        size="small"
        @update:value="onProductChange"
      />
      <a-select
        v-model:value="activeCanvasId"
        :options="canvasOptions"
        placeholder="保存的画布"
        style="width:180px"
        size="small"
        allow-clear
        @update:value="onCanvasSelect"
      />
      <a-button size="small" :disabled="!selectedProductId" @click="showGenerateModal = true">
        <template #icon>
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/></svg>
        </template>
        生图
      </a-button>
      <a-button size="small" :disabled="!canvasReady" @click="addTextToCanvas">
        <template #icon>
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 7 4 4 20 4 20 7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
        </template>
        文字
      </a-button>
      <div class="flex-1" />
      <a-button size="small" type="text" @click="saveCanvas" :disabled="!selectedProductId">保存</a-button>
      <a-button size="small" type="text" @click="exportCanvas">导出</a-button>
    </div>

    <!-- 主体 -->
    <div class="flex flex-1 overflow-hidden">
      <SideToolbar
        :activeTool="activeTool"
        @selectTool="onSelectTool"
        @save="saveCanvas"
        @export="exportCanvas"
      />

      <!-- 画布区域（含 drop） -->
      <div
        class="flex-1 relative"
        @drop="onDrop"
        @dragover.prevent
      >
        <FabricCanvas ref="fabricCanvasRef" @requestEdit="onRequestEdit" />
      </div>

      <!-- 右侧面板 -->
      <ImagePanel
        v-if="activeTool !== 'video'"
        ref="imagePanelRef"
        :images="generatedImages"
        @openGenerate="showGenerateModal = true"
      />
      <VideoPanel
        v-if="activeTool === 'video'"
        ref="videoPanelRef"
        :videoGenerating="videoGenerating"
        :videoJobId="videoJobId"
        :videoUrl="videoUrl"
        @generateVideo="handleGenerateVideo"
        @createSlideshow="handleCreateSlideshow"
      />
    </div>

    <!-- 生图弹窗 -->
    <a-modal v-model:open="showGenerateModal" title="AI 生图" style="width:540px;max-width:90vw;" :mask-closable="false" :footer="null">
      <div class="space-y-3">
        <a-textarea v-model:value="genPrompt" :rows="2" placeholder="描述你想要的图片内容，例如：白色耳机在木质桌面上，自然光..." />
        <div class="grid grid-cols-3 gap-2">
          <a-select v-model:value="genStyle" :options="styleOptions" size="small" style="width: 100%" />
          <a-select v-model:value="genSize" :options="sizeOptions" size="small" style="width: 100%" />
          <a-input-number v-model:value="genCount" :min="1" :max="4" size="small" style="width:100%" :addon-before="'数量'" />
        </div>
        <a-button type="primary" block :loading="generating" :disabled="!genPrompt.trim() || !selectedProductId" @click="handleGenerate">
          生成
        </a-button>
      </div>
    </a-modal>

    <!-- 编辑弹窗 -->
    <a-modal v-model:open="showEditModal" title="AI 编辑" style="width:480px;max-width:90vw;" :footer="null">
      <div class="space-y-3">
        <img v-if="editImageUrl" :src="editImageUrl" class="w-full max-h-48 object-contain rounded-[6px] bg-gray-100" />
        <a-textarea v-if="editMode !== 'remove-bg'" v-model:value="editPrompt" :rows="2"
          :placeholder="editMode === 'inpaint' ? '描述要重绘的区域内容...' : '描述扩展区域内容...'"
        />
        <div v-if="editMode === 'outpaint'" class="flex gap-2">
          <a-button v-for="d in directions" :key="d.value" size="small"
            :type="editDirection === d.value ? 'primary' : 'default'"
            @click="editDirection = d.value"
          >{{ d.label }}</a-button>
        </div>
        <a-button type="primary" block :loading="generating"
          :disabled="editMode !== 'remove-bg' && !editPrompt.trim()"
          @click="confirmEdit"
        >
          {{ editMode === 'remove-bg' ? '确认去背景' : editMode === 'inpaint' ? '确认重绘' : '确认扩图' }}
        </a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { productApi } from '@/api'
import { imageGenApi } from '@/api/modules/imageGen'
import { imageGenCanvasApi } from '@/api/modules/imageGenCanvas'
import { useGeneration } from '@/views/image_gen_canvas/composables/useGeneration'
import SideToolbar from '@/views/image_gen_canvas/components/SideToolbar.vue'
import FabricCanvas from '@/views/image_gen_canvas/components/FabricCanvas.vue'
import ImagePanel from '@/views/image_gen_canvas/components/ImagePanel.vue'
import VideoPanel from '@/views/image_gen_canvas/components/VideoPanel.vue'

const fabricCanvasRef = ref<InstanceType<typeof FabricCanvas> | null>(null)
const imagePanelRef = ref<InstanceType<typeof ImagePanel> | null>(null)
const videoPanelRef = ref<InstanceType<typeof VideoPanel> | null>(null)

const {
  generating, videoGenerating, videoJobId, videoUrl,
  inpaint, outpaint, startVideoGen, createSlideshow: doCreateSlideshow,
} = useGeneration()

// 商品/画布
const selectedProductId = ref<number | null>(null)
const activeCanvasId = ref<number | null>(null)
const products = ref<any[]>([])
const canvases = ref<any[]>([])
const activeTool = ref('select')
const generatedImages = ref<{ url: string; name?: string }[]>([])
const canvasReady = computed(() => fabricCanvasRef.value !== null)

const productOptions = computed(() =>
  products.value.map((p: any) => ({ label: p.name || `商品 #${p.id}`, value: p.id }))
)

const canvasOptions = computed(() =>
  canvases.value.map((c: any) => ({ label: c.name, value: c.id }))
)

// 生图弹窗
const showGenerateModal = ref(false)
const genPrompt = ref('')
const genStyle = ref('product_white')
const genSize = ref('1024x1024')
const genCount = ref(1)

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

// 编辑弹窗
const showEditModal = ref(false)
const editMode = ref<'inpaint' | 'outpaint' | 'remove-bg'>('inpaint')
const editImageUrl = ref('')
const editPrompt = ref('')
const editDirection = ref('right')
const directions = [
  { label: '← 左', value: 'left' },
  { label: '→ 右', value: 'right' },
  { label: '↑ 上', value: 'top' },
  { label: '↓ 下', value: 'bottom' },
]

// 数据加载
async function loadProducts() {
  try {
    const resp = await productApi.list({ page: 1, page_size: 100 })
    const data = resp.data as any
    products.value = data?.records || data?.items || []
  } catch {
    message.warning('加载商品失败')
  }
}

async function loadCanvases() {
  if (!selectedProductId.value) return
  try {
    const resp = await imageGenCanvasApi.listCanvases(selectedProductId.value)
    const data = resp.data as any
    canvases.value = data?.items || []
  } catch { /* silent */ }
}

function onProductChange() {
  activeCanvasId.value = null
  loadCanvases()
}

async function onCanvasSelect(canvasId: number | null) {
  if (!canvasId || !fabricCanvasRef.value) return
  try {
    const resp = await imageGenCanvasApi.loadCanvas(canvasId)
    const data = resp.data as any
    if (data?.layers && data.layers.length > 0) {
      // The layers array has a root layer with fabric_json
      const rootLayer = data.layers[0]
      if (rootLayer?.fabric_json) {
        fabricCanvasRef.value.loadFromJSON(rootLayer.fabric_json)
        message.success('画布已加载')
        return
      }
    }
    message.info('画布为空')
  } catch {
    message.error('加载画布失败')
  }
}

// 保存/导出
async function saveCanvas() {
  if (!selectedProductId.value || !fabricCanvasRef.value) return
  try {
    const json = fabricCanvasRef.value.getCanvasJSON()
    const payload = {
      product_id: selectedProductId.value,
      name: `画布 ${new Date().toLocaleString('zh-CN')}`,
      layers: [{ id: 'root', type: 'image' as const, fabric_json: json }],
    }
    await imageGenCanvasApi.saveCanvas(payload)
    message.success('画布已保存')
    loadCanvases()
  } catch {
    message.error('保存失败')
  }
}

function exportCanvas() {
  if (!fabricCanvasRef.value) return
  const dataUrl = fabricCanvasRef.value.exportImage('png')
  const link = document.createElement('a')
  link.download = 'canvas.png'
  link.href = dataUrl
  link.click()
}

// 工具切换
function onSelectTool(tool: string) {
  activeTool.value = tool
  if (tool === 'text') {
    addTextToCanvas()
  }
}

function addTextToCanvas() {
  if (!fabricCanvasRef.value) return
  fabricCanvasRef.value.addText('双击编辑文字')
  activeTool.value = 'select'
}

// 拖入画布
function onDrop(e: DragEvent) {
  const url = e.dataTransfer?.getData('text/plain')
  if (url && fabricCanvasRef.value) {
    fabricCanvasRef.value.addImage(url)
  }
}

// 生图
async function handleGenerate() {
  if (!selectedProductId.value || !genPrompt.value.trim()) return
  try {
    const resp = await imageGenApi.generate({
      product_id: selectedProductId.value,
      prompt: genPrompt.value.trim(),
      style: genStyle.value,
      size: genSize.value,
      count: genCount.value,
    })
    const data = resp.data as any
    if (data?.images) {
      for (const url of data.images) {
        generatedImages.value.push({ url, name: genPrompt.value.slice(0, 30) })
        if (fabricCanvasRef.value) {
          fabricCanvasRef.value.addImage(url)
        }
      }
      message.success(`生成完成，共 ${data.images.length} 张`)
    }
    showGenerateModal.value = false
  } catch (e: any) {
    message.error(e?.response?.data?.message || '生成失败')
  }
}

// 编辑
async function onRequestEdit(action: string) {
  if (!fabricCanvasRef.value?.selectedObject) {
    message.warning('请先选中画布中的图片')
    return
  }
  const obj = fabricCanvasRef.value.selectedObject as any
  editImageUrl.value = obj.toDataURL({ multiplier: 1 })

  if (action === 'inpaint' || action === 'outpaint') {
    editMode.value = action
    editPrompt.value = ''
    showEditModal.value = true
  } else if (action === 'remove-bg') {
    try {
      const resp = await imageGenApi.removeBg(editImageUrl.value)
      const data = resp.data as any
      if (data?.url) {
        fabricCanvasRef.value.addImage(data.url)
        generatedImages.value.push({ url: data.url, name: '去背景' })
        message.success('去背景完成')
      }
    } catch {
      message.error('去背景失败')
    }
  }
}

async function confirmEdit() {
  let result: string | null = null
  if (editMode.value === 'inpaint') {
    result = await inpaint(editImageUrl.value, '', editPrompt.value)
  } else if (editMode.value === 'outpaint') {
    result = await outpaint(editImageUrl.value, editDirection.value, editPrompt.value)
  }
  if (result && fabricCanvasRef.value) {
    fabricCanvasRef.value.addImage(result)
    generatedImages.value.push({ url: result, name: editPrompt.value.slice(0, 30) })
  }
  showEditModal.value = false
}

// 视频
async function handleGenerateVideo(prompt: string) {
  await startVideoGen(prompt)
}

async function handleCreateSlideshow(urls: string[], duration: number) {
  const result = await doCreateSlideshow(urls, duration)
  if (result) message.success('视频合成完成')
}

// 初始化
onMounted(() => {
  loadProducts()
})
</script>

<style scoped>
.canvas-editor {
  height: calc(100vh - 48px);
}
</style>
