# LingMirror Historical Direction and Priorities

> Updated: 2026-07-11
> Status: `superseded / unconfirmed`
> Authority: 历史提案，不是当前产品方向或开发授权
> Current entry: [PRODUCT.md](PRODUCT.md) 与 [CURRENT.md](CURRENT.md)

> 本文保留当时提案的方向、预算和行动顺序用于追溯。Owner 尚未重新确认这些内容，任何 Agent 不得把正文中的“当前”“P0”或固定金额自动用于规划、开发和结果判断。

## Current Direction

凌镜当前是 **Owner 本人自用的真实付费需求发现与经营验证系统**，不是对外 SaaS，也不并行建设双产品。

完整产品边界、结果定义、资金纪律和解冻门槛以 [Owner 自用经营方向](SELF_USE_OPERATING_DIRECTION.md) 为准。

## Current North Star

```text
跨市场候选与反证
→ Owner 批准已选市场
→ 该市场的真实数据
→ 有来源的需求假设
→ 独立 AI 反证
→ 可实验案件
→ Owner 批准的最小实验
→ 至少 3 名非关联买家真实付款
→ 签收与售后窗口闭合
→ 退货/争议/费用结清
→ 正的最终贡献利润
→ 停止、换品、修正后再试或小幅加码
```

页面数、Agent 数、报告数、上架数、下单数和销售额都不能替代最终净利润。

## Immediate Priorities

### P0. Correct Known Truth Gaps

- 修复最终利润为零或负数时仍可能进入 `continue` 的裁决漏口；
- 付款、签收和售后闸门必须读取同一订单和售后对象的真实状态，不能只依赖文本证据；
- 明确核验买家非 Owner、非亲友、非供应商，且退货和争议观察期已经关闭；
- 现金金额与币种必须和同一订单的结算、最终利润保持可解释的一致关系；
- 前端不得用硬编码数字或演示任务展示“正在真实运行”。

### P0. Strategic Source Alignment

- 冻结双产品、外部 SaaS、多租户、计费、公共 API 和展示性扩张；
- 统一下单、有效成交、最终有效成交、预计利润和最终净利润定义；
- 所有当前事实源指向同一开发路线。

### P0. Market Selection Before Collection

- 先定义候选市场为“国家/地区 × 目标消费者 × 需求场景 × 销售渠道”，不以平台名代替市场；
- 用统一闸门比较需求、竞争、获客、履约、合规、收款、售后和最终利润可验证性；
- 每项数据采集必须绑定候选市场、明确决策、支持字段、反证字段和淘汰条件；
- Ozon、Shopee、Shopify 只是连接器；已有账号或代码不得获得优先经营地位。

### P0. Launch Experiment Aggregate

建立需求案件和稳定 `experiment_id`，串联假设、证据、反证、审批、流量、订单、支付、售后、结算和最终贡献利润。

### P0. AI Collection Pipeline

复用 ToolBridge、平台 API 和浏览器驱动生成有来源的需求假设；侦察、反证和数据现实必须是独立 run。公开热度只能进入 `lead`，不能跨级到付款。Owner 不负责日常抄数，只处理授权、冲突、异常和高风险审批。

### P0. Capital Ledger And Stop Rules

- 实验总预算默认 3,000 CNY；
- 不可回收损失默认 1,200 CNY；
- 成本必须标记为 actual、quoted、estimated 或 unknown；
- unknown 关键成本阻止生产批准；
- 停止条件采用确定性规则，Agent 只解释，不得修改事实或阈值。

### P0. Finalized Outcome And Profit

- 区分有效成交与最终有效成交；
- 退货、争议、结算和晚到调整共同控制 finalization；
- 修复现有利润口径和潜在重复扣费；
- 未完成观察窗口不得显示“最终净利润”。

### P1. Read-Only Platform Preflight

只对已进入候选短名单的市场验证相关平台账号能取得什么真实数据；无权限保持 unknown。只有一个“市场 × 渠道”组合同时通过需求、只读数据、收款、履约、合规和费用闸门后，才可另行设计 production 实验。

## Safety Rules That Remain

自用不降低安全要求。价格、库存、订单、采购、广告、退款、资金和外部平台写入仍需审批、审计、幂等、失败可见，并明确区分 read-only、dry-run、sandbox 和 production。

## Explicitly Frozen

- Dual-Product Cathedral；
- LingMirror Intelligence / Portfolio Launch OS 拆分；
- 外部客户、自助注册、订阅、计费、公共 API；
- Outcome Proof、Evidence Warranty、跨客户知识聚合；
- 未通过只读预检的平台扩张；
- 更多 Agent、MoA、自治等级和大型视觉工程。

## Acceptance Path

当前方向只有在下列证据齐全时才算完成第一阶段：

1. 每个案件能从原始证据形成假设，并经过独立反证和数据可得性审计；
2. 预算和停止条件在投入前冻结并可审计；
3. 具体实验平台、SKU 和目标人群来自证据与 Owner 批准，而非预设；
4. 至少 3 名非关联陌生买家在至少 2 个自然日付款并完成签收；
5. 退货和争议窗口关闭，平台费用与现金支出对账完成；
6. 系统给出可由凭证复算的正/负最终贡献利润与四选一结论。

## Current Reality Check

截至 2026-07-11：

- 实验案卷、证据等级、顺序闸门以及部分利润/现金终局校验已经实现并有自动测试；
- 候选市场比较、Owner 选定市场和独立研究 run 尚未形成完整产品事实链；
- 付款、签收和售后阶段尚未充分接入已有订单及售后业务对象；
- 没有仓库证据证明真实陌生买家成交、售后闭合、最终净利润或现金回收已经发生；
- 现有测试通过只能标记 `automated_verified`，不能标记 `external_observed`、`reconciled` 或 Business Verified。

完整证据与限制见 [2026-07-11 项目真相审计](research/project-truth-audit-2026-07-11.md)。
