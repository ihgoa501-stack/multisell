# TODOs

> Current direction: Owner 自用真实付费需求发现循环。
> Source of truth: `docs/SELF_USE_OPERATING_DIRECTION.md`。

## P0 — 需求案件与证据裁决

- ✅ 已新建候选市场 `DemandCase`、Evidence 和 Verdict，并提供独立反证入口、确定性裁决与六行 Owner 决策卡。
- 待补齐 DemandExperiment、不可变 Event、污染订单状态机和真实交易终局裁决。
- 用状态机禁止代理信号跨级；关联单、测试单和不可剥离异常流量进入 polluted。
- 用 `experiment_id` 串联流量、订单、支付、物流、售后、结算和最终贡献利润。
- 任何晚到退款、拒付或费用自动重开终局裁决。

## P0 — AI 侦察、反证和数据现实契约

- ✅ 已建立三个独立 AI run 契约：侦察只创建 lead，反证只追加 counterevidence，数据现实只记录字段现实；原始 payload 使用 SHA-256 不可变快照。
- ✅ 已提供带日期的公开资料研究导入与 Owner 候选市场页面；独立反证后本轮没有权限候选，所有案件保持 evidence_missing 或 reject。
- 每个事实保存来源、时间、地区、官方字段、原始 payload、解析版本和事实状态。
- 无来源数字、客户画像、销量、费用或利润自动拒绝；关键缺失保持 unknown。
- 每个字段保存来源 URL/API、采集时间、驱动、原始 payload、解析版本和置信状态。
- 不用 Agent 投票或置信分裁决；只输出线索、证据不足、被污染、已驳回或可实验。
- 采集失败自动重试、切换驱动并进入异常队列；日常数据不得要求 Owner 手工抄录。
- 保存需求、竞争、利润、物流/退货、合规和补货证据的来源、时间与原始值。
- 增加硬淘汰项和确定性加权评分。
- LLM 只能解释证据，不得生成无来源销量、费用或合规事实。

## P0 — 实验资金账本

- 覆盖样品、采购、包装、国内运输、跨境物流、广告、平台费、税费、关税、支付提现、汇兑、退款退货、销毁和售后补偿。
- 每项标记 actual、quoted、estimated 或 unknown，并保存币种、汇率时间和凭证。
- 默认总预算 3,000 CNY，不可回收损失停止线 1,200 CNY。
- unknown 关键成本或预算突破必须阻止新增投入并要求 Owner 重新批准。

## P0 — 停止规则

- 建立版本化规则与触发记录。
- 首轮基线覆盖：账号/收款/合规/物流失败；利润门槛不足；广告 300 CNY 无单；不可回收损失 1,200 CNY；前 5 个已发货订单中 2 个取消/拒收/退款/退货。
- Agent 只说明触发事实和建议；不得自动花钱、取消订单或扩大预算。

## P0 — 有效成交与最终净利润

- 增加非关联订单排除标记。
- 区分 ordered、paid、shipped、delivered、有效成交和最终有效成交。
- 用退货/争议窗口、未决售后和结算完整性控制 finalization。
- 支持晚到费用或退款后从 finalized 重新打开。
- 修复 settlement 利润潜在重复扣费并补齐完整成本分类。
- 只输出停止、换品、修正后再试、小幅加码四种实验结论。

## P1 — 平台只读数据预检

- 验证 Ozon、Shopee、Shopify 各自真实账号权限和可取得字段；不打印凭证。
- 区分 available、requires_owner_access、requires_listing、requires_transaction、unavailable 和 unknown。
- Shopify 无目标流量时只作为自店测量工具；Shopee 未明确国家站点时不合并数据。
- 只读预检失败不切换到猜测，保持 unknown 或淘汰该数据源。

## P1 — 首个自动研究—反证批次

- 最多生成 10 个真实来源假设，每条经过独立反证。
- 最多保留 3 个值得请求更多只读数据的案件；无存活案件时不强行推荐。
- 真实采购、发布和广告另开计划，并在动作时逐项取得 Owner 批准。

## Frozen Backlog

以下项目不进入当前开发队列：双产品、外部 SaaS、多租户、订阅计费、公共 API、外部 onboarding、Outcome Proof、Evidence Warranty、跨客户聚合、未经实证的平台扩张、更多内部 Agent/MoA/自治升级以及大型视觉重构。
