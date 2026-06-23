"""Agent 操作执行服务

每个 Agent 决策可以产生可执行操作（action），
用户确认后系统自动执行对应业务操作。
"""

import logging
from datetime import datetime, timezone
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import AgentAction

logger = logging.getLogger(__name__)


# ── 各 Agent 的动作提取器 ──────────────────────────────────────


def extract_actions(agent_id: str, decision_id: int, decision: dict) -> list[dict]:
    """从 Agent 决策输出中提取可执行操作"""
    actions = []

    if agent_id == "A5":
        stock_status = decision.get("stock_status", "")
        if stock_status == "red":
            actions.append(
                {
                    "action_type": "replenish",
                    "summary": (
                        f"紧急补货：SKU {decision.get('sku_code', '')} "
                        f"可售仅 {decision.get('sellable_days', '?')} 天，"
                        f"建议补货 {decision.get('suggested_replenish_qty', '?')} 件"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "suggested_qty": decision.get("suggested_replenish_qty", 0),
                        "urgency": "urgent",
                        "suggested_logistics": decision.get("suggested_logistics", ""),
                    },
                }
            )
        elif stock_status == "yellow":
            actions.append(
                {
                    "action_type": "replenish",
                    "summary": (
                        f"补货建议：SKU {decision.get('sku_code', '')} "
                        f"可售 {decision.get('sellable_days', '?')} 天，"
                        f"建议补货 {decision.get('suggested_replenish_qty', '?')} 件"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "suggested_qty": decision.get("suggested_replenish_qty", 0),
                        "urgency": "normal",
                    },
                }
            )

    elif agent_id == "G3":
        action = decision.get("action", "")
        if action == "block":
            actions.append(
                {
                    "action_type": "discount_review",
                    "summary": (
                        f"折扣已阻断：SKU {decision.get('sku_code', '')} "
                        f"折后价 ¥{decision.get('final_price', '?')} < 成本 ¥{decision.get('cost_price', '?')}，"
                        f"请确认是否仍需执行该促销"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "final_price": decision.get("final_price"),
                        "cost_price": decision.get("cost_price"),
                        "gross_profit": decision.get("gross_profit"),
                    },
                }
            )
        elif action == "warn":
            actions.append(
                {
                    "action_type": "discount_review",
                    "summary": (
                        f"折扣需审核：SKU {decision.get('sku_code', '')} "
                        f"折后毛利率 {decision.get('gross_margin', '?')}%，低于安全线"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "gross_margin": decision.get("gross_margin"),
                        "reason": decision.get("reason", ""),
                    },
                }
            )

    elif agent_id == "A6":
        is_loss = decision.get("is_loss", False)
        if is_loss:
            actions.append(
                {
                    "action_type": "price_review",
                    "summary": (
                        f"亏损 SKU 待处理：{decision.get('sku_code', '')} "
                        f"单件亏损 ¥{abs(decision.get('profit_per_unit', 0)):.2f}，"
                        f"建议提价或优化成本"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "profit_per_unit": decision.get("profit_per_unit"),
                        "cost_price": decision.get("cost_price"),
                        "selling_price": decision.get("selling_price"),
                        "suggestions": decision.get("optimization_suggestions", []),
                    },
                }
            )
        elif decision.get("below_threshold", False):
            actions.append(
                {
                    "action_type": "price_review",
                    "summary": (
                        f"低毛利 SKU 待处理：{decision.get('sku_code', '')} "
                        f"毛利率 {decision.get('gross_margin', '?')}%，低于阈值"
                    ),
                    "action_payload": {
                        "sku_code": decision.get("sku_code", ""),
                        "gross_margin": decision.get("gross_margin"),
                        "threshold": decision.get("min_margin_threshold"),
                    },
                }
            )

    elif agent_id == "A3":
        status = decision.get("status", "")
        if status == "critical":
            actions.append(
                {
                    "action_type": "ad_action",
                    "summary": (
                        f"广告严重亏损：活动 {decision.get('campaign_id', '')} "
                        f"ACoS {decision.get('metrics', {}).get('acos', '?')}%，"
                        f"已超过毛利率，建议暂停"
                    ),
                    "action_payload": {
                        "campaign_id": decision.get("campaign_id", ""),
                        "acos": decision.get("metrics", {}).get("acos"),
                        "suggestions": decision.get("suggestions", []),
                    },
                }
            )
        elif status == "warning":
            actions.append(
                {
                    "action_type": "ad_action",
                    "summary": (
                        f"广告需优化：活动 {decision.get('campaign_id', '')} "
                        f"ACoS {decision.get('metrics', {}).get('acos', '?')}% 偏高，"
                        f"建议调整出价"
                    ),
                    "action_payload": {
                        "campaign_id": decision.get("campaign_id", ""),
                        "acos": decision.get("metrics", {}).get("acos"),
                        "bid_suggestion": decision.get("bid_suggestion"),
                        "suggestions": decision.get("suggestions", []),
                    },
                }
            )

    return actions


class AgentActionService:
    """Agent 操作执行服务"""

    @staticmethod
    async def create_actions(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_id: int,
        decision: dict,
    ) -> list[AgentAction]:
        """从决策中提取并创建待执行操作"""
        action_defs = extract_actions(agent_id, decision_id, decision)
        created = []
        for ad in action_defs:
            action = AgentAction(
                user_id=user_id,
                agent_id=agent_id,
                decision_id=decision_id,
                action_type=ad["action_type"],
                status="pending",
                summary=ad["summary"],
                action_payload=ad.get("action_payload"),
            )
            db.add(action)
            created.append(action)
        if created:
            await db.flush()
            for a in created:
                await db.refresh(a)
        return created

    @staticmethod
    async def list_pending(
        db: AsyncSession,
        user_id: int,
        agent_id: Optional[str] = None,
        status: Optional[str] = "pending",
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[AgentAction], int]:
        """获取待执行操作列表"""
        stmt = select(AgentAction).where(AgentAction.user_id == user_id)
        if agent_id:
            stmt = stmt.where(AgentAction.agent_id == agent_id)
        if status:
            stmt = stmt.where(AgentAction.status == status)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = (
            stmt.order_by(AgentAction.created_at.desc()).offset(offset).limit(page_size)
        )
        result = await db.execute(stmt)
        actions = list(result.scalars().all())
        return actions, total

    @staticmethod
    async def execute_action(
        db: AsyncSession,
        action_id: int,
        user_id: int,
    ) -> Optional[AgentAction]:
        """确认并执行操作"""
        action = await db.get(AgentAction, action_id)
        if not action or action.user_id != user_id:
            return None
        if action.status != "pending":
            return action

        action.status = "executed"
        action.execution_result = AgentActionService._mock_execute(action)
        await db.flush()
        await db.refresh(action)
        return action

    @staticmethod
    async def reject_action(
        db: AsyncSession,
        action_id: int,
        user_id: int,
    ) -> Optional[AgentAction]:
        """拒绝操作"""
        action = await db.get(AgentAction, action_id)
        if not action or action.user_id != user_id:
            return None
        action.status = "rejected"
        await db.flush()
        await db.refresh(action)
        return action

    @staticmethod
    def _mock_execute(action: AgentAction) -> dict:
        """模拟执行操作（无真实外部 API 时的 mock）"""
        now = datetime.now(timezone.utc).isoformat()
        base = {"executed_at": now, "mock": True}

        if action.action_type == "replenish":
            payload = action.action_payload or {}
            return {
                **base,
                "result": "补货任务已创建",
                "task_id": f"PO-{action.id}",
                "sku": payload.get("sku_code"),
                "quantity": payload.get("suggested_qty"),
            }
        elif action.action_type == "discount_review":
            return {
                **base,
                "result": "折扣审核工单已提交",
                "ticket_id": f"DR-{action.id}",
            }
        elif action.action_type == "price_review":
            return {
                **base,
                "result": "价格调整建议已提交",
                "ticket_id": f"PR-{action.id}",
            }
        elif action.action_type == "ad_action":
            return {
                **base,
                "result": "广告操作建议已提交",
                "ticket_id": f"AD-{action.id}",
            }
        return {**base, "result": "操作已记录"}
