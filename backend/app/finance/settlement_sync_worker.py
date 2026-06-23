"""Settlement sync worker — poll platform APIs, upsert Settlement + SettlementItem."""

import asyncio
import logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from typing import Optional

from sqlalchemy import select

from app.database import async_session_factory
from app.listing.adapters import get_listing_adapter
from app.models import Platform, Settlement, SettlementItem

logger = logging.getLogger(__name__)


_POLL_INTERVAL = 3600.0  # every hour


class SettlementSyncWorker:
    """Background worker that polls adapters' ``fetch_settlements()`` and upserts
    ``Settlement`` + ``SettlementItem`` records."""

    def __init__(self, poll_interval: float = _POLL_INTERVAL):
        self._poll_interval = poll_interval
        self._task: Optional[asyncio.Task] = None
        self._stop = asyncio.Event()

    async def start(self):
        self._stop.clear()
        self._task = asyncio.create_task(self._loop())
        logger.info("SettlementSyncWorker started (poll every %ss)", self._poll_interval)

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
                logger.exception("SettlementSyncWorker tick failed")
            await asyncio.sleep(self._poll_interval)

    async def _tick(self):
        """One poll cycle: iterate active platforms, fetch settlements, upsert."""
        async with async_session_factory() as db:
            platforms = (
                await db.execute(select(Platform).where(Platform.status == 1))
            ).scalars().all()

            for platform in platforms:
                adapter = get_listing_adapter(platform.code)
                if not hasattr(adapter, "fetch_settlements"):
                    continue
                since = datetime.now(timezone.utc) - timedelta(days=7)
                try:
                    items = await adapter.fetch_settlements(
                        platform=platform, since=since, db=db
                    )
                    for tx in items:
                        await self._upsert_tx(db, tx, platform.id)
                except Exception:
                    logger.exception(
                        "Failed to fetch settlements for %s", platform.code
                    )
            await db.commit()

    async def _upsert_tx(self, db, tx: dict, platform_id: int):
        """Find or create a SettlementItem, dedup by ``transaction_id``.

        Also creates the parent ``Settlement`` batch on demand and updates its
        aggregate totals incrementally.
        """
        # Dedup: skip if transaction_id already exists
        existing = (
            await db.execute(
                select(SettlementItem).where(
                    SettlementItem.transaction_id == tx["transaction_id"],
                )
            )
        ).scalar_one_or_none()
        if existing:
            return

        # Parse occurred_at and derive the period start (first of the month)
        occurred_at = datetime.fromisoformat(
            tx["occurred_at"].replace("Z", "+00:00")
        )
        period_start = occurred_at.replace(day=1, hour=0, minute=0, second=0, microsecond=0)

        # Find or create the Settlement batch for this platform + period
        settlement = (
            await db.execute(
                select(Settlement).where(
                    Settlement.platform_id == platform_id,
                    Settlement.period_start == period_start,
                    Settlement.status == "pending",
                )
            )
        ).scalar_one_or_none()

        if not settlement:
            settlement = Settlement(
                platform_id=platform_id,
                settlement_no=f"API-{platform_id}-{period_start.strftime('%Y%m')}",
                period_start=period_start,
                status="pending",
            )
            db.add(settlement)
            await db.flush()

        # Build numeric values
        amount = Decimal(str(tx.get("amount", "0")))
        fee = Decimal(str(tx.get("fee", "0")))
        net = amount - fee

        item = SettlementItem(
            settlement_id=settlement.id,
            transaction_id=tx["transaction_id"],
            transaction_type=tx["transaction_type"],
            order_no=tx.get("order_sn", ""),
            amount=amount,
            fee=fee,
            net=net,
            occurred_at=occurred_at,
        )
        db.add(item)
        await db.flush()

        # Update settlement totals incrementally
        if amount > 0:
            settlement.total_revenue = (settlement.total_revenue or 0) + amount
        else:
            settlement.total_refund = (settlement.total_refund or 0) + abs(amount)
        settlement.total_fee = (settlement.total_fee or 0) + fee
        settlement.total_net = (settlement.total_net or 0) + net
