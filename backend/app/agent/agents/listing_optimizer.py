"""A2 Listing 优化 Agent

设计依据: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent2
- 关键词策略 + 竞品拆解 + 文案生成
- 输入产品信息和竞品数据，输出优化后的 Listing 全案
"""
from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent

REQUIRED = ["product_name", "marketplace"]


def _sf(v: Any, d: float = 0.0) -> float:
    try: return float(v)
    except (TypeError, ValueError): return d

def _missing(c: dict, r: list) -> list:
    return [f for f in r if f not in c or c[f] is None]


@register_agent
class A2ListingOptimizerAgent(BaseAgent):
    agent_id = "A2"
    name = "Listing 优化 Agent"
    description = "基于关键词研究和竞品分析，生成优化后的标题、五点描述、Search Terms"
    decision_points = ["listing_optimize", "keyword_research"]
    version = "1.0.0"
    DEFAULT_STAGES = {
        "listing_optimize": EvolutionStage.SUGGESTION,
        "keyword_research": EvolutionStage.SUGGESTION,
    }

    async def decide(self, point: str, ctx: dict, db: Any = None) -> dict:
        if point == "listing_optimize": return self._optimize(ctx)
        if point == "keyword_research": return self._research_keywords(ctx)
        return {"action": "unknown", "confidence": 0.0}

    def _optimize(self, ctx: dict) -> dict:
        miss = _missing(ctx, REQUIRED)
        if miss: return self._insufficient("listing_optimize", miss)
        name = str(ctx.get("product_name", ""))
        mp = str(ctx.get("marketplace", "US"))
        features = ctx.get("features", [])
        bullets_input = ctx.get("current_bullets", [])
        keywords = ctx.get("keywords", [])

        # 关键词排序：高频核心词放标题前部
        sorted_kw = sorted(keywords, key=lambda k: _sf(k.get("volume", 0)), reverse=True)
        top_kw = [k.get("word", "") for k in sorted_kw[:3]]

        title = f"{' '.join(top_kw)} - {name}"
        if len(title) > 200: title = title[:200]

        bullets = []
        for i, f in enumerate(features[:5]):
            kw = sorted_kw[i].get("word", "") if i < len(sorted_kw) else ""
            bullets.append(f"{f} — {kw}" if kw else f)

        search_terms = list(set(k.get("word", "") for k in sorted_kw[:20]))
        search_terms = [w for w in search_terms if w.lower() not in name.lower()][:20]

        suggestions = []
        if not features:
            suggestions.append("请补充产品卖点 features 以获得更精准的优化")
        if len(bullets_input) < 3:
            suggestions.append("当前五点描述不足，建议补充至5条")

        return {
            "title": title,
            "bullets": bullets or bullets_input[:5],
            "search_terms": search_terms,
            "marketplace": mp,
            "keyword_count": len(sorted_kw),
            "suggestions": suggestions,
            "confidence": 0.85,
        }

    def _research_keywords(self, ctx: dict) -> dict:
        seed = ctx.get("seed_keywords", [])
        if not seed:
            return self._insufficient("keyword_research", ["seed_keywords"])
        return {
            "seed": seed,
            "expanded": [f"{s} {t}" for s in seed for t in ["for", "with", "best"]],
            "total_found": len(seed) * 3,
            "confidence": 0.80,
        }

    def _insufficient(self, p: str, m: list) -> dict:
        return {"status": "insufficient_data", "decision_point": p, "missing_fields": m, "message": f"缺少: {', '.join(m)}", "confidence": 0.0}
