# 1688 受控草稿链工程事实审计

> 审计日期：2026-07-12
> 范围：本工作树中的 1688 采集到内部待上架草稿变更
> 状态：工程变更快照；不替代同日产品方向审计

## 裁决

- `implemented`：新增受控 capture、Owner review、convert-to-draft、快照读取和草稿读取 API；旧自由写入路由不再注册。
- `implemented`：来源快照包含 URL、观察时间、采集驱动、解析版本、原始 JSON 与 SHA-256，数据库触发器拒绝更新或删除。
- `implemented`：转草稿要求已批准市场、通过 opportunity gate 的 active 实验、同一 Owner、供应商/合规证据、SKU 映射、图片权利和处理标准、11 类成本、渠道类目规则、配送模板与本地化内容。
- `implemented`：转草稿在单事务内创建内部产品、SKU、媒体、成本、追溯和 `product_listing(status=draft)`；没有平台适配器调用。
- `automated_verified`：2026-07-12 聚焦 Go 测试与 Next production build 通过。自动测试使用本地/模拟业务数据，不提高外部事实等级。
- `external_observed`：`unknown`。尚未采集并人工核验一个真实 1688 商品。
- `manually_verified`：`unknown`。尚未在真实登录、真实页面、真实图片权利、真实费用与已批准渠道条件下完成浏览器全链验收。
- 生产迁移状态：`unknown`。本次未连接生产 PostgreSQL，也未执行部署。

## 不得声称

当前不得声称 1688 生产采集可用、供应商可靠、图片有权使用、目标渠道接受草稿、最终成本准确或商品值得经营。上述事实必须由一个真实 Owner 批准商品实验及相应外部凭证核验。
