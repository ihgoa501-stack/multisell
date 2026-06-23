"""Agent 进化服务 — 信任评分、等级控制、Nudge 晋升提示

与 PRD P0-2（信任评分计算引擎）和 P0-3（等级控制 + Nudge 机制）对应。
"""

import logging
from datetime import datetime, timezone, timedelta
from typing import Optional

from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from app.agent.models import (
    AgentDecision,
    AgentEvolutionConfig,
    AgentNudge,
    PersonalRule,
)
from app.agent.base import EvolutionStage
from app.agent.registry import AgentRegistry
from app.agent.entropy.spc_control import SpcController

logger = logging.getLogger(__name__)

# ── 晋升阈值 ──────────────────────────────────────────────

PROMOTION_THRESHOLDS = {
    (EvolutionStage.OBSERVATION, EvolutionStage.SUGGESTION): {
        "min_score": 40,
        "min_decisions": 20,
    },
    (EvolutionStage.SUGGESTION, EvolutionStage.SEMI_AUTONOMOUS): {
        "min_score": 65,
        "min_decisions": 50,
    },
    (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS): {
        "min_score": 85,
        "min_decisions": 100,
        "no_regret": True,
    },
}

# ── 评分权重 ──────────────────────────────────────────────

SCORE_WEIGHTS = {
    "adoption_rate": 0.40,
    "avg_confidence": 0.25,
    "consistency": 0.20,
    "stability": 0.15,
}

NUDGE_COOLDOWN_DAYS = 7  # Nudge 冷却期
MAX_NUDGE_DISMISS = 3  # 连续忽略 N 次后延长冷却


class TrustScoreCalculator:
    """信任评分计算引擎（4 维度加权）"""

    @staticmethod
    async def calculate(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
    ) -> dict:
        """计算某个 user × agent × decision_point 的信任评分"""
        now = datetime.now(timezone.utc)
        window_start = now - timedelta(days=30)

        # ── 获取近期决策 ──
        stmt = (
            select(AgentDecision)
            .where(
                AgentDecision.user_id == user_id,
                AgentDecision.agent_id == agent_id,
                AgentDecision.decision_point == decision_point,
                AgentDecision.created_at >= window_start,
            )
            .order_by(AgentDecision.created_at.desc())
        )
        result = await db.execute(stmt)
        decisions = list(result.scalars().all())

        total = len(decisions)
        if total == 0:
            return {
                "trust_score": 0.0,
                "decision_count": 0,
                "components": {
                    "adoption_rate": 0.0,
                    "avg_confidence": 0.0,
                    "consistency": 0.0,
                    "stability": 0.0,
                },
                "is_calculable": False,
            }

        # 1. 采纳率（40%）: accepted / (accepted + modified + rejected)
        accepted = sum(1 for d in decisions if d.user_action == "accepted")
        modified = sum(1 for d in decisions if d.user_action == "modified")
        rejected = sum(1 for d in decisions if d.user_action == "rejected")
        denominator = accepted + modified + rejected
        adoption_rate = accepted / denominator if denominator > 0 else 0.0

        # 2. 平均置信度（25%）
        confidences = [
            float(d.confidence) for d in decisions if d.confidence is not None
        ]
        avg_confidence = sum(confidences) / len(confidences) if confidences else 0.0

        # 3. 规则覆盖一致性（20%）
        #    查询该决策点的活跃规则
        rules_stmt = select(PersonalRule).where(
            PersonalRule.user_id == user_id,
            PersonalRule.agent_id == agent_id,
            PersonalRule.decision_point == decision_point,
            PersonalRule.status == "active",
        )
        rules_result = await db.execute(rules_stmt)
        rules = list(rules_result.scalars().all())

        if rules:
            total_applied = sum(r.times_applied or 0 for r in rules)
            total_overridden = sum(r.times_overridden or 0 for r in rules)
            # consistency = 1 - (overridden / max(applied, 1))
            consistency = 1.0 - (total_overridden / max(total_applied, 1))
            consistency = max(0.0, min(1.0, consistency))
        else:
            consistency = 1.0  # 无规则 = 完全一致

        # 4. 稳定性（15%）: SPC 越线检查
        spc = SpcController()
        # 检查 acceptance_rate 维度的 SPC
        spc_check = await spc.check_point(
            db,
            user_id,
            agent_id,
            decision_point,
            "acceptance_rate",
            adoption_rate,
        )
        stability = 1.0
        if spc_check.get("is_out_of_control"):
            stability = 0.0
        elif spc_check.get("is_warning"):
            stability = 0.5
        # 连续同侧超过 3 点扣分
        consecutive = abs(spc_check.get("consecutive_same_side", 0))
        if consecutive > 3:
            stability -= min(0.5, (consecutive - 3) * 0.1)

        stability = max(0.0, min(1.0, stability))

        # ── 加权计算 ──
        score = (
            adoption_rate * 100 * SCORE_WEIGHTS["adoption_rate"]
            + avg_confidence * 100 * SCORE_WEIGHTS["avg_confidence"]
            + consistency * 100 * SCORE_WEIGHTS["consistency"]
            + stability * 100 * SCORE_WEIGHTS["stability"]
        )
        score = round(min(100.0, max(0.0, score)), 2)

        return {
            "trust_score": score,
            "decision_count": total,
            "components": {
                "adoption_rate": round(adoption_rate, 4),
                "avg_confidence": round(avg_confidence, 4),
                "consistency": round(consistency, 4),
                "stability": round(stability, 4),
            },
            "is_calculable": True,
        }

    @staticmethod
    def check_promotion_eligibility(
        current_stage: EvolutionStage,
        target_stage: EvolutionStage,
        trust_score: float,
        decision_count: int,
        has_regret: bool = False,
    ) -> dict:
        """检查是否满足晋升条件"""
        key = (current_stage, target_stage)
        thresholds = PROMOTION_THRESHOLDS.get(key)
        if not thresholds:
            return {
                "eligible": False,
                "reason": f"不支持的晋升路径: {current_stage.value} → {target_stage.value}",
            }

        issues = []
        if trust_score < thresholds["min_score"]:
            issues.append(
                f"信任评分不足（{trust_score:.1f}/{thresholds['min_score']}）"
            )
        if decision_count < thresholds["min_decisions"]:
            issues.append(
                f"决策样本不足（{decision_count}/{thresholds['min_decisions']}）"
            )
        if thresholds.get("no_regret") and has_regret:
            issues.append("存在 Regret 回滚记录，不符合晋升条件")

        if issues:
            return {"eligible": False, "reason": "；".join(issues)}

        return {"eligible": True}

    @staticmethod
    def get_promotion_target(current_stage: EvolutionStage) -> Optional[EvolutionStage]:
        """获取当前阶段的下一晋升目标"""
        order = [
            EvolutionStage.OBSERVATION,
            EvolutionStage.SUGGESTION,
            EvolutionStage.SEMI_AUTONOMOUS,
            EvolutionStage.FULL_AUTONOMOUS,
        ]
        try:
            idx = order.index(current_stage)
            if idx < len(order) - 1:
                return order[idx + 1]
        except ValueError:
            pass
        return None


class EvolutionService:
    """Agent 进化管理服务"""

    @staticmethod
    async def get_or_create_config(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
    ) -> AgentEvolutionConfig:
        """获取或创建进化配置"""
        stmt = select(AgentEvolutionConfig).where(
            AgentEvolutionConfig.user_id == user_id,
            AgentEvolutionConfig.agent_id == agent_id,
            AgentEvolutionConfig.decision_point == decision_point,
        )
        result = await db.execute(stmt)
        config = result.scalar_one_or_none()
        if not config:
            config = AgentEvolutionConfig(
                user_id=user_id,
                agent_id=agent_id,
                decision_point=decision_point,
                current_stage="observation",
            )
            db.add(config)
            await db.flush()
            await db.refresh(config)
        return config

    @staticmethod
    async def get_overview(
        db: AsyncSession,
        user_id: int,
    ) -> dict:
        """获取 Agent 自治等级总览面板数据"""
        now = datetime.now(timezone.utc)
        seven_days_ago = now - timedelta(days=7)
        agents_meta = AgentRegistry.list_agents()

        # ── 7 天全局统计 ──
        recent_stmt = select(AgentDecision).where(
            AgentDecision.user_id == user_id,
            AgentDecision.created_at >= seven_days_ago,
        )
        recent_result = await db.execute(recent_stmt)
        recent_decisions = list(recent_result.scalars().all())

        total_decisions_7d = len(recent_decisions)
        accepted_7d = sum(1 for d in recent_decisions if d.user_action == "accepted")
        adoption_rate_7d = (
            round(accepted_7d / total_decisions_7d, 4)
            if total_decisions_7d > 0
            else 0.0
        )

        # ── 活跃规则数 ──
        rules_stmt = select(func.count()).select_from(
            select(PersonalRule)
            .where(
                PersonalRule.user_id == user_id,
                PersonalRule.status == "active",
            )
            .subquery()
        )
        active_rules = await db.scalar(rules_stmt) or 0

        # ── 熵指数（从 EntropyService） ──
        from app.agent.entropy.service import EntropyService

        entropy_svc = EntropyService()
        entropy_dashboard = await entropy_svc.get_dashboard(db, user_id)
        entropy_index = entropy_dashboard.get("system_entropy_index", 0.0)

        # ── 按角色分组 Agent 卡片数据 ──
        governance_agents = []
        specialist_agents = []

        for meta in agents_meta:
            aid = meta["agent_id"]
            configs_stmt = select(AgentEvolutionConfig).where(
                AgentEvolutionConfig.user_id == user_id,
                AgentEvolutionConfig.agent_id == aid,
            )
            configs_result = await db.execute(configs_stmt)
            configs = list(configs_result.scalars().all())

            # 取该 Agent 所有决策点中的最高阶段
            stages = [EvolutionStage(c.current_stage) for c in configs]
            highest_stage = (
                max(stages, key=lambda s: list(EvolutionStage).index(s)).value
                if stages
                else "observation"
            )

            # 7 天决策数 + 采纳率
            agent_recent = [d for d in recent_decisions if d.agent_id == aid]
            agent_total = len(agent_recent)
            agent_acc = sum(1 for d in agent_recent if d.user_action == "accepted")
            agent_acceptance = (
                round(agent_acc / agent_total, 4) if agent_total > 0 else 0.0
            )

            # 最近一次信任评分
            latest_score = 0.0
            if configs:
                latest_score = float(max(c.trust_score or 0 for c in configs))

            # Nudge 角标
            has_pending_nudge = False
            nudge_stmt = (
                select(AgentNudge)
                .where(
                    AgentNudge.user_id == user_id,
                    AgentNudge.agent_id == aid,
                    AgentNudge.status == "pending",
                )
                .limit(1)
            )
            nudge_result = await db.execute(nudge_stmt)
            has_pending_nudge = nudge_result.scalar_one_or_none() is not None

            card = {
                "agent_id": aid,
                "name": meta["name"],
                "description": meta["description"],
                "highest_stage": highest_stage,
                "decision_points": meta["decision_points"],
                "decisions_7d": agent_total,
                "acceptance_rate_7d": agent_acceptance,
                "trust_score": latest_score,
                "has_pending_nudge": has_pending_nudge,
                "stage_color": EvolutionService._stage_color(highest_stage),
            }

            # G1-G3 = 治理官, A1-A7 = 专家
            if aid.startswith("G"):
                governance_agents.append(card)
            else:
                specialist_agents.append(card)

        return {
            "summary": {
                "total_decisions_7d": total_decisions_7d,
                "overall_acceptance_rate_7d": adoption_rate_7d,
                "active_rules": active_rules,
                "system_entropy_index": entropy_index,
            },
            "governance_agents": governance_agents,
            "specialist_agents": specialist_agents,
        }

    @staticmethod
    def _stage_color(stage: str) -> str:
        colors = {
            "observation": "gray",
            "suggestion": "blue",
            "semi_autonomous": "green",
            "full_autonomous": "gold",
        }
        return colors.get(stage, "gray")

    @staticmethod
    async def get_agent_detail(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
    ) -> Optional[dict]:
        """获取单个 Agent 详情与等级控制数据"""
        meta = AgentRegistry.get_metadata(agent_id)
        if not meta:
            return None

        decision_points = meta["decision_points"]
        point_configs = []

        for dp in decision_points:
            config = await EvolutionService.get_or_create_config(
                db, user_id, agent_id, dp
            )

            # 为该决策点计算信任评分（如超过 1 小时未计算）
            score_data = None
            needs_recalc = (
                config.last_calculated_at is None
                or (
                    datetime.now(timezone.utc) - config.last_calculated_at
                ).total_seconds()
                > 3600
            )
            if needs_recalc:
                score_data = await TrustScoreCalculator.calculate(
                    db, user_id, agent_id, dp
                )
                config.trust_score = score_data["trust_score"]
                config.decision_count = score_data["decision_count"]
                if score_data["is_calculable"]:
                    config.adoption_rate = score_data["components"]["adoption_rate"]
                    config.avg_confidence = score_data["components"]["avg_confidence"]
                    config.consistency_score = score_data["components"]["consistency"]
                    config.stability_score = score_data["components"]["stability"]
                config.last_calculated_at = datetime.now(timezone.utc)
                await db.flush()
            else:
                score_data = {
                    "trust_score": float(config.trust_score or 0),
                    "decision_count": config.decision_count or 0,
                    "components": {
                        "adoption_rate": float(config.adoption_rate or 0),
                        "avg_confidence": float(config.avg_confidence or 0),
                        "consistency": float(config.consistency_score or 0),
                        "stability": float(config.stability_score or 0),
                    },
                    "is_calculable": config.decision_count is not None
                    and config.decision_count > 0,
                }

            # 晋升目标检查
            current_stage = EvolutionStage(config.current_stage)
            target = TrustScoreCalculator.get_promotion_target(current_stage)
            promotion = None
            if target:
                promotion_check = TrustScoreCalculator.check_promotion_eligibility(
                    current_stage,
                    target,
                    score_data["trust_score"],
                    score_data["decision_count"],
                )
                promotion = {
                    "target_stage": target.value,
                    "eligible": promotion_check["eligible"],
                    "reason": promotion_check.get("reason"),
                    "progress": f"{score_data['decision_count']}/{PROMOTION_THRESHOLDS.get((current_stage, target), {}).get('min_decisions', '?')}",
                }

            point_configs.append(
                {
                    "decision_point": dp,
                    "current_stage": config.current_stage,
                    "stage_color": EvolutionService._stage_color(config.current_stage),
                    "trust_score": score_data["trust_score"],
                    "decision_count": score_data["decision_count"],
                    "components": score_data["components"],
                    "is_calculable": score_data["is_calculable"],
                    "promotion": promotion,
                    "stage_updated_at": config.stage_updated_at.isoformat()
                    if config.stage_updated_at
                    else None,
                    "stage_updated_by": config.stage_updated_by,
                }
            )

        # ── 最近 20 条决策日志 ──
        decision_stmt = (
            select(AgentDecision)
            .where(
                AgentDecision.user_id == user_id,
                AgentDecision.agent_id == agent_id,
            )
            .order_by(AgentDecision.created_at.desc())
            .limit(20)
        )
        decision_result = await db.execute(decision_stmt)
        recent_decisions = [
            {
                "id": d.id,
                "decision_point": d.decision_point,
                "final_decision": d.final_decision,
                "user_action": d.user_action,
                "evolution_stage": d.evolution_stage,
                "confidence": float(d.confidence) if d.confidence else None,
                "created_at": d.created_at.isoformat() if d.created_at else None,
            }
            for d in decision_result.scalars().all()
        ]

        return {
            "agent_id": agent_id,
            "name": meta["name"],
            "description": meta["description"],
            "decision_points": point_configs,
            "recent_decisions": recent_decisions,
        }

    @staticmethod
    async def change_stage(
        db: AsyncSession,
        user_id: int,
        agent_id: str,
        decision_point: str,
        target_stage: str,
        manual: bool = True,
    ) -> dict:
        """变更某个 Agent 决策点的自治阶段"""
        # 验证 target_stage
        try:
            target = EvolutionStage(target_stage)
        except ValueError:
            return {"success": False, "message": f"无效的阶段: {target_stage}"}

        config = await EvolutionService.get_or_create_config(
            db, user_id, agent_id, decision_point
        )
        current = EvolutionStage(config.current_stage)

        # 如果是升级操作，检查信任评分门槛（手动覆盖时跳过，允许管理员强制变更）
        is_upgrade = list(EvolutionStage).index(target) > list(EvolutionStage).index(
            current
        )
        if is_upgrade and not manual:
            score_data = await TrustScoreCalculator.calculate(
                db, user_id, agent_id, decision_point
            )
            check = TrustScoreCalculator.check_promotion_eligibility(
                current,
                target,
                score_data["trust_score"],
                score_data["decision_count"],
            )
            if not check["eligible"]:
                return {
                    "success": False,
                    "message": check["reason"],
                }

        # 执行阶段变更
        old_stage = config.current_stage
        config.current_stage = target.value
        config.stage_updated_at = datetime.now(timezone.utc)
        config.stage_updated_by = "manual" if is_upgrade else "manual"
        await db.flush()

        # 记录操作日志
        from app.operation_log.service import OperationLogService

        await OperationLogService.log(
            db=db,
            module="agent_evolution",
            action="agent_stage_changed",
            resource_id=f"{agent_id}:{decision_point}",
            content=str(
                {
                    "agent_id": agent_id,
                    "decision_point": decision_point,
                    "old_stage": old_stage,
                    "new_stage": target.value,
                    "trigger_source": "manual" if is_upgrade else "manual_downgrade",
                }
            ),
        )

        return {
            "success": True,
            "agent_id": agent_id,
            "decision_point": decision_point,
            "old_stage": old_stage,
            "new_stage": target.value,
        }

    @staticmethod
    async def get_pending_nudges(
        db: AsyncSession,
        user_id: int,
    ) -> list[dict]:
        """获取用户待处理的 Nudge 晋升提示列表"""
        stmt = (
            select(AgentNudge)
            .where(
                AgentNudge.user_id == user_id,
                AgentNudge.status == "pending",
            )
            .order_by(AgentNudge.created_at.desc())
            .limit(20)
        )
        result = await db.execute(stmt)
        nudges = result.scalars().all()

        return [
            {
                "id": n.id,
                "agent_id": n.agent_id,
                "decision_point": n.decision_point,
                "target_stage": n.target_stage,
                "trust_score": float(n.trust_score_at_time),
                "score_components": n.score_components,
                "created_at": n.created_at.isoformat() if n.created_at else None,
            }
            for n in nudges
        ]

    @staticmethod
    async def respond_nudge(
        db: AsyncSession,
        user_id: int,
        nudge_id: int,
        response: str,
    ) -> dict:
        """响应用户对 Nudge 的操作（accept/dismiss）"""
        nudge = await db.get(AgentNudge, nudge_id)
        if not nudge or nudge.user_id != user_id:
            return {"success": False, "message": "Nudge 不存在或无权操作"}

        if nudge.status != "pending":
            return {"success": False, "message": "Nudge 已处理"}

        nudge.status = (
            response if response in ("accepted", "dismissed") else "dismissed"
        )
        nudge.responded_at = datetime.now(timezone.utc)

        if response == "accepted":
            # 执行晋升
            stage_result = await EvolutionService.change_stage(
                db,
                user_id,
                nudge.agent_id,
                nudge.decision_point,
                nudge.target_stage,
                manual=False,
            )
            await db.flush()
            return {
                "success": stage_result["success"],
                "action": "accepted",
                "stage_result": stage_result,
            }

        elif response == "dismissed":
            # 设置冷却期
            config = await EvolutionService.get_or_create_config(
                db,
                user_id,
                nudge.agent_id,
                nudge.decision_point,
            )
            config.nudge_dismissed_count = (config.nudge_dismissed_count or 0) + 1
            config.nudge_last_shown_at = datetime.now(timezone.utc)

            # 连续忽略多次后延长冷却
            cooldown = NUDGE_COOLDOWN_DAYS
            if (config.nudge_dismissed_count or 0) >= MAX_NUDGE_DISMISS:
                cooldown *= 2
            nudge.cooling_until = datetime.now(timezone.utc) + timedelta(days=cooldown)

            await db.flush()
            return {"success": True, "action": "dismissed", "cooling_days": cooldown}

        return {"success": False, "message": f"无效响应: {response}"}

    @staticmethod
    async def generate_nudges(
        db: AsyncSession,
        user_id: int,
    ) -> list[dict]:
        """系统检查所有 Agent 的晋升条件，生成 Nudge 提示

        应由 scheduler 每日凌晨自动调用。
        """
        agents_meta = AgentRegistry.list_agents()
        generated = []

        for meta in agents_meta:
            aid = meta["agent_id"]
            for dp in meta["decision_points"]:
                config = await EvolutionService.get_or_create_config(
                    db, user_id, aid, dp
                )

                # 检查冷却期
                now = datetime.now(timezone.utc)
                if config.nudge_last_shown_at:
                    cooldown = NUDGE_COOLDOWN_DAYS
                    if (config.nudge_dismissed_count or 0) >= MAX_NUDGE_DISMISS:
                        cooldown *= 2
                    if (now - config.nudge_last_shown_at).days < cooldown:
                        continue

                current_stage = EvolutionStage(config.current_stage)
                target = TrustScoreCalculator.get_promotion_target(current_stage)
                if not target:
                    continue

                # 计算评分
                score_data = await TrustScoreCalculator.calculate(db, user_id, aid, dp)
                check = TrustScoreCalculator.check_promotion_eligibility(
                    current_stage,
                    target,
                    score_data["trust_score"],
                    score_data["decision_count"],
                )
                if not check["eligible"]:
                    continue

                # 检查是否已有待处理的 Nudge
                existing_stmt = (
                    select(AgentNudge)
                    .where(
                        AgentNudge.user_id == user_id,
                        AgentNudge.agent_id == aid,
                        AgentNudge.decision_point == dp,
                        AgentNudge.status == "pending",
                    )
                    .limit(1)
                )
                existing = await db.execute(existing_stmt)
                if existing.scalar_one_or_none():
                    continue

                # 创建 Nudge
                nudge = AgentNudge(
                    user_id=user_id,
                    agent_id=aid,
                    decision_point=dp,
                    target_stage=target.value,
                    trust_score_at_time=score_data["trust_score"],
                    score_components=score_data["components"],
                    status="pending",
                )
                db.add(nudge)
                config.nudge_last_shown_at = now
                await db.flush()
                await db.refresh(nudge)

                generated.append(
                    {
                        "id": nudge.id,
                        "agent_id": aid,
                        "decision_point": dp,
                        "target_stage": target.value,
                        "trust_score": score_data["trust_score"],
                    }
                )

        if generated:
            await db.flush()

        return generated
