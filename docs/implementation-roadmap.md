# LingMirror 实现路线图

> 更新时间：2026-07-06
> 范围：19 个 Open Issues + CURRENT_DIRECTION P0-P1 门禁
> 参考：docs/CURRENT_DIRECTION_AND_PRIORITIES.md

## 总则

这份路线图依据 **PLATFORM CONSTITUTION** 的"Owner-first / Platform-first / Audit-first / 小步可逆"原则排期。
总共 **6 轮迭代**（每轮对应一个 `docs/specs/iteration-N-*.md` 详细 spec）。

### 分层说明

每轮迭代标注其触达的系统层（参考 Constitution §2）：

| 层 | 代号 | 说明 |
|----|------|------|
| 1 | Kernel | Auth, RBAC, EventBus, Scheduler, ToolBridge, Audit, Config |
| 2 | Domain | 业务域模块 |
| 3 | Agent | Agent 工作流 |
| 4 | Integration | 外部平台集成 |
| 5 | UI | 前端体验 |
| 6 | Docs | 文档 / 治理 |

## 迭代总览

```
Iter 1 ─ 安全速赢 + 数据地基 ── Layer 1/2/6
  ├ #280 JWT_SECRET 写死在 docker-compose.yml → env var 配置化
  ├ #281 DB 无自动备份 → pg_dump + cron + 云存储
  └ #130 利润计算硬编码 CNY→USD → config.yaml + test

Iter 2 ─ 生产可观测 + 成本控制 ── Layer 1/6
  ├ #282 线上无告警通知 → Slack/钉钉/邮件推送
  ├ #283 LLM 预算控制 → token 配额 + 限流
  └ P0 EventBus/Scheduler 生命周期验证

Iter 3 ─ 执行门禁合规 ── Layer 1/2/5/6
  ├ 统一 Action Execution Gate (CURRENT_DIRECTION P0)
  ├ Approval Identity + RBAC 绑定 (CURRENT_DIRECTION P0)
  ├ 审计日志敏感字段脱敏 (CURRENT_DIRECTION P1)
  └ #143 ToolBridge 三类工具扩展 (KERNEL_CONTRACTS §5)

Iter 4 ─ 发布流程 + 风险交互 ── Layer 4/5/6
  ├ #285 无法回退历史版本 → 版本回滚机制
  ├ #284 无法验证 AI 交付物 → 验证框架
  ├ 外部平台写安全 (CURRENT_DIRECTION P1)
  └ 前端高风险动作确认 UX (CURRENT_DIRECTION P1)

Iter 5 ─ Workflow 平台 ── Layer 2/3/5
  ├ #189 Workflow 前端管理页面
  ├ #192 Workflow 引擎增强 (condition/fork/execEvent)
  └ #202 Workflow 监控面板

Iter 6~7 ─ P3 功能群 ── Layer 2/3/4/5
  ├ #197 供应商评分管理
  ├ #199 竞品价格监控
  ├ #200 多平台接入扩展
  ├ #201 库存健康度优化
  ├ #193 Prism 商品图生成引擎
  ├ #194 Ad Pilot Agent
  ├ #195 Listing Genius Agent
  └ #196 Support Mate Agent
```

## 依赖关系图

```
Iter 1 (安全/数据)
   │
   ▼
Iter 2 (可观测/成本) ──── 不阻塞，但建议先做（提高诊断效率）
   │
   ▼
Iter 3 (执行门禁) ──── #143 ToolBridge 依赖 Approval 基础设施先完成
   │
   ▼
Iter 4 (发布流程) ──── 依赖 Iter 3 门禁做安全回退
   │
   ├──── Iter 5 (Workflow)
   │
   └──── Iter 6~7 (P3 功能) ── 依赖 Iter 3 门禁保护外部写动作
```

## 迭代产出规范

每轮迭代产出：

1. **Spec** — `docs/specs/iteration-N-*.md`，经 Owner 确认后开工
2. **实现** — 按 spec-driven development 四阶段完成
3. **验证** — `go test ./...` / `go vet ./...` + 功能端到端验证
4. **PR** — 每个 issue 或功能组一个 PR
5. **更新** — 更新 PROJECT_STATUS.md + 本文档状态

## 当前状态

| 迭代 | 范围 | Spec | 实现 | 验证 | 状态 |
|------|------|------|------|------|------|
| 1 | #280, #281, #130 | ✅ 已创建 | ✅ 已实现 | ✅ 已验证 | **完成** |
| 2 | #282, #283, EventBus | 📝 待创建 | ⏳ | ⏳ | 排队 |
| 3 | 执行门禁 + #143 | 📝 待创建 | ⏳ | ⏳ | 排队 |
| 4 | #284, #285 + 外部写/UX | 📝 待创建 | ⏳ | ⏳ | 排队 |
| 5 | #189, #192, #202 | 📝 待创建 | ⏳ | ⏳ | 排队 |
| 6~7 | #193-#201 | 📝 待创建 | ⏳ | ⏳ | 排队 |
