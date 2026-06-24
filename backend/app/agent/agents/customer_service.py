"""A4 多语言客服 Agent (Phase 2 — LLM 增强版)

设计依据: docs/aiagent/final-integrated-solution.md §6.2

Phase 2 改进：LLM 参与意图分类和回复建议，关键词匹配降级为安全网。
- 优先调用 LLM 理解消息含义、识别意图、分析情绪
- LLM 失败时自动降级为关键词匹配兜底

Phase 1 原始设计：
- L1 自动回复 60%+ FAQ，L2 辅助人工处理复杂投诉
- 意图分类 + 置信度路由
- 不接真实消息接口，数据通过请求体传入
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent
from app.agent.llm_service import AgentLlmService

REQUIRED = ["message", "language"]

FAQ_RESPONSES = {
    "shipping": "感谢您的咨询。您的订单正在正常处理中，预计 {eta} 天内送达。您可以登录查看最新物流状态。",
    "return": "关于退换货，您可以在签收后 30 天内申请。请登录订单详情页提交退换货申请，我们将为您免费安排取件。",
    "size": "该产品的尺寸信息已在商品详情页列出。如果您不确定哪个尺码适合您，请参考页面上的尺码对照表。",
    "promotion": "当前促销活动已在商品页面显示，折扣会在结算时自动应用。",
    "product_use": "感谢您的咨询！关于产品的使用方式，您可以查看商品详情页的使用说明部分。如有其他问题请随时联系我们。",
    "default": "感谢您的咨询。我们已收到您的消息，客服团队将在 24 小时内回复您。如需加急处理，请回复 URGENT。",
}


def _sf(v: Any, d: float = 0.0) -> float:
    try:
        return float(v)
    except (TypeError, ValueError):
        return d


def _missing(c: dict, r: list) -> list:
    return [f for f in r if f not in c or c[f] is None]


@register_agent
class A4CustomerServiceAgent(BaseAgent):
    agent_id = "A4"
    name = "多语言客服 Agent"
    description = "FAQ 自动回复 + 意图分类路由，支持多语言，L1自动/L2辅助"
    decision_points = ["auto_reply", "intent_classify"]
    version = "1.0.0"
    DEFAULT_STAGES = {
        "auto_reply": EvolutionStage.SEMI_AUTONOMOUS,
        "intent_classify": EvolutionStage.SUGGESTION,
    }

    async def decide(self, point: str, ctx: dict, db: Any = None) -> dict:
        if point == "auto_reply":
            return await self._auto_reply_with_llm(ctx, db=db)
        if point == "intent_classify":
            return self._formula_classify(ctx)
        return {"action": "unknown", "confidence": 0.0}

    # ──────────────────────────────
    #  1. 自动回复（LLM 增强版）
    # ──────────────────────────────
    async def _auto_reply_with_llm(self, ctx: dict, db: Any = None) -> dict:
        """自动回复主入口"""
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("auto_reply", miss)

        # ① 公式兜底
        formula_result = self._formula_auto_reply(ctx)

        # ② LLM 分析
        llm_decision = None
        stage = self.get_stage("auto_reply")
        if stage in (EvolutionStage.SEMI_AUTONOMOUS, EvolutionStage.FULL_AUTONOMOUS):
            llm_ctx = {
                "message": ctx.get("message", "")[:500],
                "language": ctx.get("language", "en"),
                "order_context": str(ctx.get("order_context", {})),
            }
            try:
                llm_raw = await AgentLlmService.analyze(
                    "A4", "auto_reply", llm_ctx, db=db
                )
                if llm_raw and self._validate_llm_output(llm_raw):
                    llm_decision = llm_raw
            except Exception:
                pass

        # ③ 仲裁
        if llm_decision:
            suggests_human = llm_decision.get("requires_human", False)
            result = {
                "auto_reply": None
                if suggests_human
                else llm_decision.get("reply_suggestion", ""),
                "action": "escalate" if suggests_human else "auto_reply",
                "intent": llm_decision.get("intent", "unknown"),
                "confidence": llm_decision.get("confidence", 0.5),
                "sentiment": llm_decision.get("sentiment", "neutral"),
                "reason": "LLM 判断需人工处理" if suggests_human else "LLM 自动回复",
                "suggested_reply": llm_decision.get("reply_suggestion")
                if suggests_human
                else None,
                "llm_source": True,
            }
        else:
            result = {**formula_result, "llm_source": False}

        # ④ 解释
        try:
            result["ai_explanation"] = await AgentLlmService.explain(
                "A4",
                {
                    "message": ctx.get("message", "")[:100],
                    "intent": result.get("intent", ""),
                    "reply_suggestion": str(
                        result.get("auto_reply", "")
                        or result.get("suggested_reply", "")
                    ),
                    "confidence": result.get("confidence", 0),
                },
                db=db,
            )
        except Exception:
            result["ai_explanation"] = ""

        return result

    # ──────────────────────────────
    #  1a. 公式兜底
    # ──────────────────────────────
    def _formula_classify(self, ctx: dict) -> dict:
        """纯关键词意图分类，作为 LLM 失败时的兜底"""
        msg = str(ctx.get("message", "")).lower()
        high_risk = [
            "trademark",
            "lawsuit",
            "refund",
            "a-to-z",
            "chargeback",
            "投诉",
            "律师",
            "起诉",
            "退款",
            "索赔",
        ]
        if any(kw in msg for kw in high_risk):
            return {
                "intent": "high_risk",
                "confidence": 0.95,
                "action": "escalate_human",
            }
        faq_map = {
            "where is my order": "shipping",
            "tracking": "shipping",
            "物流": "shipping",
            "return": "return",
            "refund": "return",
            "退货": "return",
            "退款": "return",
            "size": "size",
            "尺码": "size",
            "fit": "size",
            "promotion": "promotion",
            "coupon": "promotion",
            "折扣": "promotion",
            "how to use": "product_use",
            "如何使用": "product_use",
        }
        for kw, intent in faq_map.items():
            if kw in msg:
                return {"intent": intent, "confidence": 0.90, "action": "auto_reply"}
        return {"intent": "unknown", "confidence": 0.50, "action": "escalate_human"}

    def _formula_auto_reply(self, ctx: dict) -> dict:
        """纯公式自动回复，作为 LLM 失败时的兜底"""
        lang = str(ctx.get("language", "en"))
        order_ctx = ctx.get("order_context", {})
        eta = order_ctx.get("estimated_delivery_days", "5-7")

        classification = self._formula_classify(ctx)
        intent = classification.get("intent", "default")
        confidence = classification.get("confidence", 0.0)

        if classification.get("action") == "escalate_human":
            return {
                "auto_reply": None,
                "action": "escalate",
                "intent": intent,
                "confidence": confidence,
                "sentiment": "neutral",
                "reason": "高风险或低置信度，需人工处理",
                "suggested_reply": None,
            }

        template = FAQ_RESPONSES.get(intent, FAQ_RESPONSES["default"])
        reply = template.format(eta=eta)
        return {
            "auto_reply": reply,
            "action": "auto_reply",
            "intent": intent,
            "confidence": confidence,
            "sentiment": "neutral",
            "language": lang,
        }

    @staticmethod
    def _validate_llm_output(result: dict) -> bool:
        """校验 LLM 输出的合理性"""
        if not result.get("intent"):
            return False
        sentiment = result.get("sentiment")
        if sentiment is not None and sentiment not in (
            "positive",
            "neutral",
            "negative",
        ):
            return False
        if "requires_human" in result and not isinstance(
            result.get("requires_human"), bool
        ):
            return False
        conf = result.get("confidence")
        if conf is not None:
            try:
                if not 0 <= float(conf) <= 1:
                    return False
            except (TypeError, ValueError):
                return False
        return True

    def _insufficient(self, p: str, m: list) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
