# Spec: 凌镜 H1 路线图 — 从可信 Copilot 到可运营 AgentOS

## Objective

把凌镜从"可信 Copilot 原型"（v0.4.1.0）推进到"小团队可实际运营的跨境电商 AgentOS"。

**核心不是全自动公司**，而是让 Owner 能用它稳定完成选品、上架、订单、利润、异常处理和 Agent 审批闭环。

### 北极星（验收标准）

到工作完成时，应该能做到：

| # | 能力 | 验收条件 |
|---|------|----------|
| 1 | 每日运营 | Owner 每天打开系统能看到：今天该处理什么、哪些商品有机会、哪些订单有风险 |
| 2 | Agent 建议 | Agent 能主动提出建议，但高风险动作必须审批 |
| 3 | 商品链路 | 商品从候选到上架有完整链路 |
| 4 | 订单链路 | 订单从同步到利润核算再到异常处理有完整链路 |
| 5 | 写回可控 | 平台写回可控、可审计、可追踪失败 |
| 6 | 运维基线 | 系统有告警、备份、成本控制和测试基线 |
| 7 | 一致可信 | 文档、状态、路线图和实际代码保持一致 |

### 不做的

- 完全无人值守自动运营
- 大量新 Agent 名称
- 通用 no-code Agent 平台
- 没有审批和审计的真实外部写回

## Tech Stack

维持现有栈，不引入新语言/框架：

| 层 | 技术 | 说明 |
|----|------|------|
| Backend | Go 1.25, Gin, GORM | `backend-go/` |
| Frontend | Next.js 16, React 19, TypeScript, Ant Design 6 | `frontend-next/` |
| Database | PostgreSQL 15 | Docker compose |
| AI | 现有 internal/ai + agent 模块 | LLM 编排层 |
| 集成 | Ozon / Shopee / Shopify 适配器 + 按需接入 | 按需求接入，统一 Adapter 接口 |

## Commands

见 `CLAUDE.md` 完整命令列表。关键：

```bash
# 后端
cd backend-go && go test ./...   # 单元测试
cd backend-go && go vet ./...    # 静态分析

# 前端
cd frontend-next && npm test     # 前端测试
cd frontend-next && npm run build # 构建

# E2E & 烟雾测试
cd frontend-next/e2e && npx playwright test
cd backend-go && ./scripts/smoke_test.sh
```

## Project Structure

不变。新增功能走现有模块 pattern：

```
internal/domain/{module}/
  ├── routes.go    # Gin 路由
  ├── handler.go   # HTTP 请求/响应映射
  ├── service.go   # 业务逻辑
  └── model.go     # GORM 模型 + DTO

frontend-next/src/app/(main)/{module}/page.tsx
frontend-next/src/config/menu.ts  # 新页面加菜单项
```

新域模块按需要创建，不做超前抽象。

## 工作分区与依赖图

```
M1 ──→ M2 ──→ M5
  │      │
  ├──→ M3 ──→ M5
  │
  ├──→ M4 ──→ M5
  │
  └──→ M6
```

**M1（可信基础）** — 唯一严格前置，必须先收口才能放真实数据
**M2 + M3 + M4** — 完全并行，三路独立维度
**M5**（Workflow 平台）— 逻辑上依赖至少一个闭环跑通，M5 前至少一个 M2~M4 任务完成
**M6**（运营化）— 顺次收尾，不阻塞其他

### 各工作流细节

---

### M1: 可信基础收口

**前提：** 无（最高优先级）
**依赖于：** 无
**并行度：** 内部子任务可并行

| # | 任务 | 交付物 | 验收 |
|---|------|--------|------|
| M1.1 | 修复现有测试/lint/build 尾巴 | `supplier.TestHandler_GetSupplierComparison` 验证通过；`go test ./...` 全绿；`npm test` 配置能跑 | 后端全绿，前端测试不因测试框架设置阻塞 |
| M1.2 | 统一项目状态文档口径 | `PROJECT_STATUS.md` 与代码实际一致 | 文档无死引用，API 存量/模块清单准确 |
| M1.3 | 生产可观测性 | 告警 (health endpoint × 监控)、Agent 运行状态 API、失败原因追踪、LLM 成本聚合 | Owner 能看到 Agent 是否在跑、失败在哪、花了多少 token |
| M1.4 | 高风险确认UI全接入 | `HighRiskConfirmDialog` 接入 Owner 工作台 + AI action 页面 approve/execute 操作 | 价格/库存/发布/订单状态变更 都走确认弹窗 |
| M1.5 | 审批+审计全覆盖 | 所有价格/库存/发布/订单状态变更 走审批和审计 | 改动有记录、有身份绑定、可回查 |

**交付：** Owner 可以相信系统不会偷偷改价格、库存、订单或平台商品。

---

### M2: 商品经营闭环

**前提：** M1.4 + M1.5 完成
**依赖于：** 无（独立于 M3/M4）
**并行度：** 与 M3、M4 完全并行

| # | 任务 | 交付物 |
|---|------|--------|
| M2.1 | 候选商品完整度检查 | 完整度引擎（已有原型），覆盖成本/物流费/平台费/毛利测算 |
| M2.2 | Listing 建议 | 标题、类目、价格、库存、平台适配建议 |
| M2.3 | Owner 审批 → 受控上架任务 | 审批后生成受控任务，走 M1 的审批+审计链路 |
| M2.4 | 上架结果复盘 | 是否发布成功、异常捕获、预期利润 vs 实际 |

**交付：** Owner 可以用凌镜判断一个商品是否值得卖，并把建议转成可审计的上架任务。

---

### M3: 订单与履约利润闭环

**前提：** M1.5 完成
**依赖于：** 无（独立于 M2/M4）
**并行度：** 与 M2、M4 完全并行

| # | 任务 | 交付物 |
|---|------|--------|
| M3.1 | 订单导入/同步稳定化 | 平台订单持续同步不丢不重复 |
| M3.2 | 成本核算链路 | 库存匹配、物流选择、运费快照、平台结算、毛利核算 |
| M3.3 | 异常识别 | 亏损单标记、缺货预警、物流异常、费用异常 |
| M3.4 | Agent 处理建议 + Owner 审批 | Agent 给出建议卡片，Owner 审批或转人工 |

**交付：** Owner 能看到每个订单为什么赚钱或亏钱，异常订单能被系统主动指出。

---

### M4: 平台集成受控生产

**前提：** M1.4 + M1.5 完成
**依赖于：** 无（独立于 M2/M3）
**并行度：** 与 M2、M3 完全并行

| # | 任务 | 交付物 |
|---|------|--------|
| M4.1 | 平台适配器核心流程补齐 | 发布、库存同步、订单同步、物流单号回传（按需接入，统一 Adapter 接口） |
| M4.2 | 环境控制 | 分环境 dry-run/sandbox/production（已有 dry-run 模式） |
| M4.3 | 生产写回门禁 | 审批 + 外部 reference ID + 失败可见 + 重试策略 |
| M4.4 | 真实店铺生产试点 | 生产级运行，不设订单量上限 |

**交付：** 凌镜能安全连接真实平台，但仍保持 Owner 控制权。

---

### M5: AgentOS 工作流平台

**前提：** 至少一个闭环 (M2/M3 其一) 已跑通
**依赖于：** M2 或 M3 之一完成
**并行度：** 与 M6 可并行（但 M5 比 M6 价值更高，建议优先）

| # | 任务 | 交付物 |
|---|------|--------|
| M5.1 | Workflow 管理页面 | 可视化查看/管理工作流 |
| M5.2 | 条件分支 + 事件触发 + 审批节点 | 工作流引擎（基于现有 EventBus） |
| M5.3 | 工作流监控面板 | Agent 正在做什么、卡在哪里、需要谁批什么 |
| M5.4 | 固化标准工作流 | 商品闭环和订单闭环固化成标准 workflow |
| M5.5 | 任务队列 + 失败重试 + 运行历史 | Agent 任务不丢失、失败可重跑 |

**交付：** Owner 可以看到 Agent 正在做什么、卡在哪里、需要自己批准什么。

---

### M6: 运营化与 Beta 可用

**前提：** 无（并行收尾，不阻塞其他工作流）
**依赖于：** 无（可直接并行）
**并行度：** 与全部工作流可并行

| # | 任务 | 交付物 |
|---|------|--------|
| M6.1 | Owner 运营仪表盘 | 与现有 Seller Workbench 合并演进为统一仪表盘：销售、利润、异常、Agent 建议、审批结果 — 从 mock 到真实数据 |
| M6.2 | 日报/周报 | 自动生成运营简报（已有 MVP）|
| M6.3 | 运维基建 | 权限审查、审计完备、备份策略、告警规则、成本控制面板 |
| M6.4 | LLM 月度预算硬上限 | 每月 token 费用上限配置，超限停 Agent 并告警 Owner |
| M6.5 | Beta 验收 | 生产级连续运行、真实数据、真实平台、真实订单 — 2-3 个业务 demo 场景 |

**交付：** 凌镜可以作为一个跨境电商小团队的日常经营中枢使用。

## Code Style

沿用现有 pattern，不做新的抽象约定。关键规则：

- 新 domain 模块遵循 `routes.go / handler.go / service.go / model.go` 模式
- 错误处理：不静默吃掉 DB 错误（这是刚修复过的问题）
- 测试：`dbtest.NewDB(t, &Model{})` + SQLite in-memory
- 前端 API：`apiClient` + TanStack Query + 类型定义与后端 `Result[T]` 匹配
- 审批操作：必须写审计日志 + 绑定登录用户 + RBAC 检查

## Testing Strategy

| 层 | 框架 | 位置 | 目标 |
|----|------|------|------|
| Go 单元 | Go testing + testify | `_test.go` 同包 | 业务逻辑 >90% 路径覆盖 |
| Go 集成 | dbtest (SQLite) | `_test.go` 同包 | 数据访问层 + 边界条件 |
| 前端单元 | Vitest | `.test.tsx` 同目录 | 组件渲染 + 交互 |
| E2E | Playwright | `frontend-next/e2e/` | 关键用户路径 |
| 烟雾测试 | shell | `backend-go/scripts/smoke_test.sh` | 10 步端到端管道验证 |

新的审批/审计/平台写回路径必须有测试覆盖。

## Boundaries

### Always do（总是做）
- 新功能按现有模块 pattern 落地
- 审批操作之前先读 AGENTS.md 的 Project Medical Record
- 新 API 写在已有路由结构下
- 涉及价格/库存/发布的变更先检查审批门禁
- 跑通 `go test ./...` 再提交

### Ask first（先问）
- 引入新语言/框架/基础设施依赖
- 数据库 schema 重大变更（新表可以，大规模重构先问）
- 改变审批/审计/RBAC 的核心逻辑
- 触及现有 EventBus 生命周期的事件流变更
- 删除或重构已有 Agent

### Never do（不做）
- 把凭证/密钥提交到 git
- 没有审批的**真实**外部平台写回（sandbox 可以）
- 绕过 RBAC 做任何 mutation 操作
- 删掉未通过的测试而不修复
- 做"为了架构"的架构改动（不改变 observable behavior 的抽象）

## 已确认决策

| 问题 | 决策 |
|------|------|
| 平台优先级 | AI 时代，按需接入，统一 Adapter 接口，不限平台 |
| Dashboard 与 Workbench | 合并演进：Seller Workbench → 统一运营仪表盘 |
| Beta 验收规模 | 生产级，不设订单量上限 |
| LLM 预算控制 | 加 M6.4：月度硬上限 |

## 执行顺序建议

```
Phase 1 (strict): M1 — 可信基础收口
  └─→ 验证：go test ./... 全绿 + vet 通过 + 审批审计全覆盖

Phase 2 (parallel fan-out):
  ├─→ M2（商品闭环）
  ├─→ M3（订单利润闭环）
  ├─→ M4（平台受控写回）
  └─→ M6（运营化，可提前启动不阻塞）

Phase 3 (sequential):
  └─→ M5（AgentOS 工作流平台 — 至少等一个闭环跑通）
```

实际执行节奏：用 Agent 集群并行工蜂，完成一个工作流就切下一个，不是按"月"排期。
