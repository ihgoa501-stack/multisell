# Demo Data — 模拟经营数据

本目录包含三组演示 CSV 文件，与 `load_demo_data.py` 脚本配合使用，
可在无真实业务数据的情况下演示凌镜 LingMirror 的核心业务闭环。

## 文件清单

| 文件 | 用途 | 对应的导入接口 |
|---|---|---|
| `order_import_demo.csv` | 模拟 CSV 订单导入 | `POST /api/order-imports/csv` |
| `shipping_bill_demo.csv` | 模拟物流商运费账单 | `POST /api/shipping/bills/import` |
| `platform_settlement_demo.csv` | 模拟平台结算单 | `POST /api/settlements/import` |

## 场景设计

### order_import_demo.csv — 7 行 6 个独立逻辑行

- **OZ-10001**: 含 2 个 SKU 的同一订单（蓝牙耳机黑+白），有 tracking_number，正利润
- **OZ-10002**: 单 SKU 智能手表，有 tracking_number，正利润（结算含 refund 行）
- **SP-20001**: 单 SKU 夹克，无 tracking_number，正利润
- **SP-20002**: 单 SKU 精华液 50ml × 2，低利润（高 price 拉低利润率）
- **WB-30001**: 单 SKU 坚果 1kg × 3，高 shipping_fee → **负利润**（触发异常工作台）
- **WB-30002**: 单 SKU 瑜伽垫，无运费快照 → 触发 missing_snapshot

### shipping_bill_demo.csv — 5 行

- **RUS-TRK-001**: 匹配订单 OZ-10001（蓝牙耳机），金额差异：快照 35 → 账单 38 → amount_mismatch
- **RUS-TRK-002**: 匹配订单 OZ-10002（智能手表），金额差异：快照 42 → 账单 45+2=47 → amount_mismatch
- **SP-20002**: 按 order_no 匹配精华液订单，金额匹配 → matched
- **RUS-TRK-003**: 匹配订单 WB-30001（坚果），金额差异 → amount_mismatch
- **UNMATCHED-BILL-001**: 无匹配订单 → unmatched_bill（异常工作台可见）

### platform_settlement_demo.csv — 18 行

- **OZ-10001**: sale + platform_fee + payment_fee — 可匹配
- **OZ-10002**: sale + platform_fee + payment_fee + refund — 可匹配
- **SP-20001**: sale + platform_fee — 可匹配
- **SP-20002**: sale + platform_fee + adjustment — 可匹配
- **WB-30001**: sale + platform_fee + payment_fee — 可匹配
- **WB-30002**: sale + platform_fee — 可匹配
- **UNMATCHED-SETTLEMENT-001**: unmatched 行 — 无匹配订单（异常工作台可见）

## 业务闭环覆盖

- 商品 → SKU（demo seed 创建 5 个商品、14 个 SKU）
- 库存（每个 SKU 均有库存记录）
- 物流报价规则（2 家供应商、3 个渠道、3 条报价规则）
- 平台费用规则（3 条规则覆盖 Ozon、Shopee、Wildberries）
- 上架前利润测算（决策模块配置就绪）
- 上架任务（listing 权限码就绪）
- CSV 订单导入 → 订单创建 → 运费快照 → 利润账本
- 运费账单导入 → 对账
- 平台结算导入 → 匹配
- 异常工作台（负利润、unmatched bill、unmatched settlement）
- 利润看板

## 前提条件

1. PostgreSQL 数据库已运行
2. 测试数据库 `product_management_test` 已创建
3. Python 虚拟环境已激活（`backend/.venv`）

详细步骤见 `docs/DEMO_SCENARIO.md`。
