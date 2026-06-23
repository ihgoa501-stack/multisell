"""Return sync worker — poll platform APIs, create AfterSalesOrder records."""

import asyncio
import logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from typing import Optional

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import AfterSalesOrder, Order, Platform, Sku

logger = logging.getLogger(__name__)


class ReturnSyncWorker:
    """Background worker that polls adapters' ``fetch_returns()`` and creates
    ``AfterSalesOrder`` records for new returns."""

    def __init__(self, poll_interval: float = 600.0):
        self._poll_interval = poll_interval
        self._task: Optional[asyncio.Task] = None
        self._stop = asyncio.Event()

    async def start(self):
        self._stop.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("ReturnSyncWorker started (poll every %ss)", self._poll_interval)

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
                logger.exception("ReturnSyncWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        async with async_session_factory() as db:
            platforms = (
                (await db.execute(select(Platform).where(Platform.status == 1)))
                .scalars()
                .all()
            )

            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_returns"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(hours=1)
                try:
                    returns = await adapter.fetch_returns(
                        platform=platform, since=since, db=db
                    )
                    for r in returns:
                        await self._upsert_return(db, r)
                except Exception:
                    logger.exception("Failed to fetch returns for %s", platform.code)
            await db.commit()

    async def _upsert_return(self, db: AsyncSession, r: dict):
        """Dedup by (order_id, sku_id), then create AfterSalesOrder."""

        return_id = r.get("return_id", "")
        order_sn = r.get("order_sn", "")
        sku_code = r.get("sku_code", "")
        quantity = r.get("quantity", 1)
        reason = r.get("reason", "平台发起退货")

        # Look up local Order by order_no
        order = (
            await db.execute(select(Order).where(Order.order_no == order_sn))
        ).scalar_one_or_none()

        if not order:
            logger.warning(
                "Order %s not found for return %s — skipping", order_sn, return_id
            )
            return

        # Look up local Sku by sku_code
        sku = (
            await db.execute(select(Sku).where(Sku.code == sku_code))
        ).scalar_one_or_none()

        if not sku:
            logger.warning(
                "SKU %s not found for return %s — skipping",
                sku_code,
                return_id,
            )
            return

        # Dedup: check if an AfterSalesOrder already exists for this order + sku
        existing = (
            await db.execute(
                select(AfterSalesOrder).where(
                    AfterSalesOrder.order_id == order.id,
                    AfterSalesOrder.sku_id == sku.id,
                )
            )
        ).scalar_one_or_none()

        if existing:
            logger.debug(
                "Return %s already exists as AfterSalesOrder %s — skipping",
                return_id,
                existing.id,
            )
            return

        refund_amount = r.get("refund_amount")
        rma = AfterSalesOrder(
            order_id=order.id,
            sku_id=sku.id,
            return_quantity=quantity,
            reason=reason,
            status="pending",
            created_by="system",
            refund_amount=Decimal(str(refund_amount)) if refund_amount else None,
        )
        db.add(rma)
        await db.flush()
        logger.info(
            "Created AfterSalesOrder %s for return %s (order=%s, sku=%s)",
            rma.id,
            return_id,
            order_sn,
            sku_code,
        )
