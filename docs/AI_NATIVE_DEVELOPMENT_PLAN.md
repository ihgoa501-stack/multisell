# 通用 AI-Native AgentOS 整体开发规划蓝图 (2026-2027)

> **文档状态**：战略规划文档（开发纲领）
> **面向对象**：后续入场的所有 AI 研发 Agent 与人类 Owner
> **核心目标**：定义在 Phase P1（安全与沙盒底座）完成后，如何开发业务层、认知层及多行业扩展套件，并说明其商业与技术设计原因。

---

## 🗺️ 整体开发规划路线图 (Roadmap Overview)

整个系统的演进分为四个阶段，以**“安全底座先行 -> 电商场景跑通 -> 认知进化落地 -> 多行业降维打击”**为主线：

```
                      【阶段一：AI 动作审批与 Owner 控制台】
                                      │
                                      ▼
                      【阶段二：电商 stateful 模拟器与闭环】
                                      │
                                      ▼
                      【阶段三：Python 认知大脑与 DSPy 自进化】
                                      │
                                      ▼
                      【阶段四：金融与外贸垂直套件（多行业）】
```

---

## 1. 阶段一：AI 动作审批中枢与 Owner 决策台 (Action Gate & Owner Cockpit)

### 1.1 需要开发什么？
1. **统一动作审批网关 (Unified Action Execution Gate)**:
   * 在 Go 后端实现 `/api/v1/ai/actions/:id/execute`。所有高风险操作（调价、上架、付款）必须通过此接口。
   * 开发 **Action Catalog**（动作风险评级目录），硬编码安全红线。
2. **Owner 决策总控台 (Owner Cockpit UI)**:
   * 开发 Next.js 页面 `/owner`、`/approval` 和 `/actions`。
   * 开发通用的 **“高风险动作确认弹窗组件”**，以统一的卡片形式呈现风险：变更前后对比、环境模式（沙盒/生产）、审计去向以及异常回滚提示。

### 1.2 为什么这么做？(Why)
* **业务安全性**：AI 程序员在开发系统时，无法保证生成的 Agent 决策绝对无误。我们必须建立**“AI 只能想，Go 负责卡，人负责批”**的绝对控制链。
* **人机协作降低脑力成本**：非技术 Owner 不需要看复杂的后台日志或 Trace 链路。UI 必须将 Agent 的分析翻译成最直观的商业语言（如：“*建议将 SKU-A 在 Ozon 价格下调 10%，预计可提升 15% 销量，毛利率由 25% 降至 22%，是否批准？*”）。

---

## 2. 阶段二：电商领域套件与 Stateful 模拟器 (E-Commerce Stateful Simulation)

### 2.1 需要开发什么？
1. **Ozon/Shopee Stateful Mock 驱动层**:
   * 实现 Go 后端的 `MockPlatformAdapter`，让上架、订单同步、运费查询、佣金扣减等 API 读写本地的 `mock_` 数据库表。
2. **电商双轨运行适配器 (Production Adapters)**:
   * 编写真实的 Ozon、Shopee 客户端，自动包装 `FailSafeRoundTripper` 安全网关，在生产模式下发送真实 API 请求。
3. **电商两大经营闭环**:
   * **商品闭环**：候选商品 -> 完整性自检 -> 自动测算物流/平台费 -> 推荐售价 -> 审批 -> 上架 -> 销量监控。
   * **履约闭环**：新订单 -> 自动匹配物流与备货 -> 运费账单快照 -> 利润核算 -> 异常报警（断货、折扣叠加）。

### 2.2 为什么这么做？(Why)
* **API 封禁与真实财务避险**：直接用真实 API 调试极易触发平台风控（封店）或产生真实物流扣费。Stateful 模拟器能让 AI 开发 Agent 自主完成 100% 逼真的全链路测试。
* **跑通第一个 Vertical（垂直App）**：跨境电商是当前最成熟、闭环最短的场景。跑通电商的商品与履约闭环，能为系统立刻带来商业变现和 ROI 现金流。

---

## 3. 阶段三：Python 认知大脑与 DSPy 自进化 (Cognitive Brain & DSPy)

### 3.1 需要开发什么？
1. **Python 认知微服务 (`python-agentos/`)**:
   * 采用 FastAPI 搭建独立的 Python 服务，通过 gRPC/NATS 与 Go 通信。
2. **GEPA 三层记忆系统**:
   * 基于 SQLite-vec 向量数据库和 FTS5 全文索引，实现 **Working Memory**（单会话上下文）、**Episodic Memory**（历史成功/失败决策情景）和 **Semantic Memory**（业务知识库与 Owner 长周期偏好）的检索。
3. **DSPy Prompt 自动编译器**:
   * 收集 Owner 审批历史数据。针对被拒绝的决策，自动运行 DSPy 算法在后台优化 Agent 的提示词，让 Agent 随着使用变得越来越聪明。
4. **Honcho 辩证用户建模**:
   * 学习 Owner 的决策风格（激进型 vs. 保守型），并在生成建议时自动进行风格适配。

### 3.2 为什么这么做？(Why)
* **AI 时代的上限是“自我进化”**：如果 Agent 逻辑写死在代码里，那它只是个传统软件。利用 Python 构建认知大脑，让 Agent 每天复盘自己的决策，实现**“AI 逻辑自净化，越用越好用”**。
* **利用 Python 最强 AI 生态**：DSPy、NousResearch Hermes、sqlite-vec 等前沿 Agent 技术几乎全部是用 Python 开发的。双轨架构能直接释放这些 AI 底层工具的全部潜力。

---

## 4. 阶段四：金融与外贸套件扩展 (Finance & Trade Verticals)

### 4.1 需要开发什么？
1. **金融 Domain 模块 (`internal/domain/finance`)**:
   * 建立 `mock_transactions`、`ledger_entries` 数据库表。
   * 实现 **`BankAdapter` 接口**：对接模拟/真实 Stripe、TransferWise 等银行网关。
   * 编写 **量化理财与合规 Agent**：监控资金池水位，分析财务日报，拦截洗钱与违规账目。
2. **外贸 Domain 模块 (`internal/domain/foreigntrade`)**:
   * 建立 `rfq_records`、`quotations`、`invoices` 数据库表。
   * 实现 **`RFQAdapter` 接口**：对接外贸询盘平台和海关数据库。
   * 编写 **询盘响应与报关 Agent**：AI 自动解析询盘 PDF，匹配供应商价格，自动生成正式 B2B 报价单。

### 4.2 为什么这么做？(Why)
* **降维打击与商业天花板**：当电商的“安全底座 + 动作审批 + AI 认知大脑”跑通后，LingMirror 已经是一个成熟的 **通用业务 Agent 操作系统**。
* **低开发成本扩展**：因为 Go 内核（权限、审计、审批）与 Python 大脑是完全解耦的，AI 程序员后续只需用几周时间，就能在同一套底座上长出金融和外贸套件，以极低的成本将产品线拓宽数倍，获取多维度的商业模式（SaaS、交易抽成、企业版定制）。
