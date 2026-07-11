# 功能需求：商品出海决策与执行层

> **状态：`superseded` / 已冻结（2026-07-11）**
> 本文保留为历史实现规格，不再是 P0、半年总纲或当前开发准绳。固定平台、固定候选数量和面向外部运营人员的设计均已冻结。当前唯一方向见 [Owner 自用经营方向](../SELF_USE_OPERATING_DIRECTION.md)。
>
> 添加时间：2026-07-07
> 提出人：Owner
> 优先级：P0
> 原状态：当前业务层开发总纲（已被替代）

## 一句话说明

建设一条可追溯、可审批、可执行、可复盘的商品出海闭环，让系统能够回答：

```text
这个候选商品能不能卖？
为什么建议卖或不卖？
风险、成本、利润和缺口是什么？
Owner 批准后系统会执行什么？
执行结果如何回流到商品档案和下一轮判断？
```

本文曾作为未来半年“商品出海决策与执行层”的开发总纲。自 2026-07-11 起，后续 Agent 只能把它当作历史背景和已有代码说明，不得从中恢复平台优先级、固定漏斗或建设范围。

## 总目标

把现有 `candidate`、`producthub`、`loop`、`owner`、`approval`、`listingtask`、`listing`、`integrations`、`operationlog` 等模块统一成一条主业务链路：

```text
Candidate Product
-> completeness check
-> cost / logistics / platform fee / profit evaluation
-> listing recommendation
-> Owner approval
-> controlled listing task
-> dry-run / sandbox / production-aware platform execution
-> execution result
-> ProductHub evidence trace and recommendation feedback
```

最终产品不是“更多 CRUD 页面”，也不是“一个 Agent 聊天入口”，而是 LingMirror 的商品出海业务控制层：

- AI 负责收集、计算、建议、解释和准备执行。
- Owner 负责高风险决策、审批和策略边界。
- 系统负责门禁、状态机、审计、幂等、执行模式和结果回流。

## 当前情况

项目已经具备多段骨架，但还没有统一成稳定业务层。

### 已有基础

- `candidate_product` 已承载候选商品、目标售价、目标平台、目的国、来源和完整度状态。
- `loop.Service.Evaluate` 已能执行候选商品评估，生成 `listing_recommendation`，并在推荐上架时创建 `listing_task` 和审批请求。
- `owner` 模块已有建议队列、风险摘要和反馈能力。
- `approval` 模块已有审批记录、审批状态和事件发布能力。
- `listingtask` 模块已有状态机、审批门禁、执行入口、Prism 检查、审计和平台发布 hook。
- `integrations` 模块已有平台 adapter 接口和 dry-run / sandbox / production 的执行模式概念。
- 前端 `/owner`、`/candidates`、`/approval`、`/listing-tasks/:id` 已经接近一条业务闭环。

### 主要断点

- 存在两套上架路径：`loop -> listing_recommendation -> listing_task` 与 `pre_listing_decision -> listing/listing-tasks`。
- 审批通过后，`listing_task.approval_id` 和 `listing_task.status=approved` 的推进不稳定。
- 审批事件 topic 和订阅不一致，`publish` 与 `listing_task` 语义混用。
- 自动执行路径和手动执行路径没有共享同一个 `publishHook` 和执行模式语义。
- `publishHook` 需要统一读取 dry-run / sandbox / production，不得绕过平台写入门禁。
- `ProductHub` 还不是完整证据链，未稳定展示候选、评估、建议、审批、任务、执行、结果回流。
- `/decision/prelisting` 仍偏模拟页面，和 Owner 主审批链路割裂。
- `/listings` 仍偏 listing record CRUD，尚未成为发布结果和平台状态复盘页。

## 半年目标

未来半年目标不是“把所有跨境电商能力做完”，而是把商品出海这条核心经营链路做成可用、可信、可扩展。

### 1 个月目标

跑通一条可信 dry-run 闭环：

```text
候选商品 -> 完整度/利润评估 -> 上架建议 -> Owner 审批 -> ListingTask 执行 -> dry-run 结果 -> 证据回流
```

Owner 能在 `/owner` 看懂建议，能批准或拒绝，系统不会发生未审批的真实平台写操作。

### 3 个月目标

跑通一个平台的 sandbox 级商品出海链路：

```text
候选商品 -> 平台适配字段 -> 类目/属性/价格/库存/包装校验
-> Owner 审批 -> sandbox publish -> 平台返回结果 -> ProductHub 复盘
```

至少一个平台可被证明具备 sandbox 发布能力，失败可解释、可重试、可审计。

### 6 个月目标

形成半自动商品出海操作系统：

```text
AI 批量发现候选商品
-> 自动评估利润和风险
-> 自动生成上架建议和待审批任务
-> Owner 批准高风险动作
-> 系统按 execution mode 执行
-> 订单、销售、利润、失败原因回流
-> 下一轮推荐质量持续提升
```

此时系统应支持多平台扩展，但生产写入仍必须受 Owner 策略、审批、RBAC、审计和执行模式约束。

## 阶段规划

### Phase 0：统一业务口径与开发边界

时间：第 1 周。

目标：

- 明确“商品出海决策与执行层”是当前第一业务层。
- 将旧栈计划、历史审计、当前事实、未来路线分层，避免 Agent 被过期文档误导。
- 定义唯一主路径和辅助页面职责。

后端交付：

- 不新增业务能力。
- 梳理并确认主链路涉及模块：`candidate`、`loop`、`owner`、`approval`、`listingtask`、`integrations`、`producthub`、`operationlog`。
- 标出需要收敛或废弃的旧路径：`decision/prelisting` 与 `listing/listing-tasks/from-decisions` 不得继续作为主出海链路扩展。

前端交付：

- 定义主入口：`/owner`。
- 定义辅助入口：
  - `/candidates`：候选池和资料补齐。
  - `/approval`：审批队列和审批历史。
  - `/listing-tasks`：执行监控和失败处理。
  - `/product-hub`：商品 360 档案和证据链。
  - `/listings`：平台发布结果和 listing record。

文档交付：

- 本文件作为业务层开发总纲。
- `docs/INDEX.md` 增加本文件入口。
- 不把业务规划写回 `AGENTS.md` 或 `CLAUDE.md`。

验收标准：

- 后续 Agent 能通过本文件知道当前业务层目标、阶段和边界。
- 不再把旧 FastAPI/Vue 计划当作当前实现路径。

### Phase 1：打通单商品 dry-run 闭环

时间：第 2-4 周。

目标：

打通一条单商品、非真实外写的可信闭环：

```text
candidate -> loop evaluate -> listing_recommendation
-> owner decision -> approval -> listing_task approved
-> ExecuteTask dry-run -> execution result -> recommendation feedback
```

后端交付：

- 统一 `loop` 路径为商品出海主路径。
- 修复审批到任务的状态推进：
  - 审批通过后必须回写 `listing_task.approval_id`。
  - 审批通过后必须把任务推进到 `approved`。
  - `ExecuteTask` 必须能基于 approved 审批记录正确执行。
- 统一 approval event topic：
  - 不再混用 `publish` 和 `listing_task` 作为同一动作语义。
  - 事件订阅和审批请求类型必须一致。
- 确保审批路由、权限、审计可用。
- 确保 dry-run 不发生真实平台写操作。
- 失败必须写入 `last_error`、operation log 和可展示的业务错误。

前端交付：

- `/owner` 能展示建议、理由、风险、预期动作、执行模式和审计去向。
- Owner 在 `/owner` 可以批准或拒绝建议。
- 批准后能跳转到对应 `listing_task`。
- `/listing-tasks/:id` 能展示审批状态、执行状态、dry-run 结果和错误原因。

数据与追溯：

- `listing_recommendation` 记录建议、理由、风险、置信度、关联任务。
- `listing_task.decision_snapshot` 保存评估快照。
- `operation_log` 记录关键状态变化。

验收标准：

- 用一个 demo candidate 可以完整跑通 dry-run。
- 未审批任务不能执行。
- 审批记录、任务状态、执行结果、审计日志可互相追溯。
- Owner 不需要读日志或代码就能知道批准后发生了什么。

不得做：

- 不接 production 平台发布。
- 不扩展更多 CRUD 页面。
- 不新增第二套决策路径。

### Phase 2：ProductHub 证据链与主路径 UI 收敛

时间：第 2 个月。

目标：

把商品从候选到执行结果的过程沉淀为 ProductHub 可查看的生命周期证据链。

后端交付：

- 为 ProductHub 聚合以下信息：
  - candidate 基础信息。
  - completeness 状态和缺失字段。
  - profit / logistics / platform fee 评估摘要。
  - listing recommendation。
  - approval request。
  - listing task。
  - execution result。
  - listing record / platform reference。
- 建立稳定的 evidence trace 查询接口，避免前端拼多个不一致接口。

前端交付：

- `/product-hub/[id]` 或等价详情页展示商品生命周期：
  - 候选来源。
  - 当前资料缺口。
  - 评估结果。
  - AI 建议。
  - Owner 决策。
  - 执行任务。
  - 平台结果。
  - 复盘状态。
- `/decision/prelisting` 降级为历史/调试入口，不能作为主工作台。
- `/listings/create` 不作为主上架入口；上架应从审批通过的 listing task 来。

验收标准：

- Owner 从 ProductHub 能看到“这个商品为什么走到当前状态”。
- 任一 listing task 都能追溯到 candidate、recommendation、approval 和 execution result。
- 前端主路径明确，不再依赖孤立 CRUD 页面完成业务闭环。

### Phase 3：单平台 sandbox 发布能力

时间：第 3 个月。

目标：

选择一个平台作为第一条真实适配链路，优先 Ozon 或 Shopee，完成 sandbox 发布闭环。

后端交付：

- 统一 platform account 的 execution mode：
  - dry-run：只模拟，不外写。
  - sandbox：写入平台沙箱或测试环境。
  - production：真实外写，必须审批。
- `publishHook` 必须读取 execution mode，不能直接绕过 integrations service。
- 发布请求必须包含：
  - product metadata。
  - SKU。
  - price。
  - inventory。
  - package dimensions。
  - category / attributes。
  - target platform account。
- 平台返回的 external reference id 必须保存。
- 网络失败、平台校验失败、字段缺失必须可解释、可重试、可审计。
- 建立幂等策略，避免重复发布。

前端交付：

- `/listing-tasks/:id` 展示平台校验结果、sandbox/production mode、外部 reference id、失败原因和重试动作。
- 高风险确认组件必须用于 production publish。

验收标准：

- 一个商品能完成 sandbox publish。
- 缺字段时阻断在执行前，并说明缺什么。
- 平台返回失败时，任务进入 failed，保留错误和重试入口。
- production mode 没有审批不能执行。

不得做：

- 不同时铺多个平台。
- 不允许为了演示绕过 approval / RBAC / audit。

### Phase 4：批量候选商品与 Owner 决策队列产品化

时间：第 4 个月。

目标：

从单商品闭环扩展到批量候选商品，让 Owner 每天处理“最值得决策的商品”。

后端交付：

- 支持批量 candidate evaluation。
- 建立候选商品排序逻辑：
  - 资料完整度。
  - 预估利润。
  - 市场机会。
  - 风险等级。
  - 缺失字段数量。
  - 上架准备度。
- Owner suggestions 需要支持状态过滤：
  - waiting_data。
  - ready_for_decision。
  - pending_approval。
  - executing。
  - completed。
  - failed。

前端交付：

- `/owner` 首屏回答：
  - 今天要决定什么？
  - 哪些商品最值得上架？
  - 哪些商品缺资料？
  - 哪些任务卡在审批或执行？
  - 哪些失败需要处理？
- `/candidates` 支持按完整度、利润、平台、目的国、推荐状态筛选。

验收标准：

- Owner 可以从 20 个候选商品中快速找到需要处理的 3-5 个。
- 每条建议都有 what / why / risk / expected outcome / execution mode / audit trail。
- 批量评估不产生未审批执行。

### Phase 5：结果回流与推荐质量复盘

时间：第 5 个月。

目标：

让执行结果和经营结果反向影响下一轮商品判断。

后端交付：

- `RecordExecutionResult` 覆盖成功、失败、平台返回、业务阻断。
- listing record 与 listing task、candidate、ProductHub 关联稳定。
- 引入基础复盘指标：
  - 是否成功发布。
  - 发布耗时。
  - 失败原因类型。
  - 首批曝光 / 订单 / 利润数据，如数据已接入。
  - AI 建议是否被 Owner 采纳。
  - 采纳后结果是否符合预期。
- 输出 recommendation feedback summary。

前端交付：

- ProductHub 展示复盘卡片。
- Owner 工作台展示建议命中率、失败类别和待改进数据。

验收标准：

- 已完成任务不只是 completed，而是能说明结果是否达成预期。
- AI 建议被采纳/拒绝/失败的原因可被后续分析。

### Phase 6：多平台扩展与半自动运营

时间：第 6 个月。

目标：

在第一平台闭环稳定后，扩展到第二平台，并形成半自动出海运营能力。

后端交付：

- 抽象平台差异：
  - category mapping。
  - attribute mapping。
  - price rule。
  - inventory policy。
  - publish validation。
  - external status sync。
- 支持平台间对比建议：
  - 哪个平台更适合该商品。
  - 利润差异。
  - 物流差异。
  - 风险差异。
- 生产发布必须继续受 approval / RBAC / audit / execution mode 控制。

前端交付：

- Owner 可以看到商品适合哪个平台以及原因。
- 同一商品可以展示多个平台的发布状态和复盘结果。

验收标准：

- 第二平台接入不需要复制一套业务链路。
- 新平台只扩展 adapter 和 mapping，不破坏主业务层。
- 生产动作仍然可控、可审计、可回滚或可补偿。

## 目标架构

### 业务层分层

```text
Owner Experience
  /owner
  /candidates
  /approval
  /listing-tasks
  /product-hub
  /listings

Product Export Decision & Execution Layer
  Candidate intake
  Readiness / completeness
  Profit / logistics / platform fee evaluation
  Listing recommendation
  Owner decision
  Approval binding
  Listing task orchestration
  Execution mode control
  Evidence trace
  Feedback loop

Domain Modules
  candidate
  producthub
  loop
  owner
  approval
  listingtask
  listing
  integrations
  operationlog

Platform Kernel
  JWT / RBAC
  Approval
  Audit
  EventBus
  Scheduler
  Action policy
  Observability

External Platforms
  Ozon
  Shopee
  Shopify
  Lazada
  Amazon
```

### 主路径约束

- `loop` 是商品出海评估主路径。
- `listingtask.ExecuteTask` 是受控执行主入口。
- `approval` 是高风险动作进入执行的必经门禁。
- `ProductHub` 是商品证据链主视图。
- `operationlog` 是审计事实源之一，但 Owner 不应必须读日志才能决策。
- `integrations` 是平台写入边界，所有外部写必须明确 execution mode。

### 状态流转

建议统一为：

```text
candidate: incomplete | needs_review | research_ready | listing_ready

recommendation:
  draft | ready | adopted | rejected | executed | failed

approval:
  pending | approved | rejected

listing_task:
  blocked | pending_approval | approved | executing | completed | failed

execution_mode:
  dry_run | sandbox | production
```

## 关键开发原则

- 先闭环，再扩功能。
- 先 dry-run / sandbox，再 production。
- 先单平台稳定，再多平台扩展。
- 先业务主路径，再辅助 CRUD。
- 先证据链可追溯，再做自动化规模化。
- 高风险动作默认不允许自动执行。
- Owner 决策必须用业务语言解释，不要求 Owner 理解模块内部实现。

## 验收总标准

半年内，以下场景必须可以稳定演示：

```text
1. 创建或导入一个候选商品。
2. 系统检查资料完整度，指出缺失字段。
3. 系统计算成本、物流、平台费和利润。
4. 系统生成是否建议上架的结论，并解释原因和风险。
5. Owner 在 /owner 批准或拒绝。
6. 批准后生成或推进受控 listing task。
7. 任务在 dry-run / sandbox / production-aware 模式执行。
8. 未审批的真实平台写入被阻断。
9. 执行结果、失败原因、平台 reference 和审计记录回流。
10. ProductHub 能展示完整证据链。
```

## Claude 开发指令摘要

后续派 Claude 开发时，可直接使用以下指令：

```text
请以 docs/features/product-export-decision-execution-layer.md 为业务总纲，
建设 LingMirror 的商品出海决策与执行层。

当前首要目标不是新增页面或新增 Agent，而是收敛现有 candidate、loop、owner、
approval、listingtask、integrations、producthub、operationlog 模块，打通一条
可审批、可执行、可追溯的商品出海主链路。

优先完成 Phase 1：
candidate -> loop evaluate -> listing_recommendation -> owner decision
-> approval -> listing_task approved -> ExecuteTask dry-run -> execution feedback。

所有高风险动作必须经过 Owner approval、RBAC、audit、状态机和 execution mode。
不得绕过 dry-run/sandbox/production 语义，不得创建第二套上架路径。
```

## 涉及模块

后端：

- `backend-go/internal/domain/candidate`
- `backend-go/internal/domain/loop`
- `backend-go/internal/domain/owner`
- `backend-go/internal/domain/approval`
- `backend-go/internal/domain/listingtask`
- `backend-go/internal/domain/listing`
- `backend-go/internal/domain/producthub`
- `backend-go/internal/domain/integrations`
- `backend-go/internal/domain/operationlog`
- `backend-go/internal/httpx/router.go`

前端：

- `frontend-next/src/app/(main)/owner`
- `frontend-next/src/app/(main)/candidates`
- `frontend-next/src/app/(main)/approval`
- `frontend-next/src/app/(main)/listing-tasks`
- `frontend-next/src/app/(main)/product-hub`
- `frontend-next/src/app/(main)/listings`
- `frontend-next/src/config/menu.ts`

文档：

- `docs/INDEX.md`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `docs/PROJECT_STATUS.md`
- `docs/reference-module-catalog.md`
- `docs/ACCEPTANCE_GATE.md`

## 估算

- Phase 0：1 周，文档和架构口径收敛。
- Phase 1：2-3 周，单商品 dry-run 闭环。
- Phase 2：4 周，ProductHub 证据链和前端主路径收敛。
- Phase 3：4 周，单平台 sandbox 发布。
- Phase 4：4 周，批量候选和 Owner 决策队列产品化。
- Phase 5：4 周，结果回流和推荐复盘。
- Phase 6：4 周，多平台扩展和半自动运营。

数据库变更：是。预计需要补充或调整任务关联、执行结果、平台 reference、证据链聚合字段或视图。

## 依赖

- 当前 Go + Next 活跃栈。
- Approval / RBAC / Audit / EventBus / execution mode 必须保持可用。
- 至少一个平台 adapter 可用于 sandbox 或 dry-run。
- Demo data 需要覆盖候选商品、平台、SKU、价格、库存、包装尺寸和平台账号。

## 不做范围

- 不做完全自动生产发布。
- 不做无代码 Agent 搭建平台。
- 不以扩展 CRUD 页面作为阶段目标。
- 不同时推进多个平台 production 接入。
- 不把旧 FastAPI/Vue 计划作为当前实现路径。
- 不把业务规划写入 `AGENTS.md` 或 `CLAUDE.md`。
