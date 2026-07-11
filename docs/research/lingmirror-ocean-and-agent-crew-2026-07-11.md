# 凌镜的“海洋”与 Agent 舰队：基于 gstack ETHOS 的项目调研

> 调研日期：2026-07-11
> 范围：gstack `ETHOS.md`、凌镜当前治理文档、活跃 Go/Next 源码
> 结论标签：**事实**＝来源直接支持；**推断**＝基于事实的解释；**建议**＝下一步取舍，不代表已实现

## 一页结论

**推断：凌镜的“海洋”不是“拥有很多 Agent”，而是：一个 Owner 能在可信数据、审批和审计保护下，把跨境电商从选品一路经营到复盘的完整闭环。**

可以用一句可验收的话表达：

> 给定一个真实候选商品，凌镜能用真实成本、物流、平台费和市场数据说明“值不值得卖”；Owner 批准后形成受控上架任务；真实订单进入后，系统能追踪库存、履约、结算和净利润，并把结果反馈给下一轮决策，全程可解释、可审批、可审计、可恢复。

当前不是一片空海。项目已有 15 个具象业务 Agent（A1–A11、G0–G3），另有内容生成与生命周期调度的注册定义；Kernel 也已有 EventBus、Command、Scheduler、ToolBridge、审批、审计、RBAC 等航道。但是，**Agent 数量领先于闭环完成度**：缺的主要不是更多名字，而是商品资料完整性、平台费用、订单异常、结算真相、结果复盘这几类岗位的明确归属，以及真实数据、真实 LLM/工具、统一执行闸门和反馈学习的端到端证据。

因此，近期应先“煮第一片湖”：

```text
真实候选商品
-> 资料完整性
-> 成本 / 物流 / 平台费 / 利润
-> 上架建议
-> Owner 审批
-> 受控 Listing 任务
-> 结果回看
```

这与仓库已确定的第一业务循环完全一致，而不是另起一套战略。

---

## 1. ETHOS 里的 “ocean” 到底是什么

### 1.1 原文含义

**事实：** gstack 的 “Boil the Ocean” 反转了传统的“不要煮沸海洋”建议。它认为 AI 辅助开发显著降低了完整实现的边际成本；如果完整方案只比 90% 方案多一点工作，就应完成全部实现、测试、边界条件和错误路径。原文把 **ocean** 定义为目的地，例如“模块 100% 测试覆盖、完整功能、所有边界情况、完整错误路径”；把 **lake** 定义为一次能煮完的单元。湖是通往海洋的增量，不是降低终点标准的借口。[gstack ETHOS — Boil the Ocean](https://github.com/garrytan/gstack/blob/main/ETHOS.md#1-boil-the-ocean)

**事实：** 它同时明确排除“与当前任务真正无关”的工作，例如无关的多年平台迁移，应单独列为新范围。这意味着“煮海”不是无限扩张 scope。[同上](https://github.com/garrytan/gstack/blob/main/ETHOS.md#1-boil-the-ocean)

**事实：** ETHOS 的另外两条约束是：先搜索已有解法，再完整构建正确的东西；AI 只推荐，用户最终决定，不能因为多个模型同意就越过用户验证。[Search Before Building](https://github.com/garrytan/gstack/blob/main/ETHOS.md#2-search-before-building)、[User Sovereignty](https://github.com/garrytan/gstack/blob/main/ETHOS.md#3-user-sovereignty)

### 1.2 它没有说什么

**事实：** `ETHOS.md` 没有定义一套名为 “ocean agents / 海洋派 Agent” 的官方 Agent 原型或花名册。“海洋派 Agent”是把其完整性哲学映射到凌镜团队时形成的本项目术语，不应伪装成 gstack 原文概念。

**推断：** 因此，“你的海洋是什么”应先回答**完整业务结果**；“谁组成海洋派”再回答哪些岗位共同对这个结果负责。若反过来先堆 Agent，容易得到一支花名册，却没有完成一次可验证航行。

### 1.3 与凌镜治理的相容边界

**事实：** 凌镜治理要求 Owner-first、审计优先、可逆小步；高风险价格、库存、订单、资金、平台发布等动作必须受审批和审计约束（`docs/governance/PLATFORM_CONSTITUTION.md:9-15,84-100`）。当前产品定位也明确是“短期 Copilot，长期 Autopilot”，当前高风险动作即使 Agent 信任分高也不能无人监管执行（`docs/CURRENT_DIRECTION_AND_PRIORITIES.md:18-51`）。

**推断：** 凌镜版本的 “Boil the Ocean” 应解释为**结果完整、证据完整、安全路径完整**，而不是“立即开放所有自动写操作”。完整性包括失败、审批、审计与恢复；没有它们的“全自动”反而是不完整。

---

## 2. 凌镜的海洋：一个完整、可信、可控的经营闭环

### 2.1 战略终点

**事实：** 项目长期方向已经写明“让一个人也能经营跨境电商公司”：人负责目标、判断和关键审批；Agent 负责观察、分析、建议、低风险执行和复盘；系统负责可信数据、规则、权限、审计、平台对接与经营闭环（`docs/ONE_PERSON_AGENT_COMPANY_STRATEGY.md:5-21`）。

**事实：** 仓库定义的完整链路是“选品 → 商品资料 → 成本和利润测算 → 上架决策 → 发布任务 → 订单同步 → 库存变化 → 物流履约 → 结算利润 → 异常处理 → 复盘优化”（`docs/ONE_PERSON_AGENT_COMPANY_STRATEGY.md:58-81`）。

**推断：** 这条链路就是凌镜的业务“海洋”。更精确地说，它由四种完整性共同组成：

1. **事实完整性**：商品、SKU、供应商、成本、库存、价格、物流、平台费、订单、结算、利润和操作记录是真实、带来源、可追溯的。
2. **决策完整性**：每个建议都说明发生了什么、为什么、建议什么、风险和预期结果。
3. **执行完整性**：建议能进入 Owner 审批、受控任务、外部执行、失败处理和审计，不停在聊天文字。
4. **学习完整性**：把真实销售、退款、物流账单、结算净利润与 Owner 采纳/拒绝反馈回决策，知道哪些 Agent 判断有效。

### 2.2 当前先煮哪几片湖

**事实：** 当前方向文档指定两个优先业务循环：第一是候选商品到结果复盘；第二是订单到履约/结算异常的 Owner 决策（`docs/CURRENT_DIRECTION_AND_PRIORITIES.md:53-81`）。长期路线则按“能算清楚、能看清楚、能接进来、能建议”推进（`docs/ONE_PERSON_AGENT_COMPANY_STRATEGY.md:133-190` 及后续章节）。

**建议：** 将海洋拆成四片有验收门槛的湖，而不是按 Agent 数量拆：

| 湖 | 完成定义 | 主要业务价值 |
|---|---|---|
| L1 上架前真相湖 | 1 个真实商品能给出资料缺口、全成本、物流、平台费、净利和可解释建议 | 先回答“能不能卖” |
| L2 Owner 决策湖 | 建议进入统一待办；可采纳/拒绝/稍后；显示风险、影响、模式和审计 | Owner 不用追日志 |
| L3 履约真相湖 | 真实订单、库存、物流、结算可关联；异常变成可处理建议 | 回答“卖完是否赚钱” |
| L4 反馈学习湖 | 建议、动作、结果、人工反馈形成可比较 episode，更新规则/信任而不越权 | 判断 Agent 是否真的变好 |

L1/L2 是当前近期目标；L3/L4 是完成海洋不可缺的后续湖。

---

## 3. 现有“海洋派”Agent：谁已经在船上

### 3.1 业务航行组（已有具体实现）

**事实：** 当前具体实现注册表 `impl.All` 返回 A1–A11 与 G0–G3，共 15 个实现（`backend-go/internal/agent/impl/agents.go:17-36`, symbol `All`）。以下按经营链重排，而不是沿用代码顺序：

| 环节 | 现有 Agent | 当前职责 | 海洋中的角色判断 |
|---|---|---|---|
| 发现与进入 | A1 Product Scout、A8 Sourcing | 市场/候选品发现；1688 来源与采购盈利建议 | 航海侦察与入口 |
| 商品与 Listing | A2 Listing Optimizer、`content_ai`（定义） | 标题、关键词、多平台 SEO；内容/图片生成定义 | 商品资料加工，但“完整性验收”责任尚不清晰 |
| 获客 | A3 Ad Advice | ACOS 分析与广告优化建议 | 增长桨手；当前第一湖非关键路径 |
| 成本与利润 | A6 Profit Watch | 利润检查、成本优化、亏损预警 | 经营真相核心岗位 |
| 物流与仓储 | A10 Logistics Ops、G2 Warehouse Customs | 承运商比较、运费审计、物流绩效、仓配与报关 | 履约核心岗位 |
| 库存 | A5 Inventory Alert | 库存预警、补货计划、物流选择 | 履约核心岗位 |
| 客服与售后 | A4 Customer Service、A11 Aftersales Mgmt | 意图/回复；退货、退款、争议、售后报告 | 订单后的客户与损失处理 |
| 合规与风险 | A7 Compliance Guard、G3 Discount Risk | 商品/认证合规；折扣与促销风险 | 高风险守门员 |
| 批量执行 | A9 Batch Ops | 批量价格、库存、Listing 操作和导入校验 | 执行甲板；必须受审批/审计约束 |
| 总控与协调 | G0 Coordinator、G1 Dashboard | 系统健康、异常升级、跨 Agent 协调；全局聚合 | 船长助手与雷达 |

Agent 名称、Squad、决策点与风险下限的当前事实源是 `backend-go/internal/ai/registry.go:31-104`（symbol `DefaultRegistry`）；具体实现映射以 `backend-go/internal/agent/impl/agents.go:17-36` 为准。两者并非完全同一集合：AI Registry 还定义了 `content_ai` 与 `scheduler`，但 `impl.All` 没有对应具体实现。

### 3.2 平台护航组（不是业务人格，但属于海洋派）

**事实：** 平台宪法把 Auth/JWT、RBAC、审批策略、审计、EventBus、Command Dispatcher、Scheduler、ToolBridge、Agent Registry/执行、配置与可观测性列为 Kernel（`docs/governance/PLATFORM_CONSTITUTION.md:30-53`）。

**推断：** 它们不是应该再拟人化的业务 Agent，却是“完整航行”的护航舰：

- **EventBus / Pipeline**：让库存预警、折扣风险、利润和 Listing 等岗位解耦串联。
- **Scheduler**：为周期性观察提供节拍；当前从 Agent manifest 的 schedule trigger 生成任务（`backend-go/internal/aios/setup/router_integration.go:152-196`, symbol `SetupSchedulerAgentTriggers`）。
- **ActionCatalog + Approval + DispatchSafe + Audit + RBAC**：组成高风险动作闸门。
- **TrustScore / Entropy / Evolution**：分别回答“可给多少自主权”“系统/Agent 是否退化”“规则如何改进”。
- **Owner 控制台**：把建议、审批、执行结果和异常变成人能理解的驾驶舱。

不建议为这些机制各造一个聊天人格。机制应保持确定、可测试；业务 Agent 才负责判断。

### 3.3 当前已有的协作航线

**事实：** 项目文档记录了当前事件链：A5 红色库存警报 → G3 折扣风险；G3 阻断 → A6 利润看护；A6 亏损/阈值 → A2 Listing 优化；G0 异常数大于 3 → G1 驾驶舱（`docs/explanation-agent-pipeline.md:44-49`）。Scheduler 发布 `scheduler.tick.{agent_id}`，并带 `agent_id`、`decision_point` 等载荷（`backend-go/internal/platform/scheduler/scheduler.go:266-307`, symbols `emitTick`, `runTask`）。

**推断：** 这证明“多 Agent 航线”骨架已存在，但主要覆盖监控与建议，不等于候选品→上架→订单→净利润→复盘已经闭环。

---

## 4. 缺失或职责不清的“海洋派”岗位

这里的“缺失”不一定意味着立刻新建 Agent。优先把职责并入现有 Agent；只有当输入、输出、风险边界明显不同，才新增岗位。

| 缺口 | 当前事实 | 建议归属 | 是否建议新增 ID |
|---|---|---|---|
| 商品资料完整性 | A2 擅长优化文案/SEO，但没有明确成为标题、图片、类目、属性、重量尺寸的验收 Owner | 扩展 A2 为“商品资料与 Listing 质量负责人”，先做规则化 completeness gate | 暂不新增 |
| 平台费用真相 | A6 管利润、A10 管物流，但平台佣金/活动费/支付费没有清晰 Agent Owner | A6 负责汇总，平台费保持确定性 domain tool | 不新增人格 |
| 上架决策编排 | A8 做选品，A2 做 Listing，G0 做协调；谁给最终可上架/补资料/不建议结论不清晰 | 由 G0 编排，A6 输出利润真相，A7/G3 提供阻断条件 | 暂不新增 |
| 上架执行责任 | A9 有批量 Listing 更新，Registry 另有生命周期 `scheduler`，但受控发布岗位语义不清 | 将 A9 明确为“审批后的执行器”，不是决策者 | 暂不新增 |
| 订单异常 | A4/A11 处理客服售后，A5/G2/A10 管履约片段，但缺少订单状态异常总责 | 新设或从 A11 扩展“Order Fulfillment Agent” | L3 时再决定 |
| 结算与真实净利 | A6 有利润看护，但真实结算、退款、广告和物流账单归因需要明确 | 扩展 A6 为 Profit Truth Owner；结算计算留在 domain tool | 暂不新增 |
| 结果复盘 | 长期策略明确需要“复盘 Agent”，现有花名册没有独立岗位 | 新设 Review/Learning Agent，读取 episode 与业务结果，只提规则调整建议 | 建议新增，L4 |
| Agent 质量与演化 | `GET /agents/evolution` 当前直接返回 placeholder，episodes 为 0（`backend-go/internal/agent/handler.go:84-91`, symbol `Evolution`） | 先落地 episode/评估数据，再谈自演化 | 机制优先，不先造人格 |

**关键原则：** 当前方向明确写了“不优先增加更多 Agent 名字，而没有更清楚的输入、输出、风险边界和复核指标”（`docs/CURRENT_DIRECTION_AND_PRIORITIES.md:112-122`）。所以近期“海洋派扩军”应以补责任和证据为主，而不是编号增长。

---

## 5. 海洋离完成还有多远：实现现实审计

### 已有的真实骨架

- **事实：** 15 个具体 Agent 实现可以经统一 `Agent.Decide` 接口调度（`backend-go/internal/agent/impl/agents.go:12-36`）。
- **事实：** `Orchestrator.synthesizeOutput` 优先调用具体 Agent 实现；只有没有实现时才走 LLM provider 或 stub fallback（`backend-go/internal/ai/orchestrator.go:544-555`, symbol `synthesizeOutput`）。
- **事实：** Scheduler 有任务注册、并发防重入、重试与运行状态跟踪（`backend-go/internal/platform/scheduler/scheduler.go:158-219,221-343`, symbols `Register`, `Start`, `runTask`, `emitTick`）。
- **事实：** 治理契约已经规定高风险 Agent Action 的 mode、审批、幂等、审计与状态语义（`docs/governance/KERNEL_CONTRACTS.md`, sections 4–8）。

### 仍不能当成“海已煮沸”的证据

- **事实：** `Orchestrator.Run` 的注释仍称当前流程会产生 deterministic stub output；具体实现虽然优先，但多个实现内部也存在 stub/近似数据。例如 A2 的搜索量和趋势逻辑明确标为 stub（`backend-go/internal/agent/impl/listing_optimizer.go:177-185,223-234`）。
- **事实：** 没有具体实现、预算阻断、非生产 LLM 失败或 guardrail 阻断时，编排器会回落到 `stubFinalOutput`；生产环境 LLM 失败则返回错误（`backend-go/internal/ai/orchestrator.go:610-625,637-708`）。
- **事实：** Agent Evolution 接口仍是固定 placeholder，`episodes: 0`（`backend-go/internal/agent/handler.go:84-91`）。
- **事实：** 当前方向文档仍列出 runtime 生命周期、统一 Action 闸门、审批身份/RBAC、外部写安全、审计脱敏和高风险 UX 等优先风险（`docs/CURRENT_DIRECTION_AND_PRIORITIES.md:127-180` 及后续）。这些应以当前源码再次验证后逐项关闭，不能只因文档写了机制就视为完成。
- **推断：** 项目现在更像“船、岗位和航道大多已造好，但若干仪表仍接模拟数据，完整首航尚未用一票真实货物跑通并留下证据”。

---

## 6. 推荐的海洋派分层

为避免 Agent 互相越权，建议用四层而不是一张平铺花名册：

```text
Owner（目标、关键判断、风险接受）
  └─ 指挥与守门：G0 / G1 / G3 / A7 + Approval/RBAC/Audit
      └─ 业务判断：A1/A8 → A2 → A6 → A5/A10/G2 → A4/A11
          └─ 受控执行：A9 + Domain Services + Platform Adapters
              └─ 反馈学习：Profit Truth + Episode/Review + TrustScore/Entropy/Evolution
```

- **事实：** Owner 最终决定与当前 Copilot 定位来自 `docs/CURRENT_DIRECTION_AND_PRIORITIES.md:20-51`，也与 ETHOS 的 User Sovereignty 一致。
- **建议：** G0 只编排/升级异常，不自己发明领域真相；G1 只聚合与解释；G3/A7 只守门，不替代 Owner 审批。
- **建议：** A9 是“拿着已批准工单执行”的执行器，不应自行决定价格、库存或发布。
- **建议：** 真实利润、平台费用、库存和订单状态应来自确定性 Domain Services；LLM Agent 负责解释、权衡和建议，不负责凭空计算事实。
- **建议：** 复盘层只能提出策略/规则调整建议；在 Owner 批准与离线评估前，不自动改变生产规则或自主权。

---

## 7. 下一步：不是再造 10 个 Agent，而是完成一次首航

### P0 — 定义首航验收包

选择 1 个真实商品、1 个目标平台、1 个目的国，固定一份可复跑输入。验收输出至少包含：资料缺口、采购成本、重量尺寸、物流方案、平台费、售价、净利、利润率、风险、建议、证据来源和时间。

### P1 — 给现有岗位补清责任

1. A2 对商品资料完整性负责；缺数据时输出 `blocked/needs_data`，不伪造补齐。
2. A6 对平台费 + 全成本 + 利润真相汇总负责。
3. G0 对跨 Agent 编排与缺口升级负责，不改领域事实。
4. A9 只消费已审批、可审计、带幂等键的执行任务。

### P2 — 完成安全的建议到任务链

统一建议状态、Owner 采纳/拒绝/稍后、审批身份、ActionCatalog、`DispatchSafe`、审计、失败与重试。先 dry-run/sandbox，再讨论真实平台写回。

### P3 — 用真实结果闭环

把 Listing 发布结果、订单、物流账单、退款、结算、广告费与建议关联成 episode；比较预估与实际净利润。此时再引入独立 Review/Learning Agent 才有意义。

### 每片湖的退出门槛

**建议：** 每片湖都必须同时满足五个条件才算“煮完”：

1. Happy path 可复跑；
2. 缺数据、外部失败、超时、重复执行等主要错误路径可解释；
3. 关键逻辑有测试，端到端有验收证据；
4. 高风险动作有身份、审批、审计、幂等和恢复；
5. Owner 能在 UI 中看懂发生了什么并决定下一步。

---

## 8. 最终回答

**你的海洋：** 不是一个万能聊天机器人，也不是 Agent 数量，而是“一个人能可信、可控地经营完整跨境电商闭环”，从真实候选品到真实净利润，再把结果反馈给下一轮决策。

**当前海洋派：**

- 侦察：A1、A8
- 商品与增长：A2、A3、`content_ai`（仅注册定义）
- 经营真相：A6、A10
- 履约：A5、G2、A4、A11
- 守门与指挥：A7、G3、G0、G1
- 受控执行：A9
- 护航机制：EventBus、Scheduler、Command/ActionCatalog、Approval、Audit、RBAC、ToolBridge、TrustScore、Entropy、Evolution

**最需要补的不是更多 Agent，而是：** 商品资料完整性 Owner、平台费用/真实净利 Owner、订单异常 Owner、结果复盘 Agent，以及它们之间用真实数据跑通的首航证据。前三项尽量扩展现有 A2/A6/A11；只有结果复盘在有 episode 数据后值得新增独立 Agent。

**路线：** 搜索与核实现状 → 选第一片湖 → 把正确的端到端版本做完整 → Owner 验证 → 再扩下一片湖。这同时遵守 gstack 的 Boil the Ocean、Search Before Building 和 User Sovereignty。
