"""A2 Listing 优化 Agent (Phase 2 — LLM 增强版)

设计依据: docs/aiagent/跨境电商AI_Agent深度调研报告.md §Agent2

Phase 2 改进：LLM 参与 Listing 优化核心决策，机械公式降级为安全网。
- 优先调用 LLM 生成高质量、有说服力的标题和描述
- LLM 失败时自动降级为机械关键词组合兜底

Phase 1 原始设计：
- 关键词策略 + 竞品拆解 + 文案生成
- 输入产品信息和竞品数据，输出优化后的 Listing 全案
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService

REQUIRED = ["product_name", "marketplace"]


def _sf(v: Any, d: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return d


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
        if point == "listing_optimize":
            return await self._optimize_with_llm(ctx, db=db)
        if point == "keyword_research":
            return self._research_keywords(ctx)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. Listing 优化（LLM 增强版）
    # ──────────────────────────────
    async def _optimize_with_llm(self, ctx: dict, db: Any = None) -> dict:
        """Listing 优化主入口"""
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("listing_optimize", miss)

        # ① 公式兜底
        formula_result = self._formula_optimize(ctx)

        # ② LLM 分析
        llm_decision = None
        stage = self.get_stage("listing_optimize")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "product_name": ctx.get("product_name", ""),
                "marketplace": ctx.get("marketplace", "US"),
                "features": str(ctx.get("features", [])),
                "current_bullets": str(ctx.get("current_bullets", [])),
                "keywords": str(ctx.get("keywords", [])),
                "formula_title": formula_result.get("title", ""),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A2", "listing_optimize", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ③ 仲裁
        if llm_decision:
            result = {
                **formula_result,
                "title_strategy": llm_decision.get("title_suggestion", ""),
                "bullet_strategy": llm_decision.get("bullet_points", ""),
                "keyword_strategy": llm_decision.get("keyword_strategy", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "title_strategy": "",
                "bullet_strategy": "",
                "keyword_strategy": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ④ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A2",
                {
                    "product_name": ctx.get("product_name", ""),
                    "marketplace": ctx.get("marketplace", "US"),
                    "original_title": "",
                    "optimized_title": result.get("title", ""),
                    "keyword_count": result.get("keyword_count", 0),
                    "suggestions": str(result.get("suggestions", [])),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式兜底
    # ──────────────────────────────
    def _formula_optimize(self, ctx: dict) -> dict:
        """纯公式 Listing 优化，作为 LLM 失败时的兜底"""
        name = str(ctx.get("product_name", ""))
        mp = str(ctx.get("marketplace", "US"))
        features = ctx.get("features", [])
        bullets_input = ctx.get("current_bullets", [])
        keywords = ctx.get("keywords", [])

        sorted_kw = sorted(
            keywords, key=lambda k: _sf(k.get("volume", 0)), reverse=True
        )
        top_kw = [k.get("word", "") for k in sorted_kw[:3]]

        title = f"{' '.join(top_kw)} - {name}"
        if len(title) > 200:
            title = title[:200]

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

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        if not result.get("title_suggestion"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

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
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
