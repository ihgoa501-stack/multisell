"""熵管理服务层"""
import logging
from datetime import datetime, timezone, timedelta
from decimal import Decimal
from typing import Optional
from sqlalchemy import select, func, cast, String
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

        # Count existing conflict-shadowed rules from RuleMarkChange
        conflict_count_stmt = select(func.count()).select_from(
            select(RuleMarkChange).where(
                RuleMarkChange.source_type == "gds",
                RuleMarkChange.source_id == "conflict_detector",
                RuleMarkChange.field_path == "$.status",
                cast(RuleMarkChange.new_value, String) == "shadow",
            ).subquery()
        )
        existing_conflicts = await db.scalar(conflict_count_stmt) or 0

        entropy_index = await self._calc_entropy_index(
            db, user_id, health_summary, conflict_count=existing_conflicts,
        )

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
            "conflict_count": existing_conflicts,
        }

    async def _calc_entropy_index(
        self, db: AsyncSession, user_id: int, health_summary: dict,
        conflict_count: int = 0,
    ) -> float:
        total = health_summary["total_rules"]
        if total == 0:
            return 0.0

        unhealthy_ratio = health_summary["unhealthy_count"] / max(total, 1)
        shadow_ratio = health_summary["shadow_rules"] / max(total, 1)
        avg_score = health_summary["avg_health_score"]
        conflict_ratio = conflict_count / max(total, 1)

        index = (
            unhealthy_ratio * 0.4
            + shadow_ratio * 0.3
            + (1.0 - avg_score) * 0.3
            + conflict_ratio * 0.3  # §4.1 冲突率计入熵指数
        )
        return min(1.0, index)

    async def run_defenses(self, db: AsyncSession, user_id: int) -> dict:
        expired = await self.ttl.expire_stale_rules(db, user_id)
        budget_exceeded = await self.budget.enforce_budgets(db, user_id)
        decayed = await self.decay.apply_decay(db, user_id)
        duplicates = await self.merge.find_duplicates(db, user_id)

        # 规则合并只生成候选，不自动合并（Phase 5 原则）
        shadowed = await self._shadow_overridden_rules(db, user_id)

        # §4.1 冲突检测
        conflicts = await self.detect_conflicts(db, user_id)

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
                "conflicts_detected": len(conflicts),  # §4.1 新增
            },
            "total_affected": (
                len(expired) + len(budget_exceeded) + len(decayed)
                + len(shadowed) + len(conflicts)
            ),
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
            "conflicts": [  # §4.1 新增
                {
                    "kept_rule_id": c[0],
                    "shadowed_rule_id": c[1],
                    "conflicting_field": c[2],
                }
                for c in conflicts
            ],
        }

    async def _shadow_overridden_rules(
        self, db: AsyncSession, user_id: int,
        max_override_rate: float = 0.5, min_applied: int = 3,
    ) -> list[PersonalRule]:
        """被用户频繁覆盖的规则降级为 shadow

        条件：应用次数 >= min_applied 且 覆盖率 > max_override_rate
        跳过手动编辑冷却期内的规则 (§4.3)
        """
        stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.status == "active",
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())

        # Skip rules in manual-edit cooling period (§4.3)
        now = datetime.now(timezone.utc)
        cutoff_72h = now - timedelta(hours=72)
        rules = [r for r in rules if not self._is_in_cooling_period(r, now, cutoff_72h)]
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

    def _is_in_cooling_period(
        self,
        rule: PersonalRule,
        now: Optional[datetime] = None,
        cutoff: Optional[datetime] = None,
        min_applications: int = 3,
        cooling_hours: int = 72,
    ) -> bool:
        """Check if a rule is in the manual edit cooling period.

        Entropy defenses should skip rules that:
        1. Have been applied fewer than min_applications times
        2. Were manually edited within the last cooling_hours hours
        """
        if (rule.times_applied or 0) < min_applications:
            return True
        if rule.last_manual_edit_at:
            now = now or datetime.now(timezone.utc)
            cutoff = cutoff or (now - timedelta(hours=cooling_hours))
            if rule.last_manual_edit_at > cutoff:
                return True
        return False

    async def detect_conflicts(
        self, db: AsyncSession, user_id: int, store_id: Optional[int] = None,
    ) -> list[tuple[int, int, str]]:
        """Detect conflicting PersonalRules within same (agent_id, decision_point).

        Same (field + op) condition but conflicting action.override values
        on the same field → lower priority rule is shadowed.

        Rules that are already shadow/paused/retired are excluded.
        Rules in manual-edit cooling period (§4.3) are skipped.

        Returns: list of (kept_rule_id, shadowed_rule_id, conflicting_field) tuples
        """
        now = datetime.now(timezone.utc)
        cutoff_72h = now - timedelta(hours=72)

        conditions = [
            PersonalRule.user_id == user_id,
            PersonalRule.status == "active",
        ]
        if store_id is not None:
            conditions.append(PersonalRule.store_id == store_id)

        stmt = (
            select(PersonalRule)
            .where(*conditions)
            .order_by(
                PersonalRule.agent_id,
                PersonalRule.decision_point,
                PersonalRule.priority.desc(),
                PersonalRule.created_at,
            )
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())

        # Filter out cooling-period rules
        rules = [r for r in rules if not self._is_in_cooling_period(r, now, cutoff_72h)]

        # Group by (agent_id, decision_point)
        groups: dict[tuple[str, str], list[PersonalRule]] = {}
        for rule in rules:
            key = (rule.agent_id, rule.decision_point)
            groups.setdefault(key, []).append(rule)

        conflicts: list[tuple[int, int, str]] = []

        for (_agent_id, _dp), group_rules in groups.items():
            if len(group_rules) < 2:
                continue

            # Further subgroup by (field, op) in rule_condition
            condition_groups: dict[tuple[str, str], list[PersonalRule]] = {}
            for rule in group_rules:
                cond = rule.rule_condition or {}
                field = cond.get("field")
                op = cond.get("op")
                if field and op:
                    ckey = (field, op)
                    condition_groups.setdefault(ckey, []).append(rule)

            for (_field, _op), cond_rules in condition_groups.items():
                if len(cond_rules) < 2:
                    continue

                # Compare each pair for conflicting action.override values
                for i in range(len(cond_rules)):
                    for j in range(i + 1, len(cond_rules)):
                        rule_a = cond_rules[i]
                        rule_b = cond_rules[j]

                        action_a = rule_a.rule_action or {}
                        action_b = rule_b.rule_action or {}
                        override_a = action_a.get("override", {})
                        override_b = action_b.get("override", {})

                        # Find same override field with different values
                        for over_field, val_a in override_a.items():
                            if over_field in override_b:
                                val_b = override_b[over_field]
                                if val_a != val_b:
                                    # Lower priority rule is shadowed
                                    # Rules are sorted by priority desc already
                                    if rule_a.priority >= rule_b.priority:
                                        keep, shadow = rule_a, rule_b
                                    else:
                                        keep, shadow = rule_b, rule_a

                                    # Only shadow if still active
                                    if shadow.status != "active":
                                        continue

                                    old_status = shadow.status
                                    shadow.status = "shadow"
                                    await self._log_rule_change(db, shadow, {
                                        "target_type": "personal_rule",
                                        "target_id": shadow.id,
                                        "field_path": "$.status",
                                        "old_value": old_status,
                                        "new_value": "shadow",
                                        "source_type": "gds",
                                        "source_id": "conflict_detector",
                                        "change_summary": (
                                            f"冲突自动降级为shadow: 与规则R{keep.id}在 "
                                            f"action.override.{over_field} 冲突({val_a} vs {val_b})"
                                        ),
                                        "context_json": {
                                            "conflicting_rule_id": keep.id,
                                            "conflicting_field": over_field,
                                            "value_a": val_a,
                                            "value_b": val_b,
                                            "condition_field": _field,
                                            "condition_op": _op,
                                        },
                                    })
                                    conflicts.append((keep.id, shadow.id, over_field))

        await db.flush()
        return conflicts

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
