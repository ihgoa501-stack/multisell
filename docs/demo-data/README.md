# Demo Data — 模拟经营数据

> 最后更新：2026-06-24
> 状态：历史 CSV 样例，待适配 Go 新栈 demo seed

本目录包含三组历史演示 CSV 文件：

| 文件 | 历史用途 |
|---|---|
| `order_import_demo.csv` | 模拟 CSV 订单导入 |
| `shipping_bill_demo.csv` | 模拟物流商运费账单 |
| `platform_settlement_demo.csv` | 模拟平台结算单 |

这些文件最初配合旧 Python/FastAPI demo seed 使用。当前活跃后端已迁移到 `backend-go/`，API 前缀为 `/api/v1`，因此这些 CSV 只能作为新 demo seed 的字段样例，不能直接代表当前可执行验收流程。

## 新栈适配要求

在重新启用前，需要确认：

1. CSV 字段与 `backend-go/internal/domain/*` models 对齐。
2. 导入 API 路径迁移到 `/api/v1/*`。
3. demo seed 改为 Go 实现或明确支持 Go 后端。
4. 订单、物流账单、结算行能和 Go 后端当前订单/财务/异常链路闭环。

## 建议保留场景

### `order_import_demo.csv`

建议继续覆盖：

- 同平台订单多 SKU 合并
- 缺失 SKU 行失败
- 有 tracking number 的订单
- 高运费/低利润订单

### `shipping_bill_demo.csv`

建议继续覆盖：

- matched
- amount mismatch
- unmatched bill

### `platform_settlement_demo.csv`

建议继续覆盖：

- sale
- platform fee
- payment fee
- refund
- adjustment
- unmatched settlement

## 当前参考文档

- `docs/DEMO_SCENARIO.md`
- `docs/DEMO_ACCEPTANCE_REPORT.md`
- `docs/ORDER_IMPORT_SMOKE_CHECKLIST.md`
