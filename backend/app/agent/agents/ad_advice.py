"""A3 广告建议 Agent (Phase 4 — LLM 增强版)

设计依据: docs/AI_AGENT_FEASIBLE_DEVELOPMENT_SPEC.md §7.1.6

Phase 2 改进：LLM 参与 ACoS 决策分析，公式降级为安全网。

Phase 1 原始设计：
- 分析广告投放效果，输出建议
- 不接真实 Amazon Ads API
- 不自动调价、不自动暂停广告
- 数据通过请求体传入（手动导入或 mock）
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService

REQUIRED_FIELDS = ["campaign_id", "spend", "sales"]


def _safe_float(v: Any, default: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return default


def _missing_fields(context: dict, required: list[str]) -> list[str]:
    return [f for f in required if f not in context or context[f] is None]


@register_agent
class A3AdAdviceAgent(BaseAgent):
    agent_id = "A3"
    name = "广告建议 Agent"
    description = "广告投放效果分析，ACoS 异常检测，出价/暂停/否定关键词建议（仅建议，不自动执行）"
    decision_points = ["acos_analysis", "ad_optimization"]
    version = "1.0.0"

    DEFAULT_STAGES = {
        "acos_analysis": EvolutionStage.OBSERVATION,
        "ad_optimization": EvolutionStage.SUGGESTION,
    }

    async def decide(
        self, decision_point: str, context: dict[str, Any], db: Any = None
    ) -> dict[str, Any]:
        if decision_point == "acos_analysis":
            return await self._analyze_acos(context, db=db)
        elif decision_point == "ad_optimization":
            return self._suggest_ad_optimization(context)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. ACoS 分析（LLM 增强版）
    # ──────────────────────────────
    async def _analyze_acos(self, context: dict, db: Any = None) -> dict:
        """ACoS 分析主入口

        执行流程：
        ① 公式计算（安全网）
        ② LLM 分析（SEMI_AUTONOMOUS+ 阶段尝试）
        ③ 结果仲裁
        """
        missing = _missing_fields(context, REQUIRED_FIELDS)
        if missing:
            return self._insufficient("acos_analysis", missing)

        # ---- ① 公式计算（安全网） ----
        formula_result = self._formula_acos_check(context)

        # ---- ② LLM 分析 ----
        llm_decision = None
        stage = self.get_stage("acos_analysis")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            metrics = formula_result.get("metrics", {})
            llm_context = {
                "campaign_id": formula_result.get("campaign_id", ""),
                "spend": metrics.get("spend", 0),
                "sales": metrics.get("sales", 0),
                "acos": metrics.get("acos", 0),
                "target_acos": metrics.get("target_acos", 0),
                "ctr": metrics.get("ctr", 0),
                "cvr": metrics.get("conversion_rate", 0),
                "cpc": metrics.get("cpc", 0),
                "budget_usage": metrics.get("budget_usage", 0),
                "status": formula_result.get("status", "normal"),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A3", "acos_analysis", llm_context, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ---- ③ 结果仲裁 ----
        if llm_decision:
            result = {
                **formula_result,
                "root_cause": llm_decision.get("root_cause", ""),
                "bid_suggestion_llm": llm_decision.get("bid_suggestion", ""),
                "keyword_suggestion": llm_decision.get("keyword_suggestion", ""),
                "additional_notes": llm_decision.get("additional_notes", ""),
                "llm_source": True,
            }
        else:
            result = {
                **formula_result,
                "root_cause": "",
                "bid_suggestion_llm": "",
                "keyword_suggestion": "",
                "additional_notes": "",
                "llm_source": False,
            }

        # ---- ④ LLM 自然语言解释 ----
        metrics = result.get("metrics", {})
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A3",
                {
                    "campaign_id": result.get("campaign_id", ""),
                    "spend": metrics.get("spend", 0),
                    "sales": metrics.get("sales", 0),
                    "acos": metrics.get("acos", 0),
                    "target_acos": metrics.get("target_acos", 0),
                    "ctr": metrics.get("ctr", 0),
                    "cvr": metrics.get("conversion_rate", 0),
                    "status": result.get("status", ""),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式计算（安全网）
    # ──────────────────────────────
    def _formula_acos_check(self, context: dict) -> dict:
        """纯公式 ACoS 检查，作为 LLM 失败时的兜底"""
        campaign_id = str(context.get("campaign_id", ""))
        sku_code = str(context.get("sku_code", context.get("asin", "")))
        spend = _safe_float(context["spend"])
        sales = _safe_float(context["sales"])
        clicks = _safe_float(context.get("clicks", 0))
        impressions = _safe_float(context.get("impressions", 0))
        conversions = _safe_float(context.get("conversions", 0))
        budget = _safe_float(context.get("budget", 0))
        inventory_status = str(context.get("inventory_status", "normal"))
        gross_margin = _safe_float(context.get("gross_margin", 0))
        target_acos = _safe_float(context.get("target_acos", 30.0))

        acos = round(spend / sales * 100, 2) if sales > 0 else 0.0
        ctr = round(clicks / impressions * 100, 2) if impressions > 0 else 0.0
        conversion_rate = round(conversions / clicks * 100, 2) if clicks > 0 else 0.0
        cpc = round(spend / clicks, 2) if clicks > 0 else 0.0
        budget_usage = round(spend / budget * 100, 2) if budget > 0 else 0.0

        alerts = []
        suggestions = []
        confidence = 0.85
        status = "normal"

        acos_abnormal = acos > target_acos
        if acos_abnormal:
            if acos > gross_margin and gross_margin > 0:
                status = "critical"
                alerts.append(
                    {
                        "level": "critical",
                        "message": f"ACoS ({acos}%) 超过毛利率 ({gross_margin}%)，广告亏损",
                    }
                )
                suggestions.append("建议暂停或大幅降低广告出价")
                confidence = 0.95
            else:
                status = "warning"
                alerts.append(
                    {
                        "level": "warning",
                        "message": f"ACoS ({acos}%) 超过目标阈值 ({target_acos}%)",
                    }
                )
                suggestions.append("建议降低广告出价或优化否定关键词")
                confidence = 0.88

        if budget > 0 and budget_usage > 90:
            alerts.append(
                {"level": "info", "message": f"预算已使用 {budget_usage}%，接近上限"}
            )
        elif budget > 0 and budget_usage < 10:
            alerts.append(
                {
                    "level": "info",
                    "message": f"预算使用率仅 {budget_usage}%，建议检查广告是否正常投放",
                }
            )

        if inventory_status in ("low", "out_of_stock"):
            alerts.append(
                {
                    "level": "warning",
                    "message": f"库存状态为 {inventory_status}，建议暂停广告避免浪费",
                }
            )
            suggestions.append("库存不足时暂停广告")

        if impressions > 0 and ctr < 0.5:
            alerts.append(
                {
                    "level": "info",
                    "message": f"点击率 ({ctr}%) 偏低，建议优化主图或标题",
                }
            )
        if clicks > 0 and conversion_rate < 5:
            alerts.append(
                {
                    "level": "info",
                    "message": f"转化率 ({conversion_rate}%) 偏低，建议检查 Listing 或价格",
                }
            )

        bid_suggestion = None
        if acos > target_acos and sales > 0:
            ideal_acos = target_acos * 0.8
            suggested_spend = sales * ideal_acos / 100
            if clicks > 0:
                suggested_cpc = round(suggested_spend / clicks, 2)
                current_cpc = cpc
                if suggested_cpc < current_cpc:
                    bid_suggestion = {
                        "current_cpc": current_cpc,
                        "suggested_cpc": suggested_cpc,
                        "reduction_pct": round(
                            (current_cpc - suggested_cpc) / current_cpc * 100, 1
                        ),
                        "description": f"建议 CPC 从 ¥{current_cpc} 降至 ¥{suggested_cpc}",
                    }

        return {
            "status": status,
            "campaign_id": campaign_id,
            "sku_code": sku_code,
            "metrics": {
                "acos": acos,
                "target_acos": target_acos,
                "ctr": ctr,
                "conversion_rate": conversion_rate,
                "cpc": cpc,
                "budget_usage": budget_usage,
                "spend": spend,
                "sales": sales,
                "clicks": int(clicks),
                "impressions": int(impressions),
                "conversions": int(conversions),
            },
            "acos_abnormal": acos_abnormal,
            "alerts": alerts,
            "suggestions": suggestions,
            "bid_suggestion": bid_suggestion,
            "confidence": confidence,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 返回的结构化结果是否合理"""
        status = result.get("status")
        if status is not None and status not in ("normal", "warning", "critical"):
            return False
        if not result.get("root_cause"):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    # ──────────────────────────────
    #  2. 广告优化建议
    # ──────────────────────────────
    def _suggest_ad_optimization(self, context: dict) -> dict:
        missing = _missing_fields(context, ["campaign_id"])
        if missing:
            return self._insufficient("ad_optimization", missing)

        campaign_id = str(context.get("campaign_id", ""))
        spend = _safe_float(context.get("spend", 0))
        sales = _safe_float(context.get("sales", 0))
        acos = round(spend / sales * 100, 2) if sales > 0 else 0.0
        _safe_float(context.get("clicks", 0))
        search_terms = context.get("search_terms", [])
        target_acos = _safe_float(context.get("target_acos", 30.0))

        optimization_items = []

        # 否定关键词建议
        negative_keywords = []
        if search_terms and isinstance(search_terms, list):
            for term in search_terms:
                term_spend = _safe_float(term.get("spend", 0))
                term_sales = _safe_float(term.get("sales", 0))
                term_clicks = _safe_float(term.get("clicks", 0))
                term_acos = (
                    round(term_spend / term_sales * 100, 2) if term_sales > 0 else 0.0
                )
                if term_clicks >= 10 and term_acos > target_acos * 1.5:
                    negative_keywords.append(
                        {
                            "keyword": term.get("keyword", ""),
                            "clicks": int(term_clicks),
                            "acos": term_acos,
                            "reason": f"ACoS {term_acos}% 远超目标 {target_acos}%，建议添加为否定关键词",
                        }
                    )

        if negative_keywords:
            optimization_items.append(
                {
                    "type": "negative_keyword",
                    "items": negative_keywords[:10],
                    "description": f"发现 {len(negative_keywords)} 个高 ACOS 搜索词，建议添加否定关键词",
                }
            )

        # 出价优化
        if acos > target_acos:
            reduction = min(round((acos - target_acos) / acos * 100, 1), 50)
            optimization_items.append(
                {
                    "type": "bid_reduction",
                    "description": f"建议降低出价 {reduction}%（当前 ACoS {acos}%，目标 {target_acos}%）",
                    "suggested_reduction_pct": reduction,
                }
            )

        # 预算建议
        budget = _safe_float(context.get("budget", 0))
        if budget > 0 and spend > budget * 0.9:
            optimization_items.append(
                {
                    "type": "budget_increase",
                    "description": f"预算即将耗尽（已用 {(spend / budget * 100):.0f}%），如效果良好建议增加预算",
                }
            )

        return {
            "campaign_id": campaign_id,
            "current_acos": acos,
            "target_acos": target_acos,
            "optimization_items": optimization_items,
            "confidence": 0.85,
        }

    def _insufficient(self, point: str, missing: list[str]) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": point,
            "missing_fields": missing,
            "message": f"缺少必要字段: {', '.join(missing)}，请补充完整数据",
            "confidence": 0.0,
        }
