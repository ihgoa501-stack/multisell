# 小Q Owner 协作层审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 7 单元
> 工程状态：`implemented / automated_verified`

## Active 能力边界

- 需求案件、决策卡、trace-only 经营事实核验案卷、1688 受控内部草稿。
- 以 `Owner + exact order_id` 为主体读取不可变订单事实、库存 ledger、外部承运商事件、售后请求与终局回执、平台结算、最终利润和可归属现金对账。
- 读取经营决定案卷，并可保存绑定冻结 manifest、固定为 `inferred`、带幂等键的 AI 建议。

## 明确禁止

- 旧 `business_closure + experiment_id` 与 `order.fulfillment.read` 已退役并失败关闭。
- 小Q不能形成 Owner 决定、执行 Command、采购、发布、退款或绕过审批。
- 不向模型提供买家 PII，以及平台、承运商、结算和银行原始载荷。
- 多订单结算批次的现金不能归属单订单，只能返回批次级 blocker。
- `businessfeedback` 尚无小Q专用脱敏只读 Capability，保持 `deferred`。

## Owner 交互

- `/xiaoq` 支持 `operating_facts(order_id)` 与 `business_decision(decision_case_id)`。
- 页面分开显示事实证据、unknown、blocker 和 inferred 建议，并深链到订单与经营决定权威页面。
- `/business-decisions/[id]` 只允许 Owner 保存拒绝、暂停、补证等非执行决定；不提供 selected 执行捷径。

## 自动验证

- 后端全量测试：3357 项通过，122 个 package；vet/build 通过。
- 小Q与经营决定前端聚焦：28 项通过；定向 ESLint 无错误。
- 路由安全：455 个 mutation 全部显式分类；文档引用 242 个、缺失 0。
- Next.js 生产构建验证见本次审计后的最终验证记录。

## 证据边界

- 真实订单、现金或经营决定数据仍可能为空；这里只证明协作能力和安全边界已实现。
- 模型回答默认 `inferred`；真实模型失败不会回退成可信结论，stub 回答固定为 `mock` 且不能保存建议。
