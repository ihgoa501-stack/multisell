"""A6 利润监控 Agent (Phase 2 MVP)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.4
- 输入 SKU、平台、国家、售价、采购成本、包装信息、物流费用、平台佣金、折扣、广告成本、退款成本
- 输出单件利润、毛利率、费用拆分、亏损风险、调价建议
- 数据不足时返回 insufficient_data
"""
from typing import Any, Optional
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent

REQUIRED_FIELDS = ["sku_code", "selling_price", "cost_price"]
OPTIONAL_FIELDS = [
    "platform", "country", "weight_kg", "length", "width", "height",
    "shipping_fee", "platform_fee", "platform_fee_rate", "fixed_fee",
    "discounts", "ad_cost_per_unit", "refund_rate",
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

    async def decide(self, decision_point: str, context: dict[str, Any]) -> dict[str, Any]:
        if decision_point == "profit_check":
            return self._check_profit(context)
        elif decision_point == "cost_optimization":
            return self._suggest_cost_optimization(context)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 利润检查（核心入口）
    # ──────────────────────────────
    def _check_profit(self, context: dict) -> dict:
        missing = _missing_fields(context, REQUIRED_FIELDS)
        if missing:
            return self._insufficient("profit_check", missing)

        sku_code = str(context.get("sku_code", ""))
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context["cost_price"])
        platform = str(context.get("platform", "unknown"))
        country = str(context.get("country", "unknown"))
        min_margin_threshold = _safe_float(context.get("min_margin_threshold", 15.0))

        # ---- 费用计算 ----
        fees = {}

        # 平台佣金
        platform_fee_rate = _safe_float(context.get("platform_fee_rate", 0))
        platform_fee = _safe_float(context.get("platform_fee", 0))
        if platform_fee == 0 and platform_fee_rate > 0:
            platform_fee = selling_price * platform_fee_rate / 100
        fees["platform_fee"] = round(platform_fee, 2)

        # 固定费用
        fixed_fee = _safe_float(context.get("fixed_fee", 0))
        fees["fixed_fee"] = round(fixed_fee, 2)

        # 物流费用
        shipping_fee = _safe_float(context.get("shipping_fee", 0))
        fees["shipping_fee"] = round(shipping_fee, 2)

        # 折扣摊销
        discounts = context.get("discounts", [])
        discount_rate = 0.0
        if isinstance(discounts, list) and len(discounts) > 0:
            # 简单求和折扣率
            for d in discounts:
                d_val = _safe_float(d.get("value", 0))
                d_type = str(d.get("type", "")).lower()
                if d_type in ("percentage", "coupon", "promotion", "member_discount"):
                    discount_rate += d_val
                elif d_type in ("fixed", "fixed_amount"):
                    discount_rate += d_val / selling_price * 100 if selling_price > 0 else 0
            discount_rate = min(discount_rate, 100.0)
        elif "discount_rate" in context:
            discount_rate = _safe_float(context["discount_rate"], 0)
        discount_amount = selling_price * discount_rate / 100
        fees["discount"] = round(discount_amount, 2)

        # 广告成本
        ad_cost = _safe_float(context.get("ad_cost_per_unit", 0))
        fees["ad_cost"] = round(ad_cost, 2)

        # 退款成本（按比例估算）
        refund_rate = _safe_float(context.get("refund_rate", 0))
        refund_cost = selling_price * refund_rate / 100
        fees["refund_cost"] = round(refund_cost, 2)

        # 总费用
        total_fees = sum(fees.values())
        fees["total"] = round(total_fees, 2)

        # ---- 利润计算 ----
        effective_revenue = selling_price - discount_amount
        profit_per_unit = round(effective_revenue - cost_price - total_fees, 2)
        gross_margin = round(
            (profit_per_unit / effective_revenue * 100) if effective_revenue > 0 else 0.0,
            2,
        )

        # ---- 风险评估 ----
        is_loss = profit_per_unit < 0
        below_threshold = gross_margin < min_margin_threshold

        alerts = []
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
            anomaly_reason = (
                f"毛利率 {gross_margin}% 低于阈值 {min_margin_threshold}%，"
                f"建议优化成本结构"
            )
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

        # ---- 检测费用异常 ----
        fee_warnings = []
        cost_ratio_threshold = 0.5
        if total_fees > effective_revenue * cost_ratio_threshold:
            fee_warnings.append(f"总费用占比过高({total_fees/effective_revenue*100:.0f}%)")
        if platform_fee > effective_revenue * 0.2:
            fee_warnings.append(f"平台佣金({platform_fee})占比较高")
        if shipping_fee > effective_revenue * 0.25:
            fee_warnings.append(f"物流费用({shipping_fee})占比较高")

        result = {
            "profit_check_status": "block" if is_loss else ("warn" if below_threshold else "allow"),
            "sku_code": sku_code,
            "platform": platform,
            "country": country,
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
        return result

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
            (selling_price - cost_price) / selling_price * 100 if selling_price > 0 else 0, 2
        )
        target_margin = _safe_float(context.get("target_margin", 20.0))

        suggestions = []

        # 提价建议
        if current_margin < target_margin:
            needed_revenue = cost_price / (1 - target_margin / 100)
            price_suggest = round(needed_revenue, 2)
            if price_suggest > selling_price:
                suggestions.append({
                    "type": "price_increase",
                    "current_price": selling_price,
                    "suggested_price": price_suggest,
                    "increase_pct": round((price_suggest - selling_price) / selling_price * 100, 1),
                    "description": f"提价至 ¥{price_suggest:.2f} 可达到 {target_margin}% 毛利率",
                })

            # 降本建议
            needed_cost = selling_price * (1 - target_margin / 100)
            if needed_cost < cost_price:
                suggestions.append({
                    "type": "cost_reduction",
                    "current_cost": cost_price,
                    "target_cost": round(needed_cost, 2),
                    "reduction_pct": round((cost_price - needed_cost) / cost_price * 100, 1),
                    "description": f"采购成本需降至 ¥{needed_cost:.2f} 以下",
                })

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
