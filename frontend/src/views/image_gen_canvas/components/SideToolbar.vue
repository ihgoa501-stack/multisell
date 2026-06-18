<template>
  <div class="side-toolbar">
    <n-tooltip placement="right" trigger="hover">
      <template #trigger>
        <n-button :type="activeTool === 'select' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'select')">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z"/><path d="M13 13l6 6"/></svg>
          </template>
        </n-button>
      </template>
      选择
    </n-tooltip>

    <n-tooltip placement="right" trigger="hover">
      <template #trigger>
        <n-button :type="activeTool === 'text' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'text')">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="4 7 4 4 20 4 20 7"/><line x1="9" y1="20" x2="15" y2="20"/><line x1="12" y1="4" x2="12" y2="20"/></svg>
          </template>
        </n-button>
      </template>
      文字
    </n-tooltip>

    <n-divider style="margin:8px 0" />

    <n-tooltip placement="right" trigger="hover">
      <template #trigger>
        <n-button size="small" quaternary @click="$emit('save')">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
          </template>
        </n-button>
      </template>
      保存
    </n-tooltip>

    <n-tooltip placement="right" trigger="hover">
      <template #trigger>
        <n-button size="small" quaternary @click="$emit('export')">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
          </template>
        </n-button>
      </template>
      导出
    </n-tooltip>

    <n-divider style="margin:8px 0" />

    <n-tooltip placement="right" trigger="hover">
      <template #trigger>
        <n-button :type="activeTool === 'video' ? 'primary' : 'tertiary'" size="small" @click="$emit('selectTool', 'video')">
          <template #icon>
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>
          </template>
        </n-button>
      </template>
      视频
    </n-tooltip>
  </div>
</template>

<script setup lang="ts">
defineProps<{ activeTool: string }>()
defineEmits<{
  selectTool: [tool: string]
  save: []
  export: []
}>()
</script>

<style scoped>
.side-toolbar {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 6px;
  background: white;
  border-right: 1px solid #e5e7eb;
  width: 44px;
  align-items: center;
  z-index: 5;
}
</style>
