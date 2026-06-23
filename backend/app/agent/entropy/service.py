"""熵管理服务层"""
import logging
from datetime import datetime, timezone, timedelta
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import RuleMarkChange, PersonalRule
from app.agent.entropy.defenses import TTLSweeper, BudgetEnforcer, DecayScheduler, MergeDetector, RegretAnalyzer
from app.agent.entropy.health_score import RuleHealthScorer
from app.agent.entropy.spc_control import SpcController

logger = logging.getLogger(__name__)


class EntropyService:

    def __init__(self):
        self.ttl = TTLSweeper()
        self.budget = BudgetEnforcer()
        self.decay = DecayScheduler()
        self.merge = MergeDetector()
        self.regret = RegretAnalyzer()
        self.scorer = RuleHealthScorer()
        self.spc = SpcController()

    async def get_dashboard(self, db: AsyncSession, user_id: int) -> dict:
        health_summary = await self.scorer.get_summary(db, user_id)

        now = datetime.now(timezone.utc)
        recent_24h = now - timedelta(hours=24)
        recent_stmt = select(func.count()).select_from(
            select(RuleMarkChange).where(
                RuleMarkChange.created_at >= recent_24h,
            ).subquery()
        )
        recent_changes = await db.scalar(recent_stmt) or 0

        total_stmt = select(func.count()).select_from(
            select(PersonalRule).where(
                PersonalRule.user_id == user_id,
                PersonalRule.status == "active",
            ).subquery()
        )
        total_rules = await db.scalar(total_stmt) or 0

        duplicates = await self.merge.find_duplicates(db, user_id)

        entropy_index = await self._calc_entropy_index(db, user_id, health_summary)

        return {
            "total_rules": total_rules,
            "active_rules": health_summary["active_rules"],
            "shadow_rules": health_summary["shadow_rules"],
            "avg_health_score": health_summary["avg_health_score"],
            "unhealthy_rule_count": health_summary["unhealthy_count"],
            "warning_rule_count": health_summary["warning_count"],
            "pending_merge_count": len(duplicates),
            "recent_changes_count": recent_changes,
            "system_entropy_index": round(entropy_index, 4),
        }

    async def _calc_entropy_index(self, db: AsyncSession, user_id: int, health_summary: dict) -> float:
        total = health_summary["total_rules"]
        if total == 0:
            return 0.0

        unhealthy_ratio = health_summary["unhealthy_count"] / max(total, 1)
        shadow_ratio = health_summary["shadow_rules"] / max(total, 1)
        avg_score = health_summary["avg_health_score"]

        index = (unhealthy_ratio * 0.4 + shadow_ratio * 0.3 + (1.0 - avg_score) * 0.3)
        return min(1.0, index)

    async def run_defenses(self, db: AsyncSession, user_id: int) -> dict:
        expired = await self.ttl.expire_stale_rules(db, user_id)
        budget_exceeded = await self.budget.enforce_budgets(db, user_id)
        decayed = await self.decay.apply_decay(db, user_id)
        duplicates = await self.merge.find_duplicates(db, user_id)

        # 规则合并只生成候选，不自动合并（Phase 5 原则）
        shadowed = await self._shadow_overridden_rules(db, user_id)

        mark_changes = []
        for rule in expired + budget_exceeded:
            mark_changes.append({
                "target_type": "personal_rule",
                "target_id": rule.id,
                "change_summary": f"防守动作: {rule.rule_name} → {rule.status}",
            })

        return {
            "actions": {
                "expired_rules": len(expired),
                "budget_exceeded": len(budget_exceeded),
                "decay_applied": len(decayed),
                "merged_candidates": len(duplicates),  # 只生成候选，不自动合并
                "shadowed_by_overrides": len(shadowed),
            },
            "total_affected": len(expired) + len(budget_exceeded) + len(decayed) + len(shadowed),
            "mark_changes": mark_changes,
            "duplicates_found": len(duplicates),
            "merge_candidates": [
                {
                    "keep_id": dup["keep"].id,
                    "keep_name": dup["keep"].rule_name,
                    "remove_id": dup["remove"].id,
                    "remove_name": dup["remove"].rule_name,
                    "similarity": dup["similarity"],
                }
                for dup in duplicates[:10]
            ],
        }

    async def _shadow_overridden_rules(
        self, db: AsyncSession, user_id: int,
        max_override_rate: float = 0.5, min_applied: int = 5,
    ) -> list[PersonalRule]:
        """被用户频繁覆盖的规则降级为 shadow

        条件：应用次数 >= min_applied 且 覆盖率 > max_override_rate
        """
        stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.status == "active",
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())
        shadowed = []

        for rule in rules:
            applied = rule.times_applied or 0
            overridden = rule.times_overridden or 0
            if applied >= min_applied and overridden > 0:
                rate = overridden / applied
                if rate > max_override_rate:
                    old_status = rule.status
                    rule.status = "shadow"
                    await self._log_rule_change(db, rule, {
                        "target_type": "personal_rule",
                        "target_id": rule.id,
                        "field_path": "$.status",
                        "old_value": old_status,
                        "new_value": "shadow",
                        "source_type": "gds",
                        "source_id": "override_shadow",
                        "change_summary": (
                            f"覆盖率过高({rate:.0%})自动降级为shadow: "
                            f"应用{applied}次, 覆盖{overridden}次"
                        ),
                        "context_json": {
                            "applied": applied,
                            "overridden": overridden,
                            "override_rate": rate,
                        },
                    })
                    shadowed.append(rule)

        await db.flush()
        return shadowed

    async def _log_rule_change(self, db: AsyncSession, rule: PersonalRule, data: dict) -> RuleMarkChange:
        change = RuleMarkChange(**data)
        db.add(change)
        return change

    async def get_health_scores(self, db: AsyncSession, user_id: int) -> list[dict]:
        return await self.scorer.score_all_rules(db, user_id)

    async def get_spc_status(self, db: AsyncSession, user_id: int) -> list[dict]:
        limits = await self.spc.get_all_limits(db, user_id)
        results = []
        for limit in limits:
            results.append({
                "agent_id": limit.agent_id,
                "decision_point": limit.decision_point,
                "metric_name": limit.metric_name,
                "baseline_mean": float(limit.baseline_mean),
                "ucl": float(limit.ucl),
                "lcl": float(limit.lcl),
                "uwl": float(limit.uwl),
                "lwl": float(limit.lwl),
                "consecutive_same_side": limit.consecutive_same_side,
                "is_out_of_control": False,
                "is_warning": False,
                "last_breach_at": limit.last_breach_at,
                "next_recalc_at": limit.next_recalc_at,
            })
        return results

    async def get_change_log(
        self, db: AsyncSession, user_id: int,
        source_type: Optional[str] = None,
        page: int = 1, page_size: int = 20,
    ) -> tuple[list[RuleMarkChange], int]:
        stmt = select(RuleMarkChange)
        if source_type:
            stmt = stmt.where(RuleMarkChange.source_type == source_type)

        count_stmt = select(func.count()).select_from(stmt.subquery())
        total = await db.scalar(count_stmt) or 0

        offset = (page - 1) * page_size
        stmt = stmt.order_by(RuleMarkChange.created_at.desc()).offset(offset).limit(page_size)
        result = await db.execute(stmt)
        changes = list(result.scalars().all())
        return changes, total
