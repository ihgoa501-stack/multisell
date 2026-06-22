# 功能需求：AgentOS Phase 2 操作闭环

> 添加时间：2026-06-22
> 提出人：项目推进
> 优先级：P0

## 一句话说明

在一天内完成 AgentOS 从任务建议到审批、执行、复盘的最小可用闭环，让任务中心可以真实驱动 ActionProposal。

## 详细描述

用户在 AgentOS 任务中心可以新建动作提案，并在同一任务卡上完成审批、拒绝、执行和复盘。执行结果写回 ActionProposal 生命周期，WorkItem 列表在每次操作后刷新，操作过程保留审计日志。

验收标准：

- 可以创建 `daily_report`、`listing_draft`、`profit_review`、`inventory_allocate`、`notify` 类型提案。
- 需要审批的提案必须先审批通过才能执行。
- 不需要审批的低风险提案可以直接执行。
- 已执行提案可以提交复盘结果。
- 任务列表能反映提案状态变化。
- 后端 AgentOS 聚焦测试通过，前端构建通过。

## 涉及模块

- 后端：`backend/app/agentos/router.py`
- 后端：`backend/app/agentos/action_center_service.py`
- 后端：`backend/app/agentos/service.py`
- 后端测试：`backend/tests/test_agentos_action_center.py`
- 后端测试：`backend/tests/test_agentos_phase1.py`
- 前端 API：`frontend/src/api/modules/agentos.ts`
- 前端页面：`frontend/src/views/agentos/WorkItems.vue`
- 前端组件：`frontend/src/components/agentos/WorkItemCard.vue`

## 估算

- 后端工作量：1.5 小时，主要用于回归测试和窄修复。
- 前端工作量：3 小时，补执行/复盘入口、刷新和错误提示。
- 数据库变更：否。

## 依赖

- PostgreSQL 测试库可用。
- `backend/.venv/` 可运行 pytest。
- `frontend/node_modules/` 已安装。
- AgentOS ActionProposal 相关表和迁移已存在。

## 备注

详细实施步骤见 `docs/superpowers/plans/2026-06-22-agentos-phase2-one-day.md`。
