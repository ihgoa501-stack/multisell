"""G1 运营驾驶舱 Agent (Phase 2 — LLM 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.5

Phase 2 改进：从数据库聚合基本运营数据，LLM 生成运营摘要。
- 有 DB 时自动查询产品/SKU/预警/规则健康等数据
- 调用 LLM 生成自然语言运营摘要
- LLM 失败时回退到基础统计数据
"""

from datetime import datetime, timezone
from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService
from sqlalchemy import select, func
from app.models import Product, Sku


@register_agent
class G1DashboardAgent(BaseAgent):
    agent_id = "G1"
    name = "运营驾驶舱 Agent"
    description = "汇总库存、利润、折扣风险，展示决策概览和规则健康状态"
    decision_points = ["dashboard_overview"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "dashboard_overview": EvolutionStage.OBSERVATION,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        if decision_point == "dashboard_overview":
            return await self._overview(context, db=db)
        return {"action": "unknown", "confidence": 0.0}

    async def _overview(self, ctx: dict, db: Any = None) -> dict:
        """运营概览主入口"""
        # ① 公式兜底（从 DB 聚合数据）
        formula_result = await self._formula_overview(ctx, db=db)

        # ② LLM 分析（SEMI_AUTONOMOUS+ 阶段）
        llm_decision = None
        stage = self.get_stage("dashboard_overview")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "product_count": formula_result.get("product_count", "N/A"),
                "sku_count": formula_result.get("sku_count", "N/A"),
                "alert_count": formula_result.get("alert_count", 0),
                "timestamp": formula_result.get("timestamp", ""),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "G1", "dashboard_overview", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        if llm_decision:
            result = {
                **formula_result,
                "summary": llm_decision.get("summary", ""),
                "llm_source": True,
            }
        else:
            result = {**formula_result, "summary": "", "llm_source": False}

        # ③ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "G1",
                {
                    "product_count": result.get("product_count", 0),
                    "sku_count": result.get("sku_count", 0),
                    "alert_count": result.get("alert_count", 0),
                    "timestamp": result.get("timestamp", ""),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    async def _formula_overview(self, ctx: dict, db: Any = None) -> dict:
        """从数据库聚合运营基础数据"""
        product_count = 0
        sku_count = 0
        alert_count = 0

        if db is not None:
            try:
                product_count = await db.scalar(select(func.count(Product.id))) or 0
                sku_count = await db.scalar(select(func.count(Sku.id))) or 0
            except Exception:
                pass

        return {
            "product_count": product_count,
            "sku_count": sku_count,
            "alert_count": alert_count,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "confidence": 0.85 if product_count > 0 else 0.60,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        if not result.get("summary"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True
