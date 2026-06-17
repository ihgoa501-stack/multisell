"""Agent 服务层"""
import time
import logging
from typing import Optional, Any
from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import AgentDecision, PersonalRule, AgentEpisode, HonchoProfile, RuleConflict
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import AgentRegistry

logger = logging.getLogger(__name__)


class AgentService:

    @staticmethod
    async def get_decision_logs(
        db: AsyncSession,
        user_id: int,
        agent_id: Optional[str] = None,
        decision_point: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[AgentDecision], int]:
        stmt = select(AgentDecision).where(AgentDecision.user_id == user_id)
        if agent_id:
            stmt = stmt.where(AgentDecision.agent_id == agent_id)
        if decision_point:
            stmt = stmt.where(AgentDecision.decision_point == decision_point)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(AgentDecision.created_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        logs = result.scalars().all()
        return list(logs), total

    @staticmethod
    async def create_decision(db: AsyncSession, data: dict) -> AgentDecision:
        decision = AgentDecision(**data)
        db.add(decision)
        await db.flush()
        await db.refresh(decision)
        return decision

    @staticmethod
    async def update_decision_feedback(
        db: AsyncSession,
        decision_id: int,
        user_action: str,
        user_overrides: Optional[dict] = None,
        user_feedback: Optional[str] = None,
    ) -> Optional[AgentDecision]:
        decision = await db.get(AgentDecision, decision_id)
        if not decision:
            return None
        decision.user_action = user_action
        if user_overrides:
            decision.user_overrides = user_overrides
        if user_feedback:
            decision.user_feedback = user_feedback
        await db.flush()
        await db.refresh(decision)
        return decision

    @staticmethod
    async def list_rules(
        db: AsyncSession,
        user_id: int,
        agent_id: Optional[str] = None,
        decision_point: Optional[str] = None,
        status: Optional[str] = "active",
    ) -> list[PersonalRule]:
        stmt = select(PersonalRule).where(PersonalRule.user_id == user_id)
        if agent_id:
            stmt = stmt.where(PersonalRule.agent_id == agent_id)
        if decision_point:
            stmt = stmt.where(PersonalRule.decision_point == decision_point)
        if status:
            stmt = stmt.where(PersonalRule.status == status)
        stmt = stmt.order_by(PersonalRule.priority.desc(), PersonalRule.created_at.desc())
        result = await db.execute(stmt)
        return list(result.scalars().all())

    @staticmethod
    async def create_rule(db: AsyncSession, user_id: int, data: dict) -> PersonalRule:
        rule = PersonalRule(user_id=user_id, **data)
        db.add(rule)
        await db.flush()
        await db.refresh(rule)
        return rule

    @staticmethod
    async def update_rule(db: AsyncSession, rule_id: int, user_id: int, data: dict) -> Optional[PersonalRule]:
        rule = await db.get(PersonalRule, rule_id)
        if not rule or rule.user_id != user_id:
            return None
        old_status = rule.status
        for k, v in data.items():
            if v is not None:
                setattr(rule, k, v)
        rule.updated_at = func.now()

        # 记录状态变更到 rule_mark_change
        if "status" in data and data["status"] != old_status:
            from app.agent.models import RuleMarkChange
            change = RuleMarkChange(
                target_type="personal_rule",
                target_id=rule.id,
                field_path="$.status",
                old_value=old_status,
                new_value=data["status"],
                source_type="manual",
                source_id=f"user_{user_id}",
                change_summary=f"用户手动修改状态: {old_status} → {data['status']}",
                context_json={"rule_name": rule.rule_name},
            )
            db.add(change)

        await db.flush()
        await db.refresh(rule)
        return rule

    @staticmethod
    async def delete_rule(db: AsyncSession, rule_id: int, user_id: int) -> bool:
        rule = await db.get(PersonalRule, rule_id)
        if not rule or rule.user_id != user_id:
            return False
        await db.delete(rule)
        await db.flush()
        return True

    @staticmethod
    async def apply_rules(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
        decision: dict,
    ) -> tuple[dict, list[int]]:
        rules = await AgentService.list_rules(db, user_id, agent_id, decision_point)
        applied_rule_ids = []

        type_order = {"veto": 0, "threshold": 1, "strategy": 2, "style": 3}
        rules.sort(key=lambda r: (type_order.get(r.rule_type, 99), -r.priority))

        for rule in rules:
            if AgentService._matches_condition(rule.rule_condition, decision):
                decision = AgentService._apply_action(rule.rule_action, decision)
                applied_rule_ids.append(rule.id)
                rule.times_applied = (rule.times_applied or 0) + 1
                rule.last_applied_at = func.now()
                if rule.rule_type == "veto":
                    break

        return decision, applied_rule_ids

    @staticmethod
    def _matches_condition(condition: dict, decision: dict) -> bool:
        field = condition.get("field")
        op = condition.get("op")
        value = condition.get("value")
        actual = decision.get(field)
        if actual is None:
            return False
        if op == "gt":
            return actual > value
        elif op == "gte":
            return actual >= value
        elif op == "lt":
            return actual < value
        elif op == "lte":
            return actual <= value
        elif op == "eq":
            return actual == value
        elif op == "neq":
            return actual != value
        elif op == "in":
            return actual in value
        elif op == "contains":
            return value in str(actual)
        return False

    @staticmethod
    def _apply_action(action: dict, decision: dict) -> dict:
        override = action.get("override", {})
        if override:
            decision.update(override)
        modifier = action.get("modifier")
        if modifier:
            for k, v in modifier.items():
                if k in decision and isinstance(decision[k], (int, float)):
                    if modifier.get("type") == "percentage":
                        decision[k] = decision[k] * (1 + v / 100)
                    elif modifier.get("type") == "absolute":
                        decision[k] = decision[k] + v
        return decision

    @staticmethod
    async def execute_decision(
        db: AsyncSession,
        agent: BaseAgent,
        decision_point: str,
        context: dict,
        dry_run: bool = False,
    ) -> dict:
        start_time = time.time()

        agent_output = await agent.decide(decision_point, context, db=db)
        confidence = agent_output.get("confidence", 0.0)
        stage = agent.get_stage(decision_point)

        decision, applied_rule_ids = await AgentService.apply_rules(
            db, agent.user_id, agent.agent_id, decision_point, agent_output
        )

        elapsed = int((time.time() - start_time) * 1000)

        record = agent.build_decision_record(
            decision_point=decision_point,
            context=context,
            agent_output=agent_output,
            final_decision=decision,
            confidence=confidence,
            response_time_ms=elapsed,
            rules_applied=applied_rule_ids,
        )

        if not dry_run:
            created = await AgentService.create_decision(db, record)
            decision_id = created.id
            # 自动提取待执行操作
            from app.agent.action_service import AgentActionService
            actions = await AgentActionService.create_actions(
                db, agent.user_id, agent.agent_id, decision_id, decision
            )
        else:
            decision_id = None

        return {
            "agent_id": agent.agent_id,
            "decision_point": decision_point,
            "decision": decision,
            "stage": stage.value,
            "confidence": confidence,
            "rules_applied": applied_rule_ids,
            "decision_id": decision_id,
        }

    @staticmethod
    async def get_or_create_honcho_profile(db: AsyncSession, user_id: int) -> HonchoProfile:
        stmt = select(HonchoProfile).where(HonchoProfile.user_id == user_id)
        result = await db.execute(stmt)
        profile = result.scalar_one_or_none()
        if not profile:
            profile = HonchoProfile(user_id=user_id)
            db.add(profile)
            await db.flush()
            await db.refresh(profile)
        return profile

    @staticmethod
    async def update_honcho_profile(db: AsyncSession, user_id: int, data: dict) -> Optional[HonchoProfile]:
        profile = await AgentService.get_or_create_honcho_profile(db, user_id)
        for k, v in data.items():
            if v is not None:
                setattr(profile, k, v)
        await db.flush()
        await db.refresh(profile)
        return profile

    @staticmethod
    async def list_episodes(
        db: AsyncSession,
        user_id: int,
        agent_id: Optional[str] = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[AgentEpisode], int]:
        stmt = select(AgentEpisode).where(AgentEpisode.user_id == user_id)
        if agent_id:
            stmt = stmt.where(AgentEpisode.agent_id == agent_id)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(AgentEpisode.ended_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        episodes = result.scalars().all()
        return list(episodes), total

    @staticmethod
    async def get_dashboard(
        db: AsyncSession,
        user_id: int,
    ) -> dict:
        """G1 运营驾驶舱数据聚合"""
        from datetime import datetime, timezone, timedelta

        now = datetime.now(timezone.utc)
        seven_days_ago = now - timedelta(days=7)

        # ── 最近 7 天决策统计 ──
        recent_stmt = select(AgentDecision).where(
            AgentDecision.user_id == user_id,
            AgentDecision.created_at >= seven_days_ago,
        )
        recent_result = await db.execute(recent_stmt)
        recent_decisions = recent_result.scalars().all()

        total_recent = len(recent_decisions)
        accepted = sum(1 for d in recent_decisions if d.user_action == "accepted")
        acceptance_rate = round(accepted / total_recent, 3) if total_recent > 0 else 0.0

        by_agent: dict[str, int] = {}
        for d in recent_decisions:
            by_agent[d.agent_id] = by_agent.get(d.agent_id, 0) + 1

        # ── 待确认决策（ignored 且 recent） ──
        pending_stmt = select(AgentDecision).where(
            AgentDecision.user_id == user_id,
            AgentDecision.user_action == "ignored",
            AgentDecision.created_at >= seven_days_ago,
        ).order_by(AgentDecision.created_at.desc()).limit(20)
        pending_result = await db.execute(pending_stmt)
        pending_decisions = pending_result.scalars().all()

        # ── 风险汇总：从 agent_output 中提取风险信息 ──
        risk_items = []
        for d in recent_decisions:
            output = d.agent_output or {}
            decision = d.final_decision or {}

            if d.agent_id == "A5":
                status = output.get("stock_status", decision.get("stock_status", ""))
                if status == "red":
                    risk_items.append({
                        "agent_id": "A5",
                        "decision_id": d.id,
                        "risk_type": "即将断货",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("risk_reason", decision.get("risk_reason", "")),
                        "severity": "high",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })
                elif status == "yellow":
                    risk_items.append({
                        "agent_id": "A5",
                        "decision_id": d.id,
                        "risk_type": "库存预警",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("risk_reason", decision.get("risk_reason", "")),
                        "severity": "medium",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })

            elif d.agent_id == "G3":
                action = output.get("action", decision.get("action", ""))
                if action == "block":
                    risk_items.append({
                        "agent_id": "G3",
                        "decision_id": d.id,
                        "risk_type": "折扣风险（已阻断）",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("reason", decision.get("reason", "")),
                        "severity": "high",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })
                elif action == "warn":
                    risk_items.append({
                        "agent_id": "G3",
                        "decision_id": d.id,
                        "risk_type": "折扣风险（预警）",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("reason", decision.get("reason", "")),
                        "severity": "medium",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })

            elif d.agent_id == "A6":
                is_loss = output.get("is_loss", decision.get("is_loss", False))
                below = output.get("below_threshold", decision.get("below_threshold", False))
                if is_loss:
                    risk_items.append({
                        "agent_id": "A6",
                        "decision_id": d.id,
                        "risk_type": "亏损 SKU",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("anomaly_reason", decision.get("anomaly_reason", "")),
                        "severity": "high",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })
                elif below:
                    risk_items.append({
                        "agent_id": "A6",
                        "decision_id": d.id,
                        "risk_type": "低毛利 SKU",
                        "sku": output.get("sku_code", decision.get("sku_code", "")),
                        "detail": output.get("anomaly_reason", decision.get("anomaly_reason", "")),
                        "severity": "medium",
                        "created_at": d.created_at.isoformat() if d.created_at else None,
                    })

        # ── 规则健康概览（从 PersonalRule 统计） ──
        from app.agent.models import PersonalRule
        rules_stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
        )
        rules_result = await db.execute(rules_stmt)
        all_rules = rules_result.scalars().all()

        total_rules = len(all_rules)
        active_rules = sum(1 for r in all_rules if r.status == "active")
        shadow_rules = sum(1 for r in all_rules if r.status == "shadow")
        retired_rules = sum(1 for r in all_rules if r.status in ("retired", "paused"))

        return {
            "summary": {
                "total_decisions_7d": total_recent,
                "acceptance_rate_7d": acceptance_rate,
                "pending_confirmations": len(pending_decisions),
                "active_risks": len(risk_items),
            },
            "decisions_by_agent": by_agent,
            "recent_risks": risk_items[:20],
            "pending_decisions": [
                {
                    "id": d.id,
                    "agent_id": d.agent_id,
                    "decision_point": d.decision_point,
                    "created_at": d.created_at.isoformat() if d.created_at else None,
                }
                for d in pending_decisions[:10]
            ],
            "rule_health": {
                "total": total_rules,
                "active": active_rules,
                "shadow": shadow_rules,
                "retired_or_paused": retired_rules,
            },
        }
