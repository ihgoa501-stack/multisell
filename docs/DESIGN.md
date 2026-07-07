# 凌镜 LingMirror — 设计系统

> **规范文档** — 所有 UI 决策以此为据。修改需经 Owner 或 UI lead 确认。

## 风格

| 维度 | 值 |
|------|----|
| **设计风格** | Flat Design（简约 2D，无装饰性阴影，干净线条） |
| **风格关键词** | SaaS, data-dense, modern, professional, icon-heavy, clean typography |
| **模式** | Dark 为主 + Light 为辅（双模式均需全量支持） |
| **适合场景** | 跨境电商 AI AgentOS，数据面板，运营后台 |
| **性能影响** | ⚡ 极高，阴影极少，渲染开销低 |
| **无障碍** | ✓ WCAG AA 目标 |

## 色彩体系

### CSS 变量映射

项目使用语义化缩写变量体系，定义在 `src/app/globals.css` 中：

| 变量 | 语义 | 用途 |
|------|------|------|
| `--bg` | Background | 页面背景 |
| `--s1` | Surface 01 | 卡片 / 面板 / 侧栏背景 |
| `--s2` | Surface 02 | 浮动元素 / 高亮悬停态 |
| `--s3` | Surface 03 | Skeleton 填充 |
| `--t1` | Text 01 | 主文字（最高对比度） |
| `--t2` | Text 02 | 次文字（次要信息） |
| `--t3` | Text 03 | 辅助文字（placeholder） |
| `--t4` | Text 04 | 最浅文字（分组标题） |
| `--i4` | Interactive | 焦点环 / 交互状态强调色 |
| `--bd` | Border | 标准边框 |
| `--bd2` | Border 02 | Hover 高亮边框 |

> **注意**：`globals.css` 只声明变量名，具体值由 `layout.tsx` 中的 `data-theme` 控制，在不同主题下映射不同色值。

### Ant Design Token 映射

| Token | 值 | 用途 |
|-------|-----|------|
| `colorPrimary` | `#6366F1` (Indigo) | 主操作色 / 链接 |
| `colorPrimaryHover` | `#818CF8` | 主色悬停 |
| `colorPrimaryActive` | `#4F46E5` | 主色按下 |
| `colorInfo` | `#6366F1` | 信息色 |
| `colorSuccess` | `#34D399` (Emerald) | 成功状态 |
| `colorWarning` | `#FBBF24` (Amber) | 警告状态 |
| `colorError` | `#F87171` (Red) | 错误 / 危险状态 |
| `colorLink` | `#818CF8` | 链接色 |
| `borderRadius` | 6px | 全局圆角 |
| `fontFamily` | `'DM Sans', sans-serif` | 正文字体 |

Ant Design 主题配置见 `frontend-next/src/components/layout/AntdProvider.tsx`。

### 功能色彩约定

| 用途 | Light 模式 | Dark 模式 |
|------|-----------|-----------|
| AI 模块强调色（AgentOS、AI 指挥中心） | `#7C3AED` (Purple) | `#A78BFA` |
| 关键 CTA / 紧急审批 | `#EC4899` (Pink) | `#F472B6` |
| 🟢 成功 | Ant Design `colorSuccess` | 同 |
| 🟡 警告 | Ant Design `colorWarning` | 同 |
| 🔴 错误 / 风险 | Ant Design `colorError` | 同 |
| 🔵 信息 | Ant Design `colorInfo` | 同 |

## 字体体系

| 角色 | 字体 | 权重 | 来源 |
|------|------|------|------|
| 正文字体 (body) | DM Sans | 400 / 500 / 600 / 700 | Google Fonts |
| 展示字体 (display) | DM Sans | 600 / 700 | 同一字族保证一致性 |
| 代码字体 | JetBrains Mono | 400 / 500 | Google Fonts |
| CSS 变量 | `--body` / `--ds` | — | `globals.css` 中引用 |

### Type Scale

| 级别 | 大小 | 权重 | 应用 |
|------|------|------|------|
| h1 | 32px (2rem) | 700 | 页面大标题 |
| h2 | 24px (1.5rem) | 600 | 区域标题 |
| h3 | 18px (1.125rem) | 600 | 卡片标题 |
| body | 14px (0.875rem) | 400 | 正文 |
| body-sm | 13px (0.8125rem) | 400 | 辅助文字 |
| caption | 12px (0.75rem) | 400 | 标签 / 注释 |
| code | 13px (0.8125rem) | 400 | 内联代码 / JSON |

## 间距体系

| Token | 值 | 用途 |
|-------|-----|------|
| `--space-xs` | 4px (0.25rem) | 微间距 |
| `--space-sm` | 8px (0.5rem) | 图标与文字间距 |
| `--space-md` | 16px (1rem) | 标准内边距 |
| `--space-lg` | 24px (1.5rem) | 卡片 / 章节内边距 |
| `--space-xl` | 32px (2rem) | 大段间距 |
| `--space-2xl` | 48px (3rem) | 章节间隔 |
| `--space-3xl` | 64px (4rem) | Hero 间距 |

> 使用基于 4px 的增量体系。不要使用 3px / 7px / 11px 等非常规值。

## 动画

| 参数 | 值 |
|------|-----|
| 微交互时长 | 150～300ms |
| 过渡时长 | 200～400ms |
| 缓动函数 | CSS `ease`（标准） |
| CSS 变量 | `--dur-micro`(150ms) / `--dur-short`(200ms) |

### 原则

- ✅ 只用 `transform`/`opacity` 做动画，不操作 `width`/`height`/`top`/`left`
- ✅ 进入动画用 `ease-out`，退出用 `ease-in`
- ✅ 尊重 `prefers-reduced-motion`（已在 `globals.css` 中实现）
- ✅ Hover 态用 150~300ms 平滑过渡
- ❌ 不要有装饰性无限循环动画
- ❌ 动画不阻挡用户交互

## 组件风格

### 按钮

- Ant Design Button，默认圆角 6px
- Primary: `colorPrimary`（Indigo `#6366F1`）
- Danger: `colorError`（Red `#F87171`）
- AI 模块特殊色：Purple `#7C3AED`

### 卡片 / SectionCard

- 背景 `var(--s1)`，边框 `1px solid var(--bd)`，圆角 8px
- Hover 时 `border-color` 过渡到 `var(--bd2)`
- 见 `src/components/ui/SectionCard.tsx`

### 表格

- Ant Design Table
- 表头背景 `var(--s1)`，表头文字 `var(--t3)`
- 行悬停背景 `var(--s1)`
- 分页标准 Ant Design

### 骨架屏

- 使用 `src/components/ui/PageSkeleton.tsx` 中的 `CardSkeleton` / `TableSkeleton` / `StatRowSkeleton`
- 脉冲动画 `skeleton-pulse` 1.4s，50% 不透明度变化
- 所有列表 / 卡片加载态必须显示骨架屏，不能用空白或 Spinner 替代

### 空状态

- Ant Design `<Empty>` 组件 + 引导性文字
- 提供"新建"或"去添加"操作按钮
- 不能显示空白页

## 布局

- 侧边栏导航（固定） + 顶部栏 + 内容区
- 内容区最大宽度无硬限制（自适应）
- 移动端：375px 断点优先，桌面：1440px 基准
- 固定侧边栏时，主内容区预留对应偏移量
- 无水平滚动

## 图标规范

| 规则 | 说明 |
|------|------|
| 图标库 | `@ant-design/icons` — 统一由 Ant Design 提供 |
| Style 一致性 | 导航栏用 outlined，功能按钮用 filled |
| ❌ 禁止 | 使用 Emoji 作为图标（🎨 🚀 ⚙️ 等） |
| ✅ 替代 | 用 SVG 图标（`@ant-design/icons`） |
| 触摸目标 | 最小 44×44px（含 padding） |

## 无障碍

- `:focus-visible` 全局启用（2px solid + `--i4` 色 + `--r2` 圆角）
- 表单必须关联 `<label>`，不得只用 placeholder
- 错误信息使用 `role="alert"` 或 `aria-live`
- 函数色（红/绿/黄）必须配合图标或文字，不单独靠颜色传达信息
- 颜色对比度最低 4.5:1（WCAG AA）
- 骨架屏使用 `aria-busy="true"` 和 `aria-label`

## Ant Design 组件覆写

见 `src/components/layout/AntdProvider.tsx` 中的 `getDesignTokens()` 函数。涉及：

- Layout（背景/侧栏色）
- Menu（选中态、悬停态、分组标题）
- Button（主按钮阴影）
- Typography（代码字体）
- Table（表头、边框、行悬停）

## Pre-Delivery Checklist

每次提交 UI 代码前验证：

- [ ] 不使用 Emoji 作为图标（用 SVG / Ant Design 图标）
- [ ] 所有交互元素有 `cursor:pointer`
- [ ] Hover / 悬停态有平滑过渡（150~300ms）
- [ ] Focus 态可见（`:focus-visible`）
- [ ] Light 模式下文字对比度 ≥ 4.5:1
- [ ] `prefers-reduced-motion` 受尊重
- [ ] 响应式：375px / 768px / 1024px / 1440px
- [ ] 固定导航不遮挡内容
- [ ] 移动端无水平滚动
- [ ] 空状态有引导性操作
- [ ] 表单有 `<label>`，错误紧邻输入框
- [ ] 所有列表/表格加载态使用骨架屏

## 变更记录

| 日期 | 变更 | 批准 |
|------|------|------|
| 2026-07-07 | 初始版 — UI/UX Pro Max 分析生成 | — |
