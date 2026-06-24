"""A1 选品扫描 Agent (Phase 2 — LLM 增强版)

设计依据: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent1

Phase 2 改进：LLM 参与选品评分和市场分析，固定权重降级为安全网。
- 优先调用 LLM 分析产品候选数据的市场机会
- LLM 失败时自动降级为固定权重评分兜底

Phase 1 原始设计：
- 多维度选品评分（需求/竞争/利润/壁垒/趋势）
- 输入市场数据，输出候选产品列表
- 数据不足时返回 insufficient_data
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService

REQUIRED = ["category", "marketplace"]


def _sf(v: Any, d: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return d


def _missing(c: dict, r: list) -> list:
    return [f for f in r if f not in c or c[f] is None]


@register_agent
class A1ProductScoutAgent(BaseAgent):
    agent_id = "A1"
    name = "选品扫描 Agent"
    description = "多维度自动化选品评分与市场机会发现，支持需求/竞争/利润/趋势分析"
    decision_points = ["product_scout", "market_analysis"]
    version = "1.0.0"
    DEFAULT_STAGES = {
        "product_scout": EvolutionStage.SUGGESTION,
        "market_analysis": EvolutionStage.OBSERVATION,
    }

    async def decide(self, point: str, ctx: dict, db: Any = None) -> dict:
        if point == "product_scout":
            return await self._scout_with_llm(ctx, db=db)
        if point == "market_analysis":
            return self._analyze_market(ctx)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 选品扫描（LLM 增强版）
    # ──────────────────────────────
    async def _scout_with_llm(self, ctx: dict, db: Any = None) -> dict:
        """选品扫描主入口"""
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("product_scout", miss)

        # ① 公式兜底
        formula_result = self._formula_scout(ctx)

        # ② LLM 分析
        llm_decision = None
        stage = self.get_stage("product_scout")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "category": ctx.get("category", ""),
                "marketplace": ctx.get("marketplace", "US"),
                "candidates": str(ctx.get("candidates", [])),
                "formula_top": formula_result.get("candidates", [{}])[0].get("name", "")
                if formula_result.get("candidates")
                else "",
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A1", "product_scout", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ③ 仲裁
        if llm_decision:
            result = {
                **formula_result,
                "top_product": llm_decision.get("top_product", ""),
                "market_insight": llm_decision.get("market_insight", ""),
                "risk_flags": llm_decision.get("risk_flags", ""),
                "scoring_approach": llm_decision.get("scoring_approach", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "top_product": "",
                "market_insight": "",
                "risk_flags": "",
                "scoring_approach": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ④ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A1",
                {
                    "category": ctx.get("category", ""),
                    "marketplace": ctx.get("marketplace", "US"),
                    "candidate_count": result.get("total_scanned", 0),
                    "top_score": result.get("candidates", [{}])[0].get("score", "?")
                    if result.get("candidates")
                    else "?",
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式兜底
    # ──────────────────────────────
    def _formula_scout(self, ctx: dict) -> dict:
        """纯公式选品评分，作为 LLM 失败时的兜底"""
        category = str(ctx.get("category", ""))
        marketplace = str(ctx.get("marketplace", "US"))
        candidates_input = ctx.get("candidates", [])
        if not candidates_input:
            return {
                "status": "insufficient_data",
                "decision_point": "product_scout",
                "missing_fields": ["candidates"],
                "message": "请提供候选产品列表 candidates",
                "confidence": 0.0,
            }
        scored = []
        for item in candidates_input:
            demand = _sf(item.get("search_volume", 0)) * 0.01
            growth = _sf(item.get("trend_growth", 0)) / 100
            competition = max(0, 1 - _sf(item.get("review_count", 0)) / 1000)
            price = _sf(item.get("price", 0))
            cost = _sf(item.get("cost", 0))
            margin = (price - cost) / price if price > 0 else 0
            score = round(demand * 30 + growth * 25 + competition * 20 + margin * 25, 1)
            scored.append(
                {
                    "name": item.get("name", ""),
                    "score": score,
                    "demand_score": round(demand * 100, 1),
                    "competition_score": round(competition * 100, 1),
                    "margin_score": round(margin * 100, 1),
                    "trend_score": round(growth * 100, 1),
                    "estimated_margin": round(margin * 100, 1),
                    "risk_flags": ["高竞争"] if competition < 0.3 else [],
                }
            )
        scored.sort(key=lambda x: x["score"], reverse=True)
        return {
            "category": category,
            "marketplace": marketplace,
            "candidates": scored[:20],
            "total_scanned": len(scored),
            "confidence": 0.85,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        if not result.get("top_product"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    def _analyze_market(self, ctx: dict) -> dict:
        return {
            "category": ctx.get("category", ""),
            "marketplace": ctx.get("marketplace", "US"),
            "market_size_estimate": "medium",
            "trend_direction": ctx.get("trend", "stable"),
            "confidence": 0.80,
        }

    def _insufficient(self, p: str, m: list) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
