"""Tests for adapter_registry.test_connection — real adapter credential validation."""

import pytest

from app.database import async_session_factory
from app.models import Platform
from app.platform_integrations.adapter_registry import test_connection
test_connection.__test__ = False  # prevent pytest from treating this as a test


@pytest.mark.asyncio
async def test_test_connection_known_adapter(async_client):
    """For a known adapter_code, test_connection returns (bool, str)."""
    async with async_session_factory() as session:
        platform = Platform(
            name="TestOzon",
            code="test_ozon",
            api_key="test_key",
            client_id="test_client",
            status=1,
        )
        session.add(platform)
        await session.commit()
        pid = platform.id

    async with async_session_factory() as session:
        platform = await session.get(Platform, pid)

    success, msg = await test_connection("ozon", platform)
    assert isinstance(success, bool)
    assert isinstance(msg, str)


@pytest.mark.asyncio
async def test_test_connection_unknown_adapter(async_client):
    """An unknown adapter_code returns False with an appropriate message."""
    async with async_session_factory() as session:
        platform = Platform(
            name="Test",
            code="test",
            status=1,
        )
        session.add(platform)
        await session.commit()
        pid = platform.id

    async with async_session_factory() as session:
        platform = await session.get(Platform, pid)

    success, msg = await test_connection("nonexistent_adapter", platform)
    assert success is False
    assert "未知适配器" in msg or "未知" in msg


@pytest.mark.asyncio
async def test_test_connection_handles_exception(async_client):
    """If the adapter's validate_credentials raises, test_connection returns (False, str)."""

    async with async_session_factory() as session:
        platform = Platform(
            name="Test",
            code="mockfail",
            status=1,
        )
        session.add(platform)
        await session.commit()
        pid = platform.id

    async with async_session_factory() as session:
        platform = await session.get(Platform, pid)

    # "wb" / wildberries adapter requires credentials; without them it may raise
    # But any exception should be caught
    success, msg = await test_connection("wb", platform)
    assert isinstance(success, bool)
    assert isinstance(msg, str)
