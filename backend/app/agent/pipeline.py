"""Agent 协作流水线

实现 Agent 之间的自动链式调用：
- G1(驾驶舱) 发现风险 → 派 A5(库存)/G3(折扣) 详细分析
- A5(库存预警) 发现断货 → 建议自动转入 action → 通知 A2 优化 listing
- G3(折扣阻断) → 派 A6(利润监控) 检查影响面

每个链规则定义：触发条件、目标 Agent、上下文映射。
链式调用深度限制为 1 级（不递归），避免级联失控。
"""

import logging
from typing import Callable

from app.agent.registry import AgentRegistry
from app.agent.service import AgentService

logger = logging.getLogger(__name__)

# ── 链规则类型 ──────────────────────────────────────────────

ChainCondition = Callable[[dict], bool]
ContextMapper = Callable[[dict], dict]


class ChainRule:
    """一条 Agent 协作链规则"""

    def __init__(
        self,
        trigger_condition: ChainCondition,
        target_agent: str,
        target_decision_point: str,
        context_mapper: ContextMapper,
        description: str = "",
    ):
        self.trigger_condition = trigger_condition
        self.target_agent = target_agent
        self.target_decision_point = target_decision_point
        self.context_mapper = context_mapper
        self.description = description


# ── 链规则定义 ──────────────────────────────────────────────


def _chain_stock_red_to_a5(result: dict) -> bool:
    """A5 红色预警 → 触发补货分析"""
    return (
        result.get("final_decision", {}).get("stock_status") == "red"
        or result.get("agent_output", {}).get("stock_status") == "red"
    )


def _chain_stock_ctx(result: dict) -> dict:
    decision = result.get("final_decision", result.get("agent_output", {}))
    return {
        "sku_code": decision.get("sku_code", ""),
        "sellable_stock": decision.get("sellable_stock", 0),
        "sellable_days": decision.get("sellable_days", 0),
        "suggested_replenish_qty": decision.get("suggested_replenish_qty", 0),
        "source_decision_id": result.get("decision_id"),
    }


def _chain_discount_block(result: dict) -> bool:
    """G3 阻断折扣 → 触发利润核查"""
    decision = result.get("final_decision", result.get("agent_output", {}))
    return decision.get("action") == "block"


def _chain_discount_ctx(result: dict) -> dict:
    decision = result.get("final_decision", result.get("agent_output", {}))
    return {
        "sku_code": decision.get("sku_code", ""),
        "original_price": decision.get("original_price", 0),
        "proposed_discount": decision.get("proposed_discount", 0),
        "block_reason": decision.get("reason", ""),
        "source_decision_id": result.get("decision_id"),
    }


def _chain_a6_loss(result: dict) -> bool:
    """A6 亏损 SKU → 通知 A2 下架优化"""
    decision = result.get("final_decision", result.get("agent_output", {}))
    return decision.get("is_loss") is True or decision.get("below_threshold") is True


def _chain_a6_loss_ctx(result: dict) -> dict:
    decision = result.get("final_decision", result.get("agent_output", {}))
    return {
        "sku_code": decision.get("sku_code", ""),
        "margin_pct": decision.get("margin_pct", 0),
        "anomaly_reason": decision.get("anomaly_reason", ""),
        "source_decision_id": result.get("decision_id"),
    }


# ── 主链表 ──────────────────────────────────────────────────
# 格式: {source_agent_id: {source_decision_point: [ChainRule, ...]}}

CHAINS: dict[str, dict[str, list[ChainRule]]] = {
    "A5": {
        "stock_alert": [
            ChainRule(
                trigger_condition=_chain_stock_red_to_a5,
                target_agent="G3",
                target_decision_point="discount_risk_check",
                context_mapper=_chain_stock_ctx,
                description="A5 红色库存预警 → G3 检查促销折扣风险",
            ),
        ],
    },
    "G3": {
        "discount_risk_check": [
            ChainRule(
                trigger_condition=_chain_discount_block,
                target_agent="A6",
                target_decision_point="profit_watch",
                context_mapper=_chain_discount_ctx,
                description="G3 阻断折扣 → A6 核算利润影响",
            ),
        ],
    },
    "A6": {
        "profit_watch": [
            ChainRule(
                trigger_condition=_chain_a6_loss,
                target_agent="A2",
                target_decision_point="listing_optimize",
                context_mapper=_chain_a6_loss_ctx,
                description="A6 发现亏损 SKU → A2 优化 listing 止损",
            ),
        ],
    },
}


# ── 流水线执行 ──────────────────────────────────────────────


async def evaluate_chains(
    source_agent_id: str,
    decision_point: str,
    decision_result: dict,
    user_id: int,
    db,
    max_depth: int = 1,
) -> list[dict]:
    """执行 Agent 协作链

    Args:
        source_agent_id: 源 Agent
        decision_point: 源决策点
        decision_result: AgentService.execute_decision() 的返回结果
        user_id: 系统用户 ID
        db: 数据库会话
        max_depth: 链深度限制（默认 1，不递归）

    Returns:
        触发链的决策结果列表
    """
    agent_chains = CHAINS.get(source_agent_id, {})
    rules = agent_chains.get(decision_point, [])
    if not rules:
        return []

    triggered = []
    deep = max_depth > 1

    for rule in rules:
        try:
            if not rule.trigger_condition(decision_result):
                continue

            target_cls = AgentRegistry.get_agent_class(rule.target_agent)
            if not target_cls:
                logger.warning("协作链目标 Agent %s 未注册", rule.target_agent)
                continue

            ctx = rule.context_mapper(decision_result)
            agent = target_cls(user_id=user_id)

            result = await AgentService.execute_decision(
                db, agent, rule.target_decision_point, ctx, dry_run=False,
            )
            result["chain_source"] = source_agent_id
            result["chain_source_decision_id"] = decision_result.get("decision_id")
            triggered.append(result)

            logger.info(
                "协作链触发: %s[%s] → %s[%s] (%s)",
                source_agent_id, decision_point,
                rule.target_agent, rule.target_decision_point,
                rule.description,
            )

            # 仅 1 级递归：目标 Agent 的结果再次触发的链
            if deep and result.get("decision_id"):
                nested = await evaluate_chains(
                    rule.target_agent,
                    rule.target_decision_point,
                    result,
                    user_id, db,
                    max_depth=0,
                )
                triggered.extend(nested)

        except Exception as e:
            logger.exception(
                "协作链失败: %s → %s: %s",
                source_agent_id, rule.target_agent, e,
            )

    return triggered
