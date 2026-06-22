"""Integration tests for OrderSyncWorker — real DB upsert flow."""

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import Order, OrderItem, OrderStatusLog, Platform, Product, Sku
from app.order_import.sync_worker import OrderSyncWorker


async def _ensure_data(db):
    """Create test Platform + Product + Sku in a single session."""
    p = Platform(id=999, code="test", name="TestPlatform", status=1)
    db.add(p)
    prod = Product(id=9999, name="Test Product", status=1)
    db.add(prod)
    await db.flush()
    s = Sku(id=9999, product_id=prod.id, code="INTEG-SKU-001", spec_desc="Test Spec")
    db.add(s)


async def _cleanup(db):
    """Remove test data."""
    await db.execute(OrderItem.__table__.delete().where(OrderItem.sku_code.like("INTEG-%")))
    await db.execute(OrderStatusLog.__table__.delete())
    await db.execute(Order.__table__.delete().where(Order.order_no.like("INTEG-%")))
    await db.execute(Sku.__table__.delete().where(Sku.id == 9999))
    await db.execute(Product.__table__.delete().where(Product.id == 9999))
    await db.execute(Platform.__table__.delete().where(Platform.id == 999))


@pytest.mark.asyncio
async def test_upsert_new_order_db():
    """A new platform order creates Order + OrderItem + OrderStatusLog."""
    worker = OrderSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        await db.commit()

    try:
        platform_order = {
            "order_sn": "INTEG-001",
            "status": "delivered",
            "shipping_fee": "10.00",
            "paid_at": "2026-06-20T10:00:00Z",
            "recipient_name": "Bob",
            "recipient_phone": "999",
            "shipping_address": "Test City, Street 1",
            "items": [{"sku_code": "INTEG-SKU-001", "quantity": 3, "unit_price": "25.00"}],
        }

        async with async_session_factory() as db:
            await worker._upsert_order(db, platform_order, 999)
            await db.commit()

        async with async_session_factory() as db:
            order = (await db.execute(
                select(Order).where(Order.order_no == "INTEG-001")
            )).scalar_one_or_none()
            assert order is not None
            assert order.platform_id == 999
            assert order.status == "delivered"
            assert float(order.total_amount) == 75.00

            item = (await db.execute(
                select(OrderItem).where(
                    OrderItem.order_id == order.id,
                    OrderItem.sku_code == "INTEG-SKU-001",
                )
            )).scalar_one_or_none()
            assert item is not None
            assert item.sku_id == 9999
            assert item.product_id == 9999
            assert item.quantity == 3

            log = (await db.execute(
                select(OrderStatusLog).where(OrderStatusLog.order_id == order.id)
            )).scalar_one_or_none()
            assert log is not None
            assert log.to_status == "delivered"

    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.commit()


@pytest.mark.asyncio
async def test_upsert_existing_order_updates_status():
    """An existing order gets its status updated, no duplicate."""
    worker = OrderSyncWorker()
    async with async_session_factory() as db:
        await _ensure_data(db)
        o = Order(
            order_no="INTEG-002",
            platform_id=999,
            status="pending",
            total_amount=50,
            shipping_fee=5,
            pay_amount=55,
        )
        db.add(o)
        await db.commit()

    try:
        platform_order = {
            "order_sn": "INTEG-002",
            "status": "shipped",
            "shipping_fee": "5.00",
            "paid_at": None,
            "recipient_name": "",
            "recipient_phone": "",
            "shipping_address": "",
            "items": [],
        }
        async with async_session_factory() as db:
            await worker._upsert_order(db, platform_order, 999)
            await db.commit()

        async with async_session_factory() as db:
            order = (await db.execute(
                select(Order).where(Order.order_no == "INTEG-002")
            )).scalar_one_or_none()
            assert order is not None
            assert order.status == "shipped"

    finally:
        async with async_session_factory() as db:
            await _cleanup(db)
            await db.execute(Order.__table__.delete().where(Order.order_no == "INTEG-002"))
            await db.commit()
