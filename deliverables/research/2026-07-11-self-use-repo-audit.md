# 凌镜 Owner 自用方向：仓库能力与真实成交闭环审计

日期：2026-07-11
范围：只读审计；未修改任何现有代码或文档。
目标：判断现有仓库能否支撑 Owner 本人完成第一个 Ozon 商品实验，并给出收敛后的开发顺序。

## 一、结论

**方向应该正式改为：凌镜是 Owner 单人自用的跨境商品实验与经营内部系统。** 当前不应继续双产品、对外 SaaS、多租户商业化或“更多 Agent/更多页面”的建设。

现有仓库不需要推倒重来。商品候选、利润预估、审批与审计、Listing、Ozon API、订单、售后、结算、逐单利润等底座已经存在，约可复用一半以上的技术能力。但这些模块目前是“并排存在”，不是一个能证明真实经营结果的实验闭环。

最关键的缺口不是再增加模块，而是缺少一个统一的 **商品实验（Launch Experiment）业务对象和事实账本**，将以下链路锁定在同一个实验 ID 下：

```text
20 个市场线索
→ 硬条件淘汰
→ 3 个候选评分
→ 1 个批准商品
→ 预算与停止条件
→ Ozon 发布
→ 陌生客户订单
→ 签收与退货观察期
→ 平台结算对账
→ 最终净利润
→ 停止 / 换品 / 修正再试 / 小幅加码
```

因此建议保留平台与经营底座，冻结大教堂方向，先用人工录入补足真实数据，再做自动化。第一阶段唯一验收不是“页面完成”，而是系统能对一轮真实 Ozon 实验给出可追溯的最终结论。

## 二、审计方法与证据边界

- 先读取 `OWNER_FIRST_PROTOCOL.md`、`PLATFORM_CONSTITUTION.md`、`AGENT_DEVELOPMENT_PROTOCOL.md`、`KERNEL_CONTRACTS.md`。
- 代码理解优先使用仓库 `.codegraph/` 的 `codegraph explore`，再针对 Markdown 和精确字段使用 `rg`，并复核相关源码。
- 本报告判断的是“代码结构和当前实现是否具备能力”，**不代表真实 Ozon 账号、凭证、费用口径或 API 请求已在生产环境验证**。
- 未运行全量测试，因为任务是只读方向审计，不是发布验收；已有测试存在不等于覆盖真实账号、真实结算和最终利润定义。

## 三、可以直接复用的能力

### 1. 候选商品与采集线索：保留并收窄

证据：`backend-go/internal/domain/candidate/`。

现有 `CandidateProduct` 已有标题、来源 URL、采购价、供应商、重量与尺寸、HS Code、目标售价、目标平台、目的国和完整度；`CollectLead` 可保存前台搜索/列表页线索。它适合作为“20 个候选链接 → 详情采集”的底座。

可复用：

- 来源 URL 去重；
- 人工创建与补字段；
- 包装重量/尺寸、采购价、目标售价；
- 完整度检查；
- 候选状态机和前端候选页。

不足：没有 Ozon 需求证据（销量、评价增长、搜索结果数量、头部集中度）、合规/侵权排除项、供应商数量、最小起订量、最坏损失、加权评分和“20→3→1”批次关系。现有 seed 商品含带电、食品、服装等首轮应排除品类，必须明确标识为演示数据，不能参与真实实验。

### 2. 成本与预估利润：保留计算框架，替换默认假设

证据：

- `backend-go/internal/domain/profit/model.go`
- `backend-go/internal/domain/profit/service.go`
- `backend-go/internal/domain/sourcing/profit.go`
- `backend-go/internal/domain/platformfee/`
- `backend-go/internal/domain/logistics/`
- `backend-go/internal/domain/exchangerate/`
- `backend-go/internal/domain/tariff/`

现有能力可保存商品级利润快照，并包含采购、物流、平台费、关税、其他成本、收入、利润率；订单级也有 `OrderProfitRecord`。

但当前预估仍大量依赖简化默认值：默认运费、默认 15% 平台费、默认 5% 关税、目标价 2% 作为“其他成本”；`sourcing/profit.go` 还以目的地区域静态费率估算。它只能作为初筛，不能作为批准花钱的可信依据。

首轮应保留计算器代码结构，但要求每个成本项有 `actual / quoted / estimated / unknown` 来源状态、币种、汇率时间和证据。未知成本不得被静默默认值掩盖。

### 3. 审批、RBAC、审计与安全门禁：完整保留

证据：治理文档及 `actionpolicy`、Approval、Command Dispatcher、Action Catalog、Audit、EventBus Mutation Guard。

自用不等于取消门禁。真实发布、广告花费、采购、库存、订单、退款和资金动作损失的是 Owner 自己的钱。以下原则继续有效：

- 建议与执行分离；
- Ozon 发布和资金动作必须 Owner 批准；
- dry-run / sandbox / production 明确区分；
- 外部副作用有审计、幂等与失败恢复；
- 不让 Agent 直接写核心业务表。

多角色企业级流程可简化为单 Owner 决策体验，但底层审批与审计机制不应删除。

### 4. Listing 与 Ozon 适配器：保留，但视为“待真实验收”

证据：

- `backend-go/internal/domain/listing/`
- `backend-go/internal/domain/integrations/adapter.go`
- `backend-go/internal/domain/integrations/ozon.go`

`PlatformAdapter` 已定义 Publish、SyncStatus、SyncInventory、PushTracking、FetchOrders、FetchSettlements、FetchReturns、FetchRaw。Ozon 实现也存在对应方法，并使用账号凭证访问外部 API。这是选择 Ozon 做第一市场的重要复用优势。

但仓库证据只证明“有代码和 fixture 测试”，不能证明：

- 当前 Ozon 卖家账号权限与 API 凭证可用；
- 发布 payload 符合 Owner 实际类目；
- FBS/FBO 模式、币种、订单状态和费用交易类型映射准确；
- 真实订单、退货、结算均可稳定同步；
- API 版本未漂移。

因此第一轮必须先做只读凭证验证，再 sandbox/dry-run，再以一个商品做 production canary。不要同时扩展 Shopee、Shopify 或更多平台。

### 5. 订单、售后、结算：保留

证据：

- `backend-go/internal/domain/order/model.go`
- `backend-go/internal/domain/aftersales/model.go`
- `backend-go/internal/domain/settlement/model.go`
- `backend-go/internal/domain/settlement/recalculator.go`

现有订单模型能记录付款、发货、签收、取消时间和基础费用；售后模型能记录退货数量、退款金额、收货/退款状态；结算模型能保存交易行、费用、退款、净额和对账状态。它们是最终利润闭环的重要底座。

## 四、当前的干扰项与应冻结能力

### 产品方向干扰

立即冻结，不删除历史：

- 双产品独立架构和 Outcome Proof Protocol；
- Evidence Warranty 与自动补救；
- 跨客户 Do-Not-Launch 聚合；
- 结果收费、公共 API、定价、账单、自助注册；
- 多租户/客户 workspace 扩张；
- 面向外部商家 onboarding、销售和支持体系；
- 为 40+ 页面做统一视觉大翻修；
- 与首轮 Ozon 实验无关的 Shopify、Shopee、Amazon、Lazada 扩展；
- 新 Agent 名称、MOA 大编排、自治升级、Agent 市场；
- 不能影响选品、成交、止损或最终净利润的仪表盘美化。

这些不是永久删除，而是不得进入活跃 roadmap 和 TODO。只有 Owner 完成至少三轮真实实验、出现真实重复劳动后，才按证据解冻自动化。

### 代码层面的噪声

- `backend-go/internal/domain/mock/` 的 MockOrder、MockSettlement 和 Owner mock dashboard 不能出现在生产实验判断中。
- 候选 seed 数据必须与真实数据强隔离；`is_seed_data` 已存在，应在所有实验查询中强制排除。
- 广告 Agent 有分析字段，但部分工具明确返回 stub；不能把它当 Ozon 广告真实同步。
- 60+ 领域模块和大量 Agent 可继续留在仓库，但本阶段不应成为开发顺序的驱动力。

## 五、真实 LLM / Mock 状态

证据：`backend-go/internal/ai/orchestrator.go`、`llm_provider.go`、`moa.go`。

判断比旧“全部是 stub”描述更细：

- 系统已经支持 OpenAI-compatible、Anthropic 等真实 Provider；生产环境禁止 stub Provider。
- Orchestrator 在 Provider 为真实模型时会调用真实 LLM，失败时生产环境不会静默伪装成成功建议。
- 但工具调用事件仍有 stub 路径，未注册实现或开发环境会回落到 deterministic stub；默认未配置 Provider 时使用 stub。
- MOA `synthesize()` 仍是确定性结构拼装，不是 LLM 综合推理。

所以：**“支持真实 LLM”不等于“真实选品 Agent 已完成”。** 首轮商品决策必须以可核查市场数据、成本证据和 Owner 审批为主；LLM 只可整理与解释，不可创造销量、费用、合规或利润事实。UI 必须醒目标明 `real / estimated / mock / stub / unknown`。

## 六、关键闭环缺失

### P0：没有统一商品实验实体

当前 Candidate、ProfitSummary、ListingTask、Order、AfterSales、Settlement 各自存在，但没有一个稳定 `experiment_id` 串联它们。无法回答：这笔广告费、订单、退货、结算和库存损失究竟属于哪一轮实验。

需要一个最小 `launch_experiment` 及关联账本，至少包含：市场、三个候选与胜出商品、开始/结束时间、状态、Owner 批准、预算上限、不可回收损失上限、观察窗口、结论。

### P0：没有实验预算与真实现金账本

仓库中的 budget 主要是 LLM 成本或广告分析输入，不是商品实验资金治理。缺少采购、样品、包装、国内运输、跨境物流、广告、平台费、税费、汇兑、退货/销毁、售后等现金项目，也没有 3,000 元总上限和 1,200 元不可回收损失触发器。

### P0：没有可执行停止条件

没有把“14 天无单 + 100 详情访问”“广告花费 300 元无单”“300 点击无单”“实际单件贡献利润不足”“前 5 单 2 笔取消/退款/退货”“累计不可回收损失”等定义为版本化规则、事实输入和触发记录。现有 Agent 建议不能替代确定性停止规则。

### P0：最终有效成交定义缺失

现有 Order 状态可表达 paid/shipped/delivered/cancelled，AfterSales 可表达退款退货，但没有：

- 陌生客户/关联订单排除标记；
- “有效成交”和“最终有效成交”两个业务状态；
- 每个平台适用的退货/争议观察窗口；
- 窗口关闭、结算完整、无未决售后的 finalization gate；
- late adjustment 后从 finalized 重新打开的机制。

### P0：最终净利润公式不完整且存在重复扣费风险

`settlement.Recalculator.computeProfit()` 已读取 sale、platform_fee、payment_fee、shipping_fee、refund，但仍有关键问题：

- `revenue := orderSaleAmt - orderSaleFee` 后，`totalCost` 又加 `orderSaleFee`，有重复扣佣金的风险；
- 没有包装、国内运输、广告、税费/关税、汇兑、售后补偿、销毁/不可售库存等完整成本；
- `TariffCost` 明确写为 0；
- 最终利润没有 finalization 状态，订单利润随时可能仍是 provisional；
- 没有实验级汇总，也没有按真实结算币种与汇率证据统一换算。

因此现有 `profit_amount` 不能直接命名为“最终净利润”。修复和定义该口径应先于任何智能加码建议。

### P1：候选筛选缺少真实市场证据与评分

需要保存每条证据的来源 URL、采集时间、原始值和可信度；对需求、竞争、利润、物流退货、合规、小批补货做确定性评分。LLM 可解释，不应直接生成无来源分数。

### P1：Ozon 真实链路未完成账号级验收

需要使用 Owner 账号完成 credentials → 类目/属性读取 → dry-run payload → 单商品发布 → 状态同步 → 真实订单 → 退货 → 结算的逐阶段验证。每个外部写操作仍要单独批准。

## 七、新开发顺序

### Sprint 0：战略事实源归一（先做）

1. 把 Owner 自用、单经营主体、Ozon 首轮实验写成最高优先级。
2. 冻结双产品、对外 SaaS 和多平台扩张文档。
3. 统一“下单 / 有效成交 / 最终有效成交 / provisional profit / final net profit”的词典。
4. 明确第一轮预算和停止条件可由 Owner 配置，但初始基线为总预算 3,000 CNY、不可回收损失 1,200 CNY。

验收：任一新 Agent 读当前事实源后，不会继续建设大教堂或外部客户功能。

### Sprint 1：商品实验最小闭环（人工优先）

1. 新建 `launch_experiment`、候选评分、实验成本账本、停止规则与触发记录。
2. 复用 Candidate，支持一轮 20 个线索、3 个入围、1 个胜出。
3. 成本项必须标记 actual/quoted/estimated/unknown；unknown 阻止批准。
4. Owner 批准胜出商品、预算和 production 发布。

验收：系统能在不调用 LLM 的情况下，完整记录并选择唯一商品；总预算无法被静默突破。

### Sprint 2：Ozon 单商品 production canary

1. 验证真实账号凭证和只读 API。
2. 校验类目、属性、物流模式与费用字段。
3. dry-run/sandbox 后，由 Owner 批准发布一个商品。
4. 所有发布状态、外部 ID、失败原因和重试进入审计。

验收：Ozon 后台能看到唯一真实商品，系统能读回并关联同一 experiment_id。

### Sprint 3：成交、流量和止损

1. 同步订单，并标记关联/非自然订单排除项。
2. 人工录入或同步曝光、详情访问、点击、广告花费。
3. 确定性评估停止条件；只提醒和阻断新增投入，不自动取消订单或广告。
4. 区分 ordered、paid、shipped、delivered、有效成交。

验收：任一停止条件触发时，系统能说明数据、阈值和建议；Owner 决定执行。

### Sprint 4：最终净利润与实验结论

1. 修复结算利润重复扣费，并建立完整成本分类。
2. 退货/争议窗口、未决售后、结算完整性共同控制 finalization。
3. 支持 late adjustment 后 reopening。
4. 汇总实验级最终确认收入、最终净利润和净利率。
5. 只输出停止、换品、修正后再试、小幅加码四种结论。

验收：每一分钱能回到来源证据；未关闭窗口不得显示“最终”；最终利润可由 Ozon 结算单和现金凭证复算。

### Sprint 5：三轮后再决定自动化

连续完成至少三轮真实实验后，按实际耗时排序自动化：采集、费用同步、评分、Listing 文案或广告数据。没有真实重复劳动证据的能力继续冻结。

## 八、文档处置清单

### 必须修改为当前事实源

- `README.md`：核心定位改为 Owner 自用内部经营系统；当前唯一任务改为首轮 Ozon 实验。
- `AGENTS.md`、`CLAUDE.md`：加入 Owner 最新方向的最高优先级禁令；更新“已知问题”和当前验收目标。
- `docs/CURRENT_DIRECTION_AND_PRIORITIES.md`：用真实成交实验替换通用 Copilot/AgentOS 建设优先级。
- `docs/ROADMAP.md`：替换为 Sprint 0–5。
- `TODOS.md`：移除活跃的 Dual-Product Cathedral Phase 3，建立实验 P0 清单。
- `docs/SPEC.md`：唯一主要用户改为 Owner；加入实验、预算、停止、成交和最终利润定义。
- `docs/PROJECT_STATUS.md`：准确区分真实、估算、mock、stub；记录 Ozon 尚待账号级验证。
- `DESIGN.md`：从 B2B SaaS/merchants 改为单 Owner 决策工作台；优先呈现实验风险、证据和利润状态。
- `docs/INDEX.md`：把当前事实源置顶，冻结文档移入历史区。
- `docs/PRODUCT_VISION_AND_MVP.md`、`docs/explanation-business-loops.md`、`docs/howto-first-business-loop.md`：统一成一个真实 Ozon 实验闭环。

### 必须冻结并在文件顶部加醒目标记

- `docs/designs/dual-product-cathedral.md`
- `docs/LINGMIRROR_AGENT_COMMERCE_OS_BLUEPRINT.md`
- `docs/AI_NATIVE_DEVELOPMENT_PLAN.md`（如仍作为活跃路线引用）
- 所有 Outcome Proof、Evidence Warranty、跨客户聚合、结果收费、公共 API、自助注册、多租户商业化计划。

标记应写明：Owner 于 2026-07-11 改为单 Owner 自用；仅作历史决策记录，不得作为开发指令。

### 保留

- `docs/governance/*` 的 Owner-first、审批、审计、外部写安全、恢复与发布检查；
- Active Stack Policy、API/配置/部署/测试文档；
- Ozon 集成和结算对账操作指南；
- 商品、供应商、SKU、Listing、订单、售后、结算、利润的领域说明。

治理文档中的“长期 AgentOS 平台”语言可随后收敛，但安全契约不应因自用而弱化。

## 九、代码模块处置清单

### 重点复用和连接

- `domain/candidate`
- `domain/supplier`
- `domain/profit`（预估与逐单利润需修正）
- `domain/platformfee`
- `domain/logistics` / `shipping` / `tariff` / `exchangerate`
- `domain/listing`
- `domain/integrations` 中的 Ozon
- `domain/order`
- `domain/aftersales`
- `domain/settlement`
- Approval / ActionPolicy / Audit / RBAC / Command / EventBus 安全机制

### 必须新增或补齐

- 商品实验聚合根与 experiment_id 关联；
- 候选证据、硬淘汰和加权评分；
- 实验现金账本、承诺金额、可回收/不可回收损失；
- 版本化停止规则及触发记录；
- 流量与广告事实记录；
- 有效成交、最终有效成交和 finalization 状态机；
- 完整最终净利润分类及实验级汇总；
- Ozon 真实账号 canary 验收和 fixture/契约测试。

### 冻结开发但不必删除

- 非 Ozon 平台的新能力；
- MOA 与更多 Agent 编排；
- 外部客户、租户、订阅、计费、公开 API；
- Mock Owner dashboard 和纯展示性页面；
- 与三轮真实实验无关的自进化、自治和大型视觉工程。

## 十、Owner 现在应采用的方向声明

建议把下面这段作为所有当前事实源的统一摘要：

> 凌镜是 Owner 本人经营跨境电商的内部商品实验系统，不面向外部客户。当前唯一目标是在 Ozon 用不超过 3,000 元完成一轮真实商品实验：从 20 个线索筛出 3 个候选、批准 1 个商品、获得非关联真实买家订单，在退货与争议窗口关闭后核算最终净利润。任何不能直接提高证据质量、促成成交、控制损失或证明最终利润的开发暂停。真实发布、广告、采购、库存、订单、退款和资金动作必须由 Owner 批准并留痕。

## 十一、方向确定后的第一个工程任务

不是重做首页，也不是接更多平台。第一个工程任务应是：

**设计并实现最小商品实验聚合根，把 Candidate → Profit → Approval → Ozon Listing → Order → Aftersales → Settlement → Final Net Profit 用 experiment_id 串起来，并先支持人工录入所有不可自动获得的真实事实。**

这是把现有“模块仓库”变成 Owner 可实际经营的系统所需的最短路径。
