<template>
  <div ref="containerRef" class="fabric-canvas-container">
    <canvas ref="canvasEl" />
    <!-- 左上角缩放控制 -->
    <div class="absolute top-3 left-3 flex items-center gap-1 z-10">
      <n-button size="tiny" quaternary @click="fitToScreen">
        <template #icon>
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/>
          </svg>
        </template>
      </n-button>
      <n-button size="tiny" quaternary @click="setZoom(1)">100%</n-button>
      <span class="text-[11px] text-[var(--text-tertiary)] select-none">{{ Math.round(zoom * 100) }}%</span>
    </div>
    <!-- 右键菜单 -->
    <n-dropdown
      :show="contextMenuVisible"
      placement="bottom-start"
      trigger="manual"
      :x="contextMenuX"
      :y="contextMenuY"
      :options="contextMenuOptions"
      @select="handleContextMenu"
      @clickoutside="contextMenuVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useFabricCanvas } from '../composables/useFabricCanvas'
import type { DropdownOption } from 'naive-ui'

const emit = defineEmits<{
  requestEdit: [action: string]
}>()

const containerRef = ref<HTMLDivElement | null>(null)
const canvasEl = ref<HTMLCanvasElement | null>(null)

const {
  canvas, zoom, selectedObject,
  addImage, addText, deleteSelected,
  bringForward, sendBackward,
  exportImage, getCanvasJSON, loadFromJSON,
  setZoom, fitToScreen,
} = useFabricCanvas(canvasEl)

const contextMenuVisible = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuOptions: DropdownOption[] = [
  { label: '置顶', key: 'bring-forward' },
  { label: '置底', key: 'send-backward' },
  { label: '复制', key: 'duplicate' },
  { label: '删除', key: 'delete' },
  { type: 'divider' as const },
  { label: '局部重绘', key: 'inpaint' },
  { label: '扩图', key: 'outpaint' },
  { label: '去背景', key: 'remove-bg' },
  { type: 'divider' as const },
  { label: '导出此图层', key: 'export-layer' },
]

function handleContextMenu(key: string) {
  contextMenuVisible.value = false
  if (key === 'delete') deleteSelected()
  else if (key === 'bring-forward') bringForward()
  else if (key === 'send-backward') sendBackward()
  else if (key === 'duplicate') duplicateSelected()
  else if (key === 'export-layer') exportSelectedLayer()
  else emit('requestEdit', key)
}

function duplicateSelected() {
  if (!canvas.value || !selectedObject.value) return
  selectedObject.value.clone((cloned: fabric.Object) => {
    cloned.set({
      left: (cloned.left || 0) + 20,
      top: (cloned.top || 0) + 20,
    })
    canvas.value!.add(cloned)
    canvas.value!.setActiveObject(cloned)
    canvas.value!.renderAll()
  })
}

function exportSelectedLayer() {
  if (!selectedObject.value) return
  const dataUrl = (selectedObject.value as any).toDataURL({ multiplier: 2 })
  const link = document.createElement('a')
  link.download = 'layer.png'
  link.href = dataUrl
  link.click()
}

function onRightClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target.closest('.n-dropdown')) return
  e.preventDefault()
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuVisible.value = true
}

onMounted(() => {
  document.addEventListener('contextmenu', onRightClick)
})

onUnmounted(() => {
  document.removeEventListener('contextmenu', onRightClick)
})

defineExpose({
  addImage,
  addText,
  getCanvasJSON,
  loadFromJSON,
  exportImage,
  setZoom,
  fitToScreen,
  canvas,
  selectedObject,
})
</script>

<style scoped>
.fabric-canvas-container {
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
  background: #e8e8e8;
}
.fabric-canvas-container :deep(canvas) {
  display: block;
}
</style>
