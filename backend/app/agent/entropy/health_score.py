"""Layer 1 — 规则健康评分

评估维度:
  1. 采纳率 (weight: 0.30): 1 - (overrides / max(applied, 1))
  2. 置信度 (weight: 0.25): confidence / 1.0
  3. 新鲜度 (weight: 0.20): recency_decay based on last_applied_at
  4. 应用频次 (weight: 0.15): sigmoid(times_applied)
  5. 规则类型 (weight: 0.10): veto > threshold > strategy > style
"""
import math
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import PersonalRule


class RuleHealthScorer:

    WEIGHTS = {
        "acceptance": Decimal("0.30"),
        "confidence": Decimal("0.25"),
        "freshness": Decimal("0.20"),
        "frequency": Decimal("0.15"),
        "type_weight": Decimal("0.10"),
    }

    TYPE_SCORES = {
        "veto": 1.0,
        "threshold": 0.8,
        "strategy": 0.6,
        "style": 0.4,
    }

    UNHEALTHY_THRESHOLD = Decimal("0.40")
    WARNING_THRESHOLD = Decimal("0.60")

    async def score_rule(self, rule: PersonalRule) -> dict:
        now = datetime.now(timezone.utc)

        applied = rule.times_applied or 0
        overridden = rule.times_overridden or 0

        acceptance = Decimal("1.0") - (Decimal(str(overridden)) / max(Decimal(str(applied)), Decimal("1")))
        acceptance_score = max(Decimal("0"), acceptance)

        confidence_score = Decimal(str(rule.confidence or 0))

        if rule.last_applied_at:
            days_since = (now - rule.last_applied_at).days
            freshness_score = Decimal(str(max(0, 1.0 - (days_since / 180.0))))
        else:
            freshness_score = Decimal("0.2")

        frequency_score = Decimal("1.0") / (Decimal("1.0") + Decimal(str(math.exp(-0.1 * applied))))
        frequency_score = Decimal(str(round(float(frequency_score), 4)))

        type_score = Decimal(str(self.TYPE_SCORES.get(rule.rule_type, 0.5)))

        total = (
            acceptance_score * self.WEIGHTS["acceptance"]
            + confidence_score * self.WEIGHTS["confidence"]
            + freshness_score * self.WEIGHTS["freshness"]
            + frequency_score * self.WEIGHTS["frequency"]
            + type_score * self.WEIGHTS["type_weight"]
        )

        return {
            "rule_id": rule.id,
            "rule_name": rule.rule_name,
            "rule_type": rule.rule_type,
            "agent_id": rule.agent_id,
            "decision_point": rule.decision_point,
            "status": rule.status,
            "score": round(float(total), 4),
            "dimensions": {
                "acceptance": round(float(acceptance_score), 4),
                "confidence": round(float(confidence_score), 4),
                "freshness": round(float(freshness_score), 4),
                "frequency": round(float(frequency_score), 4),
                "type_weight": round(float(type_score), 4),
            },
            "times_applied": applied,
            "times_overridden": overridden,
            "override_rate": round(float(overridden / max(applied, 1)), 4),
            "days_since_last_applied": (now - rule.last_applied_at).days if rule.last_applied_at else None,
            "confidence": float(rule.confidence or 0),
            "risk_level": "unhealthy" if total < self.UNHEALTHY_THRESHOLD
                         else "warning" if total < self.WARNING_THRESHOLD
                         else "healthy",
        }

    async def score_all_rules(
        self, db: AsyncSession, user_id: int,
    ) -> list[dict]:
        stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.status.in_(["active", "shadow"]),
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())

        scores = []
        for rule in rules:
            scores.append(await self.score_rule(rule))

        scores.sort(key=lambda s: s["score"])
        return scores

    async def get_summary(self, db: AsyncSession, user_id: int) -> dict:
        scores = await self.score_all_rules(db, user_id)
        total = len(scores)
        if total == 0:
            return {
                "total_rules": 0, "active_rules": 0, "shadow_rules": 0,
                "avg_health_score": 0, "unhealthy_count": 0,
                "healthy_count": 0, "warning_count": 0,
            }

        avg_score = sum(s["score"] for s in scores) / total
        unhealthy = [s for s in scores if s["risk_level"] == "unhealthy"]
        warning = [s for s in scores if s["risk_level"] == "warning"]
        healthy = [s for s in scores if s["risk_level"] == "healthy"]

        stmt_active = select(func.count()).select_from(
            select(PersonalRule).where(
                PersonalRule.user_id == user_id,
                PersonalRule.status == "active",
            ).subquery()
        )
        stmt_shadow = select(func.count()).select_from(
            select(PersonalRule).where(
                PersonalRule.user_id == user_id,
                PersonalRule.status == "shadow",
            ).subquery()
        )

        active_count = await db.scalar(stmt_active) or 0
        shadow_count = await db.scalar(stmt_shadow) or 0

        return {
            "total_rules": total,
            "active_rules": active_count,
            "shadow_rules": shadow_count,
            "avg_health_score": round(avg_score, 4),
            "unhealthy_count": len(unhealthy),
            "warning_count": len(warning),
            "healthy_count": len(healthy),
        }
