"""Orders sync worker — poll platform APIs, upsert locally."""

import asyncio
import logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import Order, OrderItem, OrderStatusLog, Platform, Sku

logger = logging.getLogger(__name__)

STATUS_MAP = {
    "delivered": "delivered",
    "shipped": "shipped",
    "ready_to_ship": "paid",
    "cancelled": "cancelled",
}


def map_platform_order(platform_order: dict, platform_id: int) -> dict:
    """Map a platform order dict to local Order representation.

    Returns a dict with fields suitable for Order creation plus an ``items``
    list containing item-level data (no DB lookups — sku_id/product_id added
    later in ``_upsert_order``).
    """
    items = []
    for i in platform_order.get("items", []):
        unit_price = Decimal(str(i.get("unit_price", "0")))
        quantity = i.get("quantity", 0)
        items.append({
            "sku_code": i.get("sku_code", ""),
            "quantity": quantity,
            "unit_price": unit_price,
            "subtotal": unit_price * quantity,
        })

    total = sum(i["subtotal"] for i in items)
    fee = Decimal(str(platform_order.get("shipping_fee", "0")))

    paid_at = None
    raw_paid = platform_order.get("paid_at")
    if raw_paid:
        try:
            paid_at = datetime.fromisoformat(raw_paid.replace("Z", "+00:00"))
        except (ValueError, TypeError):
            pass

    return {
        "order_no": platform_order["order_sn"],
        "status": STATUS_MAP.get(platform_order.get("status", ""), "pending"),
        "total_amount": total,
        "shipping_fee": fee,
        "pay_amount": total + fee,
        "recipient_name": platform_order.get("recipient_name", ""),
        "recipient_phone": platform_order.get("recipient_phone", ""),
        "shipping_address": platform_order.get("shipping_address", ""),
        "paid_at": paid_at,
        "items": items,
    }


class OrderSyncWorker:
    """Background worker that polls adapters' ``fetch_orders()`` and upserts
    ``Order`` + ``OrderItem`` records via ``order_no`` unique constraint."""

    def __init__(self, poll_interval: float = 300.0):
        self._poll_interval = poll_interval
        self._task: Optional[asyncio.Task] = None
        self._stop = asyncio.Event()

    async def start(self):
        self._stop.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("OrderSyncWorker started (poll every %ss)", self._poll_interval)

    async def stop(self):
        self._stop.set()
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass

    async def _loop(self):
        while not self._stop.is_set():
            try:
                await self._tick()
            except Exception:
                logger.exception("OrderSyncWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        async with async_session_factory() as db:
            platforms = (
                await db.execute(
                    select(Platform).where(Platform.status == 1)
                )
            ).scalars().all()

            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_orders"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(hours=1)
                try:
                    orders = await adapter.fetch_orders(
                        platform=platform, since=since, db=db
                    )
                    for order in orders:
                        await self._upsert_order(db, order, platform.id)
                except Exception:
                    logger.exception(
                        "Failed to fetch orders for %s", platform.code
                    )
            await db.commit()

    async def _upsert_order(
        self, db: AsyncSession, platform_order: dict, platform_id: int
    ):
        mapped = map_platform_order(platform_order, platform_id)

        # Check for existing order by order_no
        existing = (
            await db.execute(
                select(Order).where(Order.order_no == mapped["order_no"])
            )
        ).scalar_one_or_none()

        if existing:
            # Update status if changed; also set platform_id if missing
            if existing.status != mapped["status"]:
                existing.status = mapped["status"]
            if existing.platform_id is None:
                existing.platform_id = platform_id
            order = existing
        else:
            order = Order(
                order_no=mapped["order_no"],
                platform_id=platform_id,
                status=mapped["status"],
                total_amount=mapped["total_amount"],
                shipping_fee=mapped["shipping_fee"],
                pay_amount=mapped["pay_amount"],
                recipient_name=mapped["recipient_name"],
                recipient_phone=mapped["recipient_phone"],
                shipping_address=mapped["shipping_address"],
                paid_at=mapped["paid_at"],
            )
            db.add(order)
            await db.flush()

            # Create status log for new orders
            status_log = OrderStatusLog(
                order_id=order.id,
                from_status=None,
                to_status=mapped["status"],
                operator="system",
                remark="从平台同步",
            )
            db.add(status_log)

        # Upsert order items
        await self._upsert_order_items(db, order, mapped["items"])

    async def _upsert_order_items(
        self, db: AsyncSession, order: Order, items: list[dict]
    ):
        for item in items:
            sku_code = item.get("sku_code", "")
            if not sku_code:
                continue

            # Look up SKU by code
            sku = (
                await db.execute(select(Sku).where(Sku.code == sku_code))
            ).scalar_one_or_none()

            unit_price = item["unit_price"]
            quantity = item["quantity"]

            if sku:
                sku_id = sku.id
                product_id = sku.product_id
                product_name = ""
                spec_desc = sku.spec_desc or ""
            else:
                logger.warning(
                    "SKU %s not found for order %s — skipping item",
                    sku_code, order.order_no,
                )
                continue

            # Check if this item already exists (same sku_code within the order)
            existing_item = (
                await db.execute(
                    select(OrderItem).where(
                        OrderItem.order_id == order.id,
                        OrderItem.sku_code == sku_code,
                    )
                )
            ).scalar_one_or_none()

            if existing_item:
                existing_item.quantity = quantity
                existing_item.unit_price = unit_price
                existing_item.subtotal = unit_price * quantity
            else:
                db.add(OrderItem(
                    order_id=order.id,
                    sku_id=sku_id,
                    product_id=product_id,
                    product_name=product_name,
                    sku_code=sku_code,
                    spec_desc=spec_desc,
                    unit_price=unit_price,
                    quantity=quantity,
                    subtotal=unit_price * quantity,
                ))
