# TODOs

## Metabolism M1 — Phase 1 Migration

- **What:** Migration for `metabolism_log` table + `event_outbox` indexed columns
- **Why:** M1 scores records and needs a table to store score results. `event_outbox` needs `excreted_at` (tagged for deletion) and `excretion_reason` (why it was scored/excreted) columns for scheduled cleanup in Phase 2.
- **Context:** Added during /plan-eng-review on 2026-06-26. Phase 1 is dry-run (no actual deletion), but the schema should ship from day 1 so Phase 2 doesn't need a second migration.
- **Action:** `backend-go/migrations/XXX_add_metabolism.sql` — CREATE TABLE metabolism_log + ALTER TABLE event_outbox ADD COLUMNS.
- **Depends on:** Design approval of MetabolismModel fields (see design doc).
- **Blocked by:** Nothing.

## UI Redesign — 未完成项目

### P5: 跨页一致性检查

- **What:** 检查 P2-P4 三个页面（prelisting、dashboard、agentos）之间是否存在样式 drift — 边距不一致、字体不一致、颜色偏差、共享组件使用方式不一致。
- **Why:** P1 创建了 PageContainer / AgentDecisionPanel 等共享组件，P2-P4 可能各自改写了样式造成漂移。
- **Action:** 逐页面对比 padding、font-size、color token、AgentDecisionPanel 使用方式。输出差异清单并修复。
- **Blocked by:** P1-P4 完成（已交付）。

### Mock 数据 → 真实 API 迁移

- **What:** 三个新页面目前使用硬编码 mock 数据。替换为从后端 API 获取真实数据。
  - `/decision/prelisting`：用 `/api/v1/decision` 系列接口
  - `/dashboard`：用 `/api/v1/dashboard/overview`
  - `/agentos`：用 `/api/v1/agentos`、`/api/v1/agentos/work-items`
- **Why:** Mock 数据不能用于生产。
- **Action:** 替换每个页面的 `mockSkus`/`mockPriority`/`mockWorkItems` 为 React Query 的 `useQuery` 调用后端 API。保留 mock 作为 fallback。
- **Depends on:** 后端对应 API endpoint 已就绪。
- **Blocked by:** `backend-go` 对应 handler 的验证。

### 旧 40+ 页面在新 layout 中的视觉验证

- **What:** P1 将 layout 从三栏改为四栏，20+ 原有 Ant Design CRUD 页面自动被新 shell 包裹。需要逐个验证它们在新 shell 中的渲染效果——宽度、滚动、边框、嵌套。
- **Why:** 旧页面可能在窄的中心区域显示异常（表格被截断、按钮重叠等）。
- **Action:** 在本地 dev 环境中逐一访问旧页面，截图检查布局。修复发现的异常。
- **Blocked by:** P1 完成（已交付）。

### 组件测试覆盖

- **What:** P1 新增的共享组件（DomainSidebar、AgentDecisionPanel、DecisionCard、RiskBadge、PageContainer）缺少测试，需要至少覆盖 loading/empty/error/normal 四种状态。
- **Why:** 按 review 决策要求全量测试覆盖。
- **Action:** 为每个新增组件创建 `*.test.tsx` 文件，使用 Vitest + React Testing Library 渲染四种状态并断言。
- **Blocked by:** P1 完成（已交付）。

### Ant Design 6 暗色 Token 收敛

- **What:** P1 在 `AntdProvider.tsx` 中设置了基础 token 覆盖（colorBgBase、colorBgContainer），但部分 Ant Design 组件的暗色适配可能不完整（如 Table 表头、Modal 遮罩、Dropdown 菜单）。
- **Why:** 如果 AntD 默认暗色 token 和新 DESIGN.md 不一致，会出现肉眼可见的对比度差异。
- **Action:** 在浏览器中逐组件检查 Table、Modal、Dropdown、Select、DatePicker 的暗色渲染，补充 token 覆盖。
- **Blocked by:** P1 完成（已交付）。
