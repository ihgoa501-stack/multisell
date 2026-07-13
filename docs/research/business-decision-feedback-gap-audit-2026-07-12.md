# Owner 经营决策与反馈纵向单元审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 6 单元
> 工程状态：`implemented / automated_verified`；因果关系仍为 `not_established`

## 权威链

1. **事实快照**：决策案卷冻结同一 Owner 的权威订单事实、观察时间、事实等级、unknown 和 SHA-256 manifest。
2. **AI 建议**：建议独立追加，只能标 `quoted / estimated / unknown / mock / inferred`，不能冒充 `actual / external_observed / reconciled`。
3. **Owner 决定**：`selected / rejected / paused / request_more_evidence` 独立不可变；`selected` 必须精确冻结 capability、command、target 和输入 SHA-256。
4. **受控行动**：仅允许当前 Dispatcher 已登记的 `command.<type>.v1`，执行前重新核验最新 Owner 决定，并经过 production/high-risk `DispatchSafe`、独立审批、幂等与审计。
5. **结果观察**：只接收与同一订单对象一致的订单、最终利润或现金对账事实，作用区分 `support / counter / conflict`；不自动产生因果结论。
6. **下一轮建议**：必须已有结果观察，且固定为 `inferred/proposed`；下一次 Owner 决定仍需独立记录。

## 旧案卷真相门禁

- `experiment` 只保留经营事实核验案卷含义。
- 后端拒绝写入 `decision` 阶段、`final_decision`、`completed/stopped` 或 decision gate。
- Owner Summary、经营终局投影和页面明确返回/展示 `trace_only / not_established / not_authorized`。
- 小Q不得把旧 experiment gate 当作经营授权或反馈闭环证明。

## 自动验证

- 后端全量测试：3350 项通过，122 个 package；vet/build 通过。
- PostgreSQL：131 对迁移完成全量 `up → down → up`，最终版本 140。
- 路由安全：452 个 mutation 全部显式分类。
- 前端：完整套件 161 项首轮通过、8 项因并行 5 秒超时失败；失败的 5 个文件以单 worker/20 秒重跑 15 项全部通过。该证据说明是测试资源上限，不是断言失败。

## 证据边界

- 没有真实受控经营行动及其后续真实订单、利润或现金结果。
- 一次行动前后的结果只能成为支持、反证或冲突证据；平台不会据此声称行动造成了结果。
