# 订单、库存、履约与售后纵向单元审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 4 单元
> 工程状态：`implemented / automated_verified`；真实外部经营结果仍为 `unknown`

## 已完成的工程事实链

1. **订单与库存**
   - 受保护的 Owner/account 入口保存不可变平台订单原始事件、外部事件身份、观察时间和 SHA-256。
   - 商品行必须命中同一 Owner 的规范 SKU 映射；未知或跨 Owner SKU 失败关闭。
   - 金额权威为 `unit_price_minor + currency`；旧浮点订单金额仅是兼容投影。
   - `reserve → commit/release` 在同一事务内写订单行、库存变化和不可变逐行 ledger。
   - generic AI mapper 直接写 `sales_order` 的旧旁路已冻结。

2. **履约**
   - tracking 的创建、读取、列表和更新按 Owner 隔离。
   - 承运商事件保存不可变原始 payload，由服务端计算 SHA-256；同一来源事件支持相同内容幂等、不同内容冲突。
   - 只有 `external_observed` 的 `delivered` 事件可以投影 `actual_delivery`；人工更新不能填写真实签收时间。
   - 非空订单号必须关联同 Owner、已应用的外部订单事实；运行时 mock 承运商注入已移除。

3. **售后**
   - 新事实链为 `requested → approved/rejected → execution_submitted → succeeded/failed`。
   - 请求必须绑定同 Owner 的外部订单和平台账号；Owner 决策、外部执行与终局回执分别记录并保持幂等。
   - 只有 `platform_receipt` 或 `controlled_reconciliation` 能形成成功/失败终局，退款金额使用最小货币单位和币种。
   - 库存和财务后果明确为 `deferred`，不会因售后申请或内部决定自动声称已经发生。
   - 旧 `/aftersales/:id/refund` HTTP 直接退款入口已冻结。

## 自动验证

- `go test ./...`：3307 项通过，121 个 package。
- `go vet ./...` 与 `go build ./...`：通过。
- mutation policy：438 个写接口全部显式分类，65 个高风险路由登记。
- PostgreSQL：124 对迁移完成全量 `up → down → up`，最终版本 130。

## 仍然未知或待后续单元闭合

- 没有真实平台订单、库存动作、承运商签收或退款回执；不能写成 `external_observed` 已发生。
- 真实平台 webhook 和承运商 connector 的账号级签名解析尚未接入；当前为受保护的连接器归一入口。
- 售后成功后的现金、利润和库存返还后果属于 ADR-001 第 5 单元，当前明确保持 `deferred`。
- 旧售后 Service 中的兼容 `Refund` 方法仍存在但没有 HTTP 路由可达；后续完成调用方迁移后再删除，不能把 `delete` 建议当作删除授权。
# 2026-07-12 补充：采购/补货权威事实链

- `implemented / automated_verified`：采购请求按 Owner 隔离并冻结权威 supplier、canonical SKU mapping、精确 cost version、inventory、数量、minor-unit amount/currency 和 request SHA-256。
- `implemented / automated_verified`：`requested → owner_approved` 必须引用经营决定系统中绑定 exact purchase ID、`purchase.submit` 和 request SHA-256 的 `selected` Owner 决定。
- `implemented / automated_verified`：`external_submitted → ordered/failed → partially_received/fully_received` 只由服务端 SHA-256 的不可变 `external_observed` 回执推进；外部事件按 Owner + event ID 幂等且同一采购不能更换 external order identity。
- `implemented / automated_verified`：只有真实 receiving fact 能在同一事务内增加 exact SKU inventory 并追加不可变 receipt ledger；旧 `/purchase/orders` mutation 固定失败关闭，旧内部 `supplychain.order.received` 不再修改库存。
- `implemented / automated_verified`：Owner `/purchase` 页面已改为权威流程，显示冻结 minor-unit 金额、request SHA、外部事实和库存 ledger，并只开放当前状态允许的下一动作；Owner可在同一流程中不可变保存 exact selected 经营决定，但不会自动向供应商下单；3项聚焦测试和95页面生产构建通过。
- `automated_verified`：purchase/businessdecision/routecatalog 聚焦测试 54 项通过；后端全量 122 包 3365 项测试、Go vet/build 通过；461 个 mutation 路由显式分类；133 对 PostgreSQL 迁移完成全量 `up → down → up` 至版本 142。
- `unknown`：尚无真实供应商提交、真实采购单或真实收货凭证，不能声明外部采购与入库已经发生。
