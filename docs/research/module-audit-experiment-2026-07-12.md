# 经营实验模块独立审计

> **状态：部分结论已被纠偏。** 本文保留为当时的工程审计记录，但“经营实验闭环”“真实闭环”等表述自 2026-07-12 Owner 裁决后均为 `superseded`。现有模块只能证明经营事实追踪、闸门和对账能力，不能证明因果实验或反馈闭环。以 [经营闭环模型纠偏](project-truth-audit-2026-07-12-business-loop-correction.md) 为准。

> 审计日期：2026-07-12
> 范围：`backend-go/internal/domain/experiment/`、`/api/v1/experiments`、前端 `/experiments`，以及它直接读取的订单、售后、结算、利润和现金事实
> 审计基线：工作区当前源码（HEAD `8126d909b513e348eb1c19e80a26da92f63c507f`，工作区存在其他未提交改动）
> 任务类型：只读调研；除本报告外未修改代码或业务数据
> 验证：`cd backend-go && go test ./internal/domain/experiment`，2026-07-12 本地通过

## 1. 结论

经营实验模块的正确定位不是“实验管理页面”，而是 **Owner 自营商品实验的裁决案卷**：它必须阻止需求线索、内部记录、估算或 AI 推断跨级成为有效成交、最终有效成交、正的最终净利润或现金已回收。

当前模块已经具备可用的工程骨架：Owner 隔离、十阶段状态链、证据真实性、支持/反证/冲突、逐阶段闸门、经营对象关联，以及结算—订单利润—现金的后半段校验。聚焦后端测试在本次审计中通过，因此这些属于 `implemented / automated_verified`，不是 `external_observed`。

当前仍不能称为“好的完整经营实验闭环”。最关键的原因是：

1. 实验可以直接创建，未强制来自 Owner 已批准且 `experiment_ready` 的候选市场，也没有冻结 SKU、预算、主要终点和停止条件。
2. 订单付款、签收和售后闸门仍可仅靠 Owner 核验后的文本证据通过，没有确定性读取同一订单的付款、签收、非关联买家资格、退款/退货/拒付和观察期。
3. 利润闸门能核验可信结算和同订单最终利润记录，但不要求利润大于零；`continue` 也不要求 `final_profit_amount > 0`。
4. 现金记录要求同订单和同结算，但金额、币种尚未与结算及最终利润做一致性校验。
5. 前端能录入和推进案卷，但核心页面没有组件或浏览器 E2E 证据；页面可见不等于真实闭环可操作。

所以当前裁决为：**可信的后半段工程基础，尚未闭合真实经营裁决链**。

## 2. 模块定义

### 2.1 应该是什么

经营实验模块应以一个 `experiment_id` 串联：

```text
Owner 批准的已选市场
→ 冻结的商品实验方案与停止条件
→ 真实订单付款
→ 发货与签收
→ 售后/争议观察期关闭
→ 同一订单可信结算与完整最终成本
→ 正的最终贡献利润
→ 同一订单与结算的现金到账及一致性核验
→ stop / switch / adjust / 小幅 continue
```

它只裁决 Owner 的商品经营实验，不验证凌镜的软件需求，也不服务外部软件用户。这一边界来自 `docs/SELF_USE_OPERATING_DIRECTION.md:7-18,67-90` 和 `docs/research/project-truth-audit-2026-07-12.md:8-31`。

### 2.2 当前代码实际定义

- 案件有十个阶段：`opportunity → product → supply → channel → order → fulfillment → aftersales → profit → cash → decision`；真实性为 `actual / quoted / estimated / unknown / mock / inferred`，闸门结果为 `pass / conditional / return / reject / expired`（`backend-go/internal/domain/experiment/model.go:5-38`）。
- 案件保存利润和现金两个独立状态及金额（`backend-go/internal/domain/experiment/model.go:40-59`）。
- 证据区分 `support / counter / conflict`，并保存来源、观察时间、失效时间和 Owner 核验身份（`backend-go/internal/domain/experiment/model.go:77-92`）。
- 经营对象通过 `(experiment_id, object_type, object_id)` 关联，包括订单、售后、结算、利润记录和现金交易（`backend-go/internal/domain/experiment/service.go:27-34`；`backend-go/internal/domain/experiment/model.go:94-102`）。
- API 注册列表、创建、详情、更新、证据录入/核验、对象关联、闸门评估和 Owner 摘要（`backend-go/internal/domain/experiment/routes.go:9-21`）。这些路由位于 JWT 保护组，并经过写路由分类中间件（`backend-go/internal/httpx/router.go:591-599,860`）。
- 前端列表能创建并进入案卷（`frontend-next/src/app/(main)/experiments/page.tsx:16-69`）；详情页能录入证据、核验、关联对象、评估闸门、逐阶段推进和记录最终决定（`frontend-next/src/app/(main)/experiments/[experimentId]/page.tsx:28-79,99-180`）。

## 3. “好”与“不好”的可验证标准

| 范围 | 好的定义（必须能验证） | 不好的定义（应阻断） |
|---|---|---|
| 起点 | 实验唯一关联 Owner 已批准且 `experiment_ready` 的候选市场；市场组合、批准人和批准时间可追溯 | 输入一个名称就创建“真实案件”；候选市场、消费者或渠道未知 |
| 实验方案 | SKU/规格、预算上限、不可回收损失停止线、流量方式、主要终点、观察期和停止条件在外部动作前冻结并留哈希/版本 | 实验边做边改；成功标准事后定义；没有预算和停止条件 |
| 证据 | 每条证据有作用、真实性、来源、观察时间、适用对象；`actual` 只能由 Owner 对真实来源单独核验 | AI 文本、模拟、缺来源记录或内部字段被包装成真实事件 |
| 闸门 | 每个阶段只能读取当前实验、同一经营对象的确定性事实；失败时明确 `blocked / unknown` | 只要勾选一条文字证据即可通过付款、签收或售后闸门 |
| 有效成交 | 同一订单可核验：买家非 Owner/亲友/供应商，已付款、已发货、已签收，且外部来源与时间可追溯 | 只有下单、内部 `paid_at`、自购/关联买家或未签收也算成交 |
| 最终有效成交 | 已过适用退货/争议窗口，且无未决退款、退货、拒付或争议；观察期规则有来源 | “没有售后记录”就视为售后关闭；忽略平台争议窗口 |
| 最终利润 | 同一订单的可信平台结算全部对账；成本项完整；利润状态 final；币种明确；`profit > 0` 才允许小幅继续 | 预计利润、缺成本、手工未对账结算、零/负利润进入 continue |
| 现金 | 银行/现金账户真实交易同时关联同一订单与结算；金额和币种按明确口径与结算/最终利润一致 | 有任意一笔收入交易就标记回收；金额或币种不一致仍通过 |
| 终局 | `stop / switch / adjust / continue` 的条件确定、理由留痕；一次结果只裁决本次实验 | 一次成交推断可重复性或规模化；工程测试被当成经营成功 |
| 安全 | Owner 隔离；写操作有认证、审计分类；采购、发布、广告、退款、资金动作另行审批 | 实验闸门直接触发外部写；跨 Owner 读取；绕过审批和审计 |
| 可操作性 | 后端聚焦测试 + 前端组件测试 + 浏览器 E2E + 一次隔离环境人工演练 + 真实外部证据验收 | 仅模块存在、单元测试通过或页面能打开就宣称生产可用 |

## 4. 当前证据状态

| 声明 | 当前等级 | 证据 |
|---|---|---|
| 模型、迁移、API 和前端页面存在 | `implemented` | `backend-go/migrations/000076_create_experiment_core.up.sql:1-6`；`backend-go/internal/domain/experiment/routes.go:9-21`；前端两个页面 |
| 新案件强制从 opportunity 开始，并清空客户端注入的终局金钱状态 | `automated_verified` | `backend-go/internal/domain/experiment/service.go:59-85`；`service_test.go:130-140` |
| 阶段不能向前跳级，推进前要求当前阶段 pass | `automated_verified` | `backend-go/internal/domain/experiment/service.go:87-114`；`service_test.go:235-260` |
| 普通录入不能直接写 actual；Owner 核验要求来源和观察时间 | `automated_verified` | `backend-go/internal/domain/experiment/service.go:190-228`；`service_test.go:185-210` |
| mock/inferred/unknown、过期证据不能通过 pass；指定高风险阶段必须 actual | `automated_verified` | `backend-go/internal/domain/experiment/service.go:250-280`；`service_test.go:66-85,142-163` |
| opportunity pass 同时要求支持和反证 | `automated_verified` | `backend-go/internal/domain/experiment/service.go:261-280`；`service_test.go:235-260` |
| 最终利润要求可信来源结算、全部对账、同订单 final 利润记录且无 missing costs | `automated_verified`（测试环境） | `backend-go/internal/domain/experiment/service.go:333-384`；`closure_validation.go:10-22`；`service_test.go:262-365` |
| 现金要求正金额、银行/现金账户、收入交易、同订单和同结算 | `automated_verified`（测试环境） | `backend-go/internal/domain/experiment/service.go:386-415`；`service_test.go:306-330` |
| Owner 只读闭环视图保持内部订单真实性为 unknown、脱敏 PII，并阻断混合结算 | `implemented / automated_verified`（当前工作区新增源码） | `backend-go/internal/domain/experiment/owner_closure.go:15-29,151-217,299-345`；`owner_closure_test.go:14-98` |
| 聚焦后端包当前通过 | `automated_verified` | 本次运行 `go test ./internal/domain/experiment`，退出码 0 |
| `/experiments` 核心页面经过组件或浏览器 E2E | `unknown` | 仅发现展示语义单测 `frontend-next/src/lib/__tests__/experiment-display.test.ts:1-23`，未发现两个页面的组件/E2E 测试 |
| 真实非关联买家、真实付款签收、售后关闭、最终利润和现金到账发生 | `unknown` | 本次未连接生产数据库、平台账号、结算文件或银行凭证；不得从测试推断 |

## 5. 事实链缺口

### 5.1 闸门与实验起点

**缺口 A：未强制来自已选市场。** 创建接口只要求 Owner、名称和 `opportunity` 阶段（`backend-go/internal/domain/experiment/service.go:47-85`），前端也只提交名称和固定阶段（`frontend-next/src/app/(main)/experiments/page.tsx:20-33,64-68`）。没有要求关联 `demand_case`、其状态为 `experiment_ready` 或 Owner 批准记录。

**缺口 B：实验协议没有冻结。** `ExperimentCase` 没有 SKU/规格、预算、损失停止线、流量方式、主要终点、观察期、停止条件或审批快照字段（`backend-go/internal/domain/experiment/model.go:40-59`）。因此当前模块能管理阶段，不能证明“Owner 批准的最小真实实验”已经被定义且未被事后修改。

**缺口 C：闸门 pass 仍以通用证据为主。** 除 profit/cash 外，`EvaluateGate` 只检查证据枚举、真实性和 Owner 核验，不读取对应领域对象（`backend-go/internal/domain/experiment/service.go:239-309`）。`actual` 表示 Owner 核验过来源和时间，不等于付款、签收或售后状态已由系统确定性验证。

**缺口 D：可任意向后改阶段。** `Update` 只限制向前跳级；`nextIndex <= currentIndex` 没有回退条件、回退原因或旧闸门失效规则（`backend-go/internal/domain/experiment/service.go:95-114`）。前端不会主动这样做，但 API 合约允许，可能造成旧 pass 与新证据状态混杂。

### 5.2 订单与有效成交

订单闸门 `paid_order` 和履约闸门 `delivered` 被列为需要 actual 的阶段，但没有读取链接订单（`backend-go/internal/domain/experiment/service.go:33-40,250-280`）。订单领域虽然有 `pay_amount / paid_at / shipped_at / delivered_at / cancelled_at`（`backend-go/internal/domain/order/model.go:8-33`），实验闸门没有使用这些字段。

当前订单模型也没有确定性的“非 Owner、非亲友、非供应商”资格字段。因此即使某条付款证据被 Owner 核验为 actual，也不能由系统证明它是有效成交。新增的只读闭环视图更诚实：它把订单内部记录固定为 `unknown`，并明确提示缺外部来源（`backend-go/internal/domain/experiment/owner_closure.go:174-186`）；但这套读取目前没有接入订单/履约闸门。

### 5.3 售后与最终有效成交

`aftersales_closed` 仍只靠 actual 文本证据。只读闭环逻辑仅统计 `after_sales_order` 数量，并无论数量多少都把观察窗口标为 unknown，明确写着“无记录不代表观察期已关闭”（`backend-go/internal/domain/experiment/owner_closure.go:241-248`）。

它还没有统一核对：适用平台/地区的退货和争议截止时间、未决售后状态、`dispute_case`、退款、拒付以及结算后调整。因此当前系统不能确定性证明“最终有效成交”。

### 5.4 最终利润

已经做得较好的部分是：profit pass 会读取唯一链接中的最新结算、利润记录和订单，要求结算来源为 `platform_import / api_sync`、状态已 reconciled/closed、所有结算项匹配且至少一项关联实验订单，并要求利润记录属于同订单、状态 final、`missing_costs` 为空（`backend-go/internal/domain/experiment/service.go:321-384`）。

但仍有四个缺口：

1. `validateProfitClosure` 用“最新一条链接”而不是要求每种事实唯一；存在多个结算/订单/利润链接时可静默选择最后一条（`backend-go/internal/domain/experiment/service.go:321-330`）。新增只读视图会拒绝歧义，但闸门路径尚未复用它（`backend-go/internal/domain/experiment/owner_closure.go:220-238`）。
2. `missing_costs == ""` 依赖利润模块上游正确填充，没有在实验闸门逐项证明采购、样品摊销、包装、国内/跨境物流、平台费、广告、折扣、税关、提现、汇兑、退货销毁和补偿都已结清。
3. profit pass 不要求 `Profit > 0`（`backend-go/internal/domain/experiment/service.go:372-384`）。
4. `continue` 只要求利润状态 final 和现金 recovered，不要求正利润（`backend-go/internal/domain/experiment/service.go:47-57`）。因此零利润或负利润仍可能被记录为 continue；这是当前最严重的终局逻辑缺口。

### 5.5 现金回收

现金门禁要求交易为正金额 revenue、来自 bank/cash 账户、有日期，并同时关联同一订单与结算（`backend-go/internal/domain/experiment/service.go:386-415`）。利润状态和现金状态也被正确分开保存。

但是它没有比较现金交易币种与结算币种，也没有定义现金金额应与平台结算净额、收入还是最终利润按什么口径一致。新增只读视图明确把这一点保留为 unknown（`backend-go/internal/domain/experiment/owner_closure.go:315-344`）。所以当前只能证明“存在一笔符合形状的关联收款记录”，不能证明现金与结算和最终利润已经 `reconciled`。

### 5.6 前端可用性与误导风险

前端的优点是明确展示真实性标签、反证、闸门阻塞、利润与现金分离，并提示闸门不会触发外部动作（`frontend-next/src/app/(main)/experiments/[experimentId]/page.tsx:93-123,133-180`）。

风险包括：

- 列表按钮写“创建真实案件”，但创建时尚未验证已选市场和冻结实验方案，措辞证据等级过高（`frontend-next/src/app/(main)/experiments/page.tsx:39-47`）。
- `truthMeta` 把 quoted 和 estimated 标记为 `trustedForHighRisk: true`，而后端在 order/fulfillment/aftersales/profit/cash 阶段只接受 actual；前后端语义不完全一致（`frontend-next/src/lib/experiment-display.ts:9-16`；`backend-go/internal/domain/experiment/service.go:33,272-274`）。
- 详情页允许 Owner 手工输入任意对象类型和编号，没有对象搜索、存在性预检或唯一性提示（详情页 `178-180`）；这增加错链风险。
- 页面没有展示订单—售后—结算—利润—现金的只读闭环核验结果，也未显示“内部订单事实仍为 unknown”。
- 目前只有展示映射单测，未发现核心页面交互测试或真实浏览器 E2E。

## 6. 最小验证方法

建议按“小而完整、无外部写”的顺序验证，不先扩建新功能：

### 6.1 工程回归（可立即自动执行）

1. 后端聚焦：`cd backend-go && go test ./internal/domain/experiment`。
2. 增加失败用例后再验：负利润不能 continue；多个同类链接不能关闭利润/现金；订单/履约/售后不能只靠文本证据 pass；现金币种或金额不一致不能标记 reconciled。
3. 前端最小组件测试：创建表单、证据 actual 核验、闸门阻断、负利润终局、错链提示。
4. 最小浏览器 E2E：从一个 `experiment_ready` 候选进入实验，依次看到 blocked/unknown，不使用 mock 绕过。

### 6.2 隔离环境人工演练（不连接真实平台、不产生外部写）

准备一组可审计的隔离数据：一个已批准候选市场、一个实验、一个订单、一个售后记录、一个结算及明细、一个利润记录和一个银行交易。逐项验证：

1. 缺任一事实时对应闸门保持 blocked/unknown。
2. 错订单、错结算、手工未对账结算、混合结算、缺成本、未决售后、负利润、现金错币种均失败关闭。
3. 页面只显示脱敏信息，不暴露收件人、电话、地址、账户名、余额或原始结算 payload。
4. 任何闸门操作都不采购、不发布、不退款、不移动资金。

### 6.3 首次真实实验验收（需要 Owner 另行批准外部动作）

只选一个 Owner 已批准的最小实验。每个外部事实保留平台/银行原始凭证和观察时间，由系统自动采集优先，人工录入仅作异常兜底。直到退货/争议窗口关闭、结算全部对账、完整最终利润为正、现金一致性完成前，结论保持 unknown/blocked。一次成功只证明该次闭环发生，不证明可重复或可规模化。

## 7. 不扩大范围的建议

推荐只补齐当前一条链，优先级如下：

1. **P0：封死错误终局。** `continue` 必须要求正的最终贡献利润；利润/现金门禁要求同类对象唯一；定义并校验现金金额和币种口径。
2. **P0：把订单、签收、售后接到真实领域事实。** 闸门读取同一订单及售后/争议状态；缺外部来源时保持 unknown；没有售后记录绝不等于观察期关闭。
3. **P1：绑定实验起点。** 只允许从 Owner 已批准的 `experiment_ready` 候选市场创建，并冻结最小实验方案、预算和停止条件。不要建设通用实验平台。
4. **P1：让页面展示确定性闭环视图。** 复用现有 `ReadOwnerBusinessClosure` 的脱敏、unknown 和 blocker 语义，避免再造一套判断。
5. **P1：补最小组件测试和一条浏览器 E2E。** 验证 blocked/unknown 不能被 UI 绕过。

本轮不要新增国家、平台、类目、Agent、MoA、自治能力、仪表盘、大型视觉重构、SaaS、多租户、订阅、公共 API 或外部 onboarding。现有 Ozon/Shopee/Shopify 连接器也不应进入实验，除非对应市场已先通过候选市场闸门。

## 8. 仍然未知

- 当前生产环境是否部署了本工作区中的最新闭环读取代码。
- 是否存在 Owner 已批准的已选市场和冻结实验方案。
- 真实平台账号能否稳定提供付款、签收、退货、争议和结算来源。
- 订单买家是否可被可靠判定为非关联买家。
- 各候选渠道适用的退货/争议观察期及拒付数据能否获得。
- 上游利润记录是否逐项覆盖 Owner 定义的全部最终成本。
- 银行/现金交易与平台结算之间的正确金额、汇率和币种对账口径。
- `/experiments` 在正式域名、真实 Owner 登录和生产数据下的浏览器可用性。

在这些事实被逐项验证前，模块状态应保持 `implemented / automated_verified`，真实经营闭环保持 `unknown`。
