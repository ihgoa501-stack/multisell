"""Tests for the background listing task worker."""

import asyncio
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


async def _create_listing_task(session, product_id: int, platform_id: int) -> ListingTask:
    lt = ListingTask(
        product_id=product_id,
        platform_id=platform_id,
        source_type="test",
        status="ready",
    )
    session.add(lt)
    await session.flush()
    return lt


async def _create_pending_item(session, task_id: int, product_id: int, platform_id: int) -> ListingTaskItem:
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
        assert row.retry_count >= 1
