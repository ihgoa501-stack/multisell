"""Agent 服务层"""
import time
import logging
from typing import Optional, Any
from sqlalchemy import select, func, and_
from sqlalchemy.ext.asyncio import AsyncSession
from app.agent.models import AgentDecision, PersonalRule, AgentEpisode, HonchoProfile, RuleConflict
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import AgentRegistry
from app.agent.llm_resilience import (
    LLMCircuitBreakerManager,
    _get_cached_decision,
    _set_cached_decision,
)

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
        stage = agent.get_stage(decision_point)

<<<<<<< HEAD
        # ── OBSERVATION: 仅生成数据报告，不提出建议 ──
        if stage == EvolutionStage.OBSERVATION:
            agent_output = await AgentService._call_with_resilience(
                agent, decision_point, context, db=db
            )
            decision = agent_output  # 直接输出，不应用规则
            elapsed = int((time.time() - start_time) * 1000)
            record = agent.build_decision_record(
                decision_point=decision_point,
                context=context,
                agent_output=agent_output,
                final_decision=decision,
                confidence=0.0,
                user_action="ignored",
                response_time_ms=elapsed,
                rules_applied=[],
            )
            if not dry_run:
                created = await AgentService.create_decision(db, record)
                decision_id = created.id
            else:
                decision_id = None
            return {
                "agent_id": agent.agent_id,
                "decision_point": decision_point,
                "decision": decision,
                "stage": stage.value,
                "confidence": 0.0,
                "rules_applied": [],
                "decision_id": decision_id,
                "note": "OBSERVATION: 仅收集数据，未应用规则或创建操作",
            }

        # ── SUGGESTION / SEMI_AUTONOMOUS / FULL_AUTONOMOUS ──
        agent_output = await AgentService._call_with_resilience(
            agent, decision_point, context, db=db
        )
        confidence = agent_output.get("confidence", 0.0)

        decision, applied_rule_ids = await AgentService.apply_rules(
            db, agent.user_id, agent.agent_id, decision_point, agent_output
        )

        # §5.3 输出验证
        decision = AgentService._sanitize_output(agent.agent_id, decision)

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

        decision_id = None
        actions_created = 0

        if not dry_run:
            created = await AgentService.create_decision(db, record)
            decision_id = created.id

            # ── 根据阶段决定是否创建待执行操作 ──
            # SUGGESTION: 不创建操作（仅建议）
            # SEMI_AUTONOMOUS: 创建操作，高风险需审批
            # FULL_AUTONOMOUS: 自动执行全部操作
            if stage != EvolutionStage.SUGGESTION:
                from app.agent.action_service import AgentActionService, extract_actions
                from app.agent.models import AgentAction

                action_defs = extract_actions(agent.agent_id, decision_id, decision)
                if action_defs:
                    # 高风险判定（基于金额和批量大小）
                    is_high_risk = AgentService._is_high_risk_action(
                        agent.agent_id, decision, stage
                    )

                    for ad in action_defs:
                        action_status = "pending" if (
                            stage == EvolutionStage.SEMI_AUTONOMOUS and is_high_risk
                        ) else "executed"

                        action = AgentAction(
                            user_id=agent.user_id,
                            agent_id=agent.agent_id,
                            decision_id=decision_id,
                            action_type=ad["action_type"],
                            status=action_status,
                            summary=ad["summary"],
                            action_payload=ad.get("action_payload"),
                        )
                        db.add(action)
                        actions_created += 1

                    if actions_created:
                        await db.flush()

            # ── 桥接到动作中枢：将 Agent 操作转为 ActionProposal ──
            try:
                from app.agentos.action_center_service import ActionCenterService
                from app.agentos.schemas import ActionProposalCreate, RiskLevel
                from app.agentos.service import AGENT_TO_SQUAD

                squad_id = AGENT_TO_SQUAD.get(agent.agent_id, "governance")

                # SEMI_AUTONOMOUS / FULL_AUTONOMOUS: 从 AgentAction 桥接
                if stage != EvolutionStage.SUGGESTION and actions_created:
                    for aa in db.new if hasattr(db, 'new') else []:
                        if hasattr(aa, 'action_type') and hasattr(aa, 'action_payload'):
                            risk, needs_approval = AgentService._derive_action_risk(aa.action_type, aa.action_payload or {}, stage)
                            payload = ActionProposalCreate(
                                source_type="agent_action",
                                source_id=str(getattr(aa, 'id', 0)),
                                agent_id=agent.agent_id,
                                squad_id=squad_id,
                                action_type=AgentService._map_action_type(aa.action_type),
                                title=aa.summary or f"{agent.agent_id}: {aa.action_type}",
                                description=aa.summary or "",
                                proposed_payload=aa.action_payload or {},
                                risk_level=risk,
                                requires_approval=needs_approval,
                                confidence=0.85,
                            )
                            await ActionCenterService.create_proposal(db, payload, operator="system")

                # SUGGESTION: 从决策输出直接桥接
                elif stage == EvolutionStage.SUGGESTION and decision:
                    payload = ActionProposalCreate(
                        source_type="agent_decision",
                        source_id=str(decision_id or 0),
                        agent_id=agent.agent_id,
                        squad_id=squad_id,
                        action_type=AgentService._map_action_type(decision.get("action_type", "notify")),
                        title=f"{agent.agent_id} 建议: {decision.get('summary', decision_point)}",
                        description=decision.get("detail", decision.get("summary", "")),
                        proposed_payload=decision,
                        risk_level=RiskLevel.MEDIUM,
                        requires_approval=True,
                        confidence=confidence,
                    )
                    await ActionCenterService.create_proposal(db, payload, operator="system")
            except Exception:
                pass  # 桥接失败不阻塞主流程

        return {
            "agent_id": agent.agent_id,
            "decision_point": decision_point,
            "decision": decision,
            "stage": stage.value,
            "confidence": confidence,
            "rules_applied": applied_rule_ids,
            "decision_id": decision_id,
            "actions_created": actions_created,
        }

    # ── LLM 韧性辅助 ──────────────────────────────

    @staticmethod
    async def _call_with_resilience(
        agent: BaseAgent,
        decision_point: str,
        context: dict,
        db: Any = None,
    ) -> dict:
        """对 agent.decide() 调用添加熔断检查 + 缓存

        熔断关闭时直接调用 agent.decide() 并记录成功/失败；
        熔断开启时尝试命中缓存，都失败则返回降级决策。
        """
        cb_mgr = LLMCircuitBreakerManager()
        agent_id = agent.agent_id

        # 1. 检查熔断器
        if not cb_mgr.can_proceed(agent_id, decision_point):
            cached = await _get_cached_decision(db, agent_id, decision_point, context)
            if cached:
                return {**cached, "_from_cache": True, "_circuit_open": True}
            return {
                "confidence": 0.0,
                "error": "circuit_open",
                "summary": "断路器开启，跳过本轮",
            }

        # 2. 尝试缓存（跳过 LLM 调用）
        cached = await _get_cached_decision(db, agent_id, decision_point, context)
        if cached:
            return {**cached, "_from_cache": True}

        # 3. 正常调用
        try:
            result = await agent.decide(decision_point, context, db=db)
            cb_mgr.record_success(agent_id, decision_point)
            # 非错误响应才写入缓存
            if result.get("error") is None:
                await _set_cached_decision(db, agent_id, decision_point, context, result)
            return result
        except Exception as e:
            cb_mgr.record_failure(agent_id, decision_point)
            logger.warning(
                "Agent.decide() failed for %s/%s: %s",
                agent_id, decision_point, e,
            )
            # 尝试缓存兜底
            cached = await _get_cached_decision(db, agent_id, decision_point, context)
            if cached:
                return {**cached, "_from_cache": True, "_circuit_open": True}
            return {
                "confidence": 0.0,
                "error": str(e),
                "summary": f"Agent.decide() 异常: {e}",
            }

    # ── §5.3 输出验证 ──────────────────────────────

    @staticmethod
    def _sanitize_output(agent_id: str, decision: dict) -> dict:
        """轻量输出验证 — 不额外调 LLM

        非负检查会阻断 action；存在性/合理性检查仅发出警告。
        """
        warnings = []
        payload = decision.get("action_payload", {}) or {}

        # 存在性检查（warn, don't block）
        sku_code = payload.get("sku_code")
        if sku_code is not None and not isinstance(sku_code, str):
            warnings.append("sku_code invalid type")

        # 非负检查（block）
        for field in ["suggested_qty", "suggested_price", "final_price"]:
            val = payload.get(field)
            if val is not None and (not isinstance(val, (int, float)) or val < 0):
                warnings.append(f"{field}={val} is negative, dropping action")
                return {}  # 空决策 = 跳过该 action

        # price > cost 合理性（warn only）
        selling = payload.get("final_price") or payload.get("suggested_price")
        cost = payload.get("cost_price")
        if selling is not None and cost is not None and selling < cost:
            warnings.append(f"selling_price({selling}) < cost_price({cost})")

        if warnings:
            decision["_validation_warnings"] = warnings

        return decision

    @staticmethod
    def _is_high_risk_action(agent_id: str, decision: dict, stage: EvolutionStage) -> bool:
        """判定是否高风险操作（SEMI_AUTONOMOUS 阶段需审批）"""
        if stage != EvolutionStage.SEMI_AUTONOMOUS:
            return False

        # 涉及金额的操作
        amount_fields = {
            "A5": "suggested_replenish_qty",  # 补货数量（结合 cost_price 判断金额）
            "G3": "final_price",
            "A6": "suggested_price",
            "A3": "bid_suggestion",
        }
        # 检查金额相关
        for field, _ in amount_fields.items():
            if agent_id.startswith(field[0]) or agent_id == field:
                cost = abs(decision.get("cost_price", 0) or 0)
                qty = abs(decision.get("suggested_replenish_qty", 0) or 0)
                total_amount = cost * qty if qty > 0 else cost
                if total_amount > 500:  # 默认阈值 500 元
                    return True

        # 批量操作 > 20 个 SKU
        sku_count = len(decision.get("sku_codes", []) or [])
        if sku_count > 20:
            return True

        # 首次操作（无历史决策记录）
        if decision.get("is_first_operation"):
            return True

        return False

    # ── 动作中枢桥接辅助函数 ──────────────────────────────────

    ACTION_TYPE_MAP: dict[str, str] = {
        "replenish": "inventory_allocate",
        "price_review": "profit_review",
        "discount_review": "profit_review",
        "ad_action": "notify",
    }

    @staticmethod
    def _map_action_type(action_type: str) -> str:
        """将旧 AgentAction 类型映射到动作中枢 action_type"""
        return AgentService.ACTION_TYPE_MAP.get(action_type, action_type)

    @staticmethod
    def _derive_action_risk(
        action_type: str,
        payload: dict,
        stage: object,
    ) -> tuple[object, bool]:
        """根据操作类型、payload 和自治等级推导风险等级和是否需要审批"""
        from app.agentos.schemas import RiskLevel
        from app.agent.base import EvolutionStage

        urgency = payload.get("urgency", "")
        amount = abs(float(payload.get("suggested_qty", 0) or 0)) * \
                 abs(float(payload.get("cost_price", 0) or 0))

        if stage == EvolutionStage.FULL_AUTONOMOUS:
            return RiskLevel.LOW, False

        if stage == EvolutionStage.SUGGESTION:
            return RiskLevel.MEDIUM, True

        # SEMI_AUTONOMOUS
        if action_type == "replenish" and urgency == "urgent":
            return RiskLevel.HIGH, True
        if action_type == "discount_review":
            return RiskLevel.HIGH, True
        if action_type == "price_review":
            return RiskLevel.MEDIUM, True
        if action_type == "ad_action":
            return RiskLevel.CRITICAL if payload.get("status") == "critical" else RiskLevel.MEDIUM, False
        if amount > 5000:
            return RiskLevel.HIGH, True

        return RiskLevel.MEDIUM, True

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
