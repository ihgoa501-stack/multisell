"""A1 选品扫描 Agent

设计依据: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent1
- 多维度选品评分（需求/竞争/利润/壁垒/趋势）
- 输入市场数据，输出候选产品列表
- 数据不足时返回 insufficient_data
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent

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
            return self._scout(ctx)
        if point == "market_analysis":
            return self._analyze_market(ctx)
        return {"action": "unknown", "confidence": 0.0}

    def _scout(self, ctx: dict) -> dict:
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("product_scout", miss)
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
