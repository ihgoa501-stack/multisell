"""Tests for the background listing task worker."""

import asyncio
from datetime import datetime, timezone, timedelta
from uuid import uuid4

import pytest
from sqlalchemy import select

from app.database import async_session_factory
from app.models import ListingTask, ListingTaskItem, Platform, Product


def _code(prefix: str = "test") -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _create_product(session) -> Product:
    code = _code("prod")
    p = Product(name=code)
    session.add(p)
    await session.flush()
    return p


async def _create_platform(session, code: str = None) -> Platform:
    if code is None:
        code = _code("plat")
    p = Platform(name=code, code=code, status=1)
    session.add(p)
    await session.flush()
    return p


async def _get_or_create_mockfail_platform(session) -> Platform:
    """Get existing mockfail platform or create one (avoids unique-constraint collisions across tests)."""
    stmt = select(Platform).where(Platform.code == "mockfail")
    platform = (await session.execute(stmt)).scalar_one_or_none()
    if platform:
        return platform
    p = Platform(name="mockfail", code="mockfail", status=1)
    session.add(p)
    await session.flush()
    return p


async def _create_listing_task(
    session, product_id: int, platform_id: int
) -> ListingTask:
    lt = ListingTask(
        product_id=product_id,
        platform_id=platform_id,
        source_type="test",
        status="ready",
    )
    session.add(lt)
    await session.flush()
    return lt


async def _create_pending_item(
    session, task_id: int, product_id: int, platform_id: int
) -> ListingTaskItem:
    item = ListingTaskItem(
        task_id=task_id,
        product_id=product_id,
        platform_id=platform_id,
        status="pending",
    )
    session.add(item)
    await session.flush()
    return item


@pytest.mark.asyncio
async def test_worker_marks_item_success(async_client):
    """Worker picks up a pending item and succeeds via MockListingAdapter."""
    async with async_session_factory() as session:
        platform = await _create_platform(session)
        product = await _create_product(session)
        task = await _create_listing_task(session, product.id, platform.id)
        item = await _create_pending_item(session, task.id, product.id, platform.id)
        await session.commit()
        item_id = item.id

    from app.listing.worker import ListingWorker

    worker = ListingWorker(poll_interval=0.05)
    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with async_session_factory() as session:
        stmt = select(ListingTaskItem).where(ListingTaskItem.id == item_id)
        row = (await session.execute(stmt)).scalar_one_or_none()
        assert row is not None, "Item should exist"
        assert row.status == "success", f"Expected success, got {row.status}"
        assert row.result is not None
        assert "platform_product_id" in row.result
        assert row.executed_at is not None


@pytest.mark.asyncio
async def test_worker_marks_item_failed_on_adapter_error(async_client):
    """If the adapter raises, item gets status='failed' + error_message."""
    async with async_session_factory() as session:
        platform = await _get_or_create_mockfail_platform(session)
        product = await _create_product(session)
        task = await _create_listing_task(session, product.id, platform.id)
        item = await _create_pending_item(session, task.id, product.id, platform.id)
        await session.commit()
        item_id = item.id

    from app.listing.worker import ListingWorker

    worker = ListingWorker(poll_interval=0.05, max_retries=0)
    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with async_session_factory() as session:
        stmt = select(ListingTaskItem).where(ListingTaskItem.id == item_id)
        row = (await session.execute(stmt)).scalar_one_or_none()
        assert row is not None, "Item should exist"
        assert row.status == "failed", f"Expected failed, got {row.status}"
        assert row.error_message, "Should have error_message"
        assert row.retry_count >= 1


@pytest.mark.asyncio
async def test_worker_re_queues_retryable_failures(async_client):
    """Item with retry_count < max_retries gets re-queued to 'pending' after execution failure."""
    async with async_session_factory() as session:
        platform = await _get_or_create_mockfail_platform(session)
        product = await _create_product(session)
        task = await _create_listing_task(session, product.id, platform.id)
        item = ListingTaskItem(
            task_id=task.id,
            product_id=product.id,
            platform_id=platform.id,
            status="failed",
            retry_count=1,
            error_message="transient",
            executed_at=datetime.now(timezone.utc) - timedelta(seconds=300),
        )
        session.add(item)
        await session.flush()
        await session.commit()
        item_id = item.id

    from app.listing.worker import ListingWorker

    worker = ListingWorker(poll_interval=1.0, max_retries=3, retry_delay_seconds=0)
    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with async_session_factory() as session:
        stmt = select(ListingTaskItem).where(ListingTaskItem.id == item_id)
        row = (await session.execute(stmt)).scalar_one_or_none()
        assert row is not None, "Item should exist"
        assert row.status == "pending", f"Expected pending, got {row.status}"
        assert row.retry_count == 2, f"Expected retry_count=2, got {row.retry_count}"


@pytest.mark.asyncio
async def test_worker_marks_item_failed_platform_not_found(async_client):
    """Item with non-existent platform_id gets status='failed' with appropriate error."""
    from sqlalchemy import text

    async with async_session_factory() as session:
        product = await _create_product(session)
        platform = await _create_platform(session)
        task = await _create_listing_task(session, product.id, platform.id)
        # Bypass FK checks to insert an item referencing a non-existent platform
        await session.execute(text("SET session_replication_role = replica;"))
        result = await session.execute(
            text("""
                INSERT INTO listing_task_item (task_id, product_id, platform_id, status)
                VALUES (:task_id, :product_id, :platform_id, :status)
                RETURNING id
            """),
            {
                "task_id": task.id,
                "product_id": product.id,
                "platform_id": 99999,
                "status": "pending",
            },
        )
        await session.execute(text("SET session_replication_role = origin;"))
        item_id = result.scalar()
        await session.commit()

    from app.listing.worker import ListingWorker

    worker = ListingWorker(poll_interval=0.05)
    try:
        await worker.start()
        await asyncio.sleep(0.2)
    finally:
        await worker.stop()

    async with async_session_factory() as session:
        stmt = select(ListingTaskItem).where(ListingTaskItem.id == item_id)
        row = (await session.execute(stmt)).scalar_one_or_none()
        assert row is not None, "Item should exist"
        assert row.status == "failed", f"Expected failed, got {row.status}"
        assert row.error_message, "Should have error_message"
        assert "Platform or Product not found" in row.error_message
        assert row.retry_count >= 1
