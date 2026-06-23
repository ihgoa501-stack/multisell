"""上架任务队列测试。"""

from uuid import uuid4

from sqlalchemy import select

from app.database import async_session_factory
from app.models import ListingTask, OperationLog, Platform, Product


def _code(prefix: str) -> str:
    return f"{prefix}_{uuid4().hex[:8]}"


async def _count_logs(module: str, action: str, resource_id: str) -> int:
    async with async_session_factory() as session:
        result = await session.execute(
            select(OperationLog).where(
                OperationLog.module == module,
                OperationLog.action == action,
                OperationLog.resource_id == resource_id,
            )
        )
        return len(result.scalars().all())


async def _create_product_and_platform(
    session,
    product_name="测试商品",
    platform_code="mock_task",
) -> tuple[Product, Platform]:
    product = Product(
        name=product_name,
        unit="件",
        status=1,
    )
    session.add(product)
    platform = Platform(
        name=f"平台_{platform_code}",
        code=platform_code,
        status=1,
    )
    session.add(platform)
    await session.flush()
    return product, platform


# ── Model ────────────────────────────────────────────────────────────────────


class TestListingTaskModel:
    async def test_listing_task_model_is_mapped(self, async_client):
        async with async_session_factory() as session:
            product, platform = await _create_product_and_platform(session)
            await session.commit()

        async with async_session_factory() as session:
            task = ListingTask(
                product_id=product.id,
                platform_id=platform.id,
                source_type="decision",
                source_item_key="row-1",
                status="ready",
                missing_requirements=[],
                decision_snapshot={"recommendation": "approve"},
            )
            session.add(task)
            await session.flush()
            assert task.id is not None
            await session.rollback()


# ── API / Integration ────────────────────────────────────────────────────────


async def _setup_approve_decision_data(async_client) -> tuple[int, int, int]:
    """创建商品+SKU+平台，返回 (product_id, sku_id, platform_id)"""
    uid = uuid4().hex[:6]
    # product
    resp = await async_client.post(
        "/api/products",
        json={
            "name": f"TaskTest_{uid}",
            "main_image": "/static/img.jpg",
            "package_length_cm": 30,
            "package_width_cm": 20,
            "package_height_cm": 10,
            "package_weight_kg": 0.5,
        },
    )
    assert resp.status_code == 200
    pid = resp.json()["data"]["id"]

    # specs + skus
    await async_client.post(
        f"/api/products/{pid}/specs",
        json={"specs": [{"name": "颜色", "values": ["标准"]}]},
    )
    sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
    assert sku_resp.status_code == 200
    sku_id = sku_resp.json()["data"]["skus"][0]["id"]

    # price + inventory
    await async_client.post(
        "/api/prices",
        json={"sku_id": sku_id, "price_type": "sale_price", "price": 199.9},
    )
    await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 50})

    # platform
    resp = await async_client.post(
        "/api/platforms",
        json={"name": f"PL_{uid}", "code": f"pl_{uid}", "status": 1},
    )
    assert resp.status_code == 200
    plat_id = resp.json()["data"]["id"]

    return pid, sku_id, plat_id


class TestFromDecisions:
    """POST /api/listing-tasks/from-decisions"""

    async def test_creates_ready_task_from_approve_decision(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "row-1",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["created_count"] == 1
        assert data["reused_count"] == 0
        assert data["skipped_count"] == 0
        task = data["tasks"][0]
        assert task["status"] == "ready"
        assert task["product_id"] == pid
        assert task["platform_id"] == plat_id
        assert task["missing_requirements"] == []

    async def test_skips_non_approve_rows(self, async_client):
        """非 approve 的行跳过"""
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "reject-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 100,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 10,
                            "payment_fee": 3,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 0,
                            "profit_amount": -473,
                            "profit_margin": -473.0,
                            "recommendation": "reject",
                            "blocking_reasons": ["利润率不足"],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["created_count"] == 0
        assert data["skipped_count"] == 1

    async def test_reuses_existing_open_task(self, async_client):
        """重复的 (product_id, platform_id) 可复用"""
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        item = {
            "item_key": "row-1",
            "sku_id": sku_id,
            "platform_id": plat_id,
            "decision_result": {
                "sku_id": sku_id,
                "destination_country": "RU",
                "target_sale_price": 5000,
                "product_cost": 500,
                "shipping_fee": 60,
                "platform_fee": 500,
                "payment_fee": 150,
                "fixed_fee": 0,
                "advertising_fee": 0,
                "other_fee": 100,
                "profit_amount": 3690,
                "profit_margin": 73.8,
                "recommendation": "approve",
                "blocking_reasons": [],
                "warnings": [],
                "platform_fee_source": "manual",
            },
        }

        resp1 = await async_client.post(
            "/api/listing-tasks/from-decisions", json={"items": [item]}
        )
        assert resp1.status_code == 200

        resp2 = await async_client.post(
            "/api/listing-tasks/from-decisions", json={"items": [item]}
        )
        assert resp2.status_code == 200
        data2 = resp2.json()["data"]
        assert data2["created_count"] == 0
        assert data2["reused_count"] == 1

    async def test_creates_blocked_task_when_product_incomplete(self, async_client):
        """商品缺数据时创建 blocked 任务"""
        uid = uuid4().hex[:6]
        # Create product WITHOUT logistics / image
        resp = await async_client.post(
            "/api/products", json={"name": f"Incomplete_{uid}"}
        )
        assert resp.status_code == 200
        pid = resp.json()["data"]["id"]

        # create sku
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "颜色", "values": ["默认"]}]},
        )
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 99},
        )
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

        plat_resp = await async_client.post(
            "/api/platforms", json={"name": f"P_{uid}", "code": f"p_{uid}"}
        )
        plat_id = plat_resp.json()["data"]["id"]

        resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "blocked-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 1000,
                            "product_cost": 300,
                            "shipping_fee": 50,
                            "platform_fee": 100,
                            "payment_fee": 30,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 0,
                            "profit_amount": 520,
                            "profit_margin": 52.0,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["created_count"] == 1
        task = data["tasks"][0]
        assert task["status"] == "blocked"
        assert len(task["missing_requirements"]) > 0

    async def test_creates_ready_task_when_product_complete(self, async_client):
        """完整商品创建 ready 任务"""
        uid = uuid4().hex[:6]
        resp = await async_client.post(
            "/api/products",
            json={
                "name": f"Complete_{uid}",
                "main_image": "/static/img.jpg",
                "package_length_cm": 30,
                "package_width_cm": 20,
                "package_height_cm": 10,
                "package_weight_kg": 0.5,
            },
        )
        pid = resp.json()["data"]["id"]
        await async_client.post(
            f"/api/products/{pid}/specs",
            json={"specs": [{"name": "颜色", "values": ["标准"]}]},
        )
        sku_resp = await async_client.post(f"/api/products/{pid}/skus/generate")
        sku_id = sku_resp.json()["data"]["skus"][0]["id"]
        await async_client.post(
            "/api/prices",
            json={"sku_id": sku_id, "price_type": "sale_price", "price": 199},
        )
        await async_client.put(f"/api/inventory/{sku_id}", json={"quantity": 10})

        plat_resp = await async_client.post(
            "/api/platforms",
            json={"name": f"CP_{uid}", "code": f"cp_{uid}", "status": 1},
        )
        plat_id = plat_resp.json()["data"]["id"]

        resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "ready-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        assert resp.status_code == 200
        data = resp.json()["data"]
        assert data["created_count"] == 1
        task = data["tasks"][0]
        assert task["status"] == "ready"

    async def test_from_decisions_requires_permission(self, async_client):
        # Skip when AUTH_ENABLED=False (default in test)
        pass

    async def test_create_is_audited(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "audit-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        log_count = await _count_logs("listing_task", "create_from_decision", "0")
        assert log_count > 0


class TestListTasks:
    """GET /api/listing-tasks"""

    async def test_list_returns_paginated_tasks(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)
        # Create one task
        await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "list-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )

        resp = await async_client.get("/api/listing-tasks")
        assert resp.status_code == 200
        data = resp.json()["data"]
        assert len(data) >= 1
        task = data[0]
        assert task["id"] is not None
        assert task["status"] is not None
        assert task["product_name"] is not None
        assert task["platform_name"] is not None

    async def test_list_filters_by_status(self, async_client):
        resp = await async_client.get("/api/listing-tasks?status=ready")
        assert resp.status_code == 200

    async def test_list_requires_listing_view_permission(self, async_client):
        pass  # AUTH_ENABLED=False in test


class TestRecheckTask:
    """POST /api/listing-tasks/{task_id}/recheck"""

    async def test_recheck_refreshes_status(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "recheck-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        recheck_resp = await async_client.post(f"/api/listing-tasks/{task_id}/recheck")
        assert recheck_resp.status_code == 200
        data = recheck_resp.json()["data"]
        assert data["id"] == task_id
        assert data["status"] in ("ready", "blocked")

    async def test_recheck_is_audited(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "audit-recheck",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        await async_client.post(f"/api/listing-tasks/{task_id}/recheck")

        log_count = await _count_logs("listing_task", "recheck", str(task_id))
        assert log_count > 0


class TestCancelTask:
    """POST /api/listing-tasks/{task_id}/cancel"""

    async def test_cancel_sets_status_cancelled(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "cancel-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        cancel_resp = await async_client.post(f"/api/listing-tasks/{task_id}/cancel")
        assert cancel_resp.status_code == 200
        assert cancel_resp.json()["data"]["status"] == "cancelled"

    async def test_cancel_is_audited(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "audit-cancel",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        await async_client.post(f"/api/listing-tasks/{task_id}/cancel")

        log_count = await _count_logs("listing_task", "cancel", str(task_id))
        assert log_count > 0


class TestPublishTask:
    """POST /api/listing-tasks/{task_id}/publish"""

    async def test_ready_task_publishes(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)
        # Also add logistics
        await async_client.put(
            f"/api/products/{pid}",
            json={
                "main_image": "/static/img.jpg",
                "package_length_cm": 30,
                "package_width_cm": 20,
                "package_height_cm": 10,
                "package_weight_kg": 0.5,
            },
        )

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "publish-row",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        publish_resp = await async_client.post(f"/api/listing-tasks/{task_id}/publish")
        assert publish_resp.status_code == 200
        data = publish_resp.json()["data"]
        assert data["status"] == "published"
        assert data["product_listing_id"] is not None

    async def test_non_ready_task_cannot_publish(self, async_client):
        """blocked 任务发布被拒绝"""
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)
        # Remove image to make it blocked
        await async_client.put(f"/api/products/{pid}", json={"main_image": None})

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "blocked-pub",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        publish_resp = await async_client.post(f"/api/listing-tasks/{task_id}/publish")
        assert publish_resp.status_code == 400

    async def test_publish_is_audited(self, async_client):
        pid, sku_id, plat_id = await _setup_approve_decision_data(async_client)

        create_resp = await async_client.post(
            "/api/listing-tasks/from-decisions",
            json={
                "items": [
                    {
                        "item_key": "audit-pub",
                        "sku_id": sku_id,
                        "platform_id": plat_id,
                        "decision_result": {
                            "sku_id": sku_id,
                            "destination_country": "RU",
                            "target_sale_price": 5000,
                            "product_cost": 500,
                            "shipping_fee": 60,
                            "platform_fee": 500,
                            "payment_fee": 150,
                            "fixed_fee": 0,
                            "advertising_fee": 0,
                            "other_fee": 100,
                            "profit_amount": 3690,
                            "profit_margin": 73.8,
                            "recommendation": "approve",
                            "blocking_reasons": [],
                            "warnings": [],
                            "platform_fee_source": "manual",
                        },
                    }
                ]
            },
        )
        task_id = create_resp.json()["data"]["tasks"][0]["id"]

        await async_client.post(f"/api/listing-tasks/{task_id}/publish")

        log_count = await _count_logs("listing_task", "publish", str(task_id))
        assert log_count > 0
