# 市场与商品机会 Owner 流程

## 目的

完成 ADR-001 第二纵向单元中的权威决策链：候选市场研究只提供可追溯材料，系统评估只判断是否可供审议，Owner 决定市场，随后形成独立商品机会。任何一步都不自动采购、发布、投放或证明真实需求。

## 状态与边界

```text
scout_result + falsifier_result + data_reality_result
→ 系统研究评估（evidence_missing / rejected / experiment_ready兼容名）
→ Owner市场决定（selected / rejected / paused / request_more_evidence）
→ 商品机会 draft
→ 完整性检查（evidence_missing / ready_for_owner）
→ Owner商品机会决定（approved / rejected / paused）
→ approved 仅授权进入货源研究
→ 私人1688收藏关联冻结的商品机会决定
→ 受控采集、Owner复核、草稿与独立发布审批
```

- 三类研究必须绑定同案、同 Owner 的不可变快照、原始载荷 SHA-256、来源和观察时间。
- 无快照的手写 evidence 即使写入也不能改变研究评估。
- support 必须来自 scout 的 `quoted` 证据；counter 必须来自独立 falsifier；缺少 data reality run 或存在其 conflict 都阻断。
- `experiment_ready` 是暂留技术名，不代表 Owner 已选择市场。
- 市场决定和机会决定均要求非空理由、幂等键，并冻结对应 verdict 或内容哈希。
- 市场评估冻结当时的 `evidence_max_id`；新增证据后旧 verdict 不得继续用于 `selected`，必须重新评估。
- 1688任务关联同时冻结 `product_opportunity_id` 和 `opportunity_decision_id`。市场暂停/拒绝、机会批准变化、渠道不一致或历史 `legacy_experiment` 关联都会 fail closed；`experiment_id` 只保留追踪作用。
- 旧 `candidate_product` 和旧 `experiment` 均不是商品机会权威来源。

## API 与权限

- `market.read`：读取案件、决定和机会。
- `market.write`：创建案件、导入研究、评估和创建机会草案。
- `market.decide`：Owner 市场决定与商品机会决定，仅授予 admin/Owner。

所有路由仍在 JWT、mutation policy 和同步 HTTP 审计保护下。决定记录本身是追加式业务凭证；数据库迁移包括 `000105_market_opportunity_owner_decisions`、`000106_demand_research_provenance_constraints`、`000109_sourcing_opportunity_authority` 和 `000110_demand_verdict_evidence_watermark`。

## 自动验证（2026-07-12）

- 候选比较、Owner决定、商品机会完整性与授权聚焦测试通过。
- PostgreSQL 临时空库完成全部迁移上行、全量回退和再次上行，最终版本为 111；临时库随后删除。
- PostgreSQL 专项测试验证跨 Owner / 跨案件快照、证据来源错配、快照修改和删除均被数据库拒绝。
- 前端候选比较测试、目标文件 ESLint 和 Next.js 生产构建通过。

## 当前证据限制

代码、测试和页面只证明工程行为。真实市场是否成立、商品是否值得经营、渠道权限、真实费用、付款、售后和最终利润仍为 `unknown`，必须由后续外部观察和对账升级。
