"""A5 库存预警 Agent (Phase 2 — LLM 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.2

Phase 2 改进：LLM 参与核心决策，公式降级为安全网。
- 优先调用 LLM 做情景分析（考虑季节、趋势、供应商因素）
- LLM 失败/不合理时自动降级为公式兜底
- 永远不因 LLM 异常中断执行

Phase 1 原始设计：
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

# ── LLM 输出合理性校验常量 ──
VALID_RISK_LEVELS = ("low", "medium", "high")
VALID_LOGISTICS = ("sea_freight", "express_sea", "air_freight", "air_express")


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
    #  1. 库存预警（核心入口，LLM 增强）
    # ──────────────────────────────
    async def _check_stock_alert(self, context: dict, db: Any = None) -> dict:
        """库存预警主入口

        执行流程：
        ① 公式计算（安全网，永远可用）
        ② LLM 分析（仅 SEMI_AUTONOMOUS 及以上阶段尝试）
        ③ 结果仲裁：LLM 成功且合理→用 LLM，否则用公式
        """
        # ---- 向后兼容旧字段名 ----
        context = self._backfill_legacy_fields(context)

        missing = _missing_fields(context, REQUIRED_STOCK_FIELDS)
        if missing:
            return self._insufficient("stock_alert", missing)

        # ---- ① 公式计算（安全网） ----
        formula_result = self._formula_check(context)

        # ---- ② LLM 分析（仅 SEMI_AUTONOMOUS+ 阶段尝试） ----
        stage = self.get_stage("stock_alert")
        llm_decision = None
        if stage in (
            EvolutionStage.SEMI_AUTONOMOUS,
            EvolutionStage.FULL_AUTONOMOUS,
        ):
            # 构造 LLM 上下文（展开关键指标）
            llm_context = {
                "sku_code": context.get("sku_code", ""),
                "sellable_stock": context.get("sellable_stock", 0),
                "locked_stock": context.get("locked_stock", 0),
                "in_transit_stock": context.get("in_transit_stock", 0),
                "sales_7d": context.get("sales_7d", 0),
                "sales_14d": context.get("sales_14d", "N/A"),
                "sales_30d": context.get("sales_30d", "N/A"),
                "lead_time_days": context.get("lead_time_days", 0),
                "safety_stock_days": context.get("safety_stock_days", 0),
                "moq": context.get("moq", 0),
                "sellable_days": formula_result.get("sellable_days", "N/A"),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A5", "stock_alert", llm_context, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass  # LLM 失败不中断，用公式兜底

        # ---- ③ 结果仲裁 ----
        if llm_decision:
            # LLM 优先：数值用公式保证精确，语义用 LLM 保证丰富
            result = {
                **formula_result,
                "risk_reason": llm_decision.get(
                    "risk_reason", formula_result["risk_reason"]
                ),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            # 公式兜底
            result = {**formula_result, "additional_notes": "", "llm_source": False}

        # ---- ④ LLM 自然语言解释（后处理，不影响决策） ----
        if formula_result.get("confidence", 0) > 0:
            llm_summary = {
                "status": result.get("stock_status", ""),
                "sku": result.get("sku_code", ""),
                "sellable": result.get("sellable_stock", 0),
                "transit": result.get("in_transit_stock", 0),
                "daily_sales": result.get("daily_sales_used", 0),
                "sellable_days": result.get("sellable_days", 0),
                "lead_time": result.get("lead_time_days", 0),
                "safety_days": result.get("safety_stock_days", 0),
                "risk_reason": result.get("risk_reason", ""),
            }
            try:
                result["ai_explanation"] = await AgentLlmService.explain(
                    "A5", llm_summary, db=db
                )
            except Exception:
                result["ai_explanation"] = ""
        else:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式计算（安全网）
    # ──────────────────────────────
    def _formula_check(self, context: dict) -> dict:
        """纯公式库存检查，结果与旧版 _check_stock_alert 一致

        作为 LLM 失败时的兜底安全网，永不抛异常。
        """
        sku_code = str(context["sku_code"])
        sellable = _safe_int(context["sellable_stock"])
        locked = _safe_int(context.get("locked_stock", 0))
        in_transit = _safe_int(context.get("in_transit_stock", 0))
        valid_stock = sellable + in_transit

        sales_7d = _safe_float(context.get("sales_7d", 0))
        sales_14d = _safe_float(context.get("sales_14d", 0))
        sales_30d = _safe_float(context.get("sales_30d", 0))

        daily_sales, sales_source = self._estimate_daily_sales(
            sales_7d, sales_14d, sales_30d
        )

        lead_time = _safe_int(context["lead_time_days"], 30)
        safety_days = _safe_int(context["safety_stock_days"], 14)
        moq = _safe_int(context.get("moq", 0))

        sellable_days = (
            round(valid_stock / daily_sales, 1) if daily_sales > 0 else 999.0
        )

        # 三级预警
        red_threshold = lead_time * 0.5
        yellow_threshold = lead_time + safety_days

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

        suggested_replenish_qty = self._calc_replenish_qty(
            daily_sales=daily_sales,
            sellable_days=sellable_days,
            target_days=lead_time + safety_days,
            current_stock=sellable,
            in_transit=in_transit,
            moq=moq,
        )

        suggested_logistics = self._pick_logistics(
            stock_status, sellable_days, lead_time
        )

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
            "confidence": confidence,
            "sku_code": sku_code,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 返回的结构化结果是否合理"""
        # risk_level 必须为合法值
        if result.get("risk_level") not in VALID_RISK_LEVELS:
            return False
        # suggested_logistics 必须为合法值
        logistics = result.get("suggested_logistics")
        if logistics is not None and logistics not in VALID_LOGISTICS:
            return False
        # suggested_replenish_qty 必须为非负整数
        qty = result.get("suggested_replenish_qty")
        if qty is not None:
            try:
                if int(qty) < 0:
                    return False
            except (TypeError, ValueError):
                return False
        # confidence 必须在 0-1 之间
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        # risk_reason 不能为空
        if not result.get("risk_reason"):
            return False
        return True

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
