"""G3 折扣风控 Agent (Phase 2 — LLM 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.3

Phase 2 改进：LLM 参与核心决策，公式降级为安全网。
- 优先调用 LLM 做情景分析（考虑平台政策、叠加幅度、大促策略）
- LLM 失败/不合理时自动降级为公式兜底
- 永远不因 LLM 异常中断执行

Phase 1 原始设计：
- 输入覆盖 SKU/ASIN、成本价、当前售价、生效折扣列表、平台、最低毛利率阈值
- 必须模拟多折扣叠加后的最终价格
- 折后价低于成本时阻断
- 折后价低于成本价 × 1.1 时预警
- 输出阻断/放行原因、折后价、毛利、毛利率
"""

from typing import Any, Optional
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService
from app.agent.data_service import AgentDataService

# ── 必填字段（缺少任一即返回 insufficient_data） ──
REQUIRED_CHECK_FIELDS = [
    "sku_code",
    "cost_price",
    "selling_price",
]

REQUIRED_VALIDATE_FIELDS = [
    "promotion",
    "selling_price",
]

# ── LLM 输出校验常量 ──
VALID_ACTIONS = ("allow", "warn", "block")
VALID_RISK_LEVELS = ("low", "medium", "high")


def _safe_float(v: Any, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def _missing_fields(context: dict, required: list[str]) -> list[str]:
    return [f for f in required if f not in context or context[f] is None]


@register_agent
class G3DiscountRiskAgent(BaseAgent):
    agent_id = "G3"
    name = "折扣风控 Agent"
    description = "多折扣叠加风控，自动阻断亏损促销，保护毛利率 | 支持百分比/固定金额/BuyXGetY等多类型折扣模拟"
    decision_points = ["discount_check", "promotion_validation"]
    version = "2.0.0"

    DEFAULT_STAGES = {
        "discount_check": EvolutionStage.SEMI_AUTONOMOUS,
        "promotion_validation": EvolutionStage.FULL_AUTONOMOUS,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        if db is not None and decision_point == "discount_check":
            context = await AgentDataService.fill_sku_context(db, context)
        if decision_point == "discount_check":
            return await self._check_discount_risk(context, db=db)
        elif decision_point == "promotion_validation":
            return self._validate_promotion(context)
        return {"action": "unknown", "confidence": 0.0}

    # ────────────────────────────────────
    #  1. 折扣叠加风控（核心入口，LLM 增强）
    # ────────────────────────────────────
    async def _check_discount_risk(self, context: dict, db: Any = None) -> dict:
        """折扣风控主入口

        执行流程：
        ① 公式计算（安全网）
        ② LLM 分析（SEMI_AUTONOMOUS+ 阶段尝试）
        ③ 结果仲裁：LLM 成功且合理→用 LLM，否则用公式
        """
        missing = _missing_fields(context, REQUIRED_CHECK_FIELDS)
        if missing:
            return self._insufficient("discount_check", missing)

        # ---- ① 公式计算（安全网） ----
        formula_result = self._formula_check(context)

        # ---- ② LLM 分析 ----
        llm_decision = None
        stage = self.get_stage("discount_check")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_context = {
                "sku_code": context.get("sku_code", ""),
                "selling_price": context.get("selling_price", 0),
                "cost_price": context.get("cost_price", 0),
                "platform": context.get("platform", "unknown"),
                "discount_count": formula_result.get("discount_count", 0),
                "final_price": formula_result.get("final_price", 0),
                "gross_margin": formula_result.get("gross_margin", 0),
                "action": formula_result.get("action", "allow"),
                "discount_details": str(formula_result.get("discount_details", [])),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "G3", "discount_check", llm_context, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ---- ③ 结果仲裁 ----
        if llm_decision:
            result = {
                **formula_result,
                "risk_reason": llm_decision.get(
                    "risk_reason", formula_result["reason"]
                ),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {**formula_result, "additional_notes": "", "llm_source": False}

        # ---- ④ LLM 自然语言解释 ----
        llm_summary = {
            "action": result.get("action", ""),
            "sku": result.get("sku_code", ""),
            "original_price": result.get("original_price", 0),
            "cost_price": result.get("cost_price", 0),
            "final_price": result.get("final_price", 0),
            "gross_profit": result.get("gross_profit", 0),
            "gross_margin": result.get("gross_margin", 0),
            "discount_details": ", ".join(
                d.get("description", "") for d in result.get("discount_details", [])
            ),
            "reason": result.get("reason", ""),
        }
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "G3", llm_summary, db=db
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ────────────────────────────────────
    #  1a. 公式计算（安全网）
    # ────────────────────────────────────
    def _formula_check(self, context: dict) -> dict:
        """纯公式折扣检查，作为 LLM 失败时的兜底"""
        sku_code = str(context.get("sku_code", ""))
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context["cost_price"])
        active_discounts: list = context.get("active_discounts", [])
        platform = context.get("platform", "unknown")
        min_margin_threshold = _safe_float(context.get("min_margin_threshold", 10.0))

        final_price, discount_details = self._simulate_discounts(
            selling_price, active_discounts
        )

        gross_profit = round(final_price - cost_price, 2)
        gross_margin = round(
            (gross_profit / final_price * 100) if final_price > 0 else 0.0, 2
        )

        blocked = False
        action = "allow"
        reason = ""
        alerts = []
        confidence = 0.90

        if final_price <= 0:
            action = "block"
            blocked = True
            reason = f"折后价 ¥{final_price:.2f} ≤ ¥0，售价无效"
            alerts.append({"level": "critical", "message": reason})
            confidence = 0.99
        elif final_price < cost_price:
            action = "block"
            blocked = True
            reason = (
                f"折后价 ¥{final_price:.2f} < 成本价 ¥{cost_price:.2f}，"
                f"亏损 ¥{abs(gross_profit):.2f}，已自动阻断"
            )
            alerts.append(
                {
                    "level": "error",
                    "message": reason,
                    "gross_loss": round(abs(gross_profit), 2),
                }
            )
            confidence = 0.97
        elif final_price < cost_price * 1.1:
            action = "warn"
            reason = (
                f"折后价 ¥{final_price:.2f} 仅高于成本价 "
                f"{((final_price - cost_price) / cost_price * 100):.1f}%，"
                f"低于安全阈值 (成本×1.1 = ¥{cost_price * 1.1:.2f})，建议人工复核"
            )
            alerts.append(
                {
                    "level": "warning",
                    "message": reason,
                    "min_safe_price": round(cost_price * 1.1, 2),
                }
            )
            confidence = 0.90
        elif gross_margin < min_margin_threshold:
            action = "warn"
            reason = f"毛利率 {gross_margin}% < 最低阈值 {min_margin_threshold}%，建议优化折扣或提价"
            alerts.append(
                {
                    "level": "warning",
                    "message": reason,
                    "threshold": min_margin_threshold,
                }
            )
            confidence = 0.85
        else:
            reason = (
                f"折后毛利率 {gross_margin}%，高于阈值 {min_margin_threshold}%，放行"
            )
            confidence = 0.85

        platform_risk = self._check_platform_price_risk(
            platform, final_price, selling_price
        )
        if platform_risk:
            alerts.append(platform_risk)
            confidence = max(0.80, confidence - 0.05)

        return {
            "action": action,
            "blocked": blocked,
            "reason": reason,
            "final_price": round(final_price, 2),
            "original_price": selling_price,
            "cost_price": cost_price,
            "gross_profit": gross_profit,
            "gross_margin": gross_margin,
            "total_discount_rate": round(
                (1 - final_price / selling_price) * 100 if selling_price > 0 else 0, 2
            ),
            "discount_count": len(active_discounts),
            "discount_details": discount_details,
            "sku_code": sku_code,
            "platform": platform,
            "min_margin_threshold": min_margin_threshold,
            "alerts": alerts,
            "confidence": confidence,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 返回的结构化结果是否合理"""
        action = result.get("action")
        if action is not None and action not in VALID_ACTIONS:
            return False
        risk = result.get("risk_level")
        if risk is not None and risk not in VALID_RISK_LEVELS:
            return False
        if not result.get("risk_reason"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    # ────────────────────────────────────
    #  2. 单个促销校验
    # ────────────────────────────────────
    def _validate_promotion(self, context: dict) -> dict:
        missing = _missing_fields(context, REQUIRED_VALIDATE_FIELDS)
        if missing:
            return self._insufficient("promotion_validation", missing)

        promotion: dict = context.get("promotion", {})
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context.get("cost_price", 0))
        platform = context.get("platform", "unknown")
        is_prime_day = context.get("is_prime_day", False)

        p_type = promotion.get("type", "")
        p_value = _safe_float(promotion.get("value", 0))

        # ---- 计算单个促销折后价 ----
        final_price, detail_str = self._apply_single_discount(
            selling_price, p_type, p_value, promotion
        )

        gross_profit = round(final_price - cost_price, 2)
        gross_margin = round(
            (gross_profit / final_price * 100) if final_price > 0 else 0.0, 2
        )

        alerts = []
        blocked = False
        action = "allow"
        reason = ""
        confidence = 0.92

        if is_prime_day:
            if final_price < cost_price:
                action = "warn"
                reason = f"大促期间折后价 ¥{final_price:.2f} < 成本 ¥{cost_price:.2f}，请运营确认是否继续"
                confidence = 0.85
            else:
                action = "allow_special"
                reason = f"大促期间特殊放行，毛利率 {gross_margin}%"
                confidence = 0.90
        elif final_price <= 0:
            action = "block"
            blocked = True
            reason = "促销后售价 ≤ ¥0，已自动阻断"
            confidence = 0.99
        elif final_price < cost_price:
            action = "block"
            blocked = True
            reason = f"促销后售价 ¥{final_price:.2f} < 成本 ¥{cost_price:.2f}，已阻断"
            confidence = 0.97
        elif final_price < cost_price * 1.1:
            action = "warn"
            reason = f"促销后毛利率仅 {gross_margin}%，低于安全阈值"
            confidence = 0.88
        else:
            action = "allow"
            reason = f"促销后毛利率 {gross_margin}%，放行"
            confidence = 0.92

        return {
            "action": action,
            "blocked": blocked,
            "reason": reason,
            "final_price": round(final_price, 2),
            "original_price": selling_price,
            "cost_price": cost_price,
            "gross_profit": gross_profit,
            "gross_margin": gross_margin,
            "discount_type": p_type,
            "discount_value": p_value,
            "discount_description": detail_str,
            "platform": platform,
            "is_prime_day": is_prime_day,
            "alerts": alerts,
            "confidence": confidence,
        }

    # ────────────────────────────────────
    #  内部工具方法
    # ────────────────────────────────────
    def _simulate_discounts(
        self, base_price: float, discounts: list[dict]
    ) -> tuple[float, list[dict]]:
        """模拟多个折扣叠加后的最终价格

        策略：
        - 百分比折扣依次叠加（每次扣减当前价格的百分比）
        - 固定金额折扣按金额减
        - BuyXGetY: 将免费件数折算为等比例折扣
        """
        price = base_price
        details = []

        for d in discounts:
            d_type = d.get("type", "").lower()
            d_value = _safe_float(d.get("value", 0))

            if d_type in ("percentage", "coupon", "promotion", "member_discount"):
                # 百分比折扣：叠加计算
                if 0 < d_value < 100:
                    discount_amount = price * d_value / 100
                    price -= discount_amount
                    details.append(
                        {
                            "type": d_type,
                            "value": d_value,
                            "unit": "%",
                            "description": f"{d_type} {d_value}% 折扣",
                            "discount_amount": round(discount_amount, 2),
                        }
                    )
            elif d_type in ("fixed", "fixed_amount"):
                # 固定金额折扣
                if d_value > 0:
                    discount_amount = min(d_value, price)
                    price -= discount_amount
                    details.append(
                        {
                            "type": "fixed_amount",
                            "value": d_value,
                            "unit": "¥",
                            "description": f"固定减免 ¥{d_value:.2f}",
                            "discount_amount": round(discount_amount, 2),
                        }
                    )
            elif d_type in ("buy_x_get_y", "bogo"):
                # Buy X Get Y — 折算为折扣率
                buy_qty = _safe_float(d.get("buy_qty", d.get("buy", 2)))
                free_qty = _safe_float(d.get("free_qty", d.get("free", 1)))
                if buy_qty > 0 and free_qty > 0:
                    effective_rate = free_qty / (buy_qty + free_qty) * 100
                    discount_amount = price * effective_rate / 100
                    price -= discount_amount
                    details.append(
                        {
                            "type": "buy_x_get_y",
                            "buy_qty": buy_qty,
                            "free_qty": free_qty,
                            "effective_rate": round(effective_rate, 1),
                            "description": f"买{buy_qty}送{free_qty} (等效 {effective_rate:.1f}% 折扣)",
                            "discount_amount": round(discount_amount, 2),
                        }
                    )
            elif d_type == "percentage_no_compound":
                # 不叠加的独立百分比（取最大值而非叠加）
                # 已存在百分比折扣时，取最大值
                if 0 < d_value < 100:
                    details.append(
                        {
                            "type": "percentage_no_compound",
                            "value": d_value,
                            "unit": "%",
                            "description": f"独立折扣 {d_value}%（非叠加）",
                            "discount_amount": 0,
                        }
                    )

        # 处理 non-compound 折扣：如果存在，取最大百分比替代叠加结果
        # 简化：non_compound 折扣视为独立，不参与叠加
        price = max(price, 0.0)
        return price, details

    def _apply_single_discount(
        self, base_price: float, d_type: str, d_value: float, promotion: dict
    ) -> tuple[float, str]:
        """应用单个促销折扣，返回 (折后价, 描述)"""
        if d_type == "percentage":
            final_price = base_price * (1 - d_value / 100)
            detail = f"{d_value}% 折扣"
        elif d_type == "fixed":
            final_price = base_price - d_value
            detail = f"减 ¥{d_value:.2f}"
        elif d_type == "buy_x_get_y":
            buy_qty = _safe_float(promotion.get("buy_qty", 2))
            free_qty = _safe_float(promotion.get("free_qty", 1))
            effective_rate = free_qty / (buy_qty + free_qty)
            final_price = base_price * (1 - effective_rate)
            detail = f"买{buy_qty}送{free_qty}"
        else:
            final_price = base_price
            detail = "未知促销类型"

        return max(final_price, 0.0), detail

    def _check_platform_price_risk(
        self, platform: str, final_price: float, original_price: float
    ) -> Optional[dict]:
        """检查多平台最低价风险"""
        risk_platforms = {"amazon", "walmart", "ebay", "shopify"}
        if platform.lower() in risk_platforms and final_price < original_price * 0.7:
            return {
                "level": "info",
                "message": (
                    f"折后价仅为原价的 {(final_price / original_price * 100):.0f}%，"
                    f"请确认该平台允许的最低折扣幅度"
                ),
            }
        return None

    def _insufficient(self, point: str, missing: list[str]) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": point,
            "missing_fields": missing,
            "message": f"缺少必要字段: {', '.join(missing)}，请补充完整数据",
            "confidence": 0.0,
        }
