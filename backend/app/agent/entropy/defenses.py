"""熵防守五道防线 — TTL / Budget / Decay / Merge / Regret

设计依据: hermes-self-evolving-agent-design.md §17.3
Phase 1b (M2): TTL + Budget + Decay
Phase 2a (M3): Regret + Merge
"""

import logging
from datetime import datetime, timedelta, timezone
from decimal import Decimal
from typing import Optional
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import PersonalRule, AgentDecision, RuleMarkChange

logger = logging.getLogger(__name__)


class TTLSweeper:
    """防线1 — TTL: 过期规则自动退休"""

    DEFAULT_TTL_DAYS = 90

    async def expire_stale_rules(
        self,
        db: AsyncSession,
        user_id: int,
        ttl_days: int = DEFAULT_TTL_DAYS,
    ) -> list[PersonalRule]:
        cutoff = datetime.now(timezone.utc) - timedelta(days=ttl_days)
        stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.status == "active",
            PersonalRule.last_applied_at.isnot(None),
            PersonalRule.last_applied_at < cutoff,
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())

        for rule in rules:
            rule.status = "retired"
            await self._log_change(
                db,
                rule,
                {
                    "target_type": "personal_rule",
                    "target_id": rule.id,
                    "field_path": "$.status",
                    "old_value": "active",
                    "new_value": "retired",
                    "source_type": "gds",
                    "source_id": "ttl_sweeper",
                    "change_summary": f"TTL过期自动退休: 超过{ttl_days}天未使用",
                    "context_json": {
                        "ttl_days": ttl_days,
                        "cutoff": cutoff.isoformat(),
                    },
                },
            )

        await db.flush()
        return rules

    async def _log_change(
        self, db: AsyncSession, rule: PersonalRule, data: dict
    ) -> RuleMarkChange:
        change = RuleMarkChange(**data)
        db.add(change)
        return change


class BudgetEnforcer:
    """防线2 — Budget: 按类别限制规则数量"""

    DEFAULT_BUDGETS = {
        "veto": 10,
        "threshold": 20,
        "strategy": 15,
        "style": 10,
    }

    async def enforce_budgets(
        self,
        db: AsyncSession,
        user_id: int,
        budgets: Optional[dict[str, int]] = None,
    ) -> list[PersonalRule]:
        budgets = budgets or self.DEFAULT_BUDGETS
        all_exceeded = []

        for rule_type, limit in budgets.items():
            stmt = (
                select(PersonalRule)
                .where(
                    PersonalRule.user_id == user_id,
                    PersonalRule.rule_type == rule_type,
                    PersonalRule.status == "active",
                )
                .order_by(
                    PersonalRule.priority.desc(),
                    PersonalRule.last_applied_at.desc().nullslast(),
                )
            )
            result = await db.execute(stmt)
            rules = list(result.scalars().all())

            if len(rules) > limit:
                excess = rules[limit:]
                for rule in excess:
                    old_status = rule.status
                    rule.status = "shadow"
                    await self._log_change(
                        db,
                        rule,
                        {
                            "target_type": "personal_rule",
                            "target_id": rule.id,
                            "field_path": "$.status",
                            "old_value": old_status,
                            "new_value": "shadow",
                            "source_type": "gds",
                            "source_id": "budget_enforcer",
                            "change_summary": f"Budget超限: {rule_type}最多{limit}条, 降级为shadow",
                            "context_json": {
                                "rule_type": rule_type,
                                "limit": limit,
                                "total": len(rules),
                            },
                        },
                    )
                all_exceeded.extend(excess)

        await db.flush()
        return all_exceeded

    async def _log_change(
        self, db: AsyncSession, rule: PersonalRule, data: dict
    ) -> RuleMarkChange:
        change = RuleMarkChange(**data)
        db.add(change)
        return change


class DecayScheduler:
    """防线3 — Decay: 未使用规则置信度衰减"""

    DECAY_RATE = Decimal("0.05")
    MIN_CONFIDENCE = Decimal("0.1")

    async def apply_decay(
        self,
        db: AsyncSession,
        user_id: int,
        decay_rate: Decimal = DECAY_RATE,
    ) -> list[PersonalRule]:
        stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.status.in_(["active", "shadow"]),
            PersonalRule.confidence > self.MIN_CONFIDENCE,
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())
        affected = []

        for rule in rules:
            old_conf = rule.confidence
            new_conf = max(rule.confidence - decay_rate, self.MIN_CONFIDENCE)
            rule.confidence = new_conf
            await self._log_change(
                db,
                rule,
                {
                    "target_type": "personal_rule",
                    "target_id": rule.id,
                    "field_path": "$.confidence",
                    "old_value": float(old_conf),
                    "new_value": float(new_conf),
                    "source_type": "gds",
                    "source_id": "decay_scheduler",
                    "change_summary": f"置信度衰减: {float(old_conf):.3f} → {float(new_conf):.3f}",
                    "context_json": {"decay_rate": float(decay_rate)},
                },
            )
            affected.append(rule)

        await db.flush()
        return affected

    async def _log_change(
        self, db: AsyncSession, rule: PersonalRule, data: dict
    ) -> RuleMarkChange:
        change = RuleMarkChange(**data)
        db.add(change)
        return change


class MergeDetector:
    """防线4 — Merge: 检测重复/重叠规则"""

    SIMILARITY_THRESHOLD = 0.85

    async def find_duplicates(
        self,
        db: AsyncSession,
        user_id: int,
    ) -> list[dict]:
        stmt = (
            select(PersonalRule)
            .where(
                PersonalRule.user_id == user_id,
                PersonalRule.status.in_(["active", "shadow"]),
            )
            .order_by(
                PersonalRule.agent_id,
                PersonalRule.decision_point,
                PersonalRule.created_at,
            )
        )
        result = await db.execute(stmt)
        rules = list(result.scalars().all())

        groups: dict[str, list[PersonalRule]] = {}
        for rule in rules:
            key = f"{rule.agent_id}:{rule.decision_point}:{rule.rule_type}"
            groups.setdefault(key, []).append(rule)

        duplicates = []
        for key, group in groups.items():
            if len(group) < 2:
                continue
            for i in range(len(group)):
                for j in range(i + 1, len(group)):
                    if self._are_similar(group[i], group[j]):
                        duplicates.append(
                            {
                                "keep": group[i],
                                "remove": group[j],
                                "similarity": self.SIMILARITY_THRESHOLD,
                            }
                        )

        return duplicates

    def _are_similar(self, a: PersonalRule, b: PersonalRule) -> bool:
        if a.rule_condition == b.rule_condition and a.rule_action == b.rule_action:
            return True
        if a.rule_condition and b.rule_condition:
            a_field = a.rule_condition.get("field")
            b_field = b.rule_condition.get("field")
            if a_field and a_field == b_field:
                return True
        return False

    async def merge_rules(
        self,
        db: AsyncSession,
        keep_id: int,
        remove_id: int,
    ) -> Optional[PersonalRule]:
        keep = await db.get(PersonalRule, keep_id)
        remove = await db.get(PersonalRule, remove_id)
        if not keep or not remove:
            return None

        keep.times_applied = (keep.times_applied or 0) + (remove.times_applied or 0)
        keep.times_overridden = (keep.times_overridden or 0) + (
            remove.times_overridden or 0
        )
        old_confidence = keep.confidence
        keep.confidence = max(keep.confidence, remove.confidence)

        old_status = remove.status
        remove.status = "retired"

        change = RuleMarkChange(
            target_type="personal_rule",
            target_id=keep.id,
            field_path="$.merged_from",
            old_value=None,
            new_value=remove_id,
            source_type="gds",
            source_id="merge_detector",
            change_summary=f"合并规则: {remove.rule_name}(#{remove_id}) → {keep.rule_name}(#{keep_id})",
            context_json={
                "keep_id": keep_id,
                "remove_id": remove_id,
                "old_confidence": float(old_confidence) if old_confidence else None,
                "new_confidence": float(keep.confidence),
            },
        )
        db.add(change)

        change2 = RuleMarkChange(
            target_type="personal_rule",
            target_id=remove.id,
            field_path="$.status",
            old_value=old_status,
            new_value="retired",
            source_type="gds",
            source_id="merge_detector",
            change_summary=f"因合并到#{keep_id}而退休",
        )
        db.add(change2)

        await db.flush()
        return keep


class RegretAnalyzer:
    """防线5 — Regret: 检测变更后指标恶化的回滚机制"""

    REGRET_WINDOW_HOURS = 48
    REGRET_THRESHOLD = Decimal("0.15")

    async def find_regrettable_changes(
        self,
        db: AsyncSession,
        user_id: int,
    ) -> list[dict]:
        window_start = datetime.now(timezone.utc) - timedelta(
            hours=self.REGRET_WINDOW_HOURS
        )

        recent_changes_stmt = (
            select(RuleMarkChange)
            .where(
                RuleMarkChange.source_type.in_(["gds", "gds_proxy", "nudge"]),
                RuleMarkChange.created_at >= window_start,
            )
            .order_by(RuleMarkChange.created_at.desc())
        )
        result = await db.execute(recent_changes_stmt)
        changes = list(result.scalars().all())

        regrettable = []
        for change in changes:
            if change.target_type != "personal_rule":
                continue
            rule = await db.get(PersonalRule, change.target_id)
            if not rule:
                continue

            before_stmt = select(func.avg(AgentDecision.confidence)).where(
                AgentDecision.user_id == user_id,
                AgentDecision.agent_id == rule.agent_id,
                AgentDecision.decision_point == rule.decision_point,
                AgentDecision.created_at < window_start,
                AgentDecision.created_at >= window_start - timedelta(days=30),
            )
            after_stmt = select(func.avg(AgentDecision.confidence)).where(
                AgentDecision.user_id == user_id,
                AgentDecision.agent_id == rule.agent_id,
                AgentDecision.decision_point == rule.decision_point,
                AgentDecision.created_at >= window_start,
            )

            before_avg = await db.scalar(before_stmt) or Decimal(0)
            after_avg = await db.scalar(after_stmt) or Decimal(0)

            if before_avg > 0 and (before_avg - after_avg) >= self.REGRET_THRESHOLD:
                regrettable.append(
                    {
                        "change": change,
                        "rule": rule,
                        "before_avg": float(before_avg),
                        "after_avg": float(after_avg),
                        "drop": float(before_avg - after_avg),
                    }
                )

        return regrettable

    async def rollback_change(
        self,
        db: AsyncSession,
        change_id: int,
    ) -> Optional[PersonalRule]:
        change = await db.get(RuleMarkChange, change_id)
        if not change or change.target_type != "personal_rule":
            return None

        rule = await db.get(PersonalRule, change.target_id)
        if not rule:
            return None

        field = change.field_path.replace("$.", "")
        if hasattr(rule, field):
            setattr(rule, field, change.old_value)

        rollback_log = RuleMarkChange(
            target_type="personal_rule",
            target_id=rule.id,
            field_path=change.field_path,
            old_value=change.new_value,
            new_value=change.old_value,
            source_type="gds",
            source_id="regret_analyzer",
            change_summary=f"遗憾回滚: 还原变更 #{change.id} ({change.change_summary})",
            parent_change_id=change.id,
            context_json={"parent_change_summary": change.change_summary},
        )
        db.add(rollback_log)
        await db.flush()
        return rule
