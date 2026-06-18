from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Any

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.agent.models import AgentAction as PendingAgentAction
from app.agent.models import AgentDecision
from app.models import ExceptionItem, Notification


AGENT_TO_SQUAD = {
    "A1": "growth",
    "A2": "growth",
    "A3": "growth",
    "A4": "growth",
    "A5": "fulfillment",
    "G2": "fulfillment",
    "A6": "risk",
    "A7": "risk",
    "G3": "risk",
    "G1": "risk",
}

MODULE_TO_SQUAD = {
    "product": "growth",
    "listing": "growth",
    "listing_task": "growth",
    "inventory": "fulfillment",
    "order": "fulfillment",
    "shipping": "fulfillment",
    "settlement": "risk",
    "finance": "risk",
    "platform_fee": "risk",
    "compliance": "risk",
}

ALERT_TO_SQUAD = {
    "inventory_low_stock": "fulfillment",
    "inventory_out_of_stock": "fulfillment",
    "order_pending": "fulfillment",
    "listing_failed": "growth",
    "settlement_pending": "risk",
    "settlement_discrepancy": "risk",
}

AGENT_SQUADS = [
    {
        "id": "growth",
        "name": "增长小队",
        "description": "负责选品、Listing、广告建议和上新质量。",
        "agents": ["A1", "A2", "A3", "A4"],
    },
    {
        "id": "fulfillment",
        "name": "履约小队",
        "description": "负责库存、订单、仓储、海关和物流闭环。",
        "agents": ["A5", "G2"],
    },
    {
        "id": "risk",
        "name": "风控小队",
        "description": "负责利润、折扣、合规和平台安全红线。",
        "agents": ["A6", "A7", "G3", "G1"],
    },
]

TEMPLATE_CARDS = [
    {
        "id": "pre_listing_decision",
        "title": "上架前经营决策",
        "squad": "risk",
        "description": "串联商品、物流、平台费和利润，判断是否值得上架。",
        "mode": "Agent",
        "route": "/decisions/prelisting",
    },
    {
        "id": "listing_optimization",
        "title": "Listing 优化",
        "squad": "growth",
        "description": "生成标题、描述、关键词和平台适配建议。",
        "mode": "Agent",
        "route": "/listings/ai-workbench",
    },
    {
        "id": "inventory_replenishment",
        "title": "库存补货",
        "squad": "fulfillment",
        "description": "识别低库存和断货风险，生成补货建议。",
        "mode": "Agent",
        "route": "/inventory/alerts",
    },
    {
        "id": "profit_risk",
        "title": "利润风控",
        "squad": "risk",
        "description": "识别毛利率下降、费用异常和折扣风险。",
        "mode": "Ask",
        "route": "/finance",
    },
    {
        "id": "customer_service_draft",
        "title": "客服草稿",
        "squad": "growth",
        "description": "生成多语言客服回复草稿，不自动回复真实客户。",
        "mode": "Ask",
        "route": "/agents/A4",
    },
]


class AgentOSService:
    @staticmethod
    def _iso(dt: Any) -> Any:
        return dt if not hasattr(dt, "isoformat") else dt

    @staticmethod
    def _squad_for_agent(agent_id: str | None) -> str:
        return AGENT_TO_SQUAD.get(agent_id or "", "risk")

    @staticmethod
    def _risk_from_severity(severity: str | None) -> str:
        mapping = {"critical": "critical", "error": "high", "warning": "medium", "info": "low"}
        return mapping.get(severity or "", "medium")

    @staticmethod
    def _risk_from_action(action_type: str | None, payload: dict[str, Any] | None = None) -> str:
        payload = payload or {}
        amount = abs(float(payload.get("amount") or payload.get("total_amount") or 0))
        sku_count = len(payload.get("sku_codes") or [])
        if amount >= 500 or sku_count > 20:
            return "high"
        if action_type in {"replenish", "price_adjust", "discount_review", "ad_action"}:
            return "medium"
        return "low"

    @staticmethod
    def _approval_required(risk_level: str, status: str) -> bool:
        return status == "pending" and risk_level in {"high", "critical"}

    @staticmethod
    def normalize_agent_pending_action(row: PendingAgentAction) -> dict[str, Any]:
        risk = AgentOSService._risk_from_action(row.action_type, row.action_payload)
        return {
            "id": f"agent_action:{row.id}",
            "source_type": "agent_action",
            "source_id": str(row.id),
            "source_module": "agent",
            "business_object": {"type": "decision", "id": str(row.decision_id) if row.decision_id else None},
            "squad": AgentOSService._squad_for_agent(row.agent_id),
            "agent_id": row.agent_id,
            "title": row.summary,
            "summary": row.summary,
            "recommendation": row.summary,
            "risk_level": risk,
            "approval_required": AgentOSService._approval_required(risk, row.status),
            "status": row.status,
            "action_type": row.action_type,
            "context": row.action_payload or {},
            "audit_link": f"/agents/{row.agent_id}",
            "created_at": AgentOSService._iso(row.created_at),
        }

    @staticmethod
    def normalize_exception(row: ExceptionItem) -> dict[str, Any]:
        squad = MODULE_TO_SQUAD.get(row.source_module or "", "risk")
        risk = AgentOSService._risk_from_severity(row.severity)
        return {
            "id": f"exception:{row.id}",
            "source_type": "exception",
            "source_id": str(row.id),
            "source_module": row.source_module,
            "business_object": {"type": row.source_type, "id": str(row.source_id) if row.source_id else None},
            "squad": squad,
            "agent_id": None,
            "title": row.title,
            "summary": row.description,
            "recommendation": row.recommended_action,
            "risk_level": risk,
            "approval_required": False,
            "status": row.status,
            "action_type": None,
            "context": {},
            "audit_link": f"/exceptions",
            "created_at": AgentOSService._iso(row.created_at),
        }

    @staticmethod
    def normalize_notification(row: Notification) -> dict[str, Any]:
        risk = AgentOSService._risk_from_severity(row.severity)
        return {
            "id": f"notification:{row.id}",
            "source_type": "notification",
            "source_id": str(row.id),
            "source_module": row.alert_type,
            "business_object": {"type": row.alert_type, "id": row.source_id},
            "squad": ALERT_TO_SQUAD.get(row.alert_type, "risk"),
            "agent_id": None,
            "title": row.title,
            "summary": row.content,
            "recommendation": "查看并处理该预警",
            "risk_level": risk,
            "approval_required": False,
            "status": "read" if row.is_read else "unread",
            "action_type": None,
            "context": {"link_url": row.link_url},
            "audit_link": row.link_url or "/notifications",
            "created_at": AgentOSService._iso(row.created_at),
        }

    @staticmethod
    def _sort_key(item: dict[str, Any]) -> datetime:
        value = item.get("created_at")
        if isinstance(value, datetime):
            return value
        return datetime.min.replace(tzinfo=timezone.utc)

    @staticmethod
    async def list_work_items(
        db: AsyncSession,
        user_id: int,
        source_type: str | None = None,
        squad: str | None = None,
        status: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[dict[str, Any]], int]:
        items: list[dict[str, Any]] = []

        if source_type in (None, "agent_action"):
            action_stmt = select(PendingAgentAction).where(PendingAgentAction.user_id == user_id)
            if status:
                action_stmt = action_stmt.where(PendingAgentAction.status == status)
            action_result = await db.execute(action_stmt.order_by(PendingAgentAction.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_agent_pending_action(row) for row in action_result.scalars().all())

        if source_type in (None, "exception"):
            exception_stmt = select(ExceptionItem)
            if status:
                exception_stmt = exception_stmt.where(ExceptionItem.status == status)
            exception_result = await db.execute(exception_stmt.order_by(ExceptionItem.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_exception(row) for row in exception_result.scalars().all())

        if source_type in (None, "notification"):
            notification_stmt = select(Notification).where(Notification.user_id == user_id)
            if status == "unread":
                notification_stmt = notification_stmt.where(Notification.is_read == 0)
            notification_result = await db.execute(notification_stmt.order_by(Notification.created_at.desc()).limit(100))
            items.extend(AgentOSService.normalize_notification(row) for row in notification_result.scalars().all())

        if squad:
            items = [item for item in items if item["squad"] == squad]

        items.sort(key=AgentOSService._sort_key, reverse=True)
        total = len(items)
        start = (page - 1) * page_size
        return items[start:start + page_size], total

    @staticmethod
    async def get_squads(db: AsyncSession, user_id: int) -> list[dict[str, Any]]:
        seven_days_ago = datetime.now(timezone.utc) - timedelta(days=7)
        squads = [dict(squad) for squad in AGENT_SQUADS]

        for squad in squads:
            agents = squad["agents"]
            decisions = await db.scalar(
                select(func.count()).select_from(AgentDecision).where(
                    AgentDecision.user_id == user_id,
                    AgentDecision.agent_id.in_(agents),
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            accepted = await db.scalar(
                select(func.count()).select_from(AgentDecision).where(
                    AgentDecision.user_id == user_id,
                    AgentDecision.agent_id.in_(agents),
                    AgentDecision.user_action == "accepted",
                    AgentDecision.created_at >= seven_days_ago,
                )
            ) or 0
            pending = await db.scalar(
                select(func.count()).select_from(PendingAgentAction).where(
                    PendingAgentAction.user_id == user_id,
                    PendingAgentAction.agent_id.in_(agents),
                    PendingAgentAction.status == "pending",
                )
            ) or 0
            squad["decision_count_7d"] = int(decisions)
            squad["pending_approvals"] = int(pending)
            squad["risk_count"] = int(pending)
            squad["adoption_rate"] = round(accepted / decisions, 3) if decisions else 0
            squad["autonomy_level"] = "semi_autonomous" if pending else "suggestion"

        return squads

    @staticmethod
    async def get_summary(db: AsyncSession, user_id: int) -> dict[str, Any]:
        work_items, total = await AgentOSService.list_work_items(db, user_id, page=1, page_size=200)
        pending_approvals = sum(1 for item in work_items if item["approval_required"])
        inventory_risks = sum(1 for item in work_items if item["squad"] == "fulfillment")
        executed = sum(1 for item in work_items if item["status"] in {"executed", "read", "resolved"})
        return {
            "sales_today": 0,
            "profit_today": 0,
            "inventory_risks": inventory_risks,
            "pending_approvals": pending_approvals,
            "active_work_items": total,
            "agent_automation_rate": round(executed / total, 3) if total else 0,
        }

    @staticmethod
    async def get_control_center(db: AsyncSession, user_id: int) -> dict[str, Any]:
        work_items, _ = await AgentOSService.list_work_items(db, user_id, page=1, page_size=8)
        return {
            "summary": await AgentOSService.get_summary(db, user_id),
            "work_items": work_items,
            "squads": await AgentOSService.get_squads(db, user_id),
            "templates": TEMPLATE_CARDS,
        }
