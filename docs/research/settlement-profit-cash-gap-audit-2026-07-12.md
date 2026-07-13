# 结算、最终利润与现金纵向单元审计

> 日期：2026-07-12
> 对应路线：ADR-001 第 5 单元
> 工程状态：`implemented / automated_verified`；真实金额事实仍为 `unknown`

## 单一权威链

### 1. 平台结算

- 受保护入口按 Owner、平台账号和权威订单保存不可变结算原始字节、服务端 payload/content SHA-256、外部事件及结算身份。
- 结算行使用 `amount_minor + currency`，区分 `sale / fee / refund / commission` 及费用类型。
- `external_observed` 与 `mock` 只能绑定相同事实等级的订单；同键同内容幂等，不同内容失败关闭。
- 旧 AI mapper 直接写 `settlement_item` 的旁路已冻结。

### 2. 订单最终利润

- 每条订单行必须绑定同 Owner、同 SKU 的精确商品成本版本；成本明细只要包含 `quoted / estimated` 就不能形成最终利润。
- 最终利润只读取 `external_observed` 平台销售/费用/退款、分类履约费和可信售后终局回执。
- 缺项、混币、跨 Owner、售后未终局、退款金额不一致或缺少履约费用全部失败关闭。
- 结果以不可变版本和来源 manifest 保存；旧浮点 `order_profit_record` 不参与新权威计算。

### 3. 现金到账与对账

- 银行/支付到账凭证按 Owner 和账户隔离，保存原始 JSON、服务端 SHA-256、最小货币单位、币种、观察时间、价值日和固定 `external_observed` 等级。
- 应收只来自新平台结算权威；同币种、同对象分配支持 `unmatched / partial / reconciled / conflict`。
- 只有累计到账与结算净应收精确相等时才可标记 `reconciled`；结算应收不会自动冒充银行到账。

## 自动验证

- `go test ./...`：3341 项通过，121 个 package。
- `go vet ./...` 与 `go build ./...`：通过。
- 路由安全：443 个 mutation 全部显式分类，66 个高风险路由登记。
- PostgreSQL：128 对迁移完成全量 `up → down → up`，最终版本 136。

## 证据边界

- 没有真实平台结算、真实供应商成本、真实售后退款或真实银行/支付到账凭证。
- 因此平台不能声称任何订单已经形成真实最终利润或现金回收；当前只证明工程门禁和计算合同存在且自动验证通过。
