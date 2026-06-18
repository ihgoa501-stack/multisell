"""AgentOS Phase 1 测试 — 归一化逻辑 + API 契约测试"""

from datetime import datetime, timezone
from types import SimpleNamespace

import pytest
from sqlalchemy import select

from app.agent.models import AgentAction
from app.agentos.schemas import (
    AgentOSWorkItem,
    AutonomyLevel,
    RiskLevel,
    WorkItemPriority,
    WorkItemStatus,
)
from app.agentos.service import (
    AGENT_SQUADS,
    AgentOSService,
    TEMPLATE_CARDS,
)


# ─── 归一化纯函数测试 ───────────────────────────────────────


class TestNormalizeAgentPendingAction:
    def test_basic_normalization(self):
        row = SimpleNamespace(
            id=11,
            agent_id="A5",
            decision_id=7,
            action_type="replenish",
            status="pending",
            summary="SKU-001 建议补货 120 件",
            action_payload={"sku_id": 1, "amount": 120},
            created_at=datetime(2026, 6, 18, 8, 0, tzinfo=timezone.utc),
            updated_at=datetime(2026, 6, 18, 8, 0, tzinfo=timezone.utc),
        )
        item = AgentOSService.normalize_agent_pending_action(row)

        assert isinstance(item, AgentOSWorkItem)
        assert item.id == "agent_action:11"
        assert item.source_type == "agent_action"
        assert item.source_id == "11"
        assert item.squad_id == "fulfillment"
        assert item.agent_id == "A5"
        assert item.title == "SKU-001 建议补货 120 件"
        assert item.risk_level == RiskLevel.HIGH
        assert item.requires_approval is True
        assert item.status == WorkItemStatus.PENDING
        assert item.action_url == "/agents/A5"

    def test_high_risk_amount(self):
        row = SimpleNamespace(
            id=12,
            agent_id="A5",
            decision_id=8,
            action_type="replenish",
            status="pending",
            summary="大额补货",
            action_payload={"amount": 1000},
            created_at=None,
            updated_at=None,
        )
        item = AgentOSService.normalize_agent_pending_action(row)
        assert item.risk_level == RiskLevel.CRITICAL
        assert item.priority == WorkItemPriority.CRITICAL
        assert item.requires_approval is True

    def test_low_risk_action(self):
        row = SimpleNamespace(
            id=13,
            agent_id="A3",
            decision_id=9,
            action_type="notify",
            status="pending",
            summary="通知: 广告预算接近上限",
            action_payload={},
            created_at=None,
            updated_at=None,
        )
        item = AgentOSService.normalize_agent_pending_action(row)
        assert item.risk_level == RiskLevel.MEDIUM
        assert item.requires_approval is False


class TestNormalizeException:
    def test_critical_exception(self):
        row = SimpleNamespace(
            id=21,
            source_module="settlement",
            source_type="settlement_item",
            source_id=9,
            severity="critical",
            status="open",
            title="结算金额差异",
            description="平台结算与订单金额不一致",
            recommended_action="检查平台费用规则",
            created_at=datetime(2026, 6, 18, 9, 0, tzinfo=timezone.utc),
            updated_at=datetime(2026, 6, 18, 9, 0, tzinfo=timezone.utc),
        )
        item = AgentOSService.normalize_exception(row)

        assert item.id == "exception:21"
        assert item.source_type == "exception"
        assert item.squad_id == "risk"
        assert item.risk_level == RiskLevel.CRITICAL
        assert item.requires_approval is False
        assert item.status == WorkItemStatus.PENDING
        assert item.action_url == "/exceptions/21"

    def test_warning_exception(self):
        row = SimpleNamespace(
            id=22,
            source_module="shipping",
            source_type="shipping_delay",
            source_id=10,
            severity="warning",
            status="open",
            title="物流延迟",
            description="",
            recommended_action="",
            created_at=None,
            updated_at=None,
        )
        item = AgentOSService.normalize_exception(row)
        assert item.risk_level == RiskLevel.MEDIUM
        assert item.priority == WorkItemPriority.MEDIUM


class TestNormalizeNotification:
    def test_inventory_alert(self):
        row = SimpleNamespace(
            id=31,
            alert_type="inventory_low_stock",
            title="SKU 2 库存预警",
            content="当前库存 3，安全库存 10",
            link_url="/inventory/2",
            severity="warning",
            is_read=0,
            source_id="sku=2",
            created_at=datetime(2026, 6, 18, 10, 0, tzinfo=timezone.utc),
        )
        item = AgentOSService.normalize_notification(row)

        assert item.id == "notification:31"
        assert item.source_type == "notification"
        assert item.squad_id == "fulfillment"
        assert item.risk_level == RiskLevel.MEDIUM
        assert item.status == WorkItemStatus.PENDING
        assert item.action_url == "/inventory/2"
        assert item.agent_id == "A5"

    def test_read_notification(self):
        row = SimpleNamespace(
            id=32,
            alert_type="listing_failed",
            title="刊登失败",
            content="",
            link_url="",
            severity="error",
            is_read=1,
            source_id="",
            created_at=None,
        )
        item = AgentOSService.normalize_notification(row)
        assert item.status == WorkItemStatus.COMPLETED
        assert item.agent_id == "A2"


class TestNormalizeListingTask:
    def test_blocked_task(self):
        row = SimpleNamespace(
            id=41,
            product_id=1,
            platform_id=1,
            status="blocked",
            last_error="缺少商品描述",
            created_at=datetime(2026, 6, 18, 11, 0, tzinfo=timezone.utc),
            updated_at=datetime(2026, 6, 18, 11, 0, tzinfo=timezone.utc),
        )
        item = AgentOSService.normalize_listing_task(row)
        assert item.source_type == "listing_task"
        assert item.squad_id == "growth"
        assert item.agent_id == "A2"
        assert item.status == WorkItemStatus.BLOCKED
        assert item.risk_level == RiskLevel.HIGH
        assert item.requires_approval is True

    def test_ready_task(self):
        row = SimpleNamespace(
            id=42,
            product_id=2,
            platform_id=2,
            status="ready",
            last_error="",
            created_at=None,
            updated_at=None,
        )
        item = AgentOSService.normalize_listing_task(row)
        assert item.status == WorkItemStatus.PENDING
        assert item.risk_level == RiskLevel.MEDIUM


class TestConstants:
    def test_squad_ids(self):
        ids = [s["id"] for s in AGENT_SQUADS]
        assert ids == ["growth", "fulfillment", "risk"]

    def test_template_ids(self):
        template_ids = {t["id"] for t in TEMPLATE_CARDS}
        assert "pre_listing_decision" in template_ids
        assert "listing_optimization" in template_ids
        assert "inventory_replenishment" in template_ids
        assert "profit_risk" in template_ids


# ─── API 契约测试（需数据库） ──────────────────────────────


@pytest.mark.usefixtures("prepare_db")
class TestControlCenterAPI:
    """总控台接口测试"""

    async def test_control_center_structure(self, async_client):
        resp = await async_client.get("/api/agentos/control-center")
        assert resp.status_code == 200
        data = resp.json()

        assert data["code"] == 200
        result = data["data"]
        assert "overview" in result
        assert "squads" in result
        assert "priority_work_items" in result
        assert "metrics" in result
        assert "recent_activity" in result

        overview = result["overview"]
        assert "health_score" in overview
        assert "active_agents" in overview
        assert overview["active_agents"] == 10  # 静态定义
        assert "pending_approvals" in overview
        assert "critical_items" in overview

    async def test_control_center_returns_squads(self, async_client):
        resp = await async_client.get("/api/agentos/control-center")
        assert resp.status_code == 200
        squads = resp.json()["data"]["squads"]
        assert len(squads) == 3
        squad_ids = {s["id"] for s in squads}
        assert squad_ids == {"growth", "fulfillment", "risk"}


class TestWorkItemsAPI:
    """任务中心接口测试"""

    async def test_work_items_basic(self, async_client):
        resp = await async_client.get("/api/agentos/work-items")
        assert resp.status_code == 200
        data = resp.json()

        assert data["code"] == 200
        assert "records" in data
        assert "total" in data
        assert isinstance(data["records"], list)

    async def test_work_items_with_filters(self, async_client):
        # 按 squad 筛选
        resp = await async_client.get("/api/agentos/work-items?squad=growth")
        assert resp.status_code == 200
        data = resp.json()
        for item in data["records"]:
            assert item["squad_id"] == "growth"

        # 按 status 筛选
        resp = await async_client.get("/api/agentos/work-items?status=pending")
        assert resp.status_code == 200

        # 按 priority 筛选
        resp = await async_client.get("/api/agentos/work-items?priority=high")
        assert resp.status_code == 200

        # 按 requires_approval 筛选
        resp = await async_client.get("/api/agentos/work-items?requires_approval=true")
        assert resp.status_code == 200

    async def test_work_items_pagination(self, async_client):
        resp = await async_client.get("/api/agentos/work-items?limit=5&offset=0")
        assert resp.status_code == 200
        data = resp.json()
        assert len(data["records"]) <= 5
        assert data["page_size"] == 5


class TestSquadsAPI:
    """团队页接口测试"""

    async def test_squads_structure(self, async_client):
        resp = await async_client.get("/api/agentos/squads")
        assert resp.status_code == 200
        data = resp.json()

        assert data["code"] == 200
        result = data["data"]
        assert "squads" in result
        assert "summary" in result

        squads = result["squads"]
        assert len(squads) == 3
        for squad in squads:
            assert "id" in squad
            assert "name" in squad
            assert "agents" in squad
            assert "health_score" in squad
            assert "risk_level" in squad

    async def test_squad_has_agents(self, async_client):
        resp = await async_client.get("/api/agentos/squads")
        assert resp.status_code == 200
        squads = resp.json()["data"]["squads"]
        growth = [s for s in squads if s["id"] == "growth"][0]
        assert len(growth["agents"]) == 4
        for agent in growth["agents"]:
            assert "id" in agent
            assert "name" in agent
            assert "role" in agent


class TestTemplatesAPI:
    """模板接口测试"""

    async def test_templates_structure(self, async_client):
        resp = await async_client.get("/api/agentos/templates")
        assert resp.status_code == 200
        data = resp.json()

        assert data["code"] == 200
        templates = data["data"]["templates"]
        assert len(templates) >= 4
        for tmpl in templates:
            assert "id" in tmpl
            assert "title" in tmpl
            assert "route" in tmpl


class TestAuthAndFallback:
    """权限和降级测试"""

    async def test_endpoints_accessible_when_auth_disabled(self, async_client):
        """AUTH_ENABLED=False 时接口可访问"""
        endpoints = [
            "/api/agentos/control-center",
            "/api/agentos/work-items",
            "/api/agentos/squads",
            "/api/agentos/templates",
        ]
        for ep in endpoints:
            resp = await async_client.get(ep)
            assert resp.status_code == 200, f"{ep} returned {resp.status_code}"


# ─── Phase 2: Mutation 测试 ──────────────────────────────


@pytest.mark.usefixtures("prepare_db")
class TestWorkItemStatusUpdate:
    """WorkItem 状态更新测试"""

    @staticmethod
    async def _seed_exception():
        """插入一条异常记录"""
        from app.database import async_session_factory
        from app.models import ExceptionItem

        async with async_session_factory() as db:
            existing = (await db.execute(
                select(ExceptionItem).where(ExceptionItem.id == 100)
            )).scalar_one_or_none()
            if existing:
                return existing.id
            exc = ExceptionItem(
                id=100,
                source_module="settlement",
                source_type="settlement_item",
                source_id=9,
                severity="critical",
                status="open",
                title="结算金额差异(Phase2测试)",
                description="测试用异常",
            )
            db.add(exc)
            await db.commit()
            return exc.id

    @staticmethod
    async def _seed_notification():
        """插入一条通知记录"""
        from app.database import async_session_factory
        from app.models import Notification

        async with async_session_factory() as db:
            existing = (await db.execute(
                select(Notification).where(Notification.id == 100)
            )).scalar_one_or_none()
            if existing:
                return existing.id
            notif = Notification(
                id=100,
                user_id=1,
                alert_type="inventory_low_stock",
                title="SKU 库存预警(Phase2测试)",
                content="当前库存不足",
                severity="warning",
                is_read=0,
                source_id="sku=100",
            )
            db.add(notif)
            await db.commit()
            return notif.id

    @staticmethod
    async def _seed_agent_action():
        """插入一条 AgentAction 记录"""
        from app.database import async_session_factory

        async with async_session_factory() as db:
            existing = (await db.execute(
                select(AgentAction).where(AgentAction.id == 100)
            )).scalar_one_or_none()
            if existing:
                return existing.id
            action = AgentAction(
                user_id=1,
                agent_id="A5",
                action_type="replenish",
                status="pending",
                summary="SKU-100 建议补货(Phase2测试)",
                action_payload={"amount": 120},
            )
            db.add(action)
            await db.commit()
            return action.id

    async def test_update_exception_status(self, async_client):
        """标记异常为已完成"""
        eid = await self._seed_exception()
        resp = await async_client.patch(
            f"/api/agentos/work-items/exception:{eid}/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["ok"] is True

    async def test_update_notification_status(self, async_client):
        """标记通知为已读"""
        nid = await self._seed_notification()
        resp = await async_client.patch(
            f"/api/agentos/work-items/notification:{nid}/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["ok"] is True

    async def test_update_agent_action_status(self, async_client):
        """标记 Agent 动作为已完成"""
        aid = await self._seed_agent_action()
        resp = await async_client.patch(
            f"/api/agentos/work-items/agent_action:{aid}/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["ok"] is True

    async def test_update_nonexistent_item(self, async_client):
        """更新不存在的 WorkItem 返回 code 404"""
        resp = await async_client.patch(
            "/api/agentos/work-items/exception:99999/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200  # Result always returns HTTP 200
        assert resp.json()["code"] == 404

    async def test_update_invalid_id_format(self, async_client):
        """格式错误返回 code 400"""
        resp = await async_client.patch(
            "/api/agentos/work-items/badformat/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 400

    async def test_update_unknown_source_type(self, async_client):
        """未知 source_type 返回 code 400"""
        resp = await async_client.patch(
            "/api/agentos/work-items/unknown_type:1/status",
            json={"status": "completed"},
        )
        assert resp.status_code == 200
        assert resp.json()["code"] == 400


class TestWorkItemApproval:
    """WorkItem 审批测试"""

    @staticmethod
    async def _seed_agent_action():
        """插入一条 AgentAction 记录（approve/reject 用）"""
        from app.database import async_session_factory

        async with async_session_factory() as db:
            existing = (await db.execute(
                select(AgentAction).where(AgentAction.id == 200)
            )).scalar_one_or_none()
            if existing:
                return existing.id
            action = AgentAction(
                user_id=1,
                agent_id="A5",
                action_type="replenish",
                status="pending",
                summary="SKU-200 建议补货(审批测试)",
                action_payload={"amount": 120},
            )
            db.add(action)
            await db.commit()
            return action.id

    @staticmethod
    async def _seed_exception():
        """插入一条异常记录（approve fallback 用）"""
        from app.database import async_session_factory
        from app.models import ExceptionItem

        async with async_session_factory() as db:
            existing = (await db.execute(
                select(ExceptionItem).where(ExceptionItem.id == 201)
            )).scalar_one_or_none()
            if existing:
                return existing.id
            exc = ExceptionItem(
                id=201,
                source_module="shipping",
                source_type="shipping_delay",
                source_id=10,
                severity="warning",
                status="open",
                title="物流延迟(审批测试)",
                description="测试用异常审批",
            )
            db.add(exc)
            await db.commit()
            return exc.id

    async def test_approve_agent_action(self, async_client):
        """审批通过 Agent 动作"""
        aid = await self._seed_agent_action()
        resp = await async_client.post(
            f"/api/agentos/work-items/agent_action:{aid}/approve",
            json={"action": "approve", "comment": "同意执行"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["ok"] is True
        assert data["data"]["action"] == "approved"

    async def test_reject_agent_action(self, async_client):
        """拒绝 Agent 动作"""
        aid = await self._seed_agent_action()
        resp = await async_client.post(
            f"/api/agentos/work-items/agent_action:{aid}/reject",
            json={"action": "reject", "comment": "风险过高"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["code"] == 200
        assert data["data"]["ok"] is True
        assert data["data"]["action"] == "rejected"

    async def test_approve_nonexistent(self, async_client):
        """审批不存在的 item 返回 code 404"""
        resp = await async_client.post(
            "/api/agentos/work-items/agent_action:99999/approve",
            json={"action": "approve"},
        )
        assert resp.status_code == 200  # Result always returns HTTP 200
        assert resp.json()["code"] == 404

    async def test_approve_exception_fallback(self, async_client):
        """审批异常类型降级到 status update"""
        eid = await self._seed_exception()
        resp = await async_client.post(
            f"/api/agentos/work-items/exception:{eid}/approve",
            json={"action": "approve"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["data"]["ok"] is True
