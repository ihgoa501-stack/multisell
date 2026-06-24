"""A6 利润监控 Agent (Phase 2 — LLM 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.4

Phase 2 改进：LLM 参与核心决策，公式降级为安全网。
- 优先调用 LLM 做情景利润分析（考虑费用结构、行业基准、平台差异）
- LLM 失败/不合理时自动降级为公式兜底

Phase 1 原始设计：
- 输入 SKU、平台、国家、售价、采购成本、包装信息、物流费用、平台佣金、折扣、广告成本、退款成本
- 输出单件利润、毛利率、费用拆分、亏损风险、调价建议
- 数据不足时返回 insufficient_data
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService
from app.agent.data_service import AgentDataService

REQUIRED_FIELDS = ["sku_code", "selling_price", "cost_price"]
OPTIONAL_FIELDS = [
    "platform",
    "country",
    "weight_kg",
    "length",
    "width",
    "height",
    "shipping_fee",
    "platform_fee",
    "platform_fee_rate",
    "fixed_fee",
    "discounts",
    "ad_cost_per_unit",
    "refund_rate",
]


def _safe_float(v: Any, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def _missing_fields(context: dict, required: list[str]) -> list[str]:
    return [f for f in required if f not in context or context[f] is None]


@register_agent
class A6ProfitWatchAgent(BaseAgent):
    agent_id = "A6"
    name = "利润监控 Agent"
    description = "SKU 级利润拆分和毛利率预警，支持费用拆分、亏损检测和调价建议"
    decision_points = ["profit_check", "cost_optimization"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "profit_check": EvolutionStage.SEMI_AUTONOMOUS,
        "cost_optimization": EvolutionStage.SUGGESTION,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        if db is not None and decision_point == "profit_check":
            context = await AgentDataService.fill_sku_context(db, context)
        if decision_point == "profit_check":
            return await self._check_profit(context, db=db)
        elif decision_point == "cost_optimization":
            return self._suggest_cost_optimization(context)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 利润检查（LLM 增强版）
    # ──────────────────────────────
    async def _check_profit(self, context: dict, db: Any = None) -> dict:
        """利润检查主入口

        执行流程：
        ① 公式计算（安全网）
        ② LLM 分析（SEMI_AUTONOMOUS+ 阶段尝试）
        ③ 结果仲裁
        """
        missing = _missing_fields(context, REQUIRED_FIELDS)
        if missing:
            return self._insufficient("profit_check", missing)

        # ---- ① 公式计算（安全网） ----
        formula_result = self._formula_check(context)

        # ---- ② LLM 分析 ----
        llm_decision = None
        stage = self.get_stage("profit_check")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_context = {
                "sku_code": formula_result.get("sku_code", ""),
                "selling_price": formula_result.get("selling_price", 0),
                "cost_price": formula_result.get("cost_price", 0),
                "effective_revenue": formula_result.get("effective_revenue", 0),
                "profit_per_unit": formula_result.get("profit_per_unit", 0),
                "gross_margin": formula_result.get("gross_margin", 0),
                "is_loss": formula_result.get("is_loss", False),
                "fee_breakdown": str(formula_result.get("fee_breakdown", {})),
                "platform": formula_result.get("platform", "unknown"),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A6", "profit_check", llm_context, db=db
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
                    "risk_reason", formula_result["anomaly_reason"]
                ),
                "cost_suggestion": llm_decision.get("cost_suggestion", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "risk_reason": formula_result["anomaly_reason"],
                "cost_suggestion": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ---- ④ LLM 自然语言解释 ----
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A6",
                {
                    "sku": result.get("sku_code", ""),
                    "selling_price": result.get("selling_price", 0),
                    "cost_price": result.get("cost_price", 0),
                    "fee_breakdown": str(result.get("fee_breakdown", {})),
                    "profit": result.get("profit_per_unit", 0),
                    "margin": result.get("gross_margin", 0),
                    "is_loss": result.get("is_loss", False),
                    "anomaly_reason": result.get("anomaly_reason", ""),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式计算（安全网）
    # ──────────────────────────────
    def _formula_check(self, context: dict) -> dict:
        """纯公式利润检查，作为 LLM 失败时的兜底"""
        sku_code = str(context.get("sku_code", ""))
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context["cost_price"])
        platform = str(context.get("platform", "unknown"))
        min_margin_threshold = _safe_float(context.get("min_margin_threshold", 15.0))

        fees = {}

        platform_fee_rate = _safe_float(context.get("platform_fee_rate", 0))
        platform_fee = _safe_float(context.get("platform_fee", 0))
        if platform_fee == 0 and platform_fee_rate > 0:
            platform_fee = selling_price * platform_fee_rate / 100
        fees["platform_fee"] = round(platform_fee, 2)

        fixed_fee = _safe_float(context.get("fixed_fee", 0))
        fees["fixed_fee"] = round(fixed_fee, 2)

        shipping_fee = _safe_float(context.get("shipping_fee", 0))
        fees["shipping_fee"] = round(shipping_fee, 2)

        discounts = context.get("discounts", [])
        discount_rate = 0.0
        if isinstance(discounts, list) and len(discounts) > 0:
            for d in discounts:
                d_val = _safe_float(d.get("value", 0))
                d_type = str(d.get("type", "")).lower()
                if d_type in ("percentage", "coupon", "promotion", "member_discount"):
                    discount_rate += d_val
                elif d_type in ("fixed", "fixed_amount"):
                    discount_rate += (
                        d_val / selling_price * 100 if selling_price > 0 else 0
                    )
            discount_rate = min(discount_rate, 100.0)
        elif "discount_rate" in context:
            discount_rate = _safe_float(context["discount_rate"], 0)
        discount_amount = selling_price * discount_rate / 100
        fees["discount"] = round(discount_amount, 2)

        ad_cost = _safe_float(context.get("ad_cost_per_unit", 0))
        fees["ad_cost"] = round(ad_cost, 2)

        refund_rate = _safe_float(context.get("refund_rate", 0))
        refund_cost = selling_price * refund_rate / 100
        fees["refund_cost"] = round(refund_cost, 2)

        total_fees = sum(fees.values())
        fees["total"] = round(total_fees, 2)

        effective_revenue = selling_price - discount_amount
        profit_per_unit = round(effective_revenue - cost_price - total_fees, 2)
        gross_margin = round(
            (profit_per_unit / effective_revenue * 100)
            if effective_revenue > 0
            else 0.0,
            2,
        )

        is_loss = profit_per_unit < 0
        below_threshold = gross_margin < min_margin_threshold

        anomaly_reason = ""
        optimization_suggestions = []
        confidence = 0.90

        if is_loss:
            anomaly_reason = (
                f"单件亏损 ¥{abs(profit_per_unit):.2f}，"
                f"营收 ¥{effective_revenue:.2f} 不足以覆盖成本 ¥{cost_price:.2f} + 费用 ¥{total_fees:.2f}"
            )
            optimization_suggestions = [
                "考虑提高售价",
                "降低采购成本",
                "减少折扣力度",
                "优化物流渠道降低成本",
            ]
            confidence = 0.95
        elif below_threshold:
            anomaly_reason = f"毛利率 {gross_margin}% 低于阈值 {min_margin_threshold}%，建议优化成本结构"
            optimization_suggestions = [
                "适当提高售价",
                "检查平台佣金是否有优化空间",
                "评估广告成本是否可控",
            ]
            confidence = 0.88
        else:
            anomaly_reason = "毛利率正常，在安全范围内"
            optimization_suggestions = ["维持当前策略，定期监控"]
            confidence = 0.85

        fee_warnings = []
        cost_ratio_threshold = 0.5
        if total_fees > effective_revenue * cost_ratio_threshold:
            fee_warnings.append(
                f"总费用占比过高({total_fees / effective_revenue * 100:.0f}%)"
            )
        if platform_fee > effective_revenue * 0.2:
            fee_warnings.append(f"平台佣金({platform_fee})占比较高")
        if shipping_fee > effective_revenue * 0.25:
            fee_warnings.append(f"物流费用({shipping_fee})占比较高")

        return {
            "profit_check_status": "block"
            if is_loss
            else ("warn" if below_threshold else "allow"),
            "sku_code": sku_code,
            "platform": platform,
            "selling_price": selling_price,
            "cost_price": cost_price,
            "effective_revenue": round(effective_revenue, 2),
            "discount_rate": round(discount_rate, 2),
            "profit_per_unit": profit_per_unit,
            "gross_margin": gross_margin,
            "min_margin_threshold": min_margin_threshold,
            "is_loss": is_loss,
            "below_threshold": below_threshold,
            "fee_breakdown": fees,
            "fee_warnings": fee_warnings,
            "anomaly_reason": anomaly_reason,
            "optimization_suggestions": optimization_suggestions,
            "confidence": confidence,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 返回的结构化结果是否合理"""
        if "is_loss" in result and not isinstance(result.get("is_loss"), bool):
            return False
        risk = result.get("risk_level")
        if risk is not None and risk not in ("low", "medium", "high"):
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

    # ──────────────────────────────
    #  2. 成本优化建议
    # ──────────────────────────────
    def _suggest_cost_optimization(self, context: dict) -> dict:
        missing = _missing_fields(context, ["sku_code", "selling_price", "cost_price"])
        if missing:
            return self._insufficient("cost_optimization", missing)

        sku_code = str(context.get("sku_code", ""))
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context["cost_price"])
        current_margin = round(
            (selling_price - cost_price) / selling_price * 100
            if selling_price > 0
            else 0,
            2,
        )
        target_margin = _safe_float(context.get("target_margin", 20.0))

        suggestions = []

        # 提价建议
        if current_margin < target_margin:
            needed_revenue = cost_price / (1 - target_margin / 100)
            price_suggest = round(needed_revenue, 2)
            if price_suggest > selling_price:
                suggestions.append(
                    {
                        "type": "price_increase",
                        "current_price": selling_price,
                        "suggested_price": price_suggest,
                        "increase_pct": round(
                            (price_suggest - selling_price) / selling_price * 100, 1
                        ),
                        "description": f"提价至 ¥{price_suggest:.2f} 可达到 {target_margin}% 毛利率",
                    }
                )

            # 降本建议
            needed_cost = selling_price * (1 - target_margin / 100)
            if needed_cost < cost_price:
                suggestions.append(
                    {
                        "type": "cost_reduction",
                        "current_cost": cost_price,
                        "target_cost": round(needed_cost, 2),
                        "reduction_pct": round(
                            (cost_price - needed_cost) / cost_price * 100, 1
                        ),
                        "description": f"采购成本需降至 ¥{needed_cost:.2f} 以下",
                    }
                )

        return {
            "sku_code": sku_code,
            "current_margin": current_margin,
            "target_margin": target_margin,
            "suggestions": suggestions,
            "confidence": 0.85,
        }

    def _insufficient(self, point: str, missing: list[str]) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": point,
            "missing_fields": missing,
            "message": f"缺少必要字段: {', '.join(missing)}，请补充完整数据",
            "confidence": 0.0,
        }
