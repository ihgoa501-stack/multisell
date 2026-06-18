<template>
  <div class="video-panel">
    <div class="panel-header">
      <span class="text-[12px] font-semibold text-[var(--text-secondary)] uppercase tracking-wide">视频</span>
    </div>
    <div class="p-3 space-y-3">
      <n-tabs v-model:value="mode" size="small" type="segment">
        <n-tab name="ai">AI 生成</n-tab>
        <n-tab name="slideshow">Slideshow</n-tab>
      </n-tabs>

      <!-- AI 视频模式 -->
      <div v-if="mode === 'ai'" class="space-y-2">
        <n-input
          v-model:value="aiPrompt"
          type="textarea"
          :rows="3"
          placeholder="描述视频内容，例如：产品 360 度旋转展示，白色背景"
        />
        <n-button
          size="small"
          type="primary"
          block
          :loading="videoGenerating"
          :disabled="!aiPrompt.trim()"
          @click="handleGenerateVideo"
        >
          生成视频
        </n-button>
        <div v-if="videoJobId" class="text-[11px] text-[var(--text-tertiary)] flex items-center gap-1">
          <n-spin size="small" />
          任务处理中...
        </div>
      </div>

      <!-- Slideshow 模式 -->
      <div v-if="mode === 'slideshow'" class="space-y-2">
        <div class="text-[12px] text-[var(--text-secondary)] font-medium">
          已选帧: {{ selectedFrameUrls.length }}
          <n-button v-if="selectedFrameUrls.length > 0" size="tiny" text type="error" @click="selectedFrameUrls = []">
            清空
          </n-button>
        </div>

        <!-- 帧预览缩略图 -->
        <div class="flex flex-wrap gap-1.5 min-h-[60px] p-2 rounded-[6px] border border-dashed border-[var(--border-light)]">
          <div v-for="(url, i) in selectedFrameUrls" :key="i"
            class="relative w-14 h-14 rounded-[4px] overflow-hidden border border-[var(--border-light)] group">
            <img :src="url" class="w-full h-full object-cover" />
            <button
              class="absolute -top-1.5 -right-1.5 w-4 h-4 bg-red-500 text-white rounded-full text-[9px] flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
              @click="removeFrame(i)"
            >×</button>
          </div>
          <div v-if="selectedFrameUrls.length === 0" class="w-full text-center text-[11px] text-[var(--text-tertiary)] py-3">
            在画布中选中图片，右键「设为视频帧」
          </div>
        </div>

        <div class="flex items-center gap-2">
          <span class="text-[12px] text-[var(--text-secondary)] shrink-0">每帧</span>
          <n-input-number v-model:value="durationPerFrame" :min="0.5" :max="10" :step="0.5" size="tiny" style="width:80px" />
          <span class="text-[12px] text-[var(--text-secondary)]">秒</span>
        </div>

        <n-button
          size="small"
          type="primary"
          block
          :loading="videoGenerating"
          :disabled="selectedFrameUrls.length < 2"
          @click="handleCreateSlideshow"
        >
          合成视频 ({{ selectedFrameUrls.length }} 帧)
        </n-button>
      </div>

      <!-- 视频预览 -->
      <div v-if="videoUrl" class="mt-3">
        <div class="text-[12px] font-medium text-[var(--text-secondary)] mb-1">预览</div>
        <video :src="videoUrl" controls class="w-full rounded-[6px]" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  videoGenerating: boolean
  videoJobId: string | null
  videoUrl: string | null
}>()

const emit = defineEmits<{
  generateVideo: [prompt: string]
  createSlideshow: [urls: string[], duration: number]
}>()

const mode = ref<'ai' | 'slideshow'>('ai')
const aiPrompt = ref('')
const selectedFrameUrls = ref<string[]>([])
const durationPerFrame = ref(2)

function addFrame(url: string) {
  if (!selectedFrameUrls.value.includes(url)) {
    selectedFrameUrls.value.push(url)
  }
}

function removeFrame(index: number) {
  selectedFrameUrls.value.splice(index, 1)
}

function handleGenerateVideo() {
  if (aiPrompt.value.trim()) {
    emit('generateVideo', aiPrompt.value.trim())
  }
}

function handleCreateSlideshow() {
  if (selectedFrameUrls.value.length >= 2) {
    emit('createSlideshow', [...selectedFrameUrls.value], durationPerFrame.value)
  }
}

defineExpose({ addFrame })
</script>

<style scoped>
.video-panel {
  width: 260px;
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
</style>
