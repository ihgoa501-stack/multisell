# Owner 单人跨境经营闭环：最小系统与 AI 工作设计

> 日期：2026-07-12
> 研究范围：从候选市场比较到退货/争议后最终净利润与现金对账
> 产品边界：只供 Owner 本人使用；不是 SaaS、ERP 平台或外部软件需求验证
> 访问日期：本文所有网页来源均于 2026-07-12 访问
> 证据限制：本研究未连接 Owner 的真实平台账号、银行、广告、物流或税务数据，不能证明任何市场、商品、订单或利润已经成立。

## 1. 结论

凌镜下一阶段不需要建设“完整跨境电商 AgentOS”。最小可用系统只需可靠完成一件事：

> 把一次 Owner 批准的小额商品实验，从市场依据、预算和外部动作，一直追到订单、签收、售后、平台结算、银行到账及最终净利润，并把缺证、冲突和超预算集中交给 Owner 裁决。

真正的实操链不是官方教程式的“选品 → 上架 → 出单 → 发货”，而是：

```text
候选市场比较
→ 权限与成本预检
→ 商品机会与反证
→ 冻结实验预算/停止线
→ Owner 批准外部动作
→ 少量采购/发布/获客
→ 订单付款/履约/签收
→ 退款、退货、拒付、索赔观察
→ 平台逐笔费用进入结算批次
→ 平台回款匹配银行入账
→ 外部成本、税费和汇兑补齐
→ 订单利润、实验利润与现金三方对账
→ 停止 / 换品 / 修正后再试 / 小幅加码
```

`inferred`：系统的核心不是更多 Agent，而是 **不可变原始证据 + 少量事实对象 + 确定性状态机 + 异常队列 + Owner 审批**。AI 只能整理、匹配、估算和建议，不能把估算升级成真实利润，也不能代表 Owner 花钱或改变商品经营状态。

## 2. 研究观察：实际卖家为什么会算不清

### 2.1 平台回款不是订单收入

- `quoted`：Amazon 卖家社区的实操建议是，把平台结算摘要拆出销售、费用、退款等科目，再把结算净额与银行入账匹配；不能把银行收到的净回款直接记成营业收入。退款可能出现在后续结算期，平台还可能保留部分资金。[Amazon Seller Community：Settlement Report](https://sellercentral.amazon.co.uk/seller-forums/discussions/t/290a3a8c-bab9-404b-9d6b-b50d5bcb1fb6)
- `quoted`：卖家讨论中反复出现“订单/费用报表与最终银行净入账存在缺口”，以及保留金、跨期退款、费用分类难以审计的问题。[Amazon seller reconciliation discussion](https://www.reddit.com/r/Amazonsellercentral/comments/1uc9n9x/how_do_you_reconcile_seller_central_fee_reports/)；[Amazon/QBO settlement discussion](https://www.reddit.com/r/FulfillmentByAmazon/comments/1mig5v3/do_you_integrate_your_bank_with_quickbooksxero_or/)
- `quoted`：A2X 的真实工具工作流也是“拉取每个 settlement → 分类销售和费用 → 生成结算摘要 → 与银行中的平台回款匹配”，默认不会未经配置直接写入会计系统。[A2X 与 Xero 工作流](https://support.a2xaccounting.com/en/articles/959803-how-a2x-works-with-xero-accounting-software)

`inferred`：凌镜必须同时保存“订单应收”“平台结算”“银行现金”三层，不能只有订单利润表。

### 2.2 订单完成不等于利润最终成立

- `quoted`：跨境卖家复盘指出，退货带来的不只是退款，还包括逆向物流、库存滞留/错区、再次销售或销毁处理，以及费用晚于订单出现造成的对账混乱。[跨境退货实操讨论](https://www.reddit.com/r/ecommerce/comments/1pyj9kr/international_expansion_how_do_you_keep_shipping/)
- `quoted`：Amazon 的付款说明显示，销售额会先加到账户，再扣除费用和退款，并可能保留 reserve；结算状态完成后，银行到账仍可能滞后。[Amazon Seller Community：付款报告](https://sellercentral.amazon.com/seller-forums/discussions/t/2a0fc0bf-d770-41e1-bd79-9ee6c5d2d7fd)

`inferred`：签收只是履约事实。最终利润至少要等适用售后/争议窗口结束、相关结算行出现、外部成本凭证补齐、平台回款进入银行并完成匹配。

### 2.3 “利润估算”最容易漏掉跨境成本

- `quoted`：卖家社区的商品研究通常会检查历史价格/销量代理指标、卖家数量、供应价、ROI、利润和知识产权风险，并建议小批量或先用低库存方式验证，而不是按预测直接放大。[商品研究实操讨论](https://www.reddit.com/r/AmazonFBA/comments/1t5z30n/any_guidetips_for_product_research/)；[小批量测试讨论](https://www.reddit.com/r/AmazonFBA/comments/1kg0qxf/how_do_you_find_products_to_sell/)
- `quoted`：跨境实操资料提醒 landed cost（到岸总成本）还可能包含关税、增值税、保险、报关/清关费和承运商附加费，Incoterms（国际贸易术语）决定由谁承担部分费用。[跨境到岸成本计算](https://shopilery.com/how-to-calculate-landed-cost-for-cross-border-ecommerce/)
- `quoted`：币种不同会引入换汇费，准确的单笔利润难以事前确定，因为卡费、运费和汇率会变化。[Shopify Community：换汇与交易费](https://community.shopify.com/c/shopify-discussions/card-transaction-fees-and-currency-conversion-fees/m-p/2676748)

`inferred`：系统必须同时保存 `estimated` 预算与 `actual` 凭证金额，且预算缺项时只能显示“利润未知”，不能自动按 0 处理。

## 3. 最小系统边界

### 3.1 首轮必须有的 8 项能力

1. **候选市场案件**：明确国家/地区 × 消费者 × 需求场景 × 渠道，记录支持证据、反证、淘汰条件和关键未知。
2. **数据/权限预检**：验证平台账号权限、可取得的订单/费用/退款/结算字段、物流/税费来源；连接器存在不等于权限可用。
3. **商品机会与单位经济预算**：记录供应报价、采购、包装、头程/尾程、平台费、广告、税费、汇兑、退货处置等，并保留区间和来源状态。
4. **实验冻结与审批**：冻结 SKU、渠道、数量、现金上限、不可回收损失线、主要结果和停止条件；之后任何扩大都要新审批。
5. **外部动作与现金承诺账本**：采购、广告、发布、调价、退款等动作必须生成待审批命令；执行后保存外部回执和金额。
6. **订单—履约—售后链**：订单、付款、发货、签收、退款、退货、拒付、索赔与处置结果通过外部 ID 串联。
7. **结算与银行对账**：保留平台结算批次及逐行费用，将净回款匹配银行交易；显示 reserve、跨期和未匹配差额。
8. **最终利润裁决与异常队列**：只有凭证齐全且三方对账通过才标记 final；其余按原因排队，Owner 只处理少量例外。

### 3.2 当前明确不建

- 通用 ERP、总账系统、完整税务申报系统；首轮只保存实验所需事实和凭证。
- 多平台统一抽象后再接真实平台；先为 Owner 批准的一个实验做窄连接器。
- 自动找“爆品”、自动定战略、自治采购/投放/退款。
- 更多角色型 Agent、Agent 互评、置信分仪表盘。
- 预测性大屏；首轮只要事实卡、实验时间线、利润桥和异常队列。

## 4. 系统事实对象

每个对象必须保存：`source_system`、`source_object_id`、`observed_at`、`ingested_at`、`raw_snapshot_id`、`parser_version`、`truth_status`、`currency`（涉及金额时）和关联的 `experiment_id`。原始快照追加写，不静默覆盖。

| 对象 | 最少字段 | 关键约束 |
|---|---|---|
| `market_case` | 地区、消费者、场景、渠道、八维状态、淘汰条件 | 未经 Owner 批准不能进入实验 |
| `evidence_snapshot` | 来源、URL/API、时间、原始 payload 哈希、作用 support/counter/conflict | AI 摘要不能替代原始证据 |
| `product_hypothesis` | 商品、消费者用途、差异、反证、合规/IP 风险 | 风险未知不得发布 |
| `cost_quote` | 成本类型、金额/区间、币种、有效期、承担方、状态 | unknown 不得按 0 计 |
| `experiment` | SKU、渠道、数量、预算、止损线、终点、观察期 | 冻结后扩大须重新审批 |
| `approval` | 动作、参数、最大金额、有效期、批准人、外部回执 | 批准必须绑定具体参数 |
| `purchase_lot` | 供应商、数量、单价、付款、运费、批次 | 实际采购不能只引用报价 |
| `ad_spend_line` | 账户、活动、日期、花费、币种、税、归因字段 | 广告平台归因不是订单付款事实 |
| `order` / `order_line` | 平台订单、SKU、数量、买家资格状态、销售额、税、折扣 | 下单、付款、签收分开 |
| `shipment` | 承运商、跟踪号、运费、发货/签收时间、异常 | 平台状态与承运商证据冲突时排队 |
| `after_sale_case` | 退款/退货/拒付/索赔、金额、物流、处置、关闭时间 | 未关闭阻止最终利润 |
| `settlement_batch` | 平台、期间、净额、币种、reserve、deposit date | 保存期初/期末余额与完整性校验 |
| `settlement_line` | 类型、说明、金额、订单/SKU/调整 ID、posted date | 未识别费用不得丢弃或归“其他=0” |
| `external_cost` | 包装、头程、仓储、3PL、关税、税、保险、销毁等凭证 | 可按批次分摊，但保留算法版本 |
| `fx_conversion` | 原币、目标币、原额、到账额、汇率、换汇费、时间 | 不用当天中间价冒充实际成交汇率 |
| `bank_transaction` | 账户、日期、金额、币种、参考号、原始凭证 | 只读导入；不可由 AI 创建银行事实 |
| `profit_record` | 预计/最终、各成本桥、未决项、计算版本 | final 必须来自实际且已对账数据 |
| `reconciliation` | 两侧对象、预期差额、实际差额、状态、解释 | 自动匹配失败不能强制平账 |
| `exception` | 类型、影响对象、金额风险、截止时间、建议动作 | 所有失败进入同一 Owner 队列 |

## 5. 确定性状态机

### 5.1 市场与实验

```text
market_case:
draft → evidence_missing → comparable → owner_selected
                         ↘ rejected

experiment:
draft → cost_unknown → ready_for_approval → approved
     → running → fulfillment_open → after_sale_open
     → settlement_open → reconciliation_open → final
                                         ↘ stopped
```

硬门：

- `comparable`：八维均有来源与观察时间；关键项不能是 unknown/mock/inferred；至少有独立反证。
- `approved`：Owner 批准冻结参数；预算和不可回收损失线存在。
- `running`：只允许执行批准范围内动作。
- `after_sale_open`：签收后仍有退货/争议风险，预计利润不可升级为 final。
- `final`：所有订单售后已关闭或明确处置；结算完整；平台净额与银行现金对账；外部成本、税费、汇兑均为 actual/reconciled；实验最终利润已计算。

### 5.2 订单、结算和现金是三条独立状态

```text
order: created → paid → shipped → delivered → after_sale_closed
                                  ↘ returned/refunded/disputed

settlement: detected → lines_complete → allocated → platform_reconciled
                                          ↘ mismatch

cash: expected → bank_seen → fx_verified → cash_reconciled
                            ↘ short/late/unknown
```

禁止跨级：平台显示“已付款”不等于已签收；平台显示“已结算”不等于银行到账；银行到账不等于收入；预计利润为正不等于最终利润为正。

## 6. AI 在每一步怎样工作

| 步骤 | 输入与工具 | AI 输出 | 证据/失败处理 | 刷新 | 权限 |
|---|---|---|---|---|---|
| 候选市场侦察 | 公开搜索、平台公开页、行业/社区资料 | 标准化候选、支持点、待核问题 | 保存网页快照；来源冲突标 conflict；访问失败重试后排队 | 周度或决策前 | 只读 |
| 独立反证 | 不读取侦察结论的独立 run；同类独立来源 | 淘汰理由、最坏情景、需验证字段 | 无可核来源只能 `inferred` | 每次候选变更 | 建议 |
| 数据现实审计 | 账号权限、API/报表样本、字段字典 | 哪些事实可自动取得、缺什么 | 授权/验证码请求 Owner；空成功视为异常 | 候选入围时 | 只读；授权由 Owner |
| 商品/供应链研究 | 搜索趋势代理、竞品、评论、供应报价、合规资料 | 假设、差异、成本区间、反证 | 报价保留有效期；合规未知阻止发布 | 周度/报价过期 | 建议 |
| 单位经济预算 | 成本报价、平台费率、物流方案、税/汇率假设 | 悲观/基准/乐观利润与缺项 | unknown 不按 0；显示利润对缺项敏感性 | 成本变化时 | 计算/建议 |
| 实验编排 | Owner 冻结参数、预算和停止线 | 待审批动作清单、计划时间线 | 参数超范围自动阻止并生成新审批 | 事件触发 | 不能自行批准 |
| 采购/发布/广告 | 已批准命令、平台接口或浏览器 | 执行预览、执行后回执 | 外部写前二次校验；失败不自动扩大重试金额 | 按批准动作 | 必须 Owner 审批 |
| 订单监控 | 平台 API/报表、webhook | 新订单、付款、取消、异常 | 去重；状态回退/冲突进入队列 | 5–30 分钟或 webhook | 只读 |
| 履约监控 | 平台、承运商、3PL | 发货/签收/延误、预计额外成本 | 承运商与平台冲突不自动择一 | 2–6 小时 | 只读；补发需审批 |
| 广告成本 | 广告报表、平台账单 | 实验/SKU/日花费、超预算预警 | 归因销售仅作分析，花费以账单/结算为准 | 每日；高风险可小时 | 只读；启停/调价需审批 |
| 售后监控 | 退款、退货、拒付、索赔、退货物流 | 案件状态、损失区间、建议处置 | 未关闭一律阻止 final | 每日 | 只读；退款/补偿需审批 |
| 结算解析 | 平台 settlement 原文件/API | 批次、逐行费用、订单/SKU 分配、reserve | 未知费用保留原文并排队；批次校验不等则 mismatch | 每次新结算 | 只读 |
| 银行/现金对账 | 只读银行流水/导入、回款参考号、结算净额 | 候选匹配、差额、到账延迟 | 模糊匹配只建议；不得自动制造调账行 | 每日/银行导入后 | 只读 |
| 汇兑 | 原币结算、支付服务商/银行换汇记录 | 实际汇率、换汇费、差额 | 无真实换汇记录时只 estimated | 每笔换汇 | 只读 |
| 税费/关税 | 税票、清关单、平台代扣、专业规则来源 | 实验税费归集、缺票提醒 | AI 不作最终税务裁决；复杂项交专业人员 | 每月/单据出现 | 建议 |
| 最终利润 | 已对账对象和算法版本 | 每订单与实验利润桥、未决项、终局建议 | 只有确定性门通过才能 final | 事件触发 | 计算；终局 Owner 确认 |

### AI 输出的统一结构

每次 AI 输出只允许包含：

```text
decision_object_id
claim
truth_status: actual | quoted | estimated | inferred | unknown
source_snapshot_ids[]
observed_at
assumptions[]
counter_evidence[]
missing_fields[]
recommended_action
owner_approval_required
expires_at
```

不得用“置信度 92%”替代事实状态。多个 Agent 一致仍然只是多个推断；不能升级为 `actual`。

## 7. 最终利润与现金对账规则

### 7.1 利润桥

```text
最终净利润（Owner 报告币种）
= 商品销售收入 + 买家承担运费 + 平台补偿/赔付
- 折扣/优惠
- 退款、拒付、索赔和售后补偿
- 商品采购与样品摊销
- 包装、质检、国内运输
- 头程、跨境运输、保险、清关/报关
- 关税、进口税、销售税/VAT/GST（按实际承担口径）
- 平台佣金、履约、仓储、移除、长期仓储及其他费用
- 广告及广告税费
- 尾程、逆向物流、重新入库、销毁和不可售损失
- 支付、提现、银行及换汇费用
- 其他有凭证且归属于实验的成本
```

平台 settlement line 是平台侧事实；供应商发票、物流账单、广告账单、税票/清关单是外部成本事实；银行流水是现金事实。三者不可互相替代。

### 7.2 三层核对

1. **平台批次守恒**：期初余额 + 批次全部贷项 − 全部借项 − reserve/递延变化 = 应回款/期末余额。
2. **平台到银行**：应回款按 settlement/payout ID、币种、金额和时间匹配银行入账；换汇时通过支付服务商/银行的转换记录串联原币与到账币。
3. **订单到实验**：每条订单相关收入/退款/费用分配到订单；不可分配费用按明确规则分配到实验，保存算法版本和未分配余额。

Stripe 的官方对账模型也明确区分 payout 批次、组成它的 balance transactions 和未结余额；自动 payout 可按 payout ID 取得逐项交易，而手动 payout 需要自行完成匹配。[Stripe Payout Reconciliation](https://docs.stripe.com/payouts/reconciliation?locale=en-GB) 这支持上述结构，但不代表 Owner 当前使用 Stripe。

Amazon 官方 SP-API 的 Flat File V2 settlement report 提供 settlement、deposit、currency、transaction、order、SKU、amount type/description/amount 等字段，且旧格式将在 2026-11-11 移除；连接器应保留未知的新费用类型，而非硬编码有限枚举。[Amazon Settlement Reports](https://developer-docs.amazon.com/sp-api/lang-zh/docs/report-type-values-settlement)；[Amazon 2026 格式迁移公告](https://developer-docs.amazon.com/sp-api/changelog/update-removal-of-xml-settlement-report-and-flat-file-settlement-report-date-changed-to-november-11-2026)

### 7.3 final 的必要条件

- 所有关联订单均达到 `after_sale_closed`，或损失已实际确认并入账；
- 新结算批次没有该实验的未分配金额；
- reserve、递延、退款和调整已解释；
- 银行回款与平台 payout 已匹配，汇兑费用与实际汇率已确认；
- 采购、物流、广告、退货、税费等外部单据已齐，或明确标记“不适用”并有依据；
- `unallocated_amount = 0`，`unreconciled_cash = 0`，关键 unknown = 0；
- 计算版本与原始快照可重放；
- Owner 确认终局。

## 8. 异常队列：Owner 每天只处理这里

按风险排序，不按模块分散：

| 优先级 | 异常 | 自动处理 | Owner 要做什么 |
|---|---|---|---|
| P0 | 超止损、未授权外部写、疑似账号/资金风险 | 立即冻结相关动作 | 决定停止、撤销或人工调查 |
| P1 | 结算与银行不符、退款/拒付、未知大额费用、税/合规阻断 | 重试采集、提出候选匹配，禁止强制平账 | 查看原始凭证并裁决/开平台 case |
| P2 | 物流延误、广告接近预算、报价过期、成本缺票 | 刷新来源、计算影响区间 | 批准处置或补证 |
| P3 | 小额未分配、字段映射未知、数据延迟 | 批量重试和聚类 | 周度集中处理 |

每张异常卡只显示：发生了什么、影响哪个实验、最大现金风险、原始证据、AI 已做什么、建议动作、最晚决定时间、批准按钮。不得隐藏在聊天记录里。

## 9. 最小 UI：5 个页面足够

1. **今日 Owner 队列**：审批、异常、即将触发停止线、待补证；默认首页。
2. **候选市场比较**：同口径八维表、支持/反证、关键 unknown、淘汰条件；只能 Owner 选择。
3. **实验案卷**：冻结参数、预算、时间线、订单/物流/售后状态和所有外部动作。
4. **利润与现金桥**：预计 vs 最终，各费用来源、平台 settlement → 银行到账、未分配/未对账差额。
5. **证据抽屉**：点击任何数字即可看到原始快照、采集时间、来源、解析版本和历史变更。

不需要单独的“Agent 驾驶舱”。AI 状态只在具体案件中显示为：最后刷新、失败原因、下一次重试和是否需要 Owner。

## 10. 首个实现切片（推荐）

不要同时接多个市场和平台。待 Owner 选定市场后，用一个真实实验完成以下窄切片：

1. 导入/抓取一个平台的订单、退款、结算 V2 原文件以及一份银行流水；
2. 建立 `experiment → order → settlement_line → payout → bank_transaction` 确定性关联；
3. 手工上传供应商、物流、广告、税/清关凭证作为采集失败兜底；
4. 输出可重放的利润桥和未决项；
5. 把所有无法匹配、未知费用和缺证放到一个异常队列；
6. 以一笔真实结算完整对上银行到账作为工程验收，以售后闭合后正的最终净利润作为经营结果（结果可能为负，负结果不是工程失败）。

### 验收标准

- 同一原始文件重复导入不重复记账；
- 任一利润数字可追溯到原始外部凭证；
- 未知费用不会丢失或默认为 0；
- 平台结算与银行不符时系统阻止 final；
- AI 关闭、失败或给错建议时，确定性状态机仍不跨级；
- 采购、广告、发布、退款和资金动作均无自动执行路径；
- Owner 能在一个页面看清“现在缺什么、最多可能亏多少、下一步批准什么”。

## 11. 已知、推断与未知

### `actual`

- 本仓库当前政策把凌镜定义为 Owner 单人自用系统，并要求经营实验以最终净利润与现金对账结束。
- 现有审计指出仓库已有实验、利润和现金关联的部分实现，但未连接真实外部经营数据。

### `quoted`

- 平台结算由销售、退款、费用、调整、reserve/递延等构成，银行只收到净额；实操工具采用 settlement 摘要匹配银行回款。
- 跨境退货、物流、税费和汇兑会使订单表面毛利与最终利润显著不同。
- Amazon/Stripe 官方接口能提供逐结算批次或逐 payout 的组成交易，用于核实系统可取得的字段。

### `inferred`

- 对 Owner 单人系统，异常优先、证据可追溯的窄工作流比通用 ERP 或更多 Agent 更可控。
- 先打通一个平台的一笔真实结算，比先抽象所有平台接口更能暴露真实缺口。

### `estimated`

- 刷新频率、异常优先级和 5 页 UI 是本研究提出的首轮设计估算，需由真实平台限制和 Owner 使用体验校正。

### `unknown`

- 首轮选哪个国家、消费者、渠道、平台、品类和 SKU；
- Owner 当前真实平台账号权限、报表格式、银行导出能力和税务责任；
- 现有代码对该具体平台真实数据的解析完整度；
- 第一轮实验会盈利还是亏损。

这些未知应通过候选市场比较、数据权限预检和一笔小额真实实验解决，不能靠增加 Agent 数量解决。
