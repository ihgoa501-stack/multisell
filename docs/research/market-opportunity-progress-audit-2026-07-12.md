# 市场与商品机会纵向单元进展审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 2 单元“市场与机会”
> 状态：工程纵向单元完成；真实市场结果仍为 unknown

## 本次实际建立

1. 研究评估只接受绑定同案、同 Owner、同 run、同来源和观察时间且原始 SHA-256 可复算的证据。无快照手写 evidence 不再能推动 `experiment_ready`。
2. support 只接受 scout quoted，counter 只接受独立 falsifier quoted；必须存在 data reality run，其 conflict 无论 truth 等级都阻断。
3. 新增不可变、幂等的 Owner 市场决定 `market_owner_decision`，将系统评估与 `selected/rejected/paused/request_more_evidence` 分开。
4. 新增 Owner 隔离的 `product_opportunity` 和不可变决定；只有最新 selected 市场决定可以创建机会，机会完整性达到 `ready_for_owner` 后才可批准。
5. 新增 `market.read / market.write / market.decide` 权限，决定权只授 admin/Owner。
6. Owner 页面可以选择或淘汰市场、填写理由、进入商品机会页、创建机会、检查完整性和批准；页面明确不触发采购、Listing、投放或发布。
7. 候选市场可在同一界面选择 2—4 个，按八个经营维度比较证据、反证、冲突、unknown 和最新 Owner 决定。
8. 评估冻结当时最后证据 ID；新增证据会使旧 verdict 失效，Owner 必须重新评估后才能选择。
9. PostgreSQL 约束保证研究批次、快照、案件、Owner、run、来源和观察时间一致，并阻止快照修改、删除及跨 Owner/跨案件拼接。
10. 1688 任务关联冻结商品机会和批准决定；受控采集、复核、草稿、验收、小Q只读以及发布请求/决定/执行均重新验证当前授权。历史 experiment 只供追踪。

## 验证证据

- DemandCase、sourcing1688、platformtruth、routecatalog 和 productimage 聚焦测试通过。
- 新增受控三 run、无快照伪造、Owner 市场决定、幂等冲突、旧选择失效、机会完整性和批准状态测试。
- 前端候选比较 4 个测试通过；定向 ESLint 通过；Next.js 生产构建成功并生成 `/demand-cases` 与 `/product-opportunities`。
- 后端全量 121 个包、3207 个测试通过；Go build 及 touched package vet 通过。
- 全新临时 PostgreSQL 完成 108 对迁移全上行、全回退、再次上行，最终版本 111；临时库随后删除。
- PostgreSQL provenance 专项测试验证合法绑定可写，跨 Owner、跨案件、run 错配、Owner 篡改、快照修改和删除均被拒绝。
- 路由审计确认 407 个 mutation endpoint 全部分类，图片集 API 改为显式注册并纳入策略清单。

## 证据边界与后续依赖

- `automated_verified` 只证明工程链和数据库约束，不证明任何候选市场值得进入、任何商品会成交或渠道账号可用。
- 真实候选、消费者需求、获客、履约、合规、渠道权限、费用和消费者行为仍为 `unknown`；必须由 Owner 后续真实经营输入升级证据等级。
- 第 2 单元工程验收完成；ADR-001 第 3—8 单元和“全部平台目标”仍未完成。
