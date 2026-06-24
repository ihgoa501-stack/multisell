"""G2 仓储海关 Agent (Phase 2 — LLM 增强版)

设计依据: docs/aiagent/final-integrated-solution.md §5.2

Phase 2 改进：LLM 参与通关分析，HS 编码表降级为安全网。
- 优先调用 LLM 根据产品描述推荐 HS 编码和清关建议
- LLM 失败时自动降级为编码表兜底

Phase 1 原始设计：
- 仓储物流 + 海关三单匹配建议
- 输入货品信息和目的地，输出清关建议、仓储策略
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService

REQUIRED = ["product_name", "destination_country", "cargo_type"]
HS_CODE_DB = {
    ("electronics", "US"): "8471.30.0100",
    ("electronics", "EU"): "8471.30.00",
    ("electronics", "JP"): "8471.30.000",
    ("clothing", "US"): "6204.62.3030",
    ("clothing", "EU"): "6204.62.00",
    ("food", "US"): "2008.19.9090",
    ("food", "EU"): "2008.19.00",
    ("cosmetics", "US"): "3304.99.0000",
    ("cosmetics", "EU"): "3304.99.00",
    ("baby", "US"): "3924.10.4000",
    ("toys", "US"): "9503.00.0073",
    ("furniture", "US"): "9403.60.8081",
}


def _sf(v: Any, d: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return d


def _missing(c: dict, r: list) -> list:
    return [f for f in r if f not in c or c[f] is None]


@register_agent
class G2WarehouseCustomsAgent(BaseAgent):
    agent_id = "G2"
    name = "仓储海关 Agent"
    description = "HS 编码建议、清关文件清单、仓储策略推荐"
    decision_points = ["customs_clearance", "warehouse_advice"]
    version = "1.0.0"
    DEFAULT_STAGES = {
        "customs_clearance": EvolutionStage.SUGGESTION,
        "warehouse_advice": EvolutionStage.SUGGESTION,
    }

    async def decide(self, point: str, ctx: dict, db: Any = None) -> dict:
        if point == "customs_clearance":
            return await self._clearance_with_llm(ctx, db=db)
        if point == "warehouse_advice":
            return self._warehouse(ctx)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 通关建议（LLM 增强版）
    # ──────────────────────────────
    async def _clearance_with_llm(self, ctx: dict, db: Any = None) -> dict:
        """通关建议主入口"""
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("customs_clearance", miss)

        # ① 公式兜底
        formula_result = self._formula_clearance(ctx)

        # ② LLM 分析
        llm_decision = None
        stage = self.get_stage("customs_clearance")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "product_name": ctx.get("product_name", ""),
                "destination_country": ctx.get("destination_country", ""),
                "cargo_type": ctx.get("cargo_type", ""),
                "declared_value": ctx.get("declared_value", 0),
                "weight_kg": ctx.get("weight_kg", 0),
                "formula_hs_code": formula_result.get("hs_code", ""),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "G2", "customs_clearance", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ③ 仲裁
        if llm_decision:
            result = {
                **formula_result,
                "hs_code": llm_decision.get(
                    "hs_code_suggestion", formula_result["hs_code"]
                ),
                "risk_reason": llm_decision.get("risk_reason", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "risk_reason": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ④ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "G2",
                {
                    "product": result.get("product", ""),
                    "destination": result.get("destination", ""),
                    "hs_code": result.get("hs_code", ""),
                    "documents": ", ".join(result.get("required_documents", [])),
                    "strategy": result.get("strategy", ""),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式兜底
    # ──────────────────────────────
    def _formula_clearance(self, ctx: dict) -> dict:
        """纯公式通关检查，作为 LLM 失败时的兜底"""
        name = str(ctx.get("product_name", ""))
        country = str(ctx.get("destination_country", "")).upper().strip()
        cargo = str(ctx.get("cargo_type", "")).lower().strip()
        value = _sf(ctx.get("declared_value", 0))

        hs_code = HS_CODE_DB.get((cargo, country), "需要人工归类")
        duty_free = value < 800 and country == "US"
        estimated_duty = 0 if duty_free else round(value * 0.05, 2) if value > 0 else 0

        docs = ["Commercial Invoice", "Packing List"]
        if cargo in ("electronics", "baby", "toys"):
            docs.append("Safety Certification (FCC/CE/CPC)")
        if cargo == "food":
            docs.append("FDA Prior Notice" if country == "US" else "Health Certificate")
        if value > 2500:
            docs.append("Customs Bond / Formal Entry")

        return {
            "product": name,
            "destination": country,
            "hs_code": hs_code,
            "required_documents": docs,
            "estimated_duty": estimated_duty,
            "duty_free": duty_free,
            "value_threshold_note": "> $2500 需正式报关"
            if value > 2500
            else "≤ $2500 可简易报关",
            "confidence": 0.85 if hs_code != "需要人工归类" else 0.60,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        risk = result.get("risk_level")
        if risk is not None and risk not in ("low", "medium", "high"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    def _warehouse(self, ctx: dict) -> dict:
        country = str(ctx.get("destination_country", "")).upper().strip()
        sales_volume = _sf(ctx.get("monthly_sales_volume", 0))
        _sf(ctx.get("weight_kg", 0))

        if country == "US":
            if sales_volume > 500:
                strategy = "FBA + 海外仓双轨"
                note = "建议同时使用 FBA（Prime 流量）和第三方海外仓（降低成本风险）"
            elif sales_volume > 100:
                strategy = "FBA 优先"
                note = "建议以 FBA 为主，搭配少量自发货测试市场"
            else:
                strategy = "国内直发"
                note = "销量较小，建议国内直发或使用 Amazon 自配送"
        elif country in ("DE", "FR", "IT", "ES", "UK"):
            strategy = "欧洲海外仓"
            note = "建议使用欧洲海外仓（FBA 欧洲或第三方），注意 VAT 注册要求"
        else:
            strategy = "国内直发"
            note = "非主流市场建议国内直发，根据订单增长再考虑海外仓"

        return {
            "strategy": strategy,
            "note": note,
            "destination": country,
            "estimated_monthly_volume": int(sales_volume),
            "confidence": 0.85,
        }

    def _insufficient(self, p: str, m: list) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
