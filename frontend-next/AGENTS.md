<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

<!-- BEGIN:frontend-components -->
## Frontend Components

新建页面优先使用以下组件，不要手动写 `div` + `style={{}}`：

| 组件 | 用途 | 路径 |
|------|------|------|
| `PageContainer` | 页面容器（标题/加载态/空态/错误态） | `@/components/ui/PageContainer` |
| `SectionCard` | 带标题的区域卡片（替代 div+header+body） | `@/components/ui/SectionCard` |
| `StatCard` | 统计数字卡片（替代 div+Statistic） | `@/components/ui/StatCard` |
| `PageSkeleton` | 骨架屏（StatRowSkeleton/CardSkeleton/TableSkeleton） | `@/components/ui/PageSkeleton` |
| `ErrorBoundary` | 错误边界（捕获渲染异常+重试按钮） | `@/components/ui/ErrorBoundary` |

颜色用 `var(--*)` CSS 变量，间距用 `var(--space-*)` scale（8px 基准）。
设计系统完整预览在 `/design-system` 路由。

<!-- END:frontend-components -->
