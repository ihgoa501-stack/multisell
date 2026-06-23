"""Integration tests for ReturnSyncWorker — real DB upsert flow."""

import pytest
from decimal import Decimal
from sqlalchemy import select

from app.database import async_session_factory
from app.models import AfterSalesOrder, Order, Platform, Product, Sku
from app.aftersales.sync_worker import ReturnSyncWorker


async def _ensure_data(db):
    """Create test Platform + Product + Sku + Order in a single session."""
    p = Platform(id=999, code="test", name="TestPlatform", status=1)
    db.add(p)
    prod = Product(id=9999, name="Test Product", status=1)
    db.add(prod)
    await db.flush()
    s = Sku(id=9999, product_id=prod.id, code="RET-SKU-001", spec_desc="Test Spec")
    db.add(s)
    o = Order(
        id=9999,
        order_no="RET-ORDER-001",
        platform_id=999,
        status="delivered",
        total_amount=Decimal("100.00"),
        shipping_fee=Decimal("10.00"),
        pay_amount=Decimal("110.00"),
        recipient_name="Test",
    )
    db.add(o)


async def _cleanup(db):
    """Remove test data."""
    await db.execute(
        AfterSalesOrder.__table__.delete().where(AfterSalesOrder.order_id == 9999)
    )
    await db.execute(Order.__table__.delete().where(Order.id == 9999))
    await db.execute(Sku.__table__.delete().where(Sku.id == 9999))
    await db.execute(Product.__table__.delete().where(Product.id == 9999))
    await db.execute(Platform.__table__.delete().where(Platform.id == 999))


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_new_return():
    """A valid return dict creates an AfterSalesOrder with status=pending."""
    worker = ReturnSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        await db.commit()

    try:
        return_data = {
            "return_id": "RET-12345",
            "order_sn": "RET-ORDER-001",
            "sku_code": "RET-SKU-001",
            "quantity": 2,
            "reason": "商品破损",
            "status": "pending",
            "created_at": "2026-06-22T10:00:00Z",
            "refund_amount": "50.00",
        }

        async with async_session_factory() as db:
            await worker._upsert_return(db, return_data)
            await db.commit()

        async with async_session_factory() as db:
            rma = (
                await db.execute(
                    select(AfterSalesOrder).where(
                        AfterSalesOrder.order_id == 9999,
                    )
                )
            ).scalar_one_or_none()

            assert rma is not None
            assert rma.order_id == 9999
            assert rma.sku_id == 9999
            assert rma.return_quantity == 2
            assert rma.reason == "商品破损"
            assert rma.status == "pending"
            assert rma.created_by == "system"
            assert float(rma.refund_amount) == 50.00
    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.commit()


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_duplicate_return_skipped():
    """Calling _upsert_return twice with the same data does not create a duplicate."""
    worker = ReturnSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        await db.commit()

    try:
        return_data = {
            "return_id": "RET-99999",
            "order_sn": "RET-ORDER-001",
            "sku_code": "RET-SKU-001",
            "quantity": 1,
            "reason": "发错颜色",
            "created_at": "2026-06-22T12:00:00Z",
        }

        async with async_session_factory() as db:
            await worker._upsert_return(db, return_data)
            await worker._upsert_return(db, return_data)
            await db.commit()

        async with async_session_factory() as db:
            rmas = (
                (
                    await db.execute(
                        select(AfterSalesOrder).where(
                            AfterSalesOrder.order_id == 9999,
                            AfterSalesOrder.sku_id == 9999,
                        )
                    )
                )
                .scalars()
                .all()
            )

            assert len(rmas) == 1, "Duplicate return should be skipped"
    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.commit()


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_return_no_order_skipped():
    """Return with unknown order_sn is skipped gracefully."""
    worker = ReturnSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        await db.commit()

    try:
        return_data = {
            "return_id": "RET-55555",
            "order_sn": "NONEXISTENT-ORDER",
            "sku_code": "RET-SKU-001",
            "quantity": 1,
            "reason": "无此订单",
            "created_at": "2026-06-22T14:00:00Z",
        }

        async with async_session_factory() as db:
            await worker._upsert_return(db, return_data)
            await db.commit()

        async with async_session_factory() as db:
            rma = (
                await db.execute(
                    select(AfterSalesOrder).where(AfterSalesOrder.reason == "无此订单")
                )
            ).scalar_one_or_none()
            assert rma is None, (
                "No AfterSalesOrder should be created when order is missing"
            )
    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.commit()


@pytest.mark.usefixtures("prepare_db")
@pytest.mark.asyncio
async def test_upsert_return_no_sku_skipped():
    """Return with unknown sku_code is skipped gracefully."""
    worker = ReturnSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        await db.commit()

    try:
        return_data = {
            "return_id": "RET-66666",
            "order_sn": "RET-ORDER-001",
            "sku_code": "NONEXISTENT-SKU",
            "quantity": 1,
            "reason": "无此SKU",
            "created_at": "2026-06-22T15:00:00Z",
        }

        async with async_session_factory() as db:
            await worker._upsert_return(db, return_data)
            await db.commit()

        async with async_session_factory() as db:
            rma = (
                await db.execute(
                    select(AfterSalesOrder).where(AfterSalesOrder.reason == "无此SKU")
                )
            ).scalar_one_or_none()
            assert rma is None, (
                "No AfterSalesOrder should be created when SKU is missing"
            )
    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.commit()
