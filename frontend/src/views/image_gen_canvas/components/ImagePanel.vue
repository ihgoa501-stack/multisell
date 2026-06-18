<template>
  <div class="image-panel">
    <div class="panel-header">
      <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">图片</span>
    </div>
    <div class="panel-body">
      <div class="px-3 pt-3 pb-2 space-y-2">
        <n-input v-model:value="searchQuery" size="tiny" placeholder="搜索商品..." />
        <n-button size="tiny" block secondary @click="$emit('openGenerate')">
          <template #icon>
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/></svg>
          </template>
          生成新图片
        </n-button>
      </div>
      <div class="grid grid-cols-2 gap-1.5 px-3 pb-3">
        <div v-for="(img, i) in images" :key="i"
          class="relative aspect-square rounded-[4px] overflow-hidden border border-[var(--border-light)] cursor-grab active:cursor-grabbing bg-[var(--bg-subtle)] group"
          draggable="true"
          @dragstart="onDragStart($event, img.url)">
          <img :src="img.url" class="w-full h-full object-cover" loading="lazy" />
          <div class="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors" />
        </div>
        <div v-if="images.length === 0" class="col-span-2 text-center py-8 text-[11px] text-[var(--text-tertiary)]">
          暂无生成图片<br/>
          <span class="text-[10px]">点击"生成新图片"开始</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  images: { url: string; name?: string }[]
}>()

defineEmits<{
  openGenerate: []
}>()

const searchQuery = ref('')

function onDragStart(e: DragEvent, url: string) {
  if (e.dataTransfer) {
    e.dataTransfer.setData('text/plain', url)
    e.dataTransfer.effectAllowed = 'copy'
  }
}
</script>

<style scoped>
.image-panel {
  width: 200px;
  background: white;
  border-left: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  z-index: 5;
}
.panel-header {
  padding: 12px 16px 8px;
  border-bottom: 1px solid #e5e7eb;
}
.panel-body {
  flex: 1;
  overflow-y: auto;
}
</style>
