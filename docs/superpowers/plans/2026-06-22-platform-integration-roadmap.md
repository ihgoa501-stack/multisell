# 平台对接完整路线图 — 发布之后的 5 个阶段

> **For agentic workers:** This is a roadmap index, not an execution plan. Each phase has its own plan file. Implement them in order.

## 当前状态

```
发布 → 平台 ✅ (已通: worker + retry + credential validation)
  ↓
订单 → CSV 手动导入 ❌
  ↓
物流追踪 → 不回传 ❌
  ↓
结算对账 → CSV 手动导入 ❌
  ↓
售后同步 → 仅本地创建 ❌
  ↓
库存回写 → 不回传 ❌
```

## 5 个阶段

| # | 阶段 | 产出 | 前置依赖 |
|---|------|------|---------|
| 1 | **平台订单实时导入** | Ozon/Shopee 订单自动同步到 `sales_order` | — |
| 2 | **物流追踪回传** | 发货后追踪号自动推回平台 | ① |
| 3 | **平台结算导入** | 结算单从 API 拉取，利润账本能对账 | ① |
| 4 | **售后单同步** | 平台退款/退货自动创建 `after_sales_order` | ① |
| 5 | **库存回写** | 本地库存变动自动同步到平台 | ① |

## 架构模式（各阶段复用）

每阶段都遵循同一模式，与 listing worker 一致：

```
Adapter 新增方法 → Background worker 轮询 → HTTP API 调用 → 本地写入/状态回写
                                                        ↓
                                                  重试(3次) + 日志
```

**Adapter Protocol 扩展**（`app/listing/adapters/base.py`）：

```python
class ListingAdapter(Protocol):
    # 已有
    async def publish(...)
    async def sync_status(...)
    async def validate_credentials(...)
    
    # Phase 1 —
    async def fetch_orders(self, *, platform, since: datetime) -> list[dict]
    
    # Phase 2 —
    async def push_tracking(self, *, platform, order_sn: str, tracking_no: str) -> bool
    
    # Phase 3 —
    async def fetch_settlements(self, *, platform, since: datetime) -> list[dict]
    
    # Phase 5 —
    async def sync_inventory(self, *, platform, sku_code: str, quantity: int) -> bool
```

## 计划文件索引

- [`2026-06-22-p1-order-import.md`](2026-06-22-p1-order-import.md)
- [`2026-06-22-p2-tracking-push.md`](2026-06-22-p2-tracking-push.md)
- [`2026-06-22-p3-settlement-import.md`](2026-06-22-p3-settlement-import.md)
- [`2026-06-22-p4-aftersales-sync.md`](2026-06-22-p4-aftersales-sync.md)
- [`2026-06-22-p5-inventory-sync.md`](2026-06-22-p5-inventory-sync.md)
