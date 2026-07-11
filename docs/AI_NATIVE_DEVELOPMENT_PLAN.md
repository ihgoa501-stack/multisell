# AI-Native AgentOS 长期愿景与架构方向 (2026-2027)

> [!CAUTION]
> **已冻结的长期参考。** 当前只开发 Owner 自用的跨市场选择、真实商品实验和最终净利润闭环；不得预设 Ozon、欧洲或任何平台。本文件不得驱动 AgentOS 扩张、多租户、外部 SaaS 或自治升级。见 `SELF_USE_OPERATING_DIRECTION.md`。

> **文档状态**：长期愿景 / 架构方向文档，不是当前 Sprint、季度排期或自动执行计划
> **面向对象**：后续入场的所有 AI 研发 Agent 与人类 Owner
> **核心目标**：定义 LingMirror 在安全、可审计、Owner 可控前提下走向 AI-Native AgentOS 的长期方向，并说明业务层、认知层和多行业扩展的技术设计原因。

## 使用边界（必须先读）

本文用于回答“长期往哪里走”，不直接决定“现在先做什么”。当前执行优先级、验收口径和事实状态以以下文档为准：

- `docs/governance/*`
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`
- `docs/PROJECT_STATUS.md`
- `docs/reference-module-catalog.md`

若本文与上述当前事实源冲突，以上述当前事实源为准，除非 Owner 明确覆盖。

### 当前阶段口径

到 Q4 2026 之前，LingMirror 的产品形态仍应保持为 **Copilot**：

```text
AI recommends; the Owner decides.
```

价格、库存、订单、资金、外部平台发布、账号权限、凭证和破坏性数据变更仍必须默认走 read-only / suggestion / approval-required 路径。本文提到的自进化、多行业扩展和更高自治能力，都是长期方向，不得被后续 Agent 解释为当前可直接实现的生产自治。

### 架构红线

Go Platform Kernel 是唯一执行权威。Python 认知服务、DSPy、GEPA、Honcho、Memory 或任何 AIOS 组件都不得：

- 直接修改业务数据库；
- 直接调用真实外部平台写接口；
- 绕过 Auth、RBAC、Approval、Audit、ActionCatalog 或 Command 契约；
- 自行批准高风险动作；
- 自动推广会影响生产建议或生产执行的 prompt / model / policy 变更。

认知层只能输出分析、检索结果、排序结果、prompt 候选、Owner 偏好摘要，或结构化的 proposed `AgentAction`。所有高风险动作必须回到 Go Kernel 的 AgentAction / ActionCatalog / Approval / RBAC / Audit / Command 执行链路。

### CEO / CTO 读法

后续 Agent 读取本文时，必须同时保留 CEO 和 CTO 两种判断口径：

- **CEO 口径**：先证明一个清晰付费场景和 daily Owner workflow，不要急着扩多行业或宣传自进化。
- **CTO 口径**：长期 AIOS 可以保留，但 sidecar、Tool、Memory、Prompt、Sandbox、CI/E2E 必须有硬契约和可验证门禁。

#### First Customer / ICP

当前第一客户假设应保持窄口径：

- **目标客户**：小型或中型跨境电商卖家 / 运营团队，SKU 数量约 50-500，正在处理 Ozon、Shopee 或相似平台。
- **核心痛点**：候选商品、平台费、物流成本、汇率、库存、结算和退款数据分散，导致“商品能不能卖、订单会不会亏”难以快速判断。
- **购买触发**：上新效率低、亏损订单频发、平台规则复杂、人工审批慢、Owner 无法追踪 AI 建议依据。
- **首个付费楔子**：跨境电商利润与上架 Copilot，而不是通用 AIOS 或全自动公司。

#### First Paid Pilot

进入任何 Phase 3 / Phase 4 之前，应先完成一个可度量的付费或准付费试点：

```text
一个 Owner / 一个平台 / 一个 SKU 或类目批次
-> read-only 或 sandbox 数据接入
-> 商品利润与上架建议
-> Owner 审批
-> 受控执行或模拟执行
-> 结果复盘
```

试点必须记录：

- 上架决策耗时是否下降；
- 毛利测算准确性是否提升；
- 是否避免亏损商品或亏损订单；
- Owner 接受 / 拒绝 / 修改建议的比例；
- 静默失败数是否为 0；
- 所有失败是否有 Owner 可读解释和恢复建议。

#### ROI Scoreboard

任何“ROI”“商业变现”“现金流”表述都必须由 scoreboard 支撑。至少追踪：

- 每个 listing 决策节省的人工时间；
- 推荐毛利 vs. 实际或模拟结算毛利偏差；
- 被拦截的亏损商品 / 亏损订单数量；
- 推荐接受率、覆盖率、回滚率、人工改写率；
- 从候选商品到 Owner 可读建议的 time-to-decision；
- 接入、维护和人工复核成本。

没有这些证据前，只能说“具备可度量 ROI 的路径”，不能说“已经带来 ROI”或“立刻带来现金流”。

#### Sales / Investor Claim Boundary

本文是内部长期愿景，不等同于对外承诺。对客户、投资人、官网和演示材料，当前允许的主张是：

- LingMirror 是跨境电商决策 Copilot；
- AI 生成可解释建议，Owner 审批高风险动作；
- 系统强调 dry-run / sandbox / approval / audit；
- 目标是帮助 Owner 更快、更安全地判断商品、订单、履约和风险。

当前禁止对外承诺：

- 全自动经营公司；
- 高风险生产动作无需 Owner 批准；
- 保证销量提升、利润提升或现金流改善；
- 自动完成金融、AML、报关、付款、发票或合规判断；
- 自进化模型自动变好并自动上线。

涉及收入、毛利、销量、费用、物流成本、平台风险的示例，必须标注“模拟”“估算”或“不构成保证”。

#### Business Verified Gate

当前阶段真正的 CEO/CTO 共同 gate 是 Business Verified，而不是完成更多架构文档。进入认知层或多行业扩展前，至少需要：

- 产品闭环浏览器 E2E：候选商品 -> 完整性检查 -> 成本/物流/平台费/利润 -> 上架建议 -> Owner 审批 -> 受控执行 -> 结果复盘。
- 履约闭环浏览器 E2E：订单 -> 库存/物流选择 -> 运费快照 -> 结算/利润检查 -> 异常建议 -> 审批或人工处理。
- 高风险动作 gate：price / inventory / order / money / publish / credential 类动作均能证明审批、RBAC、审计、幂等、失败可见。
- Trial readiness：至少 5-10 次 seeded realistic browser runs；静默失败为 0；所有失败有 Owner 可读状态。
- CI / release gate：E2E seed、迁移、回滚、备份恢复、release tagging 必须有可追溯证据，不能只靠本地人工判断。

#### Sidecar / Tool / Memory / Prompt Contract

Phase 3 前必须先写出可执行契约，最低要求：

- **Sidecar ingress**：任何非 Go 认知服务只能通过认证 IPC/API 向 Go Kernel 提交 `ProposedAgentAction` 或 analysis artifact。不得直接访问 domain service、integration adapter、command dispatch、mutation tool 或业务数据库。
- **Tool Registry**：Tool Registry 是能力发现和调用元数据，不是独立执行权威。mutation-capable tool 必须生成 `ProposedAgentAction`，或在 Go Kernel 已完成 ActionCatalog、RBAC、approval、mode、idempotency、audit 后执行。
- **IPC envelope**：必须包含 service identity、tenant、actor、correlation ID、schema version、prompt/model/policy version、source citations、data sensitivity、requested mode、timeout/retry/failure semantics。
- **Memory lineage**：memory item 必须记录 source trace/action ID、来源表或文档、提取任务、embedding model/version、freshness、tenant scope、deletion policy、training eligibility。
- **Memory advisory only**：Memory 只能作为建议上下文，不得授予权限、豁免审批、改变风险等级、覆盖合规政策或选择 production mode。
- **Prompt/model registry**：prompt version、model version、dataset version、evaluator version、approval record、rollout mode、shadow metrics、rollback target、Owner 可读 diff 必须可追踪。
- **Sandbox/outbound enforcement**：除文档约定外，还需要静态检查或 CI 规则禁止 adapter 中绕过统一构造器直接创建 raw `http.Client` 或 provider SDK mutation client。

---

## 整体演进路线图 (Roadmap Overview)

整个系统的长期演进分为四个方向，以**“安全底座先行 -> 电商场景跑通 -> 认知进化落地 -> 多行业扩展”**为主线。阶段三、阶段四是条件触发的长期方向，不是当前排期承诺：

```
                      【阶段一：AI 动作审批与 Owner 控制台】
                                      │
                                      ▼
                      【阶段二：电商 stateful 模拟器与闭环】
                                      │
                                      ▼
                      【阶段三：Python 认知大脑与 DSPy 自进化（条件触发）】
                                      │
                                      ▼
                      【阶段四：金融与外贸垂直套件（条件触发）】
```

进入后续阶段之前，必须先通过对应 gate：

- **电商闭环 gate**：Owner 可在浏览器中完成候选商品 -> 完整性检查 -> 成本/利润测算 -> 上架建议 -> 审批 -> 受控执行 -> 结果复盘。
- **履约闭环 gate**：Owner 可完成订单 -> 库存/物流选择 -> 运费快照 -> 结算/利润检查 -> 异常建议 -> 审批或人工处理。
- **安全 gate**：高风险动作具备 dry-run / sandbox / production 区分、审批、RBAC、审计、幂等、失败可见和恢复提示。
- **数据 gate**：有足够干净的审批、拒绝、结果和业务收益数据，才能启动 prompt/model 自进化。
- **扩展 gate**：非电商垂直领域必须先有 ICP、工作流地图、合规风险地图和 sandbox adapter 规格。

---

## 1. 阶段一：AI 动作审批中枢与 Owner 决策台 (Action Gate & Owner Cockpit)

> 当前状态：安全底座与执行门禁已有实现进展。后续 Agent 在引用本阶段时，必须先核对 `docs/PROJECT_STATUS.md`，不得把已完成项当成新建任务。现阶段重点应是收口旁路、完成 E2E 证明、统一 Owner 可理解的验收证据。

### 1.1 需要开发什么？
1. **统一动作审批网关 (Unified Action Execution Gate)**:
   * 所有高风险操作（调价、上架、付款）必须回到统一 AgentAction / ActionCatalog / Approval / RBAC / Audit / Command 链路。
   * 继续检查并收口仍绕过统一动作门禁的 Owner 工作台、listing-task、integration 或其他历史路径。
2. **Owner 决策总控台 (Owner Cockpit UI)**:
   * Owner 工作台、审批队列和动作执行页面必须让非技术 Owner 看懂：发生了什么、为什么重要、批准会怎样、拒绝或等待会怎样、当前是 dry-run/sandbox/production、审计在哪里。
   * 高风险动作确认组件应统一呈现风险：变更前后对比、环境模式、审批要求、审计去向以及异常回滚提示。

### 1.2 为什么这么做？(Why)
* **业务安全性**：AI 程序员在开发系统时，无法保证生成的 Agent 决策绝对无误。必须建立“AI 提出结构化动作；Go Kernel 执行策略、审批、审计和边界；Owner 批准高风险变更”的控制链。
* **人机协作降低脑力成本**：非技术 Owner 不需要看复杂的后台日志或 Trace 链路。UI 必须将 Agent 的分析翻译成最直观的商业语言。所有销量、利润或风险影响示例都必须标注为估算或模拟，不能被写成保证结果。

---

## 2. 阶段二：电商领域套件与 Stateful 模拟器 (E-Commerce Stateful Simulation)

### 2.1 需要开发什么？
1. **Ozon/Shopee Stateful Mock 驱动层**:
   * 实现 Go 后端的 `MockPlatformAdapter`，让上架、订单同步、运费查询、佣金扣减等 API 读写本地或隔离环境中的 mock 状态。
   * 明确模拟器覆盖场景、已知差距、失败模式、数据重置方式和不可代表真实平台的边界。
2. **电商双轨运行适配器 (Production Adapters)**:
   * 真实 Ozon、Shopee 客户端必须默认 dry-run 或 sandbox。进入 production 前必须具备有效审批、RBAC、审计、幂等键、外部 reference ID 捕获、失败可见和恢复提示。
   * `FailSafeRoundTripper` 只能作为外部请求安全组件之一，不能替代审批、审计和执行门禁。
3. **电商两大经营闭环**:
   * **商品闭环**：候选商品 -> 完整性自检 -> 自动测算物流/平台费 -> 推荐售价 -> 审批 -> 上架 -> 销量监控。
   * **履约闭环**：新订单 -> 自动匹配物流与备货 -> 运费账单快照 -> 利润核算 -> 异常报警（断货、折扣叠加）。

### 2.2 为什么这么做？(Why)
* **API 封禁与真实财务避险**：直接用真实 API 调试极易触发平台风控（封店）或产生真实物流扣费。Stateful 模拟器能让 AI 开发 Agent 在受控环境中验证关键路径，但不能声称覆盖真实平台的全部行为。
* **跑通第一个 Vertical（垂直App）**：跨境电商是当前最成熟、闭环最短的场景。跑通电商的商品与履约闭环，是形成可度量 ROI 的第一条路径；在 Business Verified 之前，不得宣称已经带来 ROI 或现金流改善。

---

## 3. 阶段三：Python 认知大脑与 DSPy 自进化 (Cognitive Brain & DSPy)

> 启动条件：只有在电商核心闭环、沙盒/生产隔离、审计可观测性和足够的审批/结果数据通过 gate 后，才可进入本阶段。进入本阶段前必须先写清 IPC、Auth、Audit、Correlation ID、Prompt/Model 版本、回滚、成本控制和数据保留契约。

### 3.1 需要开发什么？
1. **Python 认知微服务 (`python-agentos/`)**:
   * 采用 FastAPI 搭建独立的 Python 服务，通过受控 IPC 与 Go 通信。
   * Python 服务只能作为认知侧车：分析、检索、排序、生成建议、生成 proposed `AgentAction`，不得直接执行生产变更。
2. **GEPA 三层记忆系统**:
   * 基于 SQLite-vec 向量数据库和 FTS5 全文索引，实现 **Working Memory**（单会话上下文）、**Episodic Memory**（历史成功/失败决策情景）和 **Semantic Memory**（业务知识库与 Owner 长周期偏好）的检索。
   * Memory 必须定义租户边界、PII/secret 脱敏、保留/删除策略、来源引用、置信度和审计方式。
3. **DSPy Prompt 自动编译器**:
   * 收集 Owner 审批历史数据。针对被拒绝的决策，可在离线环境运行 DSPy 算法生成 prompt 候选。
   * Prompt / model / policy 变更必须经过离线评测、版本登记、Owner 可读变更摘要、审批推广、shadow mode 和回滚机制，禁止自动影响生产建议或生产执行。
4. **Honcho 辩证用户建模**:
   * 学习 Owner 的决策风格（激进型 vs. 保守型），并在生成建议时自动进行风格适配。
   * Owner 偏好只能影响建议排序和解释风格，不能覆盖硬性安全政策、合规要求或审批门禁。

### 3.2 为什么这么做？(Why)
* **AI 时代的上限是受控学习能力**：如果 Agent 逻辑完全写死在代码里，它难以从 Owner 反馈和业务结果中学习。Python 认知层用于生成可评测、可版本化、可审批、可回滚的改进候选，而不是自动上线自我修改。
* **利用 Python 最强 AI 生态**：DSPy、NousResearch Hermes、sqlite-vec 等前沿 Agent 技术几乎全部是用 Python 开发的。双轨架构可以利用这些工具，但生产执行仍必须由 Go Kernel 统一控制。

---

## 4. 阶段四：金融与外贸套件扩展 (Finance & Trade Verticals)

> 启动条件：本阶段属于长期扩展方向，不是当前排期。进入任何非电商垂直领域前，必须先完成电商闭环验证、业务 ROI 证明、合规风险地图、目标客户画像、sandbox adapter 规格和 Owner 明确授权。

### 4.1 需要开发什么？
1. **金融能力扩展**:
   * 先复用并扩展现有 finance 领域事实源，避免重复创建领域概念。
   * 在合规风险明确后，再评估是否需要 `mock_transactions`、`ledger_entries` 等新增表。
   * 实现 **`BankAdapter` 接口**：对接模拟/真实 Stripe、TransferWise 等银行网关。
   * 编写 **资金监控与合规建议 Agent**：监控资金池水位，分析财务日报，提示异常账目。涉及付款、转账、投资、AML 或真实资金动作时必须保持建议/审批模式。
2. **外贸 Domain 模块 (`internal/domain/foreigntrade`)**:
   * 建立 `rfq_records`、`quotations`、`invoices` 数据库表。
   * 实现 **`RFQAdapter` 接口**：对接外贸询盘平台和海关数据库。
   * 编写 **询盘响应与报关建议 Agent**：AI 自动解析询盘 PDF，匹配供应商价格，生成待审核 B2B 报价草案。正式报价、报关、发票和平台提交必须经过 Owner 审批。

### 4.2 为什么这么做？(Why)
* **商业天花板**：当电商的“安全底座 + 动作审批 + AI 认知大脑”被真实业务验证后，LingMirror 才具备向更多业务垂直领域迁移的基础。
* **低边际扩展成本**：因为 Go 内核（权限、审计、审批）与认知侧车解耦，后续可在同一套底座上探索金融和外贸套件。但这些方向涉及资金、合规、报关、发票和新客户画像，必须按独立产品验证处理，不能被视为电商阶段的自然延伸任务。
