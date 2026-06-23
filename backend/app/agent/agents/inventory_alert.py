"""A5 库存预警 Agent (Phase 1 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.2
- 输入覆盖 SKU、可售库存、锁定库存、在途库存、近 7/14/30 天销量、采购提前期、MOQ、安全库存天数
- 输出库存状态、可售天数、建议补货量、建议物流方式、风险原因、建议操作
- 数据不足时返回 insufficient_data，不允许编造数据
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService
from app.agent.data_service import AgentDataService


# ── 必填字段列表（缺少其中任一即返回 insufficient_data） ──
REQUIRED_STOCK_FIELDS = [
    "sku_code",
    "sellable_stock",
    "sales_7d",
    "lead_time_days",
    "safety_stock_days",
]
REQUIRED_REPLENISH_FIELDS = [
    "sku_code",
    "sellable_stock",
    "sales_30d",
    "lead_time_days",
    "moq",
]


def _safe_int(v: Any, default: int = 0) -> int:
    try:
        return int(v)
    except (TypeError, ValueError):
        return default


def _safe_float(v: Any, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def _missing_fields(context: dict, required: list[str]) -> list[str]:
    return [f for f in required if f not in context or context[f] is None]


@register_agent
class A5InventoryAlertAgent(BaseAgent):
    agent_id = "A5"
    name = "库存预警 Agent"
    description = "智能库存监控、三级预警、补货建议与物流方式推荐 | 支持可售/锁定/在途库存、多周期销量、采购提前期与MOQ"
    decision_points = ["stock_alert", "replenishment_plan", "logistics_choice"]
    version = "2.0.0"

    DEFAULT_STAGES = {
        "stock_alert": EvolutionStage.SEMI_AUTONOMOUS,
        "replenishment_plan": EvolutionStage.SUGGESTION,
        "logistics_choice": EvolutionStage.SUGGESTION,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        # 自动从 DB 补齐数据
        if db is not None and decision_point in ("stock_alert", "replenishment_plan"):
            context = await AgentDataService.fill_sku_context(db, context)
        if decision_point == "stock_alert":
            return await self._check_stock_alert(context, db=db)
        elif decision_point == "replenishment_plan":
            return self._calculate_replenishment(context)
        elif decision_point == "logistics_choice":
            return self._recommend_logistics(context)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 库存预警（核心入口）
    # ──────────────────────────────
    async def _check_stock_alert(self, context: dict, db: Any = None) -> dict:
        # ---- 向后兼容旧字段名 ----
        context = self._backfill_legacy_fields(context)

        missing = _missing_fields(context, REQUIRED_STOCK_FIELDS)
        if missing:
            return self._insufficient("stock_alert", missing)

        sku_code = str(context["sku_code"])
        sellable = _safe_int(context["sellable_stock"])
        locked = _safe_int(context.get("locked_stock", 0))
        in_transit = _safe_int(context.get("in_transit_stock", 0))
        valid_stock = sellable + in_transit  # 锁定库存不计入可售

        # 从多周期销量推算日均销量
        sales_7d = _safe_float(context.get("sales_7d", 0))
        sales_14d = _safe_float(context.get("sales_14d", 0))
        sales_30d = _safe_float(context.get("sales_30d", 0))

        daily_sales, sales_source = self._estimate_daily_sales(
            sales_7d, sales_14d, sales_30d
        )

        lead_time = _safe_int(context["lead_time_days"], 30)
        safety_days = _safe_int(context["safety_stock_days"], 14)
        moq = _safe_int(context.get("moq", 0))

        # 可售天数计算
        sellable_days = (
            round(valid_stock / daily_sales, 1) if daily_sales > 0 else 999.0
        )

        # ---- 三级预警判定 ----
        red_threshold = lead_time * 0.5  # 小于半个提前期 → 红色
        yellow_threshold = lead_time + safety_days  # 提前期 + 安全天数 → 黄色

        if sellable_days <= red_threshold:
            stock_status = "red"
            confidence = 0.95
            risk_reason = (
                f"可售天数({sellable_days}天)不足提前期({lead_time}天)的一半，"
                f"存在断货风险"
            )
            suggested_actions = [
                "紧急补货",
                "暂停广告投放",
                "通知采购部门",
                "考虑提价控量",
            ]
        elif sellable_days <= yellow_threshold:
            stock_status = "yellow"
            confidence = 0.88
            risk_reason = (
                f"可售天数({sellable_days}天)小于提前期+安全库存天数"
                f"({lead_time}+{safety_days}={yellow_threshold}天)，建议尽快补货"
            )
            suggested_actions = [
                "安排补货",
                "关注在途到货时间",
                "适当降低广告预算",
            ]
        else:
            stock_status = "green"
            confidence = 0.85
            risk_reason = "库存充足，暂无风险"
            suggested_actions = ["常规监控"]

        # ---- 建议补货量 ----
        suggested_replenish_qty = self._calc_replenish_qty(
            daily_sales=daily_sales,
            sellable_days=sellable_days,
            target_days=lead_time + safety_days,
            current_stock=sellable,
            in_transit=in_transit,
            moq=moq,
        )

        # ---- 建议物流方式 ----
        suggested_logistics = self._pick_logistics(
            stock_status, sellable_days, lead_time
        )

        # ---- LLM 自然语言解释 ----
        llm_summary = {
            "status": stock_status,
            "sku": sku_code,
            "sellable": sellable,
            "transit": in_transit,
            "daily_sales": round(daily_sales, 1),
            "sellable_days": sellable_days,
            "lead_time": lead_time,
            "safety_days": safety_days,
            "risk_reason": risk_reason,
        }
        try:
            ai_explanation = await AgentLlmService.explain("A5", llm_summary, db=db)
        except Exception:
            ai_explanation = ""

        return {
            "stock_status": stock_status,
            "sellable_days": sellable_days,
            "sellable_stock": sellable,
            "locked_stock": locked,
            "in_transit_stock": in_transit,
            "daily_sales_used": round(daily_sales, 1),
            "daily_sales_source": sales_source,
            "lead_time_days": lead_time,
            "safety_stock_days": safety_days,
            "moq": moq,
            "suggested_replenish_qty": suggested_replenish_qty,
            "suggested_logistics": suggested_logistics,
            "risk_reason": risk_reason,
            "suggested_actions": suggested_actions,
            "ai_explanation": ai_explanation,
            "confidence": confidence,
        }

    # ──────────────────────────────
    #  2. 补货计算
    # ──────────────────────────────
    def _calculate_replenishment(self, context: dict) -> dict:
        context = self._backfill_legacy_fields(context)

        missing = _missing_fields(context, REQUIRED_REPLENISH_FIELDS)
        if missing:
            return self._insufficient("replenishment_plan", missing)

        sku_code = str(context["sku_code"])
        sellable = _safe_int(context["sellable_stock"])
        in_transit = _safe_int(context.get("in_transit_stock", 0))
        lead_time = _safe_int(context["lead_time_days"], 30)
        moq = _safe_int(context["moq"], 100)
        safety_days_input = _safe_int(context.get("safety_stock_days", 14))

        sales_7d = _safe_float(context.get("sales_7d", 0))
        sales_14d = _safe_float(context.get("sales_14d", 0))
        sales_30d = _safe_float(context.get("sales_30d", 0))
        daily_sales, sales_source = self._estimate_daily_sales(
            sales_7d, sales_14d, sales_30d
        )

        if daily_sales <= 0:
            return self._insufficient("replenishment_plan", ["sales_7d/sales_30d"])

        target_days = lead_time + safety_days_input
        target_stock = int(daily_sales * target_days)
        available = sellable + in_transit
        replenish_qty = max(target_stock - available, moq)

        risk_reason = ""
        urgency = "normal"
        if replenish_qty > 0 and available < int(daily_sales * lead_time):
            urgency = "urgent"
            risk_reason = f"当前可用库存({available})小于提前期({lead_time}天)预估销量({int(daily_sales * lead_time)})"

        return {
            "sku_code": sku_code,
            "sellable_stock": sellable,
            "in_transit_stock": in_transit,
            "available_stock": sellable + in_transit,
            "daily_sales_used": round(daily_sales, 1),
            "daily_sales_source": sales_source,
            "lead_time_days": lead_time,
            "safety_stock_days": safety_days_input,
            "target_stock": target_stock,
            "suggested_replenish_qty": replenish_qty,
            "moq": moq,
            "urgency": urgency,
            "risk_reason": risk_reason,
            "confidence": 0.90 if urgency == "normal" else 0.85,
        }

    # ──────────────────────────────
    #  3. 物流推荐
    # ──────────────────────────────
    def _recommend_logistics(self, context: dict) -> dict:
        stock_status = context.get("stock_status", "green")
        sellable_days = _safe_float(context.get("sellable_days", 999))
        lead_time = _safe_int(context.get("lead_time_days", 20))
        cargo_value = _safe_float(context.get("cargo_value", 0))
        weight_kg = _safe_float(context.get("weight_kg", 0))
        destination = context.get("destination", "US")
        is_peak_season = context.get("is_peak_season", False)

        options = []
        confidence = 0.85
        suggested_logistics = "海运"

        if stock_status == "red" or sellable_days < lead_time * 0.5:
            # 紧急：空运或快递
            suggested_logistics = "空运/国际快递"
            if cargo_value > 50:
                options.append(
                    {
                        "method": "air_express",
                        "name": "空运/国际快递 (DHL/UPS)",
                        "estimated_days": "3-7",
                        "cost_estimate": "高",
                        "suitability": "recommended",
                    }
                )
            options.append(
                {
                    "method": "air_freight",
                    "name": "空运 + 海运快船并行",
                    "estimated_days": "7-15",
                    "cost_estimate": "中高",
                    "suitability": "alternative",
                }
            )
            confidence = 0.90
        elif stock_status == "yellow":
            # 预警：快船或空运备选
            suggested_logistics = "快船"
            options.append(
                {
                    "method": "express_sea",
                    "name": "快船 (美森/以星)",
                    "estimated_days": "10-15",
                    "cost_estimate": "中",
                    "suitability": "recommended",
                }
            )
            options.append(
                {
                    "method": "air_freight",
                    "name": "空运 (备选)",
                    "estimated_days": "7-12",
                    "cost_estimate": "高",
                    "suitability": "alternative",
                }
            )
            confidence = 0.88
        else:
            # 正常：海运
            sea_days = "15-20" if destination == "US" else "25-35"
            options.append(
                {
                    "method": "sea_freight",
                    "name": "海运",
                    "estimated_days": sea_days,
                    "cost_estimate": "低",
                    "suitability": "recommended",
                }
            )
            if destination == "EU":
                options.append(
                    {
                        "method": "rail",
                        "name": "中欧班列",
                        "estimated_days": "15-20",
                        "cost_estimate": "中低",
                        "suitability": "alternative",
                    }
                )
            confidence = 0.85

        if is_peak_season and options:
            options.append(
                {
                    "method": "advance_buffer",
                    "name": "旺季建议提前2周备货",
                    "estimated_days": "提前2周",
                    "cost_estimate": "—",
                    "suitability": "warning",
                }
            )
            confidence = max(0.80, confidence - 0.05)

        return {
            "suggested_logistics": suggested_logistics,
            "destination": destination,
            "stock_status": stock_status,
            "sellable_days": sellable_days,
            "cargo_value": cargo_value,
            "weight_kg": weight_kg,
            "options": options,
            "confidence": confidence,
        }

    # ──────────────────────────────
    #  内部工具方法
    # ──────────────────────────────
    def _estimate_daily_sales(
        self, sales_7d: float, sales_14d: float, sales_30d: float
    ) -> tuple[float, str]:
        """从多周期销量估算日均销量，优先用短周期"""
        if sales_7d > 0:
            return sales_7d / 7, "7d"
        if sales_14d > 0:
            return sales_14d / 14, "14d"
        if sales_30d > 0:
            return sales_30d / 30, "30d"
        return 0.0, "none"

    def _calc_replenish_qty(
        self,
        daily_sales: float,
        sellable_days: float,
        target_days: int,
        current_stock: int,
        in_transit: int,
        moq: int = 0,
    ) -> int:
        target_stock = int(daily_sales * target_days)
        available = current_stock + in_transit
        qty = max(target_stock - available, 0)
        if moq > 0 and qty < moq and qty > 0:
            qty = moq
        return qty

    def _pick_logistics(
        self, stock_status: str, sellable_days: float, lead_time: int
    ) -> str:
        if stock_status == "red" or sellable_days < lead_time * 0.5:
            return "空运/国际快递"
        elif stock_status == "yellow":
            return "快船"
        return "海运"

    def _backfill_legacy_fields(self, context: dict) -> dict:
        """向后兼容旧字段名（quantity→sellable_stock, in_transit→in_transit_stock）"""
        ctx = dict(context)
        if "sellable_stock" not in ctx and "quantity" in ctx:
            ctx["sellable_stock"] = ctx["quantity"]
        if "in_transit_stock" not in ctx and "in_transit" in ctx:
            ctx["in_transit_stock"] = ctx["in_transit"]
        if "moq" not in ctx and "min_moq" in ctx:
            ctx["moq"] = ctx["min_moq"]
        if "sellable_stock" not in ctx and "current_stock" in ctx:
            ctx["sellable_stock"] = ctx["current_stock"]
        # 兼容旧版 daily_sales → sales_7d
        if "sales_7d" not in ctx and "daily_sales" in ctx:
            ctx["sales_7d"] = float(ctx["daily_sales"]) * 7
        return ctx

    def _insufficient(self, point: str, missing: list[str]) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": point,
            "missing_fields": missing,
            "message": f"缺少必要字段: {', '.join(missing)}，请补充完整数据",
            "confidence": 0.0,
        }
