"""AgentOS 业务命令适配器 — 将 ActionProposal 调度到真实业务服务。

每个 handler 是 async 函数，接受 (db, payload) 返回结果 dict。
所有 handler 通过 HANDLER_MAP 注册，由 ActionCenterService.execute() 调度。
"""

from __future__ import annotations

from typing import Any

from sqlalchemy.ext.asyncio import AsyncSession


# ── Handler 1: profit_review ──────────────────────────────────


async def handle_profit_review(
    db: AsyncSession, payload: dict[str, Any]
) -> dict[str, Any]:
    """利润复核：调用 PreListingDecisionService.calculate() 试算"""
    from app.decision.schemas import PreListingDecisionRequest
    from app.decision.service import PreListingDecisionService

    sku_id = payload.get("sku_id")
    target_sale_price = payload.get("target_sale_price")
    if sku_id is None or target_sale_price is None:
        raise ValueError(
            f"profit_review 需要 sku_id 和 target_sale_price, 收到 payload keys: {list(payload.keys())}"
        )

    kwargs = dict(destination_country="RU", cargo_type="normal")
    for key in (
        "destination_country",
        "platform_id",
        "product_cost",
        "platform_fee_pct",
        "payment_fee_pct",
        "shipping_fee",
        "other_fee",
        "minimum_margin_pct",
        "cargo_type",
    ):
        if key in payload:
            kwargs[key] = payload[key]
    req = PreListingDecisionRequest(
        sku_id=sku_id,
        target_sale_price=target_sale_price,
        **kwargs,
    )
    result = await PreListingDecisionService.calculate(db, req)
    return result.model_dump()


# ── Handler 2: inventory_allocate ─────────────────────────────


async def handle_inventory_allocate(
    db: AsyncSession, payload: dict[str, Any]
) -> dict[str, Any]:
    """库存分配：手动或按规则自动分配"""
    from app.allocation.service import AllocationService

    sku_id = payload["sku_id"]
    warehouse_id = payload.get("warehouse_id")
    quantity = payload.get("quantity")

    if warehouse_id is not None and quantity is not None:
        result = await AllocationService.allocate(
            db,
            sku_id=sku_id,
            warehouse_id=warehouse_id,
            quantity=quantity,
        )
        return {"mode": "manual", "allocations": [result]}

    results = await AllocationService.auto_allocate(db, sku_id=sku_id)
    return {"mode": "auto", "allocations": results}


# ── Handler 3: listing_draft ──────────────────────────────────


async def handle_listing_draft(
    db: AsyncSession, payload: dict[str, Any]
) -> dict[str, Any]:
    """生成 Listing 草稿：调用 A2 优化器生成内容 + 持久化到 ProductListing"""
    from app.agent.agents.listing_optimizer import A2ListingOptimizerAgent
    from app.listing.service import ListingService
    from app.models import Product

    product_id = payload.get("product_id")
    if not product_id:
        raise ValueError("product_id is required for listing_draft")

    product = await db.get(Product, product_id)
    if not product:
        raise ValueError(f"Product {product_id} not found")

    # Step 1: 调用 A2 生成优化内容
    agent = A2ListingOptimizerAgent(user_id=0)
    context = {
        "product_name": payload.get("product_name", product.name or ""),
        "marketplace": payload.get("marketplace", "US"),
        "features": payload.get("features", []),
        "keywords": payload.get("keywords", []),
        "current_bullets": payload.get("current_bullets", []),
    }
    optimization = await agent.decide("listing_optimize", context)

    # Step 2: 持久化到 ProductListing
    platform_id = payload.get("platform_id")
    listing_id = None
    if platform_id:
        listing = await ListingService._get_or_create_listing(
            db, product_id, platform_id
        )
        if listing:
            listing.published_data = optimization
            await db.flush()
            listing_id = listing.id
        else:
            logger.warning(
                "listing_draft: _get_or_create_listing returned None for product=%s platform=%s",
                product_id,
                platform_id,
            )

    if listing_id is None and platform_id:
        raise RuntimeError(
            f"listing_draft: 无法为 product={product_id} platform={platform_id} 创建 listing, "
            f"优化内容已计算但未持久化"
        )

    return {
        "product_id": product_id,
        "optimization": optimization,
        "listing_id": listing_id,
    }


# ── Handler 4: daily_report ───────────────────────────────────


async def handle_daily_report(
    db: AsyncSession, payload: dict[str, Any]
) -> dict[str, Any]:
    """生成经营日报：调用 DashboardService 获取经营总览"""
    from app.dashboard.service import DashboardService

    result = await DashboardService.get_dashboard(db)
    return result


# ── Handler 5: notify ─────────────────────────────────────────


async def handle_notify(db: AsyncSession, payload: dict[str, Any]) -> dict[str, Any]:
    """执行预警检查：调用 NotificationService 扫描预警规则并创建通知"""
    from app.notification.service import NotificationService

    alerts = await NotificationService.check_and_create_alerts(db)
    try:
        unread = await NotificationService.get_unread_count(db)
    except Exception:
        unread = {}
    return {"alerts_created": alerts, "unread_summary": unread}


# ── 注册表 ────────────────────────────────────────────────────

HANDLER_MAP: dict[str, Any] = {
    "profit_review": handle_profit_review,
    "inventory_allocate": handle_inventory_allocate,
    "listing_draft": handle_listing_draft,
    "daily_report": handle_daily_report,
    "notify": handle_notify,
}
