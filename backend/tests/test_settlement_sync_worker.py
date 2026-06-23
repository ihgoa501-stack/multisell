"""Tests for settlement sync worker — uses async session (test DB)."""

import pytest
from datetime import datetime, timezone, timedelta
from decimal import Decimal

from sqlalchemy import select

from app.database import async_session_factory
from app.models import Settlement, SettlementItem, Platform
from app.finance.settlement_sync_worker import SettlementSyncWorker


_COUNTER = 0


async def _ensure_platform(db) -> Platform:
    """Create a test platform with a unique code in the given session."""
    global _COUNTER
    _COUNTER += 1
    code = f"test_settle_sync_{_COUNTER}"
    p = Platform(code=code, name=f"TestSettleSync_{_COUNTER}", status=1)
    db.add(p)
    await db.commit()
    await db.refresh(p)
    return p


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_tx_creates_settlement_and_item():
    """_upsert_tx creates a Settlement batch + SettlementItem."""
    async with async_session_factory() as db:
        platform = await _ensure_platform(db)
        worker = SettlementSyncWorker(poll_interval=999999)
        tx = {
            "transaction_id": "TXN-001",
            "transaction_type": "order_sale",
            "order_sn": "ORD-001",
            "amount": "100.00",
            "fee": "10.00",
            "currency": "RUB",
            "occurred_at": "2026-06-15T12:00:00Z",
            "description": "Test sale",
        }
        await worker._upsert_tx(db, tx, platform.id)
        await db.commit()

    async with async_session_factory() as db:
        settlements = (
            await db.execute(
                select(Settlement).where(Settlement.platform_id == platform.id)
            )
        ).scalars().all()
        assert len(settlements) == 1
        s = settlements[0]
        assert s.period_start.month == 6
        assert s.period_start.year == 2026
        assert s.status == "pending"
        assert s.total_revenue == Decimal("100.00")
        assert s.total_fee == Decimal("10.00")
        assert s.total_net == Decimal("90.00")  # 100 - 10
        assert s.total_refund == Decimal("0")

        # Verify SettlementItem was created
        items = (
            await db.execute(
                select(SettlementItem).where(
                    SettlementItem.settlement_id == s.id
                )
            )
        ).scalars().all()
        assert len(items) == 1
        item = items[0]
        assert item.transaction_id == "TXN-001"
        assert item.transaction_type == "order_sale"
        assert item.order_no == "ORD-001"
        assert item.amount == Decimal("100.00")
        assert item.fee == Decimal("10.00")
        assert item.net == Decimal("90.00")


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_tx_dedup():
    """Duplicate transaction_id is silently skipped."""
    async with async_session_factory() as db:
        platform = await _ensure_platform(db)
        worker = SettlementSyncWorker(poll_interval=999999)
        tx = {
            "transaction_id": "TXN-DUP",
            "transaction_type": "order_sale",
            "order_sn": "ORD-DUP",
            "amount": "50.00",
            "fee": "5.00",
            "currency": "RUB",
            "occurred_at": "2026-06-20T08:00:00Z",
        }
        await worker._upsert_tx(db, tx, platform.id)
        await worker._upsert_tx(db, tx, platform.id)  # same tx again
        await db.commit()

    async with async_session_factory() as db:
        settlements = (
            await db.execute(
                select(Settlement).where(Settlement.platform_id == platform.id)
            )
        ).scalars().all()
        assert len(settlements) == 1
        items = (
            await db.execute(
                select(SettlementItem).join(Settlement).where(
                    Settlement.platform_id == platform.id
                )
            )
        ).scalars().all()
        assert len(items) == 1  # still one item after dedup


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_tx_grouped_by_period():
    """Transactions in the same month go to the same Settlement."""
    async with async_session_factory() as db:
        platform = await _ensure_platform(db)
        worker = SettlementSyncWorker(poll_interval=999999)
        for i in range(3):
            tx = {
                "transaction_id": f"TXN-GRP-{i}",
                "transaction_type": "order_sale",
                "order_sn": f"ORD-GRP-{i}",
                "amount": "30.00",
                "fee": "3.00",
                "currency": "RUB",
                "occurred_at": f"2026-06-{10 + i:02d}T00:00:00Z",
            }
            await worker._upsert_tx(db, tx, platform.id)
        await db.commit()

    async with async_session_factory() as db:
        settlements = (
            await db.execute(
                select(Settlement).where(Settlement.platform_id == platform.id)
            )
        ).scalars().all()
        assert len(settlements) == 1  # same month → one batch
        s = settlements[0]
        assert s.total_revenue == Decimal("90.00")  # 30 * 3
        assert s.total_fee == Decimal("9.00")  # 3 * 3
        assert s.total_net == Decimal("81.00")  # (30-3) * 3

        items = (
            await db.execute(
                select(SettlementItem).where(
                    SettlementItem.settlement_id == s.id
                )
            )
        ).scalars().all()
        assert len(items) == 3


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_tx_refund_sets_refund_total():
    """A negative amount transaction updates total_refund."""
    async with async_session_factory() as db:
        platform = await _ensure_platform(db)
        worker = SettlementSyncWorker(poll_interval=999999)
        sale_tx = {
            "transaction_id": "TXN-SALE",
            "transaction_type": "order_sale",
            "order_sn": "ORD-SALE",
            "amount": "200.00",
            "fee": "20.00",
            "currency": "RUB",
            "occurred_at": "2026-06-10T00:00:00Z",
        }
        refund_tx = {
            "transaction_id": "TXN-REFUND",
            "transaction_type": "refund",
            "order_sn": "ORD-REFUND",
            "amount": "-50.00",
            "fee": "0.00",
            "currency": "RUB",
            "occurred_at": "2026-06-15T00:00:00Z",
        }
        await worker._upsert_tx(db, sale_tx, platform.id)
        await worker._upsert_tx(db, refund_tx, platform.id)
        await db.commit()

    async with async_session_factory() as db:
        settlements = (
            await db.execute(
                select(Settlement).where(Settlement.platform_id == platform.id)
            )
        ).scalars().all()
        assert len(settlements) == 1
        s = settlements[0]
        assert s.total_revenue == Decimal("200.00")
        assert s.total_refund == Decimal("50.00")  # absolute value of -50
        assert s.total_fee == Decimal("20.00")
        assert s.total_net == Decimal("130.00")  # (200-20) + (-50-0) = 180 - 50


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_tick_skips_missing_adapter():
    """_tick handles platforms without fetch_settlements gracefully."""
    async with async_session_factory() as db:
        platform = await _ensure_platform(db)
        worker = SettlementSyncWorker(poll_interval=999999)
        # platform.code = "test_settle_sync" won't match any real adapter
        await worker._tick()

    # No exception → pass
