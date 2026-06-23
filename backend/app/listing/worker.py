"""Background worker that executes pending listing task items."""

import asyncio
import logging
from datetime import datetime, timedelta, timezone

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import Inventory, ListingTaskItem, Platform, Price, Product, Sku

logger = logging.getLogger(__name__)


class ListingWorker:
    """Polls and executes pending listing task items."""

    def __init__(
        self,
        poll_interval: float = 10.0,
        max_retries: int = 3,
        retry_delay_seconds: float = 60.0,
    ):
        self._poll_interval = poll_interval
        self._max_retries = max_retries
        self._retry_delay = retry_delay_seconds
        self._task: asyncio.Task | None = None
        self._stop_event = asyncio.Event()

    async def start(self):
        self._stop_event.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("ListingWorker started (poll every %ss)", self._poll_interval)

    async def stop(self):
        self._stop_event.set()
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        logger.info("ListingWorker stopped")

    async def _loop(self):
        while not self._stop_event.is_set():
            try:
                await self._tick()
            except Exception:
                logger.exception("ListingWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        async with async_session_factory() as db:
            now = datetime.now(timezone.utc)
            cutoff = now - timedelta(seconds=self._retry_delay)
            stmt = (
                select(ListingTaskItem)
                .where(
                    (ListingTaskItem.status == "pending")
                    | (
                        (ListingTaskItem.status == "failed")
                        & (ListingTaskItem.retry_count < self._max_retries)
                        & (ListingTaskItem.executed_at < cutoff)
                    )
                )
                .order_by(ListingTaskItem.executed_at.asc().nullsfirst())
                .limit(5)
            )
            rows = (await db.execute(stmt)).scalars().all()
            for item in rows:
                await self._execute_one(db, item)
            await db.commit()

    async def _execute_one(self, db: AsyncSession, item: ListingTaskItem):
        try:
            platform = await db.get(Platform, item.platform_id)
            product = await db.get(Product, item.product_id)
            if not platform or not product:
                item.status = "failed"
                item.error_message = "Platform or Product not found"
                item.retry_count = (item.retry_count or 0) + 1
                item.executed_at = datetime.now(timezone.utc)
                await db.flush()
                return

            skus = (
                (await db.execute(select(Sku).where(Sku.product_id == product.id)))
                .scalars()
                .all()
            )
            sku_ids = [s.id for s in skus]

            prices = {}
            if sku_ids:
                price_rows = (
                    (await db.execute(select(Price).where(Price.sku_id.in_(sku_ids))))
                    .scalars()
                    .all()
                )
                prices = {p.sku_id: p for p in price_rows}

            inventories = {}
            if sku_ids:
                inv_rows = (
                    (
                        await db.execute(
                            select(Inventory).where(Inventory.sku_id.in_(sku_ids))
                        )
                    )
                    .scalars()
                    .all()
                )
                inventories = {i.sku_id: i for i in inv_rows}

            adapter = get_listing_adapter(platform.code)
            result = await adapter.publish(
                product=product,
                platform=platform,
                skus=skus,
                prices=prices,
                inventories=inventories,
                db=db,
            )
            item.status = "success"
            item.result = {
                "platform_product_id": result.platform_product_id,
                "platform_url": result.platform_url,
                "platform_sku": result.platform_sku,
            }
            item.executed_at = datetime.now(timezone.utc)
        except Exception as exc:
            item.retry_count = (item.retry_count or 0) + 1
            if item.retry_count >= self._max_retries:
                item.status = "failed"
                item.error_message = f"[exhausted] {exc}"[:500]
            else:
                item.status = "pending"
                item.error_message = str(exc)[:500]
            item.executed_at = datetime.now(timezone.utc)
            logger.warning(
                "Listing item %s failed (retry %s/%s): %s",
                item.id,
                item.retry_count,
                self._max_retries,
                exc,
            )
        await db.flush()
