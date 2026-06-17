"""A5 库存预警 Agent

设计依据: final-integrated-solution.md §6.3
- L1自动监控 + L1自动补货建议，关键决策L2人工确认
- 三级预警体系 (红/黄/绿)
- 补货数量公式 (Holt-Winters简化版)
- 物流方式决策树
"""
from typing import Any
from datetime import datetime, timezone, timedelta
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent


@register_agent
class A5InventoryAlertAgent(BaseAgent):
    agent_id = "A5"
    name = "库存预警 Agent"
    description = "智能库存监控、三级预警、补货建议与物流方式推荐"
    decision_points = ["stock_alert", "replenishment_plan", "logistics_choice"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "stock_alert": EvolutionStage.SEMI_AUTONOMOUS,
        "replenishment_plan": EvolutionStage.SUGGESTION,
        "logistics_choice": EvolutionStage.SUGGESTION,
    }

    async def decide(self, decision_point: str, context: dict[str, Any]) -> dict[str, Any]:
        if decision_point == "stock_alert":
            return self._check_stock_alert(context)
        elif decision_point == "replenishment_plan":
            return self._calculate_replenishment(context)
        elif decision_point == "logistics_choice":
            return self._recommend_logistics(context)
        return {"action": "unknown", "confidence": 0.0}

    def _check_stock_alert(self, context: dict) -> dict:
        sku_code = context.get("sku_code", "")
        product_name = context.get("product_name", "")
        quantity = int(context.get("quantity", 0))
        safety_stock = int(context.get("safety_stock", 0))
        daily_sales = float(context.get("daily_sales", 0))
        is_fast_moving = context.get("is_fast_moving", False)
        in_transit = int(context.get("in_transit", 0))

        sellable_days = int(quantity / daily_sales) if daily_sales > 0 else 999

        red_threshold = 3 if is_fast_moving else 7
        yellow_threshold = 7 if is_fast_moving else 14

        alerts = []
        actions = []
        alert_level = "green"
        confidence = 0.90

        if sellable_days <= red_threshold:
            alert_level = "red"
            confidence = 0.95
            alerts.append({
                "level": "red",
                "sku_code": sku_code,
                "product_name": product_name,
                "sellable_days": sellable_days,
                "current_stock": quantity,
                "in_transit": in_transit,
                "daily_sales": daily_sales,
                "message": f"[紧急] {product_name}({sku_code}) 仅可售 {sellable_days} 天",
            })
            actions = ["pause_ads", "notify_purchase", "notify_operations", "consider_raising_price"]
        elif sellable_days <= yellow_threshold:
            alert_level = "yellow"
            confidence = 0.88
            replenish_qty = self._calc_replenish_qty(daily_sales, sellable_days, 30, quantity, in_transit)
            alerts.append({
                "level": "yellow",
                "sku_code": sku_code,
                "product_name": product_name,
                "sellable_days": sellable_days,
                "current_stock": quantity,
                "in_transit": in_transit,
                "daily_sales": daily_sales,
                "suggested_replenish": replenish_qty,
                "message": f"[预警] {product_name}({sku_code}) 可售 {sellable_days} 天，建议补货 {replenish_qty}",
            })
            actions = ["generate_replenishment", "notify_purchase", "consider_reduce_ads"]
        else:
            alert_level = "green"
            confidence = 0.85
            alerts.append({
                "level": "green",
                "sku_code": sku_code,
                "sellable_days": sellable_days,
                "message": f"{product_name}({sku_code}) 库存正常，可售 {sellable_days} 天",
            })
            actions = ["monitor"]

        return {
            "alert_level": alert_level,
            "sellable_days": sellable_days,
            "alerts": alerts,
            "actions": actions,
            "confidence": confidence,
        }

    def _calc_replenish_qty(
        self,
        daily_sales: float,
        sellable_days: int,
        target_days: int,
        current_stock: int,
        in_transit: int,
    ) -> int:
        safety_days = int(target_days * 1.5)
        needed = int(daily_sales * safety_days)
        available = current_stock + in_transit
        return max(0, needed - available)

    def _calculate_replenishment(self, context: dict) -> dict:
        sku_code = context.get("sku_code", "")
        daily_sales = float(context.get("daily_sales", 0))
        current_stock = int(context.get("quantity", 0))
        in_transit = int(context.get("in_transit", 0))
        lead_time_days = int(context.get("lead_time_days", 20))
        min_moq = int(context.get("min_moq", 100))
        season_factor = float(context.get("season_factor", 1.0))
        trend_factor = float(context.get("trend_factor", 1.0))

        adjusted_daily_sales = daily_sales * season_factor * trend_factor
        safety_stock_days = lead_time_days * 1.5
        safety_stock_qty = int(adjusted_daily_sales * safety_stock_days)

        replenish_qty = safety_stock_qty - current_stock - in_transit
        buffer = int(adjusted_daily_sales * 7)
        replenish_qty = max(replenish_qty + buffer, min_moq)

        urgency = "normal"
        if replenish_qty > 0 and (current_stock + in_transit) < int(adjusted_daily_sales * lead_time_days):
            urgency = "urgent"

        confidence = 0.85
        if trend_factor > 1.2 or trend_factor < 0.8:
            confidence = 0.75

        return {
            "sku_code": sku_code,
            "current_stock": current_stock,
            "in_transit": in_transit,
            "adjusted_daily_sales": round(adjusted_daily_sales, 1),
            "safety_stock_qty": safety_stock_qty,
            "suggested_replenish_qty": replenish_qty,
            "min_moq": min_moq,
            "urgency": urgency,
            "lead_time_days": lead_time_days,
            "confidence": confidence,
        }

    def _recommend_logistics(self, context: dict) -> dict:
        urgency = context.get("urgency", "normal")
        cargo_value = float(context.get("cargo_value", 0))
        weight_kg = float(context.get("weight_kg", 0))
        destination = context.get("destination", "US")
        is_peak_season = context.get("is_peak_season", False)
        has_fba_capacity = context.get("has_fba_capacity", True)

        options = []
        confidence = 0.85

        if urgency == "urgent":
            if cargo_value > 50:
                options.append({
                    "method": "air_express",
                    "name": "空运/国际快递 (DHL/UPS)",
                    "estimated_days": "3-7",
                    "cost_estimate": "高",
                    "suitability": "recommended",
                })
            options.append({
                "method": "air_freight",
                "name": "空运 + 海运快船并行",
                "estimated_days": "7-15",
                "cost_estimate": "中高",
                "suitability": "alternative",
            })
            confidence = 0.90
        elif urgency == "normal":
            sea_freight_days = "15-20" if destination == "US" else "25-35"
            options.append({
                "method": "sea_freight",
                "name": "海运",
                "estimated_days": sea_freight_days,
                "cost_estimate": "低",
                "suitability": "recommended",
            })
            if destination == "US":
                options.append({
                    "method": "express_sea",
                    "name": "快船 (美森/以星)",
                    "estimated_days": "10-15",
                    "cost_estimate": "中",
                    "suitability": "alternative",
                })
            if destination == "EU":
                options.append({
                    "method": "rail",
                    "name": "中欧班列",
                    "estimated_days": "15-20",
                    "cost_estimate": "中低",
                    "suitability": "alternative",
                })
            if is_peak_season:
                options.append({
                    "method": "advance_buffer",
                    "name": "建议提前2周预留旺季缓冲",
                    "estimated_days": "提前2周",
                    "cost_estimate": "—",
                    "suitability": "warning",
                })
                confidence = 0.80

        return {
            "destination": destination,
            "urgency": urgency,
            "cargo_value": cargo_value,
            "has_fba_capacity": has_fba_capacity,
            "options": options,
            "confidence": confidence,
        }
