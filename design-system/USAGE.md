# 凌镜 LingMirror 设计系统使用文档

> **如何使用设计系统创建新页面、沿用设计 Token**

---

## 📖 设计系统文件结构

```
design-system/
├── index.html                  # 设计系统展示文档（可离线打开）
├── USAGE.md                   # 本文档（使用指南）
├── Dashboard-Enhanced.vue     # 数据仪表盘（增强版）
├── ProductList-Enhanced.vue   # 商品列表页（增强版）
├── ProductDetail-Enhanced.vue # 商品详情页（增强版）
├── AgentDashboard-Enhanced.vue # AI Agent 操作面板（增强版）
└── ListingManage-Enhanced.vue # 多平台刊登管理（增强版）
```

**项目集成文件：**

```
frontend/src/
├── config/theme.ts            # Naive UI 主题配置（已更新）
├── styles/fonts.css           # 字体导入文件
└── main.ts                    # 已导入字体（已修改）
```

---

## 🚀 新页面如何自动沿用设计系统？

### 原理说明

在 `App.vue` 中，整个应用已被 `<n-config-provider>` 包裹：

```vue
<!-- frontend/src/App.vue -->
<n-config-provider :theme-overrides="themeOverrides">
  <n-loading-bar-provider>
    <n-dialog-provider>
      <n-message-provider>
        <n-notification-provider>
          <router-view />
        </n-notification-provider>
      </n-message-provider>
    </n-dialog-provider>
  </n-loading-bar-provider>
</n-config-provider>
```

**这意味着：**
- ✅ 所有页面自动继承 `themeOverrides` 中定义的颜色、字体、圆角等
- ✅ 新增任何 `.vue` 页面，直接使用 Naive UI 组件即可
- ✅ 无需每个页面单独引入样式

---

## 🎨 在页面中使用设计 Token

### 方式一：使用 Naive UI 组件（推荐）

**直接使用组件，样式自动应用：**

```vue
<template>
  <!-- 按钮：自动使用主题配置的主色 -->
  <n-button type="primary">新增商品</n-button>
  
  <!-- 输入框：自动使用主题配置的圆角、边框色 -->
  <n-input v-model:value="search" placeholder="搜索商品..." />
  
  <!-- 卡片：自动使用主题配置的阴影、圆角 -->
  <n-card title="商品统计">
    内容区
  </n-card>
</template>
```

### 方式二：在 CSS 中使用设计变量

**当需要自定义样式时，使用 CSS 变量：**

```vue
<template>
  <div class="custom-banner">
    自定义横幅
  </div>
</template>

<style scoped>
.custom-banner {
  /* 使用设计 Token */
  background: var(--color-brand-50);      /* 浅蓝背景 */
  color: var(--color-brand-700);           /* 深蓝文字 */
  border-radius: var(--radius-lg);         /* 12px 圆角 */
  padding: var(--space-6);                 /* 24px 内边距 */
  box-shadow: var(--shadow-md);             /* 中等阴影 */
  transition: var(--transition-normal);     /* 300ms 过渡 */
}

.custom-banner:hover {
  box-shadow: var(--shadow-lg);            /* 悬浮时加深阴影 */
  transform: translateY(-2px);             /* 微小上移 */
}
</style>
```

**需要在页面中导入设计变量：**

```typescript
// 在 main.ts 或页面中导入
import '@/styles/design-tokens.css'
```

---

## 🎯 设计 Token 速查表

### 颜色变量

| 用途 | CSS 变量 | 示例值 |
|------|----------|--------|
| **品牌主色** | `var(--color-brand-500)` | `#0ea5e9` |
| 品牌深色 | `var(--color-brand-600)` | `#0284c7` |
| 品牌浅色 | `var(--color-brand-50)` | `#f0f9ff` |
| **强调色** | `var(--color-accent-500)` | `#8b5cf6` |
| **成功色** | `var(--color-success-500)` | `#10b981` |
| **警告色** | `var(--color-warning-500)` | `#f59e0b` |
| **错误色** | `var(--color-error-500)` | `#ef4444` |
| **中性文字** | `var(--color-neutral-900)` | `#171717` |
| 辅助文字 | `var(--color-neutral-500)` | `#737373` |
| 边框颜色 | `var(--color-neutral-200)` | `#e5e5e5` |
| 背景颜色 | `var(--color-neutral-50)` | `#fafafa` |

### 间距变量

| 用途 | CSS 变量 | 像素值 |
|------|----------|--------|
| 微小间距 | `var(--space-1)` | 4px |
| 小间距 | `var(--space-2)` | 8px |
| 基础间距 | `var(--space-4)` | 16px |
| 中间距 | `var(--space-6)` | 24px |
| 大间距 | `var(--space-8)` | 32px |
| 超大间距 | `var(--space-12)` | 48px |

### 圆角变量

| 用途 | CSS 变量 | 像素值 |
|------|----------|--------|
| 小圆角 | `var(--radius-sm)` | 4px |
| 基础圆角 | `var(--radius-md)` | 6px |
| 大圆角 | `var(--radius-lg)` | 12px |
| 圆形 | `var(--radius-full)` | 9999px |

### 字体变量

| 用途 | CSS 变量 | 字体族 |
|------|----------|--------|
| 英文标题 | `var(--font-display)` | Outfit, sans-serif |
| 英文正文 | `var(--font-body)` | Source Sans 3, sans-serif |
| 中文字体 | - | 苹方, 思源黑体, sans-serif |
| 等宽字体 | `var(--font-mono)` | JetBrains Mono, monospace |

---

## 🧩 常用组件使用示例

### 1. 数据卡片（KPI 展示）

```vue
<template>
  <n-card class="stat-card" :class="`stat-card--${color}`">
    <div class="stat-card__header">
      <span class="stat-card__title">{{ title }}</span>
      <n-icon :component="icon" />
    </div>
    <div class="stat-card__value">{{ value }}</div>
    <div class="stat-card__trend">
      <n-icon :class="trendUp ? 'trend-up' : 'trend-down'" />
      <span>{{ trendValue }}</span>
    </div>
  </n-card>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  value: string | number
  icon: any
  color: 'blue' | 'purple' | 'green' | 'amber'
  trendUp: boolean
  trendValue: string
}>()
</script>

<style scoped>
.stat-card {
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  background: white;
  box-shadow: var(--shadow-sm);
  transition: var(--transition-normal);
}

.stat-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.stat-card__value {
  font-family: var(--font-display);
  font-size: 2rem;
  font-weight: 700;
  color: var(--color-neutral-900);
  margin: var(--space-2) 0;
}

.stat-card--blue { border-left: 4px solid var(--color-brand-500); }
.stat-card--purple { border-left: 4px solid var(--color-accent-500); }
.stat-card--green { border-left: 4px solid var(--color-success-500); }
.stat-card--amber { border-left: 4px solid var(--color-warning-500); }

.trend-up { color: var(--color-success-500); }
.trend-down { color: var(--color-error-500); }
</style>
```

### 2. 状态徽章

```vue
<template>
  <n-tag
    :type="tagType"
    round
    size="small"
  >
    {{ statusText }}
  </n-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: 'active' | 'draft' | 'syncing' | 'error'
}>()

const tagType = computed(() => {
  const map = {
    active: 'success',
    draft: 'default',
    syncing: 'info',
    error: 'error'
  }
  return map[props.status]
})

const statusText = computed(() => {
  const map = {
    active: '已上架',
    draft: '草稿',
    syncing: '同步中',
    error: '异常'
  }
  return map[props.status]
})
</script>
```

### 3. 操作工具栏

```vue
<template>
  <n-space justify="space-between" align="center" class="toolbar">
    <n-space>
      <n-button type="primary" @click="handleCreate">
        <template #icon>
          <n-icon :component="AddIcon" />
        </template>
        新增
      </n-button>
      <n-button :disabled="!hasSelection" @click="handleBatch">
        批量操作
      </n-button>
    </n-space>
    
    <n-space>
      <n-input
        v-model:value="search"
        placeholder="搜索..."
        clearable
        style="width: 280px"
      >
        <template #prefix>
          <n-icon :component="SearchIcon" />
        </template>
      </n-input>
      <n-select
        v-model:value="filter"
        :options="filterOptions"
        placeholder="筛选"
        style="width: 140px"
      />
    </n-space>
  </n-space>
</template>

<style scoped>
.toolbar {
  padding: var(--space-4) 0;
  margin-bottom: var(--space-6);
}
</style>
```

---

## 📐 页面布局规范

### 标准页面结构

```vue
<template>
  <div class="page-container">
    <!-- 1. 页面标题区 -->
    <n-page-header
      title="页面标题"
      subtitle="页面描述文字"
    >
      <template #extra>
        <n-button type="primary">主要操作</n-button>
      </template>
    </n-page-header>

    <!-- 2. 统计卡片区（可选） -->
    <n-grid :cols="4" :x-gap="16" :y-gap="16" class="stat-grid">
      <n-gi v-for="stat in stats" :key="stat.label">
        <StatCard v-bind="stat" />
      </n-gi>
    </n-grid>

    <!-- 3. 工具栏 -->
    <div class="toolbar">
      <!-- 按钮组 + 搜索/筛选 -->
    </div>

    <!-- 4. 内容区 -->
    <n-card>
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
      />
    </n-card>
  </div>
</template>

<style scoped>
.page-container {
  padding: var(--space-6);
  max-width: 1400px;
  margin: 0 auto;
}

.stat-grid {
  margin: var(--space-6) 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: var(--space-6) 0;
}
</style>
```

---

## ♿ 无障碍检查清单

在提交新页面前，请检查：

- [ ] **颜色对比度** ≥ 4.5:1（使用 https://webaim.org/resources/contrastchecker/ 检查）
- [ ] **所有图片**有 `alt` 属性
- [ ] **表单元素**有 `<label>` 或 `placeholder`
- [ ] **交互元素**可用 Tab 键访问
- [ ] **焦点状态**清晰可见（不依赖 `outline: none`）
- [ ] **触摸目标** ≥ 44px × 44px
- [ ] **文字大小**支持浏览器缩放至 200%

---

## 🔄 如何应用增强版页面？

### 方式一：直接替换（需要备份）

```bash
# 1. 备份原文件
cp frontend/src/views/dashboard/Dashboard.vue \
   frontend/src/views/dashboard/Dashboard.vue.bak

# 2. 应用增强版
cp design-system/Dashboard-Enhanced.vue \
   frontend/src/views/dashboard/Dashboard.vue

# 3. 重复其他页面...
```

### 方式二：手动整合（推荐）

将增强版中的设计改进点（如状态徽章、空状态、AI 提示栏等）**手动整合**到现有页面中，保持现有业务逻辑不变。

---

## 📞 设计系统维护

### 修改设计 Token

如果需要调整颜色、字体等：

1. 修改 `frontend/src/config/theme.ts` 中的 `themeOverrides`
2. 同步更新 `design-system/index.html` 中的 `:root` 变量（用于展示文档）
3. 所有页面自动继承修改

### 添加新组件样式

在 `frontend/src/styles/` 目录下创建新的 CSS 文件：

```css
/* frontend/src/styles/components.css */

/* 自定义组件样式 */
.custom-component {
  /* 使用设计 Token */
}
```

然后在 `main.ts` 中导入：

```typescript
import '@/styles/components.css'
```

---

## 📚 参考资料

- **设计系统展示**：用浏览器打开 `design-system/index.html`
- **Naive UI 文档**：https://www.naiveui.com/zh-CN/os-theme
- **WCAG AA 标准**：https://www.w3.org/WAI/WCAG21/quickref/#contrast-minimum
- **CSS 变量列表**：查看 `design-system/index.html` 中的 `:root` 部分

---

**最后更新**：2026-06-18  
**维护者**：凌镜 LingMirror 设计团队
