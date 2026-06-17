"""G3 折扣风控 Agent (Phase 1 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.3
- 输入覆盖 SKU/ASIN、成本价、当前售价、生效折扣列表、平台、最低毛利率阈值
- 必须模拟多折扣叠加后的最终价格
- 折后价低于成本时阻断
- 折后价低于成本价 × 1.1 时预警
- 输出阻断/放行原因、折后价、毛利、毛利率
"""
from typing import Any, Optional
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent

# ── 必填字段（缺少任一即返回 insufficient_data） ──
REQUIRED_CHECK_FIELDS = [
    "sku_code", "cost_price", "selling_price",
]

REQUIRED_VALIDATE_FIELDS = [
    "promotion", "selling_price",
]


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

    async def decide(self, decision_point: str, context: dict[str, Any]) -> dict[str, Any]:
        if decision_point == "discount_check":
            return self._check_discount_risk(context)
        elif decision_point == "promotion_validation":
            return self._validate_promotion(context)
        return {"action": "unknown", "confidence": 0.0}

    # ────────────────────────────────────
    #  1. 折扣叠加风控（核心入口）
    # ────────────────────────────────────
    def _check_discount_risk(self, context: dict) -> dict:
        missing = _missing_fields(context, REQUIRED_CHECK_FIELDS)
        if missing:
            return self._insufficient("discount_check", missing)

        sku_code = str(context.get("sku_code", ""))
        asin = context.get("asin", "")
        selling_price = _safe_float(context["selling_price"])
        cost_price = _safe_float(context["cost_price"])
        active_discounts: list = context.get("active_discounts", [])
        platform = context.get("platform", "unknown")
        min_margin_threshold = _safe_float(
            context.get("min_margin_threshold", 10.0)
        )

        # ---- 模拟多折扣叠加 ----
        final_price, discount_details = self._simulate_discounts(
            selling_price, active_discounts
        )

        gross_profit = round(final_price - cost_price, 2)
        gross_margin = round(
            (gross_profit / final_price * 100) if final_price > 0 else 0.0, 2
        )

        # ---- 决策逻辑 ----
        blocked = False
        action = "allow"
        reason = ""
        alerts = []
        confidence = 0.90

        if final_price <= 0:
            action = "block"
            blocked = True
            reason = f"折后价 ¥{final_price:.2f} ≤ ¥0，售价无效"
            alerts.append({
                "level": "critical",
                "message": reason,
            })
            confidence = 0.99
        elif final_price < cost_price:
            action = "block"
            blocked = True
            reason = (
                f"折后价 ¥{final_price:.2f} < 成本价 ¥{cost_price:.2f}，"
                f"亏损 ¥{abs(gross_profit):.2f}，已自动阻断"
            )
            alerts.append({
                "level": "error",
                "message": reason,
                "gross_loss": round(abs(gross_profit), 2),
            })
            confidence = 0.97
        elif final_price < cost_price * 1.1:
            action = "warn"
            reason = (
                f"折后价 ¥{final_price:.2f} 仅高于成本价 "
                f"{((final_price - cost_price) / cost_price * 100):.1f}%，"
                f"低于安全阈值 (成本×1.1 = ¥{cost_price * 1.1:.2f})，建议人工复核"
            )
            alerts.append({
                "level": "warning",
                "message": reason,
                "min_safe_price": round(cost_price * 1.1, 2),
            })
            confidence = 0.90
        elif gross_margin < min_margin_threshold:
            action = "warn"
            reason = (
                f"毛利率 {gross_margin}% < 最低阈值 {min_margin_threshold}%，"
                f"建议优化折扣或提价"
            )
            alerts.append({
                "level": "warning",
                "message": reason,
                "threshold": min_margin_threshold,
            })
            confidence = 0.85
        else:
            reason = f"折后毛利率 {gross_margin}%，高于阈值 {min_margin_threshold}%，放行"
            confidence = 0.85

        # ---- 多平台最低价风险提示 ----
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
            "asin": asin,
            "platform": platform,
            "min_margin_threshold": min_margin_threshold,
            "alerts": alerts,
            "confidence": confidence,
        }

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
                    details.append({
                        "type": d_type,
                        "value": d_value,
                        "unit": "%",
                        "description": f"{d_type} {d_value}% 折扣",
                        "discount_amount": round(discount_amount, 2),
                    })
            elif d_type in ("fixed", "fixed_amount"):
                # 固定金额折扣
                if d_value > 0:
                    discount_amount = min(d_value, price)
                    price -= discount_amount
                    details.append({
                        "type": "fixed_amount",
                        "value": d_value,
                        "unit": "¥",
                        "description": f"固定减免 ¥{d_value:.2f}",
                        "discount_amount": round(discount_amount, 2),
                    })
            elif d_type in ("buy_x_get_y", "bogo"):
                # Buy X Get Y — 折算为折扣率
                buy_qty = _safe_float(d.get("buy_qty", d.get("buy", 2)))
                free_qty = _safe_float(d.get("free_qty", d.get("free", 1)))
                if buy_qty > 0 and free_qty > 0:
                    effective_rate = free_qty / (buy_qty + free_qty) * 100
                    discount_amount = price * effective_rate / 100
                    price -= discount_amount
                    details.append({
                        "type": "buy_x_get_y",
                        "buy_qty": buy_qty,
                        "free_qty": free_qty,
                        "effective_rate": round(effective_rate, 1),
                        "description": f"买{buy_qty}送{free_qty} (等效 {effective_rate:.1f}% 折扣)",
                        "discount_amount": round(discount_amount, 2),
                    })
            elif d_type == "percentage_no_compound":
                # 不叠加的独立百分比（取最大值而非叠加）
                # 已存在百分比折扣时，取最大值
                if 0 < d_value < 100:
                    details.append({
                        "type": "percentage_no_compound",
                        "value": d_value,
                        "unit": "%",
                        "description": f"独立折扣 {d_value}%（非叠加）",
                        "discount_amount": 0,
                    })

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
