"""A4 多语言客服 Agent

设计依据: docs/aiagent/final-integrated-solution.md §6.2
- L1 自动回复 60%+ FAQ，L2 辅助人工处理复杂投诉
- 意图分类 + 置信度路由
- 不接真实消息接口，数据通过请求体传入
"""

from typing import Any
from app.agent.base import BaseAgent, EvolutionStage
from app.agent.registry import register_agent

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
            return self._auto_reply(ctx)
        if point == "intent_classify":
            return self._classify(ctx)
        return {"action": "unknown", "confidence": 0.0}

    def _classify(self, ctx: dict) -> dict:
        msg = str(ctx.get("message", "")).lower()
        str(ctx.get("language", "en"))
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

    def _auto_reply(self, ctx: dict) -> dict:
        miss = _missing(ctx, REQUIRED)
        if miss:
            return self._insufficient("auto_reply", miss)
        str(ctx.get("message", ""))
        lang = str(ctx.get("language", "en"))
        order_ctx = ctx.get("order_context", {})
        eta = order_ctx.get("estimated_delivery_days", "5-7")

        classification = self._classify(ctx)
        intent = classification.get("intent", "default")
        confidence = classification.get("confidence", 0.0)

        if classification.get("action") == "escalate_human":
            return {
                "auto_reply": None,
                "action": "escalate",
                "intent": intent,
                "confidence": confidence,
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
            "language": lang,
        }

    def _insufficient(self, p: str, m: list) -> dict:
        return {
            "status": "insufficient_data",
            "decision_point": p,
            "missing_fields": m,
            "message": f"缺少: {', '.join(m)}",
            "confidence": 0.0,
        }
