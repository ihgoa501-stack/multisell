"""G3 折扣风控 Agent

设计依据: final-integrated-solution.md §6.6
- L1全自动防错 + 实时阻断
- 检测折扣叠加 (Coupon + Promotion + Member Discount)
- 模拟折后价格 vs 成本价
"""
from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent


@register_agent
class G3DiscountRiskAgent(BaseAgent):
    agent_id = "G3"
    name = "折扣风控 Agent"
    description = "自动检测折扣叠加风险，阻断零元订单，保护利润率"
    decision_points = ["discount_check", "promotion_validation"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "discount_check": EvolutionStage.SEMI_AUTONOMOUS,
        "promotion_validation": EvolutionStage.FULL_AUTONOMOUS,
    }

    async def decide(self, decision_point: str, context: dict[str, Any]) -> dict[str, Any]:
        if decision_point == "discount_check":
            return self._check_discount_risk(context)
        elif decision_point == "promotion_validation":
            return self._validate_promotion(context)
        return {"action": "unknown", "confidence": 0.0}

    def _check_discount_risk(self, context: dict) -> dict:
        product_name = context.get("product_name", "")
        selling_price = float(context.get("selling_price", 0))
        cost_price = float(context.get("cost_price", 0))
        active_discounts = context.get("active_discounts", [])

        total_discount_rate = 0.0
        discount_details = []
        for d in active_discounts:
            d_type = d.get("type", "")
            d_value = float(d.get("value", 0))
            if d_type == "coupon":
                total_discount_rate += d_value
                discount_details.append(f"Coupon: {d_value}%")
            elif d_type == "promotion":
                total_discount_rate += d_value
                discount_details.append(f"Promotion: {d_value}%")
            elif d_type == "member_discount":
                total_discount_rate += d_value
                discount_details.append(f"会员折扣: {d_value}%")

        final_price = selling_price * (1 - total_discount_rate / 100)
        discount_count = len(active_discounts)

        alerts = []
        blocked = False
        confidence = 0.95

        if discount_count >= 2:
            if final_price <= 0:
                action = "block"
                blocked = True
                alerts.append({
                    "level": "critical",
                    "message": f"[{product_name}] 折扣叠加后售价 ≤ ¥0，已自动阻断",
                    "final_price": round(final_price, 2),
                    "discounts": discount_details,
                })
                confidence = 0.99
            elif final_price < cost_price:
                action = "block"
                blocked = True
                alerts.append({
                    "level": "error",
                    "message": f"[{product_name}] 折后价 ¥{final_price:.2f} < 成本 ¥{cost_price:.2f}，已阻断",
                    "final_price": round(final_price, 2),
                    "cost_price": cost_price,
                    "discounts": discount_details,
                })
                confidence = 0.97
            elif final_price < cost_price * 1.1:
                action = "warn"
                alerts.append({
                    "level": "warning",
                    "message": f"[{product_name}] 折后利润率不足10%",
                    "final_price": round(final_price, 2),
                    "profit_margin": round((final_price - cost_price) / cost_price * 100, 2),
                    "discounts": discount_details,
                })
                confidence = 0.90
            else:
                action = "allow"
                confidence = 0.85
        else:
            action = "allow"
            confidence = 0.80

        return {
            "action": action,
            "blocked": blocked,
            "final_price": round(final_price, 2),
            "total_discount_rate": total_discount_rate,
            "discount_count": discount_count,
            "discount_details": discount_details,
            "alerts": alerts,
            "confidence": confidence,
        }

    def _validate_promotion(self, context: dict) -> dict:
        promotion = context.get("promotion", {})
        p_type = promotion.get("type", "")
        p_value = float(promotion.get("value", 0))
        selling_price = float(context.get("selling_price", 0))
        cost_price = float(context.get("cost_price", 0))
        is_prime_day = context.get("is_prime_day", False)

        alerts = []
        blocked = False
        discount_rate = 0.0

        if p_type == "percentage":
            discount_rate = p_value
            final_price = selling_price * (1 - p_value / 100)
        elif p_type == "fixed":
            discount_rate = p_value / selling_price * 100
            final_price = selling_price - p_value
        else:
            final_price = selling_price

        if is_prime_day:
            if final_price < cost_price:
                action = "warn"
                alerts.append({
                    "level": "warning",
                    "message": "大促期间折后价低于成本，请运营确认",
                    "final_price": round(final_price, 2),
                })
                confidence = 0.85
            else:
                action = "allow_special"
                alerts.append({
                    "level": "info",
                    "message": "大促期间特殊放行",
                })
                confidence = 0.90
        elif final_price <= 0:
            action = "block"
            blocked = True
            alerts.append({"level": "critical", "message": "促销后售价≤0，已阻断"})
            confidence = 0.99
        elif final_price < cost_price:
            action = "block"
            blocked = True
            alerts.append({"level": "error", "message": "促销后售价低于成本，已阻断"})
            confidence = 0.97
        elif final_price < cost_price * 1.15:
            action = "warn"
            alerts.append({"level": "warning", "message": "促销后利润率不足15%"})
            confidence = 0.88
        else:
            action = "allow"
            confidence = 0.92

        return {
            "action": action,
            "blocked": blocked,
            "final_price": round(final_price, 2),
            "discount_rate": discount_rate,
            "alerts": alerts,
            "confidence": confidence,
        }
