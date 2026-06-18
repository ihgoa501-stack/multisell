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
